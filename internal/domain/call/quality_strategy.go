package call

// DegradationAction describes a recommended action when call quality degrades.
type DegradationAction string

const (
	ActionNone        DegradationAction = "none"
	ActionReduceBitrate DegradationAction = "reduce_bitrate"
	ActionSwitchCodec   DegradationAction = "switch_codec"
	ActionPSTNFallback  DegradationAction = "pstn_fallback"
)

// DegradationRecommendation is the suggested action based on quality metrics.
type DegradationRecommendation struct {
	Action         DegradationAction `json:"action"`
	TargetBitrate  int               `json:"target_bitrate_kbps,omitempty"`
	TargetCodec    string            `json:"target_codec,omitempty"`
	Reason         string            `json:"reason"`
}

// QualityThresholds configures when degradation actions trigger.
type QualityThresholds struct {
	MOSReduceBitrate  float64 // MOS below this → reduce bitrate (default 3.5)
	MOSSwitchCodec    float64 // MOS below this → switch to low-bandwidth codec (default 2.5)
	MOSPSTNFallback   float64 // MOS below this → fall back to PSTN (default 2.0)
	PacketLossHigh    float64 // packet loss above this → action (default 0.05)
	JitterHigh        float64 // jitter above this (ms) → action (default 50)
}

// DefaultQualityThresholds returns production-safe defaults.
func DefaultQualityThresholds() QualityThresholds {
	return QualityThresholds{
		MOSReduceBitrate: 3.5,
		MOSSwitchCodec:   2.5,
		MOSPSTNFallback:  2.0,
		PacketLossHigh:   0.05,
		JitterHigh:       50,
	}
}

// EvaluateDegradation recommends an action based on current quality metrics.
func EvaluateDegradation(summary QualitySummary, currentBitrate int, thresholds QualityThresholds) DegradationRecommendation {
	if summary.SampleCount == 0 {
		return DegradationRecommendation{Action: ActionNone}
	}

	if thresholds.MOSReduceBitrate == 0 {
		thresholds = DefaultQualityThresholds()
	}

	// PSTN fallback for severe degradation.
	if summary.AvgMOS < thresholds.MOSPSTNFallback || summary.AvgLoss > thresholds.PacketLossHigh*3 {
		return DegradationRecommendation{
			Action: ActionPSTNFallback,
			Reason: "severe quality degradation — recommend PSTN callback",
		}
	}

	// Switch to low-bandwidth codec.
	if summary.AvgMOS < thresholds.MOSSwitchCodec {
		return DegradationRecommendation{
			Action:      ActionSwitchCodec,
			TargetCodec: "PCMU", // G.711 — lower quality but more resilient
			Reason:      "poor quality — switch to resilient codec",
		}
	}

	// Reduce bitrate.
	if summary.AvgMOS < thresholds.MOSReduceBitrate || summary.AvgJitter > thresholds.JitterHigh {
		target := currentBitrate / 2
		if target < 16 {
			target = 16
		}
		return DegradationRecommendation{
			Action:        ActionReduceBitrate,
			TargetBitrate: target,
			Reason:        "degraded quality — reducing bitrate",
		}
	}

	return DegradationRecommendation{Action: ActionNone}
}
