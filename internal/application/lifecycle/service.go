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
	"github.com/divord97/ccc/pkg/bizlog"
	"github.com/divord97/ccc/pkg/metrics"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/rs/zerolog"
)

// AgentNotifier pushes real-time events to connected agent WebSocket clients.
type AgentNotifier interface {
	NotifyAgent(agentID int64, eventType string, callID int64, payload interface{})
}

// ConcurrencyGuard tracks per-tenant concurrent call counts.
type ConcurrencyGuard interface {
	Acquire(ctx context.Context, tenantID int64, maxConcurrent int) (bool, error)
	Release(ctx context.Context, tenantID int64)
}

// TenantSettingsLookup resolves tenant settings for concurrency limits.
type TenantSettingsLookup interface {
	GetByTenantID(ctx context.Context, tenantID int64) (maxConcurrentCalls int)
}

// EventPublisher pushes call/agent lifecycle events to an external bus
// (typically NATS JetStream). Implementations must be safe for concurrent use
// and should not block the caller; failures are best-effort.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, data interface{}) error
}

// FamiliarAgentRecorder updates the per-caller "last agent" affinity cache so
// the ACD familiar routing policy can prefer the same agent on repeat calls.
type FamiliarAgentRecorder interface {
	RememberAgent(ctx context.Context, tenantID int64, caller string, agentUserID int64, ttlDays int)
}

// QAAutoTrigger runs quality inspection after a call ends.
type QAAutoTrigger interface {
	AutoInspect(ctx context.Context, tenantID, callID int64)
}

// Service orchestrates cross-domain side effects for call lifecycle events.
type Service struct {
	callSvc           *call.CallService
	presenceSvc       *identity.AgentPresenceService
	csatSvc           *csat.Service
	webhookSvc        *webhook.Service
	customerSvc       *crm.CustomerService
	screenPop         *screenpop.Service
	recordingRepo     call.RecordingRepository
	queueSnapshotRepo call.QueueSnapshotRepository
	eslClient         *esl.Client
	notifier          AgentNotifier
	ivrEngine         *ivr.Engine
	campaignSvc       *campaign.CampaignService
	publisher         EventPublisher
	familiar          FamiliarAgentRecorder
	familiarTTLDays   func(tenantID int64) int
	concurrency       ConcurrencyGuard
	tenantSettings    TenantSettingsLookup
	recordingAnnounce func(ctx context.Context, tenantID int64) bool
	qaTrigger         QAAutoTrigger
	logger            zerolog.Logger
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
	logger zerolog.Logger,
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
		logger:        logger,
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

// SetEventPublisher wires an external bus (e.g. NATS) for call/agent events.
// Without one, publishing is silently skipped.
func (s *Service) SetEventPublisher(p EventPublisher) {
	s.publisher = p
}

// SetFamiliarRecorder wires the ACD "last agent" cache. ttlDays returns the
// retention window per tenant; callers typically read tenant_settings.familiar_agent_days.
func (s *Service) SetFamiliarRecorder(rec FamiliarAgentRecorder, ttlDays func(tenantID int64) int) {
	s.familiar = rec
	s.familiarTTLDays = ttlDays
}

// SetRecordingAnnounceLookup wires a function that returns whether the tenant
// requires a recording compliance announcement before call recording starts.
func (s *Service) SetRecordingAnnounceLookup(fn func(ctx context.Context, tenantID int64) bool) {
	s.recordingAnnounce = fn
}

// SetQAAutoTrigger wires QA auto-inspection for completed calls.
func (s *Service) SetQAAutoTrigger(t QAAutoTrigger) {
	s.qaTrigger = t
}

// SetConcurrencyGuard wires the per-tenant concurrent call limiter.
func (s *Service) SetConcurrencyGuard(cg ConcurrencyGuard, tsl TenantSettingsLookup) {
	s.concurrency = cg
	s.tenantSettings = tsl
}

func (s *Service) publish(ctx context.Context, subject string, payload interface{}) {
	if s.publisher == nil {
		return
	}
	_ = s.publisher.Publish(ctx, subject, payload)
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
func (s *Service) EndCall(ctx context.Context, callID int64, reason call.HangupReason, hangupBy ...call.HangupBy) (*call.Call, error) {
	c, err := s.callSvc.EndCall(ctx, callID, reason, hangupBy...)
	if err != nil {
		return nil, err
	}

	bizlog.CallEvent(s.logger, c.TenantID, c.ID, "call.ended").
		Str("direction", string(c.Direction)).
		Str("hangup_reason", string(reason)).
		Msg("call ended")

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

	// Post-call hooks run asynchronously to avoid blocking the call teardown path.
	go s.postCallHooksAsync(c)

	if c.AnsweredAt == nil && c.Direction == call.DirectionInbound {
		metrics.CallsAbandoned.Inc()
	}

	if s.concurrency != nil {
		s.concurrency.Release(ctx, c.TenantID)
	}
	hangupByLabel := "system"
	if c.HangupBy != nil {
		hangupByLabel = string(*c.HangupBy)
	}
	metrics.CallsEnded.WithLabelValues(hangupByLabel).Inc()
	metrics.ActiveCallsGauge.Dec()
	metrics.TenantActiveCalls.WithLabelValues(fmt.Sprintf("%d", c.TenantID)).Dec()

	s.publish(ctx, "ccc.call.ended", c)

	return c, nil
}

// checkConcurrencyLimit verifies the tenant has not exceeded max_concurrent_calls.
func (s *Service) checkConcurrencyLimit(ctx context.Context, tenantID int64) error {
	if s.concurrency == nil || s.tenantSettings == nil {
		return nil
	}
	maxConcurrent := s.tenantSettings.GetByTenantID(ctx, tenantID)
	if maxConcurrent <= 0 {
		return nil
	}
	allowed, err := s.concurrency.Acquire(ctx, tenantID, maxConcurrent)
	if err != nil {
		return nil // fail open
	}
	if !allowed {
		metrics.ConcurrencyRejected.Inc()
		return fmt.Errorf("tenant %d exceeded max concurrent calls (%d)", tenantID, maxConcurrent)
	}
	return nil
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

	// SLO: track answer latency and SLA compliance
	if c.AnsweredAt != nil {
		latency := c.AnsweredAt.Sub(c.StartedAt).Seconds()
		metrics.CallAnswerLatency.Observe(latency)
		if latency <= 20 {
			metrics.SLAMet.Inc()
		} else {
			metrics.SLAMissed.Inc()
		}
	}
	tenantStr := fmt.Sprintf("%d", c.TenantID)
	metrics.TenantActiveCalls.WithLabelValues(tenantStr).Inc()

	// Agent → Talking state
	if s.presenceSvc != nil {
		_, _ = s.presenceSvc.TransitionTo(ctx, agentUserID, identity.PresenceTalking)
	}

	// Play recording compliance announcement if configured, then start recording.
	if s.eslClient != nil && c.ChannelUUID != "" {
		if s.recordingAnnounce != nil && s.recordingAnnounce(ctx, c.TenantID) {
			_ = s.eslClient.WhisperAnnouncement(ctx, c.ChannelUUID, "ivr/recording_announce.wav")
		}
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

	s.publish(ctx, "ccc.call.answered", c)

	return c, popData, nil
}

// HandleInboundCall creates an inbound call, runs IVR if configured, and transitions to queue.
func (s *Service) HandleInboundCall(ctx context.Context, in call.CreateCallInput) (*call.Call, error) {
	if err := s.checkConcurrencyLimit(ctx, in.TenantID); err != nil {
		return nil, err
	}

	c, err := s.callSvc.CreateInboundCall(ctx, in)
	if err != nil {
		return nil, err
	}
	metrics.CallsCreated.WithLabelValues("inbound").Inc()
	metrics.ActiveCallsGauge.Inc()

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

	s.publish(ctx, "ccc.call.created", c)

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

// postCallHooksAsync runs non-critical post-call side effects asynchronously.
func (s *Service) postCallHooksAsync(c *call.Call) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// CSAT survey trigger
	if s.csatSvc != nil {
		_ = s.csatSvc.TriggerSurvey(ctx, c.TenantID, c.ID, c.AgentUserID)
	}

	// Webhook notification
	if s.webhookSvc != nil {
		s.webhookSvc.Deliver(ctx, webhook.Event{
			TenantID:  c.TenantID,
			Type:      "call.ended",
			Payload:   c,
			Timestamp: time.Now(),
		})
	}

	// CRM interaction record + bidirectional call↔customer linking
	if s.customerSvc != nil && c.AgentUserID != nil {
		phone := c.Caller
		if c.Direction == call.DirectionOutbound {
			phone = c.Callee
		}
		customer, _ := s.customerSvc.FindByPhone(ctx, c.TenantID, phone)
		if customer != nil {
			if c.CustomerID == nil {
				c.CustomerID = &customer.ID
				_ = s.callSvc.UpdateCustomerID(ctx, c.ID, customer.ID)
			}
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

	// Campaign case writeback
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

	// Familiar-customer affinity
	if s.familiar != nil && c.AgentUserID != nil && c.Direction == call.DirectionInbound && c.Caller != "" {
		ttlDays := 30
		if s.familiarTTLDays != nil {
			if d := s.familiarTTLDays(c.TenantID); d > 0 {
				ttlDays = d
			}
		}
		s.familiar.RememberAgent(ctx, c.TenantID, c.Caller, *c.AgentUserID, ttlDays)
	}

	// QA auto-inspection
	if s.qaTrigger != nil && c.AnsweredAt != nil {
		s.qaTrigger.AutoInspect(ctx, c.TenantID, c.ID)
	}
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
