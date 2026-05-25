package callback

import (
	"context"
	"time"

	"github.com/divord97/ccc/internal/application/outbound"
	"github.com/divord97/ccc/internal/domain/call"
	"github.com/rs/zerolog"
)

// Scheduler manages callback request execution.
type Scheduler struct {
	callbackRepo call.CallbackRequestRepository
	callSvc      *call.CallService
	outboundSvc  *outbound.Service
	logger       zerolog.Logger
}

func NewScheduler(cbRepo call.CallbackRequestRepository, callSvc *call.CallService, outboundSvc *outbound.Service, logger zerolog.Logger) *Scheduler {
	return &Scheduler{callbackRepo: cbRepo, callSvc: callSvc, outboundSvc: outboundSvc, logger: logger}
}

// Run drives the callback scheduler in the background, polling every interval
// until ctx is canceled.
func (s *Scheduler) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n, err := s.ProcessAllPending(ctx)
			if err != nil {
				s.logger.Warn().Err(err).Msg("callback: scheduler tick failed")
			} else if n > 0 {
				s.logger.Info().Int("processed", n).Msg("callback: scheduler tick completed")
			}
		}
	}
}

// ProcessAllPending finds pending callbacks across all tenants and attempts them.
func (s *Scheduler) ProcessAllPending(ctx context.Context) (int, error) {
	pending, err := s.callbackRepo.ListAllPending(ctx)
	if err != nil {
		return 0, err
	}
	return s.processList(ctx, pending)
}

// ProcessPending finds pending callbacks for a specific tenant and attempts them.
func (s *Scheduler) ProcessPending(ctx context.Context, tenantID int64) (int, error) {
	pending, err := s.callbackRepo.ListPending(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	return s.processList(ctx, pending)
}

func (s *Scheduler) processList(ctx context.Context, pending []*call.CallbackRequest) (int, error) {
	processed := 0
	for _, cb := range pending {
		maxAttempts := cb.MaxAttempts
		if maxAttempts <= 0 {
			maxAttempts = 3
		}
		if cb.AttemptCount >= maxAttempts {
			now := time.Now()
			cb.Status = "max_attempts"
			cb.LastAttemptAt = &now
			if err := s.callbackRepo.Update(ctx, cb); err != nil {
				s.logger.Warn().Err(err).Int64("callback_id", cb.ID).Msg("failed to update max_attempts status")
			}
			continue
		}

		if cb.ScheduledAt != nil && cb.ScheduledAt.After(time.Now()) {
			continue
		}

		var err error
		if s.outboundSvc != nil {
			_, err = s.outboundSvc.Dial(ctx, outbound.DialRequest{
				TenantID:  cb.TenantID,
				Callee:    cb.Caller,
				MediaType: call.MediaTypeAudio,
			})
		} else {
			_, err = s.callSvc.CreateOutboundCall(ctx, call.CreateCallInput{
				TenantID:  cb.TenantID,
				Caller:    "callback",
				Callee:    cb.Caller,
				CallType:  call.CallTypeCallback,
				Direction: call.DirectionOutbound,
			})
		}

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

		if err := s.callbackRepo.Update(ctx, cb); err != nil {
			s.logger.Warn().Err(err).Int64("callback_id", cb.ID).Msg("failed to update callback status")
		}
	}
	return processed, nil
}
