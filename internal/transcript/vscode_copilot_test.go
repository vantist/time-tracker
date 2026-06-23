package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVSCodeCopilotProvider_ResolvePath(t *testing.T) {
	p := &VSCodeCopilotProvider{}

	t.Run("uses stdin path when provided", func(t *testing.T) {
		result := p.ResolvePath("session-123", "/custom/path.jsonl")
		if result != "/custom/path.jsonl" {
			t.Errorf("expected /custom/path.jsonl, got %s", result)
		}
	})

	t.Run("falls back to workspaceStorage path", func(t *testing.T) {
		result := p.ResolvePath("session-123", "")
		if result == "" {
			t.Error("expected non-empty path")
		}
	})
}

func TestVSCodeCopilotProvider_ExtractWindow_SessionShutdown(t *testing.T) {
	content := `{"type":"session.start","data":{"sessionId":"test-123","startTime":"2026-06-24T00:00:00.000Z","copilotVersion":"0.52.0","vscodeVersion":"1.124.0"},"timestamp":"2026-06-24T00:00:00.000Z"}
{"type":"user.message","data":{"content":"hello"},"timestamp":"2026-06-24T00:00:01.000Z"}
{"type":"assistant.message","data":{"content":"hi there","toolRequests":[],"reasoningText":""},"timestamp":"2026-06-24T00:00:02.000Z"}
{"type":"session.shutdown","data":{"mainModel":"gpt-4o","modelMetrics":{"gpt-4o":{"usage":{"inputTokens":100,"outputTokens":50,"cacheReadTokens":10,"cacheWriteTokens":5}}}},"timestamp":"2026-06-24T00:00:03.000Z"}
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p := &VSCodeCopilotProvider{}
	result, err := p.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Usages) == 0 {
		t.Fatal("expected at least one usage")
	}

	u := result.Usages[0]
	if u.Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", u.Model)
	}
	if u.InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", u.InputTokens)
	}
	if u.OutputTokens != 50 {
		t.Errorf("expected 50 output tokens, got %d", u.OutputTokens)
	}
	if u.CacheReadTokens != 10 {
		t.Errorf("expected 10 cache read tokens, got %d", u.CacheReadTokens)
	}
	if u.CacheCreationTokens != 5 {
		t.Errorf("expected 5 cache creation tokens, got %d", u.CacheCreationTokens)
	}
}

func TestVSCodeCopilotProvider_HandlesMalformedJSON(t *testing.T) {
	content := `{"type":"session.start","data":{"sessionId":"test-123"},"timestamp":"2026-06-24T00:00:00.000Z"}
this is not valid json
{"type":"session.shutdown","data":{"mainModel":"gpt-4o","modelMetrics":{"gpt-4o":{"usage":{"inputTokens":100,"outputTokens":50}}}},"timestamp":"2026-06-24T00:00:01.000Z"}
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p := &VSCodeCopilotProvider{}
	result, err := p.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Usages) == 0 {
		t.Error("expected at least one usage despite malformed JSON")
	}
}

func TestVSCodeCopilotProvider_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	p := &VSCodeCopilotProvider{}
	result, err := p.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Usages) != 0 {
		t.Errorf("expected 0 usages for empty file, got %d", len(result.Usages))
	}
}

func TestVSCodeCopilotProvider_SupportsSubagents(t *testing.T) {
	p := &VSCodeCopilotProvider{}
	if p.SupportsSubagents() {
		t.Error("expected false for VS Code Copilot provider")
	}
}
