package reconcile

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			project TEXT,
			tool TEXT,
			model TEXT,
			branch TEXT,
			work_item TEXT,
			started_at DATETIME NOT NULL,
			ended_at DATETIME,
			process_pid INTEGER,
			process_start INTEGER,
			conversation_id TEXT
		);
		CREATE TABLE turns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL REFERENCES sessions(id),
			prompt_at DATETIME NOT NULL,
			response_at DATETIME,
			input_tokens INTEGER,
			output_tokens INTEGER,
			cache_read_tokens INTEGER,
			cache_creation_tokens INTEGER,
			estimated_cost_usd REAL,
			transcript_path TEXT,
			prompt_line_offset INTEGER
		);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func insertSession(t *testing.T, db *sql.DB, id string, pid, start int64) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO sessions (id, started_at, process_pid, process_start) VALUES (?, ?, ?, ?)`,
		id, time.Now().UTC().Format(time.RFC3339), pid, start,
	)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}
}

func insertTurn(t *testing.T, db *sql.DB, sessionID, transcriptPath string, offset int, promptAt time.Time) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO turns (session_id, prompt_at, transcript_path, prompt_line_offset) VALUES (?, ?, ?, ?)`,
		sessionID, promptAt.UTC().Format(time.RFC3339Nano), transcriptPath, offset,
	)
	if err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func writeTranscriptLines(t *testing.T, dir string, lines []string) string {
	t.Helper()
	path := filepath.Join(dir, "transcript.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create transcript: %v", err)
	}
	for _, l := range lines {
		f.WriteString(l + "\n")
	}
	f.Close()
	return path
}

// TestReconcile_DanglingMiddleTurn: turn with a successor gets response_at = next_prompt_at - 1ms.
func TestReconcile_DanglingMiddleTurn(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	// Dead session (pid=0 = won't be alive)
	insertSession(t, db, "sess1", 0, 0)

	transcriptLines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50}}}`,
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":200,"output_tokens":80}}}`,
	}
	transcriptPath := writeTranscriptLines(t, dir, transcriptLines)

	prompt1 := time.Now().Add(-10 * time.Minute)
	prompt2 := time.Now().Add(-5 * time.Minute)

	// Turn 1: dangling, offset=0
	turnID1 := insertTurn(t, db, "sess1", transcriptPath, 0, prompt1)
	// Turn 2: also dangling, offset=2 (but it's the "next" turn for turn 1)
	insertTurn(t, db, "sess1", transcriptPath, 2, prompt2)

	reconcile(db)

	var responseAt sql.NullString
	var inputTokens sql.NullInt64
	db.QueryRow("SELECT response_at, input_tokens FROM turns WHERE id=?", turnID1).Scan(&responseAt, &inputTokens)

	if !responseAt.Valid {
		t.Error("response_at should be set after reconcile")
	}
	if !inputTokens.Valid || inputTokens.Int64 != 100 {
		t.Errorf("input_tokens = %v, want 100", inputTokens)
	}

	// response_at should be prompt2 - 1ms
	want := prompt2.Add(-time.Millisecond).UTC().Truncate(time.Millisecond)
	got, _ := time.Parse(time.RFC3339Nano, responseAt.String)
	got = got.UTC().Truncate(time.Millisecond)
	if !got.Equal(want) {
		t.Errorf("response_at = %v, want %v (next_prompt_at - 1ms)", got, want)
	}
}

// TestReconcile_ActiveTurnSkipped: latest turn of alive process must not be updated.
func TestReconcile_ActiveTurnSkipped(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	// Current process as "alive" session
	pid := int64(os.Getpid())
	insertSession(t, db, "sess2", pid, 0)

	transcriptPath := writeTranscriptLines(t, dir, []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50}}}`,
	})

	insertTurn(t, db, "sess2", transcriptPath, 0, time.Now().Add(-1*time.Minute))

	reconcile(db)

	var responseAt sql.NullString
	db.QueryRow("SELECT response_at FROM turns WHERE session_id='sess2'").Scan(&responseAt)
	if responseAt.Valid {
		t.Error("active turn must not be updated (process still alive)")
	}
}

// TestReconcile_Idempotency: turn already having response_at must not be modified.
func TestReconcile_Idempotency(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	insertSession(t, db, "sess3", 0, 0)
	transcriptPath := writeTranscriptLines(t, dir, []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50}}}`,
	})

	// Turn already has response_at (Stop hook already wrote it)
	alreadySet := time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO turns (session_id, prompt_at, response_at, transcript_path, prompt_line_offset, input_tokens) VALUES (?, ?, ?, ?, ?, ?)`,
		"sess3", time.Now().Add(-3*time.Minute).UTC().Format(time.RFC3339), alreadySet, transcriptPath, 0, 999,
	)
	if err != nil {
		t.Fatalf("insert turn: %v", err)
	}

	reconcile(db)

	var inputTokens int
	db.QueryRow("SELECT input_tokens FROM turns WHERE session_id='sess3'").Scan(&inputTokens)
	if inputTokens != 999 {
		t.Errorf("input_tokens = %d, want 999 (must not be overwritten)", inputTokens)
	}
}

// TestReconcile_StopHookWroteResponseAtButNoTokens: when Stop hook set response_at but tokens
// are null (e.g. transcript reset after /clear), reconcile must backfill tokens.
func TestReconcile_StopHookWroteResponseAtButNoTokens(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	insertSession(t, db, "sess4", 0, 0)
	transcriptPath := writeTranscriptLines(t, dir, []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":300,"output_tokens":120}}}`,
	})

	// Stop hook wrote response_at but tokens are null (tokensJSON was empty at fire time)
	alreadySet := time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO turns (session_id, prompt_at, response_at, transcript_path, prompt_line_offset) VALUES (?, ?, ?, ?, ?)`,
		"sess4", time.Now().Add(-2*time.Minute).UTC().Format(time.RFC3339), alreadySet, transcriptPath, 0,
	)
	if err != nil {
		t.Fatalf("insert turn: %v", err)
	}

	reconcile(db)

	var inputTokens sql.NullInt64
	db.QueryRow("SELECT input_tokens FROM turns WHERE session_id='sess4'").Scan(&inputTokens)
	if !inputTokens.Valid || inputTokens.Int64 != 300 {
		t.Errorf("input_tokens = %v, want 300 (reconcile must backfill tokens even when response_at already set)", inputTokens)
	}
}

// TestHasActiveSession: returns true when at least one session process is alive.
func TestHasActiveSession(t *testing.T) {
	db := newTestDB(t)

	// Session with current process (alive)
	insertSession(t, db, "alive1", int64(os.Getpid()), 0)

	if !HasActiveSession(db) {
		t.Error("HasActiveSession = false, want true (current process is alive)")
	}
}

// TestHasActiveSession_NoAlive: returns false when all processes are dead.
func TestHasActiveSession_NoAlive(t *testing.T) {
	db := newTestDB(t)

	// PID 0 is never a valid process
	insertSession(t, db, "dead1", 0, 0)

	if HasActiveSession(db) {
		t.Error("HasActiveSession = true, want false (no alive processes)")
	}
}
