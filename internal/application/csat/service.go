package csat

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/rs/zerolog"
)

// Service manages CSAT survey triggering and result collection.
type Service struct {
	configs integration.CSATConfigRepository
	results integration.CSATResultRepository
	logger  zerolog.Logger
}

func NewService(configs integration.CSATConfigRepository, results integration.CSATResultRepository, logger zerolog.Logger) *Service {
	return &Service{configs: configs, results: results, logger: logger}
}

// TriggerSurvey initiates a CSAT survey for a completed call.
func (s *Service) TriggerSurvey(ctx context.Context, tenantID, callID int64, agentID *int64) error {
	cfg, err := s.configs.GetActive(ctx, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug().Int64("tenant_id", tenantID).Msg("csat: no active config, skip survey")
			return nil
		}
		return err
	}

	switch cfg.TriggerType {
	case "ivr":
		s.logger.Info().Int64("call_id", callID).Msg("csat: IVR survey triggered")
	case "sms":
		s.logger.Info().Int64("call_id", callID).Msg("csat: SMS survey triggered")
	case "both":
		s.logger.Info().Int64("call_id", callID).Msg("csat: IVR+SMS survey triggered")
	}

	_ = cfg
	return nil
}

// RecordResult saves a CSAT rating.
func (s *Service) RecordResult(ctx context.Context, tenantID, callID, configID int64, agentID *int64, rating int, comment, channel string) error {
	result := &integration.CSATResult{
		ID:        snowflake.NextID(),
		TenantID:  tenantID,
		CallID:    callID,
		ConfigID:  configID,
		AgentID:   agentID,
		Rating:    rating,
		Comment:   comment,
		Channel:   channel,
		CreatedAt: time.Now(),
	}
	return s.results.Create(ctx, result)
}
