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
	args := codexExecArgs("", "C:/tmp/out.txt", false)
	if args[len(args)-1] != "-" {
		t.Fatalf("stdin placeholder must be last, got %v", args)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "-m ") {
		t.Fatalf("model flag present without override: %v", args)
	}
	for _, want := range []string{"exec", "--skip-git-repo-check", "--ignore-user-config", "--ignore-rules", "--strict-config", "--sandbox", "read-only", "--ask-for-approval", "never", "--ephemeral", "--color", "never", "--json", "--output-last-message"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %v", want, args)
		}
	}
	for _, feature := range []string{"shell_tool", "unified_exec", "view_image", "apps", "browser_use", "computer_use", "image_generation", "multi_agent", "skill_search", "hooks"} {
		if !strings.Contains(joined, "--disable "+feature) {
			t.Fatalf("capability %q was not disabled in %v", feature, args)
		}
	}

	withModel := codexExecArgs("gpt-5-codex", "out.txt", true)
	joined = strings.Join(withModel, " ")
	if !strings.Contains(joined, "-m gpt-5-codex") {
		t.Fatalf("model override missing: %v", withModel)
	}
	if withModel[len(withModel)-1] != "-" {
		t.Fatalf("stdin placeholder must remain last with model override: %v", withModel)
	}
	if !strings.Contains(joined, `model_reasoning_effort="low"`) {
		t.Fatalf("lightweight request did not lower Codex reasoning effort: %v", withModel)
	}
}

func TestClaudeCodeExecArgsDisableAllTools(t *testing.T) {
	args := claudeCodeExecArgs("claude-haiku", true)
	joined := strings.Join(args, " ")
	for _, want := range []string{"--safe-mode", "--disable-slash-commands", "--strict-mcp-config", "--tools", "--permission-mode dontAsk", "--no-session-persistence", "--effort low"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %v", want, args)
		}
	}
	for i := range args {
		if args[i] == "--tools" {
			if i+1 >= len(args) || args[i+1] != "" {
				t.Fatalf("Claude tool list is not explicitly empty: %v", args)
			}
			return
		}
	}
	t.Fatal("Claude --tools flag missing")
}

func TestClaudeFailureMessageDoesNotEchoPrompt(t *testing.T) {
	stderr := "Error: request failed while processing PRIVATE MANUSCRIPT CONTENT"
	message := claudeFailureMessage(stderr, errors.New("exit status 1"))
	if strings.Contains(message, "PRIVATE MANUSCRIPT") {
		t.Fatalf("prompt leaked into Claude error message: %q", message)
	}
}

func TestSelectCodexLightweightModelUsesAvailableFastTier(t *testing.T) {
	cache := []byte(`{"models":[{"slug":"gpt-5.6-sol"},{"slug":"gpt-5.4-mini"},{"slug":"gpt-5.6-luna"}]}`)
	if got := selectCodexLightweightModel(cache); got != "gpt-5.6-luna" {
		t.Fatalf("expected Luna preference, got %q", got)
	}
	if got := selectCodexLightweightModel([]byte(`{"models":[{"slug":"gpt-5.4-mini"}]}`)); got != "gpt-5.4-mini" {
		t.Fatalf("expected compatible mini fallback, got %q", got)
	}
	if got := selectCodexLightweightModel([]byte(`{"models":[{"slug":"gpt-5.6-sol"}]}`)); got != "" {
		t.Fatalf("heavy-only cache should use safe CLI fallback, got %q", got)
	}
}

func TestLightweightRewriteProfileAndProviderModels(t *testing.T) {
	for _, mode := range []string{"line_edit", "copy_edit"} {
		if !rewriteRequestProfile(mode).lightweight {
			t.Fatalf("%s did not select lightweight routing", mode)
		}
	}
	for _, mode := range []string{"expand", "smooth", "custom"} {
		if rewriteRequestProfile(mode).lightweight {
			t.Fatalf("%s unexpectedly selected lightweight routing", mode)
		}
	}

	a := &App{}
	a.settings.AIModel = "claude-opus-4-6"
	if got := a.resolveRequestModel("claude-sonnet-4-6", "claude-haiku-4-5-20251001", rewriteRequestProfile("line_edit")); got != "claude-haiku-4-5-20251001" {
		t.Fatalf("line edit did not override Opus with Haiku: %q", got)
	}
	if got := a.resolveRequestModel("claude-sonnet-4-6", "claude-haiku-4-5-20251001", standardAIRequest); got != "claude-opus-4-6" {
		t.Fatalf("standard request did not preserve configured model: %q", got)
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
	res := a.callCodexCLI(nil, "sys", "msg", standardAIRequest)
	if res.Error == "" || !strings.Contains(res.Error, "not installed") {
		t.Fatalf("expected not-installed error, got %+v", res)
	}
}
