package transcript_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/tt/internal/transcript"
)

// TestExtractLastTurn_ClearRace: /clear race — lastUserIdx has no following assistant entries,
// fallback to previous turn window.
func TestExtractLastTurn_ClearRace(t *testing.T) {
	lines := []string{
		// prev turn
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50}}}`,
		// /clear user entry — no assistant follows (race condition)
		`{"type":"user","isSidechain":false}`,
	}
	path := writeLines(t, lines)

	result, err := transcript.ExtractLastTurn(path)
	if err != nil {
		t.Fatalf("ExtractLastTurn: %v", err)
	}
	// should fall back to prev turn: input=100, output=50
	if result.InputTokens() != 100 {
		t.Errorf("InputTokens = %d, want 100 (fallback to prev turn)", result.InputTokens())
	}
	if result.OutputTokens() != 50 {
		t.Errorf("OutputTokens = %d, want 50 (fallback to prev turn)", result.OutputTokens())
	}
}

// TestExtractLastTurn_Normal: normal last turn extraction.
func TestExtractLastTurn_Normal(t *testing.T) {
	lines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-haiku-4-5","usage":{"input_tokens":10,"output_tokens":5}}}`,
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-haiku-4-5","usage":{"input_tokens":200,"output_tokens":80}}}`,
	}
	path := writeLines(t, lines)

	result, err := transcript.ExtractLastTurn(path)
	if err != nil {
		t.Fatalf("ExtractLastTurn: %v", err)
	}
	if result.InputTokens() != 200 {
		t.Errorf("InputTokens = %d, want 200", result.InputTokens())
	}
}

// TestExtractWindow_SubagentBoundary: subagent tokens from after [to] are NOT counted.
func TestExtractWindow_SubagentBoundary(t *testing.T) {
	dir := t.TempDir()

	// Turn 1: lines 0-9 (agent toolu_turn1 in line 1)
	// Turn 2: lines 10-19 (agent toolu_turn2 in line 11)
	var lines []string
	for i := 0; i < 10; i++ {
		if i == 1 {
			lines = append(lines, `{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":10,"output_tokens":5},"content":[{"type":"tool_use","id":"toolu_turn1","name":"Agent"}]}}`)
		} else {
			lines = append(lines, `{"type":"user","isSidechain":false}`)
		}
	}
	for i := 0; i < 10; i++ {
		if i == 1 {
			lines = append(lines, `{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":20,"output_tokens":8},"content":[{"type":"tool_use","id":"toolu_turn2","name":"Agent"}]}}`)
		} else {
			lines = append(lines, `{"type":"user","isSidechain":false}`)
		}
	}

	path := filepath.Join(dir, "transcript.jsonl")
	f, _ := os.Create(path)
	for _, l := range lines {
		f.WriteString(l + "\n")
	}
	f.Close()

	subDir := filepath.Join(dir, "transcript", "subagents")
	os.MkdirAll(subDir, 0755)
	// subagent for turn 1
	os.WriteFile(filepath.Join(subDir, "agent-t1.meta.json"),
		[]byte(`{"toolUseId":"toolu_turn1"}`), 0644)
	os.WriteFile(filepath.Join(subDir, "agent-t1.jsonl"),
		[]byte(`{"type":"assistant","isSidechain":true,"message":{"usage":{"input_tokens":100,"output_tokens":40}}}`+"\n"), 0644)
	// subagent for turn 2
	os.WriteFile(filepath.Join(subDir, "agent-t2.meta.json"),
		[]byte(`{"toolUseId":"toolu_turn2"}`), 0644)
	os.WriteFile(filepath.Join(subDir, "agent-t2.jsonl"),
		[]byte(`{"type":"assistant","isSidechain":true,"message":{"usage":{"input_tokens":200,"output_tokens":80}}}`+"\n"), 0644)

	// Turn 1 window [0,10): should include toolu_turn1 but NOT toolu_turn2
	r1, err := transcript.ExtractWindow(path, 0, 10)
	if err != nil {
		t.Fatalf("ExtractWindow turn1: %v", err)
	}
	if r1.InputTokens() != 110 { // 10 main + 100 subagent
		t.Errorf("turn1 InputTokens = %d, want 110", r1.InputTokens())
	}

	// Turn 2 window [10,20): should include toolu_turn2 but NOT toolu_turn1
	r2, err := transcript.ExtractWindow(path, 10, 20)
	if err != nil {
		t.Fatalf("ExtractWindow turn2: %v", err)
	}
	if r2.InputTokens() != 220 { // 20 main + 200 subagent
		t.Errorf("turn2 InputTokens = %d, want 220", r2.InputTokens())
	}
}

// TestExtractWindow_WindowResult_CacheCreate5m: ExtractWindow returns WindowResult with CacheCreate5m.
func TestExtractWindow_WindowResult_CacheCreate5m(t *testing.T) {
	lines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_creation":{"ephemeral_5m_input_tokens":200,"ephemeral_1h_input_tokens":300}}}}`,
	}
	path := writeLines(t, lines)

	result, err := transcript.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("ExtractWindow: %v", err)
	}
	if result.CacheCreate5m() != 200 {
		t.Errorf("CacheCreate5m = %d, want 200", result.CacheCreate5m())
	}
	if result.CacheCreate1h() != 300 {
		t.Errorf("CacheCreate1h = %d, want 300", result.CacheCreate1h())
	}
	if result.InputTokens() != 100 {
		t.Errorf("InputTokens = %d, want 100", result.InputTokens())
	}
	if result.Model() != "claude-sonnet-4-6" {
		t.Errorf("Model = %q, want claude-sonnet-4-6", result.Model())
	}
}

// TestExtractWindow_WindowResult_NoCacheCreation: no cache_creation field returns 0, no error.
func TestExtractWindow_WindowResult_NoCacheCreation(t *testing.T) {
	lines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":50,"output_tokens":20}}}`,
	}
	path := writeLines(t, lines)

	result, err := transcript.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("ExtractWindow: %v", err)
	}
	if result.CacheCreate5m() != 0 {
		t.Errorf("CacheCreate5m = %d, want 0", result.CacheCreate5m())
	}
	if result.CacheCreate1h() != 0 {
		t.Errorf("CacheCreate1h = %d, want 0", result.CacheCreate1h())
	}
}

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

	result, err := transcript.ExtractWindow(path, 2, 4)
	if err != nil {
		t.Fatalf("ExtractWindow: %v", err)
	}
	if result.Model() != "claude-sonnet-4-6" {
		t.Errorf("Model = %q, want claude-sonnet-4-6", result.Model())
	}
	if result.InputTokens() != 100 {
		t.Errorf("InputTokens = %d, want 100", result.InputTokens())
	}
	if result.OutputTokens() != 50 {
		t.Errorf("OutputTokens = %d, want 50", result.OutputTokens())
	}
	if result.CacheReadTokens() != 10 {
		t.Errorf("CacheReadTokens = %d, want 10", result.CacheReadTokens())
	}
	if result.CacheCreationTokens() != 5 {
		t.Errorf("CacheCreationTokens = %d, want 5", result.CacheCreationTokens())
	}
}

// TestExtractWindow_ToEOF: to=-1 reads to end of file.
func TestExtractWindow_ToEOF(t *testing.T) {
	lines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-haiku-4-5","usage":{"input_tokens":200,"output_tokens":80}}}`,
	}
	path := writeLines(t, lines)

	result, err := transcript.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("ExtractWindow: %v", err)
	}
	if result.Model() != "claude-haiku-4-5" {
		t.Errorf("Model = %q, want claude-haiku-4-5", result.Model())
	}
	if result.InputTokens() != 200 {
		t.Errorf("InputTokens = %d, want 200", result.InputTokens())
	}
}

// TestExtractWindow_NotFound: missing file returns error.
func TestExtractWindow_NotFound(t *testing.T) {
	_, err := transcript.ExtractWindow("/nonexistent/path.jsonl", 0, -1)
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// TestExtractWindow_WithSubagents: subagent tokens included.
func TestExtractWindow_WithSubagents(t *testing.T) {
	dir := t.TempDir()
	mainLines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":10,"output_tokens":5},"content":[{"type":"tool_use","id":"toolu_sub1","name":"Agent"}]}}`,
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

	result, err := transcript.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("ExtractWindow: %v", err)
	}
	if result.InputTokens() != 110 {
		t.Errorf("InputTokens = %d, want 110 (10 main + 100 subagent)", result.InputTokens())
	}
	if result.OutputTokens() != 55 {
		t.Errorf("OutputTokens = %d, want 55 (5 main + 50 subagent)", result.OutputTokens())
	}
}

func TestExtractWindow_MultiModelSubagents(t *testing.T) {
	dir := t.TempDir()
	mainLines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-3-5-sonnet","usage":{"input_tokens":10,"output_tokens":5},"content":[{"type":"tool_use","id":"toolu_sub1","name":"Agent"},{"type":"tool_use","id":"toolu_sub2","name":"Agent"},{"type":"tool_use","id":"toolu_sub3","name":"Agent"}]}}`,
	}
	path := filepath.Join(dir, "transcript.jsonl")
	f, _ := os.Create(path)
	for _, l := range mainLines {
		f.WriteString(l + "\n")
	}
	f.Close()

	subDir := filepath.Join(dir, "transcript", "subagents")
	os.MkdirAll(subDir, 0755)

	// Subagent 1: claude-3-5-haiku
	os.WriteFile(filepath.Join(subDir, "agent-1.meta.json"),
		[]byte(`{"toolUseId":"toolu_sub1"}`), 0644)
	os.WriteFile(filepath.Join(subDir, "agent-1.jsonl"),
		[]byte(`{"type":"assistant","isSidechain":true,"message":{"model":"claude-3-5-haiku","usage":{"input_tokens":100,"output_tokens":50}}}`+"\n"), 0644)

	// Subagent 2: claude-3-5-haiku
	os.WriteFile(filepath.Join(subDir, "agent-2.meta.json"),
		[]byte(`{"toolUseId":"toolu_sub2"}`), 0644)
	os.WriteFile(filepath.Join(subDir, "agent-2.jsonl"),
		[]byte(`{"type":"assistant","isSidechain":true,"message":{"model":"claude-3-5-haiku","usage":{"input_tokens":200,"output_tokens":80}}}`+"\n"), 0644)

	// Subagent 3: gpt-4o
	os.WriteFile(filepath.Join(subDir, "agent-3.meta.json"),
		[]byte(`{"toolUseId":"toolu_sub3"}`), 0644)
	os.WriteFile(filepath.Join(subDir, "agent-3.jsonl"),
		[]byte(`{"type":"assistant","isSidechain":true,"message":{"model":"gpt-4o","usage":{"input_tokens":300,"output_tokens":150}}}`+"\n"), 0644)

	result, err := transcript.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("ExtractWindow: %v", err)
	}

	// We expect 3 distinct usages:
	// 1. main agent: claude-3-5-sonnet (is_subagent=false, in=10, out=5)
	// 2. subagent: claude-3-5-haiku (is_subagent=true, in=300, out=130)
	// 3. subagent: gpt-4o (is_subagent=true, in=300, out=150)
	if len(result.Usages) != 3 {
		t.Fatalf("expected 3 usages, got %d: %+v", len(result.Usages), result.Usages)
	}

	var mainSonnet, subHaiku, subGpt transcript.ModelUsage
	for _, u := range result.Usages {
		if !u.IsSubagent && u.Model == "claude-3-5-sonnet" {
			mainSonnet = u
		} else if u.IsSubagent && u.Model == "claude-3-5-haiku" {
			subHaiku = u
		} else if u.IsSubagent && u.Model == "gpt-4o" {
			subGpt = u
		}
	}

	if mainSonnet.Model == "" {
		t.Error("missing main agent claude-3-5-sonnet usage")
	} else if mainSonnet.InputTokens != 10 || mainSonnet.OutputTokens != 5 {
		t.Errorf("expected main agent usage (10, 5), got (%d, %d)", mainSonnet.InputTokens, mainSonnet.OutputTokens)
	}

	if subHaiku.Model == "" {
		t.Error("missing subagent claude-3-5-haiku usage")
	} else if subHaiku.InputTokens != 300 || subHaiku.OutputTokens != 130 {
		t.Errorf("expected subagent haiku usage (300, 130), got (%d, %d)", subHaiku.InputTokens, subHaiku.OutputTokens)
	}

	if subGpt.Model == "" {
		t.Error("missing subagent gpt-4o usage")
	} else if subGpt.InputTokens != 300 || subGpt.OutputTokens != 150 {
		t.Errorf("expected subagent gpt-4o usage (300, 150), got (%d, %d)", subGpt.InputTokens, subGpt.OutputTokens)
	}
}

func TestExtractWindow_CorruptAndTrailingLines(t *testing.T) {
	// Construct a long content block text to make the line size > 64KB (e.g. 100KB)
	longText := make([]byte, 100*1024)
	for i := range longText {
		longText[i] = 'a'
	}
	longLine := `{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50},"content":[{"type":"text","text":"` + string(longText) + `"}]}}`

	lines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":10,"output_tokens":5}}}`,
		// corrupt JSON line
		`{"type":"assistant","isSidechain":false, "message": { "model":`,
		// empty line
		``,
		// whitespace line
		`   `,
		// valid long line (>64KB)
		longLine,
		// empty line at EOF
		``,
	}
	path := writeLines(t, lines)

	result, err := transcript.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("ExtractWindow: %v", err)
	}

	if result.InputTokens() != 110 {
		t.Errorf("InputTokens = %d, want 110", result.InputTokens())
	}
	if result.OutputTokens() != 55 {
		t.Errorf("OutputTokens = %d, want 55", result.OutputTokens())
	}
}

