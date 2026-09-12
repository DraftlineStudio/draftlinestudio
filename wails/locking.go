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

// acquireBookLock takes the per-book lock before an open (or a save to a new
// path). A living owner elsewhere is foregrounded and errBookAlreadyOpen is
// returned. Lock-machinery failures (unwritable lock dir etc.) are logged
// and IGNORED — a broken lock directory must never stop an author opening
// their manuscript; the lock is a guard, not a gate.
func (a *App) acquireBookLock(path string) error {
	lock, owner, err := instancelock.Acquire(path)
	if owner != nil {
		if !platform.FocusProcessWindow(owner.PID) {
			log.Printf("book locked by pid %d but its window was not found", owner.PID)
		}
		return errBookAlreadyOpen
	}
	if err != nil {
		log.Printf("book lock unavailable (continuing without): %v", err)
		return nil
	}
	a.bookLock.mu.Lock()
	old := a.bookLock.lock
	a.bookLock.lock = lock
	a.bookLock.mu.Unlock()
	if old != nil && old != lock {
		old.Release()
	}
	return nil
}

// releaseBookLock drops the current book's lock (close to welcome, new
// unsaved book, app shutdown).
func (a *App) releaseBookLock() {
	a.bookLock.mu.Lock()
	old := a.bookLock.lock
	a.bookLock.lock = nil
	a.bookLock.mu.Unlock()
	old.Release()
}

// CloseBookFile tells the backend the frontend returned to the launch
// screen: the current book is no longer open here, so its lock is freed for
// other instances.
func (a *App) CloseBookFile() {
	a.setCurrentFile("")
	a.releaseBookLock()
}
