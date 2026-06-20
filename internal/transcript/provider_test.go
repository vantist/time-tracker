package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

type mockProvider struct{}

func (m *mockProvider) ResolvePath(sessionID string, stdinPath string) string {
	return "mock-path"
}

func (m *mockProvider) ExtractWindow(path string, fromOffset int, toOffset int) (WindowResult, error) {
	return WindowResult{}, nil
}

func (m *mockProvider) ExtractLastTurn(path string) (WindowResult, error) {
	return WindowResult{}, nil
}

func (m *mockProvider) SupportsSubagents() bool {
	return true
}

func TestRegistry(t *testing.T) {
	mock := &mockProvider{}
	Register("mock-tool", mock)

	got, ok := GetProvider("mock-tool")
	if !ok {
		t.Fatal("expected mock-tool to be registered")
	}
	if got != mock {
		t.Errorf("got %v, want %v", got, mock)
	}

	_, ok = GetProvider("non-existent")
	if ok {
		t.Error("expected non-existent to not be registered")
	}
}

func TestJSONLProvider(t *testing.T) {
	dir := t.TempDir()

	// Write main transcript
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

	// Setup subagent
	subDir := filepath.Join(dir, "transcript", "subagents")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "agent-aaa.meta.json"),
		[]byte(`{"toolUseId":"toolu_sub1"}`), 0644)
	subLines := `{"type":"assistant","isSidechain":true,"message":{"model":"claude-haiku-4-5","usage":{"input_tokens":100,"output_tokens":50}}}` + "\n"
	os.WriteFile(filepath.Join(subDir, "agent-aaa.jsonl"), []byte(subLines), 0644)

	// Test ExtractWindow via JSONLProvider
	jp := &JSONLProvider{SupportsSub: true}
	res, err := jp.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("ExtractWindow: %v", err)
	}
	if res.InputTokens() != 110 {
		t.Errorf("InputTokens = %d, want 110", res.InputTokens())
	}
	if res.OutputTokens() != 55 {
		t.Errorf("OutputTokens = %d, want 55", res.OutputTokens())
	}

	// Test ExtractLastTurn via JSONLProvider
	resLT, err := jp.ExtractLastTurn(path)
	if err != nil {
		t.Fatalf("ExtractLastTurn: %v", err)
	}
	if resLT.InputTokens() != 110 {
		t.Errorf("ExtractLastTurn InputTokens = %d, want 110", resLT.InputTokens())
	}

	// Test SupportsSubagents
	if !jp.SupportsSubagents() {
		t.Error("expected SupportsSubagents to be true")
	}
}

func TestAntigravityProvider_ResolvePath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	sessionID := "sess-xyz"
	ap := &AntigravityProvider{}

	cliPath := filepath.Join(tmpDir, ".gemini", "antigravity-cli", "brain", sessionID, ".system_generated", "logs", "transcript.jsonl")
	legacyPath := filepath.Join(tmpDir, ".gemini", "antigravity", "brain", sessionID, ".system_generated", "logs", "transcript.jsonl")

	// 1. Neither exists -> should fallback to legacy path
	got := ap.ResolvePath(sessionID, "")
	wantLegacy := filepath.Join("~", ".gemini", "antigravity", "brain", sessionID, ".system_generated", "logs", "transcript.jsonl")
	if got != wantLegacy {
		t.Errorf("ResolvePath empty state = %q, want %q", got, wantLegacy)
	}

	// 2. Legacy path exists -> should return legacy path
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	got = ap.ResolvePath(sessionID, "")
	if got != wantLegacy {
		t.Errorf("ResolvePath legacy exists = %q, want %q", got, wantLegacy)
	}

	// 3. CLI path exists -> should return CLI path
	if err := os.MkdirAll(filepath.Dir(cliPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cliPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	got = ap.ResolvePath(sessionID, "")
	wantCli := filepath.Join("~", ".gemini", "antigravity-cli", "brain", sessionID, ".system_generated", "logs", "transcript.jsonl")
	if got != wantCli {
		t.Errorf("ResolvePath cli exists = %q, want %q", got, wantCli)
	}

	// 4. Stdin path is provided -> should return stdin path directly
	got = ap.ResolvePath(sessionID, "/custom/path.jsonl")
	if got != "/custom/path.jsonl" {
		t.Errorf("ResolvePath with stdinPath = %q, want /custom/path.jsonl", got)
	}
}
