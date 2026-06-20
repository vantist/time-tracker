package report_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/user/tt/internal/report"
)

// Task 10.1: HandleDashboard returns 200 text/html with <html>
func TestHandleDashboard(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	report.HandleDashboard(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<html") {
		t.Errorf("body missing <html>:\n%s", body[:min(len(body), 200)])
	}
	for _, want := range []string{
		"<h2>By Agent</h2>",
		"id=\"tbl-agent\"",
		"esc(s.tool)",
		"<th>Agent</th>",
		"<h2>By Model & Role</h2>",
		"id=\"tbl-model-usages\"",
		"ratio-bar-item",
		"model_usages",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard HTML missing element: %q", want)
		}
	}
}

// Task 10.2: HandleAPIReport returns 200 application/json with by_project and daily
func TestHandleAPIReport(t *testing.T) {
	conn := openTestDB(t)

	handler := report.HandleAPIReport(conn, report.Options{})
	req := httptest.NewRequest(http.MethodGet, "/api/report", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var m map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&m); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	for _, key := range []string{"by_project", "by_agent", "daily", "model_usages"} {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON missing key %q", key)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestHandleAPIReportNewTokenFields(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().UTC()

	// Insert session with all fields
	insertSessionFull(t, conn, "s1", "/proj/alpha", "claude-code", "gemini-2.5-flash", "main", "feature-x", now)

	// Insert turn with ID 101
	promptAt := now.Add(-10*time.Minute)
	responseAt := now.Add(-9*time.Minute)
	_, err := conn.Exec(
		`INSERT INTO turns (id, session_id, prompt_at, response_at, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd)
		 VALUES (101, 's1', ?, ?, 100, 50, 30, 20, 0.01)`,
		promptAt.Format(time.RFC3339),
		responseAt.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("failed to insert turn: %v", err)
	}

	// Insert model usage with turn_id 101
	_, err = conn.Exec(`
		INSERT INTO turn_model_usages (
			turn_id, model, is_subagent, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd
		) VALUES (101, 'gemini-2.5-flash', 0, 100, 50, 30, 20, 0.01)
	`)
	if err != nil {
		t.Fatalf("failed to insert model usage: %v", err)
	}

	handler := report.HandleAPIReport(conn, report.Options{})
	req := httptest.NewRequest(http.MethodGet, "/api/report?since=2026-06-01", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var m map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&m); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}

	// Helper to check token fields
	checkTokens := func(name string, obj map[string]interface{}, wantIn, wantOut, wantCr, wantCc float64) {
		t.Helper()
		if val, _ := obj["input_tokens"].(float64); val != wantIn {
			t.Errorf("%s input_tokens = %v, want %v", name, obj["input_tokens"], wantIn)
		}
		if val, _ := obj["output_tokens"].(float64); val != wantOut {
			t.Errorf("%s output_tokens = %v, want %v", name, obj["output_tokens"], wantOut)
		}
		if val, _ := obj["cache_read_tokens"].(float64); val != wantCr {
			t.Errorf("%s cache_read_tokens = %v, want %v", name, obj["cache_read_tokens"], wantCr)
		}
		if val, _ := obj["cache_creation_tokens"].(float64); val != wantCc {
			t.Errorf("%s cache_creation_tokens = %v, want %v", name, obj["cache_creation_tokens"], wantCc)
		}
	}

	// 1. Root level
	checkTokens("root", m, 100, 50, 30, 20)

	// 2. by_project
	bpList, _ := m["by_project"].([]interface{})
	if len(bpList) != 1 {
		t.Fatalf("by_project len = %d, want 1", len(bpList))
	}
	checkTokens("by_project[0]", bpList[0].(map[string]interface{}), 100, 50, 30, 20)

	// 3. by_agent
	baList, _ := m["by_agent"].([]interface{})
	if len(baList) != 1 {
		t.Fatalf("by_agent len = %d, want 1", len(baList))
	}
	checkTokens("by_agent[0]", baList[0].(map[string]interface{}), 100, 50, 30, 20)

	// 4. daily
	dList, _ := m["daily"].([]interface{})
	if len(dList) != 1 {
		t.Fatalf("daily len = %d, want 1", len(dList))
	}
	checkTokens("daily[0]", dList[0].(map[string]interface{}), 100, 50, 30, 20)

	// 5. model_usages
	muList, _ := m["model_usages"].([]interface{})
	if len(muList) != 1 {
		t.Fatalf("model_usages len = %d, want 1", len(muList))
	}
	checkTokens("model_usages[0]", muList[0].(map[string]interface{}), 100, 50, 30, 20)

	// 6. sessions
	sList, _ := m["sessions"].([]interface{})
	if len(sList) != 1 {
		t.Fatalf("sessions len = %d, want 1", len(sList))
	}
	checkTokens("sessions[0]", sList[0].(map[string]interface{}), 100, 50, 30, 20)

	// 7. groups
	gList, _ := m["groups"].([]interface{})
	if len(gList) != 1 {
		t.Fatalf("groups len = %d, want 1", len(gList))
	}
	checkTokens("groups[0]", gList[0].(map[string]interface{}), 100, 50, 30, 20)
}

