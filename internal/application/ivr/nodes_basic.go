package ivr

import (
	"context"

	"github.com/divord97/ccc/internal/domain/routing"
)

// StartHandler processes the start node — initializes session variables.
type StartHandler struct{}

type startConfig struct {
	Variables map[string]string `json:"variables"`
}

func (h *StartHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg startConfig
	_ = parseConfig(node.Config, &cfg)
	for k, v := range cfg.Variables {
		sess.Variables[k] = v
	}
	return "default", nil
}

// PlayHandler processes the play node — plays audio or TTS.
type PlayHandler struct{}

func (h *PlayHandler) Handle(_ context.Context, _ *Session, _ routing.FlowNode) (string, error) {
	// In production: ESL command to play audio
	return "default", nil
}

// SetVariableHandler sets variables in the session scope.
type SetVariableHandler struct{}

type setVarConfig struct {
	Assignments []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"assignments"`
}

func (h *SetVariableHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg setVarConfig
	_ = parseConfig(node.Config, &cfg)
	for _, a := range cfg.Assignments {
		sess.Variables[a.Name] = a.Value
	}
	return "default", nil
}

// HangupReasonHandler marks a hangup reason on the session.
type HangupReasonHandler struct{}

type hangupConfig struct {
	Reason string `json:"reason"`
}

func (h *HangupReasonHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg hangupConfig
	_ = parseConfig(node.Config, &cfg)
	if cfg.Reason != "" {
		sess.Variables["hangup_reason"] = cfg.Reason
	}
	return "default", nil
}

// EndHandler is a terminal node — execution stops here.
type EndHandler struct{}

func (h *EndHandler) Handle(_ context.Context, _ *Session, _ routing.FlowNode) (string, error) {
	return "", nil
}
