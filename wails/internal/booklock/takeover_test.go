package booklock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// desktop is the machine that keeps the book open and forgets to close it: the
// whole reason the handshake exists.
func desktop(session string) Identity {
	return Identity{Device: "STUDIO-DESKTOP", Platform: "windows", App: "Draftline 0.21", Session: session}
}

// hold claims a book for an identity and hands back the claim, so a test can
// ask for a takeover against a real holder rather than a made-up one.
func hold(t *testing.T, archive string, id Identity) *Holder {
	t.Helper()
	if _, _, err := Claim(archive, "bk-1", id); err != nil {
		t.Fatalf("claiming the book failed: %v", err)
	}
	holder := Inspect(archive, observer("s-observer"))
	if holder == nil {
		t.Fatal("the claim just written cannot be read back")
	}
	return holder
}

// backdate rewrites when a request says it was made.
func backdate(t *testing.T, archive string, by time.Duration) {
	t.Helper()
	request, ok := ReadTakeover(archive)
	if !ok {
		t.Fatal("no request to backdate")
	}
	request.RequestedAt = request.RequestedAt.Add(-by)
	if err := writeTakeover(archive, request); err != nil {
		t.Fatal(err)
	}
}

// The sidecar has to survive the trip through OneDrive, MEGA and the rest, all
// of which skip names beginning with "." or "~". Same rule as the claim.
func TestTakeoverSidecarNameIsSyncable(t *testing.T) {
	got := TakeoverSidecarFor("/books/novel.draftline")
	want := filepath.Join("/books", "novel.draftline.takeover")
	if got != want {
		t.Fatalf("TakeoverSidecarFor = %q, want %q", got, want)
	}
	base := filepath.Base(got)
	if strings.HasPrefix(base, ".") || strings.HasPrefix(base, "~") {
		t.Fatalf("sync clients will not upload %q", base)
	}
}

// The request must not be written into the claim. Two machines writing one file
// through a sync client is how a conflict copy of the claim appears.
func TestAskingLeavesTheClaimAlone(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop"))

	before, err := os.ReadFile(SidecarFor(archive))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	after, err := os.ReadFile(SidecarFor(archive))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("asking for the book rewrote the claim")
	}
}

// The ordinary success: the desktop grants, and the laptop can prove the bytes
// beside it are the ones the desktop left.
func TestGrantedBookVerifiesOnceTheBytesMatch(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop"))
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}

	pending := PendingTakeover(archive, desktop("s-desktop"))
	if pending == nil {
		t.Fatal("the desktop cannot see the request")
	}
	if _, err := GrantTakeover(archive, desktop("s-desktop")); err != nil {
		t.Fatalf("GrantTakeover: %v", err)
	}

	reply, ok := ReadTakeover(archive)
	if !ok || reply.Status != TakeoverGranted {
		t.Fatalf("reply = %+v, want a grant", reply)
	}
	if !reply.Fingerprinted() {
		t.Fatalf("a grant off a readable book must carry a fingerprint: %+v", reply)
	}
	state := Arrival(archive, reply)
	if !state.Arrived {
		t.Fatalf("the book beside the grant should verify: %+v", state)
	}
}

// The failure this whole feature exists for: the claim is 300 bytes and lands
// first, the manuscript is still crossing, and the stale copy on this machine
// opens perfectly and is yesterday's book.
func TestAStaleCopyDoesNotVerify(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop"))
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	if _, err := GrantTakeover(archive, desktop("s-desktop")); err != nil {
		t.Fatalf("GrantTakeover: %v", err)
	}
	reply, _ := ReadTakeover(archive)

	// What the sync client has not replaced yet: the same length, different
	// words. Size alone would call this arrived.
	current, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	stale := []byte(strings.Repeat("x", len(current)))
	if err := os.WriteFile(archive, stale, 0o644); err != nil {
		t.Fatal(err)
	}

	state := Arrival(archive, reply)
	if state.Arrived {
		t.Fatal("a copy the other machine did not write must not verify")
	}
	if state.Unverifiable {
		t.Fatal("the grant carried a fingerprint, so this is checkable")
	}
}

// Half a manuscript is reported as progress, not as a failure: it is what the
// waiting screen counts up.
func TestPartialArrivalReportsHowFarItHasGot(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop"))
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	// A bigger book than the one this machine has.
	full := []byte(strings.Repeat("manuscript", 500))
	if err := os.WriteFile(archive, full, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := GrantTakeover(archive, desktop("s-desktop")); err != nil {
		t.Fatalf("GrantTakeover: %v", err)
	}
	reply, _ := ReadTakeover(archive)

	if err := os.WriteFile(archive, full[:len(full)/4], 0o644); err != nil {
		t.Fatal(err)
	}
	state := Arrival(archive, reply)
	if state.Arrived {
		t.Fatal("a quarter of a book must not verify")
	}
	if state.ExpectedBytes != int64(len(full)) {
		t.Fatalf("ExpectedBytes = %d, want %d", state.ExpectedBytes, len(full))
	}
	if state.LocalBytes != int64(len(full)/4) {
		t.Fatalf("LocalBytes = %d, want %d", state.LocalBytes, len(full)/4)
	}
}

// A sync client holding the book open defeats the hash. The handover still
// happens, and the other side is told it has no proof rather than being handed
// a guarantee nobody checked.
func TestGrantWithoutAFingerprintSaysSo(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop"))
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	if err := os.Remove(archive); err != nil {
		t.Fatal(err)
	}
	if _, err := GrantTakeover(archive, desktop("s-desktop")); err != nil {
		t.Fatalf("GrantTakeover: %v", err)
	}

	reply, ok := ReadTakeover(archive)
	if !ok || reply.Status != TakeoverGranted {
		t.Fatalf("reply = %+v, want a grant", reply)
	}
	if reply.Fingerprinted() {
		t.Fatal("there was nothing to fingerprint")
	}
	if reply.Note == "" {
		t.Fatal("a grant with no fingerprint has to say why")
	}
	if state := Arrival(archive, reply); !state.Unverifiable {
		t.Fatalf("state = %+v, want unverifiable", state)
	}
}

// A refusal has to reach the asking device as a refusal. Silence means the
// other machine is asleep, and the two are not the same answer.
func TestDeclineIsReadable(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop"))
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	if _, err := DeclineTakeover(archive, desktop("s-desktop")); err != nil {
		t.Fatalf("DeclineTakeover: %v", err)
	}
	reply, ok := ReadTakeover(archive)
	if !ok || reply.Status != TakeoverDeclined {
		t.Fatalf("reply = %+v, want a decline", reply)
	}
	if reply.Responder != "STUDIO-DESKTOP" {
		t.Fatalf("Responder = %q, want the machine that refused", reply.Responder)
	}
}

// The guard against handing a book to nobody. A writer who cancels takes the
// request with them, and the countdown on the other machine must find it gone
// and stop rather than save, release and close a book nobody asked about.
func TestAWithdrawnRequestIsNoLongerPending(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop"))
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	if PendingTakeover(archive, desktop("s-desktop")) == nil {
		t.Fatal("the request should be pending before it is withdrawn")
	}
	WithdrawTakeover(archive)
	if pending := PendingTakeover(archive, desktop("s-desktop")); pending != nil {
		t.Fatalf("pending = %+v, want nothing to grant to", pending)
	}
}

// A grant is kept beside the book on purpose: a writer who gives up and comes
// back later resumes the same wait, with the same proof. A hash does not stale.
func TestAGrantOutlivesTheWait(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop"))
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	if _, err := GrantTakeover(archive, desktop("s-desktop")); err != nil {
		t.Fatalf("GrantTakeover: %v", err)
	}
	backdate(t, archive, 4*TakeoverExpires)

	reply, ok := ReadTakeover(archive)
	if !ok || reply.Status != TakeoverGranted {
		t.Fatalf("reply = %+v, want the grant still there", reply)
	}
	if state := Arrival(archive, reply); !state.Arrived {
		t.Fatalf("an old grant over matching bytes still verifies: %+v", state)
	}
}

// An answered request is nobody's business any more; it is a receipt, not a
// question.
func TestAnAnsweredRequestIsNotPending(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop"))
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	if _, err := GrantTakeover(archive, desktop("s-desktop")); err != nil {
		t.Fatalf("GrantTakeover: %v", err)
	}
	if pending := PendingTakeover(archive, desktop("s-desktop")); pending != nil {
		t.Fatalf("pending = %+v, want nothing", pending)
	}
}

// A device never answers itself. The laptop's own request must not raise a
// countdown on the laptop.
func TestNobodyAnswersTheirOwnRequest(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop"))
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	if pending := PendingTakeover(archive, laptop("s-laptop")); pending != nil {
		t.Fatalf("pending = %+v, want the asker to ignore itself", pending)
	}
}

// A request is addressed to the machine, not to the process. The session that
// was asked ends on every restart — constantly under a dev server, and on any
// crash, update or quit while a request is in flight — and the session that
// reopens the same book on the same machine is who the writer meant.
//
// A different machine is still not the addressee.
func TestTheMachineThatWasAskedAnswersAcrossARestart(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop-first"))
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	if PendingTakeover(archive, desktop("s-desktop-first")) == nil {
		t.Fatal("the session that was asked should see it")
	}
	if PendingTakeover(archive, desktop("s-desktop-restarted")) == nil {
		t.Fatal("the same machine after a restart should still see it")
	}
	if pending := PendingTakeover(archive, phone("s-phone")); pending != nil {
		t.Fatalf("pending = %+v, want a bystander to ignore it", pending)
	}
}

// The backstop behind the session check: a machine that was asleep does not
// wake up and hand its book to somebody who left hours ago.
func TestAnAncientRequestIsIgnored(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop"))
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	backdate(t, archive, TakeoverExpires+time.Minute)
	if pending := PendingTakeover(archive, desktop("s-desktop")); pending != nil {
		t.Fatalf("pending = %+v, want it expired", pending)
	}
}

// A clock ahead of ours makes a request look like it was made in the future.
// That is skew, not staleness, and refusing it would make the feature depend on
// two machines agreeing about the time.
func TestARequestFromAFastClockIsStillHonoured(t *testing.T) {
	archive := fixture(t)
	holder := hold(t, archive, desktop("s-desktop"))
	if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
		t.Fatalf("RequestTakeover: %v", err)
	}
	backdate(t, archive, -20*time.Minute)
	if PendingTakeover(archive, desktop("s-desktop")) == nil {
		t.Fatal("a request from a machine whose clock runs fast must still be seen")
	}
}

// Answering when nobody asked is a bug in the caller, not a silent no-op.
func TestAnsweringNothingIsAnError(t *testing.T) {
	archive := fixture(t)
	if _, err := GrantTakeover(archive, desktop("s-desktop")); err == nil {
		t.Fatal("granting with no request should fail")
	}
	if _, err := DeclineTakeover(archive, desktop("s-desktop")); err == nil {
		t.Fatal("declining with no request should fail")
	}
}

// Every way a request can be passed over is silent from the outside: the file
// is in the folder and nothing happens. Each one has to be able to say why, or
// a working feature and a broken one look identical.
func TestPassingOverARequestSaysWhy(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, archive string)
		self  Identity
		want  string
	}{
		{
			name:  "already answered",
			setup: func(t *testing.T, archive string) { mustDecline(t, archive) },
			self:  desktop("s-desktop"),
			want:  "already answered",
		},
		{
			name:  "our own request",
			setup: func(t *testing.T, archive string) {},
			self:  laptop("s-laptop"),
			want:  "own request",
		},
		{
			name:  "asked somebody else",
			setup: func(t *testing.T, archive string) {},
			self:  phone("s-phone"),
			want:  "and this is",
		},
		{
			name:  "expired",
			setup: func(t *testing.T, archive string) { backdate(t, archive, TakeoverExpires+time.Minute) },
			self:  desktop("s-desktop"),
			want:  "check the clocks",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archive := fixture(t)
			holder := hold(t, archive, desktop("s-desktop"))
			if _, err := RequestTakeover(archive, laptop("s-laptop"), holder); err != nil {
				t.Fatalf("RequestTakeover: %v", err)
			}
			tc.setup(t, archive)

			request, why := PendingTakeoverReason(archive, tc.self)
			if request != nil {
				t.Fatalf("request = %+v, want it passed over", request)
			}
			if !strings.Contains(why, tc.want) {
				t.Fatalf("reason = %q, want it to mention %q", why, tc.want)
			}
		})
	}
}

// Nothing beside the book is not a thing to explain; it is the normal state.
func TestNoRequestIsNotAReason(t *testing.T) {
	archive := fixture(t)
	hold(t, archive, desktop("s-desktop"))
	if request, why := PendingTakeoverReason(archive, desktop("s-desktop")); request != nil || why != "" {
		t.Fatalf("request = %+v, reason = %q, want silence", request, why)
	}
}

func mustDecline(t *testing.T, archive string) {
	t.Helper()
	if _, err := DeclineTakeover(archive, desktop("s-desktop")); err != nil {
		t.Fatal(err)
	}
}

// The holder says what it has the moment somebody asks, so a machine that is
// never answered can still check its own copy before taking the book.
func TestAStampedClaimLetsTheAskerConfirmItsCopy(t *testing.T) {
	archive := fixture(t)
	lock, _, err := Claim(archive, "bk-1", desktop("s-desktop"))
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if holder := Inspect(archive, laptop("s-laptop")); holder.Stamped() {
		t.Fatal("a claim says nothing about the book until it is asked for")
	}
	if err := lock.StampClaim(archive); err != nil {
		t.Fatalf("StampClaim: %v", err)
	}

	holder := Inspect(archive, laptop("s-laptop"))
	if !holder.Stamped() {
		t.Fatalf("holder = %+v, want it to say what it has", holder)
	}
	if state := ConfirmedCurrent(archive, holder); !state.Arrived {
		t.Fatalf("state = %+v, want the copy confirmed", state)
	}
}

// The stamp has to survive the heartbeat. The heartbeat rewrites the whole
// claim from memory, so a stamp written only to disk would be erased two
// minutes later and the asking machine would wait forever.
func TestAStampSurvivesTheHeartbeat(t *testing.T) {
	archive := fixture(t)
	lock, _, err := Claim(archive, "bk-1", desktop("s-desktop"))
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if err := lock.StampClaim(archive); err != nil {
		t.Fatalf("StampClaim: %v", err)
	}
	if err := lock.Heartbeat(); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if holder := Inspect(archive, laptop("s-laptop")); !holder.Stamped() {
		t.Fatalf("holder = %+v, want the stamp kept", holder)
	}
}

// A copy the sync client has not finished replacing must not confirm, and a
// holder that never said anything gives nothing to confirm against.
func TestAnUnsyncedOrUnstampedCopyIsNotConfirmed(t *testing.T) {
	archive := fixture(t)
	lock, _, err := Claim(archive, "bk-1", desktop("s-desktop"))
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if err := lock.StampClaim(archive); err != nil {
		t.Fatalf("StampClaim: %v", err)
	}

	// Same length, different words: what a half-finished sync leaves behind.
	current, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, []byte(strings.Repeat("x", len(current))), 0o644); err != nil {
		t.Fatal(err)
	}
	holder := Inspect(archive, laptop("s-laptop"))
	if state := ConfirmedCurrent(archive, holder); state.Arrived {
		t.Fatal("a copy the holder did not write must not confirm")
	}

	// And with no stamp at all there is nothing to check against, which is not
	// the same as failing the check.
	if state := ConfirmedCurrent(archive, &Holder{Device: "STUDIO-DESKTOP"}); !state.Unverifiable {
		t.Fatalf("state = %+v, want unverifiable", state)
	}
}
