//go:build windows

package platform

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// unsafeBatchArg reports why s must not be placed on a cmd.exe command line,
// or nil if it is safe. cmd.exe interprets metacharacters even inside the
// quoting produced by syscall.EscapeArg, and expands %VAR% references even
// inside double quotes. On the /S /C command line (unlike inside a batch
// file's body) doubling percent signs does NOT escape them: "%%VAR%%" still
// expands the inner %VAR%. So any argument containing two or more % signs is
// rejected outright — there is no reliable way to escape them in this
// context. A single lone % (e.g. "100% done") cannot form an expansion and
// is allowed.
func unsafeBatchArg(s string) error {
	for _, r := range s {
		switch r {
		case '&', '|', '<', '>', '^', '"', '\r', '\n', 0:
			return fmt.Errorf("argument contains cmd.exe metacharacter %q: %q", r, s)
		}
	}
	if strings.Count(s, "%") >= 2 {
		return fmt.Errorf("argument contains multiple %% signs (cmd.exe %%VAR%% expansion risk): %q", s)
	}
	return nil
}

// BatchCommand runs a .cmd/.bat script via cmd.exe. Go's default escaping quotes
// each argument individually, but cmd.exe strips only the first and last quote on
// the /C line — so a quoted script path containing a space (e.g. C:\Users\JL
// Griffin\...) gets split once any other quoted argument follows it. Verified on
// go1.25.1: plain exec.Command on such a script fails the same way. The /S /C
// "..." form makes cmd.exe strip exactly the outer quote pair and execute the
// rest verbatim, which requires assembling the raw command line ourselves.
//
// Because the raw line is interpreted by cmd.exe, the path and every argument
// are validated with unsafeBatchArg; on a violation the returned Cmd has
// cmd.Err set (Start/Run will fail with it) and no command line is assembled.
func BatchCommand(ctx context.Context, path string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "cmd.exe")
	if err := unsafeBatchArg(path); err != nil {
		if cmd.Err == nil {
			cmd.Err = fmt.Errorf("refusing to run batch script: %w", err)
		}
		return cmd
	}
	for _, a := range args {
		if err := unsafeBatchArg(a); err != nil {
			if cmd.Err == nil {
				cmd.Err = fmt.Errorf("refusing to run batch script: %w", err)
			}
			return cmd
		}
	}
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
