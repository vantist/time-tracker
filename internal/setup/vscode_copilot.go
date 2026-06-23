package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// IsVSCodeCopilotActive checks if VS Code is installed and has GitHub Copilot Chat data.
func IsVSCodeCopilotActive() bool {
	// Check if VS Code is installed
	codePath := findVSCodePath()
	if codePath == "" {
		return false
	}

	// Check if Copilot Chat data exists in workspaceStorage
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	var workspaceStorageBase string
	switch runtime.GOOS {
	case "darwin":
		workspaceStorageBase = filepath.Join(home, "Library", "Application Support")
	case "linux":
		workspaceStorageBase = filepath.Join(home, ".config")
	case "windows":
		workspaceStorageBase = os.Getenv("APPDATA")
		if workspaceStorageBase == "" {
			workspaceStorageBase = filepath.Join(home, "AppData", "Roaming")
		}
	}

	if workspaceStorageBase == "" {
		return false
	}

	// Check for GitHub.copilot-chat directories in workspaceStorage
	variants := []string{"Code", "Code - Insiders", "VSCodium", "Cursor"}
	for _, variant := range variants {
		workspaceStorage := filepath.Join(workspaceStorageBase, variant, "User", "workspaceStorage")
		if hasCopilotChatData(workspaceStorage) {
			return true
		}
	}

	return false
}

// hasCopilotChatData checks if any workspaceStorage directory has GitHub.copilot-chat data.
func hasCopilotChatData(workspaceStorage string) bool {
	entries, err := os.ReadDir(workspaceStorage)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		chatDir := filepath.Join(workspaceStorage, entry.Name(), "GitHub.copilot-chat")
		if info, err := os.Stat(chatDir); err == nil && info.IsDir() {
			return true
		}
	}

	return false
}

// SetupVSCodeCopilot installs the VS Code Copilot bridge extension.
func SetupVSCodeCopilot() error {
	codePath := findVSCodePath()
	if codePath == "" {
		return fmt.Errorf("VS Code not found, skipping VS Code Copilot bridge installation")
	}

	// For now, just print instructions since we don't have a .vsix to install
	fmt.Println("To install the VS Code Copilot bridge:")
	fmt.Println("  1. Open VS Code")
	fmt.Println("  2. Press Ctrl+Shift+P (or Cmd+Shift+P on macOS)")
	fmt.Println("  3. Type 'Extensions: Install from VSIX...'")
	fmt.Println("  4. Select the tt-copilot-bridge.vsix file")
	fmt.Println("")
	fmt.Println("Alternatively, run:")
	fmt.Printf("  %s --install-extension <path-to-vsix>\n", codePath)

	return nil
}

func findVSCodePath() string {
	// Try common locations
	paths := []string{
		"code",
		"/usr/local/bin/code",
		"/usr/bin/code",
	}

	if runtime.GOOS == "darwin" {
		paths = append(paths,
			"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
			filepath.Join(os.Getenv("HOME"), "Applications", "Visual Studio Code.app", "Contents", "Resources", "app", "bin", "code"),
		)
	} else if runtime.GOOS == "linux" {
		paths = append(paths,
			"/snap/bin/code",
			"/usr/share/code/bin/code",
		)
	} else if runtime.GOOS == "windows" {
		programFiles := os.Getenv("PROGRAMFILES")
		if programFiles != "" {
			paths = append(paths,
				filepath.Join(programFiles, "Microsoft VS Code", "bin", "code.cmd"),
			)
		}
		programFilesX86 := os.Getenv("PROGRAMFILES(X86)")
		if programFilesX86 != "" {
			paths = append(paths,
				filepath.Join(programFilesX86, "Microsoft VS Code", "bin", "code.cmd"),
			)
		}
	}

	for _, p := range paths {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}
