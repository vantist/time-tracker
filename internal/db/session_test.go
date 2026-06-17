package db_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/user/tt/internal/db"
)

func TestUpsertSessionPreservesStartedAt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TT_DB_PATH", filepath.Join(dir, "test.db"))

	conn, err := db.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	original := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	s := db.Session{
		ID:        "sess-abc",
		Project:   "/home/user/proj",
		Tool:      "claude-code",
		StartedAt: original,
	}

	if err := db.UpsertSession(conn, s); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Second upsert with later time — started_at must stay original
	s.StartedAt = original.Add(time.Hour)
	if err := db.UpsertSession(conn, s); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	var got string
	conn.QueryRow("SELECT started_at FROM sessions WHERE id=?", s.ID).Scan(&got)
	if got != original.UTC().Format(time.RFC3339) {
		t.Errorf("started_at changed: got %q want %q", got, original.UTC().Format(time.RFC3339))
	}
}
