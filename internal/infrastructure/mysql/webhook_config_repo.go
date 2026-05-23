package mysql

import (
	"context"
	"strings"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/jmoiron/sqlx"
)

type WebhookConfigRepo struct{ db *sqlx.DB }

func NewWebhookConfigRepo(db *sqlx.DB) *WebhookConfigRepo { return &WebhookConfigRepo{db: db} }

func (r *WebhookConfigRepo) Create(ctx context.Context, w *integration.WebhookConfig) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO webhook_configs (id, tenant_id, name, url, secret, events, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.TenantID, w.Name, w.URL, w.Secret, w.Events, w.IsActive, w.CreatedAt, w.UpdatedAt)
	return err
}

func (r *WebhookConfigRepo) GetByID(ctx context.Context, id int64) (*integration.WebhookConfig, error) {
	var w integration.WebhookConfig
	err := r.db.GetContext(ctx, &w, `SELECT * FROM webhook_configs WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WebhookConfigRepo) Update(ctx context.Context, w *integration.WebhookConfig) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE webhook_configs SET name=?, url=?, secret=?, events=?, is_active=?, updated_at=? WHERE id=?`,
		w.Name, w.URL, w.Secret, w.Events, w.IsActive, w.UpdatedAt, w.ID)
	return err
}

func (r *WebhookConfigRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM webhook_configs WHERE id = ?`, id)
	return err
}

func (r *WebhookConfigRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*integration.WebhookConfig, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM webhook_configs WHERE tenant_id = ?`, tenantID)
	var items []*integration.WebhookConfig
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM webhook_configs WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	return items, total, err
}

func (r *WebhookConfigRepo) ListActiveByEvent(ctx context.Context, tenantID int64, eventType string) ([]*integration.WebhookConfig, error) {
	var all []*integration.WebhookConfig
	err := r.db.SelectContext(ctx, &all,
		`SELECT * FROM webhook_configs WHERE tenant_id = ? AND is_active = true`, tenantID)
	if err != nil {
		return nil, err
	}
	var matched []*integration.WebhookConfig
	for _, w := range all {
		if w.Events == "*" {
			matched = append(matched, w)
			continue
		}
		for _, e := range strings.Split(w.Events, ",") {
			if strings.TrimSpace(e) == eventType {
				matched = append(matched, w)
				break
			}
		}
	}
	return matched, nil
}
