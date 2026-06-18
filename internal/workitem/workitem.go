package workitem

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveProject returns the git root of dir, or dir itself if not in a git repo.
func ResolveProject(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return dir
	}
	return strings.TrimSpace(string(out))
}

func projectKey(project string) string {
	resolved := ResolveProject(project)
	h := sha256.Sum256([]byte(resolved))
	return fmt.Sprintf("%x", h[:8]) // 8 bytes = 16 hex chars
}

func workItemPath(project string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".tt", "work-items")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, projectKey(project)), nil
}

func Set(label, project string) error {
	p, err := workItemPath(project)
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(label+"\n"), 0o644)
}

func Get(project string) (string, error) {
	p, err := workItemPath(project)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\n"), nil
}

func Clear(project string) error {
	p, err := workItemPath(project)
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
