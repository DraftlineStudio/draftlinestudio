// Package instancelock provides per-book file locking across Draftline
// processes. Draftline is multi-instance (two windows with two different
// books is fine); what must never happen is the same .draftline being open
// twice — the archive write path, backup rotation, and chapter history all
// assume a single writer. A lock is a small JSON file keyed by the book's
// normalized path, holding the owning PID; liveness is checked so a crashed
// instance's lock is stolen, never shown to the user.
package instancelock

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Owner describes the living process that already holds a book's lock.
type Owner struct {
	PID      int    `json:"pid"`
	BookPath string `json:"book_path"`
	Acquired string `json:"acquired"`
}

// Lock is a held lock. Release it when the book is closed, replaced, or the
// app exits; a crash is tolerated (the next opener steals a dead PID's lock).
type Lock struct {
	file string
	pid  int
}

// lockDir returns the per-user lock directory. Per-user is deliberate for
// now: the realistic collision is one author double-opening their own book.
// (Cross-OS-user simultaneous opens of one shared file are not detected.)
func lockDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "draftline", "locks")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// normalize canonicalizes a book path so the same file always maps to the
// same lock key. Windows paths compare case-insensitively.
func normalize(bookPath string) string {
	p := bookPath
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		p = resolved
	}
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}

func lockFileFor(bookPath string) (string, error) {
	dir, err := lockDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(normalize(bookPath)))
	return filepath.Join(dir, hex.EncodeToString(sum[:8])+".lock"), nil
}

// Acquire takes the lock for bookPath. Outcomes:
//   - (lock, nil, nil): acquired (or re-acquired by the same process).
//   - (nil, owner, nil): a LIVING other process holds it — caller decides
//     what to do (Draftline foregrounds that window).
//   - (nil, nil, err): the lock directory is unusable. Callers should treat
//     this as "proceed without a lock" rather than blocking the author.
func Acquire(bookPath string) (*Lock, *Owner, error) {
	file, err := lockFileFor(bookPath)
	if err != nil {
		return nil, nil, err
	}
	self := os.Getpid()
	payload, _ := json.Marshal(Owner{PID: self, BookPath: bookPath, Acquired: time.Now().Format(time.RFC3339)})

	for attempt := 0; attempt < 3; attempt++ {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, werr := f.Write(payload)
			cerr := f.Close()
			if werr != nil || cerr != nil {
				_ = os.Remove(file)
				return nil, nil, fmt.Errorf("writing lock: %w", err)
			}
			return &Lock{file: file, pid: self}, nil, nil
		}
		if !os.IsExist(err) {
			return nil, nil, err
		}
		owner, readErr := readOwner(file)
		if readErr != nil || owner == nil {
			// Corrupt or vanished mid-read: clear and retry.
			_ = os.Remove(file)
			continue
		}
		if owner.PID == self {
			// Same process re-opening the same book (e.g. via recents).
			return &Lock{file: file, pid: self}, nil, nil
		}
		if pidAlive(owner.PID) {
			return nil, owner, nil
		}
		// Stale lock from a crashed instance: steal it.
		_ = os.Remove(file)
	}
	return nil, nil, fmt.Errorf("could not settle lock contention for %s", bookPath)
}

// CurrentOwner reports the living process holding bookPath's lock, or nil.
// Read-only: never creates, steals, or cleans anything — used by the launch
// path to decide whether to defer to an existing window before any UI.
func CurrentOwner(bookPath string) *Owner {
	file, err := lockFileFor(bookPath)
	if err != nil {
		return nil
	}
	owner, err := readOwner(file)
	if err != nil || owner == nil || owner.PID == os.Getpid() || !pidAlive(owner.PID) {
		return nil
	}
	return owner
}

func readOwner(file string) (*Owner, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var o Owner
	if err := json.Unmarshal(data, &o); err != nil || o.PID <= 0 {
		return nil, fmt.Errorf("corrupt lock file")
	}
	return &o, nil
}

// Release drops the lock if this process still holds it. Safe on nil.
func (l *Lock) Release() {
	if l == nil {
		return
	}
	owner, err := readOwner(l.file)
	if err == nil && owner != nil && owner.PID == l.pid {
		_ = os.Remove(l.file)
	}
}
