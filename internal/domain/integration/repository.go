package integration

import "context"

type DNCRepository interface {
	Create(ctx context.Context, entry *DNCEntry) error
	GetByNumber(ctx context.Context, tenantID int64, number string) (*DNCEntry, error)
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*DNCEntry, int64, error)
	Delete(ctx context.Context, id int64) error
	CheckNumbers(ctx context.Context, tenantID int64, numbers []string) ([]string, error)
}

type CallTagAssignmentRepository interface {
	Create(ctx context.Context, a *CallTagAssignment) error
	ListByCallID(ctx context.Context, callID int64) ([]*CallTagAssignment, error)
	Delete(ctx context.Context, id int64) error
}

type WebhookConfigRepository interface {
	Create(ctx context.Context, w *WebhookConfig) error
	GetByID(ctx context.Context, id int64) (*WebhookConfig, error)
	Update(ctx context.Context, w *WebhookConfig) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*WebhookConfig, int64, error)
	ListActiveByEvent(ctx context.Context, tenantID int64, eventType string) ([]*WebhookConfig, error)
}

type WebhookDeliveryLogRepository interface {
	Create(ctx context.Context, l *WebhookDeliveryLog) error
	List(ctx context.Context, webhookConfigID int64, offset, limit int) ([]*WebhookDeliveryLog, int64, error)
}

type ScreenPopConfigRepository interface {
	Create(ctx context.Context, s *ScreenPopConfig) error
	GetByID(ctx context.Context, id int64) (*ScreenPopConfig, error)
	Update(ctx context.Context, s *ScreenPopConfig) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*ScreenPopConfig, int64, error)
}

type QuickReplyRepository interface {
	Create(ctx context.Context, q *QuickReply) error
	GetByID(ctx context.Context, id int64) (*QuickReply, error)
	Update(ctx context.Context, q *QuickReply) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*QuickReply, int64, error)
	ListAvailable(ctx context.Context, tenantID int64, agentID, skillGroupID *int64) ([]*QuickReply, error)
}

type SmsConfigRepository interface {
	Create(ctx context.Context, s *SmsConfig) error
	GetByID(ctx context.Context, id int64) (*SmsConfig, error)
	Update(ctx context.Context, s *SmsConfig) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*SmsConfig, int64, error)
}
