//go:build !windows

// Package platform provides platform-specific utilities.
package platform

import "os/exec"

// HideWindow is a no-op on non-Windows platforms.
func HideWindow(cmd *exec.Cmd) {}
