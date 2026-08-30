//go:build !windows

package platform

import (
	"context"
	"os/exec"
)

// BatchCommand runs the script directly; shell scripts are executable as-is.
func BatchCommand(ctx context.Context, path string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, path, args...)
}
