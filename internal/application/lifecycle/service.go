package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/divord97/ccc/internal/application/csat"
	"github.com/divord97/ccc/internal/application/ivr"
	"github.com/divord97/ccc/internal/application/screenpop"
	"github.com/divord97/ccc/internal/application/webhook"
	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/domain/campaign"
	"github.com/divord97/ccc/internal/domain/crm"
	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/internal/infrastructure/esl"
	"github.com/divord97/ccc/pkg/snowflake"
)

// AgentNotifier pushes real-time events to connected agent WebSocket clients.
type AgentNotifier interface {
	NotifyAgent(agentID int64, eventType string, callID int64, payload interface{})
}

// Service orchestrates cross-domain side effects for call lifecycle events.
type Service struct {
	callSvc       *call.CallService
	presenceSvc   *identity.AgentPresenceService
	csatSvc       *csat.Service
	webhookSvc    *webhook.Service
	customerSvc   *crm.CustomerService
	screenPop     *screenpop.Service
	recordingRepo     call.RecordingRepository
	queueSnapshotRepo call.QueueSnapshotRepository
	eslClient         *esl.Client
	notifier          AgentNotifier
	ivrEngine         *ivr.Engine
	campaignSvc       *campaign.CampaignService
}

func NewService(
	callSvc *call.CallService,
	presenceSvc *identity.AgentPresenceService,
	csatSvc *csat.Service,
	webhookSvc *webhook.Service,
	customerSvc *crm.CustomerService,
	screenPop *screenpop.Service,
	recordingRepo call.RecordingRepository,
	eslClient *esl.Client,
) *Service {
	return &Service{
		callSvc:       callSvc,
		presenceSvc:   presenceSvc,
		csatSvc:       csatSvc,
		webhookSvc:    webhookSvc,
		customerSvc:   customerSvc,
		screenPop:     screenPop,
		recordingRepo: recordingRepo,
		eslClient:     eslClient,
	}
}

func (s *Service) SetAgentNotifier(n AgentNotifier) {
	s.notifier = n
}

func (s *Service) SetCampaignService(svc *campaign.CampaignService) {
	s.campaignSvc = svc
}

func (s *Service) SetIVREngine(e *ivr.Engine) {
	s.ivrEngine = e
}

func (s *Service) SetQueueSnapshotRepo(r call.QueueSnapshotRepository) {
	s.queueSnapshotRepo = r
}


// EndCall ends a call and triggers all post-call side effects:
//   - Hangup FreeSWITCH channel
//   - Save recording record to DB
//   - Calculate IVR/queue/ring durations from call events
//   - Agent → ACW state transition
//   - Real-time WebSocket notification to agent
//   - CSAT satisfaction survey
//   - Webhook notification to external systems
//   - CRM interaction history record
func (s *Service) EndCall(ctx context.Context, callID int64, reason call.HangupReason) (*call.Call, error) {
	c, err := s.callSvc.EndCall(ctx, callID, reason)
	if err != nil {
		return nil, err
	}

	// Hangup FreeSWITCH channel
	if s.eslClient != nil && c.ChannelUUID != "" {
		_ = s.eslClient.HangupCall(ctx, c.ChannelUUID)
	}

	// Save recording record (only for answered calls that had recording started)
	if s.recordingRepo != nil && c.AnsweredAt != nil && c.ChannelUUID != "" {
		durSec := 0
		if c.EndedAt != nil {
			durSec = int(c.EndedAt.Sub(*c.AnsweredAt).Seconds())
		}
		recPath := fmt.Sprintf("/recordings/%d/%d.wav", c.TenantID, c.ID)
		_ = s.recordingRepo.Create(ctx, &call.Recording{
			ID:          snowflake.NextID(),
			TenantID:    c.TenantID,
			CallID:      c.ID,
			AgentUserID: c.AgentUserID,
			FileName:    fmt.Sprintf("%d.wav", c.ID),
			FilePath:    recPath,
			DurationSec: durSec,
			MimeType:    "audio/wav",
			StorageTier: "hot",
			Consent:     true,
			Status:      "completed",
			CreatedAt:   time.Now(),
		})
		c.RecordingURL = &recPath
		_ = s.callSvc.UpdateDurations(ctx, c)
	}

	// Calculate IVR/queue/ring durations from call events
	events, _ := s.callSvc.ListEvents(ctx, callID)
	if len(events) > 0 {
		s.callSvc.CalculateDurations(c, events)
		_ = s.callSvc.UpdateDurations(ctx, c)
	}

	// Agent → ACW (non-blocking, best-effort)
	if s.presenceSvc != nil && c.AgentUserID != nil {
		_, _ = s.presenceSvc.SetACW(ctx, *c.AgentUserID, "")
	}

	// Real-time agent notification
	if s.notifier != nil && c.AgentUserID != nil {
		s.notifier.NotifyAgent(*c.AgentUserID, "call.ended", c.ID, c)
	}

	// CSAT survey trigger (non-blocking)
	if s.csatSvc != nil {
		_ = s.csatSvc.TriggerSurvey(ctx, c.TenantID, c.ID, c.AgentUserID)
	}

	// Webhook notification (async, non-blocking)
	if s.webhookSvc != nil {
		s.webhookSvc.Deliver(ctx, webhook.Event{
			TenantID:  c.TenantID,
			Type:      "call.ended",
			Payload:   c,
			Timestamp: time.Now(),
		})
	}

	// CRM interaction record (non-blocking)
	if s.customerSvc != nil && c.AgentUserID != nil {
		phone := c.Caller
		if c.Direction == call.DirectionOutbound {
			phone = c.Callee
		}
		customer, _ := s.customerSvc.FindByPhone(ctx, c.TenantID, phone)
		if customer != nil {
			_ = s.customerSvc.RecordInteraction(ctx, crm.RecordInteractionInput{
				CustomerID: customer.ID,
				TenantID:   c.TenantID,
				Channel:    "voice",
				Direction:  string(c.Direction),
				Summary:    fmt.Sprintf("%s call, duration %ds", c.Direction, c.DurationSec),
				CallID:     &c.ID,
				AgentName:  fmt.Sprintf("agent_%d", *c.AgentUserID),
			})
		}
	}

	// Campaign case writeback: answered→completed (with duration), unanswered→failed (retry)
	if s.campaignSvc != nil && c.CampaignCaseID != nil {
		if c.AnsweredAt != nil {
			talkSec := 0
			if c.EndedAt != nil {
				talkSec = int(c.EndedAt.Sub(*c.AnsweredAt).Seconds())
			}
			_, _ = s.campaignSvc.MarkCaseCompleted(ctx, *c.CampaignCaseID, "completed", talkSec)
		} else {
			_, _ = s.campaignSvc.MarkCaseFailed(ctx, *c.CampaignCaseID)
		}
	}

	return c, nil
}

// AnswerCall marks a call as answered and triggers side effects:
//   - Agent → Talking state
//   - Start call recording via ESL
//   - Real-time WebSocket notification
//   - Screen pop for inbound calls
//   - Webhook notification
func (s *Service) AnswerCall(ctx context.Context, callID int64, agentUserID int64) (*call.Call, *screenpop.ScreenPopData, error) {
	c, err := s.callSvc.AnswerCall(ctx, callID, agentUserID)
	if err != nil {
		return nil, nil, err
	}

	// Agent → Talking state
	if s.presenceSvc != nil {
		_, _ = s.presenceSvc.TransitionTo(ctx, agentUserID, identity.PresenceTalking)
	}

	// Start recording via ESL
	if s.eslClient != nil && c.ChannelUUID != "" {
		filePath := fmt.Sprintf("/recordings/%d/%d.wav", c.TenantID, c.ID)
		_ = s.eslClient.StartRecording(ctx, c.ChannelUUID, filePath)
	}

	// Real-time agent notification
	if s.notifier != nil {
		s.notifier.NotifyAgent(agentUserID, "call.answered", c.ID, c)
	}

	// Screen pop for inbound calls
	var popData *screenpop.ScreenPopData
	if s.screenPop != nil && c.Direction == call.DirectionInbound {
		popData, _ = s.screenPop.BuildScreenPop(ctx, c.TenantID, screenpop.CallInfo{
			CallID:      c.ID,
			Caller:      c.Caller,
			Callee:      c.Callee,
			Direction:   string(c.Direction),
			AgentUserID: c.AgentUserID,
		})
	}

	// Webhook: call.answered
	if s.webhookSvc != nil {
		s.webhookSvc.Deliver(ctx, webhook.Event{
			TenantID:  c.TenantID,
			Type:      "call.answered",
			Payload:   c,
			Timestamp: time.Now(),
		})
	}

	return c, popData, nil
}

// HandleInboundCall creates an inbound call, runs IVR if configured, and transitions to queue.
func (s *Service) HandleInboundCall(ctx context.Context, in call.CreateCallInput) (*call.Call, error) {
	c, err := s.callSvc.CreateInboundCall(ctx, in)
	if err != nil {
		return nil, err
	}

	// Run IVR flow if engine and flow are configured
	if s.ivrEngine != nil && c.IVRFlowID != nil {
		sess := ivr.NewSession(c.ID, c.TenantID, *c.IVRFlowID, c.ChannelUUID, s.eslClient, map[string]string{
			"caller_number": c.Caller,
			"callee_number": c.Callee,
			"call_id":       fmt.Sprintf("%d", c.ID),
		})
		// IVR execution is best-effort; errors are logged as events
		if err := s.ivrEngine.ExecuteFlow(ctx, sess, *c.IVRFlowID); err != nil {
			now := time.Now()
			_ = s.callSvc.RecordIVRTracking(ctx, &call.IVRTracking{
				CallID:    c.ID,
				TenantID:  c.TenantID,
				IVRFlowID: *c.IVRFlowID,
				NodeID:    "error",
				NodeType:  "error",
				ExitName:  err.Error(),
				EnteredAt: now,
				ExitedAt:  &now,
			})
		}
	}

	// Webhook: call.created (inbound)
	if s.webhookSvc != nil {
		s.webhookSvc.Deliver(ctx, webhook.Event{
			TenantID:  c.TenantID,
			Type:      "call.created",
			Payload:   c,
			Timestamp: time.Now(),
		})
	}

	return c, nil
}

// TransitionCallToQueue moves a call from IVR to Queue status via lifecycle.
func (s *Service) TransitionCallToQueue(ctx context.Context, callID, skillGroupID int64) (*call.Call, error) {
	c, err := s.callSvc.TransitionToQueue(ctx, callID, skillGroupID)
	if err != nil {
		return nil, err
	}
	if s.queueSnapshotRepo != nil {
		_ = s.queueSnapshotRepo.Create(ctx, &call.QueueSnapshot{
			ID:           snowflake.NextID(),
			TenantID:     c.TenantID,
			SkillGroupID: skillGroupID,
			WaitingCount: 1,
			SnapshotAt:   time.Now(),
		})
	}
	if s.notifier != nil && c.AgentUserID != nil {
		s.notifier.NotifyAgent(*c.AgentUserID, "call.queued", c.ID, c)
	}
	return c, nil
}

// TransitionCallToRinging moves a call from Queue/IVR to Ringing status via lifecycle.
func (s *Service) TransitionCallToRinging(ctx context.Context, callID, agentUserID int64) (*call.Call, error) {
	c, err := s.callSvc.TransitionToRinging(ctx, callID, agentUserID)
	if err != nil {
		return nil, err
	}
	if s.presenceSvc != nil {
		_, _ = s.presenceSvc.TransitionTo(ctx, agentUserID, identity.PresenceDialing)
	}
	if s.notifier != nil {
		s.notifier.NotifyAgent(agentUserID, "call.ringing", c.ID, c)
	}
	return c, nil
}
