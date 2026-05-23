package callback

import (
	"context"
	"time"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/rs/zerolog"
)

// Scheduler manages callback request execution.
type Scheduler struct {
	callbackRepo call.CallbackRequestRepository
	callSvc      *call.CallService
	logger       zerolog.Logger
}

func NewScheduler(cbRepo call.CallbackRequestRepository, callSvc *call.CallService, logger zerolog.Logger) *Scheduler {
	return &Scheduler{callbackRepo: cbRepo, callSvc: callSvc, logger: logger}
}

// ProcessPending finds pending callbacks and attempts them.
func (s *Scheduler) ProcessPending(ctx context.Context, tenantID int64) (int, error) {
	pending, err := s.callbackRepo.ListPending(ctx, tenantID)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, cb := range pending {
		if cb.AttemptCount >= 3 {
			now := time.Now()
			cb.Status = "max_attempts"
			cb.LastAttemptAt = &now
			_ = s.callbackRepo.Update(ctx, cb)
			continue
		}

		if cb.ScheduledAt != nil && cb.ScheduledAt.After(time.Now()) {
			continue
		}

		_, err := s.callSvc.CreateOutboundCall(ctx, call.CreateCallInput{
			TenantID:  cb.TenantID,
			Caller:    "callback",
			Callee:    cb.Caller,
			CallType:  call.CallTypeCallback,
			Direction: call.DirectionOutbound,
		})

		now := time.Now()
		cb.AttemptCount++
		cb.LastAttemptAt = &now

		if err != nil {
			cb.Status = "retry"
			s.logger.Warn().Err(err).Int64("callback_id", cb.ID).Msg("callback attempt failed")
		} else {
			cb.Status = "completed"
			cb.CompletedAt = &now
			processed++
		}

		_ = s.callbackRepo.Update(ctx, cb)
	}
	return processed, nil
}
