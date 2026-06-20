package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestSetupCmd_Antigravity(t *testing.T) {
	home := setupHome(t)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset flags to avoid pollution
	setupCmd.Flags().Set("claude-code", "false")
	setupCmd.Flags().Set("copilot", "false")
	setupCmd.Flags().Set("antigravity", "false")
	setupCmd.Flags().Set("codex", "false")

	rootCmd.SetArgs([]string{"setup", "--antigravity"})
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("rootCmd Execute failed: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expectedMsg := "Google Antigravity hooks configured in ~/.gemini/config/hooks.json"
	if !strings.Contains(output, expectedMsg) {
		t.Errorf("output = %q, want message containing %q", output, expectedMsg)
	}

	// Verify file was actually created in temp home
	configPath := filepath.Join(home, ".gemini", "config", "hooks.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("expected hooks file at %s, but stat failed: %v", configPath, err)
	}
}

func TestSetupCmd_Codex(t *testing.T) {
	home := setupHome(t)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset flags to avoid pollution
	setupCmd.Flags().Set("claude-code", "false")
	setupCmd.Flags().Set("copilot", "false")
	setupCmd.Flags().Set("antigravity", "false")
	setupCmd.Flags().Set("codex", "false")

	rootCmd.SetArgs([]string{"setup", "--codex"})
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("rootCmd Execute failed: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expectedMsg := "OpenAI Codex hooks configured in ~/.codex/hooks.json"
	if !strings.Contains(output, expectedMsg) {
		t.Errorf("output = %q, want message containing %q", output, expectedMsg)
	}

	// Verify file was actually created in temp home
	configPath := filepath.Join(home, ".codex", "hooks.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("expected hooks file at %s, but stat failed: %v", configPath, err)
	}
}

func TestSetupCmd_Copilot(t *testing.T) {
	home := setupHome(t)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset flags to avoid pollution
	setupCmd.Flags().Set("claude-code", "false")
	setupCmd.Flags().Set("copilot", "false")
	setupCmd.Flags().Set("antigravity", "false")
	setupCmd.Flags().Set("codex", "false")

	rootCmd.SetArgs([]string{"setup", "--copilot"})
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("rootCmd Execute failed: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expectedMsg := "GitHub Copilot CLI hooks configured in ~/.copilot/hooks/tt.json"
	if !strings.Contains(output, expectedMsg) {
		t.Errorf("output = %q, want message containing %q", output, expectedMsg)
	}

	// Verify file was actually created in temp home
	configPath := filepath.Join(home, ".copilot", "hooks", "tt.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("expected hooks file at %s, but stat failed: %v", configPath, err)
	}
}

func TestSetupCmd_MultiTool(t *testing.T) {
	home := setupHome(t)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset flags to avoid pollution
	setupCmd.Flags().Set("claude-code", "false")
	setupCmd.Flags().Set("copilot", "false")
	setupCmd.Flags().Set("antigravity", "false")
	setupCmd.Flags().Set("codex", "false")

	rootCmd.SetArgs([]string{"setup", "--claude-code", "--copilot"})
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("rootCmd Execute failed: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expectedMsg1 := "Claude Code hooks configured in ~/.claude/settings.json"
	expectedMsg2 := "GitHub Copilot CLI hooks configured in ~/.copilot/hooks/tt.json"
	if !strings.Contains(output, expectedMsg1) {
		t.Errorf("output = %q, want message containing %q", output, expectedMsg1)
	}
	if !strings.Contains(output, expectedMsg2) {
		t.Errorf("output = %q, want message containing %q", output, expectedMsg2)
	}

	// Verify files were actually created in temp home
	claudePath := filepath.Join(home, ".claude", "settings.json")
	if _, err := os.Stat(claudePath); err != nil {
		t.Errorf("expected hooks file at %s, but stat failed: %v", claudePath, err)
	}
	copilotPath := filepath.Join(home, ".copilot", "hooks", "tt.json")
	if _, err := os.Stat(copilotPath); err != nil {
		t.Errorf("expected hooks file at %s, but stat failed: %v", copilotPath, err)
	}
}

func TestSetupCmd_AutoDetect(t *testing.T) {
	home := setupHome(t)

	// Create directories to detect
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatalf("failed to create .claude: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".gemini"), 0o755); err != nil {
		t.Fatalf("failed to create .gemini: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset flags to avoid pollution
	setupCmd.Flags().Set("claude-code", "false")
	setupCmd.Flags().Set("copilot", "false")
	setupCmd.Flags().Set("antigravity", "false")
	setupCmd.Flags().Set("codex", "false")

	rootCmd.SetArgs([]string{"setup"})
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("rootCmd Execute failed: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expectedMsg1 := "Claude Code hooks configured in ~/.claude/settings.json"
	expectedMsg2 := "Google Antigravity hooks configured in ~/.gemini/config/hooks.json"
	if !strings.Contains(output, expectedMsg1) {
		t.Errorf("output = %q, want message containing %q", output, expectedMsg1)
	}
	if !strings.Contains(output, expectedMsg2) {
		t.Errorf("output = %q, want message containing %q", output, expectedMsg2)
	}

	// Verify hooks files were written
	claudePath := filepath.Join(home, ".claude", "settings.json")
	if _, err := os.Stat(claudePath); err != nil {
		t.Errorf("expected hooks file at %s, but stat failed: %v", claudePath, err)
	}
	geminiPath := filepath.Join(home, ".gemini", "config", "hooks.json")
	if _, err := os.Stat(geminiPath); err != nil {
		t.Errorf("expected hooks file at %s, but stat failed: %v", geminiPath, err)
	}
}

func TestSetupCmd_NoToolsDetected(t *testing.T) {
	_ = setupHome(t) // empty home

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset flags to avoid pollution
	setupCmd.Flags().Set("claude-code", "false")
	setupCmd.Flags().Set("copilot", "false")
	setupCmd.Flags().Set("antigravity", "false")
	setupCmd.Flags().Set("codex", "false")

	rootCmd.SetArgs([]string{"setup"})
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("rootCmd Execute failed: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expectedMsg := "No supported AI tools detected..."
	if !strings.Contains(output, expectedMsg) {
		t.Errorf("output = %q, want message containing %q", output, expectedMsg)
	}
}


