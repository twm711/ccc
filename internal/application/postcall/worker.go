package postcall

import (
	"context"
	"encoding/json"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/rs/zerolog"
)

// QAAutoTrigger optionally inspects a finished call against the tenant's default QA scheme.
type QAAutoTrigger interface {
	AutoInspect(ctx context.Context, tenantID, callID int64)
}

// Worker consumes ccc.call.* events from NATS and runs durable post-call work
// (daily CDR aggregation, AI hooks, QA auto-trigger). Lifecycle's in-process
// goroutine still owns latency-sensitive steps (CSAT, webhook, dialer release).
type Worker struct {
	callRepo  call.CallRepository
	cdrRepo   CDRRepository
	qaTrigger QAAutoTrigger
	logger    zerolog.Logger
}

// NewWorker creates a post-call processing worker. cdrRepo may be nil during
// tests; the worker degrades gracefully.
func NewWorker(callRepo call.CallRepository, cdrRepo CDRRepository, logger zerolog.Logger) *Worker {
	return &Worker{callRepo: callRepo, cdrRepo: cdrRepo, logger: logger}
}

// SetQAAutoTrigger wires QA auto-inspection into the worker.
func (w *Worker) SetQAAutoTrigger(t QAAutoTrigger) { w.qaTrigger = t }

// HandleMessage processes a single NATS message. Used as a nats.MessageHandler callback.
func (w *Worker) HandleMessage(ctx context.Context, subject string, data []byte) error {
	switch subject {
	case "ccc.call.ended":
		return w.handleCallEnded(ctx, data)
	case "ccc.call.answered":
		return w.handleCallAnswered(ctx, data)
	default:
		w.logger.Debug().Str("subject", subject).Msg("postcall: unhandled subject")
		return nil
	}
}

func (w *Worker) handleCallEnded(ctx context.Context, data []byte) error {
	var c call.Call
	if err := json.Unmarshal(data, &c); err != nil {
		w.logger.Error().Err(err).Msg("postcall: unmarshal call.ended")
		return nil // don't retry on bad data
	}

	if w.cdrRepo != nil {
		entry := CDREntry{
			TenantID:     c.TenantID,
			BucketDate:   c.StartedAt.Format("2006-01-02"),
			TalkSeconds:  c.DurationSec,
			RingSeconds:  c.RingDurationSec,
			QueueSeconds: c.QueueDurationSec,
		}
		if c.Direction == call.DirectionInbound {
			entry.Inbound = 1
		} else {
			entry.Outbound = 1
		}
		if c.AnsweredAt != nil {
			entry.Answered = 1
		} else {
			entry.Abandoned = 1
		}
		if err := w.cdrRepo.UpsertDailyCDR(ctx, entry); err != nil {
			w.logger.Error().Err(err).Int64("call_id", c.ID).Msg("postcall: daily cdr upsert failed")
			return err // let NATS retry; idempotent upsert is safe under MaxDeliver
		}
	}

	if w.qaTrigger != nil && c.AnsweredAt != nil {
		w.qaTrigger.AutoInspect(ctx, c.TenantID, c.ID)
	}

	w.logger.Info().
		Int64("call_id", c.ID).
		Int64("tenant_id", c.TenantID).
		Str("direction", string(c.Direction)).
		Int("talk_sec", c.DurationSec).
		Msg("postcall: call processed")

	return nil
}

func (w *Worker) handleCallAnswered(ctx context.Context, data []byte) error {
	var c call.Call
	if err := json.Unmarshal(data, &c); err != nil {
		w.logger.Error().Err(err).Msg("postcall: unmarshal call.answered")
		return nil
	}

	w.logger.Debug().
		Int64("call_id", c.ID).
		Int64("tenant_id", c.TenantID).
		Msg("postcall: call.answered consumed")

	return nil
}
