package aggregator_test

import (
	"testing"
	"time"

	"github.com/user/tt/internal/aggregator"
)

func ptr(t time.Time) *time.Time { return &t }

func base() time.Time {
	return time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
}

// Task 6.1: 3 turns, 3rd response_at is NULL → agent time = sum of first two gaps = 45s
// turn1: prompt=T, response=T+15s
// turn2: prompt=T+20s, response=T+35s
// turn3: prompt=T+40s, response=nil
// AgentTime = (T+15 - T) + (T+35 - T+20) = 15 + 15 = 30s
// But spec says "3 turns, 3rd response_at NULL, expected 45s":
// turn1: prompt=T, response=T+15s       → 15s
// turn2: prompt=T+20s, response=T+35s  → 15s  (total 30s?)
// Re-reading spec: probably turns overlap, total 45s means different scenario.
// Using: turn1 response=T+15, turn2 response=T+35 → agent=35-0=? No.
// Agent time = sum of (response_at - prompt_at) for turns with response.
// turn1: 15s, turn2: 15s = 30s for 2 completed turns. 3rd has no response → not counted.
// Let's use 3 turns summing to 45s: 15+20+10=45, third nil → 15+20=35. That doesn't work.
// Use: turn1=20s, turn2=25s, turn3=nil → 45s total
func TestAgentTime(t *testing.T) {
	b := base()
	turns := []aggregator.Turn{
		{PromptAt: b, ResponseAt: ptr(b.Add(20 * time.Second))},
		{PromptAt: b.Add(30 * time.Second), ResponseAt: ptr(b.Add(55 * time.Second))},
		{PromptAt: b.Add(60 * time.Second), ResponseAt: nil},
	}
	got := aggregator.AgentTime(turns)
	want := 45 * time.Second
	if got != want {
		t.Errorf("AgentTime = %v, want %v", got, want)
	}
}

func TestAgentTimeNoTurns(t *testing.T) {
	got := aggregator.AgentTime(nil)
	if got != 0 {
		t.Errorf("AgentTime(nil) = %v, want 0", got)
	}
}

// Task 6.3: UserActiveTime — gaps below threshold counted, gaps above not
func TestUserActiveTimeGapBelowThreshold(t *testing.T) {
	b := base()
	threshold := 15 * time.Minute
	turns := []aggregator.Turn{
		{PromptAt: b},
		{PromptAt: b.Add(5 * time.Minute)},   // gap 5m < 15m → counted
		{PromptAt: b.Add(8 * time.Minute)},   // gap 3m < 15m → counted
		{PromptAt: b.Add(30 * time.Minute)},  // gap 22m > 15m → NOT counted
	}
	// active time = gap(5m) + gap(3m) = 8 minutes
	got := aggregator.UserActiveTime(turns, threshold)
	want := 8 * time.Minute
	if got != want {
		t.Errorf("UserActiveTime = %v, want %v", got, want)
	}
}

func TestUserActiveTimeGapAboveThreshold(t *testing.T) {
	b := base()
	threshold := 15 * time.Minute
	turns := []aggregator.Turn{
		{PromptAt: b},
		{PromptAt: b.Add(20 * time.Minute)}, // gap 20m > 15m → NOT counted
	}
	got := aggregator.UserActiveTime(turns, threshold)
	if got != 0 {
		t.Errorf("UserActiveTime = %v, want 0", got)
	}
}

func TestUserActiveTimeCustomThreshold(t *testing.T) {
	b := base()
	turns := []aggregator.Turn{
		{PromptAt: b},
		{PromptAt: b.Add(25 * time.Minute)},
	}
	// with threshold=30m, gap 25m < 30m → counted
	got := aggregator.UserActiveTime(turns, 30*time.Minute)
	want := 25 * time.Minute
	if got != want {
		t.Errorf("UserActiveTime = %v, want %v", got, want)
	}
}
