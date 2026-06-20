package transcript_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/tt/internal/transcript"
)

func TestParseCopilotLog_Normal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	lines := []string{
		`{"type":"session.start","data":{}}`,
		`{"type":"session.shutdown","data":{"modelMetrics":{"gpt-5.4":{"usage":{"inputTokens":1000,"outputTokens":200,"cacheReadTokens":500,"cacheWriteTokens":100,"reasoningTokens":50}}}}}`,
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	for _, line := range lines {
		f.WriteString(line + "\n")
	}
	f.Close()

	res, err := transcript.ParseCopilotLog(path)
	if err != nil {
		t.Fatalf("ParseCopilotLog failed: %v", err)
	}

	if len(res.Usages) != 1 {
		t.Fatalf("expected 1 usage, got %d", len(res.Usages))
	}

	u := res.Usages[0]
	if u.Model != "gpt-5.4" {
		t.Errorf("expected Model to be gpt-5.4, got %q", u.Model)
	}
	if u.InputTokens != 1000 {
		t.Errorf("expected InputTokens to be 1000, got %d", u.InputTokens)
	}
	// OutputTokens should include reasoningTokens: 200 + 50 = 250
	if u.OutputTokens != 250 {
		t.Errorf("expected OutputTokens to be 250 (200 + 50 reasoning), got %d", u.OutputTokens)
	}
	if u.CacheReadTokens != 500 {
		t.Errorf("expected CacheReadTokens to be 500, got %d", u.CacheReadTokens)
	}
	if u.CacheCreationTokens != 100 {
		t.Errorf("expected CacheCreationTokens to be 100, got %d", u.CacheCreationTokens)
	}
	if u.IsSubagent {
		t.Errorf("expected IsSubagent to be false")
	}
}

func TestParseCopilotLog_FileNotFound(t *testing.T) {
	_, err := transcript.ParseCopilotLog("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}
