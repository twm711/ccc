package identity

import "context"

type TenantRepository interface {
	Create(ctx context.Context, t *Tenant) error
	GetByID(ctx context.Context, id int64) (*Tenant, error)
	GetByCode(ctx context.Context, code string) (*Tenant, error)
	Update(ctx context.Context, t *Tenant) error
	List(ctx context.Context, offset, limit int) ([]*Tenant, int64, error)
}

type TenantSettingsRepository interface {
	Upsert(ctx context.Context, s *TenantSettings) error
	GetByTenantID(ctx context.Context, tenantID int64) (*TenantSettings, error)
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByUsername(ctx context.Context, tenantID int64, username string) (*User, error)
	Update(ctx context.Context, u *User) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*User, int64, error)
}

type AgentRepository interface {
	Create(ctx context.Context, a *Agent) error
	GetByID(ctx context.Context, id int64) (*Agent, error)
	GetByUserID(ctx context.Context, userID int64) (*Agent, error)
	Update(ctx context.Context, a *Agent) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*Agent, int64, error)
}

type SkillGroupRepository interface {
	Create(ctx context.Context, sg *SkillGroup) error
	GetByID(ctx context.Context, id int64) (*SkillGroup, error)
	GetByCode(ctx context.Context, tenantID int64, code string) (*SkillGroup, error)
	Update(ctx context.Context, sg *SkillGroup) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*SkillGroup, int64, error)
	Delete(ctx context.Context, id int64) error
}

type SkillGroupMemberRepository interface {
	Add(ctx context.Context, m *SkillGroupMember) error
	Remove(ctx context.Context, skillGroupID, agentID int64) error
	GetBySkillGroup(ctx context.Context, skillGroupID int64) ([]*SkillGroupMember, error)
	GetByAgent(ctx context.Context, agentID int64) ([]*SkillGroupMember, error)
	Exists(ctx context.Context, skillGroupID, agentID int64) (bool, error)
}

type AgentPresenceRepository interface {
	Upsert(ctx context.Context, p *AgentPresence) error
	GetByAgentID(ctx context.Context, agentID int64) (*AgentPresence, error)
	ListByTenant(ctx context.Context, tenantID int64) ([]*AgentPresence, error)
}

type AgentPresenceLogRepository interface {
	Create(ctx context.Context, l *AgentPresenceLog) error
	ListByAgent(ctx context.Context, agentID int64, offset, limit int) ([]*AgentPresenceLog, int64, error)
}
