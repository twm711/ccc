package call

import "time"

// WebRTCQualityLog records real-time WebRTC call quality metrics.
type WebRTCQualityLog struct {
	ID              int64     `db:"id" json:"id"`
	CallID          int64     `db:"call_id" json:"call_id"`
	TenantID        int64     `db:"tenant_id" json:"tenant_id"`
	AgentID         int64     `db:"agent_id" json:"agent_id"`
	PacketLossRate  float64   `db:"packet_loss_rate" json:"packet_loss_rate"`   // 0.0-1.0
	Jitter          float64   `db:"jitter" json:"jitter"`                       // ms
	RoundTripTime   float64   `db:"round_trip_time" json:"round_trip_time"`     // ms
	MOS             float64   `db:"mos" json:"mos"`                             // 1.0-5.0
	AudioLevel      float64   `db:"audio_level" json:"audio_level"`             // dBFS
	BitrateKbps     int       `db:"bitrate_kbps" json:"bitrate_kbps"`
	CodecName       string    `db:"codec_name" json:"codec_name"`
	SampledAt       time.Time `db:"sampled_at" json:"sampled_at"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

// QualityLevel returns a traffic-light level based on MOS score.
func (w *WebRTCQualityLog) QualityLevel() string {
	switch {
	case w.MOS >= 4.0:
		return "good"
	case w.MOS >= 3.0:
		return "fair"
	default:
		return "poor"
	}
}

// QualitySummary aggregates WebRTC quality metrics for monitoring.
type QualitySummary struct {
	CallID      int64   `json:"call_id"`
	SampleCount int     `json:"sample_count"`
	AvgMOS      float64 `json:"avg_mos"`
	MinMOS      float64 `json:"min_mos"`
	MaxMOS      float64 `json:"max_mos"`
	AvgJitter   float64 `json:"avg_jitter"`
	AvgLoss     float64 `json:"avg_packet_loss"`
	AvgRTT      float64 `json:"avg_rtt"`
	Level       string  `json:"level"`
}

// SummarizeQuality aggregates quality logs for a call.
func SummarizeQuality(logs []WebRTCQualityLog) QualitySummary {
	if len(logs) == 0 {
		return QualitySummary{}
	}
	s := QualitySummary{
		CallID:      logs[0].CallID,
		SampleCount: len(logs),
		MinMOS:      logs[0].MOS,
		MaxMOS:      logs[0].MOS,
	}
	var totalMOS, totalJitter, totalLoss, totalRTT float64
	for _, l := range logs {
		totalMOS += l.MOS
		totalJitter += l.Jitter
		totalLoss += l.PacketLossRate
		totalRTT += l.RoundTripTime
		if l.MOS < s.MinMOS {
			s.MinMOS = l.MOS
		}
		if l.MOS > s.MaxMOS {
			s.MaxMOS = l.MOS
		}
	}
	n := float64(len(logs))
	s.AvgMOS = totalMOS / n
	s.AvgJitter = totalJitter / n
	s.AvgLoss = totalLoss / n
	s.AvgRTT = totalRTT / n
	switch {
	case s.AvgMOS >= 4.0:
		s.Level = "good"
	case s.AvgMOS >= 3.0:
		s.Level = "fair"
	default:
		s.Level = "poor"
	}
	return s
}
