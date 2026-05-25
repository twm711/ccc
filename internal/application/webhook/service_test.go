package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/rs/zerolog"
)

// --- stubs ---

type stubConfigRepo struct {
	url    string
	events string
}

func (r *stubConfigRepo) Create(_ context.Context, _ *integration.WebhookConfig) error       { return nil }
func (r *stubConfigRepo) GetByID(_ context.Context, _ int64) (*integration.WebhookConfig, error) {
	return nil, nil
}
func (r *stubConfigRepo) Update(_ context.Context, _ *integration.WebhookConfig) error { return nil }
func (r *stubConfigRepo) Delete(_ context.Context, _ int64) error                      { return nil }
func (r *stubConfigRepo) List(_ context.Context, _ int64, _, _ int) ([]*integration.WebhookConfig, int64, error) {
	return nil, 0, nil
}
func (r *stubConfigRepo) ListActiveByEvent(_ context.Context, _ int64, _ string) ([]*integration.WebhookConfig, error) {
	return []*integration.WebhookConfig{{
		ID:       1,
		TenantID: 1,
		URL:      r.url,
		Events:   r.events,
		IsActive: true,
	}}, nil
}

type stubLogRepo struct{}

func (r *stubLogRepo) Create(_ context.Context, _ *integration.WebhookDeliveryLog) error { return nil }
func (r *stubLogRepo) List(_ context.Context, _ int64, _, _ int) ([]*integration.WebhookDeliveryLog, int64, error) {
	return nil, 0, nil
}

// --- tests ---

func TestDeliver_SendsToMatchingConfig(t *testing.T) {
	var received bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected json content-type, got %s", ct)
		}
		if evt := r.Header.Get("X-Webhook-Event"); evt != "call.ended" {
			t.Errorf("expected call.ended event header, got %s", evt)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	svc := &Service{
		configs:  &stubConfigRepo{url: ts.URL, events: "call.ended"},
		logs:     &stubLogRepo{},
		client:   ts.Client(),
		logger:   zerolog.Nop(),
		maxRetry: 1,
		sem:      make(chan struct{}, 10),
	}

	svc.Deliver(context.Background(), Event{
		TenantID:  1,
		Type:      "call.ended",
		Payload:   map[string]string{"id": "123"},
		Timestamp: time.Now(),
	})

	time.Sleep(200 * time.Millisecond)

	if !received {
		t.Error("webhook was not delivered")
	}
}

func TestSign_Deterministic(t *testing.T) {
	svc := &Service{}
	sig1 := svc.sign([]byte("hello"), "secret")
	sig2 := svc.sign([]byte("hello"), "secret")
	if sig1 == "" {
		t.Error("signature should not be empty")
	}
	if sig1 != sig2 {
		t.Error("same input should produce same signature")
	}
}

func TestMatchesEvent(t *testing.T) {
	svc := &Service{}

	if !svc.matchesEvent("*", "call.ended") {
		t.Error("wildcard should match any event")
	}
	if !svc.matchesEvent("call.ended,call.answered", "call.ended") {
		t.Error("comma-separated should match")
	}
	if svc.matchesEvent("call.answered", "call.ended") {
		t.Error("non-matching event should not match")
	}
}
