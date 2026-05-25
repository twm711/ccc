package ivr

import (
	"context"
	"strings"

	"github.com/divord97/ccc/internal/domain/routing"
)

// NLUProvider classifies caller utterances into intents.
type NLUProvider interface {
	ClassifyIntent(ctx context.Context, text string) (intent string, confidence float64, err error)
}

// NLUHandler processes an NLU node: collects ASR text, classifies intent, routes to matching exit.
type NLUHandler struct {
	NLU NLUProvider
}

type nluConfig struct {
	Prompt         string  `json:"prompt"`
	MinConfidence  float64 `json:"min_confidence"`
}

func (h *NLUHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg nluConfig
	_ = parseConfig(node.Config, &cfg)

	// Get the ASR text from session variables (set by a preceding ASR node).
	text := sess.Variables["asr_result"]
	if text == "" {
		return "fallback", nil
	}

	if h.NLU == nil {
		return "fallback", nil
	}

	intent, confidence, err := h.NLU.ClassifyIntent(ctx, text)
	if err != nil {
		return "fallback", nil
	}

	minConf := cfg.MinConfidence
	if minConf <= 0 {
		minConf = 0.6
	}
	if confidence < minConf {
		return "fallback", nil
	}

	sess.Variables["nlu_intent"] = intent
	sess.Variables["nlu_confidence"] = strings.TrimRight(strings.TrimRight(
		func() string { return formatFloat(confidence) }(), "0"), ".")

	// Route to exit matching the intent name; fall back to "default" if no match.
	if _, ok := node.Exits[intent]; ok {
		return intent, nil
	}
	return "default", nil
}

func formatFloat(f float64) string {
	s := make([]byte, 0, 8)
	return string(appendFloat(s, f))
}

func appendFloat(b []byte, f float64) []byte {
	if f < 0 {
		b = append(b, '-')
		f = -f
	}
	whole := int64(f)
	frac := int64((f - float64(whole)) * 100)
	b = appendInt(b, whole)
	b = append(b, '.')
	if frac < 10 {
		b = append(b, '0')
	}
	b = appendInt(b, frac)
	return b
}

func appendInt(b []byte, n int64) []byte {
	if n == 0 {
		return append(b, '0')
	}
	var tmp [20]byte
	i := len(tmp)
	for n > 0 {
		i--
		tmp[i] = byte('0' + n%10)
		n /= 10
	}
	return append(b, tmp[i:]...)
}
