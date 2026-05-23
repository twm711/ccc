package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/jmoiron/sqlx"
)

type TenantRepo struct {
	db *sqlx.DB
}

func NewTenantRepo(db *sqlx.DB) *TenantRepo {
	return &TenantRepo{db: db}
}

func (r *TenantRepo) Create(ctx context.Context, t *identity.Tenant) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tenants (id, code, display_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		t.ID, t.Code, t.Name, t.Status, t.CreatedAt, t.UpdatedAt)
	return err
}

func (r *TenantRepo) GetByID(ctx context.Context, id int64) (*identity.Tenant, error) {
	var t identity.Tenant
	err := r.db.GetContext(ctx, &t, `SELECT id, code, display_name AS name, status, created_at, updated_at FROM tenants WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepo) GetByCode(ctx context.Context, code string) (*identity.Tenant, error) {
	var t identity.Tenant
	err := r.db.GetContext(ctx, &t, `SELECT id, code, display_name AS name, status, created_at, updated_at FROM tenants WHERE code = ?`, code)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepo) Update(ctx context.Context, t *identity.Tenant) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET display_name = ?, status = ?, updated_at = ? WHERE id = ?`,
		t.Name, t.Status, t.UpdatedAt, t.ID)
	return err
}

func (r *TenantRepo) List(ctx context.Context, offset, limit int) ([]*identity.Tenant, int64, error) {
	var total int64
	err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM tenants WHERE status != 'deleted'`)
	if err != nil {
		return nil, 0, err
	}

	var tenants []*identity.Tenant
	err = r.db.SelectContext(ctx, &tenants,
		`SELECT id, code, display_name AS name, status, created_at, updated_at FROM tenants WHERE status != 'deleted' ORDER BY id LIMIT ? OFFSET ?`,
		limit, offset)
	return tenants, total, err
}
