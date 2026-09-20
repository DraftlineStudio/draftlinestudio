package main

import (
	"context"
	"strings"
	"testing"
)

func TestRewriteProfileTiersAndProviderModels(t *testing.T) {
	for mode, want := range map[string]aiTier{
		"copy_edit": tierLite,
		"line_edit": tierMedium,
		"smooth":    tierMedium,
		"expand":    tierHigh,
	} {
		if got := rewriteRequestProfile(mode).tier; got != want {
			t.Fatalf("%s routed to %q, want %q", mode, got, want)
		}
	}
	if standardAIRequest.tier != tierHigh {
		t.Fatalf("a free-form prompt should get the most capable model, got %q", standardAIRequest.tier)
	}

	a := &App{}
	a.settings.AIModel = "claude-opus-4-6"
	// A bulk copy pass stays cheap even when a model is configured by hand.
	if got := a.resolveTierModel(claudeAPITierModels, rewriteRequestProfile("copy_edit")); got != claudeAPITierModels[tierLite] {
		t.Fatalf("copy edit did not stay on the lite model: %q", got)
	}
	if got := a.resolveTierModel(claudeAPITierModels, standardAIRequest); got != "claude-opus-4-6" {
		t.Fatalf("standard request did not preserve configured model: %q", got)
	}
}

// The CLI takes tier aliases, which it resolves to whatever it ships today.
func TestClaudeCodeUsesTierAliasesNotPinnedVersions(t *testing.T) {
	a := &App{}
	for mode, want := range map[string]string{
		"copy_edit": "haiku",
		"line_edit": "sonnet",
		"expand":    "opus",
	} {
		if got := a.claudeCodeModel(rewriteRequestProfile(mode)); got != want {
			t.Fatalf("%s asked for %q, want %q", mode, got, want)
		}
	}
	a.settings.AIModel = "claude-sonnet-4-6"
	if got := a.claudeCodeModel(standardAIRequest); got != "claude-sonnet-4-6" {
		t.Fatalf("a model set by hand should win: %q", got)
	}
	a.settings.AIModel = "gpt-4o"
	if got := a.claudeCodeModel(standardAIRequest); got != "opus" {
		t.Fatalf("another provider's model must not leak into Claude Code: %q", got)
	}
}

// Codex mode with no CLI installed must fail with guidance, not a crash.
func TestCallCodexCLINotInstalled(t *testing.T) {
	a := &App{}
	res := a.callCodexCLIAtPath(context.Background(), "", "sys", "msg", standardAIRequest)
	if res.Error == "" || !strings.Contains(res.Error, "not installed") {
		t.Fatalf("expected not-installed error, got %+v", res)
	}
}
