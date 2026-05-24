package ivr

import (
	"context"

	"github.com/divord97/ccc/internal/domain/routing"
)

// CollectDTMFHandler collects DTMF input from the caller.
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

func (h *CollectDTMFHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg collectDTMFConfig
	_ = parseConfig(node.Config, &cfg)
	// In production: ESL play_and_get_digits command
	// For now, simulate success with empty input
	sess.Variables["dtmf_input"] = ""
	return "success", nil
}

// VoicemailHandler records a voicemail message.
type VoicemailHandler struct{}

func (h *VoicemailHandler) Handle(_ context.Context, _ *Session, _ routing.FlowNode) (string, error) {
	// In production: ESL record command → store file → create Voicemail entity
	return "default", nil
}

// SatisfactionRatingHandler collects CSAT rating via DTMF.
type SatisfactionRatingHandler struct{}

type csatConfig struct {
	PromptAudioID string `json:"prompt_audio_id"`
	MaxRating     int    `json:"max_rating"`
}

func (h *SatisfactionRatingHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg csatConfig
	_ = parseConfig(node.Config, &cfg)
	// In production: collect single DTMF digit for rating
	sess.Variables["csat_rating"] = ""
	return "success", nil
}

// Transcriber is an interface for speech-to-text transcription.
type Transcriber interface {
	Transcribe(ctx context.Context, audioURL string) (string, error)
}

// ASRHandler uses speech recognition for voice input.
type ASRHandler struct {
	Transcriber Transcriber
}

type asrConfig struct {
	AudioVariable string `json:"audio_variable"`
	TimeoutSec    int    `json:"timeout_sec"`
}

func (h *ASRHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg asrConfig
	_ = parseConfig(node.Config, &cfg)

	if h.Transcriber == nil {
		sess.Variables["asr_result"] = ""
		return "success", nil
	}

	audioURL := sess.Variables[cfg.AudioVariable]
	if audioURL == "" {
		audioURL = sess.Variables["asr_audio_file"]
	}
	if audioURL == "" {
		sess.Variables["asr_result"] = ""
		return "no_input", nil
	}

	text, err := h.Transcriber.Transcribe(ctx, audioURL)
	if err != nil {
		sess.Variables["asr_result"] = ""
		sess.Variables["asr_error"] = err.Error()
		return "error", nil
	}

	sess.Variables["asr_result"] = text
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
