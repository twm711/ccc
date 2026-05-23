package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/jmoiron/sqlx"
)

type SkillGroupRepo struct {
	db *sqlx.DB
}

func NewSkillGroupRepo(db *sqlx.DB) *SkillGroupRepo {
	return &SkillGroupRepo{db: db}
}

func (r *SkillGroupRepo) Create(ctx context.Context, sg *identity.SkillGroup) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO skill_groups (id, tenant_id, code, name, description, routing_policy, max_wait_sec, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sg.ID, sg.TenantID, sg.Code, sg.Name, sg.Description, sg.RoutingPolicy, sg.MaxWaitSec, sg.Status, sg.CreatedAt, sg.UpdatedAt)
	return err
}

func (r *SkillGroupRepo) GetByID(ctx context.Context, id int64) (*identity.SkillGroup, error) {
	var sg identity.SkillGroup
	err := r.db.GetContext(ctx, &sg,
		`SELECT id, tenant_id, code, name, description, routing_policy, max_wait_sec, status, created_at, updated_at FROM skill_groups WHERE id = ? AND status != 'deleted'`, id)
	if err != nil {
		return nil, err
	}
	return &sg, nil
}

func (r *SkillGroupRepo) GetByCode(ctx context.Context, tenantID int64, code string) (*identity.SkillGroup, error) {
	var sg identity.SkillGroup
	err := r.db.GetContext(ctx, &sg,
		`SELECT id, tenant_id, code, name, description, routing_policy, max_wait_sec, status, created_at, updated_at FROM skill_groups WHERE tenant_id = ? AND code = ? AND status != 'deleted'`,
		tenantID, code)
	if err != nil {
		return nil, err
	}
	return &sg, nil
}

func (r *SkillGroupRepo) Update(ctx context.Context, sg *identity.SkillGroup) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE skill_groups SET name = ?, description = ?, routing_policy = ?, max_wait_sec = ?, status = ?, updated_at = ? WHERE id = ?`,
		sg.Name, sg.Description, sg.RoutingPolicy, sg.MaxWaitSec, sg.Status, sg.UpdatedAt, sg.ID)
	return err
}

func (r *SkillGroupRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*identity.SkillGroup, int64, error) {
	var total int64
	err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM skill_groups WHERE tenant_id = ? AND status != 'deleted'`, tenantID)
	if err != nil {
		return nil, 0, err
	}

	var groups []*identity.SkillGroup
	err = r.db.SelectContext(ctx, &groups,
		`SELECT id, tenant_id, code, name, description, routing_policy, max_wait_sec, status, created_at, updated_at FROM skill_groups WHERE tenant_id = ? AND status != 'deleted' ORDER BY id LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	return groups, total, err
}

func (r *SkillGroupRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE skill_groups SET status = 'deleted', updated_at = NOW() WHERE id = ?`, id)
	return err
}

type SkillGroupMemberRepo struct {
	db *sqlx.DB
}

func NewSkillGroupMemberRepo(db *sqlx.DB) *SkillGroupMemberRepo {
	return &SkillGroupMemberRepo{db: db}
}

func (r *SkillGroupMemberRepo) Add(ctx context.Context, m *identity.SkillGroupMember) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO skill_group_members (skill_group_id, user_id, tenant_id, level, created_at) VALUES (?, ?, ?, ?, ?)`,
		m.SkillGroupID, m.AgentID, 0, m.Level, m.CreatedAt)
	return err
}

func (r *SkillGroupMemberRepo) Remove(ctx context.Context, skillGroupID, agentID int64) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM skill_group_members WHERE skill_group_id = ? AND user_id = ?`,
		skillGroupID, agentID)
	return err
}

func (r *SkillGroupMemberRepo) GetBySkillGroup(ctx context.Context, skillGroupID int64) ([]*identity.SkillGroupMember, error) {
	var members []*identity.SkillGroupMember
	err := r.db.SelectContext(ctx, &members,
		`SELECT skill_group_id, user_id AS agent_id, level, created_at FROM skill_group_members WHERE skill_group_id = ?`,
		skillGroupID)
	return members, err
}

func (r *SkillGroupMemberRepo) GetByAgent(ctx context.Context, agentID int64) ([]*identity.SkillGroupMember, error) {
	var members []*identity.SkillGroupMember
	err := r.db.SelectContext(ctx, &members,
		`SELECT skill_group_id, user_id AS agent_id, level, created_at FROM skill_group_members WHERE user_id = ?`,
		agentID)
	return members, err
}

func (r *SkillGroupMemberRepo) Exists(ctx context.Context, skillGroupID, agentID int64) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM skill_group_members WHERE skill_group_id = ? AND user_id = ?`,
		skillGroupID, agentID)
	return count > 0, err
}
