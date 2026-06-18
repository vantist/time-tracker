package process

import (
	"os"
	"syscall"
)

// IsAlive reports whether the process with pid and start time is still running.
// Matches both pid and start to guard against PID reuse.
func IsAlive(pid int64, start int64) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(int(pid))
	if err != nil {
		return false
	}
	// Signal(0) checks process existence without sending a signal.
	if err := p.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	if start == 0 {
		return true
	}
	got, err := StartTime(int(pid))
	if err != nil {
		return true // can't confirm, assume alive
	}
	return got == start
}
