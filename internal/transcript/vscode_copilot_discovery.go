package transcript

import (
	"os"
	"path/filepath"
	"runtime"
)

// DiscoverWorkspaceStoragePaths returns all possible VS Code workspaceStorage paths
// for the current operating system and VS Code variants.
func DiscoverWorkspaceStoragePaths() []string {
	var paths []string

	home, err := os.UserHomeDir()
	if err != nil {
		return paths
	}

	switch runtime.GOOS {
	case "darwin":
		base := filepath.Join(home, "Library", "Application Support")
		paths = append(paths, discoverVSCodePaths(base, "Code")...)
		paths = append(paths, discoverVSCodePaths(base, "Code - Insiders")...)
		paths = append(paths, discoverVSCodePaths(base, "Code - Exploration")...)
		paths = append(paths, discoverVSCodePaths(base, "VSCodium")...)
		paths = append(paths, discoverVSCodePaths(base, "Cursor")...)

	case "linux":
		base := filepath.Join(home, ".config")
		paths = append(paths, discoverVSCodePaths(base, "Code")...)
		paths = append(paths, discoverVSCodePaths(base, "Code - Insiders")...)
		paths = append(paths, discoverVSCodePaths(base, "Code - Exploration")...)
		paths = append(paths, discoverVSCodePaths(base, "VSCodium")...)
		paths = append(paths, discoverVSCodePaths(base, "Cursor")...)

	case "windows":
		base := os.Getenv("APPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Roaming")
		}
		paths = append(paths, discoverVSCodePaths(base, "Code")...)
		paths = append(paths, discoverVSCodePaths(base, "Code - Insiders")...)
		paths = append(paths, discoverVSCodePaths(base, "Code - Exploration")...)
		paths = append(paths, discoverVSCodePaths(base, "VSCodium")...)
		paths = append(paths, discoverVSCodePaths(base, "Cursor")...)
	}

	return paths
}

// discoverVSCodePaths finds GitHub.copilot-chat directories under a VS Code variant's workspaceStorage.
func discoverVSCodePaths(base string, variant string) []string {
	workspaceStorage := filepath.Join(base, variant, "User", "workspaceStorage")
	entries, err := os.ReadDir(workspaceStorage)
	if err != nil {
		return nil
	}

	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		chatDir := filepath.Join(workspaceStorage, entry.Name(), "GitHub.copilot-chat")
		if info, err := os.Stat(chatDir); err == nil && info.IsDir() {
			paths = append(paths, chatDir)
		}
	}

	return paths
}

// FindTranscriptFiles finds all transcript JSONL files in a workspaceStorage Copilot Chat directory.
func FindTranscriptFiles(chatDir string) []string {
	transcriptsDir := filepath.Join(chatDir, "transcripts")
	entries, err := os.ReadDir(transcriptsDir)
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jsonl" {
			files = append(files, filepath.Join(transcriptsDir, entry.Name()))
		}
	}

	return files
}

// FindChatSessionFiles finds all chatSessions JSON files in a workspaceStorage Copilot Chat directory.
func FindChatSessionFiles(chatDir string) []string {
	sessionsDir := filepath.Join(chatDir, "chatSessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			files = append(files, filepath.Join(sessionsDir, entry.Name()))
		}
	}

	return files
}

// FindDebugLogFiles finds all debug log directories and their main.jsonl files.
func FindDebugLogFiles(chatDir string) []string {
	debugDir := filepath.Join(chatDir, "debug-logs")
	entries, err := os.ReadDir(debugDir)
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			mainLog := filepath.Join(debugDir, entry.Name(), "main.jsonl")
			if _, err := os.Stat(mainLog); err == nil {
				files = append(files, mainLog)
			}
		}
	}

	return files
}
