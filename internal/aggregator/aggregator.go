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

// UserActiveTime sums inter-prompt gaps that are strictly less than idleThreshold.
func UserActiveTime(turns []Turn, idleThreshold time.Duration) time.Duration {
	var total time.Duration
	for i := 1; i < len(turns); i++ {
		gap := turns[i].PromptAt.Sub(turns[i-1].PromptAt)
		if gap < idleThreshold {
			total += gap
		}
	}
	return total
}
