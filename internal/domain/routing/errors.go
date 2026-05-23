package routing

import "errors"

var (
	ErrFlowNotFound      = errors.New("ivr flow not found")
	ErrFlowCodeExists    = errors.New("ivr flow code already exists")
	ErrFlowLocked        = errors.New("ivr flow is locked by another user")
	ErrFlowNotLocked     = errors.New("ivr flow is not locked")
	ErrFlowNotOwner      = errors.New("you are not the lock owner")
	ErrFlowNotDraft      = errors.New("ivr flow must be in draft status to publish")
	ErrFlowAlreadyPublished = errors.New("ivr flow is already published")
	ErrInvalidGraph      = errors.New("invalid ivr flow graph")
	ErrNoStartNode       = errors.New("graph must have exactly one start node")
	ErrNoEndNode         = errors.New("graph must have at least one end node")
	ErrDisconnectedNode  = errors.New("graph has disconnected nodes")
	ErrInvalidNodeType   = errors.New("unknown node type")
	ErrVersionNotFound   = errors.New("flow version not found")
)
