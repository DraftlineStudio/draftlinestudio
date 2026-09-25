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

// The lid-close case: the claim went stale while this machine slept, another
// device took the book, and on waking this session must notice it is no longer
// the holder.
func TestWakingToAStolenBookStandsDown(t *testing.T) {
	archive := bookFile(t)
	app := holdingApp(t, archive)

	// What the machine wakes up to: somebody else's claim on its open book.
	if _, _, err := booklock.Force(archive, "bk-1", studio()); err != nil {
		t.Fatalf("seeding the other claim failed: %v", err)
	}

	if holder := app.claimTakenElsewhere(archive); holder != nil {
		t.Fatalf("one look is not enough to close a book: %+v", holder)
	}
	holder := app.claimTakenElsewhere(archive)
	if holder == nil {
		t.Fatal("two looks agreeing should stand the session down")
	}
	if holder.Device != "STUDIO-DESKTOP" {
		t.Fatalf("Device = %q, want whoever has it now", holder.Device)
	}
	// Standing down must not disturb the claim that is now somebody else's.
	if current := booklock.Inspect(archive, studio()); current == nil || !current.Mine {
		t.Fatalf("claim = %+v, want the new holder's claim untouched", current)
	}
	// And it says so once, not on every tick.
	if again := app.claimTakenElsewhere(archive); again != nil {
		t.Fatal("the writer should be told once")
	}
}

// A sidecar that cannot be read is not evidence of anything. A sync client
// mid-write looks exactly like this, and closing a book over it would lose
// somebody their afternoon.
func TestAMissingClaimDoesNotCloseTheBook(t *testing.T) {
	archive := bookFile(t)
	app := holdingApp(t, archive)
	if err := os.Remove(booklock.SidecarFor(archive)); err != nil {
		t.Fatal(err)
	}
	for look := 0; look < 5; look++ {
		if holder := app.claimTakenElsewhere(archive); holder != nil {
			t.Fatalf("a missing claim must not stand the session down: %+v", holder)
		}
	}
}

// One odd look followed by our own claim again is a hiccup, not a takeover.
func TestASingleOddLookIsForgotten(t *testing.T) {
	archive := bookFile(t)
	app := holdingApp(t, archive)

	if _, _, err := booklock.Force(archive, "bk-1", studio()); err != nil {
		t.Fatal(err)
	}
	if holder := app.claimTakenElsewhere(archive); holder != nil {
		t.Fatal("one look should not be acted on")
	}
	// The claim comes back as ours, as it would when the sync client finishes.
	if _, _, err := booklock.Force(archive, "bk-1", app.deviceIdentity()); err != nil {
		t.Fatal(err)
	}
	if holder := app.claimTakenElsewhere(archive); holder != nil {
		t.Fatalf("our own claim is not a takeover: %+v", holder)
	}
	// And the count started over, so the next odd look is a first look again.
	if _, _, err := booklock.Force(archive, "bk-1", studio()); err != nil {
		t.Fatal(err)
	}
	if holder := app.claimTakenElsewhere(archive); holder != nil {
		t.Fatal("the run of agreeing looks should have restarted")
	}
}

// Ask, be refused, ask again. The second request has to reach the other machine:
// a refusal is an answer to one question, not a standing one.
func TestAskingAgainAfterARefusalIsAnnouncedAgain(t *testing.T) {
	archive := bookFile(t)
	app := holdingApp(t, archive)
	asker := asking()

	askFor(t, archive, asker)
	if first := app.takeoverToAnnounce(archive); first == nil {
		t.Fatal("the first request was not announced")
	}
	if again := app.takeoverToAnnounce(archive); again != nil {
		t.Fatal("the same request should be announced once, not every five seconds")
	}
	if result := app.DeclineBookTakeover(); !result.Declined {
		t.Fatalf("result = %+v, want a refusal", result)
	}

	// The asking machine gives up on the refusal and asks again. Same machine,
	// same running application, same session: only the time has moved.
	booklock.WithdrawTakeover(archive)
	askFor(t, archive, asker)

	second := app.takeoverToAnnounce(archive)
	if second == nil {
		t.Fatal("a second request from the same application must be announced too")
	}
	if second.Device != asker.Device {
		t.Fatalf("Device = %q, want the machine that asked again", second.Device)
	}
}

// Two asks in a row without a refusal in between are still two asks.
func TestAskingTwiceRunningIsAnnouncedTwice(t *testing.T) {
	archive := bookFile(t)
	app := holdingApp(t, archive)

	askFor(t, archive, asking())
	if app.takeoverToAnnounce(archive) == nil {
		t.Fatal("the first request was not announced")
	}
	askFor(t, archive, asking())
	if app.takeoverToAnnounce(archive) == nil {
		t.Fatal("asking again must reach the writer")
	}
}
