package main

// The cross-device claim on the open book. instancelock covers this machine;
// booklock covers the others. See internal/booklock.

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"draftline/internal/book"
	"draftline/internal/booklock"
	"draftline/internal/fsutil"
	"draftline/internal/types"
)

// deviceClaim is this process's claim on the open book, plus the timer that
// keeps it looking alive to other devices.
type deviceClaim struct {
	mu   sync.Mutex
	lock *booklock.Lock
	stop chan struct{}
	// announced is the session of the last takeover request put on screen, so
	// a request is raised once rather than every time the watch looks.
	announced string
}

// sessionID identifies this run of the application. A claim carrying it is
// recognised as ours rather than stolen from a stranger, which matters when a
// book is closed and reopened without the process restarting.
var sessionID = time.Now().UTC().Format("20060102T150405.000000000")

func (a *App) deviceIdentity() booklock.Identity {
	return booklock.ThisDevice("Draftline "+AppVersion, sessionID)
}

// InspectBookLock reports whether a book looks open somewhere else, without
// claiming it. The frontend calls this before opening so it can warn; the
// Library can call it to mark a book that is busy elsewhere.
//
// Everything about the result is a hint. See booklock's package comment.
func (a *App) InspectBookLock(path string) types.BookLockInfo {
	holder := booklock.Inspect(path, a.deviceIdentity())
	if holder == nil || holder.Mine {
		return types.BookLockInfo{}
	}
	return types.BookLockInfo{
		Held:     true,
		Stale:    holder.Stale,
		Device:   holder.Device,
		Platform: holder.Platform,
		App:      holder.App,
		LastSeen: holder.LastSeen.Format(time.RFC3339),
		Message:  holder.Describe(),
	}
}

// OpenBookAsCopy duplicates a book and opens the duplicate, leaving the
// original untouched and still claimed by whichever device has it.
//
// This is the safe answer to "it may be open on your laptop": neither copy can
// overwrite the other, and the author can reconcile them later by reading both,
// which beats discovering at midnight that a sync client picked one.
//
// The copy is a DIFFERENT BOOK and is given its own identifier, for the same
// reason Save As mints one. Two files sharing an identifier would share a
// claim, and later a working copy, and would overwrite each other — which is
// the failure this whole feature exists to avoid.
func (a *App) OpenBookAsCopy(path string) (types.BookData, error) {
	copyPath, err := uniqueCopyPath(path)
	if err != nil {
		return types.BookData{}, err
	}
	if err := fsutil.CopyFileAtomic(path, copyPath, 0o644); err != nil {
		return types.BookData{}, fmt.Errorf("could not copy the book: %w", err)
	}

	b, err := a.openBook(copyPath)
	if err != nil {
		// Nothing was opened, so the half-made copy is litter. Leaving it
		// would put a second file beside the author's book with no
		// explanation of where it came from.
		_ = os.Remove(copyPath)
		return types.BookData{}, err
	}

	b.Metadata.BookID = types.NewBookID()
	if result := book.Write(copyPath, copyPath, b, AppVersion); !result.Success {
		// The copy is open and usable; it just has not been stamped with its
		// new identity yet, and the next save will do that.
		log.Printf("could not stamp the copy with a new book id: %s", result.Error)
	}
	return b, nil
}

// uniqueCopyPath picks a name beside the original that is not taken.
func uniqueCopyPath(path string) (string, error) {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(filepath.Base(path), ext)

	for n := 1; n < 1000; n++ {
		suffix := " (copy)"
		if n > 1 {
			suffix = fmt.Sprintf(" (copy %d)", n)
		}
		candidate := filepath.Join(dir, stem+suffix+ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}
	return "", errors.New("there are already too many copies of this book beside it")
}

// claimDeviceLock takes the cross-device claim for a book that has just been
// opened, and starts the heartbeat.
//
// It always takes it. By the time this runs the author has either seen no
// warning or has read one and chosen to continue, and a claim that refused at
// this point would only strand them with a book they cannot edit.
func (a *App) claimDeviceLock(path, bookID string) {
	a.releaseDeviceLock()

	lock, previous, err := booklock.Force(path, bookID, a.deviceIdentity())
	if err != nil {
		// A folder that will not take a sidecar is not a reason to stop
		// somebody working. They lose the warning, which is what they had
		// before this existed.
		log.Printf("book claim unavailable (continuing without): %v", err)
		return
	}
	if previous != nil && !previous.Mine && !previous.Stale {
		log.Printf("took a live book claim from %s (%s)", previous.Device, previous.App)
	}

	stop := make(chan struct{})
	a.device.mu.Lock()
	a.device.lock = lock
	a.device.stop = stop
	// A request raised against the session that held this book before is not a
	// request against this one.
	a.device.announced = ""
	a.device.mu.Unlock()

	go func() {
		beat := time.NewTicker(booklock.HeartbeatEvery)
		defer beat.Stop()
		// The heartbeat is minutes because nothing waits on it. The takeover
		// watch is seconds because a writer at the other machine is.
		watch := time.NewTicker(takeoverWatchEvery)
		defer watch.Stop()
		for {
			select {
			case <-stop:
				return
			case <-watch.C:
				a.device.mu.Lock()
				held := a.device.lock
				a.device.mu.Unlock()
				if held == nil {
					return
				}
				a.watchForTakeover(path)
			case <-beat.C:
				a.device.mu.Lock()
				held := a.device.lock
				a.device.mu.Unlock()
				if held == nil {
					return
				}
				if err := held.Heartbeat(); err != nil {
					// The folder went away, or a sync client is holding the
					// file. Worth a line; not worth interrupting anybody.
					log.Printf("book claim heartbeat failed: %v", err)
				}
			}
		}
	}()
}

// releaseDeviceLock gives up this device's claim and stops the heartbeat. A
// claim left behind by a crash goes stale on its own after
// booklock.StaleAfter; releasing cleanly means the other machine sees the book
// as free at once instead of in a quarter of an hour.
func (a *App) releaseDeviceLock() {
	a.device.mu.Lock()
	lock := a.device.lock
	stop := a.device.stop
	a.device.lock = nil
	a.device.stop = nil
	a.device.mu.Unlock()

	if stop != nil {
		close(stop)
	}
	if lock != nil {
		if err := lock.Release(); err != nil {
			log.Printf("releasing book claim: %v", err)
		}
	}
}
