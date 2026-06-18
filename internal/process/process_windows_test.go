//go:build windows

package process

import "testing"

func TestFiletimeToUnix_Epoch(t *testing.T) {
	// 1970-01-01T00:00:00Z in FILETIME units = 11644473600 * 10000000
	const unixEpochFT uint64 = 116444736000000000
	got := filetimeToUnix(unixEpochFT)
	if got != 0 {
		t.Errorf("filetimeToUnix(unixEpochFT) = %d, want 0", got)
	}
}

func TestFiletimeToUnix_KnownValue(t *testing.T) {
	// 2024-01-01T00:00:00Z = Unix 1704067200
	// FILETIME = (1704067200 + 11644473600) * 10000000 = 133483872000000000
	const ft uint64 = 133483872000000000
	got := filetimeToUnix(ft)
	if got != 1704067200 {
		t.Errorf("filetimeToUnix(2024-01-01) = %d, want 1704067200", got)
	}
}
