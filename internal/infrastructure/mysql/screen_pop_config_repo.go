package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/jmoiron/sqlx"
)

type ScreenPopConfigRepo struct{ db *sqlx.DB }

func NewScreenPopConfigRepo(db *sqlx.DB) *ScreenPopConfigRepo {
	return &ScreenPopConfigRepo{db: db}
}

func (r *ScreenPopConfigRepo) Create(ctx context.Context, s *integration.ScreenPopConfig) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO screen_pop_configs (id, tenant_id, name, url_template, position, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.TenantID, s.Name, s.URLTemplate, s.Position, s.IsActive, s.CreatedAt, s.UpdatedAt)
	return err
}

func (r *ScreenPopConfigRepo) GetByID(ctx context.Context, id int64) (*integration.ScreenPopConfig, error) {
	var s integration.ScreenPopConfig
	err := r.db.GetContext(ctx, &s, `SELECT * FROM screen_pop_configs WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ScreenPopConfigRepo) Update(ctx context.Context, s *integration.ScreenPopConfig) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE screen_pop_configs SET name=?, url_template=?, position=?, is_active=?, updated_at=? WHERE id=?`,
		s.Name, s.URLTemplate, s.Position, s.IsActive, s.UpdatedAt, s.ID)
	return err
}

func (r *ScreenPopConfigRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM screen_pop_configs WHERE id = ?`, id)
	return err
}

func (r *ScreenPopConfigRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*integration.ScreenPopConfig, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM screen_pop_configs WHERE tenant_id = ?`, tenantID)
	var items []*integration.ScreenPopConfig
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM screen_pop_configs WHERE tenant_id = ? ORDER BY position ASC LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	return items, total, err
}
