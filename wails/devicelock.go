package main

// Cross-device book claims. See internal/booklock for the mechanism and, more
// importantly, for why it is advisory and must stay that way.
//
// This is the second of two locks and they answer different questions:
//
//	instancelock  is this book open in another window ON THIS MACHINE?
//	              Exact. Checks whether the owning process is alive, steals a
//	              dead one's lock, and refuses the second copy outright.
//	booklock      might this book be open on ANOTHER MACHINE?
//	              A guess, carried by whatever syncs the folder. Never refuses
//	              anything; the author is told and decides.
//
// The second exists because a book in a Dropbox or OneDrive folder can be open
// on a laptop and a desktop at once, both saving, and the sync client cannot
// merge two versions of a ZIP. One of the two afternoons is simply lost, with
// nothing on screen beforehand to suggest it might be.

import (
	"log"
	"sync"
	"time"

	"draftline/internal/booklock"
	"draftline/internal/types"
)

// deviceClaim is this process's claim on the open book, plus the timer that
// keeps it looking alive to other devices.
type deviceClaim struct {
	mu   sync.Mutex
	lock *booklock.Lock
	stop chan struct{}
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
	holder := booklock.Inspect(path, sessionID)
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

// claimDeviceLock takes the cross-device claim for a book that has just been
// opened, and starts the heartbeat.
//
// It always takes it. By the time this runs the author has either seen no
// warning or has read one and chosen to continue, and a claim that refused at
// this point would only strand them with a book they cannot edit.
func (a *App) claimDeviceLock(path string) {
	a.releaseDeviceLock()

	lock, previous, err := booklock.Force(path, "", a.deviceIdentity())
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
	a.device.mu.Unlock()

	go func() {
		ticker := time.NewTicker(booklock.HeartbeatEvery)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
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
