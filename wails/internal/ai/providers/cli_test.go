package providers

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// The prompt must never appear in argv (injection surface, cmd-line length
// limit) — stdin ("-") must always be the final argument, and the model flag
// only appears when an override is configured.
func TestCodexExecArgs(t *testing.T) {
	args := CodexExecArgs("", "C:/tmp/out.txt", false)
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
	approvalIndex := -1
	execIndex := -1
	for i, arg := range args {
		if arg == "--ask-for-approval" {
			approvalIndex = i
		}
		if arg == "exec" {
			execIndex = i
		}
	}
	if approvalIndex < 0 || execIndex < 0 || approvalIndex > execIndex {
		t.Fatalf("root approval option must precede exec subcommand: %v", args)
	}
	for _, feature := range []string{"shell_tool", "unified_exec", "view_image", "apps", "browser_use", "computer_use", "image_generation", "multi_agent", "skill_search", "hooks"} {
		if !strings.Contains(joined, "--disable "+feature) {
			t.Fatalf("capability %q was not disabled in %v", feature, args)
		}
	}

	withModel := CodexExecArgs("gpt-5-codex", "out.txt", true)
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
	args := ClaudeCodeExecArgs("claude-haiku", true)
	joined := strings.Join(args, " ")
	for _, want := range []string{"--safe-mode", "--disable-slash-commands", "--strict-mcp-config", "--tools", "--permission-mode dontAsk", "--no-session-persistence", "--effort low"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %v", want, args)
		}
	}
	foundTools := false
	foundMCPConfig := false
	for i := range args {
		if args[i] == "--tools" {
			if i+1 >= len(args) || args[i+1] != "" {
				t.Fatalf("Claude tool list is not explicitly empty: %v", args)
			}
			foundTools = true
		}
		if args[i] == "--mcp-config" {
			if i+1 >= len(args) {
				t.Fatalf("Claude MCP config value missing: %v", args)
			}
			var config struct {
				MCPServers map[string]any `json:"mcpServers"`
			}
			if err := json.Unmarshal([]byte(args[i+1]), &config); err != nil || config.MCPServers == nil {
				t.Fatalf("Claude MCP config must contain an empty mcpServers record: %q (%v)", args[i+1], err)
			}
			foundMCPConfig = true
		}
	}
	if !foundTools {
		t.Fatal("Claude --tools flag missing")
	}
	if !foundMCPConfig {
		t.Fatal("Claude --mcp-config flag missing")
	}
}

func TestClaudeFailureMessageDoesNotEchoPrompt(t *testing.T) {
	stderr := "Error: request failed while processing PRIVATE MANUSCRIPT CONTENT"
	message := ClaudeFailureMessage(stderr, errors.New("exit status 1"))
	if strings.Contains(message, "PRIVATE MANUSCRIPT") {
		t.Fatalf("prompt leaked into Claude error message: %q", message)
	}
}

func TestSelectCodexLightweightModelUsesAvailableFastTier(t *testing.T) {
	cache := []byte(`{"models":[{"slug":"gpt-5.6-sol"},{"slug":"gpt-5.4-mini"},{"slug":"gpt-5.6-luna"}]}`)
	if got := SelectCodexLightweightModel(cache); got != "gpt-5.6-luna" {
		t.Fatalf("expected Luna preference, got %q", got)
	}
	if got := SelectCodexLightweightModel([]byte(`{"models":[{"slug":"gpt-5.4-mini"}]}`)); got != "gpt-5.4-mini" {
		t.Fatalf("expected compatible mini fallback, got %q", got)
	}
	if got := SelectCodexLightweightModel([]byte(`{"models":[{"slug":"gpt-5.6-sol"}]}`)); got != "" {
		t.Fatalf("heavy-only cache should use safe CLI fallback, got %q", got)
	}
}

func TestCodexModelOverrideRejectsOtherProviders(t *testing.T) {
	for _, model := range []string{"claude-sonnet-4-6", " Gemini-2.5-pro ", "grok-2", "llama3"} {
		if got := CodexModelOverride(model); got != "" {
			t.Fatalf("provider-specific model %q leaked into Codex as %q", model, got)
		}
	}
	if got := CodexModelOverride(" gpt-5.4 "); got != "gpt-5.4" {
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
	message := CodexFailureMessage(stderr, errors.New("exit status 1"))
	if strings.Contains(message, "private manuscript") || strings.Contains(message, "skilled literary") {
		t.Fatalf("prompt leaked into error message: %q", message)
	}
	if !strings.Contains(message, "not supported") {
		t.Fatalf("actionable Codex error missing: %q", message)
	}

	fallback := CodexFailureMessage("user\nprivate prompt contents", errors.New("exit status 1"))
	if strings.Contains(fallback, "private prompt") {
		t.Fatalf("prompt leaked through fallback: %q", fallback)
	}
}
