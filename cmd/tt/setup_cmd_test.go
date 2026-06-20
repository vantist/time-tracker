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

