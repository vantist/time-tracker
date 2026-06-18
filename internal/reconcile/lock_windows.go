//go:build windows

package reconcile

import (
	"os"

	"golang.org/x/sys/windows"
)

func tryLock(path string) (unlock func(), ok bool) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, false
	}

	ol := new(windows.Overlapped)
	err = windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, ol)
	if err != nil {
		f.Close()
		return nil, false
	}

	return func() {
		windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, ol) //nolint:errcheck
		f.Close()
	}, true
}
