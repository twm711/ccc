package im

import "errors"

var (
	ErrChannelNotFound    = errors.New("im channel not found")
	ErrChannelDisabled    = errors.New("im channel is disabled")
	ErrSessionNotFound    = errors.New("im session not found")
	ErrSessionNotActive   = errors.New("im session is not active")
	ErrSessionNotWaiting  = errors.New("im session is not in waiting state")
	ErrSessionClosed      = errors.New("im session is already closed")
	ErrNoAgentAvailable   = errors.New("no agent available for chat")
	ErrMaxChatSlots       = errors.New("agent has reached maximum chat slots")
	ErrEmptyMessage       = errors.New("message content cannot be empty")
	ErrInvalidContentType = errors.New("invalid content type")
	ErrInvalidChannelType = errors.New("invalid channel type")
)
