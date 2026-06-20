package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ttHooks = map[string][]any{
	"UserPromptSubmit": {
		map[string]any{
			"_owner": "tt",
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": "tt record prompt",
				},
			},
		},
	},
	"Stop": {
		map[string]any{
			"_owner": "tt",
			"hooks": []any{
				map[string]any{"type": "command", "command": "tt record response"},
			},
		},
	},
}

func SetupClaudeCode() error {
	updater := func(settings map[string]any) (map[string]any, error) {
		return updateSection(settings, "hooks", ttHooks), nil
	}
	return setupToolHooks(filepath.Join(".claude", "settings.json"), updater)
}

func SetupAntigravity() error {
	updater := func(settings map[string]any) (map[string]any, error) {
		targetHooks := map[string][]any{
			"PreInvocation": {
				map[string]any{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record prompt --tool antigravity",
				},
			},
			"Stop": {
				map[string]any{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record response --tool antigravity",
				},
			},
		}
		return updateSection(settings, "tt", targetHooks), nil
	}
	return setupToolHooks(filepath.Join(".gemini", "config", "hooks.json"), updater)
}

func SetupCodex() error {
	updater := func(settings map[string]any) (map[string]any, error) {
		targetHooks := map[string][]any{
			"UserPromptSubmit": {
				map[string]any{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record prompt --tool codex",
				},
			},
			"Stop": {
				map[string]any{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record response --tool codex",
				},
			},
		}
		return updateSection(settings, "hooks", targetHooks), nil
	}
	return setupToolHooks(filepath.Join(".codex", "hooks.json"), updater)
}

func SetupCopilot() error {
	updater := func(settings map[string]any) (map[string]any, error) {
		settings["version"] = 1
		targetHooks := map[string][]any{
			"userPromptSubmitted": {
				map[string]any{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record prompt --tool copilot-cli",
				},
			},
			"agentStop": {
				map[string]any{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record response --tool copilot-cli",
				},
			},
		}
		return updateSection(settings, "hooks", targetHooks), nil
	}
	return setupToolHooks(filepath.Join(".copilot", "hooks", "tt.json"), updater)
}

func setupToolHooks(subPath string, updater func(map[string]any) (map[string]any, error)) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return mergeHooksFile(filepath.Join(home, subPath), updater)
}

func updateSection(settings map[string]any, sectionName string, targetHooks map[string][]any) map[string]any {
	section, _ := settings[sectionName].(map[string]any)
	if section == nil {
		section = map[string]any{}
	}
	for event, newEntries := range targetHooks {
		existing, _ := section[event].([]any)
		section[event] = mergeHookEntries(existing, newEntries)
	}
	settings[sectionName] = section
	return settings
}

func mergeHooksFile(configPath string, updater func(map[string]any) (map[string]any, error)) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	var m map[string]any
	data, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		m = map[string]any{}
	} else if err != nil {
		return err
	} else {
		if len(data) == 0 {
			m = map[string]any{}
		} else {
			if err := json.Unmarshal(data, &m); err != nil {
				return fmt.Errorf("config is corrupt: %w", err)
			}
		}
	}

	updated, err := updater(m)
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, out, 0o600)
}

// mergeHookEntries filters out any existing hook entries with _owner == "tt",
// and appends the new entries to the remaining ones.
func mergeHookEntries(existing []any, newEntries []any) []any {
	var filtered []any
	for _, e := range existing {
		if em, ok := e.(map[string]any); ok && em["_owner"] == "tt" {
			continue
		}
		filtered = append(filtered, e)
	}
	return append(filtered, newEntries...)
}

func isDirActive(dirName string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	path := filepath.Join(home, dirName)
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// IsClaudeCodeActive checks if the ~/.claude directory exists.
func IsClaudeCodeActive() bool {
	return isDirActive(".claude")
}

// IsCopilotActive checks if the ~/.copilot directory exists.
func IsCopilotActive() bool {
	return isDirActive(".copilot")
}

// IsAntigravityActive checks if the ~/.gemini directory exists.
func IsAntigravityActive() bool {
	return isDirActive(".gemini")
}

// IsCodexActive checks if the ~/.codex directory exists.
func IsCodexActive() bool {
	return isDirActive(".codex")
}
