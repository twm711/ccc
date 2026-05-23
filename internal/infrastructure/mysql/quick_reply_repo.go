package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/jmoiron/sqlx"
)

type QuickReplyRepo struct{ db *sqlx.DB }

func NewQuickReplyRepo(db *sqlx.DB) *QuickReplyRepo { return &QuickReplyRepo{db: db} }

func (r *QuickReplyRepo) Create(ctx context.Context, q *integration.QuickReply) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO quick_replies (id, tenant_id, scope, scope_id, title, content, shortcut, sort_order, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		q.ID, q.TenantID, q.Scope, q.ScopeID, q.Title, q.Content, q.Shortcut, q.SortOrder, q.IsActive, q.CreatedAt, q.UpdatedAt)
	return err
}

func (r *QuickReplyRepo) GetByID(ctx context.Context, id int64) (*integration.QuickReply, error) {
	var q integration.QuickReply
	err := r.db.GetContext(ctx, &q, `SELECT * FROM quick_replies WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *QuickReplyRepo) Update(ctx context.Context, q *integration.QuickReply) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE quick_replies SET title=?, content=?, shortcut=?, sort_order=?, is_active=?, updated_at=? WHERE id=?`,
		q.Title, q.Content, q.Shortcut, q.SortOrder, q.IsActive, q.UpdatedAt, q.ID)
	return err
}

func (r *QuickReplyRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM quick_replies WHERE id = ?`, id)
	return err
}

func (r *QuickReplyRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*integration.QuickReply, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM quick_replies WHERE tenant_id = ?`, tenantID)
	var items []*integration.QuickReply
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM quick_replies WHERE tenant_id = ? ORDER BY sort_order ASC LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	return items, total, err
}

func (r *QuickReplyRepo) ListAvailable(ctx context.Context, tenantID int64, agentID, skillGroupID *int64) ([]*integration.QuickReply, error) {
	query := `SELECT * FROM quick_replies WHERE tenant_id = ? AND is_active = true AND (scope = 'global'`
	args := []interface{}{tenantID}
	if agentID != nil {
		query += ` OR (scope = 'agent' AND scope_id = ?)`
		args = append(args, *agentID)
	}
	if skillGroupID != nil {
		query += ` OR (scope = 'skill_group' AND scope_id = ?)`
		args = append(args, *skillGroupID)
	}
	query += `) ORDER BY sort_order ASC`
	var items []*integration.QuickReply
	err := r.db.SelectContext(ctx, &items, query, args...)
	return items, err
}
