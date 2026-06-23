package transcript

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDiscoverWorkspaceStoragePaths_CurrentOS(t *testing.T) {
	paths := DiscoverWorkspaceStoragePaths()
	t.Logf("Found %d workspaceStorage paths", len(paths))

	// On the current machine, we should find at least the actual VS Code paths
	for _, p := range paths {
		t.Logf("  %s", p)
	}
}

func TestFindTranscriptFiles(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptsDir := filepath.Join(tmpDir, "transcripts")
	if err := os.MkdirAll(transcriptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create some files
	os.WriteFile(filepath.Join(transcriptsDir, "session1.jsonl"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(transcriptsDir, "session2.jsonl"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(transcriptsDir, "session3.json"), []byte("test"), 0644) // wrong extension

	files := FindTranscriptFiles(tmpDir)
	if len(files) != 2 {
		t.Errorf("expected 2 transcript files, got %d", len(files))
	}
}

func TestFindChatSessionFiles(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "chatSessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	os.WriteFile(filepath.Join(sessionsDir, "session1.json"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(sessionsDir, "session2.json"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(sessionsDir, "session3.jsonl"), []byte("test"), 0644) // wrong extension

	files := FindChatSessionFiles(tmpDir)
	if len(files) != 2 {
		t.Errorf("expected 2 chatSession files, got %d", len(files))
	}
}

func TestFindDebugLogFiles(t *testing.T) {
	tmpDir := t.TempDir()
	debugDir := filepath.Join(tmpDir, "debug-logs")
	sessionDir := filepath.Join(debugDir, "session-123")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	os.WriteFile(filepath.Join(sessionDir, "main.jsonl"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(sessionDir, "models.json"), []byte("test"), 0644)

	// Empty directory (no main.jsonl)
	os.MkdirAll(filepath.Join(debugDir, "session-456"), 0755)

	files := FindDebugLogFiles(tmpDir)
	if len(files) != 1 {
		t.Errorf("expected 1 debug log file, got %d", len(files))
	}
}

func TestFindTranscriptFiles_NoDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	files := FindTranscriptFiles(tmpDir)
	if len(files) != 0 {
		t.Errorf("expected 0 files for missing directory, got %d", len(files))
	}
}

func TestDiscoverVSCodePaths(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping macOS-specific test")
	}

	// Test with a mock base directory
	tmpDir := t.TempDir()
	variant := "Code"
	vscodeDir := filepath.Join(tmpDir, variant, "User", "workspaceStorage", "ext-id", "GitHub.copilot-chat")
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		t.Fatal(err)
	}

	paths := discoverVSCodePaths(tmpDir, variant)
	if len(paths) != 1 {
		t.Errorf("expected 1 path, got %d", len(paths))
	}
	if len(paths) > 0 && paths[0] != vscodeDir {
		t.Errorf("expected %s, got %s", vscodeDir, paths[0])
	}
}
