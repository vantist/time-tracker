package report

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
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

type SessionRow struct {
	ID           string   `json:"id"`
	Project      string   `json:"project"`
	Branch       string   `json:"branch"`
	Tool         string   `json:"tool"`
	Model        string   `json:"model"`
	StartedAt    string   `json:"started_at"`
	WorkItem     string   `json:"work_item"`
	Turns        int      `json:"turns"`
	AgentTimeSec int64    `json:"agent_time_sec"`
	UserTimeSec  int64    `json:"user_time_sec"`
	CostUSD      *float64 `json:"cost_usd"`
}

type AgentSummary struct {
	Agent     string  `json:"agent"`
	Sessions  int     `json:"sessions"`
	AgentTime string  `json:"agent_time"`
	UserTime  string  `json:"user_time"`
	Tokens    string  `json:"tokens"`
	Cost      float64 `json:"cost"`
}

type Result struct {
	Empty                bool             `json:"-"`
	SessionsCount        int              `json:"sessions_count"`
	AgentTimeSec         int64            `json:"agent_time_sec"`
	UserActiveTimeSec    int64            `json:"user_active_time_sec"`
	InputTokens          int64            `json:"input_tokens"`
	OutputTokens         int64            `json:"output_tokens"`
	CacheReadTokens      int64            `json:"cache_read_tokens"`
	CacheCreationTokens  int64            `json:"cache_creation_tokens"`
	EstimatedCostUSD     *float64         `json:"estimated_cost_usd"`
	Groups               []GroupResult    `json:"groups"`
	ByProject            []ProjectSummary `json:"by_project"`
	ByAgent              []AgentSummary   `json:"by_agent"`
	Daily                []DailyStat      `json:"daily"`
	Sessions             []SessionRow     `json:"sessions"`
	ByWorkItem           bool             `json:"-"`
}

type ProjectSummary struct {
	Project            string   `json:"project"`
	SessionsCount      int      `json:"sessions"`
	AgentTimeSec       int64    `json:"agent_time_seconds"`
	UserActiveTimeSec  int64    `json:"user_active_time_sec"`
	CostUSD            *float64 `json:"cost_usd"`
	InputTokens        int64    `json:"input_tokens"`
	OutputTokens       int64    `json:"output_tokens"`
}

type DailyStat struct {
	Date         string `json:"date"`
	Sessions     int    `json:"sessions"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
}

type GroupResult struct {
	Label             string   `json:"label"`
	Project           string   `json:"project"`
	SessionsCount     int      `json:"sessions_count"`
	AgentTimeSec      int64    `json:"agent_time_sec"`
	UserActiveTimeSec int64    `json:"user_active_time_sec"`
	EstimatedCostUSD  *float64 `json:"estimated_cost_usd"`
}

func Query(conn *sql.DB, opts Options) (Result, error) {
	idleThreshold := 10 * time.Minute
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
		       COALESCE(s.model, ''), s.started_at,
		       t.prompt_at, t.response_at,
		       COALESCE(t.input_tokens, 0),
		       COALESCE(t.output_tokens, 0),
		       COALESCE(t.cache_read_tokens, 0),
		       COALESCE(t.cache_creation_tokens, 0),
		       t.estimated_cost_usd,
		       COALESCE(s.tool, '')
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
		var startedAt sql.NullString
		var toolStr sql.NullString

		if err := rows.Scan(
			&r.sessionID, &r.project, &r.branch, &wi,
			&r.model, &startedAt,
			&promptStr, &responseStr,
			&r.inputTok, &r.outputTok, &r.cacheRead, &r.cacheCreate, &r.cost,
			&toolStr,
		); err != nil {
			return Result{}, err
		}
		r.tool = normalizeAgentName(toolStr.String)
		r.workItem = wi.String
		r.startedAt = startedAt.String
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
	// session metadata keyed by sessionID
	type sessState struct {
		project   string
		branch    string
		tool      string
		model     string
		startedAt string
		workItem  string
		turns     int
		cost      *float64
	}
	sessMap := map[string]*sessState{}
	// project → {sessions, agent turns, cost}
	type projState struct {
		sessions     map[string]struct{}
		turns        []aggregator.Turn
		cost         *float64
		inputTokens  int64
		outputTokens int64
	}
	projMap := map[string]*projState{}
	// date → DailyStat
	dailyMap := map[string]*DailyStat{}
	sessDateSeen := map[string]struct{}{} // "date:sessID"

	for _, r := range allRows {
		sessTurns[r.sessionID] = append(sessTurns[r.sessionID], aggregator.Turn{
			PromptAt:   r.promptAt,
			ResponseAt: r.responseAt,
		})
		res.InputTokens += r.inputTok
		res.OutputTokens += r.outputTok
		res.CacheReadTokens += r.cacheRead
		res.CacheCreationTokens += r.cacheCreate
		addCost(&res.EstimatedCostUSD, r.cost)

		// per-session accumulation
		ss := sessMap[r.sessionID]
		if ss == nil {
			ss = &sessState{project: r.project, branch: r.branch, tool: r.tool, model: r.model, startedAt: r.startedAt, workItem: r.workItem}
			sessMap[r.sessionID] = ss
		}
		ss.turns++
		addCost(&ss.cost, r.cost)

		// by-project accumulation
		ps := projMap[r.project]
		if ps == nil {
			ps = &projState{sessions: map[string]struct{}{}}
			projMap[r.project] = ps
		}
		ps.sessions[r.sessionID] = struct{}{}
		ps.turns = append(ps.turns, aggregator.Turn{PromptAt: r.promptAt, ResponseAt: r.responseAt})
		ps.inputTokens += r.inputTok
		ps.outputTokens += r.outputTok
		addCost(&ps.cost, r.cost)

		// daily accumulation
		date := r.promptAt.UTC().Format("2006-01-02")
		ds := dailyMap[date]
		if ds == nil {
			ds = &DailyStat{Date: date}
			dailyMap[date] = ds
		}
		ds.InputTokens += r.inputTok
		ds.OutputTokens += r.outputTok

		// count sessions per day (one turn per session per day bucket)
		key := date + ":" + r.sessionID
		if _, ok := sessDateSeen[key]; !ok {
			sessDateSeen[key] = struct{}{}
			ds.Sessions++
		}
	}

	// Build per-session user intervals for cross-session merge
	sessUserIntervals := make(map[string][]aggregator.Interval, len(sessTurns))
	var totalAgent time.Duration
	for sid, turns := range sessTurns {
		totalAgent += aggregator.AgentTime(turns)
		var sessStart time.Time
		if ss := sessMap[sid]; ss != nil {
			sessStart, _ = time.Parse(time.RFC3339, ss.startedAt)
		}
		sessUserIntervals[sid] = aggregator.UserIntervals(turns, sessStart, idleThreshold)
	}

	// Total user time: collect all intervals across sessions then merge
	var allIntervals []aggregator.Interval
	for _, ivs := range sessUserIntervals {
		allIntervals = append(allIntervals, ivs...)
	}
	res.AgentTimeSec = int64(totalAgent.Seconds())
	res.UserActiveTimeSec = int64(aggregator.MergeAndSum(allIntervals).Seconds())

	res.Groups = groupByWorkItem(allRows, sessUserIntervals)
	res.ByWorkItem = opts.ByWorkItem

	// build ByProject sorted by sessions desc
	for proj, ps := range projMap {
		agentSec := int64(aggregator.AgentTime(ps.turns).Seconds())
		var projIntervals []aggregator.Interval
		for sid := range ps.sessions {
			projIntervals = append(projIntervals, sessUserIntervals[sid]...)
		}
		userSec := int64(aggregator.MergeAndSum(projIntervals).Seconds())
		res.ByProject = append(res.ByProject, ProjectSummary{
			Project:           proj,
			SessionsCount:     len(ps.sessions),
			AgentTimeSec:      agentSec,
			UserActiveTimeSec: userSec,
			CostUSD:           ps.cost,
			InputTokens:       ps.inputTokens,
			OutputTokens:      ps.outputTokens,
		})
	}
	sort.Slice(res.ByProject, func(i, j int) bool {
		return res.ByProject[i].SessionsCount > res.ByProject[j].SessionsCount
	})

	// build Sessions sorted by started_at desc
	for sid, ss := range sessMap {
		agentSec := int64(aggregator.AgentTime(sessTurns[sid]).Seconds())
		userSec := int64(aggregator.MergeAndSum(sessUserIntervals[sid]).Seconds())
		res.Sessions = append(res.Sessions, SessionRow{
			ID:           sid,
			Project:      ss.project,
			Branch:       ss.branch,
			Tool:         ss.tool,
			Model:        ss.model,
			StartedAt:    ss.startedAt,
			WorkItem:     ss.workItem,
			Turns:        ss.turns,
			AgentTimeSec: agentSec,
			UserTimeSec:  userSec,
			CostUSD:      ss.cost,
		})
	}
	sort.Slice(res.Sessions, func(i, j int) bool {
		return res.Sessions[i].StartedAt > res.Sessions[j].StartedAt
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
	tool        string
	model       string
	startedAt   string
	workItem    string
	promptAt    time.Time
	responseAt  *time.Time
	inputTok    int64
	outputTok   int64
	cacheRead   int64
	cacheCreate int64
	cost        *float64
}

func groupByWorkItem(rows []rowData, sessUserIntervals map[string][]aggregator.Interval) []GroupResult {
	type groupKey struct{ project, label string }
	type groupState struct {
		project  string
		label    string
		sessions map[string]struct{}
		turns    []aggregator.Turn
		cost     *float64
	}
	groups := map[groupKey]*groupState{}
	sessGroup := map[string]*groupState{} // sessionID → group pointer

	for _, r := range rows {
		g, seen := sessGroup[r.sessionID]
		if !seen {
			label := r.workItem
			if label == "" {
				label = r.branch
			}
			if label == "" {
				label = "untagged"
			}
			key := groupKey{r.project, label}
			g = groups[key]
			if g == nil {
				g = &groupState{project: r.project, label: label, sessions: map[string]struct{}{}}
				groups[key] = g
			}
			sessGroup[r.sessionID] = g
			g.sessions[r.sessionID] = struct{}{}
		}
		g.turns = append(g.turns, aggregator.Turn{PromptAt: r.promptAt, ResponseAt: r.responseAt})
		addCost(&g.cost, r.cost)
	}

	var result []GroupResult
	for _, g := range groups {
		agentSec := int64(aggregator.AgentTime(g.turns).Seconds())
		var groupIntervals []aggregator.Interval
		for sid := range g.sessions {
			groupIntervals = append(groupIntervals, sessUserIntervals[sid]...)
		}
		userSec := int64(aggregator.MergeAndSum(groupIntervals).Seconds())
		result = append(result, GroupResult{
			Label:             g.label,
			Project:           filepath.Base(g.project),
			SessionsCount:     len(g.sessions),
			AgentTimeSec:      agentSec,
			UserActiveTimeSec: userSec,
			EstimatedCostUSD:  g.cost,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].AgentTimeSec > result[j].AgentTimeSec
	})
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
	fmt.Fprintf(&b, "User active: %dh %dm\n\n", userH, userM)

	// Tokens block
	fmt.Fprintf(&b, "─── Tokens ───\n")
	fmt.Fprintf(&b, "  Input:        %s\n", formatInt(r.InputTokens))
	fmt.Fprintf(&b, "  Output:       %s\n", formatInt(r.OutputTokens))
	fmt.Fprintf(&b, "  Cache read:   %s\n", formatInt(r.CacheReadTokens))
	fmt.Fprintf(&b, "  Cache create: %s\n\n", formatInt(r.CacheCreationTokens))

	// Cost block
	fmt.Fprintf(&b, "─── Cost ───\n")
	fmt.Fprintf(&b, "  Est. cost:  %s\n\n", cost)

	// Daily breakdown table
	if len(r.Daily) > 0 {
		fmt.Fprintf(&b, "─── Daily (Last 7 Days) ───\n")
		fmt.Fprintf(&b, "%-12s  %8s  %12s  %13s\n", "Date", "Sessions", "Input Tokens", "Output Tokens")
		for _, stat := range r.Daily {
			fmt.Fprintf(&b, "%-12s  %8d  %12s  %13s\n", stat.Date, stat.Sessions, formatInt(stat.InputTokens), formatInt(stat.OutputTokens))
		}
		fmt.Fprintf(&b, "\n")
	}

	// By Project table
	if len(r.ByProject) > 0 {
		fmt.Fprintf(&b, "─── By Project ───\n")
		fmt.Fprintf(&b, "%-20s  %8s  %10s  %11s  %15s  %8s\n", "Project", "Sessions", "Agent Time", "User Active", "Tokens (I/O)", "Cost")
		for _, p := range r.ByProject {
			pcost := "N/A"
			if p.CostUSD != nil {
				pcost = fmt.Sprintf("$%.4f", *p.CostUSD)
			}
			fmt.Fprintf(&b, "%-20s  %8d  %10s  %11s  %15s  %8s\n",
				filepath.Base(p.Project),
				p.SessionsCount,
				formatTime(p.AgentTimeSec),
				formatTime(p.UserActiveTimeSec),
				fmt.Sprintf("%s / %s", formatInt(p.InputTokens), formatInt(p.OutputTokens)),
				pcost,
			)
		}
		fmt.Fprintf(&b, "\n")
	}

	// By Work Item table
	if len(r.Groups) > 0 && (len(r.Groups) > 1 || r.ByWorkItem) {
		fmt.Fprintf(&b, "─── By Work Item ───\n")
		fmt.Fprintf(&b, "%-20s  %-15s  %8s  %10s  %11s  %8s\n", "Work Item", "Project", "Sessions", "Agent Time", "User Active", "Cost")
		for _, g := range r.Groups {
			gcost := "N/A"
			if g.EstimatedCostUSD != nil {
				gcost = fmt.Sprintf("$%.4f", *g.EstimatedCostUSD)
			}
			fmt.Fprintf(&b, "%-20s  %-15s  %8d  %10s  %11s  %8s\n",
				g.Label,
				g.Project,
				g.SessionsCount,
				formatTime(g.AgentTimeSec),
				formatTime(g.UserActiveTimeSec),
				gcost,
			)
		}
		fmt.Fprintf(&b, "\n")
	}

	// Sessions log table
	if len(r.Sessions) > 0 {
		fmt.Fprintf(&b, "─── Sessions ───\n")
		fmt.Fprintf(&b, "%-19s  %-15s  %-12s  %-20s  %5s  %10s  %9s  %-15s  %8s\n",
			"Start Time", "Project", "Branch", "Model", "Turns", "Agent Time", "User Time", "Work Item", "Cost")
		for _, s := range r.Sessions {
			scost := "N/A"
			if s.CostUSD != nil {
				scost = fmt.Sprintf("$%.4f", *s.CostUSD)
			}
			var startTimeStr string
			if t, err := time.Parse(time.RFC3339, s.StartedAt); err == nil {
				startTimeStr = t.Local().Format("2006-01-02 15:04:05")
			} else {
				startTimeStr = s.StartedAt
			}
			fmt.Fprintf(&b, "%-19s  %-15s  %-12s  %-20s  %5d  %10s  %9s  %-15s  %8s\n",
				startTimeStr,
				filepath.Base(s.Project),
				s.Branch,
				s.Model,
				s.Turns,
				formatTime(s.AgentTimeSec),
				formatTime(s.UserTimeSec),
				s.WorkItem,
				scost,
			)
		}
	}

	return b.String()
}

func formatTime(sec int64) string {
	h := sec / 3600
	m := (sec % 3600) / 60
	return fmt.Sprintf("%dh %dm", h, m)
}

func FormatJSON(r Result) string {
	b, _ := json.Marshal(r)
	return string(b)
}

func addCost(dst **float64, val *float64) {
	if val == nil {
		return
	}
	if *dst == nil {
		v := 0.0
		*dst = &v
	}
	**dst += *val
}

func normalizeAgentName(tool string) string {
	tool = strings.TrimSpace(strings.ToLower(tool))
	if tool == "" {
		return "unknown"
	}
	switch tool {
	case "claude-code", "claudecode", "claude":
		return "Claude Code"
	case "copilot-cli", "copilotcli", "copilot":
		return "Copilot CLI"
	default:
		return tool
	}
}

