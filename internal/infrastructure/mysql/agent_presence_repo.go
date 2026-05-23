package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/jmoiron/sqlx"
)

type AgentPresenceRepo struct{ db *sqlx.DB }

func NewAgentPresenceRepo(db *sqlx.DB) *AgentPresenceRepo {
	return &AgentPresenceRepo{db: db}
}

func (r *AgentPresenceRepo) Upsert(ctx context.Context, p *identity.AgentPresence) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO agent_presence (id, tenant_id, agent_id, status, sub_state, work_mode, break_reason_code, disposition_code, current_call_id, checked_in_at, last_status_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE status=VALUES(status), sub_state=VALUES(sub_state), work_mode=VALUES(work_mode),
		 break_reason_code=VALUES(break_reason_code), disposition_code=VALUES(disposition_code),
		 current_call_id=VALUES(current_call_id), last_status_at=VALUES(last_status_at), updated_at=VALUES(updated_at)`,
		p.ID, p.TenantID, p.AgentID, p.Status, p.SubState, p.WorkMode, p.BreakReasonCode, p.DispositionCode, p.CurrentCallID, p.CheckedInAt, p.LastStatusAt, p.UpdatedAt)
	return err
}

func (r *AgentPresenceRepo) GetByAgentID(ctx context.Context, agentID int64) (*identity.AgentPresence, error) {
	var p identity.AgentPresence
	err := r.db.GetContext(ctx, &p, `SELECT * FROM agent_presence WHERE agent_id = ?`, agentID)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *AgentPresenceRepo) ListByTenant(ctx context.Context, tenantID int64) ([]*identity.AgentPresence, error) {
	var items []*identity.AgentPresence
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM agent_presence WHERE tenant_id = ?`, tenantID)
	return items, err
}

type AgentPresenceLogRepo struct{ db *sqlx.DB }

func NewAgentPresenceLogRepo(db *sqlx.DB) *AgentPresenceLogRepo {
	return &AgentPresenceLogRepo{db: db}
}

func (r *AgentPresenceLogRepo) Create(ctx context.Context, l *identity.AgentPresenceLog) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO agent_presence_log (id, tenant_id, agent_id, status, sub_state, work_mode, break_reason_code, duration_sec, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.TenantID, l.AgentID, l.Status, l.SubState, l.WorkMode, l.BreakReasonCode, l.DurationSec, l.CreatedAt)
	return err
}

func (r *AgentPresenceLogRepo) ListByAgent(ctx context.Context, agentID int64, offset, limit int) ([]*identity.AgentPresenceLog, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM agent_presence_log WHERE agent_id = ?`, agentID)
	var items []*identity.AgentPresenceLog
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM agent_presence_log WHERE agent_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		agentID, limit, offset)
	return items, total, err
}
