package report

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/user/tt/internal/aggregator"
	"github.com/user/tt/internal/config"
)

type Options struct {
	Since   time.Time
	Project string
	ByWorkItem bool
}

type Result struct {
	Empty                bool
	SessionsCount        int
	AgentTimeSec         int64
	UserActiveTimeSec    int64
	InputTokens          int64
	OutputTokens         int64
	CacheReadTokens      int64
	CacheCreationTokens  int64
	EstimatedCostUSD     *float64
	Groups               []GroupResult    // populated when ByWorkItem=true
	ByProject            []ProjectSummary // grouped by session.project
	Daily                []DailyStat      // last 7 days
}

type ProjectSummary struct {
	Project       string   `json:"project"`
	SessionsCount int      `json:"sessions"`
	AgentTimeSec  int64    `json:"agent_time_seconds"`
	CostUSD       *float64 `json:"cost_usd"`
}

type DailyStat struct {
	Date         string `json:"date"`
	Sessions     int    `json:"sessions"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
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
	if v, err := config.Get("idle-threshold"); err == nil && v != "" {
		if mins, err := strconv.Atoi(v); err == nil {
			idleThreshold = time.Duration(mins) * time.Minute
		}
	}

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
		       COALESCE(t.cache_creation_tokens, 0),
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
			&r.inputTok, &r.outputTok, &r.cacheRead, &r.cacheCreate, &r.cost,
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
	// project → {sessions, agent turns, cost}
	type projState struct {
		sessions map[string]struct{}
		turns    []aggregator.Turn
		cost     *float64
	}
	projMap := map[string]*projState{}
	// date → DailyStat
	dailyMap := map[string]*DailyStat{}

	for _, r := range allRows {
		sessTurns[r.sessionID] = append(sessTurns[r.sessionID], aggregator.Turn{
			PromptAt:   r.promptAt,
			ResponseAt: r.responseAt,
		})
		res.InputTokens += r.inputTok
		res.OutputTokens += r.outputTok
		res.CacheReadTokens += r.cacheRead
		res.CacheCreationTokens += r.cacheCreate
		if r.cost != nil {
			if res.EstimatedCostUSD == nil {
				v := 0.0
				res.EstimatedCostUSD = &v
			}
			*res.EstimatedCostUSD += *r.cost
		}

		// by-project accumulation
		ps := projMap[r.project]
		if ps == nil {
			ps = &projState{sessions: map[string]struct{}{}}
			projMap[r.project] = ps
		}
		ps.sessions[r.sessionID] = struct{}{}
		ps.turns = append(ps.turns, aggregator.Turn{PromptAt: r.promptAt, ResponseAt: r.responseAt})
		if r.cost != nil {
			if ps.cost == nil {
				v := 0.0
				ps.cost = &v
			}
			*ps.cost += *r.cost
		}

		// daily accumulation
		date := r.promptAt.UTC().Format("2006-01-02")
		ds := dailyMap[date]
		if ds == nil {
			ds = &DailyStat{Date: date}
			dailyMap[date] = ds
		}
		ds.InputTokens += r.inputTok
		ds.OutputTokens += r.outputTok
	}

	// count sessions per day (one turn per session per day bucket)
	sessDateSeen := map[string]struct{}{} // "date:sessID"
	for _, r := range allRows {
		date := r.promptAt.UTC().Format("2006-01-02")
		key := date + ":" + r.sessionID
		if _, ok := sessDateSeen[key]; !ok {
			sessDateSeen[key] = struct{}{}
			if ds, ok := dailyMap[date]; ok {
				ds.Sessions++
			}
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

	// build ByProject sorted by sessions desc
	for _, ps := range projMap {
		agentSec := int64(aggregator.AgentTime(ps.turns).Seconds())
		res.ByProject = append(res.ByProject, ProjectSummary{
			Project:      "",
			SessionsCount: len(ps.sessions),
			AgentTimeSec:  agentSec,
			CostUSD:       ps.cost,
		})
	}
	// fix Project field (need key from projMap)
	res.ByProject = res.ByProject[:0]
	for proj, ps := range projMap {
		agentSec := int64(aggregator.AgentTime(ps.turns).Seconds())
		res.ByProject = append(res.ByProject, ProjectSummary{
			Project:       proj,
			SessionsCount: len(ps.sessions),
			AgentTimeSec:  agentSec,
			CostUSD:       ps.cost,
		})
	}
	sort.Slice(res.ByProject, func(i, j int) bool {
		return res.ByProject[i].SessionsCount > res.ByProject[j].SessionsCount
	})

	// build Daily sorted by date asc
	for _, ds := range dailyMap {
		res.Daily = append(res.Daily, *ds)
	}
	sort.Slice(res.Daily, func(i, j int) bool {
		return res.Daily[i].Date < res.Daily[j].Date
	})

	return res, nil
}

type rowData struct {
	sessionID   string
	project     string
	branch      string
	workItem    string
	promptAt    time.Time
	responseAt  *time.Time
	inputTok    int64
	outputTok   int64
	cacheRead   int64
	cacheCreate int64
	cost        *float64
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

func formatInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n < 0 {
		s = s[1:]
	}
	var out []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	if n < 0 {
		return "-" + string(out)
	}
	return string(out)
}

const separator = "─────────────────────────────────────────"

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

	// Tokens block
	fmt.Fprintf(&b, "─── Tokens %s\n", separator[:30])
	fmt.Fprintf(&b, "  Input:        %s\n", formatInt(r.InputTokens))
	fmt.Fprintf(&b, "  Output:       %s\n", formatInt(r.OutputTokens))
	fmt.Fprintf(&b, "  Cache read:   %s\n", formatInt(r.CacheReadTokens))
	fmt.Fprintf(&b, "  Cache create: %s\n", formatInt(r.CacheCreationTokens))

	// Cost block
	fmt.Fprintf(&b, "─── Cost %s\n", separator[:32])
	fmt.Fprintf(&b, "  Est. cost:  %s\n", cost)

	// By Project block
	if len(r.ByProject) > 0 {
		fmt.Fprintf(&b, "─── By Project %s\n", separator[:26])
		for _, p := range r.ByProject {
			ph := p.AgentTimeSec / 3600
			pm := (p.AgentTimeSec % 3600) / 60
			pcost := "N/A"
			if p.CostUSD != nil {
				pcost = fmt.Sprintf("$%.4f", *p.CostUSD)
			}
			fmt.Fprintf(&b, "  %-30s  %3d sessions  %dh %dm  %s\n",
				p.Project, p.SessionsCount, ph, pm, pcost)
		}
	}

	return b.String()
}

func FormatJSON(r Result) string {
	m := map[string]interface{}{
		"sessions_count":        r.SessionsCount,
		"agent_time_sec":        r.AgentTimeSec,
		"user_active_time_sec":  r.UserActiveTimeSec,
		"input_tokens":          r.InputTokens,
		"output_tokens":         r.OutputTokens,
		"cache_read_tokens":     r.CacheReadTokens,
		"cache_creation_tokens": r.CacheCreationTokens,
		"estimated_cost_usd":    r.EstimatedCostUSD,
		"by_project":            r.ByProject,
		"daily":                 r.Daily,
	}
	b, _ := json.Marshal(m)
	return string(b)
}
