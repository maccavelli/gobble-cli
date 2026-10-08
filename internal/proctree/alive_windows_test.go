//go:build windows

package proctree_test

import "golang.org/x/sys/windows"

// stillActive is GetExitCodeProcess's STILL_ACTIVE.
const stillActive = 259

// alive reports whether pid is a running process.
func alive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid)) //nolint:gosec // G115: a pid fits in uint32
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h) //nolint:errcheck // a query handle
	var code uint32
	return windows.GetExitCodeProcess(h, &code) == nil && code == stillActive
}
