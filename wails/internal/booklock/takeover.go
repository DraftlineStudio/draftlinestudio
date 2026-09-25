package booklock

// The handshake that lets the writer's other machine ask for a book instead of
// forcing past the claim on it.
//
// Force is the blunt answer and stays: it takes the claim and leaves the other
// device none the wiser. This is the polite one. The asking device writes a
// request beside the claim, the holder sees it within a few seconds, saves, and
// either hands the book over or refuses. Nothing here is faster than the folder
// is synced, so every wait this produces in the UI is a wait on OneDrive or
// Dropbox rather than on the protocol.
//
// The request is its OWN sidecar rather than a field on the claim, so each file
// keeps a single writer. Two machines writing one JSON through a sync client is
// how you get a conflict copy of the very file that says who owns the book.
//
// A grant carries a fingerprint of the archive as the holder left it, and that
// is the point of the whole ceremony. The asking device cannot see whether the
// sync client has finished carrying a hundred megabytes of manuscript across,
// but it can hash what it has and compare. Bytes that match a hash the other
// machine wrote are proof. A file that has stopped growing is a guess.
//
// The request outlives the wait on purpose. A writer who gives up and comes
// back an hour later finds the grant still beside the book, fingerprint and
// all, and the open picks up where it left off. A hash does not go stale.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"draftline/internal/fsutil"
)

const takeoverVersion = 1

// TakeoverExpires bounds how long an UNANSWERED request is honoured, so a
// machine that was asleep cannot wake up and hand its book to somebody who
// stopped waiting long ago.
//
// It is a backstop and not the real guard. A request names the session it is
// asking (see Asks), and a session that has just opened the book has a new
// identifier, so a request left over from before this session is refused on
// identity rather than on the clock. What is left for a duration to catch is
// the asking device dying mid-wait, which is why this can afford to be loose:
// the comparison is between two machines' clocks, and those disagree.
const TakeoverExpires = 10 * time.Minute

// Takeover statuses. An empty status is a request nobody has answered.
const (
	TakeoverAsked    = ""
	TakeoverGranted  = "granted"
	TakeoverDeclined = "declined"
)

// Takeover is the request sidecar, and after an answer, the reply too.
//
// The asking device writes everything above Status. The holder writes Status
// and everything below it, once.
type Takeover struct {
	Version     int       `json:"version"`
	Device      string    `json:"device"`
	Platform    string    `json:"platform"`
	App         string    `json:"app"`
	Session     string    `json:"session"`
	RequestedAt time.Time `json:"requested_at"`

	// Who the request is aimed at, copied from the claim that was in the way.
	// A holder answers only for itself: a claim taken over in the meantime, or
	// a session that has reopened the book since, is not who was asked.
	TargetDevice  string `json:"target_device,omitempty"`
	TargetSession string `json:"target_session,omitempty"`

	Status      string    `json:"status,omitempty"`
	Responder   string    `json:"responder,omitempty"`
	RespondedAt time.Time `json:"responded_at,omitempty"`

	// The archive as the holder left it. Absent when the hash could not be
	// taken, which a sync client holding the file open is enough to cause.
	ArchiveSize   int64     `json:"archive_size,omitempty"`
	ArchiveSHA256 string    `json:"archive_sha256,omitempty"`
	SavedAt       time.Time `json:"saved_at,omitempty"`
	Note          string    `json:"note,omitempty"`
}

// TakeoverSidecarFor is where a request lives: beside the claim, under a name
// sync clients will carry. See SidecarFor on why it is not a dotfile.
func TakeoverSidecarFor(archivePath string) string {
	dir := filepath.Dir(archivePath)
	return filepath.Join(dir, filepath.Base(archivePath)+".takeover")
}

// Fingerprinted reports whether a grant carries proof of what was saved.
func (t *Takeover) Fingerprinted() bool {
	return t != nil && t.ArchiveSize > 0 && t.ArchiveSHA256 != ""
}

// Asks reports whether a request is aimed at this holder.
//
// The session is the precise answer and is tried first. It cannot be the only
// answer: a request names the session that held the book at the moment it was
// written, and that session ends every time the application restarts — which
// during development is every few minutes, and in ordinary use is any crash,
// update or quit-and-reopen while a request is in flight. The new session
// reclaims the same book on the same machine and would ignore the request for
// good, because it is addressed to a process that no longer exists.
//
// So a request for this DEVICE is a request for the book this device has open.
// What stops an abandoned request being acted on much later is not the session:
// it is TakeoverExpires, and the asking side withdrawing.
func (t *Takeover) Asks(self Identity) bool {
	if t == nil {
		return false
	}
	if t.TargetSession != "" && t.TargetSession == self.Session {
		return true
	}
	return t.TargetDevice == "" || strings.EqualFold(t.TargetDevice, self.Device)
}

// Mine reports whether this session wrote the request, so a device does not
// answer itself.
func (t *Takeover) Mine(self Identity) bool {
	return t != nil && t.Session != "" && t.Session == self.Session
}

// RequestTakeover asks whoever holds a book to hand it over. holder is the
// claim that was in the way, so the right session answers; it may be nil, in
// which case whoever is there will do.
func RequestTakeover(archivePath string, id Identity, holder *Holder) (*Takeover, error) {
	t := &Takeover{
		Version:     takeoverVersion,
		Device:      id.Device,
		Platform:    id.Platform,
		App:         id.App,
		Session:     id.Session,
		RequestedAt: time.Now().UTC(),
	}
	if holder != nil {
		t.TargetDevice = holder.Device
		t.TargetSession = holder.Session
	}
	if err := writeTakeover(archivePath, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ReadTakeover returns the request beside a book, answered or not.
func ReadTakeover(archivePath string) (*Takeover, bool) {
	data, err := os.ReadFile(TakeoverSidecarFor(archivePath))
	if err != nil {
		return nil, false
	}
	var t Takeover
	if err := json.Unmarshal(data, &t); err != nil || t.Version != takeoverVersion {
		return nil, false
	}
	return &t, true
}

// PendingTakeover returns a request this session should put to the writer:
// unanswered, aimed at this session, from somebody else, and recent enough that
// somebody is plausibly still waiting on it.
//
// Call it AGAIN before actually handing the book over. A request that has gone
// is a writer who gave up, and granting to nobody would save, release and close
// a book on a machine with nobody sitting at it.
func PendingTakeover(archivePath string, self Identity) *Takeover {
	request, _ := PendingTakeoverReason(archivePath, self)
	return request
}

// PendingTakeoverReason is PendingTakeover, and says why a request sitting
// beside the book was passed over.
//
// Every reason here is invisible from the outside: the file is in the folder,
// the writer can see it, and nothing happens. That is indistinguishable from a
// bug, so the reason gets logged rather than inferred.
//
// The returned reason is empty when there is nothing there to explain.
func PendingTakeoverReason(archivePath string, self Identity) (*Takeover, string) {
	t, ok := ReadTakeover(archivePath)
	if !ok {
		return nil, ""
	}
	if t.Status != TakeoverAsked {
		return nil, fmt.Sprintf("it was already answered (%s) and is a receipt now", t.Status)
	}
	if t.Mine(self) {
		return nil, "it is this session's own request"
	}
	if !t.Asks(self) {
		return nil, fmt.Sprintf("it asks %q, and this is %q", t.TargetDevice, self.Device)
	}
	// A clock behind ours makes a request look older than it is; one ahead
	// makes it look like the future. Only the first is worth refusing.
	if age := time.Since(t.RequestedAt); age > TakeoverExpires {
		return nil, fmt.Sprintf("it was made %s ago, which is past the %s limit (check the clocks on both machines)", age.Round(time.Second), TakeoverExpires)
	}
	return t, ""
}

// GrantTakeover hands a book over, recording the archive as this session is
// leaving it so the asking device can tell the manuscript apart from the copy
// its sync client has not finished replacing.
//
// Call it AFTER the save has returned. The fingerprint is read off the disk, and
// a fingerprint of the state before the save would be worse than none at all.
func GrantTakeover(archivePath string, self Identity) (*Takeover, error) {
	size, sum, err := fingerprint(archivePath)
	note := ""
	if err != nil {
		// A sync client holding the file open is enough to cause this. The
		// handover still goes ahead; what the other side loses is the proof,
		// and it is told so rather than left to assume.
		note = "the book could not be read for a fingerprint: " + err.Error()
		size, sum = 0, ""
	}
	return answerTakeover(archivePath, self, TakeoverGranted, size, sum, note)
}

// DeclineTakeover refuses: the writer at this machine said they are using it.
func DeclineTakeover(archivePath string, self Identity) (*Takeover, error) {
	return answerTakeover(archivePath, self, TakeoverDeclined, 0, "", "")
}

func answerTakeover(archivePath string, self Identity, status string, size int64, sum, note string) (*Takeover, error) {
	t, ok := ReadTakeover(archivePath)
	if !ok {
		return nil, fmt.Errorf("booklock: nobody is asking for %s", filepath.Base(archivePath))
	}
	t.Status = status
	t.Responder = self.Device
	t.RespondedAt = time.Now().UTC()
	t.ArchiveSize = size
	t.ArchiveSHA256 = sum
	t.Note = note
	if status == TakeoverGranted && sum != "" {
		if info, err := os.Stat(archivePath); err == nil {
			t.SavedAt = info.ModTime().UTC()
		}
	}
	if err := writeTakeover(archivePath, t); err != nil {
		return nil, err
	}
	return t, nil
}

// WithdrawTakeover removes a request. The asking device calls it once it has the
// book open, and when the writer cancels — but NOT when it merely gives up
// waiting. A grant left beside the book is how a later open knows to keep
// waiting for the copy rather than opening whatever is there.
func WithdrawTakeover(archivePath string) {
	_ = os.Remove(TakeoverSidecarFor(archivePath))
}

func writeTakeover(archivePath string, t *Takeover) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("booklock: encode takeover: %w", err)
	}
	file := TakeoverSidecarFor(archivePath)
	if err := fsutil.WriteFileAtomic(file, data, 0o644); err != nil {
		return fmt.Errorf("booklock: write takeover: %w", err)
	}
	hide(file)
	return nil
}

// ArrivalState is how far a granted book has got through the sync folder.
type ArrivalState struct {
	// Arrived is true only when the bytes on this machine hash to what the
	// other machine wrote. Nothing else counts.
	Arrived bool
	// Unverifiable means the grant carried no fingerprint, so there is nothing
	// to check the local copy against.
	Unverifiable  bool
	LocalBytes    int64
	ExpectedBytes int64
}

// Arrival compares the book on this machine against what a grant promised.
//
// The hash is only taken once the sizes agree. Re-reading a hundred megabytes
// every second to watch it not match yet would peg a disk for nothing, and a
// size that differs has already proved the file is still on its way.
func Arrival(archivePath string, t *Takeover) ArrivalState {
	state := ArrivalState{}
	if info, err := os.Stat(archivePath); err == nil {
		state.LocalBytes = info.Size()
	}
	if t == nil || !t.Fingerprinted() {
		state.Unverifiable = true
		return state
	}
	state.ExpectedBytes = t.ArchiveSize
	if state.LocalBytes != t.ArchiveSize {
		return state
	}
	_, sum, err := fingerprint(archivePath)
	if err != nil {
		return state
	}
	state.Arrived = sum == t.ArchiveSHA256
	return state
}

// fingerprint is the size and content hash of a book on this machine.
func fingerprint(archivePath string) (int64, string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()

	sum := sha256.New()
	size, err := io.Copy(sum, f)
	if err != nil {
		return 0, "", err
	}
	return size, hex.EncodeToString(sum.Sum(nil)), nil
}
