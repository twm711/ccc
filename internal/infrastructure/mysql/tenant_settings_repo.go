package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/jmoiron/sqlx"
)

type TenantSettingsRepo struct {
	db *sqlx.DB
}

func NewTenantSettingsRepo(db *sqlx.DB) *TenantSettingsRepo {
	return &TenantSettingsRepo{db: db}
}

func (r *TenantSettingsRepo) Upsert(ctx context.Context, s *identity.TenantSettings) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tenant_settings (
		    tenant_id, max_agents, max_concurrent_calls,
		    default_acw_seconds, recording_retention_days, recording_storage_backend,
		    timezone, locale, api_rate_limit_per_sec, familiar_agent_days
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   max_agents = VALUES(max_agents),
		   max_concurrent_calls = VALUES(max_concurrent_calls),
		   default_acw_seconds = VALUES(default_acw_seconds),
		   recording_retention_days = VALUES(recording_retention_days),
		   recording_storage_backend = VALUES(recording_storage_backend),
		   timezone = VALUES(timezone),
		   locale = VALUES(locale),
		   api_rate_limit_per_sec = VALUES(api_rate_limit_per_sec),
		   familiar_agent_days = VALUES(familiar_agent_days)`,
		s.TenantID, s.MaxAgents, s.MaxConcurrentCalls,
		s.DefaultACWSeconds, s.RecordingRetentionDays, s.RecordingStorageBackend,
		s.Timezone, s.Language, s.APIRateLimitPerSec, s.FamiliarAgentDays)
	return err
}

func (r *TenantSettingsRepo) GetByTenantID(ctx context.Context, tenantID int64) (*identity.TenantSettings, error) {
	var s identity.TenantSettings
	err := r.db.GetContext(ctx, &s,
		`SELECT tenant_id, max_agents, max_concurrent_calls,
		        default_acw_seconds, recording_retention_days, recording_storage_backend,
		        timezone, locale, api_rate_limit_per_sec, familiar_agent_days
		 FROM tenant_settings WHERE tenant_id = ?`,
		tenantID)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
