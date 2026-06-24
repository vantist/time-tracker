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

func TestParseCopilotLog_CurrentModelFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	lines := []string{
		`{"type":"session.start","data":{}}`,
		`{"type":"session.shutdown","data":{"mainModel":"","currentModel":"gpt-5","modelMetrics":{"gpt-5":{"usage":{"inputTokens":1000,"outputTokens":200}},"gpt-5-mini":{"usage":{"inputTokens":500,"outputTokens":100}}}}}`,
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

	if got := res.Model(); got != "gpt-5" {
		t.Errorf("Model() = %q, want %q (currentModel fallback)", got, "gpt-5")
	}

	var mainUsage, subUsage transcript.ModelUsage
	for _, u := range res.Usages {
		if u.Model == "gpt-5" {
			mainUsage = u
		} else if u.Model == "gpt-5-mini" {
			subUsage = u
		}
	}
	if mainUsage.IsSubagent {
		t.Error("expected main model gpt-5 to have IsSubagent=false (judged by currentModel)")
	}
	if !subUsage.IsSubagent {
		t.Error("expected subagent model gpt-5-mini to have IsSubagent=true (judged by currentModel)")
	}
}

func TestParseCopilotLog_MainModelPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	lines := []string{
		`{"type":"session.start","data":{}}`,
		`{"type":"session.shutdown","data":{"mainModel":"gpt-5","currentModel":"claude-3.5","modelMetrics":{"gpt-5":{"usage":{"inputTokens":1000,"outputTokens":200}},"claude-3.5":{"usage":{"inputTokens":500,"outputTokens":100}}}}}`,
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

	if got := res.Model(); got != "gpt-5" {
		t.Errorf("Model() = %q, want %q (mainModel takes precedence)", got, "gpt-5")
	}

	var subUsage transcript.ModelUsage
	for _, u := range res.Usages {
		if u.Model == "claude-3.5" {
			subUsage = u
		}
	}
	if !subUsage.IsSubagent {
		t.Error("expected claude-3.5 to be subagent when mainModel=gpt-5")
	}
}

func TestCopilotProvider_Subagents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	lines := []string{
		`{"type":"session.start","data":{}}`,
		`{"type":"session.shutdown","data":{"mainModel":"gpt-5.4","modelMetrics":{"gpt-5.4":{"usage":{"inputTokens":1000,"outputTokens":200}},"gpt-5-mini":{"usage":{"inputTokens":500,"outputTokens":100}}}}}`,
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	for _, line := range lines {
		f.WriteString(line + "\n")
	}
	f.Close()

	// Get provider
	p, ok := transcript.GetProvider("copilot-cli")
	if !ok {
		t.Fatal("expected copilot-cli provider to be registered")
	}

	res, err := p.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("ExtractWindow failed: %v", err)
	}

	if len(res.Usages) != 2 {
		t.Fatalf("expected 2 usages, got %d: %+v", len(res.Usages), res.Usages)
	}

	var mainUsage, subUsage transcript.ModelUsage
	for _, u := range res.Usages {
		if u.Model == "gpt-5.4" {
			mainUsage = u
		} else if u.Model == "gpt-5-mini" {
			subUsage = u
		}
	}

	if mainUsage.IsSubagent {
		t.Error("expected main model gpt-5.4 to have IsSubagent = false")
	}
	if mainUsage.InputTokens != 1000 || mainUsage.OutputTokens != 200 {
		t.Errorf("unexpected main usage: %+v", mainUsage)
	}

	if !subUsage.IsSubagent {
		t.Error("expected subagent model gpt-5-mini to have IsSubagent = true")
	}
	if subUsage.InputTokens != 500 || subUsage.OutputTokens != 100 {
		t.Errorf("unexpected subagent usage: %+v", subUsage)
	}
}
