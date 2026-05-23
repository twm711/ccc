package im

import "context"

type IMChannelRepository interface {
	Create(ctx context.Context, c *IMChannel) error
	GetByID(ctx context.Context, id int64) (*IMChannel, error)
	Update(ctx context.Context, c *IMChannel) error
	List(ctx context.Context, tenantID int64) ([]*IMChannel, error)
}

type IMSessionRepository interface {
	Create(ctx context.Context, s *IMSession) error
	GetByID(ctx context.Context, id int64) (*IMSession, error)
	Update(ctx context.Context, s *IMSession) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*IMSession, error)
	CountActiveByAgent(ctx context.Context, agentUserID int64) (int, error)
}

type IMMessageRepository interface {
	Create(ctx context.Context, m *IMMessage) error
	ListBySession(ctx context.Context, sessionID int64, offset, limit int) ([]*IMMessage, error)
}
