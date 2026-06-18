//go:build !windows

package reconcile

import (
	"os"

	"golang.org/x/sys/unix"
)

func tryLock(path string) (unlock func(), ok bool) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, false
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		return nil, false
	}
	return func() {
		unix.Flock(int(f.Fd()), unix.LOCK_UN) //nolint:errcheck
		f.Close()
	}, true
}
