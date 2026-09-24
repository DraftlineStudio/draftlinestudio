package main

// The frontend half of the takeover handshake: asking another device for a
// book, watching for somebody asking this one, and answering.
//
// The protocol itself is internal/booklock. What lives here is the part that
// needs the open project: the watch that raises the request on screen, and the
// grant, which can only run once the frontend has finished saving. Go does not
// hold the manuscript — the editor does — so the order is always save in the
// frontend, then grant here.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"draftline/internal/book"
	"draftline/internal/booklock"
	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// takeoverWatchEvery is how often a holder looks for somebody asking for its
// book. The heartbeat is minutes because nothing is waiting on it; this is
// seconds because a writer at another machine is watching a countdown.
const takeoverWatchEvery = 5 * time.Second

// takeoverRequestedEvent carries a request to the frontend, which puts the
// countdown on screen.
const takeoverRequestedEvent = "booklock:takeover_requested"

// RequestBookTakeover asks whichever device holds a book to hand it over,
// rather than forcing past its claim.
//
// The claim in the way is named in the request, so the session that was asked
// is the only one that answers.
func (a *App) RequestBookTakeover(path string) types.BookTakeoverStatus {
	if path == "" {
		return types.BookTakeoverStatus{Message: "there is no book to ask for"}
	}
	self := a.deviceIdentity()
	if _, err := booklock.RequestTakeover(path, self, booklock.Inspect(path, self)); err != nil {
		// A folder that will not take the sidecar cannot carry the request
		// either. The writer keeps the choices they already had.
		return types.BookTakeoverStatus{Message: "this folder will not take a handover request: " + err.Error()}
	}
	return a.BookTakeoverStatus(path)
}

// WithdrawBookTakeover takes a request back. Called when the book has been
// opened, and when the writer cancels — not when they merely stop watching, so
// a grant is still beside the book when they come back to it.
func (a *App) WithdrawBookTakeover(path string) {
	if path != "" {
		booklock.WithdrawTakeover(path)
	}
}

// BookTakeoverStatus is what the waiting screen polls: whether the other device
// has answered, and if it handed the book over, how much of it has arrived.
func (a *App) BookTakeoverStatus(path string) types.BookTakeoverStatus {
	if path == "" {
		return types.BookTakeoverStatus{}
	}
	self := a.deviceIdentity()
	status := types.BookTakeoverStatus{}

	if holder := booklock.Inspect(path, self); holder != nil && !holder.Mine && !holder.Stale {
		status.HeldElsewhere = true
	}

	request, ok := booklock.ReadTakeover(path)
	if !ok {
		return status
	}
	status.Asked = true
	status.Responder = request.Responder
	status.Note = request.Note

	switch request.Status {
	case booklock.TakeoverDeclined:
		status.Answered = true
		status.Declined = true
		status.Message = declinedMessage(request)
		return status
	case booklock.TakeoverGranted:
		status.Answered = true
		status.Granted = true
	default:
		status.Message = "Waiting for " + requestTarget(request) + " to answer…"
		return status
	}

	arrival := booklock.Arrival(path, request)
	status.Arrived = arrival.Arrived
	status.Unverifiable = arrival.Unverifiable
	status.LocalBytes = arrival.LocalBytes
	status.ExpectedBytes = arrival.ExpectedBytes
	status.Message = arrivalMessage(request, arrival)
	return status
}

// PendingBookTakeover is the request waiting on the open book, if any.
//
// The watch announces a request once, by event. This is how the frontend asks
// the same question itself: after a reload, or on startup, when a request may
// have been answered by nobody because there was no window to put it in.
func (a *App) PendingBookTakeover() types.BookTakeoverRequest {
	path := a.getCurrentFile()
	if path == "" {
		return types.BookTakeoverRequest{}
	}
	request := booklock.PendingTakeover(path, a.deviceIdentity())
	if request == nil {
		return types.BookTakeoverRequest{}
	}
	return types.BookTakeoverRequest{
		Device:      request.Device,
		Platform:    request.Platform,
		App:         request.App,
		RequestedAt: request.RequestedAt.Format(time.RFC3339),
		Message:     requestMessage(request),
	}
}

// GrantBookTakeover hands the open book to the device that asked for it.
//
// Call it AFTER the save has returned: the fingerprint it records is read off
// the disk, and the whole point of it is to describe the manuscript as the
// asking device will find it.
//
// A request that has gone is a writer who gave up. Granting to nobody would
// close a book on a machine with nobody sitting at it, so that is checked here
// rather than trusted to the countdown that started the handover.
func (a *App) GrantBookTakeover() types.BookTakeoverResult {
	path := a.getCurrentFile()
	if path == "" {
		return types.BookTakeoverResult{Error: "no book is open"}
	}
	self := a.deviceIdentity()
	if booklock.PendingTakeover(path, self) == nil {
		return types.BookTakeoverResult{Withdrawn: true}
	}
	reply, err := booklock.GrantTakeover(path, self)
	if err != nil {
		return types.BookTakeoverResult{Error: err.Error()}
	}
	// The released claim is what the other device is watching for. It goes
	// after the reply, so the fingerprint is already there when the book
	// appears to be free.
	a.releaseDeviceLock()
	return types.BookTakeoverResult{
		Granted:       true,
		Fingerprinted: reply.Fingerprinted(),
		Device:        reply.Device,
	}
}

// DeclineBookTakeover refuses: somebody is working at this machine.
func (a *App) DeclineBookTakeover() types.BookTakeoverResult {
	path := a.getCurrentFile()
	if path == "" {
		return types.BookTakeoverResult{Error: "no book is open"}
	}
	self := a.deviceIdentity()
	request := booklock.PendingTakeover(path, self)
	if request == nil {
		return types.BookTakeoverResult{Withdrawn: true}
	}
	if _, err := booklock.DeclineTakeover(path, self); err != nil {
		return types.BookTakeoverResult{Error: err.Error()}
	}
	return types.BookTakeoverResult{Declined: true, Device: request.Device}
}

// watchForTakeover raises a request on screen once, while this session holds
// the book. It runs inside the claim's goroutine so it stops when the claim
// does.
func (a *App) watchForTakeover(path string) {
	request := booklock.PendingTakeover(path, a.deviceIdentity())
	if request == nil {
		return
	}

	a.device.mu.Lock()
	seen := a.device.announced == request.Session
	if !seen {
		a.device.announced = request.Session
	}
	a.device.mu.Unlock()
	if seen {
		return
	}

	runtime.EventsEmit(a.ctx, takeoverRequestedEvent, types.BookTakeoverRequest{
		Device:      request.Device,
		Platform:    request.Platform,
		App:         request.App,
		RequestedAt: request.RequestedAt.Format(time.RFC3339),
		Message:     requestMessage(request),
	})
}

// claimLostEvent tells the frontend this session no longer holds the book it
// has open, so it must stop writing to it.
const claimLostEvent = "booklock:claim_lost"

// SaveBookAside writes the open book to a NEW file beside the original and
// leaves the original alone.
//
// This is for the one case where a session has words on screen that are not on
// disk and has just lost the right to write to the book: the machine slept, the
// claim went stale, and another device took it. Saving to the original would
// overwrite whatever has been written on the other machine since. Discarding
// would throw away the writer's last paragraph.
//
// It claims nothing and does not become the open project. It is a rescue file,
// and like every duplicate it gets a new identifier, or it would share a claim
// with the book it was rescued from.
func (a *App) SaveBookAside(b types.BookData) types.SaveResult {
	source := a.getCurrentFile()
	if source == "" {
		return types.SaveResult{Success: false, Error: "no book is open"}
	}
	path, err := uniqueAsidePath(source, a.deviceIdentity().Device)
	if err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}
	b.Metadata.BookID = types.NewBookID()
	// The pending assets are handed over but NOT settled: a cover attached and
	// not yet saved belongs in the rescue file, and the open project is about to
	// be closed anyway.
	assets, _ := a.pendingAssets()
	return book.WriteArchive(source, path, b, AppVersion, nil, assets)
}

// uniqueAsidePath names a rescue file after the machine that could not save it,
// so a writer finding two of them knows which is which.
func uniqueAsidePath(source, device string) (string, error) {
	if strings.TrimSpace(device) == "" {
		device = "this computer"
	}
	dir := filepath.Dir(source)
	ext := filepath.Ext(source)
	stem := strings.TrimSuffix(filepath.Base(source), ext)

	for n := 1; n < 1000; n++ {
		suffix := fmt.Sprintf(" (unsaved on %s)", device)
		if n > 1 {
			suffix = fmt.Sprintf(" (unsaved on %s %d)", device, n)
		}
		candidate := filepath.Join(dir, stem+suffix+ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}
	return "", errors.New("there are already too many rescued copies beside this book")
}

// watchForDispossession notices that the claim on the open book now belongs to
// somebody else, which is what a machine finds when it wakes from sleep: the
// claim went stale while the lid was shut and another device took the book.
//
// It reports whether the watch should stop.
func (a *App) watchForDispossession(path string) bool {
	holder := a.claimTakenElsewhere(path)
	if holder == nil {
		return false
	}
	runtime.EventsEmit(a.ctx, claimLostEvent, types.BookLockInfo{
		Held:     true,
		Stale:    holder.Stale,
		Device:   holder.Device,
		Platform: holder.Platform,
		App:      holder.App,
		LastSeen: holder.LastSeen.Format(time.RFC3339),
		Message:  dispossessedMessage(holder),
	})
	return true
}

// claimTakenElsewhere reports the device that has taken the open book, once,
// and gives up this session's claim on it. It returns nil when the book is
// still ours, and on every look after the first.
//
// What counts as evidence is deliberately narrow. A sidecar that cannot be read
// is NOT evidence — a sync client mid-write, a folder briefly gone, and a file
// somebody deleted all look like that, and treating any of them as a takeover
// would shut a writer's book for nothing. Only a claim that exists and names
// another session counts, and only when two looks in a row agree.
func (a *App) claimTakenElsewhere(path string) *booklock.Holder {
	holder := booklock.Inspect(path, a.deviceIdentity())
	if holder == nil || holder.Mine {
		a.device.mu.Lock()
		a.device.dispossessed = 0
		a.device.mu.Unlock()
		return nil
	}

	a.device.mu.Lock()
	a.device.dispossessed++
	confirmed := a.device.dispossessed >= dispossessionLooks
	told := a.device.stoodDown
	if confirmed {
		a.device.stoodDown = true
	}
	a.device.mu.Unlock()
	if !confirmed || told {
		return nil
	}

	// Give the claim up locally. Release will not delete the sidecar now that
	// it belongs to another session, so this stops the heartbeat without
	// touching the book the other device is holding.
	a.releaseDeviceLock()
	return holder
}

// dispossessionLooks is how many consecutive looks must agree before a book is
// taken off the screen. At takeoverWatchEvery that is ten seconds of a claim
// steadily naming somebody else, which a sync client's half-written file will
// not survive.
const dispossessionLooks = 2

func dispossessedMessage(holder *booklock.Holder) string {
	where := "another device"
	if holder != nil && holder.Device != "" {
		where = holder.Device
	}
	return where + " has this book now, so it was closed here."
}

// requestTarget is who the asking device is waiting on, for a sentence.
func requestTarget(request *booklock.Takeover) string {
	if request != nil && request.TargetDevice != "" {
		return request.TargetDevice
	}
	return "the other device"
}

func requestMessage(request *booklock.Takeover) string {
	where := "Another device"
	if request != nil && request.Device != "" {
		where = request.Device
	}
	return where + " is asking for this book."
}

func declinedMessage(request *booklock.Takeover) string {
	where := requestTarget(request)
	if request != nil && request.Responder != "" {
		where = request.Responder
	}
	return "Somebody is working on " + where + ", so it kept the book."
}

// arrivalMessage says what the waiting screen is waiting for. It counts up,
// because a writer watching a spinner has no way to tell a slow sync from a
// stuck one.
func arrivalMessage(request *booklock.Takeover, arrival booklock.ArrivalState) string {
	where := requestTarget(request)
	switch {
	case arrival.Arrived:
		return where + " handed the book over."
	case arrival.Unverifiable:
		return where + " handed the book over, but this copy could not be checked against it."
	case arrival.ExpectedBytes > 0 && arrival.LocalBytes < arrival.ExpectedBytes:
		return fmt.Sprintf("Waiting for the copy from %s to sync… %s of %s arrived.",
			where, megabytes(arrival.LocalBytes), megabytes(arrival.ExpectedBytes))
	default:
		return "Waiting for the copy from " + where + " to sync…"
	}
}

// megabytes is a size for a writer to read while they wait, not a precise one.
func megabytes(n int64) string {
	const mb = 1 << 20
	if n < mb {
		return fmt.Sprintf("%d KB", (n+1023)/1024)
	}
	return fmt.Sprintf("%.0f MB", float64(n)/mb)
}
