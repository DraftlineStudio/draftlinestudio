//go:build !windows

package platform

import (
	"os/exec"
	"syscall"
)

// Detach puts the command in its own session so it outlives this process:
// the updater launches the replacement Draftline (or the terminal applying a
// package update) and then quits, and the child must not go down with it.
func Detach(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}
