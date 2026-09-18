// Package booklock says which device has a book open, across devices.
//
// HOW THIS DIFFERS FROM internal/instancelock
//
// instancelock stops one machine opening the same .draftline twice. It lives
// in the user's config directory, holds a process ID, and checks whether that
// process is alive, so it is exact: a crashed instance's lock is stolen and the
// writer never sees it. It cannot see another device at all.
//
// This is the other half. A book in a Dropbox or Drive folder can be open on a
// laptop and a phone at once, and neither sync client merges: the second device
// to save wins and the first device's afternoon is replaced. Nothing local can
// detect that, so the claim has to travel with the book — a small sidecar file
// beside the archive, which the sync client carries to the other device along
// with everything else in the folder.
//
// WHY IT IS ADVISORY, AND MUST STAY ADVISORY
//
// Sync is not a lock service. A sidecar written on a phone may take seconds or
// minutes to reach a laptop, and may arrive after that laptop has written its
// own. Two devices can hold what each believes is the only claim. There is no
// fix for that short of a server, so this never blocks and never promises: it
// reports what it knows, the caller warns, and the writer decides. Any wording
// built on this must say a book MAY be open elsewhere, never that it IS.
//
// A device that goes offline, sleeps, or is put in a drawer stops refreshing
// its claim. A claim nobody has refreshed for StaleAfter is treated as
// abandoned and can be taken, because the alternative is a book locked forever
// by a phone that was reset.
package booklock

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"draftline/internal/fsutil"
)

const claimVersion = 1

// StaleAfter is how long a claim survives without being refreshed. It is long
// because the clocks on two devices disagree, sync is slow, and a laptop lid
// closed over lunch is not an abandoned book.
const StaleAfter = 15 * time.Minute

// HeartbeatEvery is how often a holder should refresh its claim. Well inside
// StaleAfter, so a device has to miss several in a row to look abandoned.
const HeartbeatEvery = 2 * time.Minute

// Identity is the device making a claim.
type Identity struct {
	// Device is what a writer will be shown: a hostname, because that is the
	// name they gave the machine themselves.
	Device string
	// Platform is windows, darwin, linux, android or ios.
	Platform string
	// App names the build, so "Draftline Mobile 0.1.0" can be distinguished
	// from the desktop in the warning.
	App string
	// Session is unique to one opening of one book, and is how a device
	// recognises its own claim after a restart rather than stealing it.
	Session string
}

// ThisDevice fills in what the operating system can answer for itself.
func ThisDevice(app, session string) Identity {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		host = "this device"
	}
	return Identity{Device: host, Platform: runtime.GOOS, App: app, Session: session}
}

// claim is the sidecar's contents.
type claim struct {
	Version  int       `json:"version"`
	BookID   string    `json:"book_id"`
	Device   string    `json:"device"`
	Platform string    `json:"platform"`
	App      string    `json:"app"`
	Session  string    `json:"session"`
	OpenedAt time.Time `json:"opened_at"`
	LastSeen time.Time `json:"last_seen"`
}

// Holder describes a claim on a book, for showing to a writer.
type Holder struct {
	Device   string
	Platform string
	App      string
	Session  string
	OpenedAt time.Time
	LastSeen time.Time
	// Stale means nobody has refreshed this claim for StaleAfter, so the
	// device holding it is probably not running any more.
	Stale bool
	// Mine means this claim belongs to the session asking about it.
	Mine bool
}

// Lock is a held claim. Heartbeat it while the book is open and Release it
// when it closes.
type Lock struct {
	file  string
	state claim
}

// ErrHeldElsewhere is returned when a live claim from another device is in the
// way. It is a fact to report, not a failure to retry.
var ErrHeldElsewhere = errors.New("this book is open on another device")

// SidecarFor is where a book's claim lives: beside the archive, so it travels
// with it through whatever is syncing the folder.
//
// The dot keeps it out of the way on macOS and Linux. It is deliberately not
// inside the archive: writing a claim into the ZIP would mean rewriting the
// whole book merely to open it, and every device that opened it would produce
// a new version of the file for the sync client to carry.
func SidecarFor(archivePath string) string {
	dir := filepath.Dir(archivePath)
	return filepath.Join(dir, "."+filepath.Base(archivePath)+".lock")
}

func read(file string) (claim, bool) {
	data, err := os.ReadFile(file)
	if err != nil {
		return claim{}, false
	}
	var c claim
	if err := json.Unmarshal(data, &c); err != nil || c.Version != claimVersion {
		return claim{}, false
	}
	return c, true
}

func (c claim) holder(now time.Time, session string) *Holder {
	return &Holder{
		Device:   c.Device,
		Platform: c.Platform,
		App:      c.App,
		Session:  c.Session,
		OpenedAt: c.OpenedAt,
		LastSeen: c.LastSeen,
		Stale:    now.Sub(c.LastSeen) > StaleAfter,
		Mine:     c.Session != "" && c.Session == session,
	}
}

// Inspect reports the current claim on a book without making one, which is
// what a library listing wants so it can show a marker beside a book that is
// open somewhere else.
func Inspect(archivePath, session string) *Holder {
	c, ok := read(SidecarFor(archivePath))
	if !ok {
		return nil
	}
	return c.holder(time.Now().UTC(), session)
}

// Claim takes the claim on a book.
//
// Outcomes:
//   - (lock, nil, nil)        nothing was in the way, or the claim was already
//     this session's.
//   - (lock, previous, nil)   an abandoned claim was taken over. previous says
//     whose it was, so the caller can mention it.
//   - (nil, holder, ErrHeldElsewhere) a live claim from another device is in
//     the way. The caller warns and offers Force or a copy.
//   - (nil, nil, err)         the sidecar could not be written. Callers should
//     open the book anyway: a lock that cannot be
//     written is not a reason to keep someone from
//     their manuscript.
func Claim(archivePath, bookID string, id Identity) (*Lock, *Holder, error) {
	file := SidecarFor(archivePath)
	now := time.Now().UTC()

	if existing, ok := read(file); ok {
		holder := existing.holder(now, id.Session)
		if !holder.Mine && !holder.Stale {
			return nil, holder, ErrHeldElsewhere
		}
		lock, err := write(file, bookID, id, existing.OpenedAt, now)
		if err != nil {
			return nil, nil, err
		}
		if holder.Mine {
			return lock, nil, nil
		}
		return lock, holder, nil
	}

	lock, err := write(file, bookID, id, now, now)
	if err != nil {
		return nil, nil, err
	}
	return lock, nil, nil
}

// Force takes the claim regardless of who holds it. This is the writer saying
// they understand the warning. Whoever was there is returned so the caller can
// record what was overridden.
func Force(archivePath, bookID string, id Identity) (*Lock, *Holder, error) {
	file := SidecarFor(archivePath)
	now := time.Now().UTC()
	var previous *Holder
	if existing, ok := read(file); ok {
		previous = existing.holder(now, id.Session)
	}
	lock, err := write(file, bookID, id, now, now)
	if err != nil {
		return nil, previous, err
	}
	return lock, previous, nil
}

func write(file, bookID string, id Identity, openedAt, now time.Time) (*Lock, error) {
	state := claim{
		Version:  claimVersion,
		BookID:   bookID,
		Device:   id.Device,
		Platform: id.Platform,
		App:      id.App,
		Session:  id.Session,
		OpenedAt: openedAt,
		LastSeen: now,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("booklock: encode claim: %w", err)
	}
	if err := fsutil.WriteFileAtomic(file, data, 0o644); err != nil {
		return nil, fmt.Errorf("booklock: write claim: %w", err)
	}
	return &Lock{file: file, state: state}, nil
}

// Heartbeat refreshes a held claim so other devices keep seeing it as live.
// Call it on HeartbeatEvery while a book is open.
func (l *Lock) Heartbeat() error {
	if l == nil {
		return nil
	}
	l.state.LastSeen = time.Now().UTC()
	data, err := json.MarshalIndent(l.state, "", "  ")
	if err != nil {
		return fmt.Errorf("booklock: encode claim: %w", err)
	}
	if err := fsutil.WriteFileAtomic(l.file, data, 0o644); err != nil {
		return fmt.Errorf("booklock: refresh claim: %w", err)
	}
	return nil
}

// Release gives the book up. A claim left behind by a crash is not a disaster
// — it goes stale — but releasing cleanly means the other device sees the book
// as free immediately rather than in a quarter of an hour.
//
// Another device's claim is never deleted: if the sidecar has been taken over
// while this session held it, the takeover stands.
func (l *Lock) Release() error {
	if l == nil {
		return nil
	}
	if current, ok := read(l.file); ok && current.Session != l.state.Session {
		return nil
	}
	if err := os.Remove(l.file); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("booklock: release claim: %w", err)
	}
	return nil
}

// Describe puts a holder into the words a writer is shown. It is deliberately
// hedged: see the package comment on why this can never be a statement of
// fact.
func (h *Holder) Describe() string {
	if h == nil {
		return ""
	}
	where := strings.TrimSpace(h.Device)
	if where == "" {
		where = "another device"
	}
	if h.Stale {
		return fmt.Sprintf("%s had this book open, and has not been heard from since %s.",
			where, h.LastSeen.Local().Format("3:04 PM on 2 Jan"))
	}
	return fmt.Sprintf("This book may be open on %s.", where)
}
