package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/tt/internal/setup"
)

type toolInfo struct {
	flagName string
	desc     string
	isActive func() bool
	setup    func() error
	msg      string
}

var tools = []toolInfo{
	{
		flagName: "claude-code",
		desc:     "Set up Claude Code hooks",
		isActive: setup.IsClaudeCodeActive,
		setup:    setup.SetupClaudeCode,
		msg:      "Claude Code hooks configured in ~/.claude/settings.json",
	},
	{
		flagName: "copilot",
		desc:     "Set up GitHub Copilot CLI hooks",
		isActive: setup.IsCopilotActive,
		setup:    setup.SetupCopilot,
		msg:      "GitHub Copilot CLI hooks configured in ~/.copilot/hooks/tt.json",
	},
	{
		flagName: "antigravity",
		desc:     "Set up Google Antigravity hooks",
		isActive: setup.IsAntigravityActive,
		setup:    setup.SetupAntigravity,
		msg:      "Google Antigravity hooks configured in ~/.gemini/config/hooks.json",
	},
	{
		flagName: "codex",
		desc:     "Set up OpenAI Codex hooks",
		isActive: setup.IsCodexActive,
		setup:    setup.SetupCodex,
		msg:      "OpenAI Codex hooks configured in ~/.codex/hooks.json",
	},
	{
		flagName: "opencode",
		desc:     "Set up OpenCode plugin",
		isActive: setup.IsOpenCodeActive,
		setup:    setup.SetupOpencode,
		msg:      "OpenCode plugin configured in ~/.config/opencode/plugins/tt-bridge.ts",
	},
	{
		flagName: "vscode-copilot",
		desc:     "Set up VS Code Copilot bridge",
		isActive: setup.IsVSCodeCopilotActive,
		setup:    setup.SetupVSCodeCopilot,
		msg:      "VS Code Copilot bridge installed",
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
	for _, t := range tools {
		setupCmd.Flags().Bool(t.flagName, false, t.desc)
	}
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure AI tool hooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		var selectedTools []toolInfo
		anyFlagSet := false

		for _, t := range tools {
			if val, _ := cmd.Flags().GetBool(t.flagName); val {
				selectedTools = append(selectedTools, t)
				anyFlagSet = true
			}
		}

		if !anyFlagSet {
			for _, t := range tools {
				if t.isActive() {
					selectedTools = append(selectedTools, t)
				}
			}
		}

	for _, t := range selectedTools {
		if err := t.setup(); err != nil {
			return err
		}
		// VS Code Copilot handles its own messaging
		if t.flagName != "vscode-copilot" {
			fmt.Println(t.msg)
		}
	}

		if !anyFlagSet && len(selectedTools) == 0 {
			fmt.Println("No supported AI tools detected...")
		}

		return nil
	},
}
