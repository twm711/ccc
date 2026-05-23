package sms

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/rs/zerolog"
)

type Service struct {
	configs integration.SmsConfigRepository
	logger  zerolog.Logger
}

func NewService(configs integration.SmsConfigRepository, logger zerolog.Logger) *Service {
	return &Service{configs: configs, logger: logger}
}

type SendRequest struct {
	TenantID     int64
	Phone        string
	TemplateCode string
	Params       map[string]string
}

// Send sends an SMS using the tenant's active config.
func (s *Service) Send(ctx context.Context, req SendRequest) error {
	configs, _, err := s.configs.List(ctx, req.TenantID, 0, 10)
	if err != nil {
		return fmt.Errorf("sms: list configs: %w", err)
	}

	var cfg *integration.SmsConfig
	for _, c := range configs {
		if c.IsActive {
			cfg = c
			break
		}
	}
	if cfg == nil {
		return fmt.Errorf("sms: no active config for tenant %d", req.TenantID)
	}

	var templateMap map[string]string
	if err := json.Unmarshal([]byte(cfg.TemplateMap), &templateMap); err != nil {
		return fmt.Errorf("sms: parse template map: %w", err)
	}

	templateID, ok := templateMap[req.TemplateCode]
	if !ok {
		return fmt.Errorf("sms: template code %q not found", req.TemplateCode)
	}

	s.logger.Info().
		Str("provider", cfg.Provider).
		Str("phone", req.Phone).
		Str("template_id", templateID).
		Str("sign", cfg.SignName).
		Msg("sms: sending (stub)")

	// In production: integrate with Alibaba Cloud SMS SDK
	// aliyun.SendSms(cfg.AccessKeyID, cfg.SignName, templateID, req.Phone, req.Params)
	return nil
}
