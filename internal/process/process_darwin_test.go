//go:build darwin

package process

import (
	"os"
	"testing"
)

func TestStartTime_Darwin_Parent(t *testing.T) {
	ts, err := StartTime(os.Getppid())
	if err != nil {
		t.Fatalf("StartTime(ppid): %v", err)
	}
	if ts <= 0 {
		t.Errorf("StartTime = %d, want positive Unix timestamp", ts)
	}
}

func TestStartTime_Darwin_InvalidPID(t *testing.T) {
	_, err := StartTime(-1)
	if err == nil {
		t.Error("StartTime(-1): want non-nil error, got nil")
	}
}
