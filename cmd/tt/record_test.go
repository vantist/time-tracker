package main

import (
	"encoding/json"
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

// TestResolvePromptInput_EnvVars_InvalidStart: invalid PROCESS_START → ProcessStart=0.
func TestResolvePromptInput_EnvVars_InvalidStart(t *testing.T) {
	t.Setenv("PROCESS_PID", "12345")
	t.Setenv("PROCESS_START", "notanumber")

	input, err := resolvePromptInputFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if input.ProcessStart != 0 {
		t.Errorf("ProcessStart = %d, want 0 (degraded)", input.ProcessStart)
	}
}
