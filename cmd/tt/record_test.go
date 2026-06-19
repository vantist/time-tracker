package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestResolvePromptInput_NoEnvVars: when PROCESS_PID and PROCESS_START are both
// unset, resolvePromptInputFromEnv must call process.StartTime(os.Getppid()).
func TestResolvePromptInput_NoEnvVars(t *testing.T) {
	t.Setenv("PROCESS_PID", "")
	t.Setenv("PROCESS_START", "")

	input, err := resolvePromptInputFromEnv()
	if err != nil {
		t.Fatalf("resolvePromptInputFromEnv: %v", err)
	}

	ppid := os.Getppid()
	if int(input.ProcessPID) != ppid {
		t.Errorf("ProcessPID = %d, want %d (ppid)", input.ProcessPID, ppid)
	}
	// On darwin, StartTime returns a valid timestamp; on other platforms it is 0.
	// We only assert non-negative to allow the degraded path.
	if input.ProcessStart < 0 {
		t.Errorf("ProcessStart = %d, want >= 0", input.ProcessStart)
	}
}

// TestResolvePromptInput_EnvVars: PROCESS_PID and PROCESS_START env vars are
// parsed and placed into PromptInput correctly.
func TestResolvePromptInput_EnvVars(t *testing.T) {
	t.Setenv("PROCESS_PID", "12345")
	t.Setenv("PROCESS_START", "1700000000")

	input, err := resolvePromptInputFromEnv()
	if err != nil {
		t.Fatalf("resolvePromptInputFromEnv: %v", err)
	}

	if input.ProcessPID != 12345 {
		t.Errorf("ProcessPID = %d, want 12345", input.ProcessPID)
	}
	if input.ProcessStart != 1700000000 {
		t.Errorf("ProcessStart = %d, want 1700000000", input.ProcessStart)
	}
}

// TestExtractFromTranscript_ClearRace: Stop fires after /clear — lastUserIdx is the
// /clear entry with no assistant entries after it. Fallback extracts tokens from
// the previous turn window.
func TestExtractFromTranscript_ClearRace(t *testing.T) {
	lines := []string{
		// Turn 1: user prompt
		`{"type":"user","isSidechain":false}`,
		// Turn 1: assistant response (two blocks with identical usage = one API call)
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10,"cache_creation_input_tokens":0}}}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10,"cache_creation_input_tokens":0}}}`,
		// /clear: new user entry appended before Stop fires
		`{"type":"user","isSidechain":false}`,
		// No assistant entry follows /clear yet (race)
	}
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	f, _ := os.Create(path)
	for _, l := range lines {
		f.WriteString(l + "\n")
	}
	f.Close()

	tokensJSON, model := extractFromTranscript(path)

	if model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want claude-sonnet-4-6", model)
	}
	if tokensJSON == "" {
		t.Fatal("tokensJSON empty, want token counts from previous turn")
	}
	var m map[string]int
	if err := json.Unmarshal([]byte(tokensJSON), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["input_tokens"] != 100 {
		t.Errorf("input_tokens = %d, want 100", m["input_tokens"])
	}
	if m["output_tokens"] != 50 {
		t.Errorf("output_tokens = %d, want 50", m["output_tokens"])
	}
	if m["cache_read_tokens"] != 10 {
		t.Errorf("cache_read_tokens = %d, want 10", m["cache_read_tokens"])
	}
}

// TestExtractFromTranscriptAtOffset_OffsetCuts: only assistant entries from offset onwards are summed.
func TestExtractFromTranscriptAtOffset_OffsetCuts(t *testing.T) {
	lines := []string{
		// lines 0-1: old turn (before offset)
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"old-model","usage":{"input_tokens":999,"output_tokens":999,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
		// line 2: user prompt — this is the offset anchor
		`{"type":"user","isSidechain":false}`,
		// lines 3-4: new turn assistant entries (dedup: same usage = 1 API call)
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10,"cache_creation_input_tokens":0}}}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10,"cache_creation_input_tokens":0}}}`,
	}
	path := writeTranscript(t, lines)

	// offset = 2 (line count when prompt was recorded, i.e. first 2 lines existed)
	tokensJSON, model := extractFromTranscriptAtOffset(path, 2)

	if model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want claude-sonnet-4-6", model)
	}
	var m map[string]int
	if err := json.Unmarshal([]byte(tokensJSON), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["input_tokens"] != 100 {
		t.Errorf("input_tokens = %d, want 100 (old turn must be excluded)", m["input_tokens"])
	}
	if m["output_tokens"] != 50 {
		t.Errorf("output_tokens = %d, want 50", m["output_tokens"])
	}
}

// TestExtractFromTranscriptAtOffset_ZeroOffset: behaves like full-transcript scan.
func TestExtractFromTranscriptAtOffset_ZeroOffset(t *testing.T) {
	lines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-haiku-4-5","usage":{"input_tokens":200,"output_tokens":80,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
	}
	path := writeTranscript(t, lines)

	tokensJSON, _ := extractFromTranscriptAtOffset(path, 0)

	var m map[string]int
	if err := json.Unmarshal([]byte(tokensJSON), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["input_tokens"] != 200 {
		t.Errorf("input_tokens = %d, want 200", m["input_tokens"])
	}
}

// TestExtractFromTranscriptAtOffset_OffsetBeyondEnd: returns empty when offset >= line count.
func TestExtractFromTranscriptAtOffset_OffsetBeyondEnd(t *testing.T) {
	lines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
	}
	path := writeTranscript(t, lines)

	tokensJSON, _ := extractFromTranscriptAtOffset(path, 999)

	if tokensJSON != "" {
		t.Errorf("tokensJSON = %q, want empty when offset beyond end", tokensJSON)
	}
}

func writeTranscript(t *testing.T, lines []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create transcript: %v", err)
	}
	for _, l := range lines {
		f.WriteString(l + "\n")
	}
	f.Close()
	return path
}

// writeTranscriptInDir writes a transcript.jsonl in a given dir and returns the path.
// subagents go in <dir>/transcript/subagents/.
func writeTranscriptInDir(t *testing.T, dir string, lines []string) string {
	t.Helper()
	path := filepath.Join(dir, "transcript.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create transcript: %v", err)
	}
	for _, l := range lines {
		f.WriteString(l + "\n")
	}
	f.Close()
	return path
}

func makeSubagentFixture(t *testing.T, dir, agentID, toolUseID string, agentLines []string) {
	t.Helper()
	subDir := filepath.Join(dir, "transcript", "subagents")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir subagents: %v", err)
	}
	// meta.json
	metaPath := filepath.Join(subDir, "agent-"+agentID+".meta.json")
	metaData := `{"toolUseId":"` + toolUseID + `","agentType":"test","description":"test"}`
	if err := os.WriteFile(metaPath, []byte(metaData), 0644); err != nil {
		t.Fatalf("write meta: %v", err)
	}
	// agent jsonl
	jsonlPath := filepath.Join(subDir, "agent-"+agentID+".jsonl")
	f, err := os.Create(jsonlPath)
	if err != nil {
		t.Fatalf("create agent jsonl: %v", err)
	}
	for _, l := range agentLines {
		f.WriteString(l + "\n")
	}
	f.Close()
}

// agentEntry creates a main-transcript assistant entry with an Agent tool_use content block.
func agentEntry(toolUseID string) string {
	return `{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":10,"output_tokens":5},"content":[{"type":"tool_use","id":"` + toolUseID + `","name":"Agent"}]}}`
}

// subagentAssistantEntry creates a subagent assistant entry (isSidechain=true).
func subagentAssistantEntry(input, output, cacheRead, cacheCreate int) string {
	return fmt.Sprintf(`{"type":"assistant","isSidechain":true,"message":{"model":"claude-haiku-4-5","usage":{"input_tokens":%d,"output_tokens":%d,"cache_read_input_tokens":%d,"cache_creation_input_tokens":%d}}}`,
		input, output, cacheRead, cacheCreate)
}

// Note: extractSubagentTokens and related types are now in internal/transcript package.
// Tests for those are in internal/transcript/extract_test.go.

// TestExtractFromTranscriptAtOffset_WithSubagents: subagent tokens are included in final result.
func TestExtractFromTranscriptAtOffset_WithSubagents(t *testing.T) {
	dir := t.TempDir()
	// Main transcript: user entry at 0, agent call at 1, assistant at 2
	mainLines := []string{
		`{"type":"user","isSidechain":false}`,
		agentEntry("toolu_sub1"),
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
	}
	path := writeTranscriptInDir(t, dir, mainLines)
	makeSubagentFixture(t, dir, "eee", "toolu_sub1", []string{
		subagentAssistantEntry(100, 50, 20, 10),
	})

	// offset=0 to include all entries
	tokensJSON, model := extractFromTranscriptAtOffset(path, 0)

	if model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want claude-sonnet-4-6", model)
	}
	if tokensJSON == "" {
		t.Fatal("tokensJSON empty, want combined token counts")
	}
	var m map[string]int
	if err := json.Unmarshal([]byte(tokensJSON), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Main: input=10, output=5 + Subagent: input=100, output=50 = total input=110, output=55
	if m["input_tokens"] != 110 {
		t.Errorf("input_tokens = %d, want 110 (10 main + 100 subagent)", m["input_tokens"])
	}
	if m["output_tokens"] != 55 {
		t.Errorf("output_tokens = %d, want 55 (5 main + 50 subagent)", m["output_tokens"])
	}
	if m["cache_read_tokens"] != 20 {
		t.Errorf("cache_read_tokens = %d, want 20", m["cache_read_tokens"])
	}
	if m["cache_creation_tokens"] != 10 {
		t.Errorf("cache_creation_tokens = %d, want 10", m["cache_creation_tokens"])
	}
}

// TestResolvePromptInput_EnvVars_InvalidStart: invalid PROCESS_START → falls back to ppid,
// ignoring the env override (both env vars must parse successfully to use override).
func TestResolvePromptInput_EnvVars_InvalidStart(t *testing.T) {
	t.Setenv("PROCESS_PID", "12345")
	t.Setenv("PROCESS_START", "notanumber")

	input, err := resolvePromptInputFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalid env vars → fallback to os.Getppid(); env PID (12345) is NOT used.
	if int(input.ProcessPID) != os.Getppid() {
		t.Errorf("ProcessPID = %d, want %d (ppid fallback)", input.ProcessPID, os.Getppid())
	}
}
