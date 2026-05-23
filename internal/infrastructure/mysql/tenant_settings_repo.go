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
		`INSERT INTO tenant_settings (tenant_id, default_acw_seconds, max_concurrent_calls, recording_retention_days, timezone, locale, api_rate_limit_per_sec)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   default_acw_seconds = VALUES(default_acw_seconds),
		   max_concurrent_calls = VALUES(max_concurrent_calls),
		   recording_retention_days = VALUES(recording_retention_days),
		   timezone = VALUES(timezone),
		   locale = VALUES(locale),
		   api_rate_limit_per_sec = VALUES(api_rate_limit_per_sec)`,
		s.TenantID, s.MaxAgents, s.MaxConcurrentCalls, s.RecordingRetentionDays, s.Timezone, s.Language, 100)
	return err
}

func (r *TenantSettingsRepo) GetByTenantID(ctx context.Context, tenantID int64) (*identity.TenantSettings, error) {
	var s identity.TenantSettings
	err := r.db.GetContext(ctx, &s,
		`SELECT tenant_id, default_acw_seconds AS max_agents, max_concurrent_calls, recording_retention_days, timezone, locale AS language FROM tenant_settings WHERE tenant_id = ?`,
		tenantID)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
