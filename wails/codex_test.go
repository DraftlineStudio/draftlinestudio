package main

import (
	"strings"
	"testing"
)

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

// Codex mode with no CLI installed must fail with guidance, not a crash.
func TestCallCodexCLINotInstalled(t *testing.T) {
	t.Setenv("PATH", "") // ensure no system codex is found
	a := &App{}
	res := a.callCodexCLI(nil, "sys", "msg", standardAIRequest)
	if res.Error == "" || !strings.Contains(res.Error, "not installed") {
		t.Fatalf("expected not-installed error, got %+v", res)
	}
}
