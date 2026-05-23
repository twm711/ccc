package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/jmoiron/sqlx"
)

// CSATConfigRepo implements integration.CSATConfigRepository.
type CSATConfigRepo struct{ db *sqlx.DB }

func NewCSATConfigRepo(db *sqlx.DB) *CSATConfigRepo { return &CSATConfigRepo{db: db} }

func (r *CSATConfigRepo) Create(ctx context.Context, c *integration.CSATConfig) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO csat_configs (id, tenant_id, name, trigger_type, ivr_flow_id, sms_template_id, scale_min, scale_max, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.TenantID, c.Name, c.TriggerType, c.IVRFlowID, c.SmsTemplateID, c.ScaleMin, c.ScaleMax, c.IsActive, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *CSATConfigRepo) GetByID(ctx context.Context, id int64) (*integration.CSATConfig, error) {
	var c integration.CSATConfig
	err := r.db.GetContext(ctx, &c, `SELECT * FROM csat_configs WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CSATConfigRepo) Update(ctx context.Context, c *integration.CSATConfig) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE csat_configs SET name=?, trigger_type=?, ivr_flow_id=?, sms_template_id=?, scale_min=?, scale_max=?, is_active=?, updated_at=? WHERE id=?`,
		c.Name, c.TriggerType, c.IVRFlowID, c.SmsTemplateID, c.ScaleMin, c.ScaleMax, c.IsActive, c.UpdatedAt, c.ID)
	return err
}

func (r *CSATConfigRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*integration.CSATConfig, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM csat_configs WHERE tenant_id = ?`, tenantID)

	var items []*integration.CSATConfig
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM csat_configs WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	return items, total, err
}

func (r *CSATConfigRepo) GetActive(ctx context.Context, tenantID int64) (*integration.CSATConfig, error) {
	var c integration.CSATConfig
	err := r.db.GetContext(ctx, &c, `SELECT * FROM csat_configs WHERE tenant_id = ? AND is_active = 1 LIMIT 1`, tenantID)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CSATResultRepo implements integration.CSATResultRepository.
type CSATResultRepo struct{ db *sqlx.DB }

func NewCSATResultRepo(db *sqlx.DB) *CSATResultRepo { return &CSATResultRepo{db: db} }

func (r *CSATResultRepo) Create(ctx context.Context, res *integration.CSATResult) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO csat_results (id, tenant_id, call_id, config_id, agent_id, rating, comment, channel, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		res.ID, res.TenantID, res.CallID, res.ConfigID, res.AgentID, res.Rating, res.Comment, res.Channel, res.CreatedAt)
	return err
}

func (r *CSATResultRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*integration.CSATResult, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM csat_results WHERE tenant_id = ?`, tenantID)

	var items []*integration.CSATResult
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM csat_results WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	return items, total, err
}
