package reconcile

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTryLock_SecondCallFails: same process acquiring lock twice — second must fail.
func TestTryLock_SecondCallFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	unlock1, ok1 := tryLock(path)
	if !ok1 {
		t.Fatal("first tryLock must succeed")
	}
	defer unlock1()

	_, ok2 := tryLock(path)
	if ok2 {
		t.Error("second tryLock must fail (LOCK_NB) when first is held")
	}
}

// TestLockPath: lockPath returns a path under ~/.tt/.
func TestLockPath(t *testing.T) {
	p := lockPath()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".tt", "reconcile.lock")
	if p != expected {
		t.Errorf("lockPath = %q, want %q", p, expected)
	}
}
