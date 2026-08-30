package main

import (
	"errors"
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
	for _, want := range []string{"exec", "--skip-git-repo-check", "--sandbox", "read-only", "--color", "never", "--json", "--output-last-message"} {
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

func TestCodexModelOverrideRejectsOtherProviders(t *testing.T) {
	for _, model := range []string{"claude-sonnet-4-6", " Gemini-2.5-pro ", "grok-2", "llama3"} {
		if got := codexModelOverride(model); got != "" {
			t.Fatalf("provider-specific model %q leaked into Codex as %q", model, got)
		}
	}
	if got := codexModelOverride(" gpt-5.4 "); got != "gpt-5.4" {
		t.Fatalf("valid Codex override changed: %q", got)
	}
}

func TestCodexFailureMessageDoesNotEchoPrompt(t *testing.T) {
	stderr := `OpenAI Codex v0.149.0
--------
model: claude-sonnet-4-6
user
You are a skilled literary prose editor. Rewrite this private manuscript.
ERROR: {"detail":"The model claude-sonnet-4-6 is not supported."}`
	message := codexFailureMessage(stderr, errors.New("exit status 1"))
	if strings.Contains(message, "private manuscript") || strings.Contains(message, "skilled literary") {
		t.Fatalf("prompt leaked into error message: %q", message)
	}
	if !strings.Contains(message, "not supported") {
		t.Fatalf("actionable Codex error missing: %q", message)
	}

	fallback := codexFailureMessage("user\nprivate prompt contents", errors.New("exit status 1"))
	if strings.Contains(fallback, "private prompt") {
		t.Fatalf("prompt leaked through fallback: %q", fallback)
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
