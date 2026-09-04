package main

import (
	"context"
	"errors"
	"testing"
)

// TestAcquireAISerializes covers the single AI-request slot semantics:
// mutual exclusion, CancelRewrite cancelling the live context without freeing
// the slot, and a stale double-release never freeing a successor's slot.
func TestAcquireAISerializes(t *testing.T) {
	app := &App{ctx: context.Background()}

	// First acquire succeeds.
	ctx1, release1, ok := app.acquireAI()
	if !ok {
		t.Fatal("first acquire should succeed")
	}
	if ctx1.Err() != nil {
		t.Fatal("fresh context must not be cancelled")
	}

	// Mutual exclusion: a second concurrent acquire is refused.
	if _, _, ok := app.acquireAI(); ok {
		t.Fatal("second concurrent acquire should be refused while the slot is held")
	}

	// CancelRewrite cancels the live context but does NOT free the slot.
	app.CancelRewrite()
	if !errors.Is(ctx1.Err(), context.Canceled) {
		t.Fatal("CancelRewrite should cancel the live context")
	}
	if _, _, ok := app.acquireAI(); ok {
		t.Fatal("cancelled request still owns the slot until its own release runs")
	}

	// The owner's release frees the slot for a successor.
	release1()
	ctx2, release2, ok := app.acquireAI()
	if !ok {
		t.Fatal("acquire should succeed after the owner's release")
	}

	// Stale double-release from the first request must not free or disturb
	// the successor's slot.
	release1()
	if ctx2.Err() != nil {
		t.Fatal("stale release must not cancel the successor's context")
	}
	if _, _, ok := app.acquireAI(); ok {
		t.Fatal("stale double-release freed the successor's slot")
	}

	// The successor's own release works normally, and release is idempotent.
	release2()
	release2()
	ctx3, release3, ok := app.acquireAI()
	if !ok {
		t.Fatal("acquire should succeed after the successor's release")
	}
	if ctx3.Err() != nil {
		t.Fatal("third context should be live")
	}
	release3()
}

func TestNormalizeAIProviderMode(t *testing.T) {
	for _, mode := range []string{"claudecode", "codex", "api", "local"} {
		if got := normalizeAIProviderMode(mode, "claudecode"); got != mode {
			t.Fatalf("explicit mode %q resolved as %q", mode, got)
		}
	}
	if got := normalizeAIProviderMode("", "codex"); got != "codex" {
		t.Fatalf("empty task route should inherit default, got %q", got)
	}
	if got := normalizeAIProviderMode("not-a-provider", "local"); got != "local" {
		t.Fatalf("invalid task route should inherit valid default, got %q", got)
	}
	if got := normalizeAIProviderMode("", "not-a-provider"); got != "claudecode" {
		t.Fatalf("invalid default should use safe legacy default, got %q", got)
	}
}

func TestProviderModelOverridesDoNotLeakAcrossTaskRoutes(t *testing.T) {
	app := &App{}
	app.settings.AIModel = "gpt-5.4"
	if got := app.resolveAIModel("claude-sonnet-4-6"); got != "claude-sonnet-4-6" {
		t.Fatalf("OpenAI model leaked into Claude route: %q", got)
	}
	app.settings.AIModel = "claude-opus-4-6"
	if got := app.resolveAIModel("gpt-4o"); got != "gpt-4o" {
		t.Fatalf("Claude model leaked into OpenAI route: %q", got)
	}
	if got := app.resolveAIModel("claude-sonnet-4-6"); got != "claude-opus-4-6" {
		t.Fatalf("compatible Claude override was not retained: %q", got)
	}
}
