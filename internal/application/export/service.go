package export

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/divord97/ccc/internal/domain/report"
)

// WriteAgentReportCSV writes agent report data as CSV.
func WriteAgentReportCSV(w io.Writer, items []*report.AgentReport) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{
		"Agent ID", "Agent Name", "Total Calls", "Inbound", "Outbound",
		"Answered", "Missed", "Transferred", "Held",
		"Avg Talk(s)", "Total Talk(s)", "Avg Hold(s)", "Avg ACW(s)", "Total ACW(s)",
		"Avg Ring(s)", "Avg Wait(s)", "FCR(%)", "SL 20s(%)", "Answer Rate(%)",
		"Online(s)", "Idle(s)", "Talk(s)", "ACW(s)", "Break(s)", "Dialing(s)",
		"Utilization(%)", "Avg Satisfaction", "CSAT Count",
		"Callback Count", "Internal Count", "Double Call Count",
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, r := range items {
		row := []string{
			fmt.Sprintf("%d", r.AgentID), r.AgentName,
			fmt.Sprintf("%d", r.TotalCalls), fmt.Sprintf("%d", r.InboundCalls), fmt.Sprintf("%d", r.OutboundCalls),
			fmt.Sprintf("%d", r.AnsweredCalls), fmt.Sprintf("%d", r.MissedCalls),
			fmt.Sprintf("%d", r.TransferredCalls), fmt.Sprintf("%d", r.HeldCalls),
			fmt.Sprintf("%.1f", r.AvgTalkDurationSec), fmt.Sprintf("%d", r.TotalTalkDurationSec),
			fmt.Sprintf("%.1f", r.AvgHoldDurationSec), fmt.Sprintf("%.1f", r.AvgACWDurationSec),
			fmt.Sprintf("%d", r.TotalACWDurationSec),
			fmt.Sprintf("%.1f", r.AvgRingDurationSec), fmt.Sprintf("%.1f", r.AvgWaitDurationSec),
			fmt.Sprintf("%.1f", r.FirstCallResolution), fmt.Sprintf("%.1f", r.ServiceLevel20s),
			fmt.Sprintf("%.1f", r.AnswerRate),
			fmt.Sprintf("%d", r.OnlineTimeSec), fmt.Sprintf("%d", r.IdleTimeSec),
			fmt.Sprintf("%d", r.TalkTimeSec), fmt.Sprintf("%d", r.ACWTimeSec),
			fmt.Sprintf("%d", r.BreakTimeSec), fmt.Sprintf("%d", r.DialingTimeSec),
			fmt.Sprintf("%.1f", r.Utilization), fmt.Sprintf("%.1f", r.AvgSatisfaction),
			fmt.Sprintf("%d", r.SatisfactionCount),
			fmt.Sprintf("%d", r.CallbackCount), fmt.Sprintf("%d", r.InternalCallCount),
			fmt.Sprintf("%d", r.DoubleCallCount),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// WriteSkillGroupReportCSV writes skill group report data as CSV.
func WriteSkillGroupReportCSV(w io.Writer, items []*report.SkillGroupReport) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{
		"Skill Group ID", "Skill Group Name", "Total Calls", "Inbound", "Outbound",
		"Answered", "Abandoned", "Queue Total", "Queue Abandoned", "Ring Abandoned",
		"SL 20s(%)", "Avg Wait(s)", "Avg Talk(s)", "Answer Rate(%)", "Agent Count",
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, r := range items {
		row := []string{
			fmt.Sprintf("%d", r.SkillGroupID), r.SkillGroupName,
			fmt.Sprintf("%d", r.TotalCalls), fmt.Sprintf("%d", r.InboundCalls), fmt.Sprintf("%d", r.OutboundCalls),
			fmt.Sprintf("%d", r.AnsweredCalls), fmt.Sprintf("%d", r.AbandonedCalls),
			fmt.Sprintf("%d", r.QueueTotal), fmt.Sprintf("%d", r.QueueAbandoned),
			fmt.Sprintf("%d", r.RingAbandoned),
			fmt.Sprintf("%.1f", r.ServiceLevel20s), fmt.Sprintf("%.1f", r.AvgWaitSec),
			fmt.Sprintf("%.1f", r.AvgTalkSec), fmt.Sprintf("%.1f", r.AnswerRate),
			fmt.Sprintf("%d", r.AgentCount),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// WriteAgentStatusLogCSV writes agent status log data as CSV.
func WriteAgentStatusLogCSV(w io.Writer, items []*report.AgentStatusLog) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{"ID", "Agent ID", "Agent Name", "Status", "Sub State", "Work Mode", "Break Reason", "Duration(s)", "Time"}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, r := range items {
		row := []string{
			fmt.Sprintf("%d", r.ID), fmt.Sprintf("%d", r.AgentID), r.AgentName,
			r.Status, r.SubState, r.WorkMode, r.BreakReasonCode,
			fmt.Sprintf("%d", r.DurationSec), r.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// WriteGroupAgentReportCSV writes group agent report data as CSV.
func WriteGroupAgentReportCSV(w io.Writer, items []*report.GroupAgentReport) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{
		"Skill Group ID", "Skill Group Name", "Agent ID", "Agent Name",
		"Total Calls", "Inbound", "Outbound", "Answered",
		"Avg Talk(s)", "Total Talk(s)",
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, r := range items {
		row := []string{
			fmt.Sprintf("%d", r.SkillGroupID), r.SkillGroupName,
			fmt.Sprintf("%d", r.AgentID), r.AgentName,
			fmt.Sprintf("%d", r.TotalCalls), fmt.Sprintf("%d", r.InboundCalls),
			fmt.Sprintf("%d", r.OutboundCalls), fmt.Sprintf("%d", r.AnsweredCalls),
			fmt.Sprintf("%.1f", r.AvgTalkDurationSec), fmt.Sprintf("%d", r.TotalTalkDurationSec),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return nil
}
