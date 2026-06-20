package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/user/tt/internal/db"
	"github.com/user/tt/internal/recorder"
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

func TestResolveResponseInput_Copilot(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Create events.jsonl under ~/.copilot/session-state/sess-copilot/
	logDir := filepath.Join(tempHome, ".copilot", "session-state", "sess-copilot")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("mkdir events: %v", err)
	}
	logPath := filepath.Join(logDir, "events.jsonl")
	content := `{"type":"session.shutdown","data":{"modelMetrics":{"gpt-5.4":{"usage":{"inputTokens":1000,"outputTokens":200,"cacheReadTokens":500,"cacheWriteTokens":100}}}}}` + "\n"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("write events: %v", err)
	}

	dbDir := t.TempDir()
	t.Setenv("TT_DB_PATH", filepath.Join(dbDir, "test.db"))
	conn, err := db.Open()
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer conn.Close()

	cmd := &cobra.Command{}
	cmd.Flags().String("session", "", "")
	cmd.Flags().String("tokens", "", "")
	cmd.Flags().String("tool", "claude-code", "")

	cmd.Flags().Set("session", "sess-copilot")
	cmd.Flags().Set("tool", "copilot-cli")

	sessionID, tokensJSON, model, err := resolveResponseInput(cmd, conn)
	if err != nil {
		t.Fatalf("resolveResponseInput: %v", err)
	}

	if sessionID != "sess-copilot" {
		t.Errorf("sessionID = %q, want sess-copilot", sessionID)
	}
	if model != "gpt-5.4" {
		t.Errorf("model = %q, want gpt-5.4", model)
	}

	var m map[string]int
	if err := json.Unmarshal([]byte(tokensJSON), &m); err != nil {
		t.Fatalf("unmarshal tokensJSON: %v, body: %q", err, tokensJSON)
	}
	if m["input_tokens"] != 1000 {
		t.Errorf("input_tokens = %d, want 1000", m["input_tokens"])
	}
	if m["output_tokens"] != 200 {
		t.Errorf("output_tokens = %d, want 200", m["output_tokens"])
	}
	if m["cache_read_tokens"] != 500 {
		t.Errorf("cache_read_tokens = %d, want 500", m["cache_read_tokens"])
	}
	if m["cache_creation_tokens"] != 100 {
		t.Errorf("cache_creation_tokens = %d, want 100", m["cache_creation_tokens"])
	}
}

func TestResolveResponseInput_Antigravity(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Create transcript.jsonl under ~/.gemini/antigravity/brain/sess-anti/.system_generated/logs/
	logDir := filepath.Join(tempHome, ".gemini", "antigravity", "brain", "sess-anti", ".system_generated", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("mkdir transcript: %v", err)
	}
	logPath := filepath.Join(logDir, "transcript.jsonl")
	lines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10,"cache_creation_input_tokens":5}}}`,
	}
	f, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("create transcript: %v", err)
	}
	for _, l := range lines {
		f.WriteString(l + "\n")
	}
	f.Close()

	dbDir := t.TempDir()
	t.Setenv("TT_DB_PATH", filepath.Join(dbDir, "test.db"))
	conn, err := db.Open()
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer conn.Close()

	cmd := &cobra.Command{}
	cmd.Flags().String("session", "", "")
	cmd.Flags().String("tokens", "", "")
	cmd.Flags().String("tool", "claude-code", "")

	cmd.Flags().Set("session", "sess-anti")
	cmd.Flags().Set("tool", "antigravity")

	sessionID, tokensJSON, model, err := resolveResponseInput(cmd, conn)
	if err != nil {
		t.Fatalf("resolveResponseInput: %v", err)
	}

	if sessionID != "sess-anti" {
		t.Errorf("sessionID = %q, want sess-anti", sessionID)
	}
	if model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want claude-sonnet-4-6", model)
	}

	var m map[string]int
	if err := json.Unmarshal([]byte(tokensJSON), &m); err != nil {
		t.Fatalf("unmarshal tokensJSON: %v, body: %q", err, tokensJSON)
	}
	if m["input_tokens"] != 100 {
		t.Errorf("input_tokens = %d, want 100", m["input_tokens"])
	}
	if m["output_tokens"] != 50 {
		t.Errorf("output_tokens = %d, want 50", m["output_tokens"])
	}
}

func TestRecordResponseCmd_Integration(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Create events.jsonl under ~/.copilot/session-state/sess-int-test/
	logDir := filepath.Join(tempHome, ".copilot", "session-state", "sess-int-test")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("mkdir events: %v", err)
	}
	logPath := filepath.Join(logDir, "events.jsonl")
	content := `{"type":"session.shutdown","data":{"modelMetrics":{"gpt-5.4":{"usage":{"inputTokens":1000,"outputTokens":200,"cacheReadTokens":500,"cacheWriteTokens":100}}}}}` + "\n"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("write events: %v", err)
	}

	dbDir := t.TempDir()
	t.Setenv("TT_DB_PATH", filepath.Join(dbDir, "test.db"))
	conn, err := db.Open()
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}

	// Seed session and turn
	_, err = conn.Exec(`
		INSERT INTO sessions (id, project, tool, model, branch, started_at)
		VALUES ('sess-int-test', '/proj', 'copilot-cli', '', 'main', '2026-06-20T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	_, err = conn.Exec(`
		INSERT INTO turns (session_id, prompt_at)
		VALUES ('sess-int-test', '2026-06-20T00:00:01Z')
	`)
	if err != nil {
		t.Fatalf("seed turn: %v", err)
	}
	conn.Close() // close connection because recordResponseCmd will open it again

	// Setup and run the response command
	recordResponseCmd.Flags().Set("session", "sess-int-test")
	recordResponseCmd.Flags().Set("tool", "copilot-cli")
	recordResponseCmd.Flags().Set("tokens", "")

	err = recordResponseCmd.RunE(recordResponseCmd, nil)
	if err != nil {
		t.Errorf("recordResponseCmd returned error: %v", err)
	}

	// Reopen DB and verify values
	conn, err = db.Open()
	if err != nil {
		t.Fatalf("reopen DB: %v", err)
	}
	defer conn.Close()

	var dbModel string
	conn.QueryRow("SELECT model FROM sessions WHERE id='sess-int-test'").Scan(&dbModel)
	if dbModel != "gpt-5.4" {
		t.Errorf("sessions.model = %q, want gpt-5.4", dbModel)
	}

	var input, output, cacheRead, cacheCreate int
	var cost float64
	var responseAt string
	err = conn.QueryRow(`
		SELECT input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd, response_at
		FROM turns WHERE session_id='sess-int-test'
	`).Scan(&input, &output, &cacheRead, &cacheCreate, &cost, &responseAt)
	if err != nil {
		t.Fatalf("query turn: %v", err)
	}

	if input != 1000 || output != 200 || cacheRead != 500 || cacheCreate != 100 {
		t.Errorf("turn tokens: input=%d output=%d read=%d create=%d", input, output, cacheRead, cacheCreate)
	}

	// pricing for gpt-5.4: input=$5.00/MTok, output=$15.00/MTok, cacheRead=$0.50/MTok, cacheCreate=$6.25/MTok
	// cost = 1000/1e6*5.00 + 200/1e6*15.00 + 500/1e6*0.50 + 100/1e6*6.25
	//      = 0.005 + 0.003 + 0.00025 + 0.000625 = 0.008875
	const expectedCost = 0.008875
	if cost < expectedCost-0.000001 || cost > expectedCost+0.000001 {
		t.Errorf("estimated_cost_usd = %f, want ~%f", cost, expectedCost)
	}

	if responseAt == "" {
		t.Error("response_at was not populated")
	}

	// Test silent error handling (no error returned, exit 0 equivalent)
	recordResponseCmd.Flags().Set("session", "non-existent-session-should-not-cause-command-error")
	err = recordResponseCmd.RunE(recordResponseCmd, nil)
	if err != nil {
		t.Errorf("expected command to exit silently with nil error even on db mismatch, got: %v", err)
	}
}

func TestReadStdinJSON_Antigravity(t *testing.T) {
	// Mock Stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	inputJSON := `{"conversationId": "gemini-session-123", "transcriptPath": "/path/to/transcript.jsonl"}`
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.Write([]byte(inputJSON))
		w.Close()
	}()

	payload, err := readStdinJSON("antigravity")
	if err != nil {
		t.Fatalf("readStdinJSON failed: %v", err)
	}
	if payload == nil {
		t.Fatal("expected payload to be non-nil")
	}

	if payload.SessionID != "gemini-session-123" {
		t.Errorf("SessionID = %q, want %q", payload.SessionID, "gemini-session-123")
	}
	if payload.TranscriptPath != "/path/to/transcript.jsonl" {
		t.Errorf("TranscriptPath = %q, want %q", payload.TranscriptPath, "/path/to/transcript.jsonl")
	}
}

func TestReadStdinJSON_Codex(t *testing.T) {
	// Mock Stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	inputJSON := `{"session_id": "codex-session-abc", "transcript_path": "/path/to/codex.jsonl"}`
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.Write([]byte(inputJSON))
		w.Close()
	}()

	payload, err := readStdinJSON("codex")
	if err != nil {
		t.Fatalf("readStdinJSON failed: %v", err)
	}
	if payload == nil {
		t.Fatal("expected payload to be non-nil")
	}

	if payload.SessionID != "codex-session-abc" {
		t.Errorf("SessionID = %q, want %q", payload.SessionID, "codex-session-abc")
	}
	if payload.TranscriptPath != "/path/to/codex.jsonl" {
		t.Errorf("TranscriptPath = %q, want %q", payload.TranscriptPath, "/path/to/codex.jsonl")
	}
}

func TestRecordPrompt_Copilot(t *testing.T) {
	// Mock Stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	inputJSON := `{"sessionId": "sess-copilot-prompt", "cwd": "/mock/cwd", "transcriptPath": "/mock/transcript.jsonl"}`
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.Write([]byte(inputJSON))
		w.Close()
	}()

	dbDir := t.TempDir()
	t.Setenv("TT_DB_PATH", filepath.Join(dbDir, "test.db"))
	conn, err := db.Open()
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer conn.Close()

	cmd := &cobra.Command{}
	cmd.Flags().String("session", "", "")
	cmd.Flags().String("project", "", "")
	cmd.Flags().String("tool", "claude-code", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("transcript-path", "", "")

	cmd.Flags().Set("tool", "copilot-cli")

	input, err := resolvePromptInput(cmd)
	if err != nil {
		t.Fatalf("resolvePromptInput failed: %v", err)
	}

	if input.SessionID != "sess-copilot-prompt" {
		t.Errorf("SessionID = %q, want sess-copilot-prompt", input.SessionID)
	}
	if input.Project != "/mock/cwd" {
		t.Errorf("Project = %q, want /mock/cwd", input.Project)
	}
	if input.TranscriptPath != "/mock/transcript.jsonl" {
		t.Errorf("TranscriptPath = %q, want /mock/transcript.jsonl", input.TranscriptPath)
	}

	// Record to DB
	err = recorder.RecordPrompt(conn, input)
	if err != nil {
		t.Fatalf("RecordPrompt failed: %v", err)
	}

	// Verify DB entry
	var dbProject, dbTool string
	err = conn.QueryRow("SELECT project, tool FROM sessions WHERE id='sess-copilot-prompt'").Scan(&dbProject, &dbTool)
	if err != nil {
		t.Fatalf("Query session failed: %v", err)
	}
	if dbProject != "/mock/cwd" {
		t.Errorf("dbProject = %q, want /mock/cwd", dbProject)
	}
	if dbTool != "copilot-cli" {
		t.Errorf("dbTool = %q, want copilot-cli", dbTool)
	}

	var dbTranscriptPath string
	err = conn.QueryRow("SELECT transcript_path FROM turns WHERE session_id='sess-copilot-prompt'").Scan(&dbTranscriptPath)
	if err != nil {
		t.Fatalf("Query turn failed: %v", err)
	}
	if dbTranscriptPath != "/mock/transcript.jsonl" {
		t.Errorf("dbTranscriptPath = %q, want /mock/transcript.jsonl", dbTranscriptPath)
	}
}

func TestResolvePromptInput_FallbackAndDefaultModel(t *testing.T) {
	// Mock HOME for settings
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create a mock settings.json for antigravity
	cliConfigDir := filepath.Join(tmpDir, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(cliConfigDir, 0o700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	settingsPath := filepath.Join(cliConfigDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"model": "Gemini 2.5 Flash Test"}`), 0o600); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	// Mock empty stdin (no interactive terminal stdin)
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Close()

	newCmd := func(tool, model string) *cobra.Command {
		cmd := &cobra.Command{}
		cmd.Flags().String("session", "", "")
		cmd.Flags().String("project", "", "")
		cmd.Flags().String("tool", tool, "")
		cmd.Flags().String("model", model, "")
		cmd.Flags().String("transcript-path", "", "")
		return cmd
	}

	t.Run("project path fallback to working dir", func(t *testing.T) {
		cmd := newCmd("claude-code", "")
		input, err := resolvePromptInput(cmd)
		if err != nil {
			t.Fatalf("resolvePromptInput: %v", err)
		}
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("os.Getwd: %v", err)
		}
		if input.Project != wd {
			t.Errorf("Project = %q, want fallback to working dir %q", input.Project, wd)
		}
	})

	t.Run("antigravity empty model fallback to settings", func(t *testing.T) {
		cmd := newCmd("antigravity", "")
		input, err := resolvePromptInput(cmd)
		if err != nil {
			t.Fatalf("resolvePromptInput (antigravity): %v", err)
		}
		if input.Model != "gemini-2.5-flash-test" {
			t.Errorf("Model = %q, want default gemini-2.5-flash-test", input.Model)
		}
	})
}
