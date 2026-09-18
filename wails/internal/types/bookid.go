package types

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Book identity.
//
// Everything that has to find a book again between sessions — the working copy
// holding its unsaved changes, the lock saying which device has it open, and
// one day whatever syncs it — needs a name for the book that is not its path.
// A path is the wrong name three times over: moving a project into a synced
// folder changes it, renaming the file changes it, and the same book on a
// laptop and a phone has two of them.

// bookIDPrefix keeps these legible in a manifest and in a directory listing.
const bookIDPrefix = "bk-"

// NewBookID mints an identifier for a book that has never had one: a new
// project, or a copy saved under a new name. A copy MUST get a fresh one —
// two files sharing an identity would share a working copy and overwrite each
// other's chapters.
func NewBookID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Randomness is not available, which on the platforms Draftline runs
		// on means something is badly wrong. A derived identifier is still
		// better than an empty one, and DeriveBookID never returns "".
		return DeriveBookID(Metadata{Title: fmt.Sprintf("%p", &b)})
	}
	return bookIDPrefix + hex.EncodeToString(b)
}

// DeriveBookID computes an identifier from what a book already says about
// itself. It is used for projects written before this field existed, and it is
// deliberately deterministic: the same book opened on a laptop and a phone
// derives the same identifier on both, so the two agree about what it is
// before either has managed to save the field into the archive.
//
// Title, author and creation date are the three pieces of metadata that do not
// change as a manuscript is written. It is possible to defeat this — two books
// created in the same second by the same author with the same title — and the
// cost of doing so is one shared working copy, which the lock and the archive
// change detection both then notice.
func DeriveBookID(m Metadata) string {
	seed := strings.Join([]string{
		strings.TrimSpace(m.Title),
		strings.TrimSpace(m.Author),
		strings.TrimSpace(m.Created),
	}, "\x00")
	sum := sha256.Sum256([]byte(seed))
	return bookIDPrefix + hex.EncodeToString(sum[:16])
}

// EnsureBookID fills in a missing identifier and reports whether it had to.
// A caller that gets true should make sure the book is saved, so the next
// reader inherits the identifier rather than deriving it again.
func (m *Metadata) EnsureBookID() bool {
	if strings.TrimSpace(m.BookID) != "" {
		return false
	}
	m.BookID = DeriveBookID(*m)
	return true
}

// ShortBookID is the identifier as it appears in a directory name: short
// enough to read, long enough not to collide.
func ShortBookID(id string) string {
	trimmed := strings.TrimPrefix(strings.TrimSpace(id), bookIDPrefix)
	if trimmed == "" {
		return "unknown"
	}
	if len(trimmed) > 12 {
		return trimmed[:12]
	}
	return trimmed
}
