//go:build windows

package platform

import "os/exec"

// Detach is a no-op on Windows: child processes are not tied to the parent's
// lifetime there, and HideWindow already sets the creation flags that matter.
func Detach(cmd *exec.Cmd) {}
