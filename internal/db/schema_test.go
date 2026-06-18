package db_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/tt/internal/db"
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
