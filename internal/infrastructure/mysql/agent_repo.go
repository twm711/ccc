package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/jmoiron/sqlx"
)

type AgentRepo struct {
	db *sqlx.DB
}

func NewAgentRepo(db *sqlx.DB) *AgentRepo {
	return &AgentRepo{db: db}
}

func (r *AgentRepo) Create(ctx context.Context, a *identity.Agent) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO agents (user_id, tenant_id, extension, max_concurrent, max_chat_slots, acw_seconds, outbound_only, personal_outbound_number_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.UserID, a.TenantID, a.Extension, a.MaxConcurrent, a.MaxChatSlots, a.ACWSeconds, a.OutboundOnly, a.PersonalOutboundNumberID, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *AgentRepo) GetByID(ctx context.Context, id int64) (*identity.Agent, error) {
	var a identity.Agent
	err := r.db.GetContext(ctx, &a,
		`SELECT user_id, tenant_id, extension, max_concurrent, max_chat_slots, acw_seconds, outbound_only, personal_outbound_number_id, created_at, updated_at FROM agents WHERE user_id = ?`, id)
	if err != nil {
		return nil, err
	}
	a.ID = a.UserID
	return &a, nil
}

func (r *AgentRepo) GetByUserID(ctx context.Context, userID int64) (*identity.Agent, error) {
	return r.GetByID(ctx, userID)
}

func (r *AgentRepo) Update(ctx context.Context, a *identity.Agent) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE agents SET extension = ?, max_concurrent = ?, max_chat_slots = ?, acw_seconds = ?, outbound_only = ?, personal_outbound_number_id = ?, updated_at = ? WHERE user_id = ?`,
		a.Extension, a.MaxConcurrent, a.MaxChatSlots, a.ACWSeconds, a.OutboundOnly, a.PersonalOutboundNumberID, a.UpdatedAt, a.UserID)
	return err
}

func (r *AgentRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*identity.Agent, int64, error) {
	var total int64
	err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM agents WHERE tenant_id = ?`, tenantID)
	if err != nil {
		return nil, 0, err
	}

	var agents []*identity.Agent
	err = r.db.SelectContext(ctx, &agents,
		`SELECT user_id, tenant_id, extension, max_concurrent, max_chat_slots, acw_seconds, outbound_only, personal_outbound_number_id, created_at, updated_at FROM agents WHERE tenant_id = ? ORDER BY user_id LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	return agents, total, err
}
