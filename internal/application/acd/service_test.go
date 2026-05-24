package acd

import (
	"testing"
	"time"
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
