//go:build windows

package platform

import (
	"context"
	"strings"
	"testing"
)

func TestBatchCommandCmdLineSafeArgs(t *testing.T) {
	cmd := BatchCommand(context.Background(),
		`C:\Users\JL Griffin\AppData\Roaming\npm\claude.cmd`,
		"--version", "hello world")
	if cmd.Err != nil {
		t.Fatalf("unexpected cmd.Err for safe args: %v", cmd.Err)
	}
	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr not set for safe args")
	}
	want := `cmd.exe /S /C ""C:\Users\JL Griffin\AppData\Roaming\npm\claude.cmd" --version "hello world""`
	if got := cmd.SysProcAttr.CmdLine; got != want {
		t.Errorf("CmdLine mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestBatchCommandRejectsMetacharacters(t *testing.T) {
	cases := []struct {
		name string
		arg  string
	}{
		{"ampersand", "foo & calc.exe"},
		{"pipe", "foo | more"},
		{"redirect-in", "foo < secret.txt"},
		{"redirect-out", "foo > out.txt"},
		{"caret", "foo^&bar"},
		{"double-quote", `foo" & calc & "`},
		{"carriage-return", "foo\rbar"},
		{"newline", "foo\nbar"},
		{"nul", "foo\x00bar"},
		{"paired-percent", "%PATH%"},
		{"paired-percent-embedded", "load %USERPROFILE% now"},
		{"doubled-percent", "%%OS%%"},
		{"doubled-percent-embedded", "x%%USERPROFILE%%y"},
		{"doubled-percent-literal", "50%% off"},
		{"two-bare-percents", "a% b%"},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/arg", func(t *testing.T) {
			cmd := BatchCommand(context.Background(), `C:\safe\script.cmd`, tc.arg)
			if cmd.Err == nil {
				t.Fatalf("expected cmd.Err for hostile arg %q, got nil", tc.arg)
			}
			if cmd.SysProcAttr != nil && cmd.SysProcAttr.CmdLine != "" {
				t.Errorf("command line was assembled despite hostile arg %q", tc.arg)
			}
			if err := cmd.Start(); err == nil {
				_ = cmd.Process.Kill()
				t.Fatalf("Start succeeded despite hostile arg %q", tc.arg)
			}
		})
		t.Run(tc.name+"/path", func(t *testing.T) {
			cmd := BatchCommand(context.Background(), `C:\evil\`+tc.arg+`.cmd`)
			if cmd.Err == nil {
				t.Fatalf("expected cmd.Err for hostile path containing %q, got nil", tc.arg)
			}
		})
	}
}

func TestBatchCommandAllowsSinglePercent(t *testing.T) {
	for _, arg := range []string{"100% done", "plain", "--flag=value"} {
		cmd := BatchCommand(context.Background(), `C:\safe\script.cmd`, arg)
		if cmd.Err != nil {
			t.Errorf("arg %q should be allowed, got err: %v", arg, cmd.Err)
			continue
		}
		if cmd.SysProcAttr == nil || !strings.Contains(cmd.SysProcAttr.CmdLine, "script.cmd") {
			t.Errorf("arg %q: command line not assembled", arg)
		}
	}
}
