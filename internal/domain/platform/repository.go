package platform

import "context"

type AuditLogRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*AuditLog, int64, error)
}
