package main

import (
	"os"
	"testing"

	"draftline/internal/booklock"
)

// These cover the wiring, not the mechanism: internal/booklock has its own
// tests for staleness, takeover and the hedged wording.

func TestInspectBookLockIsSilentForAnUnclaimedBook(t *testing.T) {
	app := &App{}
	path := writeTestBook(t)

	if info := app.InspectBookLock(path); info.Held {
		t.Fatalf("InspectBookLock = %+v, want nothing held", info)
	}
}

func TestInspectBookLockIsSilentAboutThisSession(t *testing.T) {
	app := &App{}
	path := writeTestBook(t)

	app.claimDeviceLock(path, "bk-test")
	t.Cleanup(app.releaseDeviceLock)

	// A book this window has open must not warn the author about itself.
	if info := app.InspectBookLock(path); info.Held {
		t.Fatalf("InspectBookLock = %+v, want silence about our own claim", info)
	}
}

func TestInspectBookLockReportsAnotherDevice(t *testing.T) {
	app := &App{}
	path := writeTestBook(t)

	other := booklock.Identity{
		Device: "OTHER-DESKTOP", Platform: "windows", App: "Draftline 0.21", Session: "someone-else",
	}
	if _, _, err := booklock.Claim(path, "bk-test", other); err != nil {
		t.Fatal(err)
	}

	info := app.InspectBookLock(path)
	if !info.Held || info.Stale {
		t.Fatalf("InspectBookLock = %+v, want a live claim", info)
	}
	if info.Device != "OTHER-DESKTOP" {
		t.Fatalf("Device = %q, want the other machine", info.Device)
	}
	// Hedged, always. Sync cannot promise more than "may".
	if info.Message != "This book may be open on OTHER-DESKTOP." {
		t.Fatalf("Message = %q", info.Message)
	}
}

func TestClaimingTakesOverAndReleasingFreesTheBook(t *testing.T) {
	app := &App{}
	path := writeTestBook(t)

	other := booklock.Identity{
		Device: "OTHER-DESKTOP", Platform: "linux", App: "Draftline 0.21", Session: "someone-else",
	}
	if _, _, err := booklock.Claim(path, "bk-test", other); err != nil {
		t.Fatal(err)
	}

	// The author has read the warning and opened it anyway.
	app.claimDeviceLock(path, "bk-test")
	if info := app.InspectBookLock(path); info.Held {
		t.Fatalf("after taking over, InspectBookLock = %+v, want it to be ours", info)
	}

	app.releaseDeviceLock()
	if _, err := os.Stat(booklock.SidecarFor(path)); !os.IsNotExist(err) {
		t.Fatalf("the sidecar outlived the release: %v", err)
	}
}

func TestReleasingTwiceIsHarmless(t *testing.T) {
	app := &App{}
	path := writeTestBook(t)

	app.claimDeviceLock(path, "bk-test")
	app.releaseDeviceLock()
	app.releaseDeviceLock() // closing a book that is already closed
}

// writeTestBook makes a file to claim. booklock never reads it — the claim
// lives beside the book, not inside it — so its contents do not matter.
func writeTestBook(t *testing.T) string {
	t.Helper()
	path := t.TempDir() + "/The Weather House.draftline"
	if err := os.WriteFile(path, []byte("stand-in for an archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
