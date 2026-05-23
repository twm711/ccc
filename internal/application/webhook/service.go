package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
}

func NewService(configs integration.WebhookConfigRepository, logs integration.WebhookDeliveryLogRepository, logger zerolog.Logger) *Service {
	return &Service{
		configs:  configs,
		logs:     logs,
		client:   &http.Client{Timeout: 10 * time.Second},
		logger:   logger,
		maxRetry: 3,
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
		go s.deliverToConfig(ctx, cfg, evt.Type, payloadBytes)
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

func (s *Service) deliverToConfig(ctx context.Context, cfg *integration.WebhookConfig, eventType string, payload []byte) {
	var lastErr string
	var respStatus int
	var respBody string
	success := false

	for attempt := 1; attempt <= s.maxRetry; attempt++ {
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
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}

		respStatus = resp.StatusCode
		buf := make([]byte, 1024)
		n, _ := resp.Body.Read(buf)
		resp.Body.Close()
		respBody = string(buf[:n])

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			success = true
			break
		}
		lastErr = fmt.Sprintf("HTTP %d", resp.StatusCode)
		time.Sleep(time.Duration(attempt) * 2 * time.Second)
	}

	_ = s.logs.Create(ctx, &integration.WebhookDeliveryLog{
		ID:              snowflake.NextID(),
		TenantID:        cfg.TenantID,
		WebhookConfigID: cfg.ID,
		EventType:       eventType,
		Payload:         string(payload),
		ResponseStatus:  respStatus,
		ResponseBody:    respBody,
		AttemptCount:    s.maxRetry,
		Success:         success,
		ErrorMessage:    lastErr,
		CreatedAt:       time.Now(),
	})
}

func (s *Service) sign(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
