package identity

import (
	"context"
	"sync"
)

// In-memory mock repositories for testing.

type MockTenantRepo struct {
	mu      sync.RWMutex
	tenants map[int64]*Tenant
	byCode  map[string]*Tenant
}

func NewMockTenantRepo() *MockTenantRepo {
	return &MockTenantRepo{
		tenants: make(map[int64]*Tenant),
		byCode:  make(map[string]*Tenant),
	}
}

func (r *MockTenantRepo) Create(_ context.Context, t *Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tenants[t.ID] = t
	r.byCode[t.Code] = t
	return nil
}

func (r *MockTenantRepo) GetByID(_ context.Context, id int64) (*Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tenants[id], nil
}

func (r *MockTenantRepo) GetByCode(_ context.Context, code string) (*Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byCode[code], nil
}

func (r *MockTenantRepo) Update(_ context.Context, t *Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tenants[t.ID] = t
	r.byCode[t.Code] = t
	return nil
}

func (r *MockTenantRepo) List(_ context.Context, offset, limit int) ([]*Tenant, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]*Tenant, 0, len(r.tenants))
	for _, t := range r.tenants {
		all = append(all, t)
	}
	total := int64(len(all))
	if offset >= len(all) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

type MockTenantSettingsRepo struct {
	mu       sync.RWMutex
	settings map[int64]*TenantSettings
}

func NewMockTenantSettingsRepo() *MockTenantSettingsRepo {
	return &MockTenantSettingsRepo{settings: make(map[int64]*TenantSettings)}
}

func (r *MockTenantSettingsRepo) Upsert(_ context.Context, s *TenantSettings) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.settings[s.TenantID] = s
	return nil
}

func (r *MockTenantSettingsRepo) GetByTenantID(_ context.Context, tenantID int64) (*TenantSettings, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.settings[tenantID], nil
}

type MockUserRepo struct {
	mu    sync.RWMutex
	users map[int64]*User
}

func NewMockUserRepo() *MockUserRepo {
	return &MockUserRepo{users: make(map[int64]*User)}
}

func (r *MockUserRepo) Create(_ context.Context, u *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[u.ID] = u
	return nil
}

func (r *MockUserRepo) GetByID(_ context.Context, id int64) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.users[id], nil
}

func (r *MockUserRepo) GetByUsername(_ context.Context, tenantID int64, username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.TenantID == tenantID && u.Username == username {
			return u, nil
		}
	}
	return nil, nil
}

func (r *MockUserRepo) Update(_ context.Context, u *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[u.ID] = u
	return nil
}

func (r *MockUserRepo) List(_ context.Context, tenantID int64, offset, limit int) ([]*User, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var filtered []*User
	for _, u := range r.users {
		if u.TenantID == tenantID {
			filtered = append(filtered, u)
		}
	}
	total := int64(len(filtered))
	if offset >= len(filtered) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}

type MockAgentRepo struct {
	mu     sync.RWMutex
	agents map[int64]*Agent
}

func NewMockAgentRepo() *MockAgentRepo {
	return &MockAgentRepo{agents: make(map[int64]*Agent)}
}

func (r *MockAgentRepo) Create(_ context.Context, a *Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[a.ID] = a
	return nil
}

func (r *MockAgentRepo) GetByID(_ context.Context, id int64) (*Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agents[id], nil
}

func (r *MockAgentRepo) GetByUserID(_ context.Context, userID int64) (*Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.agents {
		if a.UserID == userID {
			return a, nil
		}
	}
	return nil, nil
}

func (r *MockAgentRepo) Update(_ context.Context, a *Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[a.ID] = a
	return nil
}

func (r *MockAgentRepo) List(_ context.Context, tenantID int64, offset, limit int) ([]*Agent, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var filtered []*Agent
	for _, a := range r.agents {
		if a.TenantID == tenantID {
			filtered = append(filtered, a)
		}
	}
	total := int64(len(filtered))
	if offset >= len(filtered) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}

type MockSkillGroupRepo struct {
	mu     sync.RWMutex
	groups map[int64]*SkillGroup
}

func NewMockSkillGroupRepo() *MockSkillGroupRepo {
	return &MockSkillGroupRepo{groups: make(map[int64]*SkillGroup)}
}

func (r *MockSkillGroupRepo) Create(_ context.Context, sg *SkillGroup) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.groups[sg.ID] = sg
	return nil
}

func (r *MockSkillGroupRepo) GetByID(_ context.Context, id int64) (*SkillGroup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.groups[id], nil
}

func (r *MockSkillGroupRepo) GetByCode(_ context.Context, tenantID int64, code string) (*SkillGroup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, sg := range r.groups {
		if sg.TenantID == tenantID && sg.Code == code {
			return sg, nil
		}
	}
	return nil, nil
}

func (r *MockSkillGroupRepo) Update(_ context.Context, sg *SkillGroup) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.groups[sg.ID] = sg
	return nil
}

func (r *MockSkillGroupRepo) List(_ context.Context, tenantID int64, offset, limit int) ([]*SkillGroup, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var filtered []*SkillGroup
	for _, sg := range r.groups {
		if sg.TenantID == tenantID {
			filtered = append(filtered, sg)
		}
	}
	total := int64(len(filtered))
	if offset >= len(filtered) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}

func (r *MockSkillGroupRepo) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.groups, id)
	return nil
}

type MockSkillGroupMemberRepo struct {
	mu      sync.RWMutex
	members map[int64]*SkillGroupMember
}

func NewMockSkillGroupMemberRepo() *MockSkillGroupMemberRepo {
	return &MockSkillGroupMemberRepo{members: make(map[int64]*SkillGroupMember)}
}

func (r *MockSkillGroupMemberRepo) Add(_ context.Context, m *SkillGroupMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.members[m.ID] = m
	return nil
}

func (r *MockSkillGroupMemberRepo) Remove(_ context.Context, skillGroupID, agentID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, m := range r.members {
		if m.SkillGroupID == skillGroupID && m.AgentID == agentID {
			delete(r.members, id)
			return nil
		}
	}
	return nil
}

func (r *MockSkillGroupMemberRepo) GetBySkillGroup(_ context.Context, skillGroupID int64) ([]*SkillGroupMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*SkillGroupMember
	for _, m := range r.members {
		if m.SkillGroupID == skillGroupID {
			result = append(result, m)
		}
	}
	return result, nil
}

func (r *MockSkillGroupMemberRepo) GetByAgent(_ context.Context, agentID int64) ([]*SkillGroupMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*SkillGroupMember
	for _, m := range r.members {
		if m.AgentID == agentID {
			result = append(result, m)
		}
	}
	return result, nil
}

func (r *MockSkillGroupMemberRepo) Exists(_ context.Context, skillGroupID, agentID int64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.members {
		if m.SkillGroupID == skillGroupID && m.AgentID == agentID {
			return true, nil
		}
	}
	return false, nil
}

// --- Mock AgentPresence Repos ---

type MockAgentPresenceRepo struct {
	mu   sync.RWMutex
	data map[int64]*AgentPresence
}

func NewMockAgentPresenceRepo() *MockAgentPresenceRepo {
	return &MockAgentPresenceRepo{data: make(map[int64]*AgentPresence)}
}

func (r *MockAgentPresenceRepo) Upsert(_ context.Context, p *AgentPresence) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[p.AgentID] = p
	return nil
}

func (r *MockAgentPresenceRepo) GetByAgentID(_ context.Context, agentID int64) (*AgentPresence, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.data[agentID], nil
}

func (r *MockAgentPresenceRepo) ListByTenant(_ context.Context, tenantID int64) ([]*AgentPresence, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*AgentPresence
	for _, p := range r.data {
		if p.TenantID == tenantID {
			result = append(result, p)
		}
	}
	return result, nil
}

type MockAgentPresenceLogRepo struct {
	mu   sync.RWMutex
	logs []*AgentPresenceLog
}

func NewMockAgentPresenceLogRepo() *MockAgentPresenceLogRepo {
	return &MockAgentPresenceLogRepo{}
}

func (r *MockAgentPresenceLogRepo) Create(_ context.Context, l *AgentPresenceLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, l)
	return nil
}

func (r *MockAgentPresenceLogRepo) ListByAgent(_ context.Context, agentID int64, offset, limit int) ([]*AgentPresenceLog, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*AgentPresenceLog
	for _, l := range r.logs {
		if l.AgentID == agentID {
			result = append(result, l)
		}
	}
	total := int64(len(result))
	if offset < len(result) {
		end := offset + limit
		if end > len(result) {
			end = len(result)
		}
		result = result[offset:end]
	} else {
		result = nil
	}
	return result, total, nil
}
