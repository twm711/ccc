package identity

import "errors"

var (
	ErrTenantNotFound         = errors.New("tenant not found")
	ErrTenantCodeExists       = errors.New("tenant code already exists")
	ErrTenantSuspended        = errors.New("tenant is suspended")
	ErrUserNotFound           = errors.New("user not found")
	ErrUsernameExists         = errors.New("username already exists")
	ErrAgentNotFound          = errors.New("agent not found")
	ErrAgentAlreadyExists     = errors.New("agent already exists for this user")
	ErrSkillGroupNotFound     = errors.New("skill group not found")
	ErrSkillGroupCodeExists   = errors.New("skill group code already exists")
	ErrMemberAlreadyExists    = errors.New("agent is already a member of this skill group")
	ErrMaxAgentsReached       = errors.New("maximum number of agents reached for this tenant")
	ErrInvalidRoutingPolicy   = errors.New("invalid routing policy")
	ErrPresenceNotFound       = errors.New("agent presence not found")
	ErrInvalidStateTransition = errors.New("invalid agent state transition")
	ErrInvalidWorkMode        = errors.New("invalid work mode")
	ErrAgentNotCheckedIn      = errors.New("agent is not checked in")
)
