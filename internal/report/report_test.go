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
	for _, want := range []string{"Sessions:", "Agent time:", "User active:", "Tokens in:", "Est. cost:"} {
		if !strings.Contains(text, want) {
			t.Errorf("text output missing %q", want)
		}
	}
	if !strings.Contains(text, "2h 34m") {
		t.Errorf("agent time format wrong, got: %s", text)
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
