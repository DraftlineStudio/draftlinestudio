//go:build !windows

package instancelock

import (
	"errors"
	"syscall"
)

// pidAlive reports whether the process exists (signal 0 probe). EPERM means
// it exists but belongs to another user — alive for our purposes.
func pidAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
