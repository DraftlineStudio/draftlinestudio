package main

import (
	"strings"
	"testing"
)

// The prompt must never appear in argv (injection surface, cmd-line length
// limit) — stdin ("-") must always be the final argument, and the model flag
// only appears when an override is configured.
func TestCodexExecArgs(t *testing.T) {
	args := codexExecArgs("", "C:/tmp/out.txt")
	if args[len(args)-1] != "-" {
		t.Fatalf("stdin placeholder must be last, got %v", args)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "-m ") {
		t.Fatalf("model flag present without override: %v", args)
	}
	for _, want := range []string{"exec", "--skip-git-repo-check", "--sandbox", "read-only", "--output-last-message"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %v", want, args)
		}
	}

	withModel := codexExecArgs("gpt-5-codex", "out.txt")
	joined = strings.Join(withModel, " ")
	if !strings.Contains(joined, "-m gpt-5-codex") {
		t.Fatalf("model override missing: %v", withModel)
	}
	if withModel[len(withModel)-1] != "-" {
		t.Fatalf("stdin placeholder must remain last with model override: %v", withModel)
	}
}

// Codex mode with no CLI installed must fail with guidance, not a crash.
func TestCallCodexCLINotInstalled(t *testing.T) {
	t.Setenv("PATH", "") // ensure no system codex is found
	a := &App{}
	res := a.callCodexCLI(nil, "sys", "msg")
	if res.Error == "" || !strings.Contains(res.Error, "not installed") {
		t.Fatalf("expected not-installed error, got %+v", res)
	}
}
