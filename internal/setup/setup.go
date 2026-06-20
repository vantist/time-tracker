package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ttHooks = map[string]interface{}{
	"UserPromptSubmit": []interface{}{
		map[string]interface{}{
			"_owner": "tt",
			"hooks": []interface{}{
				map[string]interface{}{
					"type":    "command",
					"command": "tt record prompt",
				},
			},
		},
	},
	"Stop": []interface{}{
		map[string]interface{}{
			"_owner": "tt",
			"hooks": []interface{}{
				map[string]interface{}{"type": "command", "command": "tt record response"},
			},
		},
	},
}

func SetupClaudeCode() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	claudeDir := filepath.Join(home, ".claude")
	settingsPath := filepath.Join(claudeDir, "settings.json")

	updater := func(settings map[string]interface{}) (map[string]interface{}, error) {
		hooks, _ := settings["hooks"].(map[string]interface{})
		if hooks == nil {
			hooks = map[string]interface{}{}
		}
		for event, hookVal := range ttHooks {
			newEntries, _ := hookVal.([]interface{})
			existing, _ := hooks[event].([]interface{})
			var filtered []interface{}
			for _, e := range existing {
				em, _ := e.(map[string]interface{})
				if em["_owner"] != "tt" {
					filtered = append(filtered, e)
				}
			}
			hooks[event] = append(filtered, newEntries...)
		}
		settings["hooks"] = hooks
		return settings, nil
	}

	return mergeHooksFile(settingsPath, "tt", updater)
}

func mergeHooksFile(configPath string, defaultOwner string, updater func(map[string]interface{}) (map[string]interface{}, error)) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	var m map[string]interface{}
	data, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		m = map[string]interface{}{}
	} else if err != nil {
		return err
	} else {
		if len(data) == 0 {
			m = map[string]interface{}{}
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

func SetupAntigravity() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, ".gemini", "config", "hooks.json")

	updater := func(settings map[string]interface{}) (map[string]interface{}, error) {
		ttSection, _ := settings["tt"].(map[string]interface{})
		if ttSection == nil {
			ttSection = map[string]interface{}{}
		}

		targetHooks := map[string][]interface{}{
			"PreInvocation": {
				map[string]interface{}{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record prompt --tool antigravity",
				},
			},
			"Stop": {
				map[string]interface{}{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record response --tool antigravity",
				},
			},
		}

		for event, newEntries := range targetHooks {
			existing, _ := ttSection[event].([]interface{})
			var filtered []interface{}
			for _, e := range existing {
				em, _ := e.(map[string]interface{})
				if em["_owner"] != "tt" {
					filtered = append(filtered, e)
				}
			}
			ttSection[event] = append(filtered, newEntries...)
		}
		settings["tt"] = ttSection
		return settings, nil
	}

	return mergeHooksFile(configPath, "tt", updater)
}

func SetupCodex() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, ".codex", "hooks.json")

	updater := func(settings map[string]interface{}) (map[string]interface{}, error) {
		hooksSection, _ := settings["hooks"].(map[string]interface{})
		if hooksSection == nil {
			hooksSection = map[string]interface{}{}
		}

		targetHooks := map[string][]interface{}{
			"UserPromptSubmit": {
				map[string]interface{}{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record prompt --tool codex",
				},
			},
			"Stop": {
				map[string]interface{}{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record response --tool codex",
				},
			},
		}

		for event, newEntries := range targetHooks {
			existing, _ := hooksSection[event].([]interface{})
			var filtered []interface{}
			for _, e := range existing {
				em, _ := e.(map[string]interface{})
				if em["_owner"] != "tt" {
					filtered = append(filtered, e)
				}
			}
			hooksSection[event] = append(filtered, newEntries...)
		}
		settings["hooks"] = hooksSection
		return settings, nil
	}

	return mergeHooksFile(configPath, "tt", updater)
}


const CopilotInstructions = `To set up GitHub Copilot CLI hooks, add the following to ~/.copilot/settings.json:

{
  "hooks": {
    "userPromptSubmitted": "tt record prompt --tool copilot-cli",
    "agentStop": "tt record response --tool copilot-cli"
  }
}

Events:
  userPromptSubmitted  → tt record prompt --tool copilot-cli
  agentStop            → tt record response --tool copilot-cli

Note: Token data is not available in Copilot CLI hooks; token fields will be NULL.
`
