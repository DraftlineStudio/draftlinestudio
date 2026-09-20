package booklock

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "The Weather House.draftline")
	if err := os.WriteFile(path, []byte("not really a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func laptop(session string) Identity {
	return Identity{Device: "OTHER-DESKTOP", Platform: "windows", App: "Draftline 0.21", Session: session}
}

// observer is somebody neither of the fixtures is: a third machine looking at
// the claim. Tests that want "is this ours" pass laptop() or phone() instead.
func observer(session string) Identity {
	return Identity{Device: "THIRD-MACHINE", Platform: "linux", App: "Draftline 0.21", Session: session}
}

func phone(session string) Identity {
	return Identity{Device: "Pixel", Platform: "android", App: "Draftline Mobile 0.1", Session: session}
}

// age rewrites a claim's heartbeat to look older than it is.
func age(t *testing.T, archive string, by time.Duration) {
	t.Helper()
	file := SidecarFor(archive)
	c, ok := read(file)
	if !ok {
		t.Fatal("no claim to age")
	}
	c.LastSeen = c.LastSeen.Add(-by)
	if _, err := write(file, c.BookID, Identity{
		Device: c.Device, Platform: c.Platform, App: c.App, Session: c.Session,
	}, c.OpenedAt, c.LastSeen); err != nil {
		t.Fatal(err)
	}
}

func TestSidecarSitsBesideTheBookAndIsHidden(t *testing.T) {
	got := SidecarFor("/books/novel.draftline")
	want := filepath.Join("/books", ".novel.draftline.lock")
	if got != want {
		t.Fatalf("SidecarFor = %q, want %q", got, want)
	}
}

func TestClaimingAFreeBookSucceeds(t *testing.T) {
	archive := fixture(t)
	lock, previous, err := Claim(archive, "bk-1", laptop("s1"))
	if err != nil || lock == nil {
		t.Fatalf("Claim: lock = %v, err = %v", lock, err)
	}
	if previous != nil {
		t.Fatalf("previous = %+v, want none", previous)
	}
	if _, err := os.Stat(SidecarFor(archive)); err != nil {
		t.Fatal("no sidecar was written")
	}
}

func TestTheSameSessionReclaimsItsOwnBook(t *testing.T) {
	archive := fixture(t)
	if _, _, err := Claim(archive, "bk-1", laptop("s1")); err != nil {
		t.Fatal(err)
	}
	lock, previous, err := Claim(archive, "bk-1", laptop("s1"))
	if err != nil || lock == nil {
		t.Fatalf("a session could not reclaim its own book: %v", err)
	}
	if previous != nil {
		t.Fatalf("reclaiming reported a takeover: %+v", previous)
	}
}

func TestALiveClaimFromAnotherDeviceIsReported(t *testing.T) {
	archive := fixture(t)
	if _, _, err := Claim(archive, "bk-1", phone("phone-session")); err != nil {
		t.Fatal(err)
	}

	lock, holder, err := Claim(archive, "bk-1", laptop("laptop-session"))
	if lock != nil {
		t.Fatal("the laptop took a book the phone had open")
	}
	if err != ErrHeldElsewhere {
		t.Fatalf("err = %v, want ErrHeldElsewhere", err)
	}
	if holder == nil || holder.Device != "Pixel" || holder.Stale || holder.Mine {
		t.Fatalf("holder = %+v, want a live claim by the phone", holder)
	}
	// The wording a writer sees must never state it as fact; sync cannot
	// promise that much.
	if got := holder.Describe(); got != "This book may be open on Pixel." {
		t.Fatalf("Describe = %q", got)
	}
}

func TestAnAbandonedClaimIsTakenOverAndReported(t *testing.T) {
	archive := fixture(t)
	if _, _, err := Claim(archive, "bk-1", phone("phone-session")); err != nil {
		t.Fatal(err)
	}
	age(t, archive, StaleAfter+time.Minute)

	lock, previous, err := Claim(archive, "bk-1", laptop("laptop-session"))
	if err != nil || lock == nil {
		t.Fatalf("an abandoned claim blocked the book: lock = %v, err = %v", lock, err)
	}
	if previous == nil || previous.Device != "Pixel" || !previous.Stale {
		t.Fatalf("previous = %+v, want the phone's abandoned claim", previous)
	}
	if holder := Inspect(archive, observer("laptop-session")); holder == nil || !holder.Mine {
		t.Fatalf("after taking over, Inspect = %+v, want the laptop's own claim", holder)
	}
}

func TestHeartbeatKeepsAClaimAlive(t *testing.T) {
	archive := fixture(t)
	lock, _, err := Claim(archive, "bk-1", phone("phone-session"))
	if err != nil {
		t.Fatal(err)
	}
	age(t, archive, StaleAfter+time.Minute)
	if holder := Inspect(archive, observer("other")); holder == nil || !holder.Stale {
		t.Fatal("the claim should have gone stale")
	}
	if err := lock.Heartbeat(); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if holder := Inspect(archive, observer("other")); holder == nil || holder.Stale {
		t.Fatal("a heartbeat did not bring the claim back to life")
	}
}

func TestForceTakesTheBookAndSaysWhoHadIt(t *testing.T) {
	archive := fixture(t)
	if _, _, err := Claim(archive, "bk-1", phone("phone-session")); err != nil {
		t.Fatal(err)
	}
	lock, previous, err := Force(archive, "bk-1", laptop("laptop-session"))
	if err != nil || lock == nil {
		t.Fatalf("Force: lock = %v, err = %v", lock, err)
	}
	if previous == nil || previous.Device != "Pixel" {
		t.Fatalf("previous = %+v, want the phone that was overridden", previous)
	}
	if holder := Inspect(archive, observer("laptop-session")); holder == nil || !holder.Mine {
		t.Fatalf("Force did not leave the laptop holding the book: %+v", holder)
	}
}

func TestReleaseFreesTheBook(t *testing.T) {
	archive := fixture(t)
	lock, _, err := Claim(archive, "bk-1", laptop("s1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if holder := Inspect(archive, observer("s1")); holder != nil {
		t.Fatalf("Inspect = %+v, want nothing after a release", holder)
	}
}

func TestReleaseDoesNotDeleteAnotherDevicesClaim(t *testing.T) {
	archive := fixture(t)
	lock, _, err := Claim(archive, "bk-1", laptop("laptop-session"))
	if err != nil {
		t.Fatal(err)
	}
	// The writer forced it open on the phone while the laptop still held it.
	if _, _, err := Force(archive, "bk-1", phone("phone-session")); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	holder := Inspect(archive, observer("phone-session"))
	if holder == nil || holder.Device != "Pixel" {
		t.Fatalf("the laptop's release deleted the phone's claim: %+v", holder)
	}
}

func TestInspectOfABookNobodyHasOpen(t *testing.T) {
	if holder := Inspect(fixture(t), observer("s1")); holder != nil {
		t.Fatalf("Inspect = %+v, want nothing", holder)
	}
}

func TestAnUnreadableSidecarIsTreatedAsNoClaim(t *testing.T) {
	archive := fixture(t)
	if err := os.WriteFile(SidecarFor(archive), []byte("{ this is not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if holder := Inspect(archive, observer("s1")); holder != nil {
		t.Fatalf("Inspect = %+v; a damaged sidecar must not lock a writer out", holder)
	}
	if lock, _, err := Claim(archive, "bk-1", laptop("s1")); err != nil || lock == nil {
		t.Fatalf("a damaged sidecar blocked a claim: %v", err)
	}
}
