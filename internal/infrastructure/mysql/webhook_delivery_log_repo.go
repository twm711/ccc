package mysql

import (
	"context"
	"time"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/jmoiron/sqlx"
)

type WebhookDeliveryLogRepo struct{ db *sqlx.DB }

func NewWebhookDeliveryLogRepo(db *sqlx.DB) *WebhookDeliveryLogRepo {
	return &WebhookDeliveryLogRepo{db: db}
}

func (r *WebhookDeliveryLogRepo) Create(ctx context.Context, l *integration.WebhookDeliveryLog) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO webhook_delivery_log (id, tenant_id, webhook_config_id, event_type, payload, response_status, response_body, attempt_count, success, error_message, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.TenantID, l.WebhookConfigID, l.EventType, l.Payload, l.ResponseStatus, l.ResponseBody, l.AttemptCount, l.Success, l.ErrorMessage, l.CreatedAt)
	return err
}

func (r *WebhookDeliveryLogRepo) List(ctx context.Context, webhookConfigID int64, offset, limit int) ([]*integration.WebhookDeliveryLog, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM webhook_delivery_log WHERE webhook_config_id = ?`, webhookConfigID)
	var items []*integration.WebhookDeliveryLog
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM webhook_delivery_log WHERE webhook_config_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		webhookConfigID, limit, offset)
	return items, total, err
}

func (r *WebhookDeliveryLogRepo) ListFailed(ctx context.Context, tenantID int64, offset, limit int) ([]*integration.WebhookDeliveryLog, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM webhook_delivery_log WHERE tenant_id = ? AND success = false`, tenantID)
	var items []*integration.WebhookDeliveryLog
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM webhook_delivery_log WHERE tenant_id = ? AND success = false ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	return items, total, err
}

func (r *WebhookDeliveryLogRepo) GetByID(ctx context.Context, id int64) (*integration.WebhookDeliveryLog, error) {
	var l integration.WebhookDeliveryLog
	err := r.db.GetContext(ctx, &l, `SELECT * FROM webhook_delivery_log WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *WebhookDeliveryLogRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM webhook_delivery_log WHERE id = ?`, id)
	return err
}

func (r *WebhookDeliveryLogRepo) PurgeBefore(ctx context.Context, tenantID int64, before time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM webhook_delivery_log WHERE tenant_id = ? AND success = false AND created_at < ?`, tenantID, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
