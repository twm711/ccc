package call

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebRTCQuality_QualityLevel(t *testing.T) {
	tests := []struct {
		mos    float64
		expect string
	}{
		{4.5, "good"},
		{4.0, "good"},
		{3.5, "fair"},
		{3.0, "fair"},
		{2.5, "poor"},
		{1.0, "poor"},
	}
	for _, tt := range tests {
		log := &WebRTCQualityLog{MOS: tt.mos}
		assert.Equal(t, tt.expect, log.QualityLevel(), "MOS %.1f", tt.mos)
	}
}

func TestWebRTCQuality_CreateAndList(t *testing.T) {
	repo := NewMockWebRTCQualityRepo()
	ctx := context.Background()

	log1 := &WebRTCQualityLog{
		CallID: 100, TenantID: 1, AgentID: 10,
		PacketLossRate: 0.02, Jitter: 15.5, RoundTripTime: 80,
		MOS: 4.2, AudioLevel: -30, BitrateKbps: 64,
		CodecName: "opus", SampledAt: time.Now(),
	}
	require.NoError(t, repo.Create(ctx, log1))
	assert.NotZero(t, log1.ID)

	log2 := &WebRTCQualityLog{
		CallID: 100, TenantID: 1, AgentID: 10,
		PacketLossRate: 0.05, Jitter: 25, RoundTripTime: 120,
		MOS: 3.5, SampledAt: time.Now(),
	}
	require.NoError(t, repo.Create(ctx, log2))

	logs, err := repo.ListByCallID(ctx, 100)
	require.NoError(t, err)
	assert.Len(t, logs, 2)

	logs, err = repo.ListByAgent(ctx, 1, 10, 10)
	require.NoError(t, err)
	assert.Len(t, logs, 2)
}

func TestSummarizeQuality_Empty(t *testing.T) {
	s := SummarizeQuality(nil)
	assert.Equal(t, 0, s.SampleCount)
	assert.Equal(t, float64(0), s.AvgMOS)
}

func TestSummarizeQuality_Aggregation(t *testing.T) {
	logs := []WebRTCQualityLog{
		{CallID: 1, MOS: 4.0, Jitter: 10, PacketLossRate: 0.01, RoundTripTime: 50},
		{CallID: 1, MOS: 3.0, Jitter: 30, PacketLossRate: 0.05, RoundTripTime: 150},
		{CallID: 1, MOS: 5.0, Jitter: 5, PacketLossRate: 0.0, RoundTripTime: 20},
	}
	s := SummarizeQuality(logs)
	assert.Equal(t, 3, s.SampleCount)
	assert.Equal(t, int64(1), s.CallID)
	assert.InDelta(t, 4.0, s.AvgMOS, 0.01)
	assert.Equal(t, 3.0, s.MinMOS)
	assert.Equal(t, 5.0, s.MaxMOS)
	assert.InDelta(t, 15.0, s.AvgJitter, 0.01)
	assert.InDelta(t, 0.02, s.AvgLoss, 0.001)
	assert.InDelta(t, 73.33, s.AvgRTT, 0.5)
	assert.Equal(t, "good", s.Level)
}

func TestSummarizeQuality_PoorLevel(t *testing.T) {
	logs := []WebRTCQualityLog{
		{CallID: 2, MOS: 2.0, Jitter: 50, PacketLossRate: 0.1, RoundTripTime: 300},
		{CallID: 2, MOS: 2.5, Jitter: 40, PacketLossRate: 0.08, RoundTripTime: 250},
	}
	s := SummarizeQuality(logs)
	assert.Equal(t, "poor", s.Level)
	assert.Equal(t, 2.0, s.MinMOS)
	assert.Equal(t, 2.5, s.MaxMOS)
}
