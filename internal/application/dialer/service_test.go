package dialer

import (
	"testing"

	"github.com/divord97/ccc/internal/domain/campaign"
	"github.com/rs/zerolog"
)

func TestCalcAbandonRate(t *testing.T) {
	cases := []struct {
		name              string
		abandoned, total  int
		want              float64
	}{
		{"zero total returns zero (no division by zero panic)", 0, 0, 0},
		{"all abandoned", 5, 5, 100},
		{"none abandoned", 0, 100, 0},
		{"half abandoned", 50, 100, 50},
		{"one in three", 1, 3, 33.33333333333333},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := calcAbandonRate(c.abandoned, c.total)
			if got != c.want {
				t.Errorf("calcAbandonRate(%d, %d) = %v, want %v", c.abandoned, c.total, got, c.want)
			}
		})
	}
}

func TestIsWithinSchedule_DefaultHours(t *testing.T) {
	svc := &Service{logger: zerolog.Nop(), active: make(map[int64]*dialerState)}
	c := &campaign.Campaign{ScheduleStartHour: 0, ScheduleEndHour: 0}
	// Default is 9-20. Whether it passes depends on current time; we just verify no panic.
	_ = svc.isWithinSchedule(c)
}

func TestIsWithinSchedule_ExplicitHours(t *testing.T) {
	svc := &Service{logger: zerolog.Nop(), active: make(map[int64]*dialerState)}
	c := &campaign.Campaign{ScheduleStartHour: 0, ScheduleEndHour: 24}
	if !svc.isWithinSchedule(c) {
		t.Error("0-24 schedule should always be within schedule")
	}
}

func TestIsWithinSchedule_OutOfRange(t *testing.T) {
	svc := &Service{logger: zerolog.Nop(), active: make(map[int64]*dialerState)}
	// Hour range that can never match current hour (start > end, both equal and non-zero)
	c := &campaign.Campaign{ScheduleStartHour: 25, ScheduleEndHour: 26}
	if svc.isWithinSchedule(c) {
		t.Error("impossible hour range should not be within schedule")
	}
}

func TestGetStats_EmptyWhenNotActive(t *testing.T) {
	svc := &Service{logger: zerolog.Nop(), active: make(map[int64]*dialerState)}
	stats := svc.GetStats(999)
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.CampaignID != 999 {
		t.Errorf("CampaignID = %d, want 999", stats.CampaignID)
	}
	if stats.IsRunning {
		t.Error("expected IsRunning=false for inactive campaign")
	}
}
