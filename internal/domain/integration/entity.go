package integration

import "time"

type DNCEntry struct {
	ID        int64     `db:"id" json:"id"`
	TenantID  int64     `db:"tenant_id" json:"tenant_id"`
	Number    string    `db:"number" json:"number"`
	Reason    string    `db:"reason" json:"reason"`
	Source    string    `db:"source" json:"source"` // manual, import, api
	ExpiresAt *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type CallTagAssignment struct {
	ID        int64     `db:"id" json:"id"`
	TenantID  int64     `db:"tenant_id" json:"tenant_id"`
	CallID    int64     `db:"call_id" json:"call_id"`
	TagID     int64     `db:"tag_id" json:"tag_id"`
	TagName   string    `db:"tag_name" json:"tag_name"`
	CreatedBy *int64    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type WebhookConfig struct {
	ID        int64     `db:"id" json:"id"`
	TenantID  int64     `db:"tenant_id" json:"tenant_id"`
	Name      string    `db:"name" json:"name"`
	URL       string    `db:"url" json:"url"`
	Secret    string    `db:"secret" json:"secret,omitempty"`
	Events    string    `db:"events" json:"events"` // comma-separated event types
	IsActive  bool      `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type WebhookDeliveryLog struct {
	ID              int64     `db:"id" json:"id"`
	TenantID        int64     `db:"tenant_id" json:"tenant_id"`
	WebhookConfigID int64     `db:"webhook_config_id" json:"webhook_config_id"`
	EventType       string    `db:"event_type" json:"event_type"`
	Payload         string    `db:"payload" json:"payload"`
	ResponseStatus  int       `db:"response_status" json:"response_status"`
	ResponseBody    string    `db:"response_body" json:"response_body,omitempty"`
	AttemptCount    int       `db:"attempt_count" json:"attempt_count"`
	Success         bool      `db:"success" json:"success"`
	ErrorMessage    string    `db:"error_message" json:"error_message,omitempty"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

type ScreenPopConfig struct {
	ID           int64     `db:"id" json:"id"`
	TenantID     int64     `db:"tenant_id" json:"tenant_id"`
	Name         string    `db:"name" json:"name"`
	URLTemplate  string    `db:"url_template" json:"url_template"`
	Position     int       `db:"position" json:"position"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type QuickReplyScope string

const (
	QuickReplyScopeGlobal     QuickReplyScope = "global"
	QuickReplyScopeSkillGroup QuickReplyScope = "skill_group"
	QuickReplyScopeAgent      QuickReplyScope = "agent"
)

type QuickReply struct {
	ID           int64           `db:"id" json:"id"`
	TenantID     int64           `db:"tenant_id" json:"tenant_id"`
	Scope        QuickReplyScope `db:"scope" json:"scope"`
	ScopeID      *int64          `db:"scope_id" json:"scope_id,omitempty"`
	Title        string          `db:"title" json:"title"`
	Content      string          `db:"content" json:"content"`
	Shortcut     string          `db:"shortcut" json:"shortcut,omitempty"`
	SortOrder    int             `db:"sort_order" json:"sort_order"`
	IsActive     bool            `db:"is_active" json:"is_active"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
}

type SmsConfig struct {
	ID          int64     `db:"id" json:"id"`
	TenantID    int64     `db:"tenant_id" json:"tenant_id"`
	Provider    string    `db:"provider" json:"provider"` // aliyun
	AccessKeyID string    `db:"access_key_id" json:"access_key_id"`
	SignName    string    `db:"sign_name" json:"sign_name"`
	TemplateMap string    `db:"template_map" json:"template_map"` // JSON: {"verification": "SMS_12345"}
	IsActive    bool      `db:"is_active" json:"is_active"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}
