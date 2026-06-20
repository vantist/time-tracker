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
			prompt_line_offset INTEGER,
			model TEXT,
			cache_creation_5m_tokens INTEGER,
			cache_creation_1h_tokens INTEGER,
			subagent_tokens_settled BOOLEAN DEFAULT 0
		);
		CREATE TABLE turn_model_usages (
			id                          INTEGER PRIMARY KEY AUTOINCREMENT,
			turn_id                     INTEGER NOT NULL REFERENCES turns(id) ON DELETE CASCADE,
			model                       TEXT NOT NULL,
			is_subagent                 BOOLEAN NOT NULL DEFAULT 0,
			input_tokens                INTEGER NOT NULL DEFAULT 0,
			output_tokens               INTEGER NOT NULL DEFAULT 0,
			cache_read_tokens           INTEGER NOT NULL DEFAULT 0,
			cache_creation_tokens       INTEGER NOT NULL DEFAULT 0,
			cache_creation_5m_tokens    INTEGER NOT NULL DEFAULT 0,
			cache_creation_1h_tokens    INTEGER NOT NULL DEFAULT 0,
			estimated_cost_usd          REAL NOT NULL DEFAULT 0.0,
			UNIQUE(turn_id, model, is_subagent)
		);
		CREATE INDEX idx_turn_model_usages_turn_id ON turn_model_usages(turn_id);
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

// TestReconcile_Idempotency: turn with subagent_tokens_settled=1 must not be reprocessed.
func TestReconcile_Idempotency(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	insertSession(t, db, "sess3", 0, 0)
	transcriptPath := writeTranscriptLines(t, dir, []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50}}}`,
	})

	// Turn already fully settled (subagent_tokens_settled=1).
	alreadySet := time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO turns (session_id, prompt_at, response_at, transcript_path, prompt_line_offset, input_tokens, subagent_tokens_settled) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"sess3", time.Now().Add(-3*time.Minute).UTC().Format(time.RFC3339), alreadySet, transcriptPath, 0, 999, 1,
	)
	if err != nil {
		t.Fatalf("insert turn: %v", err)
	}

	reconcile(db)

	var inputTokens int
	db.QueryRow("SELECT input_tokens FROM turns WHERE session_id='sess3'").Scan(&inputTokens)
	if inputTokens != 999 {
		t.Errorf("input_tokens = %d, want 999 (settled turn must not be overwritten)", inputTokens)
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

// TestReconcile_SubagentTokensSettled: Stop hook wrote response_at but subagent_tokens_settled=0
// → reconcile picks it up and sets subagent_tokens_settled=1.
// After second reconcile, the turn is skipped (settled=1).
func TestReconcile_SubagentTokensSettled(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	insertSession(t, db, "sess5", 0, 0)
	transcriptPath := writeTranscriptLines(t, dir, []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":50,"output_tokens":20}}}`,
	})

	// Stop hook wrote response_at + tokens but subagent_tokens_settled=0
	alreadySet := time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO turns (session_id, prompt_at, response_at, input_tokens, transcript_path, prompt_line_offset, subagent_tokens_settled) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"sess5", time.Now().Add(-2*time.Minute).UTC().Format(time.RFC3339), alreadySet, 999, transcriptPath, 0, 0,
	)
	if err != nil {
		t.Fatalf("insert turn: %v", err)
	}

	reconcile(db)

	var settled int
	var inputTokens int
	db.QueryRow("SELECT subagent_tokens_settled, input_tokens FROM turns WHERE session_id='sess5'").Scan(&settled, &inputTokens)
	if settled != 1 {
		t.Errorf("subagent_tokens_settled = %d, want 1 after reconcile", settled)
	}
	// input_tokens should be re-computed from transcript (50), not left as 999
	if inputTokens != 50 {
		t.Errorf("input_tokens = %d, want 50 (re-computed by reconcile)", inputTokens)
	}

	// Second reconcile: settled=1, must be skipped entirely
	// Overwrite input_tokens to verify it stays
	db.Exec(`UPDATE turns SET input_tokens=888 WHERE session_id='sess5'`)
	reconcile(db)
	db.QueryRow("SELECT input_tokens FROM turns WHERE session_id='sess5'").Scan(&inputTokens)
	if inputTokens != 888 {
		t.Errorf("input_tokens = %d after second reconcile, want 888 (must not re-process settled turn)", inputTokens)
	}
}

// TestReconcile_Cache5m1h: transcript with 5m/1h cache → DB fields filled correctly.
func TestReconcile_Cache5m1h(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	insertSession(t, db, "sess6", 0, 0)
	transcriptPath := writeTranscriptLines(t, dir, []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":40,"cache_creation":{"ephemeral_5m_input_tokens":200,"ephemeral_1h_input_tokens":300}}}}`,
	})

	insertTurn(t, db, "sess6", transcriptPath, 0, time.Now().Add(-2*time.Minute))
	reconcile(db)

	var c5m, c1h sql.NullInt64
	db.QueryRow("SELECT cache_creation_5m_tokens, cache_creation_1h_tokens FROM turns WHERE session_id='sess6'").Scan(&c5m, &c1h)
	if !c5m.Valid || c5m.Int64 != 200 {
		t.Errorf("cache_creation_5m_tokens = %v, want 200", c5m)
	}
	if !c1h.Valid || c1h.Int64 != 300 {
		t.Errorf("cache_creation_1h_tokens = %v, want 300", c1h)
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

func TestReconcile_TurnModelUsages(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	insertSession(t, db, "sess_u2", 0, 0)

	// Create a subagent directory
	transcriptPath := writeTranscriptLines(t, dir, []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":10,"output_tokens":5},"content":[{"type":"tool_use","id":"toolu_rec1","name":"Agent"}]}}`,
	})

	subDir := filepath.Join(dir, "transcript", "subagents")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "agent-rec.meta.json"),
		[]byte(`{"toolUseId":"toolu_rec1"}`), 0644)
	os.WriteFile(filepath.Join(subDir, "agent-rec.jsonl"),
		[]byte(`{"type":"assistant","isSidechain":true,"message":{"model":"claude-haiku-4-5","usage":{"input_tokens":100,"output_tokens":50}}}`+"\n"), 0644)

	// Turn has response_at but subagent_tokens_settled = 0
	promptAt := time.Now().Add(-10 * time.Minute)
	turnID := insertTurn(t, db, "sess_u2", transcriptPath, 0, promptAt)

	// Insert an old detail in turn_model_usages to verify it gets replaced (e.g. model gpt-4)
	_, err := db.Exec(`INSERT INTO turn_model_usages (turn_id, model, is_subagent, input_tokens) VALUES (?, 'gpt-4', 0, 999)`, turnID)
	if err != nil {
		t.Fatalf("failed to insert old model usage: %v", err)
	}

	reconcile(db)

	// Check turn_model_usages
	rows, err := db.Query("SELECT model, is_subagent, input_tokens, output_tokens, estimated_cost_usd FROM turn_model_usages WHERE turn_id=? ORDER BY is_subagent ASC", turnID)
	if err != nil {
		t.Fatalf("query turn_model_usages: %v", err)
	}
	defer rows.Close()

	type usage struct {
		model      string
		isSubagent bool
		input      int
		output     int
		cost       float64
	}
	var results []usage
	for rows.Next() {
		var u usage
		rows.Scan(&u.model, &u.isSubagent, &u.input, &u.output, &u.cost)
		results = append(results, u)
	}

	// Expect 2 usages:
	// 1. main agent: claude-sonnet-4-6 (is_subagent=false, input=10, output=5)
	// 2. subagent: claude-haiku-4-5 (is_subagent=true, input=100, output=50)
	if len(results) != 2 {
		t.Fatalf("expected 2 usages, got %d: %+v", len(results), results)
	}

	var mainU, subU usage
	for _, r := range results {
		if !r.isSubagent {
			mainU = r
		} else {
			subU = r
		}
	}

	if mainU.model != "claude-sonnet-4-6" || mainU.input != 10 || mainU.output != 5 {
		t.Errorf("expected main usage claude-sonnet-4-6 (10, 5), got %+v", mainU)
	}
	if subU.model != "claude-haiku-4-5" || subU.input != 100 || subU.output != 50 {
		t.Errorf("expected subagent usage claude-haiku-4-5 (100, 50), got %+v", subU)
	}

	// The old gpt-4 usage must be deleted/replaced
	for _, r := range results {
		if r.model == "gpt-4" {
			t.Error("old gpt-4 usage was not deleted")
		}
	}

	// Verify turns table is updated with pre-aggregated sum
	var turnInput, turnOutput int
	var turnCost float64
	var settled bool
	db.QueryRow("SELECT input_tokens, output_tokens, estimated_cost_usd, subagent_tokens_settled FROM turns WHERE id=?", turnID).Scan(&turnInput, &turnOutput, &turnCost, &settled)

	if turnInput != 110 || turnOutput != 55 {
		t.Errorf("turns pre-aggregated totals mismatch: got input=%d, output=%d", turnInput, turnOutput)
	}
	if !settled {
		t.Error("turns.subagent_tokens_settled should be true/1")
	}

	// Cost:
	// Sonnet: 10/1e6 * 3.00 + 5/1e6 * 15.00 = 0.00003 + 0.000075 = 0.000105
	// Haiku: 100/1e6 * 1.00 + 50/1e6 * 5.00 = 0.000100 + 0.000250 = 0.000350
	// Total cost = 0.000455
	if turnCost < 0.000454 || turnCost > 0.000456 {
		t.Errorf("turns cost pre-aggregated sum wrong: %f, want ~0.000455", turnCost)
	}
}

func TestReconcile_AntigravityZeroTokens(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Create settings.json
	cliConfigDir := filepath.Join(dir, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(cliConfigDir, 0o700); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(cliConfigDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"model": "Gemini 3.5 Flash"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create session with tool: antigravity
	_, err := db.Exec(`
		INSERT INTO sessions (id, started_at, process_pid, process_start, tool) 
		VALUES (?, ?, ?, ?, ?)`,
		"sess-anti-rec", time.Now().UTC().Format(time.RFC3339), 8888, 1700000000, "antigravity",
	)
	if err != nil {
		t.Fatal(err)
	}

	// Write empty transcript (typical for antigravity turns before stop/shutdown)
	path := filepath.Join(dir, "transcript.jsonl")
	if err := os.WriteFile(path, []byte(`{"type":"PLANNER_RESPONSE"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Insert dangling turn (response_at is NULL, tokens are NULL)
	turnID := insertTurn(t, db, "sess-anti-rec", path, 1, time.Now().Add(-10*time.Second))

	// Reconcile
	reconcile(db)

	// Verify turn is reconciled
	var responseAt sql.NullString
	var model sql.NullString
	var inputTokens, outputTokens sql.NullInt64
	var settled bool
	err = db.QueryRow("SELECT response_at, model, input_tokens, output_tokens, subagent_tokens_settled FROM turns WHERE id=?", turnID).Scan(
		&responseAt, &model, &inputTokens, &outputTokens, &settled,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !responseAt.Valid || responseAt.String == "" {
		t.Error("expected response_at to be set by reconcile for antigravity")
	}
	if !model.Valid || model.String != "gemini-3.5-flash" {
		t.Errorf("expected model to be gemini-3.5-flash, got %q", model.String)
	}
	if !inputTokens.Valid || inputTokens.Int64 != 0 {
		t.Errorf("expected input_tokens = 0, got %v", inputTokens)
	}
	if !settled {
		t.Error("expected subagent_tokens_settled to be true")
	}
}

// TestReconcile_LogPathSwitch: when nextTranscriptPath != transcriptPath,
// reconcile sets to = -1 (reads to EOF of old transcript).
func TestReconcile_LogPathSwitch(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	insertSession(t, db, "sess-switch", 0, 0)

	// Old transcript with 4 lines
	oldPath := filepath.Join(dir, "transcript_old.jsonl")
	oldLines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":10,"output_tokens":5}}}`,
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":20,"output_tokens":10}}}`,
	}
	f, _ := os.Create(oldPath)
	for _, l := range oldLines {
		f.WriteString(l + "\n")
	}
	f.Close()

	newPath := filepath.Join(dir, "transcript_new.jsonl")

	// Insert dangling turn with nextTranscriptPath different from oldPath
	// next_offset is 2, but because nextTranscriptPath != transcript_path,
	// it should read to EOF (-1) and capture both assistant runs (10 + 20 = 30 input tokens).
	promptAt := time.Now().Add(-10 * time.Minute)
	res, err := db.Exec(
		`INSERT INTO turns (session_id, prompt_at, transcript_path, prompt_line_offset) VALUES (?, ?, ?, ?)`,
		"sess-switch", promptAt.UTC().Format(time.RFC3339Nano), oldPath, 0,
	)
	if err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	turnID, _ := res.LastInsertId()

	// We need another turn in the same session to trigger the nextOffset / nextTranscriptPath selection in query.
	// The query in reconcile.go selects the next turn's offset and path.
	_, err = db.Exec(
		`INSERT INTO turns (session_id, prompt_at, transcript_path, prompt_line_offset) VALUES (?, ?, ?, ?)`,
		"sess-switch", time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339Nano), newPath, 2,
	)
	if err != nil {
		t.Fatal(err)
	}

	reconcile(db)

	var inputTokens sql.NullInt64
	err = db.QueryRow("SELECT input_tokens FROM turns WHERE id=?", turnID).Scan(&inputTokens)
	if err != nil {
		t.Fatal(err)
	}

	if !inputTokens.Valid {
		t.Fatal("expected input_tokens to be valid")
	}
	if inputTokens.Int64 != 30 {
		t.Errorf("expected 30 input tokens, got %d", inputTokens.Int64)
	}
}

func TestReconcile_SessionModelBackfill(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	// Create transcript with assistant response having a model name
	lines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-3-5-sonnet","usage":{"input_tokens":10,"output_tokens":5}}}`,
	}
	path := writeTranscriptLines(t, dir, lines)

	t.Run("backfill model if empty or NULL", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO sessions (id, started_at, process_pid, process_start, tool, model) 
			VALUES (?, ?, ?, ?, ?, NULL)`,
			"sess-backfill", time.Now().UTC().Format(time.RFC3339), 0, 0, "claude-code",
		)
		if err != nil {
			t.Fatal(err)
		}

		turnID := insertTurn(t, db, "sess-backfill", path, 0, time.Now().Add(-10*time.Second))

		reconcile(db)

		// Verify turn model is updated
		var turnModel sql.NullString
		err = db.QueryRow("SELECT model FROM turns WHERE id=?", turnID).Scan(&turnModel)
		if err != nil {
			t.Fatal(err)
		}
		if !turnModel.Valid || turnModel.String != "claude-3-5-sonnet" {
			t.Errorf("expected turn model = claude-3-5-sonnet, got %q", turnModel.String)
		}

		// Verify session model is backfilled
		var sessModel sql.NullString
		err = db.QueryRow("SELECT model FROM sessions WHERE id=?", "sess-backfill").Scan(&sessModel)
		if err != nil {
			t.Fatal(err)
		}
		if !sessModel.Valid || sessModel.String != "claude-3-5-sonnet" {
			t.Errorf("expected session model = claude-3-5-sonnet, got %q", sessModel.String)
		}
	})

	t.Run("do not overwrite existing preset model", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO sessions (id, started_at, process_pid, process_start, tool, model) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			"sess-preset", time.Now().UTC().Format(time.RFC3339), 0, 0, "claude-code", "preset-model",
		)
		if err != nil {
			t.Fatal(err)
		}

		turnID2 := insertTurn(t, db, "sess-preset", path, 0, time.Now().Add(-10*time.Second))

		reconcile(db)

		// Verify turn model updated
		var turnModel sql.NullString
		err = db.QueryRow("SELECT model FROM turns WHERE id=?", turnID2).Scan(&turnModel)
		if err != nil {
			t.Fatal(err)
		}
		if !turnModel.Valid || turnModel.String != "claude-3-5-sonnet" {
			t.Errorf("expected turn model = claude-3-5-sonnet, got %q", turnModel.String)
		}

		// Verify session model is still the preset one
		var sessModel sql.NullString
		err = db.QueryRow("SELECT model FROM sessions WHERE id=?", "sess-preset").Scan(&sessModel)
		if err != nil {
			t.Fatal(err)
		}
		if !sessModel.Valid || sessModel.String != "preset-model" {
			t.Errorf("expected session model = preset-model, got %q", sessModel.String)
		}
	})
}
