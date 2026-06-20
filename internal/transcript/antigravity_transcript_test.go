package transcript_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/tt/internal/transcript"
)

func TestParseAntigravityLog_Normal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")

	lines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10,"cache_creation_input_tokens":5,"cache_creation":{"ephemeral_5m_input_tokens":20,"ephemeral_1h_input_tokens":30}}}}`,
		`{"type":"assistant","isSidechain":true,"message":{"model":"claude-haiku-4-5","usage":{"input_tokens":1000,"output_tokens":500}}}`, // subagent/sidechain, should be ignored
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	for _, line := range lines {
		f.WriteString(line + "\n")
	}
	f.Close()

	res, err := transcript.ParseAntigravityLog(path)
	if err != nil {
		t.Fatalf("ParseAntigravityLog failed: %v", err)
	}

	if len(res.Usages) != 1 {
		t.Fatalf("expected 1 usage, got %d", len(res.Usages))
	}

	u := res.Usages[0]
	if u.Model != "claude-sonnet-4-6" {
		t.Errorf("expected Model to be claude-sonnet-4-6, got %q", u.Model)
	}
	if u.InputTokens != 100 {
		t.Errorf("expected InputTokens to be 100, got %d", u.InputTokens)
	}
	if u.OutputTokens != 50 {
		t.Errorf("expected OutputTokens to be 50, got %d", u.OutputTokens)
	}
	if u.CacheReadTokens != 10 {
		t.Errorf("expected CacheReadTokens to be 10, got %d", u.CacheReadTokens)
	}
	if u.CacheCreationTokens != 5 {
		t.Errorf("expected CacheCreationTokens to be 5, got %d", u.CacheCreationTokens)
	}
	if u.CacheCreation5m != 20 {
		t.Errorf("expected CacheCreation5m to be 20, got %d", u.CacheCreation5m)
	}
	if u.CacheCreation1h != 30 {
		t.Errorf("expected CacheCreation1h to be 30, got %d", u.CacheCreation1h)
	}
	if u.IsSubagent {
		t.Errorf("expected IsSubagent to be false")
	}
}

func TestParseAntigravityLog_FileNotFound(t *testing.T) {
	_, err := transcript.ParseAntigravityLog("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}
