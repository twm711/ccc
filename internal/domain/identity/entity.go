package identity

import "time"

type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
)

type Tenant struct {
	ID        int64        `db:"id" json:"id"`
	Code      string       `db:"code" json:"code"`
	Name      string       `db:"name" json:"name"`
	Status    TenantStatus `db:"status" json:"status"`
	CreatedAt time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt time.Time    `db:"updated_at" json:"updated_at"`
}

type TenantSettings struct {
	TenantID                int64  `db:"tenant_id" json:"tenant_id"`
	MaxAgents               int    `db:"max_agents" json:"max_agents"`
	MaxConcurrentCalls      int    `db:"max_concurrent_calls" json:"max_concurrent_calls"`
	RecordingRetentionDays  int    `db:"recording_retention_days" json:"recording_retention_days"`
	RecordingStorageBackend string `db:"recording_storage_backend" json:"recording_storage_backend"`
	Timezone                string `db:"timezone" json:"timezone"`
	Language                string `db:"language" json:"language"`
}

type UserRole string

const (
	UserRoleAdmin      UserRole = "admin"
	UserRoleManager    UserRole = "manager"
	UserRoleAgent      UserRole = "agent"
	UserRoleSuperAdmin UserRole = "super_admin"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

type User struct {
	ID          int64      `db:"id" json:"id"`
	TenantID    int64      `db:"tenant_id" json:"tenant_id"`
	Username    string     `db:"username" json:"username"`
	DisplayName string     `db:"display_name" json:"display_name"`
	Email       string     `db:"email" json:"email"`
	Phone       string     `db:"phone" json:"phone"`
	Role        UserRole   `db:"role" json:"role"`
	Status      UserStatus `db:"status" json:"status"`
	ExternalUID string     `db:"external_uid" json:"external_uid,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

type WorkMode string

const (
	WorkModeOnSite WorkMode = "on_site"
	WorkModeOffSite WorkMode = "off_site"
	WorkModeOffice  WorkMode = "office"
)

type Agent struct {
	ID                      int64    `db:"id" json:"id"`
	TenantID                int64    `db:"tenant_id" json:"tenant_id"`
	UserID                  int64    `db:"user_id" json:"user_id"`
	EmployeeID              string   `db:"employee_id" json:"employee_id"`
	Extension               string   `db:"extension" json:"extension"`
	WorkMode                WorkMode `db:"work_mode" json:"work_mode"`
	OffSitePhone            string   `db:"off_site_phone" json:"off_site_phone,omitempty"`
	SIPDeviceStatus         string   `db:"sip_device_status" json:"sip_device_status"`
	MaxConcurrent           int      `db:"max_concurrent" json:"max_concurrent"`
	MaxChatSlots            int      `db:"max_chat_slots" json:"max_chat_slots"`
	ACWSeconds              int      `db:"acw_seconds" json:"acw_seconds"`
	OutboundOnly            bool     `db:"outbound_only" json:"outbound_only"`
	PersonalOutboundNumberID *int64  `db:"personal_outbound_number_id" json:"personal_outbound_number_id,omitempty"`
	CreatedAt               time.Time `db:"created_at" json:"created_at"`
	UpdatedAt               time.Time `db:"updated_at" json:"updated_at"`
}

type RoutingPolicy string

const (
	RoutingPolicyRoundRobin   RoutingPolicy = "round_robin"
	RoutingPolicyLeastRecent  RoutingPolicy = "least_recent"
	RoutingPolicyRandom       RoutingPolicy = "random"
	RoutingPolicySkillWeight  RoutingPolicy = "skill_weight"
	RoutingPolicyFamiliar     RoutingPolicy = "familiar"
)

type SkillGroupStatus string

const (
	SkillGroupStatusActive   SkillGroupStatus = "active"
	SkillGroupStatusDisabled SkillGroupStatus = "disabled"
)

type SkillGroup struct {
	ID            int64            `db:"id" json:"id"`
	TenantID      int64            `db:"tenant_id" json:"tenant_id"`
	Code          string           `db:"code" json:"code"`
	Name          string           `db:"name" json:"name"`
	Description   string           `db:"description" json:"description,omitempty"`
	RoutingPolicy RoutingPolicy    `db:"routing_policy" json:"routing_policy"`
	Priority      int              `db:"priority" json:"priority"`
	MaxWaitSec    int              `db:"max_wait_sec" json:"max_wait_sec"`
	OverflowGroup *int64           `db:"overflow_group_id" json:"overflow_group_id,omitempty"`
	Status        SkillGroupStatus `db:"status" json:"status"`
	CreatedAt     time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time        `db:"updated_at" json:"updated_at"`
}

type SkillGroupMember struct {
	ID           int64     `db:"id" json:"id"`
	SkillGroupID int64     `db:"skill_group_id" json:"skill_group_id"`
	AgentID      int64     `db:"agent_id" json:"agent_id"`
	Level        int       `db:"level" json:"level"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type AgentPresenceStatus string

const (
	PresenceOffline AgentPresenceStatus = "offline"
	PresenceOnline  AgentPresenceStatus = "online"
	PresenceIdle    AgentPresenceStatus = "idle"
	PresenceBreak   AgentPresenceStatus = "break"
	PresenceTalking AgentPresenceStatus = "talking"
	PresenceACW     AgentPresenceStatus = "acw"
	PresenceDialing AgentPresenceStatus = "dialing"
)

type AgentSubState string

const (
	SubStateNone       AgentSubState = ""
	SubStateMonitored  AgentSubState = "monitored"
	SubStateConsulted  AgentSubState = "consulted"
	SubStateConsulting AgentSubState = "consulting"
	SubStateConference AgentSubState = "conference"
	SubStateMonitoring AgentSubState = "monitoring"
)

type AgentPresence struct {
	ID              int64               `db:"id" json:"id"`
	TenantID        int64               `db:"tenant_id" json:"tenant_id"`
	AgentID         int64               `db:"agent_id" json:"agent_id"`
	Status          AgentPresenceStatus `db:"status" json:"status"`
	SubState        AgentSubState       `db:"sub_state" json:"sub_state"`
	WorkMode        WorkMode            `db:"work_mode" json:"work_mode"`
	BreakReasonCode string              `db:"break_reason_code" json:"break_reason_code,omitempty"`
	DispositionCode string              `db:"disposition_code" json:"disposition_code,omitempty"`
	CurrentCallID   *int64              `db:"current_call_id" json:"current_call_id,omitempty"`
	CheckedInAt     *time.Time          `db:"checked_in_at" json:"checked_in_at,omitempty"`
	LastStatusAt    time.Time           `db:"last_status_at" json:"last_status_at"`
	UpdatedAt       time.Time           `db:"updated_at" json:"updated_at"`
}

type AgentPresenceLog struct {
	ID              int64               `db:"id" json:"id"`
	TenantID        int64               `db:"tenant_id" json:"tenant_id"`
	AgentID         int64               `db:"agent_id" json:"agent_id"`
	Status          AgentPresenceStatus `db:"status" json:"status"`
	SubState        AgentSubState       `db:"sub_state" json:"sub_state"`
	WorkMode        WorkMode            `db:"work_mode" json:"work_mode"`
	BreakReasonCode string              `db:"break_reason_code" json:"break_reason_code,omitempty"`
	DurationSec     int                 `db:"duration_sec" json:"duration_sec"`
	CreatedAt       time.Time           `db:"created_at" json:"created_at"`
}
