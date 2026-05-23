package call

import (
	"context"
	"time"

	"github.com/divord97/ccc/pkg/snowflake"
)

type CallService struct {
	calls    CallRepository
	events   CallEventRepository
	tracking IVRTrackingRepository
}

func NewCallService(cr CallRepository, er CallEventRepository, tr IVRTrackingRepository) *CallService {
	return &CallService{calls: cr, events: er, tracking: tr}
}

type CreateCallInput struct {
	TenantID      int64
	Direction     CallDirection
	CallType      CallType
	Caller        string
	Callee        string
	IVRFlowID     *int64
	PhoneNumberID *int64
	CarrierID     *int64
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

func (s *CallService) GetByID(ctx context.Context, id int64) (*Call, error) {
	return s.calls.GetByID(ctx, id)
}

func (s *CallService) GetIVRTracking(ctx context.Context, callID int64) ([]*IVRTracking, error) {
	return s.tracking.ListByCallID(ctx, callID)
}

func (s *CallService) GetEvents(ctx context.Context, callID int64) ([]*CallEvent, error) {
	return s.events.ListByCallID(ctx, callID)
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
