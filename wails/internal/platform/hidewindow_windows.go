//go:build windows

// Package platform provides platform-specific utilities.
package platform

import (
	"os/exec"
	"syscall"
)

// HideWindow configures a command to run without a visible console window on Windows.
// It merges into any existing SysProcAttr so a CmdLine set by BatchCommand survives.
func HideWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= 0x08000000 // CREATE_NO_WINDOW
}
