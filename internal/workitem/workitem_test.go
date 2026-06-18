package workitem_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/user/tt/internal/workitem"
)

func project(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func setup(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
}

func TestSetAndGet(t *testing.T) {
	setup(t)
	proj := project(t)

	if err := workitem.Set("login-redesign", proj); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := workitem.Get(proj)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "login-redesign" {
		t.Errorf("Get = %q, want %q", got, "login-redesign")
	}

	// Verify file exists under ~/.tt/work-items/
	home, _ := os.UserHomeDir()
	entries, _ := os.ReadDir(filepath.Join(home, ".tt", "work-items"))
	if len(entries) == 0 {
		t.Error("expected file under ~/.tt/work-items/, found none")
	}
}

func TestGetMissingFileReturnsEmpty(t *testing.T) {
	setup(t)

	got, err := workitem.Get(project(t))
	if err != nil {
		t.Fatalf("Get on missing file: %v", err)
	}
	if got != "" {
		t.Errorf("Get = %q, want empty", got)
	}
}

func TestClearDeletesFile(t *testing.T) {
	setup(t)
	proj := project(t)

	_ = workitem.Set("some-task", proj)
	if err := workitem.Clear(proj); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	got, _ := workitem.Get(proj)
	if got != "" {
		t.Errorf("after Clear, Get = %q, want empty", got)
	}
}

func TestClearIdempotent(t *testing.T) {
	setup(t)

	if err := workitem.Clear(project(t)); err != nil {
		t.Errorf("Clear on missing file: %v", err)
	}
}

func TestResolveProjectGitRoot(t *testing.T) {
	// 建立臨時 git repo
	rawRoot := t.TempDir()
	// macOS: /var/... is a symlink to /private/var/...; EvalSymlinks gives canonical path matching git output
	root, err := filepath.EvalSymlinks(rawRoot)
	if err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(root, "pkg", "sub")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "init", root).Run(); err != nil {
		t.Skip("git not available")
	}

	got := workitem.ResolveProject(subdir)
	if got != root {
		t.Errorf("ResolveProject(subdir) = %q, want %q (git root)", got, root)
	}
}

func TestResolveProjectNonGit(t *testing.T) {
	dir := t.TempDir()
	got := workitem.ResolveProject(dir)
	if got != dir {
		t.Errorf("ResolveProject(non-git) = %q, want %q", got, dir)
	}
}

func TestGetSetClearPerProject(t *testing.T) {
	setup(t)

	projectA := t.TempDir()
	projectB := t.TempDir()

	if err := workitem.Set("TICKET-A", projectA); err != nil {
		t.Fatalf("Set projectA: %v", err)
	}
	if err := workitem.Set("TICKET-B", projectB); err != nil {
		t.Fatalf("Set projectB: %v", err)
	}

	gotA, err := workitem.Get(projectA)
	if err != nil {
		t.Fatalf("Get projectA: %v", err)
	}
	if gotA != "TICKET-A" {
		t.Errorf("Get projectA = %q, want %q", gotA, "TICKET-A")
	}

	gotB, err := workitem.Get(projectB)
	if err != nil {
		t.Fatalf("Get projectB: %v", err)
	}
	if gotB != "TICKET-B" {
		t.Errorf("Get projectB = %q, want %q", gotB, "TICKET-B")
	}

	if err := workitem.Clear(projectA); err != nil {
		t.Fatalf("Clear projectA: %v", err)
	}

	gotA, _ = workitem.Get(projectA)
	if gotA != "" {
		t.Errorf("after Clear, Get projectA = %q, want empty", gotA)
	}
	gotB, _ = workitem.Get(projectB)
	if gotB != "TICKET-B" {
		t.Errorf("after Clear projectA, Get projectB = %q, want %q", gotB, "TICKET-B")
	}
}
