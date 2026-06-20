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

	want := transcript.ModelUsage{
		Model:               "claude-sonnet-4-6",
		InputTokens:         100,
		OutputTokens:        50,
		CacheReadTokens:     10,
		CacheCreationTokens: 5,
		CacheCreation5m:     20,
		CacheCreation1h:     30,
		IsSubagent:          false,
	}
	if res.Usages[0] != want {
		t.Errorf("usage = %+v, want %+v", res.Usages[0], want)
	}
}

func TestParseAntigravityLog_FileNotFound(t *testing.T) {
	_, err := transcript.ParseAntigravityLog("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestParseAntigravityLog_SettingsAndZeroTokens(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create ~/.gemini/antigravity-cli/settings.json
	cliConfigDir := filepath.Join(tmpDir, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(cliConfigDir, 0o700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	settingsPath := filepath.Join(cliConfigDir, "settings.json")
	settingsJSON := `{"model": "Gemini 3.5 Flash (Medium)"}`
	if err := os.WriteFile(settingsPath, []byte(settingsJSON), 0o600); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	// Create transcript.jsonl with typical Antigravity lines (no token usage)
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	transcriptContent := `{"type":"USER_INPUT","content":"hello"}
{"type":"PLANNER_RESPONSE","status":"DONE"}
`
	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	// Parse
	res, err := transcript.ParseAntigravityLog(transcriptPath)
	if err != nil {
		t.Fatalf("ParseAntigravityLog failed: %v", err)
	}

	// Should return 1 usage, with normalized model name, and 0 tokens
	if len(res.Usages) != 1 {
		t.Fatalf("expected 1 usage, got %d", len(res.Usages))
	}

	u := res.Usages[0]
	if u.Model != "gemini-3.5-flash" {
		t.Errorf("expected model = %q, got %q", "gemini-3.5-flash", u.Model)
	}
	if u.InputTokens != 0 || u.OutputTokens != 0 {
		t.Errorf("expected 0 tokens, got Input=%d, Output=%d", u.InputTokens, u.OutputTokens)
	}
	if u.IsSubagent {
		t.Error("expected IsSubagent = false")
	}
}

func TestParseAntigravityLog_SettingsFallback(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create ~/.gemini/antigravity/settings.json (no -cli suffix)
	configDir := filepath.Join(tmpDir, ".gemini", "antigravity")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	settingsPath := filepath.Join(configDir, "settings.json")
	settingsJSON := `{"model": "Gemini 2.5 Pro"}`
	if err := os.WriteFile(settingsPath, []byte(settingsJSON), 0o600); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	// Create transcript
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	if err := os.WriteFile(transcriptPath, []byte(`{"type":"PLANNER_RESPONSE"}`), 0o600); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	res, err := transcript.ParseAntigravityLog(transcriptPath)
	if err != nil {
		t.Fatalf("ParseAntigravityLog failed: %v", err)
	}

	if len(res.Usages) != 1 {
		t.Fatalf("expected 1 usage, got %d", len(res.Usages))
	}
	if res.Usages[0].Model != "gemini-2.5-pro" {
		t.Errorf("expected model = %q, got %q", "gemini-2.5-pro", res.Usages[0].Model)
	}
}

func TestGetAntigravityModel(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	t.Run("default model when settings missing", func(t *testing.T) {
		if model := transcript.GetAntigravityModel(nil); model != "gemini-3.5-flash" {
			t.Errorf("expected default model gemini-3.5-flash, got %q", model)
		}
	})

	t.Run("fallback settings model", func(t *testing.T) {
		configDir := filepath.Join(tmpDir, ".gemini", "antigravity")
		if err := os.MkdirAll(configDir, 0o700); err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}
		settingsPath := filepath.Join(configDir, "settings.json")
		if err := os.WriteFile(settingsPath, []byte(`{"model": "Gemini 1.5 Pro"}`), 0o600); err != nil {
			t.Fatalf("failed to write settings: %v", err)
		}

		if model := transcript.GetAntigravityModel(nil); model != "gemini-1.5-pro" {
			t.Errorf("expected gemini-1.5-pro, got %q", model)
		}
	})

	t.Run("cli settings model precedence", func(t *testing.T) {
		cliConfigDir := filepath.Join(tmpDir, ".gemini", "antigravity-cli")
		if err := os.MkdirAll(cliConfigDir, 0o700); err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}
		cliSettingsPath := filepath.Join(cliConfigDir, "settings.json")
		if err := os.WriteFile(cliSettingsPath, []byte(`{"model": "Gemini 3.5 Flash (Medium)"}`), 0o600); err != nil {
			t.Fatalf("failed to write settings: %v", err)
		}

		if model := transcript.GetAntigravityModel(nil); model != "gemini-3.5-flash" {
			t.Errorf("expected gemini-3.5-flash, got %q", model)
		}
	})
}
