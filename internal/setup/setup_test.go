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

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatal("hooks key missing or wrong type")
	}
	for _, event := range []string{"UserPromptSubmit", "Stop"} {
		if _, ok := hooks[event]; !ok {
			t.Errorf("hooks.%s missing", event)
		}
	}
}

// TestSetupClaudeCode_HookCommand: UserPromptSubmit hook command is exactly "tt record prompt",
// no bash shell substitution syntax.
func TestSetupClaudeCode_HookCommand(t *testing.T) {
	home := setupHome(t)

	if err := setup.SetupClaudeCode(); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	hooks := settings["hooks"].(map[string]any)
	entries := hooks["UserPromptSubmit"].([]any)
	var cmd string
	for _, e := range entries {
		em := e.(map[string]any)
		hs := em["hooks"].([]any)
		for _, h := range hs {
			hm := h.(map[string]any)
			if c, ok := hm["command"].(string); ok {
				cmd = c
				break
			}
		}
	}

	if cmd == "" {
		t.Fatal("UserPromptSubmit hook command is empty")
	}
	if cmd != "tt record prompt" {
		t.Errorf("hook command = %q, want %q", cmd, "tt record prompt")
	}
	for _, banned := range []string{"$PPID", "$(", "date", "ps ", "awk"} {
		if strings.Contains(cmd, banned) {
			t.Errorf("hook command contains bash-only syntax %q: %s", banned, cmd)
		}
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
	var settings map[string]any
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]any)
	for _, event := range []string{"UserPromptSubmit", "Stop"} {
		entries, _ := hooks[event].([]any)
		ttCount := 0
		for _, e := range entries {
			em, _ := e.(map[string]any)
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
	old := map[string]any{
		"hooks": map[string]any{
			"UserPromptSubmit": []any{
				map[string]any{
					"_owner": "tt",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "tt record prompt --old",
						},
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
	var settings map[string]any
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]any)
	entries, _ := hooks["UserPromptSubmit"].([]any)
	ttCount := 0
	for _, e := range entries {
		em, _ := e.(map[string]any)
		if em["_owner"] == "tt" {
			ttCount++
			// should be the new version, not the old command
			hs, _ := em["hooks"].([]any)
			for _, h := range hs {
				hm, _ := h.(map[string]any)
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
	existing := map[string]any{
		"hooks": map[string]any{
			"UserPromptSubmit": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "user-custom-hook",
						},
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
	var settings map[string]any
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]any)
	entries, _ := hooks["UserPromptSubmit"].([]any)
	foundUser := false
	for _, e := range entries {
		em, _ := e.(map[string]any)
		if em["_owner"] != "tt" {
			hs, _ := em["hooks"].([]any)
			for _, h := range hs {
				hm, _ := h.(map[string]any)
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
	existing := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "caveman-hook",
						},
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
	var settings map[string]any
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]any)
	if _, ok := hooks["PreToolUse"]; !ok {
		t.Error("existing PreToolUse hook was removed")
	}
	if _, ok := hooks["UserPromptSubmit"]; !ok {
		t.Error("tt UserPromptSubmit hook not added")
	}
}

func TestMergeHooksFile_CreatesDirAndFileWithCorrectPermissions(t *testing.T) {
	tmp := t.TempDir()
	subDir := filepath.Join(tmp, "nested_dir")
	configPath := filepath.Join(subDir, "config.json")

	updater := func(m map[string]any) (map[string]any, error) {
		m["key"] = "value"
		return m, nil
	}

	err := setup.MergeHooksFile(configPath, updater)
	if err != nil {
		t.Fatalf("MergeHooksFile failed: %v", err)
	}

	// 1. Check directory permissions
	dirInfo, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("failed to stat subDir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("subDir permissions = %o, want %o", perm, 0o700)
	}

	// 2. Check file permissions
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat configPath: %v", err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0o600 {
		t.Errorf("configPath permissions = %o, want %o", perm, 0o600)
	}

	// 3. Check content
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read configPath: %v", err)
	}
	var res map[string]any
	if err := json.Unmarshal(data, &res); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if val := res["key"]; val != "value" {
		t.Errorf("config key = %v, want 'value'", val)
	}
}

func TestMergeHooksFile_HandlesCorruptJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.json")

	if err := os.WriteFile(configPath, []byte("{invalid json"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	updater := func(m map[string]any) (map[string]any, error) {
		return m, nil
	}

	err := setup.MergeHooksFile(configPath, updater)
	if err == nil {
		t.Error("expected error for corrupt JSON, got nil")
	}
}

func TestSetupAntigravity(t *testing.T) {
	home := setupHome(t)

	// 1. Fresh installation
	if err := setup.SetupAntigravity(); err != nil {
		t.Fatalf("SetupAntigravity: %v", err)
	}

	configPath := filepath.Join(home, ".gemini", "config", "hooks.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("hooks.json not created: %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal hooks.json: %v", err)
	}

	ttVal, ok := config["tt"].(map[string]any)
	if !ok {
		t.Fatal("tt key missing or wrong type")
	}

	preInv, ok := ttVal["PreInvocation"].([]any)
	if !ok || len(preInv) != 1 {
		t.Fatalf("PreInvocation missing or invalid size")
	}
	preMap := preInv[0].(map[string]any)
	if preMap["_owner"] != "tt" || preMap["command"] != "tt record prompt --tool antigravity" || preMap["type"] != "command" {
		t.Errorf("PreInvocation hook values unexpected: %v", preMap)
	}

	stop, ok := ttVal["Stop"].([]any)
	if !ok || len(stop) != 1 {
		t.Fatalf("Stop missing or invalid size")
	}
	stopMap := stop[0].(map[string]any)
	if stopMap["_owner"] != "tt" || stopMap["command"] != "tt record response --tool antigravity" || stopMap["type"] != "command" {
		t.Errorf("Stop hook values unexpected: %v", stopMap)
	}

	// 2. Idempotence: running again yields exactly one entry
	if err := setup.SetupAntigravity(); err != nil {
		t.Fatalf("SetupAntigravity second call: %v", err)
	}
	data, _ = os.ReadFile(configPath)
	json.Unmarshal(data, &config)
	ttVal = config["tt"].(map[string]any)
	if len(ttVal["PreInvocation"].([]any)) != 1 || len(ttVal["Stop"].([]any)) != 1 {
		t.Errorf("expected 1 hook entry per event, got PreInvocation count: %d, Stop count: %d",
			len(ttVal["PreInvocation"].([]any)), len(ttVal["Stop"].([]any)))
	}

	// 3. Preserve User hooks
	home = setupHome(t)
	configPath = filepath.Join(home, ".gemini", "config", "hooks.json")
	geminiDir := filepath.Join(home, ".gemini", "config")
	os.MkdirAll(geminiDir, 0o700)
	userConfig := map[string]any{
		"tt": map[string]any{
			"PreInvocation": []any{
				map[string]any{
					"type":    "command",
					"command": "user-pre-hook",
				},
			},
		},
	}
	uData, _ := json.Marshal(userConfig)
	os.WriteFile(configPath, uData, 0o600)

	if err := setup.SetupAntigravity(); err != nil {
		t.Fatalf("SetupAntigravity with user hooks: %v", err)
	}
	data, _ = os.ReadFile(configPath)
	json.Unmarshal(data, &config)
	ttVal = config["tt"].(map[string]any)
	preHooks := ttVal["PreInvocation"].([]any)
	if len(preHooks) != 2 {
		t.Fatalf("expected 2 prehooks, got %d", len(preHooks))
	}
	foundUser, foundTT := false, false
	for _, e := range preHooks {
		em := e.(map[string]any)
		if em["_owner"] == "tt" {
			foundTT = true
		} else if em["command"] == "user-pre-hook" {
			foundUser = true
		}
	}
	if !foundUser || !foundTT {
		t.Errorf("User hook or TT hook missing: user=%v, tt=%v", foundUser, foundTT)
	}
}

func TestSetupCodex(t *testing.T) {
	home := setupHome(t)

	// 1. Fresh installation
	if err := setup.SetupCodex(); err != nil {
		t.Fatalf("SetupCodex: %v", err)
	}

	configPath := filepath.Join(home, ".codex", "hooks.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("hooks.json not created: %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal hooks.json: %v", err)
	}

	hooksVal, ok := config["hooks"].(map[string]any)
	if !ok {
		t.Fatal("hooks key missing or wrong type")
	}

	promptSub, ok := hooksVal["UserPromptSubmit"].([]any)
	if !ok || len(promptSub) != 1 {
		t.Fatalf("UserPromptSubmit missing or invalid size")
	}
	promptMap := promptSub[0].(map[string]any)
	if promptMap["_owner"] != "tt" || promptMap["command"] != "tt record prompt --tool codex" || promptMap["type"] != "command" {
		t.Errorf("UserPromptSubmit hook values unexpected: %v", promptMap)
	}

	stop, ok := hooksVal["Stop"].([]any)
	if !ok || len(stop) != 1 {
		t.Fatalf("Stop missing or invalid size")
	}
	stopMap := stop[0].(map[string]any)
	if stopMap["_owner"] != "tt" || stopMap["command"] != "tt record response --tool codex" || stopMap["type"] != "command" {
		t.Errorf("Stop hook values unexpected: %v", stopMap)
	}

	// 2. Idempotence: running again yields exactly one entry
	if err := setup.SetupCodex(); err != nil {
		t.Fatalf("SetupCodex second call: %v", err)
	}
	data, _ = os.ReadFile(configPath)
	json.Unmarshal(data, &config)
	hooksVal = config["hooks"].(map[string]any)
	if len(hooksVal["UserPromptSubmit"].([]any)) != 1 || len(hooksVal["Stop"].([]any)) != 1 {
		t.Errorf("expected 1 hook entry per event, got UserPromptSubmit count: %d, Stop count: %d",
			len(hooksVal["UserPromptSubmit"].([]any)), len(hooksVal["Stop"].([]any)))
	}

	// 3. Preserve User hooks
	home = setupHome(t)
	configPath = filepath.Join(home, ".codex", "hooks.json")
	codexDir := filepath.Join(home, ".codex")
	os.MkdirAll(codexDir, 0o700)
	userConfig := map[string]any{
		"hooks": map[string]any{
			"UserPromptSubmit": []any{
				map[string]any{
					"type":    "command",
					"command": "user-prompt-hook",
				},
			},
		},
	}
	uData, _ := json.Marshal(userConfig)
	os.WriteFile(configPath, uData, 0o600)

	if err := setup.SetupCodex(); err != nil {
		t.Fatalf("SetupCodex with user hooks: %v", err)
	}
	data, _ = os.ReadFile(configPath)
	json.Unmarshal(data, &config)
	hooksVal = config["hooks"].(map[string]any)
	promptHooks := hooksVal["UserPromptSubmit"].([]any)
	if len(promptHooks) != 2 {
		t.Fatalf("expected 2 promptHooks, got %d", len(promptHooks))
	}
	foundUser, foundTT := false, false
	for _, e := range promptHooks {
		em := e.(map[string]any)
		if em["_owner"] == "tt" {
			foundTT = true
		} else if em["command"] == "user-prompt-hook" {
			foundUser = true
		}
	}
	if !foundUser || !foundTT {
		t.Errorf("User hook or TT hook missing: user=%v, tt=%v", foundUser, foundTT)
	}
}

func TestSetupCopilotFresh(t *testing.T) {
	home := setupHome(t)

	if err := setup.SetupCopilot(); err != nil {
		t.Fatalf("SetupCopilot: %v", err)
	}

	configPath := filepath.Join(home, ".copilot", "hooks", "tt.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("tt.json not created: %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	ver, ok := config["version"].(float64)
	if !ok || ver != 1 {
		t.Errorf("version = %v, want 1", config["version"])
	}

	hooks, ok := config["hooks"].(map[string]any)
	if !ok {
		t.Fatal("hooks key missing or wrong type")
	}

	promptHooks, ok := hooks["userPromptSubmitted"].([]any)
	if !ok || len(promptHooks) != 1 {
		t.Fatalf("userPromptSubmitted missing or wrong size")
	}
	pMap := promptHooks[0].(map[string]any)
	if pMap["_owner"] != "tt" || pMap["type"] != "command" || pMap["command"] != "tt record prompt --tool copilot-cli" {
		t.Errorf("unexpected userPromptSubmitted hook: %v", pMap)
	}

	stopHooks, ok := hooks["agentStop"].([]any)
	if !ok || len(stopHooks) != 1 {
		t.Fatalf("agentStop missing or wrong size")
	}
	sMap := stopHooks[0].(map[string]any)
	if sMap["_owner"] != "tt" || sMap["type"] != "command" || sMap["command"] != "tt record response --tool copilot-cli" {
		t.Errorf("unexpected agentStop hook: %v", sMap)
	}

	// 1. Check directory permissions
	dirInfo, err := os.Stat(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("failed to stat dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("dir permissions = %o, want %o", perm, 0o700)
	}

	// 2. Check file permissions
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0o600 {
		t.Errorf("file permissions = %o, want %o", perm, 0o600)
	}
}

func TestSetupCopilotIdempotent(t *testing.T) {
	home := setupHome(t)

	if err := setup.SetupCopilot(); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := setup.SetupCopilot(); err != nil {
		t.Fatalf("second call: %v", err)
	}

	configPath := filepath.Join(home, ".copilot", "hooks", "tt.json")
	data, _ := os.ReadFile(configPath)
	var config map[string]any
	json.Unmarshal(data, &config)

	hooks := config["hooks"].(map[string]any)
	for _, event := range []string{"userPromptSubmitted", "agentStop"} {
		entries, _ := hooks[event].([]any)
		ttCount := 0
		for _, e := range entries {
			em, _ := e.(map[string]any)
			if em["_owner"] == "tt" {
				ttCount++
			}
		}
		if ttCount != 1 {
			t.Errorf("event %s: want 1 tt entry, got %d", event, ttCount)
		}
	}
}

func TestSetupCopilotPreservesUserHooks(t *testing.T) {
	home := setupHome(t)
	copilotDir := filepath.Join(home, ".copilot", "hooks")
	os.MkdirAll(copilotDir, 0o700)

	existing := map[string]any{
		"version": float64(1),
		"hooks": map[string]any{
			"userPromptSubmitted": []any{
				map[string]any{
					"type":    "command",
					"command": "user-custom-copilot-hook",
				},
			},
		},
	}
	data, _ := json.Marshal(existing)
	os.WriteFile(filepath.Join(copilotDir, "tt.json"), data, 0o600)

	if err := setup.SetupCopilot(); err != nil {
		t.Fatalf("SetupCopilot: %v", err)
	}

	data, _ = os.ReadFile(filepath.Join(copilotDir, "tt.json"))
	var config map[string]any
	json.Unmarshal(data, &config)

	hooks := config["hooks"].(map[string]any)
	entries, _ := hooks["userPromptSubmitted"].([]any)
	foundUser, foundTT := false, false
	for _, e := range entries {
		em, _ := e.(map[string]any)
		if em["_owner"] == "tt" {
			if em["command"] == "tt record prompt --tool copilot-cli" {
				foundTT = true
			}
		} else if em["command"] == "user-custom-copilot-hook" {
			foundUser = true
		}
	}
	if !foundUser || !foundTT {
		t.Errorf("User hook or TT hook missing: user=%v, tt=%v", foundUser, foundTT)
	}
}

func TestSetupCopilotReplacesOldVersion(t *testing.T) {
	home := setupHome(t)
	copilotDir := filepath.Join(home, ".copilot", "hooks")
	os.MkdirAll(copilotDir, 0o700)

	old := map[string]any{
		"version": float64(1),
		"hooks": map[string]any{
			"userPromptSubmitted": []any{
				map[string]any{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record prompt --old-args",
				},
			},
		},
	}
	data, _ := json.Marshal(old)
	os.WriteFile(filepath.Join(copilotDir, "tt.json"), data, 0o600)

	if err := setup.SetupCopilot(); err != nil {
		t.Fatalf("SetupCopilot: %v", err)
	}

	data, _ = os.ReadFile(filepath.Join(copilotDir, "tt.json"))
	var config map[string]any
	json.Unmarshal(data, &config)

	hooks := config["hooks"].(map[string]any)
	entries, _ := hooks["userPromptSubmitted"].([]any)
	ttCount := 0
	for _, e := range entries {
		em, _ := e.(map[string]any)
		if em["_owner"] == "tt" {
			ttCount++
			if em["command"] == "tt record prompt --old-args" {
				t.Error("old hook command still present after setup")
			}
		}
	}
	if ttCount != 1 {
		t.Errorf("want 1 tt entry, got %d", ttCount)
	}
}

func TestIsActive(t *testing.T) {
	tests := []struct {
		name     string
		dirName  string
		activeFn func() bool
	}{
		{"ClaudeCode", ".claude", setup.IsClaudeCodeActive},
		{"Copilot", ".copilot", setup.IsCopilotActive},
		{"Antigravity", ".gemini", setup.IsAntigravityActive},
		{"Codex", ".codex", setup.IsCodexActive},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := setupHome(t)
			if tc.activeFn() {
				t.Errorf("expected active to be false for non-existent dir")
			}

			dirPath := filepath.Join(home, tc.dirName)
			if err := os.MkdirAll(dirPath, 0o755); err != nil {
				t.Fatalf("failed to create dir: %v", err)
			}
			if !tc.activeFn() {
				t.Errorf("expected active to be true for existing dir")
			}

			// Cleanup and test file case
			if err := os.RemoveAll(dirPath); err != nil {
				t.Fatalf("failed to remove dir: %v", err)
			}
			if err := os.WriteFile(dirPath, []byte("plain file"), 0o600); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}
			if tc.activeFn() {
				t.Errorf("expected active to be false when path is a file")
			}
		})
	}
}
