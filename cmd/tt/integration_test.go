package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "tt-test-*")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}

	binPath = filepath.Join(tmpDir, "tt")

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("failed to compile tt binary: %v\nOutput: %s", err, string(output))
	}

	code := m.Run()

	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func TestIntegration_BinaryExists(t *testing.T) {
	if binPath == "" {
		t.Fatal("binPath is not set")
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("compiled binary does not exist at %s: %v", binPath, err)
	}
}
