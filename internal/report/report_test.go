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

func insertSessionFull(t *testing.T, conn *sql.DB, id, project, tool, model, branch, workItem string) {
	t.Helper()
	_, err := conn.Exec(
		`INSERT OR IGNORE INTO sessions (id, project, tool, model, branch, work_item, started_at) VALUES (?,?,?,?,?,?,?)`,
		id, project, tool, model, branch, workItem, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insertSessionFull: %v", err)
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
	insertTurnFull(t, conn, "s1", now.Add(-time.Hour), ptr(now.Add(-time.Hour+time.Minute)), 100, 50, 0, 0, ptr(0.01))
	insertTurnFull(t, conn, "s2", now.Add(-time.Hour), ptr(now.Add(-time.Hour+time.Minute)), 200, 80, 0, 0, ptr(0.02))
	insertTurnFull(t, conn, "s3", now.Add(-time.Hour), ptr(now.Add(-time.Hour+time.Minute)), 300, 150, 0, 0, ptr(0.03))

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
	// Verify that project-specific token fields are aggregated properly
	if result.ByProject[0].InputTokens != 300 {
		t.Errorf("ByProject[0].InputTokens = %d, want 300", result.ByProject[0].InputTokens)
	}
	if result.ByProject[0].OutputTokens != 130 {
		t.Errorf("ByProject[0].OutputTokens = %d, want 130", result.ByProject[0].OutputTokens)
	}
	if result.ByProject[1].InputTokens != 300 {
		t.Errorf("ByProject[1].InputTokens = %d, want 300", result.ByProject[1].InputTokens)
	}
	if result.ByProject[1].OutputTokens != 150 {
		t.Errorf("ByProject[1].OutputTokens = %d, want 150", result.ByProject[1].OutputTokens)
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

// TestSessionDuration_SpanConversations: same (process_pid, process_start), multiple
// conversation_id values → all turns grouped under one stable session ID → work time
// spans all turns.
func TestSessionDuration_SpanConversations(t *testing.T) {
	conn := openTestDB(t)

	base := time.Date(2026, 6, 18, 9, 0, 0, 0, time.UTC)

	// Create stable session (first conversation)
	_, err := conn.Exec(`
		INSERT INTO sessions (id, project, branch, work_item, started_at, process_pid, process_start, conversation_id)
		VALUES ('stable-sess', '/proj', 'main', '', ?, 55555, 1700000000, 'conv-a')`,
		base.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}

	// Turn 1: first conversation, t=0
	t1p := base
	t1r := base.Add(30 * time.Second)
	insertTurn(t, conn, "stable-sess", t1p, &t1r, nil)

	// Turn 2: after /clear (conv-b), 10 minutes later — still same stable session
	t2p := base.Add(10 * time.Minute)
	t2r := base.Add(10*time.Minute + 30*time.Second)
	insertTurn(t, conn, "stable-sess", t2p, &t2r, nil)

	result, err := report.Query(conn, report.Options{Since: base.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(result.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(result.Sessions))
	}

	// New semantics: user time = response_at[i-1] → prompt_at[i]
	// interval = [T+30s, T+10m] = 9m30s = 570s (agent processing time excluded)
	sess := result.Sessions[0]
	if sess.UserTimeSec < 560 || sess.UserTimeSec > 580 {
		t.Errorf("UserTimeSec = %d, want ~570 (9m30s: response→next_prompt)", sess.UserTimeSec)
	}
}

// 1.1: 相同 branch 不同 project 應產生兩個不同 GroupResult（複合 key 邏輯）
func TestGroupByWorkItem_SameBranchDifferentProject(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().UTC()

	insertSession(t, conn, "gp-a", "/repo/alpha", "main", "")
	insertSession(t, conn, "gp-b", "/repo/beta", "main", "")

	ra := now.Add(time.Minute)
	insertTurn(t, conn, "gp-a", now.Add(-time.Hour), &ra, nil)
	insertTurn(t, conn, "gp-b", now.Add(-time.Hour), &ra, nil)

	result, err := report.Query(conn, report.Options{Since: now.Add(-2 * time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.Groups) != 2 {
		t.Errorf("Groups len = %d, want 2 (different projects must not merge)", len(result.Groups))
	}
}

// 1.2: GroupResult.Project == path.Base(project)，GroupResult.Label 不含 project 路徑
func TestGroupByWorkItem_ProjectField(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().UTC()

	insertSession(t, conn, "gp-c", "/repo/myproject", "main", "")

	ra := now.Add(time.Minute)
	insertTurn(t, conn, "gp-c", now.Add(-time.Hour), &ra, nil)

	result, err := report.Query(conn, report.Options{Since: now.Add(-2 * time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.Groups) != 1 {
		t.Fatalf("Groups len = %d, want 1", len(result.Groups))
	}
	g := result.Groups[0]
	if g.Project != "myproject" {
		t.Errorf("GroupResult.Project = %q, want %q", g.Project, "myproject")
	}
	if g.Label != "main" {
		t.Errorf("GroupResult.Label = %q, want %q", g.Label, "main")
	}
}

// fix-user-time-semantics 2.1: 兩個重疊 sessions，總計 UserTime 不重複計算
func TestTotalUserTime_OverlappingSessionsMerged(t *testing.T) {
	conn := openTestDB(t)
	base := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)

	// Session A: turn1 response=T+1m, turn2 prompt=T+6m → interval [T+1m, T+6m] = 5m
	insertSession(t, conn, "ov-a", "/proj", "main", "")
	raA1 := base.Add(1 * time.Minute)
	insertTurnFull(t, conn, "ov-a", base, &raA1, 0, 0, 0, 0, nil)
	insertTurnFull(t, conn, "ov-a", base.Add(6*time.Minute), nil, 0, 0, 0, 0, nil)

	// Session B: turn1 response=T+3m, turn2 prompt=T+11m → interval [T+3m, T+11m] = 8m
	// Overlap with A: [T+3m, T+6m] = 3m
	insertSession(t, conn, "ov-b", "/proj", "main", "")
	raB1 := base.Add(3 * time.Minute)
	insertTurnFull(t, conn, "ov-b", base.Add(2*time.Minute), &raB1, 0, 0, 0, 0, nil)
	insertTurnFull(t, conn, "ov-b", base.Add(11*time.Minute), nil, 0, 0, 0, 0, nil)

	// Raw sum = 5m + 8m = 13m. Merged = [T+1m, T+11m] = 10m
	result, err := report.Query(conn, report.Options{Since: base.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	// Must be 10m (600s), not 13m (780s)
	want := int64(600)
	if result.UserActiveTimeSec != want {
		t.Errorf("UserActiveTimeSec = %d, want %d (overlap must not double-count)", result.UserActiveTimeSec, want)
	}
}

// fix-user-time-semantics 2.2: ByProject 同一 project 重疊 sessions merge
func TestByProjectUserTime_OverlappingSessionsMerged(t *testing.T) {
	conn := openTestDB(t)
	base := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)

	// Session A: interval [T+1m, T+6m] = 5m
	insertSession(t, conn, "bp-ov-a", "/shared-proj", "main", "")
	raA := base.Add(1 * time.Minute)
	insertTurnFull(t, conn, "bp-ov-a", base, &raA, 0, 0, 0, 0, nil)
	insertTurnFull(t, conn, "bp-ov-a", base.Add(6*time.Minute), nil, 0, 0, 0, 0, nil)

	// Session B: interval [T+3m, T+11m] = 8m (overlaps with A by 3m)
	insertSession(t, conn, "bp-ov-b", "/shared-proj", "main", "")
	raB := base.Add(3 * time.Minute)
	insertTurnFull(t, conn, "bp-ov-b", base.Add(2*time.Minute), &raB, 0, 0, 0, 0, nil)
	insertTurnFull(t, conn, "bp-ov-b", base.Add(11*time.Minute), nil, 0, 0, 0, 0, nil)

	result, err := report.Query(conn, report.Options{Since: base.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.ByProject) == 0 {
		t.Fatal("ByProject empty")
	}
	// merged = [T+1m, T+11m] = 10m = 600s
	want := int64(600)
	if result.ByProject[0].UserActiveTimeSec != want {
		t.Errorf("ByProject UserActiveTimeSec = %d, want %d", result.ByProject[0].UserActiveTimeSec, want)
	}
}

// fix-user-time-semantics 2.3: ByWorkItem 同一 work item 重疊 sessions merge
func TestByWorkItemUserTime_OverlappingSessionsMerged(t *testing.T) {
	conn := openTestDB(t)
	base := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)

	// Session A: interval [T+1m, T+6m] = 5m
	insertSession(t, conn, "wi-ov-a", "/proj", "main", "feat-x")
	raA := base.Add(1 * time.Minute)
	insertTurnFull(t, conn, "wi-ov-a", base, &raA, 0, 0, 0, 0, nil)
	insertTurnFull(t, conn, "wi-ov-a", base.Add(6*time.Minute), nil, 0, 0, 0, 0, nil)

	// Session B: interval [T+3m, T+11m] = 8m (overlaps with A by 3m)
	insertSession(t, conn, "wi-ov-b", "/proj", "main", "feat-x")
	raB := base.Add(3 * time.Minute)
	insertTurnFull(t, conn, "wi-ov-b", base.Add(2*time.Minute), &raB, 0, 0, 0, 0, nil)
	insertTurnFull(t, conn, "wi-ov-b", base.Add(11*time.Minute), nil, 0, 0, 0, 0, nil)

	result, err := report.Query(conn, report.Options{Since: base.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.Groups) == 0 {
		t.Fatal("Groups empty")
	}
	// find feat-x group
	var g *report.GroupResult
	for i := range result.Groups {
		if result.Groups[i].Label == "feat-x" {
			g = &result.Groups[i]
			break
		}
	}
	if g == nil {
		t.Fatal("feat-x group not found")
	}
	// merged = [T+1m, T+11m] = 10m = 600s
	want := int64(600)
	if g.UserActiveTimeSec != want {
		t.Errorf("GroupResult UserActiveTimeSec = %d, want %d", g.UserActiveTimeSec, want)
	}
}

// 1.3: 相同 work_item 不同 project 應產生兩列，Project 欄各自對應正確值
func TestGroupByWorkItem_SameWorkItemDifferentProject(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().UTC()

	insertSession(t, conn, "gp-d", "/repo/alpha", "main", "feature-x")
	insertSession(t, conn, "gp-e", "/repo/beta", "main", "feature-x")

	ra := now.Add(time.Minute)
	insertTurn(t, conn, "gp-d", now.Add(-time.Hour), &ra, nil)
	insertTurn(t, conn, "gp-e", now.Add(-time.Hour), &ra, nil)

	result, err := report.Query(conn, report.Options{Since: now.Add(-2 * time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.Groups) != 2 {
		t.Fatalf("Groups len = %d, want 2", len(result.Groups))
	}
	projects := map[string]bool{}
	for _, g := range result.Groups {
		if g.Label != "feature-x" {
			t.Errorf("GroupResult.Label = %q, want feature-x", g.Label)
		}
		projects[g.Project] = true
	}
	if !projects["alpha"] || !projects["beta"] {
		t.Errorf("expected both alpha and beta in Projects, got: %v", projects)
	}
}

func TestFormatTextFull(t *testing.T) {
	costVal := 0.05
	costVal2 := 0.0
	r := report.Result{
		SessionsCount:     3,
		AgentTimeSec:      int64(2*3600 + 34*60),
		UserActiveTimeSec: int64(1*3600 + 10*60),
		InputTokens:       10000,
		OutputTokens:      2000,
		EstimatedCostUSD:  ptr(0.042),
		Daily: []report.DailyStat{
			{Date: "2026-06-15", Sessions: 1, InputTokens: 4000, OutputTokens: 800},
			{Date: "2026-06-16", Sessions: 2, InputTokens: 6000, OutputTokens: 1200},
		},
		ByProject: []report.ProjectSummary{
			{Project: "alpha", SessionsCount: 2, AgentTimeSec: 3600, UserActiveTimeSec: 1800, CostUSD: &costVal, InputTokens: 5000, OutputTokens: 1000},
			{Project: "beta", SessionsCount: 1, AgentTimeSec: 1200, UserActiveTimeSec: 600, CostUSD: nil, InputTokens: 5000, OutputTokens: 1000},
		},
		Groups: []report.GroupResult{
			{Label: "feat-a", Project: "alpha", SessionsCount: 1, AgentTimeSec: 2000, UserActiveTimeSec: 1000, EstimatedCostUSD: &costVal},
			{Label: "feat-b", Project: "beta", SessionsCount: 1, AgentTimeSec: 1200, UserActiveTimeSec: 600, EstimatedCostUSD: nil},
		},
		Sessions: []report.SessionRow{
			{
				ID:           "s1",
				Project:      "/path/to/alpha",
				Branch:       "main",
				Model:        "gemini-2.5-flash",
				StartedAt:    "2026-06-19T10:00:00Z",
				WorkItem:     "feat-a",
				Turns:        5,
				AgentTimeSec: 1200,
				UserTimeSec:  600,
				CostUSD:      &costVal,
			},
			{
				ID:           "s2",
				Project:      "/path/to/beta",
				Branch:       "dev",
				Model:        "gemini-2.5-pro",
				StartedAt:    "2026-06-19T09:00:00Z",
				WorkItem:     "feat-b",
				Turns:        2,
				AgentTimeSec: 600,
				UserTimeSec:  300,
				CostUSD:      &costVal2,
			},
		},
	}

	text := report.FormatText(r)

	// Verify Daily timeline headers and content
	for _, want := range []string{
		"─── Daily (Last 7 Days) ───",
		"Date", "Sessions", "Input Tokens", "Output Tokens",
		"2026-06-15", "4,000", "800",
		"2026-06-16", "6,000", "1,200",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("FormatText missing daily detail %q in output:\n%s", want, text)
		}
	}

	// Verify By Project headers and content
	for _, want := range []string{
		"─── By Project ───",
		"Project", "Sessions", "Agent Time", "User Active", "Tokens (I/O)", "Cost",
		"alpha", "2", "1h 0m", "0h 30m", "5,000 / 1,000", "$0.0500",
		"beta", "1", "0h 20m", "0h 10m", "5,000 / 1,000", "N/A",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("FormatText missing project detail %q in output:\n%s", want, text)
		}
	}

	// Verify By Work Item headers and content
	for _, want := range []string{
		"─── By Work Item ───",
		"Work Item", "Project", "Sessions", "Agent Time", "User Active", "Cost",
		"feat-a", "alpha", "1", "0h 33m", "0h 16m", "$0.0500",
		"feat-b", "beta", "1", "0h 20m", "0h 10m", "N/A",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("FormatText missing work item detail %q in output:\n%s", want, text)
		}
	}

	// Verify Sessions log headers and content
	for _, want := range []string{
		"─── Sessions ───",
		"Start Time", "Project", "Branch", "Model", "Turns", "Agent Time", "User Time", "Work Item", "Cost",
		"alpha", "beta", "main", "dev", "gemini-2.5-flash", "gemini-2.5-pro",
		"5", "2", "0h 20m", "0h 10m", "0h 5m", "feat-a", "feat-b", "$0.0500", "$0.0000",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("FormatText missing session detail %q in output:\n%s", want, text)
		}
	}

	// Verify timezone formatting local time check
	t1, _ := time.Parse(time.RFC3339, "2026-06-19T10:00:00Z")
	t1Local := t1.Local().Format("2006-01-02 15:04:05")
	if !strings.Contains(text, t1Local) {
		t.Errorf("FormatText missing session start time local formatted string %q, got:\n%s", t1Local, text)
	}
}

func TestNormalizeAgentName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"claude-code", "Claude Code"},
		{"ClaudeCode", "Claude Code"},
		{"claude", "Claude Code"},
		{"copilot-cli", "Copilot CLI"},
		{"CopilotCli", "Copilot CLI"},
		{"copilot", "Copilot CLI"},
		{"", "unknown"},
		{"   ", "unknown"},
		{"My-Custom-Agent  ", "my-custom-agent"},
		{"  another-one ", "another-one"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			actual := report.NormalizeAgentName(tc.input)
			if actual != tc.expected {
				t.Errorf("NormalizeAgentName(%q) = %q; want %q", tc.input, actual, tc.expected)
			}
		})
	}
}

func TestDataStructures(t *testing.T) {
	// Verify that SessionRow has the Tool field
	row := report.SessionRow{
		Tool: "Claude Code",
	}
	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("Marshal SessionRow: %v", err)
	}
	if !strings.Contains(string(data), `"tool":"Claude Code"`) {
		t.Errorf("expected marshaled SessionRow to contain Tool field, got %s", string(data))
	}

	// Verify that Result has ByAgent field
	res := report.Result{
		ByAgent: []report.AgentSummary{
			{
				Agent:     "Claude Code",
				Sessions:  5,
				AgentTime: "2h 30m",
				UserTime:  "1h 15m",
				Tokens:    "100 / 50",
				Cost:      0.15,
			},
		},
	}
	data2, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("Marshal Result: %v", err)
	}
	if !strings.Contains(string(data2), `"by_agent"`) {
		t.Errorf("expected marshaled Result to contain by_agent, got %s", string(data2))
	}
}

func TestQueryToolField(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().UTC()

	insertSessionFull(t, conn, "s1", "/proj", "claude-code", "gemini-2.5-flash", "main", "feat-1")
	insertTurn(t, conn, "s1", now.Add(-time.Hour), ptr(now.Add(-time.Hour+time.Minute)), nil)

	res, err := report.Query(conn, report.Options{Since: now.Add(-2 * time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(res.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(res.Sessions))
	}

	if res.Sessions[0].Tool != "Claude Code" {
		t.Errorf("expected SessionRow.Tool to be %q, got %q", "Claude Code", res.Sessions[0].Tool)
	}
}


