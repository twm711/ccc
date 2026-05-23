package call

import (
	"context"
	"time"

	"github.com/divord97/ccc/pkg/snowflake"
)

type CallService struct {
	calls     CallRepository
	events    CallEventRepository
	tracking  IVRTrackingRepository
	callbacks CallbackRequestRepository
}

func NewCallService(cr CallRepository, er CallEventRepository, tr IVRTrackingRepository, cbr ...CallbackRequestRepository) *CallService {
	s := &CallService{calls: cr, events: er, tracking: tr}
	if len(cbr) > 0 {
		s.callbacks = cbr[0]
	}
	return s
}

type CreateCallInput struct {
	TenantID      int64
	Direction     CallDirection
	CallType      CallType
	Caller        string
	Callee        string
	AgentUserID   *int64
	IVRFlowID     *int64
	PhoneNumberID *int64
	CarrierID     *int64
	SIPTrunkID    *int64
}

func (s *CallService) CreateInboundCall(ctx context.Context, in CreateCallInput) (*Call, error) {
	if in.Direction == "" {
		in.Direction = DirectionInbound
	}
	if in.CallType == "" {
		in.CallType = CallTypeNormal
	}

	now := time.Now()
	c := &Call{
		ID:            snowflake.NextID(),
		TenantID:      in.TenantID,
		Direction:     in.Direction,
		CallType:      in.CallType,
		MediaType:     MediaTypeAudio,
		Caller:        in.Caller,
		Callee:        in.Callee,
		IVRFlowID:     in.IVRFlowID,
		PhoneNumberID: in.PhoneNumberID,
		CarrierID:     in.CarrierID,
		Status:        CallStatusIVR,
		StartedAt:     now,
	}

	if err := s.calls.Create(ctx, c); err != nil {
		return nil, err
	}

	_ = s.events.Create(ctx, &CallEvent{
		ID:        snowflake.NextID(),
		CallID:    c.ID,
		TenantID:  c.TenantID,
		Event:     "call_created",
		Detail:    string(c.Direction),
		CreatedAt: now,
	})

	return c, nil
}

func (s *CallService) RecordIVRTracking(ctx context.Context, t *IVRTracking) error {
	t.ID = snowflake.NextID()
	return s.tracking.Create(ctx, t)
}

func (s *CallService) EndCall(ctx context.Context, id int64, reason HangupReason) (*Call, error) {
	c, err := s.calls.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCallNotFound
	}
	if c.Status == CallStatusCompleted {
		return nil, ErrCallAlreadyEnded
	}

	now := time.Now()
	c.Status = CallStatusCompleted
	c.HangupReason = &reason
	c.EndedAt = &now
	c.DurationSec = int(now.Sub(c.StartedAt).Seconds())

	if c.AnsweredAt != nil {
		c.WaitDurationSec = int(c.AnsweredAt.Sub(c.StartedAt).Seconds())
	}

	if err := s.calls.Update(ctx, c); err != nil {
		return nil, err
	}

	_ = s.events.Create(ctx, &CallEvent{
		ID:        snowflake.NextID(),
		CallID:    c.ID,
		TenantID:  c.TenantID,
		Event:     "call_ended",
		Detail:    string(reason),
		CreatedAt: now,
	})

	return c, nil
}

// CreateOutboundCall creates an outbound call record.
func (s *CallService) CreateOutboundCall(ctx context.Context, in CreateCallInput) (*Call, error) {
	if in.Callee == "" {
		return nil, ErrMissingCallee
	}
	if in.Caller == "" {
		return nil, ErrMissingCaller
	}
	in.Direction = DirectionOutbound
	if in.CallType == "" {
		in.CallType = CallTypeNormal
	}

	now := time.Now()
	c := &Call{
		ID:            snowflake.NextID(),
		TenantID:      in.TenantID,
		Direction:     in.Direction,
		CallType:      in.CallType,
		MediaType:     MediaTypeAudio,
		Caller:        in.Caller,
		Callee:        in.Callee,
		AgentUserID:   in.AgentUserID,
		PhoneNumberID: in.PhoneNumberID,
		CarrierID:     in.CarrierID,
		SIPTrunkID:    in.SIPTrunkID,
		Status:        CallStatusRinging,
		StartedAt:     now,
	}

	if err := s.calls.Create(ctx, c); err != nil {
		return nil, err
	}

	_ = s.events.Create(ctx, &CallEvent{
		ID:        snowflake.NextID(),
		CallID:    c.ID,
		TenantID:  c.TenantID,
		Event:     "call_created",
		Detail:    string(c.Direction),
		CreatedAt: now,
	})

	return c, nil
}

// CreateInternalCall creates an internal (agent-to-agent) call record.
func (s *CallService) CreateInternalCall(ctx context.Context, in CreateCallInput) (*Call, error) {
	if in.Callee == "" {
		return nil, ErrMissingCallee
	}
	if in.Caller == "" {
		return nil, ErrMissingCaller
	}
	in.Direction = DirectionOutbound
	in.CallType = CallTypeInternal

	now := time.Now()
	c := &Call{
		ID:          snowflake.NextID(),
		TenantID:    in.TenantID,
		Direction:   in.Direction,
		CallType:    in.CallType,
		MediaType:   MediaTypeAudio,
		Caller:      in.Caller,
		Callee:      in.Callee,
		AgentUserID: in.AgentUserID,
		Status:      CallStatusRinging,
		StartedAt:   now,
	}

	if err := s.calls.Create(ctx, c); err != nil {
		return nil, err
	}

	_ = s.events.Create(ctx, &CallEvent{
		ID:        snowflake.NextID(),
		CallID:    c.ID,
		TenantID:  c.TenantID,
		Event:     "call_created",
		Detail:    "INTERNAL",
		CreatedAt: now,
	})

	return c, nil
}

func (s *CallService) GetByID(ctx context.Context, id int64) (*Call, error) {
	return s.calls.GetByID(ctx, id)
}

// ListCalls returns calls with optional filtering.
func (s *CallService) ListCalls(ctx context.Context, tenantID int64, filter CallListFilter, offset, limit int) ([]*Call, int64, error) {
	return s.calls.ListWithFilter(ctx, tenantID, filter, offset, limit)
}

func (s *CallService) GetIVRTracking(ctx context.Context, callID int64) ([]*IVRTracking, error) {
	return s.tracking.ListByCallID(ctx, callID)
}

func (s *CallService) GetEvents(ctx context.Context, callID int64) ([]*CallEvent, error) {
	return s.events.ListByCallID(ctx, callID)
}

// HoldCall puts a call on hold.
func (s *CallService) HoldCall(ctx context.Context, id int64) (*Call, error) {
	c, err := s.calls.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCallNotFound
	}
	if c.Status != CallStatusActive {
		return nil, ErrCallNotActive
	}
	c.Status = CallStatusHeld
	c.HoldCount++
	if err := s.calls.Update(ctx, c); err != nil {
		return nil, err
	}
	_ = s.events.Create(ctx, &CallEvent{
		ID: snowflake.NextID(), CallID: c.ID, TenantID: c.TenantID,
		Event: "call_held", CreatedAt: time.Now(),
	})
	return c, nil
}

// RetrieveCall takes a call off hold.
func (s *CallService) RetrieveCall(ctx context.Context, id int64) (*Call, error) {
	c, err := s.calls.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCallNotFound
	}
	if c.Status != CallStatusHeld {
		return nil, ErrCallNotHeld
	}
	c.Status = CallStatusActive
	if err := s.calls.Update(ctx, c); err != nil {
		return nil, err
	}
	_ = s.events.Create(ctx, &CallEvent{
		ID: snowflake.NextID(), CallID: c.ID, TenantID: c.TenantID,
		Event: "call_retrieved", CreatedAt: time.Now(),
	})
	return c, nil
}

type TransferTarget struct {
	Type         string // "skill_group", "agent", "external"
	SkillGroupID *int64
	AgentUserID  *int64
	ExternalNum  string
}

// BlindTransfer transfers a call without consultation.
func (s *CallService) BlindTransfer(ctx context.Context, id int64, target TransferTarget) (*Call, error) {
	c, err := s.calls.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCallNotFound
	}
	if c.Status == CallStatusCompleted || c.Status == CallStatusFailed {
		return nil, ErrCallAlreadyEnded
	}
	if target.Type == "" {
		return nil, ErrMissingTransferTarget
	}

	c.TransferCount++
	detail := target.Type
	if target.ExternalNum != "" {
		detail += ":" + target.ExternalNum
	}

	if err := s.calls.Update(ctx, c); err != nil {
		return nil, err
	}
	_ = s.events.Create(ctx, &CallEvent{
		ID: snowflake.NextID(), CallID: c.ID, TenantID: c.TenantID,
		Event: "blind_transfer", Detail: detail, CreatedAt: time.Now(),
	})
	return c, nil
}

// SendDTMF records a DTMF event on a call.
func (s *CallService) SendDTMF(ctx context.Context, id int64, digits string) error {
	if digits == "" {
		return ErrMissingDTMF
	}
	c, err := s.calls.GetByID(ctx, id)
	if err != nil || c == nil {
		return ErrCallNotFound
	}
	if c.Status == CallStatusCompleted || c.Status == CallStatusFailed {
		return ErrCallAlreadyEnded
	}
	_ = s.events.Create(ctx, &CallEvent{
		ID: snowflake.NextID(), CallID: c.ID, TenantID: c.TenantID,
		Event: "dtmf_sent", Detail: digits, CreatedAt: time.Now(),
	})
	return nil
}

// RequestCallback creates a callback request for a queued caller.
func (s *CallService) RequestCallback(ctx context.Context, cb *CallbackRequest) error {
	cb.ID = snowflake.NextID()
	cb.Status = "pending"
	cb.CreatedAt = time.Now()
	return s.callbacks.Create(ctx, cb)
}

// ExecuteCallback marks a callback as attempted/completed.
func (s *CallService) ExecuteCallback(ctx context.Context, id int64, success bool) (*CallbackRequest, error) {
	cb, err := s.callbacks.GetByID(ctx, id)
	if err != nil || cb == nil {
		return nil, ErrCallbackNotFound
	}
	now := time.Now()
	cb.AttemptCount++
	cb.LastAttemptAt = &now
	if success {
		cb.Status = "completed"
		cb.CompletedAt = &now
	} else {
		cb.Status = "failed"
	}
	if err := s.callbacks.Update(ctx, cb); err != nil {
		return nil, err
	}
	return cb, nil
}

// CalculateDurations computes IVR/ring/queue/wait durations from call events.
func (s *CallService) CalculateDurations(c *Call, events []*CallEvent) {
	var ivrStart, queueStart, ringStart time.Time

	for _, e := range events {
		switch e.Event {
		case "call_created":
			ivrStart = e.CreatedAt
		case "ivr_completed":
			if !ivrStart.IsZero() {
				c.IVRDurationSec = int(e.CreatedAt.Sub(ivrStart).Seconds())
			}
			queueStart = e.CreatedAt
		case "queue_entered":
			queueStart = e.CreatedAt
		case "agent_ringing":
			if !queueStart.IsZero() {
				c.QueueDurationSec = int(e.CreatedAt.Sub(queueStart).Seconds())
			}
			ringStart = e.CreatedAt
		case "call_answered":
			if !ringStart.IsZero() {
				c.RingDurationSec = int(e.CreatedAt.Sub(ringStart).Seconds())
			}
		}
	}
}
