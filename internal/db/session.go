package db

import (
	"database/sql"
	"time"
)

type Session struct {
	ID        string
	Project   string
	Tool      string
	Model     string
	Branch    string
	WorkItem  string
	StartedAt time.Time
	EndedAt   *time.Time
}

// UpsertSession inserts a new session or ignores if it already exists,
// preserving the original started_at.
func UpsertSession(db *sql.DB, s Session) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, tool, model, branch, work_item, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Project, s.Tool, s.Model, s.Branch, s.WorkItem,
		s.StartedAt.UTC().Format(time.RFC3339),
	)
	return err
}
