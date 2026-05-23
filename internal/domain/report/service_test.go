package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDashboard_CalculateServiceLevel20s(t *testing.T) {
	// 80 out of 100 calls answered within 20s → 80%
	sl := CalculateServiceLevel20s(80, 100)
	assert.InDelta(t, 80.0, sl, 0.01)

	// 0 offered → 0%
	sl = CalculateServiceLevel20s(0, 0)
	assert.Equal(t, 0.0, sl)

	// All answered within 20s → 100%
	sl = CalculateServiceLevel20s(50, 50)
	assert.InDelta(t, 100.0, sl, 0.01)

	// None answered within 20s → 0%
	sl = CalculateServiceLevel20s(0, 100)
	assert.Equal(t, 0.0, sl)
}

func TestDashboard_CalculateAgentUtilization(t *testing.T) {
	// 3600s talk + 600s ACW + 300s dialing out of 7200s online → 62.5%
	util := CalculateAgentUtilization(3600, 600, 300, 7200)
	assert.InDelta(t, 62.5, util, 0.01)

	// 0 online → 0%
	util = CalculateAgentUtilization(100, 50, 30, 0)
	assert.Equal(t, 0.0, util)

	// All productive → 100%
	util = CalculateAgentUtilization(3000, 500, 500, 4000)
	assert.InDelta(t, 100.0, util, 0.01)
}

func TestDashboard_CallFunnel_Ratios(t *testing.T) {
	funnel := &CallFunnel{
		TotalInbound:   200,
		IVRHandled:     40,
		RobotHandled:   20,
		TransferToHuman: 140,
		FullService:    80,
		HalfService:    30,
		DirectTransfer: 30,
		ActualAnswered: 160,
		Abandoned:      40,
	}

	ivrRate, answerRate, abandonRate := CalculateCallFunnelRatios(funnel)
	// (40+20)/200 = 30%
	assert.InDelta(t, 30.0, ivrRate, 0.01)
	// 160/200 = 80%
	assert.InDelta(t, 80.0, answerRate, 0.01)
	// 40/200 = 20%
	assert.InDelta(t, 20.0, abandonRate, 0.01)
}

func TestDashboard_CallFunnel_ZeroInbound(t *testing.T) {
	funnel := &CallFunnel{TotalInbound: 0}
	ivrRate, answerRate, abandonRate := CalculateCallFunnelRatios(funnel)
	assert.Equal(t, 0.0, ivrRate)
	assert.Equal(t, 0.0, answerRate)
	assert.Equal(t, 0.0, abandonRate)
}

func TestAgentReport_Aggregate_30Fields(t *testing.T) {
	r := &AgentReport{
		AgentID:              1,
		AgentName:            "Agent1",
		TotalCalls:           100,
		InboundCalls:         60,
		OutboundCalls:        40,
		AnsweredCalls:        90,
		MissedCalls:          10,
		TransferredCalls:     5,
		HeldCalls:            15,
		AvgTalkDurationSec:   180.5,
		TotalTalkDurationSec: 18050,
		AvgHoldDurationSec:   30.0,
		AvgACWDurationSec:    45.0,
		TotalACWDurationSec:  4050,
		AvgRingDurationSec:   12.0,
		AvgWaitDurationSec:   25.0,
		FirstCallResolution:  85.0,
		ServiceLevel20s:      80.0,
		AnswerRate:           90.0,
		OnlineTimeSec:        28800,
		IdleTimeSec:          5000,
		TalkTimeSec:          18050,
		ACWTimeSec:           4050,
		BreakTimeSec:         1200,
		DialingTimeSec:       500,
		Utilization:          78.3,
		AvgSatisfaction:      4.5,
		SatisfactionCount:    80,
		CallbackCount:        3,
		InternalCallCount:    5,
		DoubleCallCount:      2,
	}

	assert.Equal(t, 100, r.TotalCalls)
	assert.Equal(t, 60, r.InboundCalls)
	assert.Equal(t, 40, r.OutboundCalls)
	assert.Equal(t, 90, r.AnsweredCalls)
	assert.Equal(t, 10, r.MissedCalls)
	assert.InDelta(t, 180.5, r.AvgTalkDurationSec, 0.01)
	assert.InDelta(t, 80.0, r.ServiceLevel20s, 0.01)
	assert.InDelta(t, 90.0, r.AnswerRate, 0.01)
	assert.InDelta(t, 78.3, r.Utilization, 0.01)
	assert.InDelta(t, 4.5, r.AvgSatisfaction, 0.01)

	// Verify utilization calculation matches
	util := CalculateAgentUtilization(r.TalkTimeSec, r.ACWTimeSec, r.DialingTimeSec, r.OnlineTimeSec)
	// (18050+4050+500)/28800 = 78.47%
	assert.InDelta(t, 78.47, util, 0.1)
}

func TestGroupAgentReport_BySkillGroup(t *testing.T) {
	r := &GroupAgentReport{
		SkillGroupID:   100,
		SkillGroupName: "VIP Support",
		AgentReport: AgentReport{
			AgentID:     1,
			AgentName:   "Agent1",
			TotalCalls:  50,
			AnswerRate:  95.0,
		},
	}

	assert.Equal(t, int64(100), r.SkillGroupID)
	assert.Equal(t, "VIP Support", r.SkillGroupName)
	assert.Equal(t, 50, r.TotalCalls)
	assert.InDelta(t, 95.0, r.AnswerRate, 0.01)
}
