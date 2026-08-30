//go:build windows

package platform

import (
	"context"
	"os/exec"
	"strings"
	"syscall"
)

// BatchCommand runs a .cmd/.bat script via cmd.exe. Go's default escaping quotes
// each argument individually, but cmd.exe strips only the first and last quote on
// the /C line — so a quoted script path containing a space (e.g. C:\Users\JL
// Griffin\...) gets split once any other quoted argument follows it. Verified on
// go1.25.1: plain exec.Command on such a script fails the same way. The /S /C
// "..." form makes cmd.exe strip exactly the outer quote pair and execute the
// rest verbatim, which requires assembling the raw command line ourselves.
func BatchCommand(ctx context.Context, path string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "cmd.exe")
	var b strings.Builder
	b.WriteString(`cmd.exe /S /C ""`)
	b.WriteString(path)
	b.WriteString(`"`)
	for _, a := range args {
		b.WriteString(" ")
		b.WriteString(syscall.EscapeArg(a))
	}
	b.WriteString(`"`)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: b.String()}
	return cmd
}
