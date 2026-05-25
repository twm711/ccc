package postcall

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/rs/zerolog"
)

func TestHandleMessage_CallEnded(t *testing.T) {
	w := NewWorker(call.NewMockCallRepo(), zerolog.Nop())

	c := call.Call{
		ID:       1,
		TenantID: 1,
		Direction: call.DirectionInbound,
		Status:   call.CallStatusCompleted,
		StartedAt: time.Now(),
	}
	data, _ := json.Marshal(c)

	err := w.HandleMessage(context.Background(), "ccc.call.ended", data)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestHandleMessage_CallAnswered(t *testing.T) {
	w := NewWorker(call.NewMockCallRepo(), zerolog.Nop())

	c := call.Call{
		ID:       2,
		TenantID: 1,
		Direction: call.DirectionInbound,
		Status:   call.CallStatusActive,
		StartedAt: time.Now(),
	}
	data, _ := json.Marshal(c)

	err := w.HandleMessage(context.Background(), "ccc.call.answered", data)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestHandleMessage_UnknownSubject(t *testing.T) {
	w := NewWorker(call.NewMockCallRepo(), zerolog.Nop())

	err := w.HandleMessage(context.Background(), "ccc.call.unknown", []byte("{}"))
	if err != nil {
		t.Fatalf("unknown subjects should be silently ignored, got %v", err)
	}
}

func TestHandleMessage_BadJSON(t *testing.T) {
	w := NewWorker(call.NewMockCallRepo(), zerolog.Nop())

	err := w.HandleMessage(context.Background(), "ccc.call.ended", []byte("not json"))
	if err != nil {
		t.Fatalf("bad JSON should not return error (non-retryable), got %v", err)
	}
}
