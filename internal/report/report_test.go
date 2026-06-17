package report_test

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/user/tt/internal/db"
	"github.com/user/tt/internal/report"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	t.Setenv("TT_DB_PATH", filepath.Join(t.TempDir(), "test.db"))
	conn, err := db.Open()
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func insertSession(t *testing.T, conn *sql.DB, id, project, branch, workItem string) {
	t.Helper()
	_, err := conn.Exec(
		`INSERT OR IGNORE INTO sessions (id, project, branch, work_item, started_at) VALUES (?,?,?,?,?)`,
		id, project, branch, workItem, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insertSession: %v", err)
	}
}

func insertTurn(t *testing.T, conn *sql.DB, sessionID string, promptAt time.Time, responseAt *time.Time, cost *float64) {
	t.Helper()
	var ra interface{}
	if responseAt != nil {
		ra = responseAt.UTC().Format(time.RFC3339)
	}
	_, err := conn.Exec(
		`INSERT INTO turns (session_id, prompt_at, response_at, estimated_cost_usd) VALUES (?,?,?,?)`,
		sessionID, promptAt.UTC().Format(time.RFC3339), ra, cost,
	)
	if err != nil {
		t.Fatalf("insertTurn: %v", err)
	}
}

func insertTurnFull(t *testing.T, conn *sql.DB, sessionID string, promptAt time.Time, responseAt *time.Time,
	inputTok, outputTok, cacheRead, cacheCreate int64, cost *float64) {
	t.Helper()
	var ra interface{}
	if responseAt != nil {
		ra = responseAt.UTC().Format(time.RFC3339)
	}
	_, err := conn.Exec(
		`INSERT INTO turns (session_id, prompt_at, response_at, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd) VALUES (?,?,?,?,?,?,?,?)`,
		sessionID, promptAt.UTC().Format(time.RFC3339), ra, inputTok, outputTok, cacheRead, cacheCreate, cost,
	)
	if err != nil {
		t.Fatalf("insertTurnFull: %v", err)
	}
}

func ptr[T any](v T) *T { return &v }

// Task 6.5: no data → "No data for the selected period."
func TestReportNoData(t *testing.T) {
	conn := openTestDB(t)

	result, err := report.Query(conn, report.Options{Since: time.Now().Add(-7 * 24 * time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if !result.Empty {
		t.Error("expected Empty=true when no data")
	}
}

// --since 7d filtering
func TestReportSinceFilter(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().UTC()

	insertSession(t, conn, "s1", "/proj", "main", "")
	insertSession(t, conn, "s2", "/proj", "main", "")

	// turn within 7 days
	insertTurn(t, conn, "s1", now.Add(-3*24*time.Hour), ptr(now.Add(-3*24*time.Hour+time.Minute)), ptr(0.005))
	// turn outside 7 days
	insertTurn(t, conn, "s2", now.Add(-10*24*time.Hour), ptr(now.Add(-10*24*time.Hour+time.Minute)), ptr(0.003))

	result, err := report.Query(conn, report.Options{Since: now.Add(-7 * 24 * time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if result.Empty {
		t.Fatal("expected data")
	}
	if result.SessionsCount != 1 {
		t.Errorf("sessions = %d, want 1", result.SessionsCount)
	}
}

// --project filter
func TestReportProjectFilter(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().UTC()

	insertSession(t, conn, "p1", "/home/user/time-tracker", "main", "")
	insertSession(t, conn, "p2", "/home/user/other-project", "main", "")

	insertTurn(t, conn, "p1", now.Add(-time.Hour), ptr(now.Add(-time.Hour+time.Minute)), ptr(0.001))
	insertTurn(t, conn, "p2", now.Add(-time.Hour), ptr(now.Add(-time.Hour+time.Minute)), ptr(0.002))

	result, err := report.Query(conn, report.Options{
		Since:   now.Add(-24 * time.Hour),
		Project: "time-tracker",
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if result.SessionsCount != 1 {
		t.Errorf("sessions = %d, want 1", result.SessionsCount)
	}
}

// text format
func TestFormatText(t *testing.T) {
	r := report.Result{
		SessionsCount:     3,
		AgentTimeSec:      int64(2*3600 + 34*60),
		UserActiveTimeSec: int64(1*3600 + 10*60),
		InputTokens:       10000,
		OutputTokens:      2000,
		EstimatedCostUSD:  ptr(0.042),
	}
	text := report.FormatText(r)
	for _, want := range []string{"Sessions:", "Agent time:", "User active:", "─── Tokens", "Est. cost:"} {
		if !strings.Contains(text, want) {
			t.Errorf("text output missing %q", want)
		}
	}
	if !strings.Contains(text, "2h 34m") {
		t.Errorf("agent time format wrong, got: %s", text)
	}
}

// Task 2.1: CacheCreationTokens sum
func TestQueryCacheCreationTokens(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().UTC()
	insertSession(t, conn, "s1", "/proj", "main", "")
	insertTurnFull(t, conn, "s1", now.Add(-time.Hour), ptr(now.Add(-time.Hour+time.Minute)), 100, 50, 30, 20, nil)
	insertTurnFull(t, conn, "s1", now.Add(-30*time.Minute), ptr(now.Add(-30*time.Minute+time.Minute)), 200, 80, 60, 40, nil)

	result, err := report.Query(conn, report.Options{Since: now.Add(-24 * time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if result.CacheCreationTokens != 60 {
		t.Errorf("CacheCreationTokens = %d, want 60", result.CacheCreationTokens)
	}
}

// Task 2.2: ByProject grouping sorted by sessions desc
func TestQueryByProject(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().UTC()
	insertSession(t, conn, "s1", "/alpha", "main", "")
	insertSession(t, conn, "s2", "/alpha", "main", "")
	insertSession(t, conn, "s3", "/beta", "main", "")
	insertTurnFull(t, conn, "s1", now.Add(-time.Hour), ptr(now.Add(-time.Hour+time.Minute)), 0, 0, 0, 0, ptr(0.01))
	insertTurnFull(t, conn, "s2", now.Add(-time.Hour), ptr(now.Add(-time.Hour+time.Minute)), 0, 0, 0, 0, ptr(0.02))
	insertTurnFull(t, conn, "s3", now.Add(-time.Hour), ptr(now.Add(-time.Hour+time.Minute)), 0, 0, 0, 0, ptr(0.03))

	result, err := report.Query(conn, report.Options{Since: now.Add(-24 * time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.ByProject) != 2 {
		t.Fatalf("ByProject len = %d, want 2", len(result.ByProject))
	}
	// first entry must be alpha (2 sessions)
	if result.ByProject[0].Project != "/alpha" {
		t.Errorf("ByProject[0].Project = %q, want /alpha", result.ByProject[0].Project)
	}
	if result.ByProject[0].SessionsCount != 2 {
		t.Errorf("ByProject[0].SessionsCount = %d, want 2", result.ByProject[0].SessionsCount)
	}
}

// Task 2.3: project with no cost → CostUSD nil
func TestQueryByProjectNilCost(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().UTC()
	insertSession(t, conn, "s1", "/nocost", "main", "")
	insertTurnFull(t, conn, "s1", now.Add(-time.Hour), ptr(now.Add(-time.Hour+time.Minute)), 0, 0, 0, 0, nil)

	result, err := report.Query(conn, report.Options{Since: now.Add(-24 * time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.ByProject) != 1 {
		t.Fatalf("ByProject len = %d, want 1", len(result.ByProject))
	}
	if result.ByProject[0].CostUSD != nil {
		t.Errorf("CostUSD should be nil, got %v", result.ByProject[0].CostUSD)
	}
}

// Task 4.1: FormatText contains Tokens block
func TestFormatTextTokensBlock(t *testing.T) {
	r := report.Result{
		SessionsCount:       1,
		AgentTimeSec:        60,
		InputTokens:         1000,
		OutputTokens:        500,
		CacheReadTokens:     200,
		CacheCreationTokens: 100,
		EstimatedCostUSD:    ptr(0.01),
	}
	text := report.FormatText(r)
	for _, want := range []string{"─── Tokens", "Input:", "Output:", "Cache read:", "Cache create:"} {
		if !strings.Contains(text, want) {
			t.Errorf("FormatText missing %q\ngot:\n%s", want, text)
		}
	}
	// check comma-formatted numbers
	if !strings.Contains(text, "1,000") {
		t.Errorf("FormatText: Input tokens should be comma-formatted, got:\n%s", text)
	}
}

// Task 4.2: FormatText contains By Project block
func TestFormatTextByProject(t *testing.T) {
	costVal := 0.05
	r := report.Result{
		SessionsCount: 1,
		ByProject: []report.ProjectSummary{
			{Project: "myproj", SessionsCount: 2, AgentTimeSec: 3600, CostUSD: &costVal},
		},
	}
	text := report.FormatText(r)
	if !strings.Contains(text, "─── By Project") {
		t.Errorf("FormatText missing By Project block:\n%s", text)
	}
	if !strings.Contains(text, "myproj") {
		t.Errorf("FormatText missing project name:\n%s", text)
	}
	if !strings.Contains(text, "$0.0500") {
		t.Errorf("FormatText missing cost:\n%s", text)
	}
}

// Task 4.3: project CostUSD nil → "N/A"
func TestFormatTextByProjectNoCost(t *testing.T) {
	r := report.Result{
		SessionsCount: 1,
		ByProject: []report.ProjectSummary{
			{Project: "nocost", SessionsCount: 1, AgentTimeSec: 120, CostUSD: nil},
		},
	}
	text := report.FormatText(r)
	if !strings.Contains(text, "N/A") {
		t.Errorf("FormatText: nil CostUSD should show N/A:\n%s", text)
	}
}

// --format json
func TestFormatJSON(t *testing.T) {
	r := report.Result{
		SessionsCount:     2,
		AgentTimeSec:      120,
		UserActiveTimeSec: 60,
		InputTokens:       500,
		OutputTokens:      100,
		EstimatedCostUSD:  ptr(0.002),
	}
	out := report.FormatJSON(r)
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	for _, key := range []string{"sessions_count", "agent_time_sec", "user_active_time_sec", "input_tokens", "output_tokens", "estimated_cost_usd"} {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON missing key %q", key)
		}
	}
}

// Task 8.1: Daily breakdown sorted by date, days without sessions not in array
func TestQueryDailyBreakdown(t *testing.T) {
	conn := openTestDB(t)
	// Use fixed dates for reproducibility
	day1 := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC) // skip 6-16
	insertSession(t, conn, "s1", "/proj", "main", "")
	insertTurnFull(t, conn, "s1", day1, ptr(day1.Add(time.Minute)), 100, 50, 0, 0, nil)
	insertTurnFull(t, conn, "s1", day2, ptr(day2.Add(time.Minute)), 200, 80, 0, 0, nil)

	since := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)
	result, err := report.Query(conn, report.Options{Since: since})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	// only 2 days with data
	if len(result.Daily) != 2 {
		t.Errorf("Daily len = %d, want 2 (gap day 6-16 must not appear)", len(result.Daily))
	}
	if result.Daily[0].Date != "2026-06-15" {
		t.Errorf("Daily[0].Date = %q, want 2026-06-15", result.Daily[0].Date)
	}
	if result.Daily[1].Date != "2026-06-17" {
		t.Errorf("Daily[1].Date = %q, want 2026-06-17", result.Daily[1].Date)
	}
}

// Task 18.1: ProjectSummary.UserActiveTimeSec computed correctly
func TestProjectSummaryUserActiveTime(t *testing.T) {
	conn := openTestDB(t)

	base := time.Date(2026, 6, 18, 9, 0, 0, 0, time.UTC)
	insertSession(t, conn, "proj-ua1", "/proj/a", "main", "")
	// Two turns 5 minutes apart (< 15m idle threshold) → user active time > 0
	t1 := base
	t2 := base.Add(5 * time.Minute)
	ra1 := t1.Add(30 * time.Second)
	ra2 := t2.Add(30 * time.Second)
	insertTurnFull(t, conn, "proj-ua1", t1, &ra1, 100, 50, 0, 0, nil)
	insertTurnFull(t, conn, "proj-ua1", t2, &ra2, 100, 50, 0, 0, nil)

	result, err := report.Query(conn, report.Options{Since: base.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.ByProject) == 0 {
		t.Fatal("ByProject empty")
	}
	if result.ByProject[0].UserActiveTimeSec <= 0 {
		t.Errorf("ProjectSummary.UserActiveTimeSec = %d, want > 0", result.ByProject[0].UserActiveTimeSec)
	}
}

// Task 18.2: SessionRow.WorkItem correctly returned
func TestSessionRowWorkItem(t *testing.T) {
	conn := openTestDB(t)

	base := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	insertSession(t, conn, "wi-sess", "/proj", "main", "my-feature")
	ra := base.Add(time.Minute)
	insertTurnFull(t, conn, "wi-sess", base, &ra, 100, 50, 0, 0, nil)

	result, err := report.Query(conn, report.Options{Since: base.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.Sessions) == 0 {
		t.Fatal("Sessions empty")
	}
	if result.Sessions[0].WorkItem != "my-feature" {
		t.Errorf("SessionRow.WorkItem = %q, want my-feature", result.Sessions[0].WorkItem)
	}
}

// Task 18.3: SessionRow.UserTimeSec computed correctly
func TestSessionRowUserTimeSec(t *testing.T) {
	conn := openTestDB(t)

	base := time.Date(2026, 6, 18, 11, 0, 0, 0, time.UTC)
	insertSession(t, conn, "user-time-sess", "/proj", "main", "")
	t1 := base
	t2 := base.Add(3 * time.Minute)
	ra1 := t1.Add(30 * time.Second)
	ra2 := t2.Add(30 * time.Second)
	insertTurnFull(t, conn, "user-time-sess", t1, &ra1, 100, 50, 0, 0, nil)
	insertTurnFull(t, conn, "user-time-sess", t2, &ra2, 100, 50, 0, 0, nil)

	result, err := report.Query(conn, report.Options{Since: base.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.Sessions) == 0 {
		t.Fatal("Sessions empty")
	}
	if result.Sessions[0].UserTimeSec <= 0 {
		t.Errorf("SessionRow.UserTimeSec = %d, want > 0", result.Sessions[0].UserTimeSec)
	}
}

// Task 18.4: Result.Groups always non-nil even when ByWorkItem=false
func TestGroupsAlwaysPopulated(t *testing.T) {
	conn := openTestDB(t)

	base := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	insertSession(t, conn, "grp-sess", "/proj", "main", "")
	ra := base.Add(time.Minute)
	insertTurnFull(t, conn, "grp-sess", base, &ra, 100, 50, 0, 0, nil)

	result, err := report.Query(conn, report.Options{Since: base.Add(-time.Hour), ByWorkItem: false})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if result.Groups == nil {
		t.Error("Result.Groups should not be nil even when ByWorkItem=false")
	}
}

// Task 18.5: FormatJSON output contains "groups" array
func TestFormatJSONContainsGroups(t *testing.T) {
	conn := openTestDB(t)

	base := time.Date(2026, 6, 18, 13, 0, 0, 0, time.UTC)
	insertSession(t, conn, "grp-json", "/proj", "main", "")
	ra := base.Add(time.Minute)
	insertTurnFull(t, conn, "grp-json", base, &ra, 100, 50, 0, 0, nil)

	result, err := report.Query(conn, report.Options{Since: base.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	out := report.FormatJSON(result)
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := m["groups"]; !ok {
		t.Error("FormatJSON missing 'groups' key")
	}
}

// Task 6.1: FormatJSON includes cache_creation_tokens, cache_read_tokens, by_project
func TestFormatJSONNewFields(t *testing.T) {
	costVal := 0.05
	r := report.Result{
		SessionsCount:       2,
		InputTokens:         100,
		OutputTokens:        50,
		CacheReadTokens:     30,
		CacheCreationTokens: 10,
		EstimatedCostUSD:    ptr(0.002),
		ByProject: []report.ProjectSummary{
			{Project: "p1", SessionsCount: 2, AgentTimeSec: 120, CostUSD: &costVal},
		},
		Daily: []report.DailyStat{
			{Date: "2026-06-18", Sessions: 2, InputTokens: 100, OutputTokens: 50},
		},
	}
	out := report.FormatJSON(r)
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	for _, key := range []string{"cache_creation_tokens", "cache_read_tokens", "by_project", "daily"} {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON missing key %q", key)
		}
	}
	bp, ok := m["by_project"].([]interface{})
	if !ok || len(bp) != 1 {
		t.Errorf("by_project should be array of 1, got: %v", m["by_project"])
	}
}
