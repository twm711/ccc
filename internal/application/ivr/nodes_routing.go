package ivr

import (
	"context"

	"github.com/divord97/ccc/internal/domain/routing"
)

// TransferToAgentHandler routes the call to a skill group / specific agent.
type TransferToAgentHandler struct{}

type transferAgentConfig struct {
	SkillGroupID            string `json:"skill_group_id"`
	RoutingStrategy         string `json:"routing_strategy"`
	QueuePriority           int    `json:"queue_priority"`
	QueueMusicID            string `json:"queue_music_id"`
	MaxWaitSeconds          int    `json:"max_wait_seconds"`
	EWTAnnounceInterval     int    `json:"ewt_announce_interval"`
	CallbackEnabled         bool   `json:"callback_enabled"`
	CallbackThresholdSec    int    `json:"callback_threshold_seconds"`
	WhisperEnabled          bool   `json:"whisper_enabled"`
	WhisperAudio            string `json:"whisper_audio"`
}

func (h *TransferToAgentHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg transferAgentConfig
	_ = parseConfig(node.Config, &cfg)
	// In production: enqueue call to ACD, signal agent via ESL/NATS
	sess.Variables["transfer_skill_group"] = cfg.SkillGroupID
	sess.Variables["transfer_strategy"] = cfg.RoutingStrategy
	return "success", nil
}

// TransferToExternalHandler transfers the call to an external phone number.
type TransferToExternalHandler struct{}

type transferExternalConfig struct {
	Number     string `json:"number"`
	CallerID   string `json:"caller_id"`
	TimeoutSec int    `json:"timeout_sec"`
}

func (h *TransferToExternalHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg transferExternalConfig
	_ = parseConfig(node.Config, &cfg)
	// In production: ESL bridge to external number via SIP trunk
	sess.Variables["transfer_external_number"] = cfg.Number
	return "success", nil
}

// BlindTransferHandler transfers directly to an agent/extension without ACD queuing.
type BlindTransferHandler struct{}

type blindTransferConfig struct {
	Target     string `json:"target"`
	TargetType string `json:"target_type"` // agent, extension, number
}

func (h *BlindTransferHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg blindTransferConfig
	_ = parseConfig(node.Config, &cfg)
	// In production: ESL uuid_transfer
	sess.Variables["blind_transfer_target"] = cfg.Target
	return "success", nil
}

// CallbackHandler offers callback option when queue wait is too long.
type CallbackHandler struct{}

type callbackConfig struct {
	PromptAudioID   string `json:"prompt_audio_id"`
	ConfirmKey      string `json:"confirm_key"`
	RejectKey       string `json:"reject_key"`
	CallbackNumber  string `json:"callback_number"` // default: caller
}

func (h *CallbackHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg callbackConfig
	_ = parseConfig(node.Config, &cfg)
	// In production: play prompt, collect DTMF, create CallbackRequest
	return "default", nil
}
