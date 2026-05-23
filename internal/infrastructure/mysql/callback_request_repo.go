package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/jmoiron/sqlx"
)

type CallbackRequestRepo struct{ db *sqlx.DB }

func NewCallbackRequestRepo(db *sqlx.DB) *CallbackRequestRepo {
	return &CallbackRequestRepo{db: db}
}

func (r *CallbackRequestRepo) Create(ctx context.Context, cb *call.CallbackRequest) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO callback_requests (id, tenant_id, call_id, skill_group_id, caller, status, scheduled_at, attempt_count, last_attempt_at, completed_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cb.ID, cb.TenantID, cb.CallID, cb.SkillGroupID, cb.Caller, cb.Status, cb.ScheduledAt, cb.AttemptCount, cb.LastAttemptAt, cb.CompletedAt, cb.CreatedAt)
	return err
}

func (r *CallbackRequestRepo) GetByID(ctx context.Context, id int64) (*call.CallbackRequest, error) {
	var cb call.CallbackRequest
	err := r.db.GetContext(ctx, &cb, `SELECT * FROM callback_requests WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &cb, nil
}

func (r *CallbackRequestRepo) Update(ctx context.Context, cb *call.CallbackRequest) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE callback_requests SET status=?, attempt_count=?, last_attempt_at=?, completed_at=? WHERE id=?`,
		cb.Status, cb.AttemptCount, cb.LastAttemptAt, cb.CompletedAt, cb.ID)
	return err
}

func (r *CallbackRequestRepo) ListPending(ctx context.Context, tenantID int64) ([]*call.CallbackRequest, error) {
	var items []*call.CallbackRequest
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM callback_requests WHERE tenant_id = ? AND status = 'pending' ORDER BY created_at ASC`,
		tenantID)
	return items, err
}
