package call

import (
	"context"
	"fmt"
	"time"

	"github.com/divord97/ccc/pkg/snowflake"
)

// TelephonyProvider issues telephony commands to the media layer (e.g. ESL).
type TelephonyProvider interface {
	Originate(ctx context.Context, dest, callerID, eslContext string) (string, error)
	Hangup(ctx context.Context, uuid string) error
	Hold(ctx context.Context, uuid string) error
	Retrieve(ctx context.Context, uuid string) error
	Transfer(ctx context.Context, uuid, dest string) error
	SendDTMF(ctx context.Context, uuid, digits string) error
	Bridge(ctx context.Context, uuid1, uuid2 string) error
	Eavesdrop(ctx context.Context, spyUUID, targetUUID string) error
	Conference(ctx context.Context, uuid, confName string) error
}

type CallService struct {
	calls     CallRepository
	events    CallEventRepository
	tracking  IVRTrackingRepository
	callbacks CallbackRequestRepository
	tp        TelephonyProvider
}

func NewCallService(cr CallRepository, er CallEventRepository, tr IVRTrackingRepository, cbr ...CallbackRequestRepository) *CallService {
	s := &CallService{calls: cr, events: er, tracking: tr}
	if len(cbr) > 0 {
		s.callbacks = cbr[0]
	}
	return s
}

func (s *CallService) SetTelephonyProvider(tp TelephonyProvider) {
	s.tp = tp
}

type CreateCallInput struct {
	TenantID      int64
	Direction     CallDirection
	CallType      CallType
	MediaType     MediaType
	Caller        string
	Callee        string
	AgentUserID   *int64
	IVRFlowID     *int64
	PhoneNumberID *int64
	CarrierID     *int64
	SIPTrunkID    *int64
}

var ErrInvalidMediaType = fmt.Errorf("call: invalid media type, must be audio or video")

func resolveMediaType(mt MediaType) (MediaType, error) {
	if mt == "" {
		return MediaTypeAudio, nil
	}
	if !ValidMediaType(mt) {
		return "", ErrInvalidMediaType
	}
	return mt, nil
}

func (s *CallService) CreateInboundCall(ctx context.Context, in CreateCallInput) (*Call, error) {
	if in.Direction == "" {
		in.Direction = DirectionInbound
	}
	if in.CallType == "" {
		in.CallType = CallTypeNormal
	}
	mt, err := resolveMediaType(in.MediaType)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	c := &Call{
		ID:            snowflake.NextID(),
		TenantID:      in.TenantID,
		Direction:     in.Direction,
		CallType:      in.CallType,
		MediaType:     mt,
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
	} else {
		c.WaitDurationSec = int(now.Sub(c.StartedAt).Seconds())
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
	mt, err := resolveMediaType(in.MediaType)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	c := &Call{
		ID:            snowflake.NextID(),
		TenantID:      in.TenantID,
		Direction:     in.Direction,
		CallType:      in.CallType,
		MediaType:     mt,
		Caller:        in.Caller,
		Callee:        in.Callee,
		AgentUserID:   in.AgentUserID,
		PhoneNumberID: in.PhoneNumberID,
		CarrierID:     in.CarrierID,
		SIPTrunkID:    in.SIPTrunkID,
		Status:        CallStatusRinging,
		StartedAt:     now,
	}

	if s.tp != nil {
		uuid, origErr := s.tp.Originate(ctx, c.Callee, c.Caller, "default")
		if origErr != nil {
			return nil, origErr
		}
		c.ChannelUUID = uuid
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
	mt, err := resolveMediaType(in.MediaType)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	c := &Call{
		ID:          snowflake.NextID(),
		TenantID:    in.TenantID,
		Direction:   in.Direction,
		CallType:    in.CallType,
		MediaType:   mt,
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
	if s.tp != nil {
		if err := s.tp.Hold(ctx, c.ChannelUUID); err != nil {
			return nil, fmt.Errorf("hold failed: %w", err)
		}
	}
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
	if s.tp != nil {
		if err := s.tp.Retrieve(ctx, c.ChannelUUID); err != nil {
			return nil, fmt.Errorf("retrieve failed: %w", err)
		}
	}
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
	if s.tp != nil {
		if err := s.tp.Transfer(ctx, c.ChannelUUID, detail); err != nil {
			return nil, fmt.Errorf("blind transfer failed: %w", err)
		}
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
	if s.tp != nil {
		if err := s.tp.SendDTMF(ctx, c.ChannelUUID, digits); err != nil {
			return fmt.Errorf("send DTMF failed: %w", err)
		}
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

// --- Phase 5: Advanced Call Control ---

// AttendedTransfer performs a warm transfer (agent stays until target answers).
func (s *CallService) AttendedTransfer(ctx context.Context, id int64, target TransferTarget) (*Call, error) {
	c, err := s.calls.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCallNotFound
	}
	if c.Status != CallStatusActive && c.Status != CallStatusHeld {
		return nil, ErrCallNotActive
	}
	if target.Type == "" {
		return nil, ErrMissingTransferTarget
	}

	c.TransferCount++
	detail := "attended:" + target.Type
	if target.ExternalNum != "" {
		detail += ":" + target.ExternalNum
	}
	if s.tp != nil {
		if err := s.tp.Transfer(ctx, c.ChannelUUID, detail); err != nil {
			return nil, fmt.Errorf("attended transfer failed: %w", err)
		}
	}

	if err := s.calls.Update(ctx, c); err != nil {
		return nil, err
	}
	_ = s.events.Create(ctx, &CallEvent{
		ID: snowflake.NextID(), CallID: c.ID, TenantID: c.TenantID,
		Event: "attended_transfer", Detail: detail, CreatedAt: time.Now(),
	})
	return c, nil
}

// InitiateConsult starts a consultation call (original call held, agent talks to target).
func (s *CallService) InitiateConsult(ctx context.Context, id int64, target TransferTarget) (*Call, error) {
	c, err := s.calls.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCallNotFound
	}
	if c.Status != CallStatusActive {
		return nil, ErrCallNotActive
	}
	if target.Type == "" {
		return nil, ErrMissingTransferTarget
	}

	c.Status = CallStatusConsulting
	if err := s.calls.Update(ctx, c); err != nil {
		return nil, err
	}
	_ = s.events.Create(ctx, &CallEvent{
		ID: snowflake.NextID(), CallID: c.ID, TenantID: c.TenantID,
		Event: "consult_initiated", Detail: target.Type, CreatedAt: time.Now(),
	})
	return c, nil
}

// CompleteConsultTransfer completes the consult and transfers the call.
func (s *CallService) CompleteConsultTransfer(ctx context.Context, id int64) (*Call, error) {
	c, err := s.calls.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCallNotFound
	}
	if c.Status != CallStatusConsulting {
		return nil, ErrCallNotConsulting
	}

	c.TransferCount++
	c.Status = CallStatusActive
	if err := s.calls.Update(ctx, c); err != nil {
		return nil, err
	}
	_ = s.events.Create(ctx, &CallEvent{
		ID: snowflake.NextID(), CallID: c.ID, TenantID: c.TenantID,
		Event: "consult_transfer_completed", CreatedAt: time.Now(),
	})
	return c, nil
}

// CancelConsult cancels a consultation and returns to original call.
func (s *CallService) CancelConsult(ctx context.Context, id int64) (*Call, error) {
	c, err := s.calls.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCallNotFound
	}
	if c.Status != CallStatusConsulting {
		return nil, ErrCallNotConsulting
	}

	c.Status = CallStatusActive
	if err := s.calls.Update(ctx, c); err != nil {
		return nil, err
	}
	_ = s.events.Create(ctx, &CallEvent{
		ID: snowflake.NextID(), CallID: c.ID, TenantID: c.TenantID,
		Event: "consult_cancelled", CreatedAt: time.Now(),
	})
	return c, nil
}

// StartConference converts an active call into a conference (three-way).
func (s *CallService) StartConference(ctx context.Context, id int64) (*Call, error) {
	c, err := s.calls.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCallNotFound
	}
	if c.Status != CallStatusActive && c.Status != CallStatusConsulting {
		return nil, ErrCallNotActive
	}

	c.Status = CallStatusConference
	if err := s.calls.Update(ctx, c); err != nil {
		return nil, err
	}
	_ = s.events.Create(ctx, &CallEvent{
		ID: snowflake.NextID(), CallID: c.ID, TenantID: c.TenantID,
		Event: "conference_started", CreatedAt: time.Now(),
	})
	return c, nil
}

// MonitorCall creates a monitor/whisper/barge/intercept session on an active call.
func (s *CallService) MonitorCall(ctx context.Context, tenantID, targetCallID, supervisorID int64, mode string) (*Call, error) {
	target, err := s.calls.GetByID(ctx, targetCallID)
	if err != nil || target == nil {
		return nil, ErrCallNotFound
	}
	if target.Status != CallStatusActive && target.Status != CallStatusConference {
		return nil, ErrMonitorTargetNotActive
	}

	var ct CallType
	switch mode {
	case "listen":
		ct = CallTypeMonitor
	case "whisper":
		ct = CallTypeWhisper
	case "barge":
		ct = CallTypeBarge
	case "intercept":
		ct = CallTypeIntercept
	default:
		return nil, ErrMissingMonitorTarget
	}

	now := time.Now()
	monitor := &Call{
		ID:           snowflake.NextID(),
		TenantID:     tenantID,
		Direction:    DirectionOutbound,
		CallType:     ct,
		MediaType:    MediaTypeAudio,
		AgentUserID:  &supervisorID,
		ParentCallID: &targetCallID,
		Status:       CallStatusActive,
		StartedAt:    now,
	}
	if err := s.calls.Create(ctx, monitor); err != nil {
		return nil, err
	}

	_ = s.events.Create(ctx, &CallEvent{
		ID: snowflake.NextID(), CallID: monitor.ID, TenantID: tenantID,
		Event: "monitor_started", Detail: mode, CreatedAt: now,
	})
	return monitor, nil
}

// CoachCall creates a coaching session (supervisor talks to agent, customer cannot hear).
func (s *CallService) CoachCall(ctx context.Context, tenantID, targetCallID, coachID int64, timeoutSec int) (*Call, error) {
	target, err := s.calls.GetByID(ctx, targetCallID)
	if err != nil || target == nil {
		return nil, ErrCallNotFound
	}
	if target.Status != CallStatusActive {
		return nil, ErrMonitorTargetNotActive
	}
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	now := time.Now()
	coach := &Call{
		ID:           snowflake.NextID(),
		TenantID:     tenantID,
		Direction:    DirectionOutbound,
		CallType:     CallTypeCoach,
		MediaType:    MediaTypeAudio,
		AgentUserID:  &coachID,
		ParentCallID: &targetCallID,
		Status:       CallStatusActive,
		StartedAt:    now,
	}
	if err := s.calls.Create(ctx, coach); err != nil {
		return nil, err
	}

	_ = s.events.Create(ctx, &CallEvent{
		ID: snowflake.NextID(), CallID: coach.ID, TenantID: tenantID,
		Event: "coach_started", Detail: fmt.Sprintf("timeout=%ds", timeoutSec), CreatedAt: now,
	})
	return coach, nil
}

// WhisperPreConnect plays a whisper announcement to the agent before connecting a call.
func (s *CallService) WhisperPreConnect(ctx context.Context, callID int64, message string) error {
	c, err := s.calls.GetByID(ctx, callID)
	if err != nil || c == nil {
		return ErrCallNotFound
	}
	_ = s.events.Create(ctx, &CallEvent{
		ID: snowflake.NextID(), CallID: c.ID, TenantID: c.TenantID,
		Event: "whisper_pre_connect", Detail: message, CreatedAt: time.Now(),
	})
	return nil
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
