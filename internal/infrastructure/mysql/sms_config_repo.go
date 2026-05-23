package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/jmoiron/sqlx"
)

type SmsConfigRepo struct{ db *sqlx.DB }

func NewSmsConfigRepo(db *sqlx.DB) *SmsConfigRepo { return &SmsConfigRepo{db: db} }

func (r *SmsConfigRepo) Create(ctx context.Context, s *integration.SmsConfig) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sms_configs (id, tenant_id, provider, access_key_id, sign_name, template_map, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.TenantID, s.Provider, s.AccessKeyID, s.SignName, s.TemplateMap, s.IsActive, s.CreatedAt, s.UpdatedAt)
	return err
}

func (r *SmsConfigRepo) GetByID(ctx context.Context, id int64) (*integration.SmsConfig, error) {
	var s integration.SmsConfig
	err := r.db.GetContext(ctx, &s, `SELECT * FROM sms_configs WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SmsConfigRepo) Update(ctx context.Context, s *integration.SmsConfig) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sms_configs SET provider=?, access_key_id=?, sign_name=?, template_map=?, is_active=?, updated_at=? WHERE id=?`,
		s.Provider, s.AccessKeyID, s.SignName, s.TemplateMap, s.IsActive, s.UpdatedAt, s.ID)
	return err
}

func (r *SmsConfigRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sms_configs WHERE id = ?`, id)
	return err
}

func (r *SmsConfigRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*integration.SmsConfig, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM sms_configs WHERE tenant_id = ?`, tenantID)
	var items []*integration.SmsConfig
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM sms_configs WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	return items, total, err
}
