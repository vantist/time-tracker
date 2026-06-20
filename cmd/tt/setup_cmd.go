package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/tt/internal/setup"
)

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().Bool("claude-code", false, "Set up Claude Code hooks")
	setupCmd.Flags().Bool("copilot", false, "Set up GitHub Copilot CLI hooks")
	setupCmd.Flags().Bool("antigravity", false, "Set up Google Antigravity hooks")
	setupCmd.Flags().Bool("codex", false, "Set up OpenAI Codex hooks")
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure AI tool hooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		claudeCode, _ := cmd.Flags().GetBool("claude-code")
		copilot, _ := cmd.Flags().GetBool("copilot")
		antigravity, _ := cmd.Flags().GetBool("antigravity")
		codex, _ := cmd.Flags().GetBool("codex")

		if claudeCode {
			if err := setup.SetupClaudeCode(); err != nil {
				return err
			}
			fmt.Println("Claude Code hooks configured in ~/.claude/settings.json")
			return nil
		}

		if copilot {
			if err := setup.SetupCopilot(); err != nil {
				return err
			}
			fmt.Println("GitHub Copilot CLI hooks configured in ~/.copilot/hooks/tt.json")
			return nil
		}

		if antigravity {
			if err := setup.SetupAntigravity(); err != nil {
				return err
			}
			fmt.Println("Google Antigravity hooks configured in ~/.gemini/config/hooks.json")
			return nil
		}

		if codex {
			if err := setup.SetupCodex(); err != nil {
				return err
			}
			fmt.Println("OpenAI Codex hooks configured in ~/.codex/hooks.json")
			return nil
		}

		return cmd.Help()
	},
}
