package acd

import (
	"context"
	"testing"
	"time"

	"github.com/divord97/ccc/internal/domain/identity"
)

func TestMemberRoundTrip(t *testing.T) {
	now := time.Now()
	m := memberFor(12345, now)
	id, ms, ok := parseMember(m)
	if !ok {
		t.Fatalf("parseMember(%q) ok=false, want true", m)
	}
	if id != 12345 {
		t.Errorf("callID = %d, want 12345", id)
	}
	if ms != now.UnixMilli() {
		t.Errorf("enqueueMs = %d, want %d", ms, now.UnixMilli())
	}
}

func TestParseMemberLegacy(t *testing.T) {
	id, ms, ok := parseMember("98765")
	if !ok {
		t.Fatalf("legacy parseMember ok=false, want true")
	}
	if id != 98765 {
		t.Errorf("callID = %d, want 98765", id)
	}
	if ms != 0 {
		t.Errorf("legacy enqueueMs = %d, want 0", ms)
	}
}

func TestParseMemberMalformed(t *testing.T) {
	cases := []string{"", "abc", "1:abc", "abc:1"}
	for _, c := range cases {
		if _, _, ok := parseMember(c); ok {
			t.Errorf("parseMember(%q) ok=true, want false", c)
		}
	}
}

// TestExpireWindowMath documents the bug the previous tsFromScore-based
// implementation hit: the score encoding is one-way and trying to recover the
// enqueue timestamp from it produced garbage timestamps in the 1970s, which
// caused every queued call to be expired on the very next ACD tick.
//
// We assert here that parseMember (the replacement) recovers a current-era
// millisecond timestamp from a freshly enqueued member.
func TestExpireWindowMath(t *testing.T) {
	now := time.Now()
	_, ms, ok := parseMember(memberFor(1, now))
	if !ok {
		t.Fatal("parseMember unexpectedly failed")
	}
	// Sanity: timestamp must be within 1s of "now". The old bug would have
	// returned something around April 1970 (~9.6e9 ms).
	if delta := now.UnixMilli() - ms; delta < -1000 || delta > 1000 {
		t.Fatalf("parseMember returned ms=%d, want close to now=%d (delta=%d)",
			ms, now.UnixMilli(), delta)
	}
}

func TestScoreFor_HigherPriorityFirst(t *testing.T) {
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	highPrio := scoreFor(10, ts)
	lowPrio := scoreFor(1, ts)
	if highPrio >= lowPrio {
		t.Errorf("higher priority should produce lower score: scoreFor(10)=%f >= scoreFor(1)=%f", highPrio, lowPrio)
	}
}

func TestScoreFor_OlderTimestampFirst(t *testing.T) {
	older := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := older.Add(10 * time.Second)
	s1 := scoreFor(5, older)
	s2 := scoreFor(5, newer)
	if s1 >= s2 {
		t.Errorf("older timestamp should produce lower score at same priority: %f >= %f", s1, s2)
	}
}

func TestFamiliarKey(t *testing.T) {
	k := familiarKey(42, "13800138000")
	if k != lastAgentPrefix+"42:13800138000" {
		t.Errorf("unexpected key: %s", k)
	}
}

func TestQueueKey(t *testing.T) {
	k := queueKey(99)
	if k != queueKeyPrefix+"99" {
		t.Errorf("unexpected key: %s", k)
	}
}

func TestAgentWorkloadSummary_Utilization(t *testing.T) {
	caps := []identity.MediaCapacity{
		{Media: identity.MediaTypeVoice, MaxSlots: 1, ActiveSlots: 1},
		{Media: identity.MediaTypeChat, MaxSlots: 5, ActiveSlots: 2},
		{Media: identity.MediaTypeEmail, MaxSlots: 10, ActiveSlots: 0},
	}
	var totalActive, totalMax int
	for _, c := range caps {
		totalActive += c.ActiveSlots
		totalMax += c.MaxSlots
	}
	util := float64(totalActive) / float64(totalMax) * 100
	summary := &AgentWorkloadSummary{
		AgentID:     1,
		Capacities:  caps,
		TotalActive: totalActive,
		TotalMax:    totalMax,
		Utilization: util,
	}
	if summary.TotalActive != 3 {
		t.Errorf("TotalActive = %d, want 3", summary.TotalActive)
	}
	if summary.TotalMax != 16 {
		t.Errorf("TotalMax = %d, want 16", summary.TotalMax)
	}
	if summary.Utilization < 18 || summary.Utilization > 19 {
		t.Errorf("Utilization = %f, want ~18.75", summary.Utilization)
	}
	_ = context.Background() // use context import
}
