package types

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const bookIDPrefix = "bk-"

// NewBookID mints an identifier for a new book or a copy saved under a new
// name. A copy must get a fresh one or the two files share a claim.
func NewBookID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return DeriveBookID(Metadata{Title: fmt.Sprintf("%p", &b)})
	}
	return bookIDPrefix + hex.EncodeToString(b)
}

// DeriveBookID computes an identifier for a book written before the field
// existed. Deterministic, so two devices derive the same one.
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
func (m *Metadata) EnsureBookID() bool {
	if strings.TrimSpace(m.BookID) != "" {
		return false
	}
	m.BookID = DeriveBookID(*m)
	return true
}

// ShortBookID is the identifier as it appears in a directory name.
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
