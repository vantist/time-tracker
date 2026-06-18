//go:build !darwin && !windows

package process

func StartTime(pid int) (int64, error) {
	return 0, nil
}
