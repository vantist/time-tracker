package setup

import (
	"encoding/json"
	"errors"
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
					"command": `PROCESS_PID=$PPID PROCESS_START=$(( $(date +%s) - $(ps -p $PPID -o etime= | tr -d ' ' | awk -F'[:-]' '{n=NF;s=0;if(n>=1)s+=$n;if(n>=2)s+=$(n-1)*60;if(n>=3)s+=$(n-2)*3600;if(n>=4)s+=$(n-3)*86400;print s}') )) tt record prompt`,
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
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return err
	}
	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Load existing settings
	var settings map[string]interface{}
	data, err := os.ReadFile(settingsPath)
	if errors.Is(err, os.ErrNotExist) {
		settings = map[string]interface{}{}
	} else if err != nil {
		return err
	} else {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = map[string]interface{}{}
		}
	}

	// Merge hooks
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

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, out, 0o644)
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
