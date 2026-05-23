package identity

import (
	"context"
	"testing"

	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = snowflake.Init(1)
}

// --- Tenant Service Tests ---

func TestTenantService_Create_Success(t *testing.T) {
	svc := NewTenantService(NewMockTenantRepo(), NewMockTenantSettingsRepo())

	tenant, err := svc.Create(context.Background(), CreateTenantInput{
		Code: "acme",
		Name: "Acme Corp",
	})

	require.NoError(t, err)
	assert.Equal(t, "acme", tenant.Code)
	assert.Equal(t, "Acme Corp", tenant.Name)
	assert.Equal(t, TenantStatusActive, tenant.Status)
	assert.NotZero(t, tenant.ID)
}

func TestTenantService_Create_DuplicateCode(t *testing.T) {
	svc := NewTenantService(NewMockTenantRepo(), NewMockTenantSettingsRepo())
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateTenantInput{Code: "acme", Name: "Acme"})
	require.NoError(t, err)

	_, err = svc.Create(ctx, CreateTenantInput{Code: "acme", Name: "Acme 2"})
	assert.ErrorIs(t, err, ErrTenantCodeExists)
}

func TestTenantService_Create_DefaultSettings(t *testing.T) {
	settingsRepo := NewMockTenantSettingsRepo()
	svc := NewTenantService(NewMockTenantRepo(), settingsRepo)

	tenant, err := svc.Create(context.Background(), CreateTenantInput{Code: "t1", Name: "T1"})
	require.NoError(t, err)

	settings, err := settingsRepo.GetByTenantID(context.Background(), tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 50, settings.MaxAgents)
	assert.Equal(t, 100, settings.MaxConcurrentCalls)
	assert.Equal(t, 365, settings.RecordingRetentionDays)
	assert.Equal(t, "local", settings.RecordingStorageBackend)
	assert.Equal(t, "Asia/Shanghai", settings.Timezone)
}

func TestTenantService_Update_Success(t *testing.T) {
	svc := NewTenantService(NewMockTenantRepo(), NewMockTenantSettingsRepo())
	ctx := context.Background()

	tenant, err := svc.Create(ctx, CreateTenantInput{Code: "acme", Name: "Acme"})
	require.NoError(t, err)

	updated, err := svc.Update(ctx, tenant.ID, "Acme Updated", TenantStatusSuspended)
	require.NoError(t, err)
	assert.Equal(t, "Acme Updated", updated.Name)
	assert.Equal(t, TenantStatusSuspended, updated.Status)
}

func TestTenantService_Update_NotFound(t *testing.T) {
	svc := NewTenantService(NewMockTenantRepo(), NewMockTenantSettingsRepo())

	_, err := svc.Update(context.Background(), 99999, "x", TenantStatusActive)
	assert.ErrorIs(t, err, ErrTenantNotFound)
}

func TestTenantService_List(t *testing.T) {
	svc := NewTenantService(NewMockTenantRepo(), NewMockTenantSettingsRepo())
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, _ = svc.Create(ctx, CreateTenantInput{Code: string(rune('a'+i)) + "co", Name: "Co"})
	}

	list, total, err := svc.List(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, list, 5)
}

// --- User Service Tests ---

func TestUserService_Create_Success(t *testing.T) {
	svc := NewUserService(NewMockUserRepo(), NewMockAgentRepo())

	user, err := svc.Create(context.Background(), CreateUserInput{
		TenantID:    1,
		Username:    "john",
		DisplayName: "John Doe",
		Email:       "john@acme.com",
		Phone:       "13800138000",
		Role:        UserRoleAgent,
	})

	require.NoError(t, err)
	assert.Equal(t, "john", user.Username)
	assert.Equal(t, UserRoleAgent, user.Role)
	assert.Equal(t, UserStatusActive, user.Status)
}

func TestUserService_Create_DuplicateUsername(t *testing.T) {
	svc := NewUserService(NewMockUserRepo(), NewMockAgentRepo())
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateUserInput{TenantID: 1, Username: "john", DisplayName: "J", Role: UserRoleAgent})
	require.NoError(t, err)

	_, err = svc.Create(ctx, CreateUserInput{TenantID: 1, Username: "john", DisplayName: "J2", Role: UserRoleAgent})
	assert.ErrorIs(t, err, ErrUsernameExists)
}

func TestUserService_Create_SameUsernameOtherTenant(t *testing.T) {
	svc := NewUserService(NewMockUserRepo(), NewMockAgentRepo())
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateUserInput{TenantID: 1, Username: "john", DisplayName: "J", Role: UserRoleAgent})
	require.NoError(t, err)

	user2, err := svc.Create(ctx, CreateUserInput{TenantID: 2, Username: "john", DisplayName: "J2", Role: UserRoleAgent})
	require.NoError(t, err)
	assert.Equal(t, int64(2), user2.TenantID)
}

func TestUserService_Update_Success(t *testing.T) {
	svc := NewUserService(NewMockUserRepo(), NewMockAgentRepo())
	ctx := context.Background()

	user, _ := svc.Create(ctx, CreateUserInput{TenantID: 1, Username: "john", DisplayName: "John", Role: UserRoleAgent})

	updated, err := svc.Update(ctx, user.ID, "John Updated", "new@acme.com", "13900139000")
	require.NoError(t, err)
	assert.Equal(t, "John Updated", updated.DisplayName)
	assert.Equal(t, "new@acme.com", updated.Email)
}

// --- Agent Service Tests ---

func TestAgentService_Create_Success(t *testing.T) {
	agentRepo := NewMockAgentRepo()
	settingsRepo := NewMockTenantSettingsRepo()
	_ = settingsRepo.Upsert(context.Background(), &TenantSettings{TenantID: 1, MaxAgents: 50})
	svc := NewAgentService(agentRepo, NewMockUserRepo(), settingsRepo)

	agent, err := svc.Create(context.Background(), CreateAgentInput{
		TenantID:   1,
		UserID:     100,
		EmployeeID: "EMP001",
		Extension:  "8001",
		WorkMode:   WorkModeOnSite,
	})

	require.NoError(t, err)
	assert.Equal(t, "EMP001", agent.EmployeeID)
	assert.Equal(t, "8001", agent.Extension)
	assert.Equal(t, WorkModeOnSite, agent.WorkMode)
	assert.Equal(t, 3, agent.MaxChatSlots)
	assert.Equal(t, 15, agent.ACWSeconds)
}

func TestAgentService_Create_DuplicateUserAgent(t *testing.T) {
	settingsRepo := NewMockTenantSettingsRepo()
	_ = settingsRepo.Upsert(context.Background(), &TenantSettings{TenantID: 1, MaxAgents: 50})
	svc := NewAgentService(NewMockAgentRepo(), NewMockUserRepo(), settingsRepo)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateAgentInput{TenantID: 1, UserID: 100, EmployeeID: "E1", Extension: "8001"})
	require.NoError(t, err)

	_, err = svc.Create(ctx, CreateAgentInput{TenantID: 1, UserID: 100, EmployeeID: "E2", Extension: "8002"})
	assert.ErrorIs(t, err, ErrAgentAlreadyExists)
}

func TestAgentService_Create_MaxAgentsReached(t *testing.T) {
	settingsRepo := NewMockTenantSettingsRepo()
	_ = settingsRepo.Upsert(context.Background(), &TenantSettings{TenantID: 1, MaxAgents: 1})
	svc := NewAgentService(NewMockAgentRepo(), NewMockUserRepo(), settingsRepo)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateAgentInput{TenantID: 1, UserID: 100, EmployeeID: "E1", Extension: "8001"})
	require.NoError(t, err)

	_, err = svc.Create(ctx, CreateAgentInput{TenantID: 1, UserID: 200, EmployeeID: "E2", Extension: "8002"})
	assert.ErrorIs(t, err, ErrMaxAgentsReached)
}

// --- SkillGroup Service Tests ---

func TestSkillGroupService_Create_Success(t *testing.T) {
	svc := NewSkillGroupService(NewMockSkillGroupRepo(), NewMockSkillGroupMemberRepo())

	sg, err := svc.Create(context.Background(), CreateSkillGroupInput{
		TenantID:      1,
		Code:          "sales",
		Name:          "Sales Team",
		RoutingPolicy: RoutingPolicyRoundRobin,
		Priority:      1,
	})

	require.NoError(t, err)
	assert.Equal(t, "sales", sg.Code)
	assert.Equal(t, RoutingPolicyRoundRobin, sg.RoutingPolicy)
	assert.Equal(t, 60, sg.MaxWaitSec) // default
	assert.Equal(t, SkillGroupStatusActive, sg.Status)
}

func TestSkillGroupService_Create_DuplicateCode(t *testing.T) {
	svc := NewSkillGroupService(NewMockSkillGroupRepo(), NewMockSkillGroupMemberRepo())
	ctx := context.Background()

	_, _ = svc.Create(ctx, CreateSkillGroupInput{TenantID: 1, Code: "sales", Name: "S1", RoutingPolicy: RoutingPolicyRandom})

	_, err := svc.Create(ctx, CreateSkillGroupInput{TenantID: 1, Code: "sales", Name: "S2", RoutingPolicy: RoutingPolicyRandom})
	assert.ErrorIs(t, err, ErrSkillGroupCodeExists)
}

func TestSkillGroupService_Create_InvalidPolicy(t *testing.T) {
	svc := NewSkillGroupService(NewMockSkillGroupRepo(), NewMockSkillGroupMemberRepo())

	_, err := svc.Create(context.Background(), CreateSkillGroupInput{
		TenantID:      1,
		Code:          "x",
		Name:          "X",
		RoutingPolicy: "invalid",
	})
	assert.ErrorIs(t, err, ErrInvalidRoutingPolicy)
}

func TestSkillGroupService_AddMember_Success(t *testing.T) {
	svc := NewSkillGroupService(NewMockSkillGroupRepo(), NewMockSkillGroupMemberRepo())

	member, err := svc.AddMember(context.Background(), 1, 100, 5)
	require.NoError(t, err)
	assert.Equal(t, int64(1), member.SkillGroupID)
	assert.Equal(t, int64(100), member.AgentID)
	assert.Equal(t, 5, member.Level)
}

func TestSkillGroupService_AddMember_Duplicate(t *testing.T) {
	svc := NewSkillGroupService(NewMockSkillGroupRepo(), NewMockSkillGroupMemberRepo())
	ctx := context.Background()

	_, _ = svc.AddMember(ctx, 1, 100, 5)

	_, err := svc.AddMember(ctx, 1, 100, 3)
	assert.ErrorIs(t, err, ErrMemberAlreadyExists)
}

func TestSkillGroupService_RemoveMember(t *testing.T) {
	svc := NewSkillGroupService(NewMockSkillGroupRepo(), NewMockSkillGroupMemberRepo())
	ctx := context.Background()

	_, _ = svc.AddMember(ctx, 1, 100, 5)
	err := svc.RemoveMember(ctx, 1, 100)
	require.NoError(t, err)

	members, _ := svc.GetMembers(ctx, 1)
	assert.Empty(t, members)
}

// --- AgentPresence Service Tests ---

func newPresenceSvc() *AgentPresenceService {
	return NewAgentPresenceService(NewMockAgentPresenceRepo(), NewMockAgentPresenceLogRepo())
}

func TestAgentPresence_StateTransition_DialingToTalking(t *testing.T) {
	svc := newPresenceSvc()
	ctx := context.Background()

	p, err := svc.CheckIn(ctx, 1, 100, WorkModeOnSite)
	require.NoError(t, err)
	assert.Equal(t, PresenceOnline, p.Status)

	// online → idle
	p, err = svc.TransitionTo(ctx, 100, PresenceIdle)
	require.NoError(t, err)
	assert.Equal(t, PresenceIdle, p.Status)

	// idle → dialing
	p, err = svc.TransitionTo(ctx, 100, PresenceDialing)
	require.NoError(t, err)
	assert.Equal(t, PresenceDialing, p.Status)

	// dialing → talking
	p, err = svc.TransitionTo(ctx, 100, PresenceTalking)
	require.NoError(t, err)
	assert.Equal(t, PresenceTalking, p.Status)
}

func TestAgentPresence_SubState_Monitored(t *testing.T) {
	svc := newPresenceSvc()
	ctx := context.Background()

	_, _ = svc.CheckIn(ctx, 1, 100, WorkModeOnSite)
	_, _ = svc.TransitionTo(ctx, 100, PresenceIdle)
	_, _ = svc.TransitionTo(ctx, 100, PresenceTalking)

	p, err := svc.SetSubState(ctx, 100, SubStateMonitored)
	require.NoError(t, err)
	assert.Equal(t, SubStateMonitored, p.SubState)
	assert.Equal(t, PresenceTalking, p.Status)
}

func TestAgentPresence_WorkMode_Switch(t *testing.T) {
	svc := newPresenceSvc()
	ctx := context.Background()

	_, _ = svc.CheckIn(ctx, 1, 100, WorkModeOnSite)

	p, err := svc.SwitchWorkMode(ctx, 100, WorkModeOffSite)
	require.NoError(t, err)
	assert.Equal(t, WorkModeOffSite, p.WorkMode)

	// invalid work mode
	_, err = svc.SwitchWorkMode(ctx, 100, "invalid")
	assert.ErrorIs(t, err, ErrInvalidWorkMode)
}

func TestAgentPresence_ACW_WithDispositionCode(t *testing.T) {
	svc := newPresenceSvc()
	ctx := context.Background()

	_, _ = svc.CheckIn(ctx, 1, 100, WorkModeOnSite)
	_, _ = svc.TransitionTo(ctx, 100, PresenceIdle)
	_, _ = svc.TransitionTo(ctx, 100, PresenceTalking)

	p, err := svc.SetACW(ctx, 100, "resolved")
	require.NoError(t, err)
	assert.Equal(t, PresenceACW, p.Status)
	assert.Equal(t, "resolved", p.DispositionCode)
}

func TestAgentPresence_InvalidTransition(t *testing.T) {
	svc := newPresenceSvc()
	ctx := context.Background()

	_, _ = svc.CheckIn(ctx, 1, 100, WorkModeOnSite)

	// online → talking (invalid, must go through idle first)
	_, err := svc.TransitionTo(ctx, 100, PresenceTalking)
	assert.ErrorIs(t, err, ErrInvalidStateTransition)
}
