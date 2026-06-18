package recorder_test

import (
	"database/sql"
	"os"
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

// TestRecordPrompt_StableSession: same (ProcessPID, ProcessStart), different SessionID
// → produces 1 session, 2 turns.
func TestRecordPrompt_StableSession(t *testing.T) {
	conn := openTestDB(t)

	base := recorder.PromptInput{
		SessionID:    "conv-a",
		ProcessPID:   99001,
		ProcessStart: 1700000000,
		Project:      "/home/user/myproject",
		Tool:         "claude-code",
		Model:        "claude-sonnet-4-6",
	}

	if err := recorder.RecordPrompt(conn, base); err != nil {
		t.Fatalf("first RecordPrompt: %v", err)
	}

	// Simulate /clear: new SessionID, same process key
	base.SessionID = "conv-b"
	if err := recorder.RecordPrompt(conn, base); err != nil {
		t.Fatalf("second RecordPrompt: %v", err)
	}

	// Expect exactly 1 session with this process key
	var sessCount int
	conn.QueryRow("SELECT COUNT(*) FROM sessions WHERE process_pid=99001 AND process_start=1700000000").Scan(&sessCount)
	if sessCount != 1 {
		t.Errorf("expected 1 session, got %d", sessCount)
	}

	// Both turns use the stable session ID ("conv-a" — the id of the first-inserted row).
	var turnCount int
	conn.QueryRow("SELECT COUNT(*) FROM turns WHERE session_id='conv-a'").Scan(&turnCount)
	if turnCount != 2 {
		t.Errorf("expected 2 turns under stable session ID, got %d", turnCount)
	}
}

// Task 10.4: RecordPrompt stores transcript_path and prompt_line_offset.
func TestRecordPromptTranscriptOffset(t *testing.T) {
	conn := openTestDB(t)

	// Create a JSONL file with 3 lines
	dir := t.TempDir()
	transcriptPath := filepath.Join(dir, "transcript.jsonl")
	content := `{"type":"user"}` + "\n" + `{"type":"assistant"}` + "\n" + `{"type":"user"}` + "\n"
	if err := os.WriteFile(transcriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	err := recorder.RecordPrompt(conn, recorder.PromptInput{
		SessionID:      "sess-t1",
		Project:        "/proj",
		Tool:           "claude-code",
		Model:          "claude-sonnet-4-6",
		TranscriptPath: transcriptPath,
	})
	if err != nil {
		t.Fatalf("RecordPrompt: %v", err)
	}

	var path string
	var offset int
	conn.QueryRow("SELECT transcript_path, prompt_line_offset FROM turns WHERE session_id='sess-t1'").Scan(&path, &offset)
	if path != transcriptPath {
		t.Errorf("transcript_path = %q, want %q", path, transcriptPath)
	}
	if offset != 3 {
		t.Errorf("prompt_line_offset = %d, want 3", offset)
	}
}

// Task 10.4 (non-existent transcript): prompt_line_offset = 0, no error.
func TestRecordPromptTranscriptMissing(t *testing.T) {
	conn := openTestDB(t)

	err := recorder.RecordPrompt(conn, recorder.PromptInput{
		SessionID:      "sess-t2",
		Project:        "/proj",
		Tool:           "claude-code",
		Model:          "claude-sonnet-4-6",
		TranscriptPath: "/nonexistent/path.jsonl",
	})
	if err != nil {
		t.Fatalf("RecordPrompt: %v", err)
	}

	var offset int
	var path string
	conn.QueryRow("SELECT transcript_path, prompt_line_offset FROM turns WHERE session_id='sess-t2'").Scan(&path, &offset)
	if offset != 0 {
		t.Errorf("prompt_line_offset = %d, want 0 for missing file", offset)
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
