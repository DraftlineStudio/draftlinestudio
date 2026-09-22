//go:build windows

package platform

import "os/exec"

// Detach is a no-op on Windows. A child that must outlive this process is
// launched through the shell instead (see openDownloadedUpdate): a job object
// can take direct children down with it, and cmd.exe exits immediately, so
// what it started is left running.
func Detach(cmd *exec.Cmd) {}
