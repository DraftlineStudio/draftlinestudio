// Package booklock says which device has a book open, using a sidecar beside
// the archive that the author's sync client carries to their other machines.
//
// Advisory and must stay so: sync is not a lock service, two devices can each
// hold what looks like the only claim, so the wording says a book MAY be open
// elsewhere. instancelock answers the same-machine question exactly.
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

// StaleAfter is how long a claim survives unrefreshed.
const StaleAfter = 15 * time.Minute

// HeartbeatEvery is how often a holder refreshes its claim.
const HeartbeatEvery = 2 * time.Minute

// Identity is the device making a claim.
type Identity struct {
	Device   string
	Platform string
	App      string
	Session  string
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
	Stale    bool
	Mine     bool
}

// Lock is a held claim.
type Lock struct {
	file  string
	state claim
}

// ErrHeldElsewhere means a live claim from another device is in the way.
var ErrHeldElsewhere = errors.New("this book is open on another device")

// SidecarFor is where a book's claim lives: beside the archive, so it travels
// with it through whatever is syncing the folder.
//
// The name is deliberately plain. Originally dotfiles were used, however all of the
// name-brand sync services refuse to upload dotfiles so now we hide best-effort
// per platform to avoid last-write-wins data loss with sync providers.
func SidecarFor(archivePath string) string {
	dir := filepath.Dir(archivePath)
	return filepath.Join(dir, filepath.Base(archivePath)+".lock")
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

func (c claim) holder(now time.Time, self Identity) *Holder {
	return &Holder{
		Device:   c.Device,
		Platform: c.Platform,
		App:      c.App,
		Session:  c.Session,
		OpenedAt: c.OpenedAt,
		LastSeen: c.LastSeen,
		Stale:    now.Sub(c.LastSeen) > StaleAfter,
		Mine:     isSelf(c, self),
	}
}

// isSelf reports whether a claim is this machine's own. instancelock answers
// the same-device question properly, so a claim naming this device is not a
// warning to raise.
func isSelf(c claim, self Identity) bool {
	if c.Session != "" && c.Session == self.Session {
		return true
	}
	return c.Device != "" && strings.EqualFold(c.Device, self.Device)
}

// Inspect reports the current claim without making one.
func Inspect(archivePath string, self Identity) *Holder {
	c, ok := read(SidecarFor(archivePath))
	if !ok {
		return nil
	}
	return c.holder(time.Now().UTC(), self)
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

	if existing, ok := read(SidecarFor(archivePath)); ok {
		holder := existing.holder(now, id)
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
	if existing, ok := read(SidecarFor(archivePath)); ok {
		previous = existing.holder(now, id)
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
	// The atomic write renames a fresh temp file into place, so the hidden
	// attribute has to be reapplied every time rather than set once.
	hide(file)
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
	hide(l.file)
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
