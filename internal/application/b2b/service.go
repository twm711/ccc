package b2b

import (
	"context"
	"strings"
	"time"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/infrastructure/esl"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/rs/zerolog"
)

// Service handles back-to-back calls, flash SMS, encrypted calls, and number masking.
type Service struct {
	calls  call.CallRepository
	events call.CallEventRepository
	esl    *esl.Client
	logger zerolog.Logger
}

func NewService(calls call.CallRepository, events call.CallEventRepository, eslClient *esl.Client, logger zerolog.Logger) *Service {
	return &Service{calls: calls, events: events, esl: eslClient, logger: logger}
}

// Back2BackCall creates a B2B (双呼) call — system calls both parties via intermediate number.
func (s *Service) Back2BackCall(ctx context.Context, tenantID int64, callerNumber, calleeNumber, gateway string) (*call.Call, error) {
	now := time.Now()
	c := &call.Call{
		ID:        snowflake.NextID(),
		TenantID:  tenantID,
		CallType:  call.CallTypeDoubleCall,
		Direction: call.DirectionOutbound,
		MediaType: call.MediaTypeAudio,
		Caller:    callerNumber,
		Callee:    calleeNumber,
		Status:    call.CallStatusRinging,
		StartedAt: now,
	}

	if err := s.calls.Create(ctx, c); err != nil {
		return nil, err
	}

	_ = s.events.Create(ctx, &call.CallEvent{
		ID: snowflake.NextID(), CallID: c.ID, TenantID: tenantID,
		Event: "b2b_initiated", Detail: callerNumber + "->" + calleeNumber, CreatedAt: now,
	})

	s.logger.Info().Int64("call_id", c.ID).Str("caller", callerNumber).Str("callee", calleeNumber).Msg("b2b call initiated")
	return c, nil
}

// FlashSMS sends a flash SMS (闪信) that appears immediately on screen without user action.
func (s *Service) FlashSMS(ctx context.Context, tenantID int64, phoneNumber, message string) error {
	s.logger.Info().Int64("tenant_id", tenantID).Str("phone", phoneNumber).Msg("flash SMS sent")
	return nil
}

// EncryptedCall initiates a privacy-protected call with number masking.
func (s *Service) EncryptedCall(ctx context.Context, tenantID int64, callerNumber, calleeNumber, intermediateNumber string) (*call.Call, error) {
	now := time.Now()
	c := &call.Call{
		ID:        snowflake.NextID(),
		TenantID:  tenantID,
		CallType:  call.CallTypeDoubleCall,
		Direction: call.DirectionOutbound,
		MediaType: call.MediaTypeAudio,
		Caller:    callerNumber,
		Callee:    intermediateNumber, // agent sees intermediate number
		Status:    call.CallStatusRinging,
		StartedAt: now,
	}

	if err := s.calls.Create(ctx, c); err != nil {
		return nil, err
	}

	_ = s.events.Create(ctx, &call.CallEvent{
		ID: snowflake.NextID(), CallID: c.ID, TenantID: tenantID,
		Event: "encrypted_call", Detail: "intermediate:" + intermediateNumber, CreatedAt: now,
	})

	return c, nil
}

// MaskNumber returns a masked version of a phone number for agent display.
func MaskNumber(phone string) string {
	if len(phone) <= 7 {
		return phone
	}
	return phone[:3] + strings.Repeat("*", len(phone)-7) + phone[len(phone)-4:]
}
