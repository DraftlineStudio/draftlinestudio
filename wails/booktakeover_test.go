package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"draftline/internal/booklock"
)

// asking is the writer's other machine. It must not be named after this one:
// a claim naming this device is read as our own.
func asking() booklock.Identity {
	return booklock.Identity{Device: "ASKING-MACHINE", Platform: "linux", App: "Draftline 0.21", Session: "s-asking"}
}

func studio() booklock.Identity {
	return booklock.Identity{Device: "STUDIO-DESKTOP", Platform: "windows", App: "Draftline 0.21", Session: "s-studio"}
}

// bookFile is a book with some bytes in it, so there is something to hash.
func bookFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "The Weather House.draftline")
	if err := os.WriteFile(path, []byte(strings.Repeat("chapter", 400)), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// holdingApp is this machine with the book open and claimed. The claim is taken
// directly rather than through claimDeviceLock, so no heartbeat goroutine is
// left running against a context that is not a Wails one.
func holdingApp(t *testing.T, archive string) *App {
	t.Helper()
	app := &App{ctx: context.Background()}
	lock, _, err := booklock.Force(archive, "bk-1", app.deviceIdentity())
	if err != nil {
		t.Fatalf("claiming the book failed: %v", err)
	}
	app.device.lock = lock
	app.setCurrentFile(archive)
	return app
}

// askFor writes the request the other machine would have written, aimed at
// whoever holds the book.
func askFor(t *testing.T, archive string, id booklock.Identity) {
	t.Helper()
	if _, err := booklock.RequestTakeover(archive, id, booklock.Inspect(archive, id)); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
}

// The handover: the book is saved by then, so granting records what is on disk
// and drops the claim the other machine is watching for.
func TestGrantingHandsTheBookOverWithProof(t *testing.T) {
	archive := bookFile(t)
	app := holdingApp(t, archive)
	askFor(t, archive, asking())

	result := app.GrantBookTakeover()
	if !result.Granted || result.Error != "" {
		t.Fatalf("result = %+v, want a grant", result)
	}
	if !result.Fingerprinted {
		t.Fatal("a readable book must be handed over with a fingerprint")
	}
	if _, err := os.Stat(booklock.SidecarFor(archive)); !os.IsNotExist(err) {
		t.Fatal("the claim should be gone so the other machine can take it")
	}

	reply, ok := booklock.ReadTakeover(archive)
	if !ok || reply.Status != booklock.TakeoverGranted {
		t.Fatalf("reply = %+v, want a grant beside the book", reply)
	}
	if state := booklock.Arrival(archive, reply); !state.Arrived {
		t.Fatalf("the book beside the grant should verify: %+v", state)
	}
}

// The guard that matters most on this side: a writer who gave up takes their
// request with them, and the countdown that was already running must not save,
// release and close a book on a machine with nobody sitting at it.
func TestGrantingWithNobodyWaitingKeepsTheBook(t *testing.T) {
	archive := bookFile(t)
	app := holdingApp(t, archive)
	askFor(t, archive, asking())
	booklock.WithdrawTakeover(archive)

	result := app.GrantBookTakeover()
	if !result.Withdrawn {
		t.Fatalf("result = %+v, want the handover called off", result)
	}
	if result.Granted {
		t.Fatal("nothing should have been handed to nobody")
	}
	if _, err := os.Stat(booklock.SidecarFor(archive)); err != nil {
		t.Fatal("the claim should still be held")
	}
}

// Refusing keeps the book and the claim, and says so where the other machine
// will read it.
func TestDecliningKeepsTheBookAndTheClaim(t *testing.T) {
	archive := bookFile(t)
	app := holdingApp(t, archive)
	askFor(t, archive, asking())

	result := app.DeclineBookTakeover()
	if !result.Declined || result.Error != "" {
		t.Fatalf("result = %+v, want a refusal", result)
	}
	if _, err := os.Stat(booklock.SidecarFor(archive)); err != nil {
		t.Fatal("refusing must not give the claim up")
	}
	reply, ok := booklock.ReadTakeover(archive)
	if !ok || reply.Status != booklock.TakeoverDeclined {
		t.Fatalf("reply = %+v, want a refusal beside the book", reply)
	}
}

// A request aimed at a session that has gone must not be answered by whoever
// holds the book now.
func TestARequestForAnotherSessionIsNotAnswered(t *testing.T) {
	archive := bookFile(t)
	app := holdingApp(t, archive)

	// Aimed at the machine that held it before this session did.
	if _, err := booklock.RequestTakeover(archive, asking(), &booklock.Holder{
		Device: "SOME-OTHER-MACHINE", Session: "s-long-gone",
	}); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	if result := app.GrantBookTakeover(); !result.Withdrawn {
		t.Fatalf("result = %+v, want it left alone", result)
	}
	if _, err := os.Stat(booklock.SidecarFor(archive)); err != nil {
		t.Fatal("the claim should still be held")
	}
}

// The asking side, mid-sync: the grant has arrived because it is 300 bytes, the
// manuscript has not because it is not. This is what the waiting screen reads,
// and it must not call the book ready.
func TestStatusWaitsWhileTheCopyIsStillArriving(t *testing.T) {
	archive := bookFile(t)
	app := &App{ctx: context.Background()}

	if _, _, err := booklock.Force(archive, "bk-1", studio()); err != nil {
		t.Fatalf("seeding the claim failed: %v", err)
	}
	askFor(t, archive, app.deviceIdentity())
	if _, err := booklock.GrantTakeover(archive, studio()); err != nil {
		t.Fatalf("GrantTakeover: %v", err)
	}

	full, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, full[:len(full)/3], 0o644); err != nil {
		t.Fatal(err)
	}

	status := app.BookTakeoverStatus(archive)
	if !status.Granted || !status.Answered {
		t.Fatalf("status = %+v, want an answered grant", status)
	}
	if status.Arrived {
		t.Fatal("a third of a book must not be called arrived")
	}
	if status.ExpectedBytes <= status.LocalBytes {
		t.Fatalf("status = %+v, want it counting up towards the full size", status)
	}
	if !strings.Contains(status.Message, "sync") {
		t.Fatalf("Message = %q, want it to say what it is waiting for", status.Message)
	}

	// And once the bytes land, the same poll says so.
	if err := os.WriteFile(archive, full, 0o644); err != nil {
		t.Fatal(err)
	}
	if status := app.BookTakeoverStatus(archive); !status.Arrived {
		t.Fatalf("status = %+v, want the book ready to open", status)
	}
}

// A refusal has to reach the asking side as a refusal. Silence is a machine
// that is asleep, and the two must not read the same.
func TestStatusReportsARefusal(t *testing.T) {
	archive := bookFile(t)
	app := &App{ctx: context.Background()}

	if _, _, err := booklock.Force(archive, "bk-1", studio()); err != nil {
		t.Fatalf("seeding the claim failed: %v", err)
	}
	askFor(t, archive, app.deviceIdentity())

	if status := app.BookTakeoverStatus(archive); status.Answered {
		t.Fatalf("status = %+v, want it still waiting", status)
	}
	if _, err := booklock.DeclineTakeover(archive, studio()); err != nil {
		t.Fatalf("DeclineTakeover: %v", err)
	}

	status := app.BookTakeoverStatus(archive)
	if !status.Declined || !status.Answered {
		t.Fatalf("status = %+v, want a refusal", status)
	}
	if status.Arrived {
		t.Fatal("a refusal hands nothing over")
	}
}

// Asking for a book nobody has claimed is harmless, and the status says there
// is nothing to wait for.
func TestStatusOfABookNobodyAskedAbout(t *testing.T) {
	archive := bookFile(t)
	app := &App{ctx: context.Background()}
	status := app.BookTakeoverStatus(archive)
	if status.Asked || status.Answered || status.HeldElsewhere {
		t.Fatalf("status = %+v, want nothing going on", status)
	}
}
