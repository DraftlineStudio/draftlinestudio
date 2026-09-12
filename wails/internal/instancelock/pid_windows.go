//go:build windows

package instancelock

import "golang.org/x/sys/windows"

// pidAlive reports whether the process exists and has not exited. A PID that
// cannot be opened at all is treated as dead (stale locks must be stealable;
// a permission-walled foreign process holding OUR per-user lock file is not a
// realistic state).
func pidAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	const stillActive = 259 // STILL_ACTIVE
	return code == stillActive
}
