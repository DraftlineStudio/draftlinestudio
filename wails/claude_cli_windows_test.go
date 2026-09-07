//go:build windows

package main

import (
	"os/exec"
	"testing"
)

func TestPrepareClaudeRequestCommandSuppressesWindowsConsole(t *testing.T) {
	cmd := exec.Command("claude")
	prepareClaudeRequestCommand(cmd, t.TempDir())
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatal("Claude request command would open a visible console window")
	}
	if cmd.SysProcAttr.CreationFlags&0x08000000 == 0 {
		t.Fatal("Claude request command is missing CREATE_NO_WINDOW")
	}
}
