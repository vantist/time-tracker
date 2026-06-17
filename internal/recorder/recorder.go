package recorder

import (
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/user/tt/internal/db"
)

type PromptInput struct {
	SessionID string
	Project   string
	Tool      string
	Model     string
}

func RecordPrompt(conn *sql.DB, input PromptInput) error {
	now := time.Now().UTC()

	branch := gitBranch(input.Project)

	if err := db.UpsertSession(conn, db.Session{
		ID:        input.SessionID,
		Project:   input.Project,
		Tool:      input.Tool,
		Model:     input.Model,
		Branch:    branch,
		StartedAt: now,
	}); err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}

	_, err := conn.Exec(
		`INSERT INTO turns (session_id, prompt_at) VALUES (?, ?)`,
		input.SessionID, now.Format(time.RFC3339),
	)
	return err
}

func gitBranch(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
