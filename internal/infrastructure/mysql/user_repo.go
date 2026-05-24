package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/jmoiron/sqlx"
)

type UserRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u *identity.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, tenant_id, user_name, display_name, email, phone, role, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.TenantID, u.Username, u.DisplayName, u.Email, u.Phone, u.Role, u.Status, u.CreatedAt, u.UpdatedAt)
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*identity.User, error) {
	var u identity.User
	err := r.db.GetContext(ctx, &u,
		`SELECT id, tenant_id, user_name AS username, display_name, email, phone, role, status, COALESCE(password_hash,'') AS password_hash, created_at, updated_at FROM users WHERE id = ? AND status != 'deleted'`, id)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, tenantID int64, username string) (*identity.User, error) {
	var u identity.User
	err := r.db.GetContext(ctx, &u,
		`SELECT id, tenant_id, user_name AS username, display_name, email, phone, role, status, created_at, updated_at FROM users WHERE tenant_id = ? AND user_name = ? AND status != 'deleted'`,
		tenantID, username)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Update(ctx context.Context, u *identity.User) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET display_name = ?, email = ?, phone = ?, role = ?, status = ?, updated_at = ? WHERE id = ?`,
		u.DisplayName, u.Email, u.Phone, u.Role, u.Status, u.UpdatedAt, u.ID)
	return err
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = ?, updated_at = NOW() WHERE id = ?`, passwordHash, id)
	return err
}

func (r *UserRepo) FindByUsernameGlobal(ctx context.Context, username string) (*identity.User, error) {
	var u identity.User
	err := r.db.GetContext(ctx, &u,
		`SELECT id, tenant_id, user_name AS username, display_name, email, phone, role, status, COALESCE(password_hash,'') AS password_hash, created_at, updated_at FROM users WHERE user_name = ? AND status != 'deleted' LIMIT 1`, username)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*identity.User, int64, error) {
	var total int64
	err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM users WHERE tenant_id = ? AND status != 'deleted'`, tenantID)
	if err != nil {
		return nil, 0, err
	}

	var users []*identity.User
	err = r.db.SelectContext(ctx, &users,
		`SELECT id, tenant_id, user_name AS username, display_name, email, phone, role, status, created_at, updated_at FROM users WHERE tenant_id = ? AND status != 'deleted' ORDER BY id LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	return users, total, err
}
