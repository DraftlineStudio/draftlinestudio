package main

// Frozen manuscripts, from the application's side.
//
// The rules are the cover art rules, for the same reasons (see cover.go):
//
//  1. Frozen text never joins types.BookData. A snapshot is a whole novel and
//     BookData is serialised as JSON across the bridge on every autosave. Only
//     the small catalogue record travels; the text goes to the writer as a
//     sibling parameter.
//  2. The cache below is authoritative only for what has not been saved yet.
//     Once a save has written a snapshot into the archive the entry is dropped,
//     and anything that wants those words reads them back out of the project
//     file.
//
// What is different is the direction of the read. A cover is displayed, over
// and over, so its cache is a read-through. A snapshot is read once, at the
// moment of an export, so there is nothing to keep: holding a second copy of
// every published edition in memory for the life of the session would be a
// cache that serves one request a month.

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"draftline/internal/book"
	"draftline/internal/types"
)

type cachedSnapshot struct {
	// files maps a full archive member name to its bytes.
	files map[string][]byte
	// version rises on every freeze, so a save that started before a second
	// freeze cannot mark the newer bytes as written.
	version uint64
}

// snapshotCache is usable as a zero value, like coverCache and for the same
// reason: App is constructed bare in places.
type snapshotCache struct {
	mu      sync.Mutex
	entries map[string]*cachedSnapshot
	nextVer uint64
}

func (c *snapshotCache) put(snapshot book.Snapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]*cachedSnapshot{}
	}
	if _, ok := c.entries[snapshot.Record.ID]; ok {
		// The same text frozen again. It is content-addressed, so the bytes
		// already pending are these bytes; replacing them would only move the
		// version forward and risk a save that is mid-flight losing them.
		return
	}
	c.nextVer++
	c.entries[snapshot.Record.ID] = &cachedSnapshot{files: snapshot.Files, version: c.nextVer}
}

// has reports whether a snapshot is held here rather than in the project file.
func (c *snapshotCache) has(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.entries[id]
	return ok
}

func (c *snapshotCache) lookup(member string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, entry := range c.entries {
		if data, ok := entry.files[member]; ok {
			return data, true
		}
	}
	return nil, false
}

// pending is every frozen manuscript the next save has to write.
func (c *snapshotCache) pending() (book.Assets, map[string]uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) == 0 {
		return book.Assets{}, nil
	}
	assets := book.Assets{Files: map[string][]byte{}}
	marks := map[string]uint64{}
	for id, entry := range c.entries {
		marks[id] = entry.version
		for name, data := range entry.files {
			assets.Files[name] = data
		}
	}
	return assets, marks
}

// settled forgets what a successful save has written. A snapshot frozen while
// that save was running keeps a newer version and stays.
func (c *snapshotCache) settled(marks map[string]uint64) {
	if len(marks) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, version := range marks {
		if entry := c.entries[id]; entry != nil && entry.version == version {
			delete(c.entries, id)
		}
	}
}

func (c *snapshotCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = map[string]*cachedSnapshot{}
}

// ── What a save carries ────────────────────────────────────────────────────

// pendingAssets is everything a save has to write from memory: cover art and
// frozen manuscripts. The returned function is called only after the save has
// succeeded, and settles both.
func (a *App) pendingAssets() (book.Assets, func()) {
	coverAssets, coverMarks := a.covers.pending()
	snapshotAssets, snapshotMarks := a.snapshots.pending()

	merged := book.Assets{Files: map[string][]byte{}, Superseded: coverAssets.Superseded}
	for name, data := range coverAssets.Files {
		merged.Files[name] = data
	}
	// A frozen manuscript is filed under the hash of its own contents, so no
	// two of them can want the same member name and nothing can supersede one.
	for name, data := range snapshotAssets.Files {
		merged.Files[name] = data
	}
	if len(merged.Files) == 0 && len(merged.Superseded) == 0 {
		return book.Assets{}, func() {}
	}
	return merged, func() {
		a.covers.settled(coverMarks)
		a.snapshots.settled(snapshotMarks)
	}
}

// ── Freezing ───────────────────────────────────────────────────────────────

// FreezeSnapshot records the manuscript as it stands as the text one format
// was published from.
//
// It does not touch the publishing record: the record belongs to the book the
// frontend is holding, and writing to it from here would mean two owners of
// the same object. The screen takes the returned catalogue record, puts it in
// the book's snapshot list and stamps the format with its identifier, and the
// ordinary autosave writes the words.
//
// Freezing the same unchanged text a second time costs nothing and stores
// nothing: the identifier is the hash of the content, so the second format
// simply points at the first one's folder. Result.Reused says so, because a
// screen that reported "frozen" twice would imply two copies.
func (a *App) FreezeSnapshot(b types.BookData, formatID string) (result types.SnapshotResult) {
	defer func() {
		if r := recover(); r != nil {
			result = types.SnapshotResult{Error: fmt.Sprintf("that text could not be frozen: %v", r)}
		}
	}()

	formatID = strings.TrimSpace(formatID)
	if formatID == "" {
		return types.SnapshotResult{Error: "choose a format to freeze this text for"}
	}
	if b.Editions == nil {
		return types.SnapshotResult{Error: "this book has no registered editions to freeze text for"}
	}
	if _, _, ok := b.Editions.FindFormat(formatID); !ok {
		return types.SnapshotResult{Error: "that format is no longer part of this book"}
	}
	if len(b.FrontMatter)+len(b.Body)+len(b.BackMatter) == 0 {
		return types.SnapshotResult{Error: "there is nothing written to freeze yet"}
	}

	snapshot := book.BuildSnapshot(b, time.Now())
	reused := false
	if existing, ok := b.Editions.FindSnapshot(snapshot.Record.ID); ok {
		// Same words, already frozen. Keep the moment it was first frozen at:
		// that is when this text became the published text, and rewriting it
		// to today would erase the date the first ISBN went out with.
		snapshot.Record = existing
		reused = true
	}
	// A reused snapshot is already in the project file, but only if a save has
	// happened since it was frozen. Handing the bytes over again is free — the
	// cache refuses a duplicate — and it is what makes freezing twice before
	// the first autosave safe.
	a.snapshots.put(snapshot)

	record := snapshot.Record
	return types.SnapshotResult{Success: true, Snapshot: &record, Reused: reused}
}

// ── Reading one back ───────────────────────────────────────────────────────

// snapshotBook rebuilds the book one format was published from.
//
// Everything that is text comes from the snapshot: the sections, their order,
// their titles, the author's own copyright page and the metadata as it stood.
// Everything that is a record comes from the book on screen — the publishing
// index above all, so that an export of the first edition still carries the
// ISBN, the cover and the copyright fields as they are held today. Those are
// not the manuscript, and an author who corrects a misprinted imprint wants
// the correction, not the misprint.
func (a *App) snapshotBook(b types.BookData, formatID string) (types.BookData, error) {
	_, format, ok := b.Editions.FindFormat(formatID)
	if !ok || strings.TrimSpace(format.SnapshotID) == "" {
		return b, nil
	}
	frozen, err := a.readSnapshot(format.SnapshotID)
	if err != nil {
		return types.BookData{}, err
	}
	frozen.Editions = b.Editions
	frozen.FilePath = b.FilePath
	frozen.StoryBible = b.StoryBible
	frozen.WritingGoals = b.WritingGoals
	frozen.StyleOptions = b.StyleOptions
	return frozen, nil
}

func (a *App) readSnapshot(id string) (types.BookData, error) {
	if a.snapshots.has(id) {
		prefix := book.SnapshotPrefix(id)
		return book.DecodeSnapshot(id, func(name string) ([]byte, error) {
			data, ok := a.snapshots.lookup(prefix + name)
			if !ok {
				return nil, fmt.Errorf("%q is missing", name)
			}
			return data, nil
		})
	}
	return book.ReadSnapshot(a.getCurrentFile(), id)
}

// exportSource is the book an export should actually render.
//
// A format with a frozen manuscript exports that manuscript; a format without
// one exports the draft on screen, which is what Draftline has always done and
// what the wizard says it is about to do. A frozen manuscript that cannot be
// read is an error rather than a quiet fallback: handing over today's words
// under a published ISBN, having told the author they were getting the book as
// it was, is the one outcome worth refusing an export for.
func (a *App) exportSource(b types.BookData, options types.ExportOptions) (types.BookData, error) {
	formatID := strings.TrimSpace(options.FormatID)
	if formatID == "" || b.Editions == nil {
		return b, nil
	}
	return a.snapshotBook(b, formatID)
}
