//go:build darwin

package process

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func StartTime(pid int) (int64, error) {
	kinfo, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return 0, fmt.Errorf("process.StartTime: %w", err)
	}
	return kinfo.Proc.P_starttime.Sec, nil
}
