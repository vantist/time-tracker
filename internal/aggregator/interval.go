package aggregator

import (
	"sort"
	"time"
)

type Interval struct{ Start, End time.Time }

// UserIntervals computes user active intervals for a session.
// Interval semantics: [response_at[i-1], prompt_at[i]].
// First interval: [sessionStart, turns[0].PromptAt] if sessionStart is non-zero.
// Intervals with length >= idleThreshold are discarded.
func UserIntervals(turns []Turn, sessionStart time.Time, idleThreshold time.Duration) []Interval {
	var result []Interval

	keep := func(iv Interval) {
		d := iv.End.Sub(iv.Start)
		if d > 0 && d < idleThreshold {
			result = append(result, iv)
		}
	}

	if !sessionStart.IsZero() && len(turns) > 0 {
		keep(Interval{Start: sessionStart, End: turns[0].PromptAt})
	}

	for i := 1; i < len(turns); i++ {
		if turns[i-1].ResponseAt == nil {
			continue
		}
		keep(Interval{Start: *turns[i-1].ResponseAt, End: turns[i].PromptAt})
	}

	return result
}

// MergeAndSum sorts intervals, merges overlaps, and returns total duration.
func MergeAndSum(intervals []Interval) time.Duration {
	if len(intervals) == 0 {
		return 0
	}

	sorted := make([]Interval, len(intervals))
	copy(sorted, intervals)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Start.Before(sorted[j].Start)
	})

	merged := []Interval{sorted[0]}
	for _, iv := range sorted[1:] {
		last := &merged[len(merged)-1]
		if !iv.Start.After(last.End) {
			if iv.End.After(last.End) {
				last.End = iv.End
			}
		} else {
			merged = append(merged, iv)
		}
	}

	var total time.Duration
	for _, iv := range merged {
		total += iv.End.Sub(iv.Start)
	}
	return total
}
