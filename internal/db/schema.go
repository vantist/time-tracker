package db

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open() (*sql.DB, error) {
	path := os.Getenv("TT_DB_PATH")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".tt", "data.db")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id          TEXT PRIMARY KEY,
			project     TEXT,
			tool        TEXT,
			model       TEXT,
			branch      TEXT,
			work_item   TEXT,
			started_at  DATETIME NOT NULL,
			ended_at    DATETIME
		);

		CREATE TABLE IF NOT EXISTS turns (
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
	`); err != nil {
		return err
	}

	if err := addSessionColumns(db); err != nil {
		return err
	}
	return addTurnColumns(db)
}

// addTurnColumns adds transcript_path and prompt_line_offset to turns
// if they don't already exist (SQLite does not support ADD COLUMN IF NOT EXISTS).
func addTurnColumns(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(turns)")
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dflt interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			rows.Close()
			return err
		}
		existing[name] = true
	}
	rows.Close()

	alters := []struct {
		col string
		def string
	}{
		{"transcript_path", "TEXT"},
		{"prompt_line_offset", "INTEGER"},
	}
	for _, a := range alters {
		if existing[a.col] {
			continue
		}
		if _, err := db.Exec("ALTER TABLE turns ADD COLUMN " + a.col + " " + a.def); err != nil {
			return err
		}
	}
	return nil
}

// addSessionColumns adds process_pid, process_start, conversation_id to sessions
// if they don't already exist (SQLite does not support ADD COLUMN IF NOT EXISTS).
func addSessionColumns(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(sessions)")
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dflt interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			rows.Close()
			return err
		}
		existing[name] = true
	}
	rows.Close()

	alters := []struct {
		col string
		def string
	}{
		{"process_pid", "INTEGER"},
		{"process_start", "INTEGER"},
		{"conversation_id", "TEXT"},
	}
	for _, a := range alters {
		if existing[a.col] {
			continue
		}
		if _, err := db.Exec("ALTER TABLE sessions ADD COLUMN " + a.col + " " + a.def); err != nil {
			return err
		}
	}
	return nil
}
