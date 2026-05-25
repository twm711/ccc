package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/rs/zerolog"
)

type Service struct {
	configs  integration.WebhookConfigRepository
	logs     integration.WebhookDeliveryLogRepository
	client   *http.Client
	logger   zerolog.Logger
	maxRetry int
	sem      chan struct{}
}

func NewService(configs integration.WebhookConfigRepository, logs integration.WebhookDeliveryLogRepository, logger zerolog.Logger) *Service {
	return &Service{
		configs:  configs,
		logs:     logs,
		client:   &http.Client{Timeout: 10 * time.Second},
		logger:   logger,
		maxRetry: 3,
		sem:      make(chan struct{}, 50),
	}
}

type Event struct {
	TenantID  int64
	Type      string
	Payload   interface{}
	Timestamp time.Time
}

// Deliver sends a webhook event to all matching configs for the tenant.
func (s *Service) Deliver(ctx context.Context, evt Event) {
	configs, err := s.configs.ListActiveByEvent(ctx, evt.TenantID, evt.Type)
	if err != nil {
		s.logger.Error().Err(err).Msg("webhook: list configs failed")
		return
	}

	payloadBytes, _ := json.Marshal(evt.Payload)

	for _, cfg := range configs {
		if !s.matchesEvent(cfg.Events, evt.Type) {
			continue
		}
		cfg := cfg // capture loop variable
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error().Interface("panic", r).Int64("config_id", cfg.ID).Msg("webhook: recovered panic in deliver")
				}
			}()
			s.sem <- struct{}{}
			defer func() { <-s.sem }()
			s.deliverToConfig(context.Background(), cfg, evt.Type, payloadBytes)
		}()
	}
}

func (s *Service) matchesEvent(events, eventType string) bool {
	if events == "*" {
		return true
	}
	for _, e := range strings.Split(events, ",") {
		if strings.TrimSpace(e) == eventType {
			return true
		}
	}
	return false
}

// backoffFor returns the sleep duration before retry N (1-indexed). Exponential
// growth (2s, 4s, 8s, 16s, 32s) with up to ±20% jitter and a 60s cap, so a
// flapping destination won't synchronize retries from every CCC instance.
func backoffFor(attempt int) time.Duration {
	base := time.Duration(1<<attempt) * time.Second // 2,4,8,16,32...
	if base > 60*time.Second {
		base = 60 * time.Second
	}
	jitter := time.Duration(rand.Int63n(int64(base) / 5)) // 0..20%
	return base + jitter
}

// isRetryable returns true for transport/server errors. 4xx responses are
// the destination's authoritative rejection — retrying them just wastes
// resources and pollutes delivery logs.
func isRetryable(statusCode int) bool {
	return statusCode == 0 || statusCode >= 500 || statusCode == 408 || statusCode == 429
}

func (s *Service) deliverToConfig(ctx context.Context, cfg *integration.WebhookConfig, eventType string, payload []byte) {
	var lastErr string
	var respStatus int
	var respBody string
	success := false
	actualAttempts := 0

	for attempt := 1; attempt <= s.maxRetry; attempt++ {
		actualAttempts = attempt
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(payload))
		if err != nil {
			lastErr = err.Error()
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-Event", eventType)

		if cfg.Secret != "" {
			sig := s.sign(payload, cfg.Secret)
			req.Header.Set("X-Webhook-Signature", sig)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err.Error()
			respStatus = 0
			if attempt < s.maxRetry {
				time.Sleep(backoffFor(attempt))
			}
			continue
		}

		respStatus = resp.StatusCode
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		respBody = string(body)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			success = true
			break
		}
		lastErr = fmt.Sprintf("HTTP %d", resp.StatusCode)
		if !isRetryable(resp.StatusCode) {
			break // permanent client rejection; don't waste retries
		}
		if attempt < s.maxRetry {
			time.Sleep(backoffFor(attempt))
		}
	}

	if !success {
		s.logger.Warn().
			Int64("config_id", cfg.ID).
			Int64("tenant_id", cfg.TenantID).
			Str("event", eventType).
			Int("attempts", actualAttempts).
			Int("last_status", respStatus).
			Str("last_error", lastErr).
			Msg("webhook: delivery failed (logged to dead-letter via webhook_deliveries.success=false)")
	}

	if err := s.logs.Create(ctx, &integration.WebhookDeliveryLog{
		ID:              snowflake.NextID(),
		TenantID:        cfg.TenantID,
		WebhookConfigID: cfg.ID,
		EventType:       eventType,
		Payload:         string(payload),
		ResponseStatus:  respStatus,
		ResponseBody:    respBody,
		AttemptCount:    actualAttempts,
		Success:         success,
		ErrorMessage:    lastErr,
		CreatedAt:       time.Now(),
	}); err != nil {
		s.logger.Warn().Err(err).Int64("config_id", cfg.ID).Str("event", eventType).Msg("failed to persist delivery log")
	}
}

func (s *Service) sign(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// ListDLQ returns failed webhook deliveries for a tenant.
func (s *Service) ListDLQ(ctx context.Context, tenantID int64, offset, limit int) ([]*integration.WebhookDeliveryLog, int64, error) {
	return s.logs.ListFailed(ctx, tenantID, offset, limit)
}

// RetryDLQ re-delivers a failed webhook event by ID.
func (s *Service) RetryDLQ(ctx context.Context, id int64) error {
	log, err := s.logs.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("webhook: dlq entry not found: %w", err)
	}
	cfg, err := s.configs.GetByID(ctx, log.WebhookConfigID)
	if err != nil || cfg == nil {
		return fmt.Errorf("webhook: config %d not found", log.WebhookConfigID)
	}
	go func() {
		s.sem <- struct{}{}
		defer func() { <-s.sem }()
		s.deliverToConfig(context.Background(), cfg, log.EventType, []byte(log.Payload))
	}()
	return nil
}

// PurgeDLQ removes failed deliveries older than the given time for a tenant.
func (s *Service) PurgeDLQ(ctx context.Context, tenantID int64, before time.Time) (int64, error) {
	return s.logs.PurgeBefore(ctx, tenantID, before)
}

// ValidateSubscription checks that each event type in a comma-separated list
// is a known event from the catalog. Returns the first invalid event type, or
// "" if all are valid. The wildcard "*" is always accepted.
func ValidateSubscription(events string) (invalid string) {
	if events == "*" || events == "" {
		return ""
	}
	known := make(map[string]bool, len(EventCatalog()))
	for _, d := range EventCatalog() {
		known[d.Type] = true
	}
	for _, e := range strings.Split(events, ",") {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !known[e] {
			return e
		}
	}
	return ""
}

// EventDescriptor describes a webhook event type for the OpenAPI event catalog.
type EventDescriptor struct {
	Type        string `json:"type"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

// EventCatalog returns all supported webhook event types.
func EventCatalog() []EventDescriptor {
	return []EventDescriptor{
		{Type: "call.created", Category: "call", Description: "A new call has been created"},
		{Type: "call.answered", Category: "call", Description: "A call has been answered by an agent"},
		{Type: "call.ended", Category: "call", Description: "A call has ended"},
		{Type: "call.transferred", Category: "call", Description: "A call has been transferred"},
		{Type: "call.queued", Category: "call", Description: "A call has entered the ACD queue"},
		{Type: "agent.status_changed", Category: "agent", Description: "An agent's presence status has changed"},
		{Type: "agent.checkin", Category: "agent", Description: "An agent has checked in"},
		{Type: "agent.checkout", Category: "agent", Description: "An agent has checked out"},
		{Type: "campaign.started", Category: "campaign", Description: "An outbound campaign has started"},
		{Type: "campaign.completed", Category: "campaign", Description: "An outbound campaign has completed"},
		{Type: "campaign.paused", Category: "campaign", Description: "An outbound campaign has been paused"},
		{Type: "ticket.created", Category: "ticket", Description: "A new ticket has been created"},
		{Type: "ticket.updated", Category: "ticket", Description: "A ticket has been updated"},
		{Type: "ticket.resolved", Category: "ticket", Description: "A ticket has been resolved"},
		{Type: "csat.submitted", Category: "csat", Description: "A CSAT survey response has been submitted"},
		{Type: "im.session.created", Category: "im", Description: "An IM session has been created"},
		{Type: "im.session.closed", Category: "im", Description: "An IM session has been closed"},
		{Type: "recording.completed", Category: "recording", Description: "A call recording has completed processing"},
		{Type: "qa.alert", Category: "qa", Description: "A real-time QA sensitive word alert was triggered"},
		{Type: "sla.alarm", Category: "sla", Description: "An SLA threshold alarm was triggered"},
	}
}
