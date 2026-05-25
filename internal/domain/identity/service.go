package identity

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unicode"

	"github.com/divord97/ccc/pkg/snowflake"
	"golang.org/x/crypto/bcrypt"
)

// validatePasswordStrength enforces minimum password complexity:
// at least 8 characters, contains at least one letter and one digit.
func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("password must contain at least one letter and one digit")
	}
	return nil
}

type TenantService struct {
	tenants  TenantRepository
	settings TenantSettingsRepository
}

func NewTenantService(tr TenantRepository, sr TenantSettingsRepository) *TenantService {
	return &TenantService{tenants: tr, settings: sr}
}

type CreateTenantInput struct {
	Code string
	Name string
}

func (s *TenantService) Create(ctx context.Context, in CreateTenantInput) (*Tenant, error) {
	existing, _ := s.tenants.GetByCode(ctx, in.Code)
	if existing != nil {
		return nil, ErrTenantCodeExists
	}

	now := time.Now()
	t := &Tenant{
		ID:        snowflake.NextID(),
		Code:      in.Code,
		Name:      in.Name,
		Status:    TenantStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.tenants.Create(ctx, t); err != nil {
		return nil, err
	}

	defaults := &TenantSettings{
		TenantID:                t.ID,
		MaxAgents:               50,
		MaxConcurrentCalls:      100,
		RecordingRetentionDays:  365,
		RecordingStorageBackend: "local",
		Timezone:                "Asia/Shanghai",
		Language:                "zh-CN",
		DefaultACWSeconds:       30,
		APIRateLimitPerSec:      100,
		FamiliarAgentDays:       30,
	}
	if err := s.settings.Upsert(ctx, defaults); err != nil {
		return nil, err
	}

	return t, nil
}

func (s *TenantService) GetByID(ctx context.Context, id int64) (*Tenant, error) {
	return s.tenants.GetByID(ctx, id)
}

func (s *TenantService) Update(ctx context.Context, id int64, name string, status TenantStatus) (*Tenant, error) {
	t, err := s.tenants.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTenantNotFound
	}
	t.Name = name
	t.Status = status
	t.UpdatedAt = time.Now()
	if err := s.tenants.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TenantService) List(ctx context.Context, offset, limit int) ([]*Tenant, int64, error) {
	return s.tenants.List(ctx, offset, limit)
}

type UserService struct {
	users  UserRepository
	agents AgentRepository
}

func NewUserService(ur UserRepository, ar AgentRepository) *UserService {
	return &UserService{users: ur, agents: ar}
}

type CreateUserInput struct {
	TenantID    int64
	Username    string
	DisplayName string
	Email       string
	Phone       string
	Role        UserRole
}

func (s *UserService) Create(ctx context.Context, in CreateUserInput) (*User, error) {
	existing, _ := s.users.GetByUsername(ctx, in.TenantID, in.Username)
	if existing != nil {
		return nil, ErrUsernameExists
	}

	now := time.Now()
	u := &User{
		ID:          snowflake.NextID(),
		TenantID:    in.TenantID,
		Username:    in.Username,
		DisplayName: in.DisplayName,
		Email:       in.Email,
		Phone:       in.Phone,
		Role:        in.Role,
		Status:      UserStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*User, error) {
	return s.users.GetByID(ctx, id)
}

func (s *UserService) Update(ctx context.Context, id int64, displayName, email, phone string) (*User, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	u.DisplayName = displayName
	u.Email = email
	u.Phone = phone
	u.UpdatedAt = time.Now()
	if err := s.users.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) List(ctx context.Context, tenantID int64, offset, limit int) ([]*User, int64, error) {
	return s.users.List(ctx, tenantID, offset, limit)
}

func (s *UserService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUserNotFound
	}
	if u.PasswordHash == "" {
		return ErrPasswordNotSet
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrWrongPassword
	}
	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}
	return s.users.UpdatePassword(ctx, userID, string(hash))
}

type AgentService struct {
	agents   AgentRepository
	users    UserRepository
	settings TenantSettingsRepository
}

func NewAgentService(ar AgentRepository, ur UserRepository, sr TenantSettingsRepository) *AgentService {
	return &AgentService{agents: ar, users: ur, settings: sr}
}

type CreateAgentInput struct {
	TenantID     int64
	UserID       int64
	EmployeeID   string
	Extension    string
	WorkMode     WorkMode
	MaxChatSlots int
	ACWSeconds   int
}

func (s *AgentService) Create(ctx context.Context, in CreateAgentInput) (*Agent, error) {
	existing, _ := s.agents.GetByUserID(ctx, in.UserID)
	if existing != nil {
		return nil, ErrAgentAlreadyExists
	}

	settings, err := s.settings.GetByTenantID(ctx, in.TenantID)
	if err != nil {
		return nil, err
	}

	if settings != nil {
		agents, count, err := s.agents.List(ctx, in.TenantID, 0, 1)
		_ = agents
		if err != nil {
			return nil, err
		}
		if int(count) >= settings.MaxAgents {
			return nil, ErrMaxAgentsReached
		}
	}

	now := time.Now()
	a := &Agent{
		ID:              snowflake.NextID(),
		TenantID:        in.TenantID,
		UserID:          in.UserID,
		EmployeeID:      in.EmployeeID,
		Extension:       in.Extension,
		WorkMode:        in.WorkMode,
		SIPDeviceStatus: "unregistered",
		MaxConcurrent:   1,
		MaxChatSlots:    in.MaxChatSlots,
		ACWSeconds:      in.ACWSeconds,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if a.MaxChatSlots == 0 {
		a.MaxChatSlots = 3
	}
	if a.ACWSeconds == 0 {
		a.ACWSeconds = 15
	}
	if err := s.agents.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AgentService) GetByID(ctx context.Context, id int64) (*Agent, error) {
	return s.agents.GetByID(ctx, id)
}

func (s *AgentService) List(ctx context.Context, tenantID int64, offset, limit int) ([]*Agent, int64, error) {
	return s.agents.List(ctx, tenantID, offset, limit)
}

type SkillGroupService struct {
	groups  SkillGroupRepository
	members SkillGroupMemberRepository
}

func NewSkillGroupService(gr SkillGroupRepository, mr SkillGroupMemberRepository) *SkillGroupService {
	return &SkillGroupService{groups: gr, members: mr}
}

type CreateSkillGroupInput struct {
	TenantID      int64
	Code          string
	Name          string
	Description   string
	RoutingPolicy RoutingPolicy
	Priority      int
	MaxWaitSec    int
}

func (s *SkillGroupService) Create(ctx context.Context, in CreateSkillGroupInput) (*SkillGroup, error) {
	existing, _ := s.groups.GetByCode(ctx, in.TenantID, in.Code)
	if existing != nil {
		return nil, ErrSkillGroupCodeExists
	}

	validPolicies := map[RoutingPolicy]bool{
		RoutingPolicyRoundRobin:  true,
		RoutingPolicyLeastRecent: true,
		RoutingPolicyRandom:      true,
		RoutingPolicySkillWeight: true,
		RoutingPolicyFamiliar:    true,
	}
	if !validPolicies[in.RoutingPolicy] {
		return nil, ErrInvalidRoutingPolicy
	}

	now := time.Now()
	sg := &SkillGroup{
		ID:            snowflake.NextID(),
		TenantID:      in.TenantID,
		Code:          in.Code,
		Name:          in.Name,
		Description:   in.Description,
		RoutingPolicy: in.RoutingPolicy,
		Priority:      in.Priority,
		MaxWaitSec:    in.MaxWaitSec,
		Status:        SkillGroupStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if sg.MaxWaitSec == 0 {
		sg.MaxWaitSec = 60
	}
	if err := s.groups.Create(ctx, sg); err != nil {
		return nil, err
	}
	return sg, nil
}

func (s *SkillGroupService) GetByID(ctx context.Context, id int64) (*SkillGroup, error) {
	return s.groups.GetByID(ctx, id)
}

func (s *SkillGroupService) List(ctx context.Context, tenantID int64, offset, limit int) ([]*SkillGroup, int64, error) {
	return s.groups.List(ctx, tenantID, offset, limit)
}

func (s *SkillGroupService) AddMember(ctx context.Context, skillGroupID, agentID int64, level int) (*SkillGroupMember, error) {
	exists, err := s.members.Exists(ctx, skillGroupID, agentID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrMemberAlreadyExists
	}
	m := &SkillGroupMember{
		ID:           snowflake.NextID(),
		SkillGroupID: skillGroupID,
		AgentID:      agentID,
		Level:        level,
		CreatedAt:    time.Now(),
	}
	if err := s.members.Add(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *SkillGroupService) RemoveMember(ctx context.Context, skillGroupID, agentID int64) error {
	return s.members.Remove(ctx, skillGroupID, agentID)
}

func (s *SkillGroupService) GetMembers(ctx context.Context, skillGroupID int64) ([]*SkillGroupMember, error) {
	return s.members.GetBySkillGroup(ctx, skillGroupID)
}

// --- AgentPresenceService ---

type AgentPresenceService struct {
	presence    AgentPresenceRepository
	logs        AgentPresenceLogRepository
	shifts      AgentShiftLogRepository
	acwTimers   map[int64]func()
	acwResolver func(ctx context.Context, tenantID, agentID int64) int
	mu          sync.Mutex
}

func NewAgentPresenceService(pr AgentPresenceRepository, lr AgentPresenceLogRepository) *AgentPresenceService {
	return &AgentPresenceService{presence: pr, logs: lr, acwTimers: make(map[int64]func())}
}

// SetShiftLogRepo wires the shift log repository for check-in/check-out auditing.
func (s *AgentPresenceService) SetShiftLogRepo(r AgentShiftLogRepository) {
	s.shifts = r
}

// SetACWResolver wires a callback that returns the effective ACW seconds for
// an agent (typically agent.acw_seconds, falling back to
// tenant_settings.default_acw_seconds). Without a resolver, SetACW uses the
// presence row's stored ACWSeconds, which is only correct if it was populated
// elsewhere; with a resolver, the value is refreshed on every ACW transition.
func (s *AgentPresenceService) SetACWResolver(fn func(ctx context.Context, tenantID, agentID int64) int) {
	s.acwResolver = fn
}

// validTransitions defines allowed state transitions.
var validTransitions = map[AgentPresenceStatus][]AgentPresenceStatus{
	PresenceOffline: {PresenceOnline},
	PresenceOnline:  {PresenceIdle, PresenceOffline},
	PresenceIdle:    {PresenceDialing, PresenceTalking, PresenceBreak, PresenceOffline},
	PresenceDialing: {PresenceTalking, PresenceACW, PresenceIdle, PresenceOffline},
	PresenceTalking: {PresenceACW, PresenceIdle, PresenceOffline},
	PresenceACW:     {PresenceIdle, PresenceOffline},
	PresenceBreak:   {PresenceIdle, PresenceOffline},
}

func isValidTransition(from, to AgentPresenceStatus) bool {
	for _, allowed := range validTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

func (s *AgentPresenceService) CheckIn(ctx context.Context, tenantID, agentID int64, workMode WorkMode) (*AgentPresence, error) {
	now := time.Now()
	p := &AgentPresence{
		ID:           snowflake.NextID(),
		TenantID:     tenantID,
		AgentID:      agentID,
		Status:       PresenceOnline,
		SubState:     SubStateNone,
		WorkMode:     workMode,
		CheckedInAt:  &now,
		LastStatusAt: now,
		UpdatedAt:    now,
	}
	if err := s.presence.Upsert(ctx, p); err != nil {
		return nil, err
	}
	if s.shifts != nil {
		_ = s.shifts.Create(ctx, &AgentShiftLog{
			ID:        snowflake.NextID(),
			TenantID:  tenantID,
			AgentID:   agentID,
			ShiftDate: now.Format("2006-01-02"),
			CheckInAt: now,
		})
	}
	return p, nil
}

func (s *AgentPresenceService) CheckOut(ctx context.Context, agentID int64) error {
	p, err := s.presence.GetByAgentID(ctx, agentID)
	if err != nil || p == nil {
		return ErrPresenceNotFound
	}
	s.logTransition(ctx, p)
	now := time.Now()
	if s.shifts != nil {
		if shift, err := s.shifts.GetOpenShift(ctx, agentID); err == nil && shift != nil {
			dur := int(now.Sub(shift.CheckInAt).Seconds())
			_ = s.shifts.EndShift(ctx, shift.ID, now, dur)
		}
	}
	p.Status = PresenceOffline
	p.SubState = SubStateNone
	p.CurrentCallID = nil
	p.UpdatedAt = now
	p.LastStatusAt = now
	return s.presence.Upsert(ctx, p)
}

func (s *AgentPresenceService) TransitionTo(ctx context.Context, agentID int64, newStatus AgentPresenceStatus) (*AgentPresence, error) {
	p, err := s.presence.GetByAgentID(ctx, agentID)
	if err != nil || p == nil {
		return nil, ErrPresenceNotFound
	}
	if !isValidTransition(p.Status, newStatus) {
		return nil, ErrInvalidStateTransition
	}
	s.logTransition(ctx, p)
	s.cancelACWTimer(agentID)
	p.Status = newStatus
	if newStatus != PresenceTalking {
		p.SubState = SubStateNone
	}
	if newStatus == PresenceIdle || newStatus == PresenceOffline {
		p.CurrentCallID = nil
		p.DispositionCode = ""
		p.BreakReasonCode = ""
	}
	p.UpdatedAt = time.Now()
	p.LastStatusAt = p.UpdatedAt
	if err := s.presence.Upsert(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *AgentPresenceService) SetSubState(ctx context.Context, agentID int64, subState AgentSubState) (*AgentPresence, error) {
	p, err := s.presence.GetByAgentID(ctx, agentID)
	if err != nil || p == nil {
		return nil, ErrPresenceNotFound
	}
	p.SubState = subState
	p.UpdatedAt = time.Now()
	if err := s.presence.Upsert(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *AgentPresenceService) SwitchWorkMode(ctx context.Context, agentID int64, mode WorkMode) (*AgentPresence, error) {
	if mode != WorkModeOnSite && mode != WorkModeOffSite && mode != WorkModeOffice {
		return nil, ErrInvalidWorkMode
	}
	p, err := s.presence.GetByAgentID(ctx, agentID)
	if err != nil || p == nil {
		return nil, ErrPresenceNotFound
	}
	p.WorkMode = mode
	p.UpdatedAt = time.Now()
	if err := s.presence.Upsert(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *AgentPresenceService) SetBreak(ctx context.Context, agentID int64, reasonCode string) (*AgentPresence, error) {
	p, err := s.TransitionTo(ctx, agentID, PresenceBreak)
	if err != nil {
		return nil, err
	}
	p.BreakReasonCode = reasonCode
	p.UpdatedAt = time.Now()
	if err := s.presence.Upsert(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *AgentPresenceService) SetACW(ctx context.Context, agentID int64, dispositionCode string) (*AgentPresence, error) {
	p, err := s.TransitionTo(ctx, agentID, PresenceACW)
	if err != nil {
		return nil, err
	}
	p.DispositionCode = dispositionCode
	if s.acwResolver != nil {
		if seconds := s.acwResolver(ctx, p.TenantID, agentID); seconds > 0 {
			p.ACWSeconds = seconds
		}
	}
	p.UpdatedAt = time.Now()
	if err := s.presence.Upsert(ctx, p); err != nil {
		return nil, err
	}

	if p.ACWSeconds > 0 {
		s.scheduleACWTimeout(agentID, p.ACWSeconds)
	}
	return p, nil
}

func (s *AgentPresenceService) cancelACWTimer(agentID int64) {
	s.mu.Lock()
	if cancel, ok := s.acwTimers[agentID]; ok {
		cancel()
		delete(s.acwTimers, agentID)
	}
	s.mu.Unlock()
}

func (s *AgentPresenceService) scheduleACWTimeout(agentID int64, seconds int) {
	s.mu.Lock()
	if cancel, ok := s.acwTimers[agentID]; ok {
		cancel()
	}
	stopCh := make(chan struct{})
	s.acwTimers[agentID] = func() { close(stopCh) }
	s.mu.Unlock()

	go func() {
		select {
		case <-time.After(time.Duration(seconds) * time.Second):
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = s.TransitionTo(ctx, agentID, PresenceIdle)
		case <-stopCh:
		}
		s.mu.Lock()
		delete(s.acwTimers, agentID)
		s.mu.Unlock()
	}()
}

// ResetGhostAgents scans agents stuck in talking/dialing longer than maxDuration
// and resets them to idle. Returns the number of agents reset.
func (s *AgentPresenceService) ResetGhostAgents(ctx context.Context, tenantID int64, maxDuration time.Duration) (int, error) {
	presences, err := s.presence.ListByTenant(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	var resetCount int
	now := time.Now()
	for _, p := range presences {
		if (p.Status == PresenceTalking || p.Status == PresenceDialing) && now.Sub(p.LastStatusAt) > maxDuration {
			s.logTransition(ctx, p)
			p.Status = PresenceIdle
			p.SubState = SubStateNone
			p.CurrentCallID = nil
			p.UpdatedAt = now
			p.LastStatusAt = now
			if err := s.presence.Upsert(ctx, p); err == nil {
				resetCount++
			}
		}
	}
	return resetCount, nil
}

func (s *AgentPresenceService) GetPresence(ctx context.Context, agentID int64) (*AgentPresence, error) {
	return s.presence.GetByAgentID(ctx, agentID)
}

func (s *AgentPresenceService) ListByTenant(ctx context.Context, tenantID int64) ([]*AgentPresence, error) {
	return s.presence.ListByTenant(ctx, tenantID)
}

func (s *AgentPresenceService) logTransition(ctx context.Context, p *AgentPresence) {
	dur := int(time.Since(p.LastStatusAt).Seconds())
	_ = s.logs.Create(ctx, &AgentPresenceLog{
		ID:              snowflake.NextID(),
		TenantID:        p.TenantID,
		AgentID:         p.AgentID,
		Status:          p.Status,
		SubState:        p.SubState,
		WorkMode:        p.WorkMode,
		BreakReasonCode: p.BreakReasonCode,
		DurationSec:     dur,
		CreatedAt:       time.Now(),
	})
}
