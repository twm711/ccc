package ivr

import (
	"context"
	"fmt"

	"github.com/divord97/ccc/internal/domain/routing"
)

// CollectDTMFHandler collects DTMF input from the caller via ESL play_and_get_digits.
type CollectDTMFHandler struct{}

type collectDTMFConfig struct {
	MinDigits     int    `json:"min_digits"`
	MaxDigits     int    `json:"max_digits"`
	TimeoutSec    int    `json:"timeout_sec"`
	TermChar      string `json:"term_char"`
	RetryCount    int    `json:"retry_count"`
	PromptAudioID string `json:"prompt_audio_id"`
	FailAudioID   string `json:"fail_audio_id"`
}

func (h *CollectDTMFHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg collectDTMFConfig
	_ = parseConfig(node.Config, &cfg)

	if cfg.MinDigits == 0 {
		cfg.MinDigits = 1
	}
	if cfg.MaxDigits == 0 {
		cfg.MaxDigits = cfg.MinDigits
	}
	if cfg.TimeoutSec == 0 {
		cfg.TimeoutSec = 10
	}
	if cfg.TermChar == "" {
		cfg.TermChar = "#"
	}
	if cfg.RetryCount == 0 {
		cfg.RetryCount = 3
	}

	if sess.ESL != nil && sess.CallUUID != "" {
		prompt := cfg.PromptAudioID
		if prompt == "" {
			prompt = "silence_stream://250"
		}
		failSound := cfg.FailAudioID
		if failSound == "" {
			failSound = "silence_stream://250"
		}

		cmd := fmt.Sprintf(
			"play_and_get_digits %d %d %d %d %s %s %s dtmf_result \\d+",
			cfg.MinDigits, cfg.MaxDigits, cfg.RetryCount,
			cfg.TimeoutSec*1000, cfg.TermChar,
			prompt, failSound,
		)
		resp, err := sess.ESL.SendCommand(ctx,
			fmt.Sprintf("uuid_broadcast %s %s both", sess.CallUUID, cmd))
		if err != nil {
			sess.Variables["dtmf_input"] = ""
			return "timeout", nil
		}
		sess.Variables["dtmf_input"] = resp
		return "success", nil
	}

	sess.Variables["dtmf_input"] = ""
	return "success", nil
}

// VoicemailHandler records a voicemail message via ESL.
type VoicemailHandler struct{}

type voicemailConfig struct {
	MaxDurationSec int    `json:"max_duration_sec"`
	BeepFile       string `json:"beep_file"`
	StoragePath    string `json:"storage_path"`
}

func (h *VoicemailHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg voicemailConfig
	_ = parseConfig(node.Config, &cfg)

	if cfg.MaxDurationSec == 0 {
		cfg.MaxDurationSec = 120
	}
	if cfg.StoragePath == "" {
		cfg.StoragePath = fmt.Sprintf("/recordings/voicemail/%d_%d.wav", sess.TenantID, sess.CallID)
	}

	if sess.ESL != nil && sess.CallUUID != "" {
		if cfg.BeepFile != "" {
			_ = sess.ESL.PlayAudio(ctx, sess.CallUUID, cfg.BeepFile)
		}
		if err := sess.ESL.StartRecording(ctx, sess.CallUUID, cfg.StoragePath); err != nil {
			sess.Variables["voicemail_error"] = err.Error()
			return "error", nil
		}
		sess.Variables["voicemail_file"] = cfg.StoragePath
	}
	return "default", nil
}

// SatisfactionRatingHandler collects CSAT rating via DTMF.
type SatisfactionRatingHandler struct{}

type csatConfig struct {
	PromptAudioID string `json:"prompt_audio_id"`
	MaxRating     int    `json:"max_rating"`
}

func (h *SatisfactionRatingHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg csatConfig
	_ = parseConfig(node.Config, &cfg)

	if cfg.MaxRating == 0 {
		cfg.MaxRating = 5
	}

	if sess.ESL != nil && sess.CallUUID != "" {
		prompt := cfg.PromptAudioID
		if prompt == "" {
			prompt = "silence_stream://250"
		}
		cmd := fmt.Sprintf(
			"play_and_get_digits 1 1 3 5000 # %s silence_stream://250 csat_result [1-%d]",
			prompt, cfg.MaxRating,
		)
		resp, err := sess.ESL.SendCommand(ctx,
			fmt.Sprintf("uuid_broadcast %s %s both", sess.CallUUID, cmd))
		if err != nil {
			sess.Variables["csat_rating"] = ""
			return "timeout", nil
		}
		sess.Variables["csat_rating"] = resp
		return "success", nil
	}

	sess.Variables["csat_rating"] = ""
	return "success", nil
}

// ASRHandler uses speech recognition for voice input.
type ASRHandler struct{}

type asrConfig struct {
	Language   string `json:"language"`
	TimeoutMs int    `json:"timeout_ms"`
	Grammar   string `json:"grammar"`
}

func (h *ASRHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg asrConfig
	_ = parseConfig(node.Config, &cfg)

	if cfg.TimeoutMs == 0 {
		cfg.TimeoutMs = 10000
	}
	if cfg.Language == "" {
		cfg.Language = "zh-CN"
	}

	if sess.ESL != nil && sess.CallUUID != "" {
		cmd := fmt.Sprintf("uuid_record %s start /tmp/asr_%d.wav %d",
			sess.CallUUID, sess.CallID, cfg.TimeoutMs/1000)
		_, err := sess.ESL.SendCommand(ctx, cmd)
		if err != nil {
			sess.Variables["asr_result"] = ""
			return "error", nil
		}
		sess.Variables["asr_audio_file"] = fmt.Sprintf("/tmp/asr_%d.wav", sess.CallID)
	}

	sess.Variables["asr_result"] = ""
	return "success", nil
}

// BranchHandler evaluates conditions and routes to matching exit.
type BranchHandler struct{}

type branchConfig struct {
	Conditions []branchCondition `json:"conditions"`
}

type branchCondition struct {
	Name     string `json:"name"`
	Variable string `json:"variable"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

func (h *BranchHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg branchConfig
	_ = parseConfig(node.Config, &cfg)

	for _, c := range cfg.Conditions {
		val := sess.Variables[c.Variable]
		if evaluateCondition(val, c.Operator, c.Value) {
			return c.Name, nil
		}
	}
	return "default", nil
}

func evaluateCondition(actual, op, expected string) bool {
	switch op {
	case "eq", "==":
		return actual == expected
	case "ne", "!=":
		return actual != expected
	case "contains":
		return len(actual) > 0 && len(expected) > 0 && contains(actual, expected)
	case "starts_with":
		return len(actual) >= len(expected) && actual[:len(expected)] == expected
	case "ends_with":
		return len(actual) >= len(expected) && actual[len(actual)-len(expected):] == expected
	case "empty":
		return actual == ""
	case "not_empty":
		return actual != ""
	default:
		return false
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
