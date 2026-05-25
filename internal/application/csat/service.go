package csat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/rs/zerolog"
)

// SMSSender sends an SMS survey to the customer.
type SMSSender interface {
	Send(ctx context.Context, tenantID int64, phone, templateCode string, params map[string]string) error
}

// CallerLookup retrieves a call record to extract the customer phone number.
type CallerLookup interface {
	GetByID(ctx context.Context, id int64) (caller string, err error)
}

// TicketCreator creates a ticket from a CSAT-triggered event.
type TicketCreator interface {
	CreateAutoTicket(ctx context.Context, tenantID int64, title, description, priority string, callID *int64) error
}

// Service manages CSAT survey triggering and result collection.
type Service struct {
	configs       integration.CSATConfigRepository
	results       integration.CSATResultRepository
	smsSender     SMSSender
	callerLook    CallerLookup
	ticketCreator TicketCreator
	lowScoreThreshold int
	logger        zerolog.Logger
}

func NewService(configs integration.CSATConfigRepository, results integration.CSATResultRepository, logger zerolog.Logger) *Service {
	return &Service{configs: configs, results: results, logger: logger}
}

// SetSMSSender wires the SMS channel for CSAT surveys.
func (s *Service) SetSMSSender(sender SMSSender) { s.smsSender = sender }

// SetCallerLookup wires a function to resolve the customer phone from a call ID.
func (s *Service) SetCallerLookup(l CallerLookup) { s.callerLook = l }

// SetTicketCreator wires automatic ticket creation for low CSAT scores.
// threshold defines the score at or below which a ticket is created (e.g. 2).
func (s *Service) SetTicketCreator(tc TicketCreator, threshold int) {
	s.ticketCreator = tc
	s.lowScoreThreshold = threshold
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
		s.sendSMSSurvey(ctx, cfg, tenantID, callID)
	case "both":
		s.logger.Info().Int64("call_id", callID).Msg("csat: IVR survey triggered")
		s.sendSMSSurvey(ctx, cfg, tenantID, callID)
	}

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
	if err := s.results.Create(ctx, result); err != nil {
		return err
	}

	// Auto-create complaint ticket on low score.
	if s.ticketCreator != nil && s.lowScoreThreshold > 0 && rating <= s.lowScoreThreshold {
		title := fmt.Sprintf("Low CSAT (score=%d) for call %d", rating, callID)
		desc := fmt.Sprintf("Customer rated %d/%d. Comment: %s", rating, 5, comment)
		if err := s.ticketCreator.CreateAutoTicket(ctx, tenantID, title, desc, "high", &callID); err != nil {
			s.logger.Warn().Err(err).Int64("call_id", callID).Int("rating", rating).Msg("csat: auto-ticket creation failed")
		}
	}

	return nil
}

// sendSMSSurvey resolves the customer phone from the call and sends the CSAT
// SMS via the configured template.
func (s *Service) sendSMSSurvey(ctx context.Context, cfg *integration.CSATConfig, tenantID, callID int64) {
	if s.smsSender == nil || s.callerLook == nil {
		s.logger.Warn().Int64("call_id", callID).Msg("csat: SMS sender or caller lookup not configured, skipping SMS survey")
		return
	}
	if cfg.SmsTemplateID == "" {
		s.logger.Warn().Int64("call_id", callID).Msg("csat: no SMS template configured, skipping")
		return
	}
	phone, err := s.callerLook.GetByID(ctx, callID)
	if err != nil || phone == "" {
		s.logger.Warn().Err(err).Int64("call_id", callID).Msg("csat: could not resolve caller phone")
		return
	}
	params := map[string]string{
		"call_id": fmt.Sprintf("%d", callID),
	}
	if err := s.smsSender.Send(ctx, tenantID, phone, cfg.SmsTemplateID, params); err != nil {
		s.logger.Error().Err(err).Int64("call_id", callID).Str("phone", phone).Msg("csat: SMS survey send failed")
		return
	}
	s.logger.Info().Int64("call_id", callID).Str("phone", phone).Msg("csat: SMS survey sent")
}
