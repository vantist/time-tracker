package recorder

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/user/tt/internal/db"
	"github.com/user/tt/internal/workitem"
)

type PromptInput struct {
	SessionID      string
	Project        string
	Tool           string
	Model          string
	ProcessPID     int64
	ProcessStart   int64
	TranscriptPath string
}

func RecordPrompt(conn *sql.DB, input PromptInput) error {
	now := time.Now().UTC()

	branch := gitBranch(input.Project)
	wi, _ := workitem.Get(input.Project)

	stableID, err := db.UpsertSession(conn, db.Session{
		ID:             input.SessionID,
		Project:        input.Project,
		Tool:           input.Tool,
		Model:          input.Model,
		Branch:         branch,
		WorkItem:       wi,
		StartedAt:      now,
		ProcessPID:     input.ProcessPID,
		ProcessStart:   input.ProcessStart,
		ConversationID: input.SessionID,
	})
	if err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}

	// Use stable session ID for turns so JOIN sessions s ON s.id = t.session_id works.
	offset := countLines(input.TranscriptPath)
	var transcriptPath interface{}
	if input.TranscriptPath != "" {
		transcriptPath = input.TranscriptPath
	}
	_, err = conn.Exec(
		`INSERT INTO turns (session_id, prompt_at, transcript_path, prompt_line_offset) VALUES (?, ?, ?, ?)`,
		stableID, now.Format(time.RFC3339), transcriptPath, offset,
	)
	return err
}

// countLines counts lines in the file using bufio.Scanner with a 1MB buffer.
// Returns 0 if the file cannot be read.
func countLines(path string) int {
	if path == "" {
		return 0
	}
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	n := 0
	for sc.Scan() {
		n++
	}
	return n
}

func gitBranch(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// resolveStableSessionID finds the sessions.id for a given sessionID which may
// be either a stable session ID or a conversation UUID stored in conversation_id.
func resolveStableSessionID(conn *sql.DB, sessionID string) string {
	// Fast path: direct match on sessions.id
	var id string
	err := conn.QueryRow("SELECT id FROM sessions WHERE id = ?", sessionID).Scan(&id)
	if err == nil {
		return id
	}
	// Fallback: sessionID is a conversation UUID stored in conversation_id.
	// Scan error (not-found or DB error) leaves id as "" — caller handles both.
	_ = conn.QueryRow("SELECT id FROM sessions WHERE conversation_id = ?", sessionID).Scan(&id)
	return id
}
