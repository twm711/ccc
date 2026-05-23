package mysql

import (
	"context"
	"database/sql"

	"github.com/divord97/ccc/internal/domain/im"
	"github.com/jmoiron/sqlx"
)

// --- IMChannel ---

type IMChannelRepo struct{ db *sqlx.DB }

func NewIMChannelRepo(db *sqlx.DB) *IMChannelRepo { return &IMChannelRepo{db: db} }

func (r *IMChannelRepo) Create(ctx context.Context, c *im.IMChannel) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO im_channels (id, tenant_id, channel_type, name, config, skill_group_id, status, created_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
		c.ID, c.TenantID, c.ChannelType, c.Name, c.Config, c.SkillGroupID, c.Status, c.CreatedAt)
	return err
}

func (r *IMChannelRepo) GetByID(ctx context.Context, id int64) (*im.IMChannel, error) {
	var c im.IMChannel
	err := r.db.GetContext(ctx, &c, "SELECT * FROM im_channels WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *IMChannelRepo) Update(ctx context.Context, c *im.IMChannel) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE im_channels SET name=?, config=?, skill_group_id=?, status=? WHERE id=?`,
		c.Name, c.Config, c.SkillGroupID, c.Status, c.ID)
	return err
}

func (r *IMChannelRepo) List(ctx context.Context, tenantID int64) ([]*im.IMChannel, error) {
	var items []*im.IMChannel
	err := r.db.SelectContext(ctx, &items, "SELECT * FROM im_channels WHERE tenant_id = ? ORDER BY created_at DESC", tenantID)
	return items, err
}

// --- IMSession ---

type IMSessionRepo struct{ db *sqlx.DB }

func NewIMSessionRepo(db *sqlx.DB) *IMSessionRepo { return &IMSessionRepo{db: db} }

func (r *IMSessionRepo) Create(ctx context.Context, s *im.IMSession) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO im_sessions (id, tenant_id, channel_id, visitor_id, customer_id, agent_user_id, skill_group_id, status, csat_score, start_at, end_at, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		s.ID, s.TenantID, s.ChannelID, s.VisitorID, s.CustomerID, s.AgentUserID, s.SkillGroupID, s.Status, s.CSATScore, s.StartAt, s.EndAt, s.CreatedAt)
	return err
}

func (r *IMSessionRepo) GetByID(ctx context.Context, id int64) (*im.IMSession, error) {
	var s im.IMSession
	err := r.db.GetContext(ctx, &s, "SELECT * FROM im_sessions WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

func (r *IMSessionRepo) Update(ctx context.Context, s *im.IMSession) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE im_sessions SET agent_user_id=?, skill_group_id=?, status=?, csat_score=?, end_at=? WHERE id=?`,
		s.AgentUserID, s.SkillGroupID, s.Status, s.CSATScore, s.EndAt, s.ID)
	return err
}

func (r *IMSessionRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*im.IMSession, error) {
	var items []*im.IMSession
	err := r.db.SelectContext(ctx, &items,
		"SELECT * FROM im_sessions WHERE tenant_id = ? ORDER BY start_at DESC LIMIT ? OFFSET ?",
		tenantID, limit, offset)
	return items, err
}

func (r *IMSessionRepo) CountActiveByAgent(ctx context.Context, agentUserID int64) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM im_sessions WHERE agent_user_id = ? AND status = 'active'", agentUserID)
	return count, err
}

// --- IMMessage ---

type IMMessageRepo struct{ db *sqlx.DB }

func NewIMMessageRepo(db *sqlx.DB) *IMMessageRepo { return &IMMessageRepo{db: db} }

func (r *IMMessageRepo) Create(ctx context.Context, m *im.IMMessage) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO im_messages (session_id, sender_type, sender_id, content_type, content, created_at)
		 VALUES (?,?,?,?,?,?)`,
		m.SessionID, m.SenderType, m.SenderID, m.ContentType, m.Content, m.CreatedAt)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	m.ID = id
	return nil
}

func (r *IMMessageRepo) ListBySession(ctx context.Context, sessionID int64, offset, limit int) ([]*im.IMMessage, error) {
	var items []*im.IMMessage
	err := r.db.SelectContext(ctx, &items,
		"SELECT * FROM im_messages WHERE session_id = ? ORDER BY created_at ASC LIMIT ? OFFSET ?",
		sessionID, limit, offset)
	return items, err
}
