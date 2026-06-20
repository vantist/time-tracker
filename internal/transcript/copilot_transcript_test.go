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

	want := transcript.ModelUsage{
		Model:               "gpt-5.4",
		InputTokens:         1000,
		OutputTokens:        250, // 200 + 50 reasoning
		CacheReadTokens:     500,
		CacheCreationTokens: 100,
		IsSubagent:          false,
	}
	if res.Usages[0] != want {
		t.Errorf("usage = %+v, want %+v", res.Usages[0], want)
	}
}

func TestParseCopilotLog_FileNotFound(t *testing.T) {
	_, err := transcript.ParseCopilotLog("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}
