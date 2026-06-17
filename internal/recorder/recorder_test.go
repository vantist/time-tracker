package recorder_test

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/user/tt/internal/db"
	"github.com/user/tt/internal/recorder"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TT_DB_PATH", filepath.Join(dir, "test.db"))
	conn, err := db.Open()
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// Task 3.1: RecordPrompt creates session + turn, prompt_at correct
func TestRecordPromptCreatesTurn(t *testing.T) {
	conn := openTestDB(t)

	before := time.Now().UTC().Truncate(time.Second)
	err := recorder.RecordPrompt(conn, recorder.PromptInput{
		SessionID: "sess-001",
		Project:   "/home/user/myproject",
		Tool:      "claude-code",
		Model:     "claude-sonnet-4-6",
	})
	if err != nil {
		t.Fatalf("RecordPrompt: %v", err)
	}
	after := time.Now().UTC().Add(time.Second)

	// session created
	var sessID string
	conn.QueryRow("SELECT id FROM sessions WHERE id='sess-001'").Scan(&sessID)
	if sessID != "sess-001" {
		t.Error("session not created")
	}

	// turn created with prompt_at in range
	var promptAt string
	conn.QueryRow("SELECT prompt_at FROM turns WHERE session_id='sess-001'").Scan(&promptAt)
	if promptAt == "" {
		t.Fatal("turn not created")
	}
	pt, err := time.Parse(time.RFC3339, promptAt)
	if err != nil {
		t.Fatalf("parse prompt_at: %v", err)
	}
	if pt.Before(before) || pt.After(after) {
		t.Errorf("prompt_at %v out of range [%v, %v]", pt, before, after)
	}
}

// Task 3.3: second RecordPrompt for same session does not recreate session
func TestRecordPromptSecondCallSameSession(t *testing.T) {
	conn := openTestDB(t)

	input := recorder.PromptInput{
		SessionID: "sess-002",
		Project:   "/home/user/proj",
		Tool:      "claude-code",
		Model:     "claude-sonnet-4-6",
	}

	if err := recorder.RecordPrompt(conn, input); err != nil {
		t.Fatal(err)
	}
	if err := recorder.RecordPrompt(conn, input); err != nil {
		t.Fatal(err)
	}

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM sessions WHERE id='sess-002'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 session, got %d", count)
	}

	conn.QueryRow("SELECT COUNT(*) FROM turns WHERE session_id='sess-002'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 turns, got %d", count)
	}
}
