package main

// Per-book instance locking. Draftline is multi-instance (Word-style): any
// number of windows, but one book is only ever open in ONE of them. Opening
// a book another instance holds brings that window to the front instead of
// opening a second racing copy. See internal/instancelock for the mechanism.

import (
	"errors"
	"log"
	"sync"

	"draftline/internal/instancelock"
	"draftline/internal/platform"
)

// errBookAlreadyOpen surfaces in the frontend's normal open-error channel.
var errBookAlreadyOpen = errors.New("this book is already open in another Draftline window — switched to that window")

type bookLockState struct {
	mu   sync.Mutex
	lock *instancelock.Lock
}

// claimBookLock takes a prospective per-book lock before an open or save to a
// new path. The caller installs it only after the file operation succeeds, so
// a failed open or Save As cannot drop the lock on the current book.
// A living owner elsewhere is foregrounded and errBookAlreadyOpen is returned.
// Lock-machinery failures are logged and ignored: the lock is a guard, not a
// gate that can prevent an author from opening their manuscript.
func (a *App) claimBookLock(path string) (*instancelock.Lock, error) {
	lock, owner, err := instancelock.Acquire(path)
	if owner != nil {
		if !platform.FocusProcessWindow(owner.PID) {
			log.Printf("book locked by pid %d but its window was not found", owner.PID)
		}
		return nil, errBookAlreadyOpen
	}
	if err != nil {
		log.Printf("book lock unavailable (continuing without): %v", err)
		return nil, nil
	}
	return lock, nil
}

// installBookLock makes a successfully opened or saved path current and
// releases the prior book's lock.
func (a *App) installBookLock(lock *instancelock.Lock) {
	a.bookLock.mu.Lock()
	old := a.bookLock.lock
	a.bookLock.lock = lock
	a.bookLock.mu.Unlock()
	if old != nil && !old.Same(lock) {
		old.Release()
	}
}

// discardBookLockClaim releases a prospective lock unless it is another
// handle for the lock already protecting the current book.
func (a *App) discardBookLockClaim(lock *instancelock.Lock) {
	a.bookLock.mu.Lock()
	current := a.bookLock.lock
	a.bookLock.mu.Unlock()
	if !current.Same(lock) {
		lock.Release()
	}
}

// releaseBookLock drops the current book's lock (close to welcome, new
// unsaved book, app shutdown).
func (a *App) releaseBookLock() {
	a.bookLock.mu.Lock()
	old := a.bookLock.lock
	a.bookLock.lock = nil
	a.bookLock.mu.Unlock()
	old.Release()
	// The cross-device claim is freed at the same moments and for the same
	// reasons, so it rides along here rather than needing every caller to
	// remember a second call. See devicelock.go.
	a.releaseDeviceLock()
}

// CloseBookFile tells the backend the frontend returned to the launch
// screen: the current book is no longer open here, so its lock is freed for
// other instances.
func (a *App) CloseBookFile() {
	a.leaveOpenProject()
	a.releaseBookLock()
}
