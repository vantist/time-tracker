package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDebugLog_LLMRequest(t *testing.T) {
	content := `{"v":1,"ts":1780000000000,"type":"llm_request","attrs":{"model":"gpt-4o","inputTokens":100,"outputTokens":50,"cachedTokens":10}}
{"v":1,"ts":1780000001000,"type":"llm_request","attrs":{"model":"gpt-4o","inputTokens":200,"outputTokens":80,"cachedTokens":20}}
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "main.jsonl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseDebugLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.InputTokens != 300 {
		t.Errorf("expected 300 input tokens, got %d", result.InputTokens)
	}
	if result.OutputTokens != 130 {
		t.Errorf("expected 130 output tokens, got %d", result.OutputTokens)
	}
	if result.CachedTokens != 30 {
		t.Errorf("expected 30 cached tokens, got %d", result.CachedTokens)
	}
	if result.ModelTurns != 2 {
		t.Errorf("expected 2 model turns, got %d", result.ModelTurns)
	}

	u := result.ModelBreakdown["gpt-4o"]
	if u.InputTokens != 300 {
		t.Errorf("expected gpt-4o input tokens 300, got %d", u.InputTokens)
	}
	if u.OutputTokens != 130 {
		t.Errorf("expected gpt-4o output tokens 130, got %d", u.OutputTokens)
	}
}

func TestParseDebugLog_SessionShutdown(t *testing.T) {
	content := `{"v":1,"ts":1780000000000,"type":"session.shutdown","data":{"mainModel":"gpt-4o","modelMetrics":{"gpt-4o":{"usage":{"inputTokens":100,"outputTokens":50,"cacheReadTokens":10,"cacheWriteTokens":5}}},"totalNanoAiu":123456789}}
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "main.jsonl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseDebugLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalNanoAiu != 123456789 {
		t.Errorf("expected totalNanoAiu 123456789, got %f", result.TotalNanoAiu)
	}

	u := result.ModelBreakdown["gpt-4o"]
	if u.InputTokens != 100 {
		t.Errorf("expected gpt-4o input tokens 100, got %d", u.InputTokens)
	}
	if u.CacheReadTokens != 10 {
		t.Errorf("expected gpt-4o cache read tokens 10, got %d", u.CacheReadTokens)
	}
}

func TestParseDebugLog_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseDebugLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.InputTokens != 0 {
		t.Errorf("expected 0 input tokens, got %d", result.InputTokens)
	}
	if result.OutputTokens != 0 {
		t.Errorf("expected 0 output tokens, got %d", result.OutputTokens)
	}
}

func TestParseDebugLog_HandlesMalformedJSON(t *testing.T) {
	content := `{"v":1,"ts":1780000000000,"type":"llm_request","attrs":{"model":"gpt-4o","inputTokens":100,"outputTokens":50}}
not valid json
{"v":1,"ts":1780000001000,"type":"llm_request","attrs":{"model":"gpt-4o","inputTokens":200,"outputTokens":80}}
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "main.jsonl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseDebugLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.InputTokens != 300 {
		t.Errorf("expected 300 input tokens despite malformed JSON, got %d", result.InputTokens)
	}
}

func TestParseDebugLog_SessionShutdown_CurrentModelFallback(t *testing.T) {
	content := `{"v":1,"ts":1780000000000,"type":"session.shutdown","data":{"mainModel":"","currentModel":"gpt-5","modelMetrics":{"gpt-5":{"usage":{"inputTokens":100,"outputTokens":50,"cacheReadTokens":10,"cacheWriteTokens":5}},"gpt-5-mini":{"usage":{"inputTokens":40,"outputTokens":20}}},"totalNanoAiu":123456789}}
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "main.jsonl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseDebugLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	main := result.ModelBreakdown["gpt-5"]
	if main.Model != "gpt-5" {
		t.Errorf("expected main model gpt-5, got %q", main.Model)
	}
	if main.IsSubagent {
		t.Error("expected gpt-5 to be main (IsSubagent=false), judged by currentModel")
	}

	sub := result.ModelBreakdown["gpt-5-mini"]
	if !sub.IsSubagent {
		t.Error("expected gpt-5-mini to be subagent (judged by currentModel)")
	}
}

func TestDiscoverDebugLogPath(t *testing.T) {
	path := DiscoverDebugLogPath("/workspace/storage", "session-123")
	expected := "/workspace/storage/debug-logs/session-123/main.jsonl"
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}
