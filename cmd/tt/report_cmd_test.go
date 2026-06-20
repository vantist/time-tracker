package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/tt/internal/db"
)

func TestReportCmd_OutputFile(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	t.Setenv("TT_DB_PATH", dbPath)

	// Initialize DB
	conn, err := db.Open()
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer conn.Close()

	// Insert test data
	_, err = conn.Exec(`
		INSERT INTO sessions (id, project, branch, started_at)
		VALUES ('sess-output-test', '/home/user/myproject', 'main', '2026-06-20T10:00:00Z')
	`)
	if err != nil {
		t.Fatalf("failed to insert session: %v", err)
	}

	_, err = conn.Exec(`
		INSERT INTO turns (session_id, prompt_at, response_at, input_tokens, output_tokens)
		VALUES ('sess-output-test', '2026-06-20T10:01:00Z', '2026-06-20T10:02:00Z', 1000, 500)
	`)
	if err != nil {
		t.Fatalf("failed to insert turn: %v", err)
	}

	outputPath := filepath.Join(tempDir, "report.txt")

	// Set args to run report with output flag
	// Use since filter that covers the test data
	rootCmd.SetArgs([]string{"report", "--output", outputPath, "--since", "2026-06-01"})

	// Run command
	cmdErr := rootCmd.Execute()
	if cmdErr != nil {
		t.Fatalf("reportCmd Execute failed: %v", cmdErr)
	}

	// Verify file exists
	info, statErr := os.Stat(outputPath)
	if statErr != nil {
		t.Fatalf("expected output file to exist, but stat failed: %v", statErr)
	}

	// Verify permissions (0600)
	// On Windows, file permissions might not be strictly 0600 but we are on macOS/Linux.
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected file permissions 0600, got %O", perm)
	}

	// Verify file content
	content, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("failed to read output file: %v", readErr)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Sessions:    1") {
		t.Errorf("output file missing session count, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "myproject") {
		t.Errorf("output file missing project name, got:\n%s", contentStr)
	}
}
