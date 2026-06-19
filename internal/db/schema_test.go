package db_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/user/tt/internal/db"
	_ "modernc.org/sqlite"
)

func TestOpenCreatesTablesOnFirstRun(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	t.Setenv("TT_DB_PATH", dbPath)

	conn, err := db.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	for _, table := range []string{"sessions", "turns"} {
		var name string
		err := conn.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestOpenDefaultPathWhenEnvUnset(t *testing.T) {
	os.Unsetenv("TT_DB_PATH")
	// Just verify Open doesn't panic or error when using default path.
	// We don't want to pollute ~/.tt in tests, so use a temp home.
	home := t.TempDir()
	t.Setenv("HOME", home)

	conn, err := db.Open()
	if err != nil {
		t.Fatalf("Open with default path: %v", err)
	}
	conn.Close()

	// Confirm file exists under temp home
	if _, err := os.Stat(filepath.Join(home, ".tt", "data.db")); err != nil {
		t.Errorf("expected data.db at ~/.tt/data.db: %v", err)
	}
}

// TestAddTurnColumns_NewFields verifies that model, cache_creation_5m_tokens,
// cache_creation_1h_tokens, subagent_tokens_settled are added on db.Open().
func TestAddTurnColumns_NewFields(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	t.Setenv("TT_DB_PATH", dbPath)

	conn, err := db.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	for _, col := range []string{"model", "cache_creation_5m_tokens", "cache_creation_1h_tokens", "subagent_tokens_settled"} {
		var val interface{}
		err := conn.QueryRow("SELECT " + col + " FROM turns LIMIT 1").Scan(&val)
		// ErrNoRows is fine — table exists with the column; only error means column missing.
		if err != nil && err.Error() != "sql: no rows in result set" {
			t.Errorf("column %q not found: %v", col, err)
		}
	}
}

// TestAddTurnColumns_Idempotent verifies no error when columns already exist.
func TestAddTurnColumns_Idempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	t.Setenv("TT_DB_PATH", dbPath)

	conn, err := db.Open()
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	conn.Close()

	// Second open should not error even though columns already exist.
	conn2, err := db.Open()
	if err != nil {
		t.Fatalf("second Open (idempotent): %v", err)
	}
	conn2.Close()
}

// TestMigrate_NewColumns verifies that process_pid, process_start, conversation_id
// columns exist after migration and that existing rows have NULL values.
func TestMigrate_NewColumns(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	t.Setenv("TT_DB_PATH", dbPath)

	conn, err := db.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	// Insert a row using only original columns (simulate old data)
	_, err = conn.Exec(`INSERT INTO sessions (id, started_at) VALUES ('old-sess', '2026-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert old row: %v", err)
	}

	// Confirm new columns exist by querying them
	for _, col := range []string{"process_pid", "process_start", "conversation_id"} {
		var val interface{}
		err := conn.QueryRow("SELECT "+col+" FROM sessions WHERE id='old-sess'").Scan(&val)
		if err != nil {
			t.Errorf("column %q missing or not queryable: %v", col, err)
			continue
		}
		if val != nil {
			t.Errorf("column %q: expected NULL for old row, got %v", col, val)
		}
	}
}

func TestTurnModelUsagesMigrationAndBackfill(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// 1. Manually create old db schema and insert data
	oldConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open manual connection: %v", err)
	}
	_, err = oldConn.Exec(`
		CREATE TABLE sessions (
			id          TEXT PRIMARY KEY,
			project     TEXT,
			tool        TEXT,
			model       TEXT,
			branch      TEXT,
			work_item   TEXT,
			started_at  DATETIME NOT NULL,
			ended_at    DATETIME
		);
		CREATE TABLE turns (
			id                  INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id          TEXT NOT NULL REFERENCES sessions(id),
			prompt_at           DATETIME NOT NULL,
			response_at         DATETIME,
			input_tokens        INTEGER,
			output_tokens       INTEGER,
			cache_read_tokens   INTEGER,
			cache_creation_tokens INTEGER,
			estimated_cost_usd  REAL
		);
	`)
	if err != nil {
		oldConn.Close()
		t.Fatalf("failed to create old schema: %v", err)
	}

	// Insert mock old sessions and turns
	_, err = oldConn.Exec(`
		INSERT INTO sessions (id, model, started_at) VALUES ('s1', 'gpt-4', '2026-06-19T00:00:00Z');
		INSERT INTO sessions (id, model, started_at) VALUES ('s2', NULL, '2026-06-19T00:00:00Z');

		-- t1: turn with tokens, but model is NULL -> should backfill using session s1's model ('gpt-4')
		INSERT INTO turns (id, session_id, prompt_at, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd)
		VALUES (1, 's1', '2026-06-19T00:01:00Z', 100, 50, 10, 5, 0.001);

		-- t2: turn with model set -> should backfill using turn's model ('claude-3-5')
		-- We need to add the turn model column manually since it's part of the later ALTER TABLE,
		-- but we can do it via ALTER TABLE to simulate an upgraded but unbackfilled DB.
	`)
	if err != nil {
		oldConn.Close()
		t.Fatalf("failed to insert initial old data: %v", err)
	}

	_, err = oldConn.Exec("ALTER TABLE turns ADD COLUMN model TEXT")
	if err != nil {
		oldConn.Close()
		t.Fatalf("failed to alter table turns: %v", err)
	}

	_, err = oldConn.Exec(`
		INSERT INTO turns (id, session_id, prompt_at, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd, model)
		VALUES (2, 's1', '2026-06-19T00:02:00Z', 200, 80, 20, 10, 0.002, 'claude-3-5');

		-- t3: both turn model and session model are NULL -> should backfill using 'unknown'
		INSERT INTO turns (id, session_id, prompt_at, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd, model)
		VALUES (3, 's2', '2026-06-19T00:03:00Z', 300, 150, 30, 15, 0.003, NULL);
	`)
	if err != nil {
		oldConn.Close()
		t.Fatalf("failed to insert data with model: %v", err)
	}
	oldConn.Close()

	// 2. Trigger migration by calling db.Open()
	t.Setenv("TT_DB_PATH", dbPath)
	conn, err := db.Open()
	if err != nil {
		t.Fatalf("failed to run migrations via db.Open(): %v", err)
	}
	defer conn.Close()

	// 3. Verify turn_model_usages table exists and is backfilled
	rows, err := conn.Query("SELECT turn_id, model, is_subagent, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd FROM turn_model_usages ORDER BY turn_id ASC")
	if err != nil {
		t.Fatalf("failed to query turn_model_usages: %v", err)
	}
	defer rows.Close()

	type usage struct {
		turnID              int
		model               string
		isSubagent          bool
		inputTokens         int
		outputTokens        int
		cacheReadTokens     int
		cacheCreationTokens int
		cost                float64
	}

	var results []usage
	for rows.Next() {
		var u usage
		if err := rows.Scan(&u.turnID, &u.model, &u.isSubagent, &u.inputTokens, &u.outputTokens, &u.cacheReadTokens, &u.cacheCreationTokens, &u.cost); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		results = append(results, u)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 backfilled rows, got %d", len(results))
	}

	expected := []usage{
		{1, "gpt-4", false, 100, 50, 10, 5, 0.001},
		{2, "claude-3-5", false, 200, 80, 20, 10, 0.002},
		{3, "unknown", false, 300, 150, 30, 15, 0.003},
	}

	for i, exp := range expected {
		if i >= len(results) {
			break
		}
		got := results[i]
		if got.turnID != exp.turnID || got.model != exp.model || got.isSubagent != exp.isSubagent ||
			got.inputTokens != exp.inputTokens || got.outputTokens != exp.outputTokens ||
			got.cacheReadTokens != exp.cacheReadTokens || got.cacheCreationTokens != exp.cacheCreationTokens ||
			got.cost != exp.cost {
			t.Errorf("row %d: expected %+v, got %+v", i, exp, got)
		}
	}
}

