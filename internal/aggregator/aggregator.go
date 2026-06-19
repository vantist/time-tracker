package aggregator

import "time"

type Turn struct {
	PromptAt   time.Time
	ResponseAt *time.Time
}

// AgentTime sums (response_at - prompt_at) for all turns with a response.
func AgentTime(turns []Turn) time.Duration {
	var total time.Duration
	for _, t := range turns {
		if t.ResponseAt != nil {
			total += t.ResponseAt.Sub(t.PromptAt)
		}
	}
	return total
}

// Deprecated: use UserIntervals + MergeAndSum instead.
// UserActiveTime sums inter-prompt gaps that are strictly less than idleThreshold.
// If sessionStart is non-zero, the gap from sessionStart to the first prompt is also included.
func UserActiveTime(turns []Turn, sessionStart time.Time, idleThreshold time.Duration) time.Duration {
	var total time.Duration
	if !sessionStart.IsZero() && len(turns) > 0 {
		if gap := turns[0].PromptAt.Sub(sessionStart); gap >= 0 && gap < idleThreshold {
			total += gap
		}
	}
	for i := 1; i < len(turns); i++ {
		gap := turns[i].PromptAt.Sub(turns[i-1].PromptAt)
		if gap < idleThreshold {
			total += gap
		}
	}
	return total
}
