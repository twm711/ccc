package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/platform"
	"github.com/jmoiron/sqlx"
)

type AuditLogRepo struct {
	db *sqlx.DB
}

func NewAuditLogRepo(db *sqlx.DB) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

func (r *AuditLogRepo) Create(ctx context.Context, log *platform.AuditLog) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_logs (tenant_id, user_id, action, resource_type, resource_id, detail, ip_address, user_agent) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		log.TenantID, log.UserID, log.Action, log.Resource, log.ResourceID, log.NewValue, log.IP, log.UserAgent)
	return err
}

func (r *AuditLogRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*platform.AuditLog, int64, error) {
	var total int64
	err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = ?`, tenantID)
	if err != nil {
		return nil, 0, err
	}

	var logs []*platform.AuditLog
	err = r.db.SelectContext(ctx, &logs,
		`SELECT id, tenant_id, user_id, action, resource_type AS resource, resource_id, ip_address AS ip, user_agent, created_at FROM audit_logs WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	return logs, total, err
}
