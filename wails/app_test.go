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
