package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestPrepareClaudeRequestCommandUsesOnlyIsolatedHome(t *testing.T) {
	tempHome := t.TempDir()
	cmd := exec.Command("claude")
	prepareClaudeRequestCommand(cmd, tempHome)
	env := envMap(cmd.Env)

	if cmd.Dir != tempHome {
		t.Fatalf("working directory = %q, want isolated home %q", cmd.Dir, tempHome)
	}
	if env["HOME"] != tempHome || env["USERPROFILE"] != tempHome {
		t.Fatalf("request escaped isolated home: HOME=%q USERPROFILE=%q", env["HOME"], env["USERPROFILE"])
	}
	if !strings.HasPrefix(env["PATH"], nodeInstallBinDir()) {
		t.Fatalf("managed Node directory is not first on PATH: %q", env["PATH"])
	}
}
