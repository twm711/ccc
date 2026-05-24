package ivr

import (
	"context"
	"fmt"
	"strconv"

	"github.com/divord97/ccc/internal/domain/routing"
)

// TransferToAgentHandler routes the call to a skill group / specific agent via ESL.
type TransferToAgentHandler struct{}

type transferAgentConfig struct {
	SkillGroupID        string `json:"skill_group_id"`
	RoutingStrategy     string `json:"routing_strategy"`
	QueuePriority       int    `json:"queue_priority"`
	QueueMusicID        string `json:"queue_music_id"`
	MaxWaitSeconds      int    `json:"max_wait_seconds"`
	EWTAnnounceInterval int    `json:"ewt_announce_interval"`
	CallbackEnabled     bool   `json:"callback_enabled"`
	CallbackThresholdSec int   `json:"callback_threshold_seconds"`
	WhisperEnabled      bool   `json:"whisper_enabled"`
	WhisperAudio        string `json:"whisper_audio"`
}

func (h *TransferToAgentHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg transferAgentConfig
	_ = parseConfig(node.Config, &cfg)

	sess.Variables["transfer_skill_group"] = cfg.SkillGroupID
	sess.Variables["transfer_strategy"] = cfg.RoutingStrategy
	sess.Variables["queue_priority"] = strconv.Itoa(cfg.QueuePriority)

	if sess.ESL != nil && sess.CallUUID != "" {
		// Play queue music while waiting
		if cfg.QueueMusicID != "" {
			_ = sess.ESL.PlayAudio(ctx, sess.CallUUID, cfg.QueueMusicID)
		}

		// Transfer to the fifo/callcenter queue for ACD distribution
		dest := fmt.Sprintf("callcenter:%s@default", cfg.SkillGroupID)
		if err := sess.ESL.TransferCall(ctx, sess.CallUUID, dest); err != nil {
			sess.Variables["transfer_error"] = err.Error()
			return "error", nil
		}
	}
	return "success", nil
}

// TransferToExternalHandler transfers the call to an external phone number via ESL bridge.
type TransferToExternalHandler struct{}

type transferExternalConfig struct {
	Number     string `json:"number"`
	CallerID   string `json:"caller_id"`
	TimeoutSec int    `json:"timeout_sec"`
}

func (h *TransferToExternalHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg transferExternalConfig
	_ = parseConfig(node.Config, &cfg)

	if cfg.TimeoutSec == 0 {
		cfg.TimeoutSec = 30
	}

	sess.Variables["transfer_external_number"] = cfg.Number

	if sess.ESL != nil && sess.CallUUID != "" {
		callerID := cfg.CallerID
		if callerID == "" {
			callerID = sess.Variables["caller_number"]
		}
		dest := fmt.Sprintf("sofia/external/%s", cfg.Number)
		cmd := fmt.Sprintf(
			"uuid_transfer %s -both 'bridge:{origination_caller_id_number=%s,call_timeout=%d}%s' inline",
			sess.CallUUID, callerID, cfg.TimeoutSec, dest,
		)
		if _, err := sess.ESL.SendCommand(ctx, cmd); err != nil {
			sess.Variables["transfer_error"] = err.Error()
			return "error", nil
		}
	}
	return "success", nil
}

// BlindTransferHandler transfers directly to an agent/extension without ACD queuing.
type BlindTransferHandler struct{}

type blindTransferConfig struct {
	Target     string `json:"target"`
	TargetType string `json:"target_type"` // agent, extension, number
}

func (h *BlindTransferHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg blindTransferConfig
	_ = parseConfig(node.Config, &cfg)

	sess.Variables["blind_transfer_target"] = cfg.Target

	if sess.ESL != nil && sess.CallUUID != "" {
		var dest string
		switch cfg.TargetType {
		case "agent", "extension":
			dest = fmt.Sprintf("user/%s", cfg.Target)
		case "number":
			dest = fmt.Sprintf("sofia/external/%s", cfg.Target)
		default:
			dest = cfg.Target
		}
		if err := sess.ESL.TransferCall(ctx, sess.CallUUID, dest); err != nil {
			sess.Variables["transfer_error"] = err.Error()
			return "error", nil
		}
	}
	return "success", nil
}

// CallbackHandler offers callback option when queue wait is too long.
type CallbackHandler struct{}

type callbackConfig struct {
	PromptAudioID  string `json:"prompt_audio_id"`
	ConfirmKey     string `json:"confirm_key"`
	RejectKey      string `json:"reject_key"`
	CallbackNumber string `json:"callback_number"`
}

func (h *CallbackHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg callbackConfig
	_ = parseConfig(node.Config, &cfg)

	if cfg.ConfirmKey == "" {
		cfg.ConfirmKey = "1"
	}
	if cfg.RejectKey == "" {
		cfg.RejectKey = "2"
	}

	cbNumber := cfg.CallbackNumber
	if cbNumber == "" {
		cbNumber = sess.Variables["caller_number"]
	}
	sess.Variables["callback_number"] = cbNumber

	if sess.ESL != nil && sess.CallUUID != "" {
		if cfg.PromptAudioID != "" {
			_ = sess.ESL.PlayAudio(ctx, sess.CallUUID, cfg.PromptAudioID)
		}

		cmd := fmt.Sprintf(
			"play_and_get_digits 1 1 3 5000 # silence_stream://250 silence_stream://250 cb_choice [%s%s]",
			cfg.ConfirmKey, cfg.RejectKey,
		)
		resp, err := sess.ESL.SendCommand(ctx,
			fmt.Sprintf("uuid_broadcast %s %s both", sess.CallUUID, cmd))
		if err != nil {
			return "reject", nil
		}

		if resp == cfg.ConfirmKey {
			sess.Variables["callback_requested"] = "true"
			return "confirm", nil
		}
	}
	return "reject", nil
}
