package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/divord97/ccc/internal/application/csat"
	"github.com/divord97/ccc/internal/application/screenpop"
	"github.com/divord97/ccc/internal/application/webhook"
	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/domain/crm"
	"github.com/divord97/ccc/internal/domain/identity"
)

// AgentNotifier pushes real-time events to connected agent WebSocket clients.
type AgentNotifier interface {
	NotifyAgent(agentID int64, eventType string, callID int64, payload interface{})
}

// Service orchestrates cross-domain side effects for call lifecycle events.
type Service struct {
	callSvc     *call.CallService
	presenceSvc *identity.AgentPresenceService
	csatSvc     *csat.Service
	webhookSvc  *webhook.Service
	customerSvc *crm.CustomerService
	screenPop   *screenpop.Service
	notifier    AgentNotifier
}

func NewService(
	callSvc *call.CallService,
	presenceSvc *identity.AgentPresenceService,
	csatSvc *csat.Service,
	webhookSvc *webhook.Service,
	customerSvc *crm.CustomerService,
	screenPop *screenpop.Service,
) *Service {
	return &Service{
		callSvc:     callSvc,
		presenceSvc: presenceSvc,
		csatSvc:     csatSvc,
		webhookSvc:  webhookSvc,
		customerSvc: customerSvc,
		screenPop:   screenPop,
	}
}

func (s *Service) SetAgentNotifier(n AgentNotifier) {
	s.notifier = n
}

// EndCall ends a call and triggers all post-call side effects:
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

	return c, nil
}

// AnswerCall marks a call as answered and triggers inbound screen pop.
func (s *Service) AnswerCall(ctx context.Context, callID int64, agentUserID int64) (*call.Call, *screenpop.ScreenPopData, error) {
	c, err := s.callSvc.AnswerCall(ctx, callID, agentUserID)
	if err != nil {
		return nil, nil, err
	}

	// Agent → Talking state
	if s.presenceSvc != nil {
		_, _ = s.presenceSvc.TransitionTo(ctx, agentUserID, identity.PresenceTalking)
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
