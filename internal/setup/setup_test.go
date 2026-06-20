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

func TestMergeHooksFile_CreatesDirAndFileWithCorrectPermissions(t *testing.T) {
	tmp := t.TempDir()
	subDir := filepath.Join(tmp, "nested_dir")
	configPath := filepath.Join(subDir, "config.json")

	updater := func(m map[string]interface{}) (map[string]interface{}, error) {
		m["key"] = "value"
		return m, nil
	}

	err := setup.MergeHooksFile(configPath, "tt", updater)
	if err != nil {
		t.Fatalf("MergeHooksFile failed: %v", err)
	}

	// 1. Check directory permissions
	dirInfo, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("failed to stat subDir: %v", err)
	}
	// On Unix/Mac, check the mode permissions. Mask with 0o777 to only keep standard permission bits.
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
	var res map[string]interface{}
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

	// Pre-populate with corrupt JSON
	if err := os.WriteFile(configPath, []byte("{invalid json"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	updater := func(m map[string]interface{}) (map[string]interface{}, error) {
		return m, nil
	}

	err := setup.MergeHooksFile(configPath, "tt", updater)
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

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal hooks.json: %v", err)
	}

	ttVal, ok := config["tt"].(map[string]interface{})
	if !ok {
		t.Fatal("tt key missing or wrong type")
	}

	preInv, ok := ttVal["PreInvocation"].([]interface{})
	if !ok || len(preInv) != 1 {
		t.Fatalf("PreInvocation missing or invalid size")
	}
	preMap := preInv[0].(map[string]interface{})
	if preMap["_owner"] != "tt" || preMap["command"] != "tt record prompt --tool antigravity" || preMap["type"] != "command" {
		t.Errorf("PreInvocation hook values unexpected: %v", preMap)
	}

	stop, ok := ttVal["Stop"].([]interface{})
	if !ok || len(stop) != 1 {
		t.Fatalf("Stop missing or invalid size")
	}
	stopMap := stop[0].(map[string]interface{})
	if stopMap["_owner"] != "tt" || stopMap["command"] != "tt record response --tool antigravity" || stopMap["type"] != "command" {
		t.Errorf("Stop hook values unexpected: %v", stopMap)
	}

	// 2. Idempotence: running again yields exactly one entry
	if err := setup.SetupAntigravity(); err != nil {
		t.Fatalf("SetupAntigravity second call: %v", err)
	}
	data, _ = os.ReadFile(configPath)
	json.Unmarshal(data, &config)
	ttVal = config["tt"].(map[string]interface{})
	if len(ttVal["PreInvocation"].([]interface{})) != 1 || len(ttVal["Stop"].([]interface{})) != 1 {
		t.Errorf("expected 1 hook entry per event, got PreInvocation count: %d, Stop count: %d",
			len(ttVal["PreInvocation"].([]interface{})), len(ttVal["Stop"].([]interface{})))
	}

	// 3. Preserve User hooks
	// Reset home with some existing user-owned hooks
	home = setupHome(t)
	configPath = filepath.Join(home, ".gemini", "config", "hooks.json")
	geminiDir := filepath.Join(home, ".gemini", "config")
	os.MkdirAll(geminiDir, 0o700)
	userConfig := map[string]interface{}{
		"tt": map[string]interface{}{
			"PreInvocation": []interface{}{
				map[string]interface{}{
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
	ttVal = config["tt"].(map[string]interface{})
	preHooks := ttVal["PreInvocation"].([]interface{})
	if len(preHooks) != 2 {
		t.Fatalf("expected 2 prehooks, got %d", len(preHooks))
	}
	foundUser, foundTT := false, false
	for _, e := range preHooks {
		em := e.(map[string]interface{})
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

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal hooks.json: %v", err)
	}

	hooksVal, ok := config["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks key missing or wrong type")
	}

	promptSub, ok := hooksVal["UserPromptSubmit"].([]interface{})
	if !ok || len(promptSub) != 1 {
		t.Fatalf("UserPromptSubmit missing or invalid size")
	}
	promptMap := promptSub[0].(map[string]interface{})
	if promptMap["_owner"] != "tt" || promptMap["command"] != "tt record prompt --tool codex" || promptMap["type"] != "command" {
		t.Errorf("UserPromptSubmit hook values unexpected: %v", promptMap)
	}

	stop, ok := hooksVal["Stop"].([]interface{})
	if !ok || len(stop) != 1 {
		t.Fatalf("Stop missing or invalid size")
	}
	stopMap := stop[0].(map[string]interface{})
	if stopMap["_owner"] != "tt" || stopMap["command"] != "tt record response --tool codex" || stopMap["type"] != "command" {
		t.Errorf("Stop hook values unexpected: %v", stopMap)
	}

	// 2. Idempotence: running again yields exactly one entry
	if err := setup.SetupCodex(); err != nil {
		t.Fatalf("SetupCodex second call: %v", err)
	}
	data, _ = os.ReadFile(configPath)
	json.Unmarshal(data, &config)
	hooksVal = config["hooks"].(map[string]interface{})
	if len(hooksVal["UserPromptSubmit"].([]interface{})) != 1 || len(hooksVal["Stop"].([]interface{})) != 1 {
		t.Errorf("expected 1 hook entry per event, got UserPromptSubmit count: %d, Stop count: %d",
			len(hooksVal["UserPromptSubmit"].([]interface{})), len(hooksVal["Stop"].([]interface{})))
	}

	// 3. Preserve User hooks
	// Reset home with some existing user-owned hooks
	home = setupHome(t)
	configPath = filepath.Join(home, ".codex", "hooks.json")
	codexDir := filepath.Join(home, ".codex")
	os.MkdirAll(codexDir, 0o700)
	userConfig := map[string]interface{}{
		"hooks": map[string]interface{}{
			"UserPromptSubmit": []interface{}{
				map[string]interface{}{
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
	hooksVal = config["hooks"].(map[string]interface{})
	promptHooks := hooksVal["UserPromptSubmit"].([]interface{})
	if len(promptHooks) != 2 {
		t.Fatalf("expected 2 promptHooks, got %d", len(promptHooks))
	}
	foundUser, foundTT := false, false
	for _, e := range promptHooks {
		em := e.(map[string]interface{})
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


