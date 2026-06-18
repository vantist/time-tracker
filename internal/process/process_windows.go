//go:build windows

package process

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// filetimeToUnix converts a Windows FILETIME (100ns intervals from 1601-01-01)
// to a Unix timestamp in seconds.
func filetimeToUnix(ft uint64) int64 {
	const epochDiff = 11644473600
	return int64(ft/10000000) - epochDiff
}

func StartTime(pid int) (int64, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return 0, fmt.Errorf("process.StartTime: OpenProcess %d: %w", pid, err)
	}
	defer windows.CloseHandle(h)

	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return 0, fmt.Errorf("process.StartTime: GetProcessTimes %d: %w", pid, err)
	}

	ft := uint64(creation.HighDateTime)<<32 | uint64(creation.LowDateTime)
	return filetimeToUnix(ft), nil
}
