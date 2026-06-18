package setup_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/tt/internal/setup"
)

func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

// Task 8.1: SetupClaudeCode writes hooks when settings.json absent
func TestSetupClaudeCodeFresh(t *testing.T) {
	home := setupHome(t)

	if err := setup.SetupClaudeCode(); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks key missing or wrong type")
	}
	for _, event := range []string{"UserPromptSubmit", "Stop"} {
		if _, ok := hooks[event]; !ok {
			t.Errorf("hooks.%s missing", event)
		}
	}
}

// TestSetupClaudeCode_HookCommand: UserPromptSubmit hook command contains PROCESS_PID env var.
func TestSetupClaudeCode_HookCommand(t *testing.T) {
	home := setupHome(t)

	if err := setup.SetupClaudeCode(); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	hooks := settings["hooks"].(map[string]interface{})
	entries := hooks["UserPromptSubmit"].([]interface{})
	var cmd string
	for _, e := range entries {
		em := e.(map[string]interface{})
		hs := em["hooks"].([]interface{})
		for _, h := range hs {
			hm := h.(map[string]interface{})
			if c, ok := hm["command"].(string); ok {
				cmd = c
				break
			}
		}
	}

	if cmd == "" {
		t.Fatal("UserPromptSubmit hook command is empty")
	}
	if !strings.Contains(cmd, "PROCESS_PID") {
		t.Errorf("hook command %q does not contain PROCESS_PID", cmd)
	}
}

// idempotent-hook-setup: running twice yields exactly one tt entry per event
func TestSetupClaudeCode_IdempotentNoDuplicates(t *testing.T) {
	home := setupHome(t)

	if err := setup.SetupClaudeCode(); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := setup.SetupClaudeCode(); err != nil {
		t.Fatalf("second call: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]interface{})
	for _, event := range []string{"UserPromptSubmit", "Stop"} {
		entries, _ := hooks[event].([]interface{})
		ttCount := 0
		for _, e := range entries {
			em, _ := e.(map[string]interface{})
			if em["_owner"] == "tt" {
				ttCount++
			}
		}
		if ttCount != 1 {
			t.Errorf("event %s: want 1 tt entry, got %d", event, ttCount)
		}
	}
}

// idempotent-hook-setup: stale tt entry replaced by updated version
func TestSetupClaudeCode_ReplacesOldVersion(t *testing.T) {
	home := setupHome(t)
	claudeDir := filepath.Join(home, ".claude")
	os.MkdirAll(claudeDir, 0o755)

	// Pre-populate with an old tt hook (has _owner:"tt" but old command)
	old := map[string]interface{}{
		"hooks": map[string]interface{}{
			"UserPromptSubmit": []interface{}{
				map[string]interface{}{
					"_owner": "tt",
					"hooks": []interface{}{
						map[string]interface{}{"type": "command", "command": "tt record prompt --old"},
					},
				},
			},
		},
	}
	data, _ := json.Marshal(old)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644)

	if err := setup.SetupClaudeCode(); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	data, _ = os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]interface{})
	entries, _ := hooks["UserPromptSubmit"].([]interface{})
	ttCount := 0
	for _, e := range entries {
		em, _ := e.(map[string]interface{})
		if em["_owner"] == "tt" {
			ttCount++
			// should be the new version, not the old command
			hs, _ := em["hooks"].([]interface{})
			for _, h := range hs {
				hm, _ := h.(map[string]interface{})
				if hm["command"] == "tt record prompt --old" {
					t.Error("old hook command still present after setup")
				}
			}
		}
	}
	if ttCount != 1 {
		t.Errorf("want 1 tt entry after update, got %d", ttCount)
	}
}

// idempotent-hook-setup: user-owned entries unaffected
func TestSetupClaudeCode_PreservesUserHooks(t *testing.T) {
	home := setupHome(t)
	claudeDir := filepath.Join(home, ".claude")
	os.MkdirAll(claudeDir, 0o755)

	// Pre-populate with a user-owned hook (no _owner field)
	existing := map[string]interface{}{
		"hooks": map[string]interface{}{
			"UserPromptSubmit": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{"type": "command", "command": "user-custom-hook"},
					},
				},
			},
		},
	}
	data, _ := json.Marshal(existing)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644)

	if err := setup.SetupClaudeCode(); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	data, _ = os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]interface{})
	entries, _ := hooks["UserPromptSubmit"].([]interface{})
	foundUser := false
	for _, e := range entries {
		em, _ := e.(map[string]interface{})
		if em["_owner"] != "tt" {
			hs, _ := em["hooks"].([]interface{})
			for _, h := range hs {
				hm, _ := h.(map[string]interface{})
				if hm["command"] == "user-custom-hook" {
					foundUser = true
				}
			}
		}
	}
	if !foundUser {
		t.Error("user-owned hook was removed or modified")
	}
}

// Task 8.1: existing hooks not overwritten
func TestSetupClaudeCodePreservesExistingHooks(t *testing.T) {
	home := setupHome(t)

	// Pre-populate settings with an existing hook
	claudeDir := filepath.Join(home, ".claude")
	os.MkdirAll(claudeDir, 0o755)
	existing := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{"hooks": []interface{}{map[string]interface{}{"type": "command", "command": "caveman-hook"}}},
			},
		},
	}
	data, _ := json.Marshal(existing)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644)

	if err := setup.SetupClaudeCode(); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	data, _ = os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]interface{})
	if _, ok := hooks["PreToolUse"]; !ok {
		t.Error("existing PreToolUse hook was removed")
	}
	if _, ok := hooks["UserPromptSubmit"]; !ok {
		t.Error("tt UserPromptSubmit hook not added")
	}
}
