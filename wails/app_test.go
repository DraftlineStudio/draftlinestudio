package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"draftline/internal/types"
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
	list := []types.AIProvider{{ID: "p1", Nickname: "Mine", Kind: "cloud"}}
	for _, mode := range []string{"claudecode", "codex", "api", "p1"} {
		if got := normalizeAIProviderMode(mode, "claudecode", list); got != mode {
			t.Fatalf("explicit mode %q resolved as %q", mode, got)
		}
	}
	if got := normalizeAIProviderMode("", "codex", list); got != "codex" {
		t.Fatalf("empty task route should inherit default, got %q", got)
	}
	if got := normalizeAIProviderMode("not-a-provider", "p1", list); got != "p1" {
		t.Fatalf("invalid task route should inherit valid default, got %q", got)
	}
	if got := normalizeAIProviderMode("", "not-a-provider", list); got != "claudecode" {
		t.Fatalf("invalid default should use safe legacy default, got %q", got)
	}
	// A provider the writer deleted must not keep routing work to it.
	if got := normalizeAIProviderMode("p1", "codex", nil); got != "codex" {
		t.Fatalf("removed provider should fall back, got %q", got)
	}
}

func TestProviderModelOverridesDoNotLeakAcrossTaskRoutes(t *testing.T) {
	app := &App{}
	app.settings.AIModel = "gpt-5.4"
	if got := app.resolveAIModel("claude-sonnet-5"); got != "claude-sonnet-5" {
		t.Fatalf("OpenAI model leaked into Claude route: %q", got)
	}
	app.settings.AIModel = "claude-opus-5"
	if got := app.resolveAIModel("gpt-4o"); got != "gpt-4o" {
		t.Fatalf("Claude model leaked into OpenAI route: %q", got)
	}
	if got := app.resolveAIModel("claude-sonnet-5"); got != "claude-opus-5" {
		t.Fatalf("compatible Claude override was not retained: %q", got)
	}
}

func TestDefaultSaveDirIsAFolderOfItsOwn(t *testing.T) {
	dir := defaultSaveDir()
	if dir == "" {
		t.Skip("no home directory on this machine")
	}
	if filepath.Base(dir) != "Draftline" {
		t.Fatalf("defaultSaveDir = %q, want it to end in Draftline", dir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(dir)
	if parent != filepath.Join(home, "Documents") && parent != home {
		t.Fatalf("defaultSaveDir = %q, want it under Documents or the home directory", dir)
	}
}

func TestBookDialogDirCreatesTheFolderItPointsAt(t *testing.T) {
	app := &App{}
	want := filepath.Join(t.TempDir(), "Draftline")
	app.setSettings(types.AppSettings{DefaultSaveDir: want})

	// A dialog handed a directory that does not exist falls back to wherever
	// the operating system feels like, which is the mess being avoided.
	if got := app.bookDialogDir(); got != want {
		t.Fatalf("bookDialogDir = %q, want %q", got, want)
	}
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Fatalf("the folder was not created: %v", err)
	}
}

func TestBookDialogDirStaysEmptyWhenUnset(t *testing.T) {
	app := &App{}
	app.setSettings(types.AppSettings{DefaultSaveDir: "  "})
	if got := app.bookDialogDir(); got != "" {
		t.Fatalf("bookDialogDir = %q, want empty so the dialog picks", got)
	}
}
