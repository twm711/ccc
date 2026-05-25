package ivr

import (
	"context"
	"strconv"

	"github.com/divord97/ccc/internal/domain/routing"
)

// SentimentAnalyzer scores the emotional tone of a text utterance.
// Score range: -1.0 (very negative) to +1.0 (very positive).
type SentimentAnalyzer interface {
	AnalyzeSentiment(ctx context.Context, text string) (score float64, err error)
}

// SentimentGateHandler routes calls based on caller sentiment.
// High-anger callers (score below negative threshold) are routed to a VIP/escalation
// queue, while normal sentiment proceeds through the default exit.
type SentimentGateHandler struct {
	Analyzer SentimentAnalyzer
}

type sentimentGateConfig struct {
	NegativeThreshold float64 `json:"negative_threshold"` // e.g. -0.5
	SourceVar         string  `json:"source_var"`         // IVR variable containing caller utterance
}

func (h *SentimentGateHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg sentimentGateConfig
	if err := parseConfig(node.Config, &cfg); err != nil {
		return "error", nil
	}

	if cfg.NegativeThreshold == 0 {
		cfg.NegativeThreshold = -0.5
	}

	sourceVar := cfg.SourceVar
	if sourceVar == "" {
		sourceVar = "last_utterance"
	}

	text := sess.Variables[sourceVar]
	if text == "" || h.Analyzer == nil {
		sess.Variables["sentiment_score"] = "0"
		return "neutral", nil
	}

	score, err := h.Analyzer.AnalyzeSentiment(ctx, text)
	if err != nil {
		sess.Variables["sentiment_score"] = "0"
		return "error", nil
	}

	sess.Variables["sentiment_score"] = strconv.FormatFloat(score, 'f', 2, 64)

	if score <= cfg.NegativeThreshold {
		return "negative", nil
	}
	if score >= -cfg.NegativeThreshold {
		return "positive", nil
	}
	return "neutral", nil
}
