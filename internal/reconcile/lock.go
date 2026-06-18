package reconcile

import (
	"os"
	"path/filepath"
)

func lockPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tt", "reconcile.lock")
}
