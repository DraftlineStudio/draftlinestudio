//go:build windows

// Package platform provides platform-specific utilities.
package platform

import (
	"os/exec"
	"syscall"
)

// HideWindow configures a command to run without a visible console window on Windows.
func HideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
