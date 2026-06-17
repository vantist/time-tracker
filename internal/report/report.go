package report

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/user/tt/internal/aggregator"
)

type Options struct {
	Since   time.Time
	Project string
	ByWorkItem bool
}

type Result struct {
	Empty             bool
	SessionsCount     int
	AgentTimeSec      int64
	UserActiveTimeSec int64
	InputTokens       int64
	OutputTokens      int64
	CacheReadTokens   int64
	EstimatedCostUSD  *float64
	Groups            []GroupResult // populated when ByWorkItem=true
}

type GroupResult struct {
	Label             string
	SessionsCount     int
	AgentTimeSec      int64
	UserActiveTimeSec int64
	EstimatedCostUSD  *float64
}

func Query(conn *sql.DB, opts Options) (Result, error) {
	idleThreshold := 15 * time.Minute

	projectFilter := ""
	args := []interface{}{opts.Since.UTC().Format(time.RFC3339)}
	if opts.Project != "" {
		projectFilter = " AND (s.project LIKE ? OR s.project LIKE ?)"
		args = append(args, "%/"+opts.Project, "%/"+opts.Project+"/%")
	}

	rows, err := conn.Query(`
		SELECT s.id, s.project, s.branch, s.work_item,
		       t.prompt_at, t.response_at,
		       COALESCE(t.input_tokens, 0),
		       COALESCE(t.output_tokens, 0),
		       COALESCE(t.cache_read_tokens, 0),
		       t.estimated_cost_usd
		FROM turns t
		JOIN sessions s ON s.id = t.session_id
		WHERE t.prompt_at >= ?`+projectFilter+`
		ORDER BY s.id, t.id`,
		args...,
	)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()

	var allRows []rowData
	sessionSet := map[string]struct{}{}

	for rows.Next() {
		var r rowData
		var promptStr, responseStr sql.NullString
		var wi sql.NullString

		if err := rows.Scan(
			&r.sessionID, &r.project, &r.branch, &wi,
			&promptStr, &responseStr,
			&r.inputTok, &r.outputTok, &r.cacheRead, &r.cost,
		); err != nil {
			return Result{}, err
		}
		r.workItem = wi.String
		if promptStr.Valid {
			r.promptAt, _ = time.Parse(time.RFC3339, promptStr.String)
		}
		if responseStr.Valid {
			t, _ := time.Parse(time.RFC3339, responseStr.String)
			r.responseAt = &t
		}
		allRows = append(allRows, r)
		sessionSet[r.sessionID] = struct{}{}
	}

	if len(allRows) == 0 {
		return Result{Empty: true}, nil
	}

	// aggregate
	var res Result
	res.SessionsCount = len(sessionSet)

	// build Turn slices per session for time aggregation
	sessTurns := map[string][]aggregator.Turn{}
	for _, r := range allRows {
		sessTurns[r.sessionID] = append(sessTurns[r.sessionID], aggregator.Turn{
			PromptAt:   r.promptAt,
			ResponseAt: r.responseAt,
		})
		res.InputTokens += r.inputTok
		res.OutputTokens += r.outputTok
		res.CacheReadTokens += r.cacheRead
		if r.cost != nil {
			if res.EstimatedCostUSD == nil {
				v := 0.0
				res.EstimatedCostUSD = &v
			}
			*res.EstimatedCostUSD += *r.cost
		}
	}

	var totalAgent, totalUser time.Duration
	for _, turns := range sessTurns {
		totalAgent += aggregator.AgentTime(turns)
		totalUser += aggregator.UserActiveTime(turns, idleThreshold)
	}
	res.AgentTimeSec = int64(totalAgent.Seconds())
	res.UserActiveTimeSec = int64(totalUser.Seconds())

	if opts.ByWorkItem {
		res.Groups = groupByWorkItem(allRows, sessTurns, idleThreshold)
	}

	return res, nil
}

type rowData struct {
	sessionID  string
	project    string
	branch     string
	workItem   string
	promptAt   time.Time
	responseAt *time.Time
	inputTok   int64
	outputTok  int64
	cacheRead  int64
	cost       *float64
}

func groupByWorkItem(rows []rowData, sessTurns map[string][]aggregator.Turn, idleThreshold time.Duration) []GroupResult {
	type groupState struct {
		sessions map[string]struct{}
		turns    []aggregator.Turn
		cost     *float64
	}
	groups := map[string]*groupState{}
	labelOf := map[string]string{} // sessionID → label

	for _, r := range rows {
		if _, seen := labelOf[r.sessionID]; !seen {
			label := r.workItem
			if label == "" {
				label = r.branch
			}
			if label == "" {
				label = "untagged"
			}
			labelOf[r.sessionID] = label
		}
		label := labelOf[r.sessionID]
		g := groups[label]
		if g == nil {
			g = &groupState{sessions: map[string]struct{}{}}
			groups[label] = g
		}
		g.sessions[r.sessionID] = struct{}{}
		if r.cost != nil {
			if g.cost == nil {
				v := 0.0
				g.cost = &v
			}
			*g.cost += *r.cost
		}
	}

	for sessID, turns := range sessTurns {
		label := labelOf[sessID]
		if g, ok := groups[label]; ok {
			g.turns = append(g.turns, turns...)
		}
	}

	var result []GroupResult
	for label, g := range groups {
		agentSec := int64(aggregator.AgentTime(g.turns).Seconds())
		userSec := int64(aggregator.UserActiveTime(g.turns, idleThreshold).Seconds())
		result = append(result, GroupResult{
			Label:             label,
			SessionsCount:     len(g.sessions),
			AgentTimeSec:      agentSec,
			UserActiveTimeSec: userSec,
			EstimatedCostUSD:  g.cost,
		})
	}
	return result
}

func FormatText(r Result) string {
	if r.Empty {
		return "No data for the selected period.\n"
	}

	agentH := r.AgentTimeSec / 3600
	agentM := (r.AgentTimeSec % 3600) / 60
	userH := r.UserActiveTimeSec / 3600
	userM := (r.UserActiveTimeSec % 3600) / 60

	cost := "N/A"
	if r.EstimatedCostUSD != nil {
		cost = fmt.Sprintf("$%.4f", *r.EstimatedCostUSD)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Sessions:    %d\n", r.SessionsCount)
	fmt.Fprintf(&b, "Agent time:  %dh %dm\n", agentH, agentM)
	fmt.Fprintf(&b, "User active: %dh %dm\n", userH, userM)
	fmt.Fprintf(&b, "Tokens in:   %d\n", r.InputTokens)
	fmt.Fprintf(&b, "Est. cost:   %s\n", cost)
	return b.String()
}

func FormatJSON(r Result) string {
	m := map[string]interface{}{
		"sessions_count":       r.SessionsCount,
		"agent_time_sec":       r.AgentTimeSec,
		"user_active_time_sec": r.UserActiveTimeSec,
		"input_tokens":         r.InputTokens,
		"output_tokens":        r.OutputTokens,
		"estimated_cost_usd":   r.EstimatedCostUSD,
	}
	b, _ := json.Marshal(m)
	return string(b)
}
