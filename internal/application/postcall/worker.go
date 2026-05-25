package postcall

import (
	"context"
	"encoding/json"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/rs/zerolog"
)

// Worker consumes ccc.call.ended events from NATS and runs post-call
// processing (CDR generation, analytics hooks, etc).
type Worker struct {
	callRepo call.CallRepository
	logger   zerolog.Logger
}

// NewWorker creates a post-call processing worker.
func NewWorker(callRepo call.CallRepository, logger zerolog.Logger) *Worker {
	return &Worker{callRepo: callRepo, logger: logger}
}

// HandleMessage processes a single NATS message. It is used as a
// nats.MessageHandler callback.
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

	hangupReason := ""
	if c.HangupReason != nil {
		hangupReason = string(*c.HangupReason)
	}

	w.logger.Info().
		Int64("call_id", c.ID).
		Int64("tenant_id", c.TenantID).
		Str("direction", string(c.Direction)).
		Str("hangup_reason", hangupReason).
		Msg("postcall: CDR recorded")

	return nil
}

func (w *Worker) handleCallAnswered(ctx context.Context, data []byte) error {
	var c call.Call
	if err := json.Unmarshal(data, &c); err != nil {
		w.logger.Error().Err(err).Msg("postcall: unmarshal call.answered")
		return nil
	}

	w.logger.Info().
		Int64("call_id", c.ID).
		Int64("tenant_id", c.TenantID).
		Msg("postcall: call answered event consumed")

	return nil
}
