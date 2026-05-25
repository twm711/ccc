package retention

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// RecordingCleaner deletes recordings older than a cutoff date for a tenant.
type RecordingCleaner interface {
	DeleteBefore(ctx context.Context, tenantID int64, before time.Time) (int64, error)
}

// CDRCleaner deletes call detail records older than a cutoff date for a tenant.
type CDRCleaner interface {
	DeleteBefore(ctx context.Context, tenantID int64, before time.Time) (int64, error)
}

// TranscriptCleaner deletes transcription data older than a cutoff date.
type TranscriptCleaner interface {
	DeleteBefore(ctx context.Context, tenantID int64, before time.Time) (int64, error)
}

// TenantRetentionConfig provides per-tenant retention settings.
type TenantRetentionConfig interface {
	GetRetentionDays(ctx context.Context, tenantID int64) (recordingDays, cdrDays, transcriptDays int, err error)
}

// Service enforces data retention policies by periodically cleaning up
// expired recordings, CDRs, and transcripts per tenant configuration.
type Service struct {
	recordings  RecordingCleaner
	cdrs        CDRCleaner
	transcripts TranscriptCleaner
	config      TenantRetentionConfig
	logger      zerolog.Logger
}

// NewService creates a retention enforcement service.
func NewService(config TenantRetentionConfig, logger zerolog.Logger) *Service {
	return &Service{config: config, logger: logger}
}

// SetRecordingCleaner sets the recording cleanup adapter.
func (s *Service) SetRecordingCleaner(c RecordingCleaner) { s.recordings = c }

// SetCDRCleaner sets the CDR cleanup adapter.
func (s *Service) SetCDRCleaner(c CDRCleaner) { s.cdrs = c }

// SetTranscriptCleaner sets the transcript cleanup adapter.
func (s *Service) SetTranscriptCleaner(c TranscriptCleaner) { s.transcripts = c }

// CleanupTenant enforces retention policy for a single tenant.
func (s *Service) CleanupTenant(ctx context.Context, tenantID int64) CleanupResult {
	recDays, cdrDays, txDays, err := s.config.GetRetentionDays(ctx, tenantID)
	if err != nil {
		s.logger.Error().Err(err).Int64("tenant_id", tenantID).Msg("retention: failed to get config")
		return CleanupResult{TenantID: tenantID}
	}

	now := time.Now()
	var result CleanupResult
	result.TenantID = tenantID

	if s.recordings != nil && recDays > 0 {
		cutoff := now.AddDate(0, 0, -recDays)
		n, err := s.recordings.DeleteBefore(ctx, tenantID, cutoff)
		if err != nil {
			s.logger.Warn().Err(err).Int64("tenant_id", tenantID).Msg("retention: recording cleanup failed")
		}
		result.RecordingsDeleted = n
	}

	if s.cdrs != nil && cdrDays > 0 {
		cutoff := now.AddDate(0, 0, -cdrDays)
		n, err := s.cdrs.DeleteBefore(ctx, tenantID, cutoff)
		if err != nil {
			s.logger.Warn().Err(err).Int64("tenant_id", tenantID).Msg("retention: CDR cleanup failed")
		}
		result.CDRsDeleted = n
	}

	if s.transcripts != nil && txDays > 0 {
		cutoff := now.AddDate(0, 0, -txDays)
		n, err := s.transcripts.DeleteBefore(ctx, tenantID, cutoff)
		if err != nil {
			s.logger.Warn().Err(err).Int64("tenant_id", tenantID).Msg("retention: transcript cleanup failed")
		}
		result.TranscriptsDeleted = n
	}

	s.logger.Info().
		Int64("tenant_id", tenantID).
		Int64("recordings", result.RecordingsDeleted).
		Int64("cdrs", result.CDRsDeleted).
		Int64("transcripts", result.TranscriptsDeleted).
		Msg("retention: cleanup complete")

	return result
}

// CleanupResult summarizes what was deleted for a tenant.
type CleanupResult struct {
	TenantID           int64 `json:"tenant_id"`
	RecordingsDeleted  int64 `json:"recordings_deleted"`
	CDRsDeleted        int64 `json:"cdrs_deleted"`
	TranscriptsDeleted int64 `json:"transcripts_deleted"`
}

// Run starts a periodic retention cleanup loop.
func (s *Service) Run(ctx context.Context, interval time.Duration, tenantIDs func(ctx context.Context) ([]int64, error)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ids, err := tenantIDs(ctx)
			if err != nil {
				s.logger.Error().Err(err).Msg("retention: failed to list tenants")
				continue
			}
			for _, id := range ids {
				s.CleanupTenant(ctx, id)
			}
		}
	}
}
