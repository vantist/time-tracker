package report_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
