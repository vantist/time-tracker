package setup

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

// SetupVSCodeCopilot downloads and installs the VS Code Copilot bridge extension.
func SetupVSCodeCopilot() error {
	codePath := findVSCodePath()
	if codePath == "" {
		return fmt.Errorf("VS Code not found, skipping VS Code Copilot bridge installation")
	}

	// Check if extension is already installed
	if isExtensionInstalled("tt.copilot-bridge") {
		fmt.Println("VS Code Copilot bridge already installed")
		return nil
	}

	// Try to download from GitHub releases
	vsixPath, err := downloadVSIX()
	if err != nil {
		fmt.Printf("Could not auto-install extension: %v\n", err)
		fmt.Println("To install manually:")
		fmt.Printf("  1. Download .vsix from https://github.com/vantist/time-tracker/releases\n")
		fmt.Printf("  2. Run: %s --install-extension <path-to-vsix>\n", codePath)
		return nil
	}
	defer os.Remove(vsixPath)

	// Install the extension
	cmd := exec.Command(codePath, "--install-extension", vsixPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install extension: %w", err)
	}

	fmt.Println("VS Code Copilot bridge installed successfully")
	return nil
}

func isExtensionInstalled(extensionID string) bool {
	codePath := findVSCodePath()
	if codePath == "" {
		return false
	}

	out, err := exec.Command(codePath, "--list-extensions").Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(out), extensionID)
}

func downloadVSIX() (string, error) {
	// Get latest release info from GitHub
	resp, err := http.Get("https://api.github.com/repos/vantist/time-tracker/releases/latest")
	if err != nil {
		return "", fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Parse response to find vsix asset URL
	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse release info: %w", err)
	}

	var vsixURL string
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".vsix") {
			vsixURL = asset.BrowserDownloadURL
			break
		}
	}

	if vsixURL == "" {
		return "", fmt.Errorf("no .vsix asset found in latest release")
	}

	// Download the .vsix file
	tmpFile, err := os.CreateTemp("", "tt-copilot-bridge-*.vsix")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	resp, err = http.Get(vsixURL)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to download .vsix: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to save .vsix: %w", err)
	}

	return tmpFile.Name(), nil
}

func findVSCodePath() string {
	// Try common locations
	paths := []string{"code"}

	if runtime.GOOS == "darwin" {
		paths = append(paths,
			"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
			filepath.Join(os.Getenv("HOME"), "Applications", "Visual Studio Code.app", "Contents", "Resources", "app", "bin", "code"),
		)
	} else if runtime.GOOS == "linux" {
		paths = append(paths,
			"/usr/local/bin/code",
			"/usr/bin/code",
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
