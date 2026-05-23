package platform

import "time"

type AuditLog struct {
	ID         int64     `db:"id" json:"id"`
	TenantID   int64     `db:"tenant_id" json:"tenant_id"`
	UserID     int64     `db:"user_id" json:"user_id"`
	Action     string    `db:"action" json:"action"`
	Resource   string    `db:"resource" json:"resource"`
	ResourceID string    `db:"resource_id" json:"resource_id"`
	OldValue   string    `db:"old_value" json:"old_value,omitempty"`
	NewValue   string    `db:"new_value" json:"new_value,omitempty"`
	IP         string    `db:"ip" json:"ip"`
	UserAgent  string    `db:"user_agent" json:"user_agent"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
