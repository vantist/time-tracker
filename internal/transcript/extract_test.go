package transcript_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/user/tt/internal/transcript"
)

func writeLines(t *testing.T, lines []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	for _, l := range lines {
		f.WriteString(l + "\n")
	}
	f.Close()
	return path
}

func unmarshal(t *testing.T, tokensJSON string) map[string]int {
	t.Helper()
	if tokensJSON == "" {
		t.Fatal("tokensJSON is empty")
	}
	var m map[string]int
	if err := json.Unmarshal([]byte(tokensJSON), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

// TestExtractWindow_Range: only entries from offset to nextOffset are counted.
func TestExtractWindow_Range(t *testing.T) {
	lines := []string{
		// old turn (before offset)
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"old-model","usage":{"input_tokens":999,"output_tokens":999}}}`,
		// offset=2: new turn starts here
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10,"cache_creation_input_tokens":5}}}`,
		// nextOffset=4: stop here (these entries not included)
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":200,"output_tokens":80}}}`,
	}
	path := writeLines(t, lines)

	tokensJSON, model, err := transcript.ExtractWindow(path, 2, 4)
	if err != nil {
		t.Fatalf("ExtractWindow: %v", err)
	}
	if model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want claude-sonnet-4-6", model)
	}
	m := unmarshal(t, tokensJSON)
	if m["input_tokens"] != 100 {
		t.Errorf("input_tokens = %d, want 100", m["input_tokens"])
	}
	if m["output_tokens"] != 50 {
		t.Errorf("output_tokens = %d, want 50", m["output_tokens"])
	}
	if m["cache_read_tokens"] != 10 {
		t.Errorf("cache_read_tokens = %d, want 10", m["cache_read_tokens"])
	}
	if m["cache_creation_tokens"] != 5 {
		t.Errorf("cache_creation_tokens = %d, want 5", m["cache_creation_tokens"])
	}
}

// TestExtractWindow_ToEOF: to=-1 reads to end of file.
func TestExtractWindow_ToEOF(t *testing.T) {
	lines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-haiku-4-5","usage":{"input_tokens":200,"output_tokens":80}}}`,
	}
	path := writeLines(t, lines)

	tokensJSON, model, err := transcript.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("ExtractWindow: %v", err)
	}
	if model != "claude-haiku-4-5" {
		t.Errorf("model = %q, want claude-haiku-4-5", model)
	}
	m := unmarshal(t, tokensJSON)
	if m["input_tokens"] != 200 {
		t.Errorf("input_tokens = %d, want 200", m["input_tokens"])
	}
}

// TestExtractWindow_NotFound: missing file returns error.
func TestExtractWindow_NotFound(t *testing.T) {
	_, _, err := transcript.ExtractWindow("/nonexistent/path.jsonl", 0, -1)
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// TestExtractWindow_WithSubagents: subagent tokens included.
func TestExtractWindow_WithSubagents(t *testing.T) {
	dir := t.TempDir()
	mainLines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":10,"output_tokens":5}},"content":[{"type":"tool_use","id":"toolu_sub1","name":"Agent"}]}`,
	}
	path := filepath.Join(dir, "transcript.jsonl")
	f, _ := os.Create(path)
	for _, l := range mainLines {
		f.WriteString(l + "\n")
	}
	f.Close()

	subDir := filepath.Join(dir, "transcript", "subagents")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "agent-aaa.meta.json"),
		[]byte(`{"toolUseId":"toolu_sub1","agentType":"test","description":"test"}`), 0644)
	subLines := `{"type":"assistant","isSidechain":true,"message":{"model":"claude-haiku-4-5","usage":{"input_tokens":100,"output_tokens":50}}}` + "\n"
	os.WriteFile(filepath.Join(subDir, "agent-aaa.jsonl"), []byte(subLines), 0644)

	tokensJSON, _, err := transcript.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("ExtractWindow: %v", err)
	}
	m := unmarshal(t, tokensJSON)
	if m["input_tokens"] != 110 {
		t.Errorf("input_tokens = %d, want 110 (10 main + 100 subagent)", m["input_tokens"])
	}
	if m["output_tokens"] != 55 {
		t.Errorf("output_tokens = %d, want 55 (5 main + 50 subagent)", m["output_tokens"])
	}
}
