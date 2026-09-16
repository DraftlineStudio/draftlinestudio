package main

// Cover art: the first binary Draftline has ever owned.
//
// Three rules shape this file, and all three are about where bytes are allowed
// to be.
//
//  1. A cover enters as a PATH. The native file dialog returns one; a
//     drag-and-drop returns one (that is what EnableFileDrop in main.go is
//     for). An HTML file input would push forty megabytes through the webview
//     and then base64 across the JSON bridge to get to the same place.
//  2. A cover is never on types.BookData. That struct is serialised as JSON on
//     every save and autosave runs five seconds after a keystroke. The bytes
//     live in the cache below, keyed by edition, and go to the writer as a
//     sibling parameter.
//  3. A cover is displayed from a same-origin URL served by this process -
//     /editions/<edition id>/<file> - not as a data: URI. Same reason.
//
// The cache is authoritative only for what has not been saved yet. Once a save
// has written a cover into the archive, the cache is a read-through: a miss
// reads the member out of the project file on disk. That is what makes a
// reopened project show its covers without holding every edition's artwork in
// memory from the moment it opens.

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"draftline/internal/book"
	"draftline/internal/coverart"
	"draftline/internal/export"
	"draftline/internal/types"
	"draftline/internal/ziputil"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// maxCachedCoverBytes bounds what the read-through cache keeps in memory.
// Unsaved covers are never evicted - losing one would lose the author's work -
// so this only limits what has been read back out of the archive for display.
const maxCachedCoverBytes = 24 << 20

type coverFiles map[string][]byte

type coverEntry struct {
	files coverFiles
	// dirty means these bytes are not in the project file yet.
	dirty bool
	// version rises on every attach, so a save that started before a second
	// attach cannot mark the newer bytes as written.
	version uint64
}

// coverCache is usable as a zero value: its map is built on first use. That
// matters because App is constructed bare in places, and a cover cache that
// needed a constructor would be a nil map waiting for the first attach.
type coverCache struct {
	mu      sync.Mutex
	entries map[string]*coverEntry
	nextVer uint64
}

// ensure runs under the lock.
func (c *coverCache) ensure() {
	if c.entries == nil {
		c.entries = map[string]*coverEntry{}
	}
}

// put replaces everything cached for one edition and marks it unsaved.
func (c *coverCache) put(editionID string, files coverFiles) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensure()
	c.nextVer++
	c.entries[editionID] = &coverEntry{files: files, dirty: true, version: c.nextVer}
}

// cache stores bytes read back out of the archive. They are already saved, so
// they are not dirty and must not be written again.
func (c *coverCache) cache(editionID, name string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensure()
	entry := c.entries[editionID]
	if entry == nil {
		entry = &coverEntry{files: coverFiles{}}
		c.entries[editionID] = entry
	}
	if entry.dirty {
		// Never overwrite unsaved work with what is on disk.
		return
	}
	if c.residentBytes()+len(data) > maxCachedCoverBytes {
		c.dropClean()
	}
	entry.files[name] = data
}

func (c *coverCache) lookup(editionID, name string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := c.entries[editionID]
	if entry == nil {
		return nil, false
	}
	data, ok := entry.files[name]
	return data, ok
}

// pending is what the next save has to write: the members of every unsaved
// cover, and the prefixes those covers replace.
//
// The versions it returns are handed back to settled after a successful save,
// so that an attach which happened while the save was running stays dirty.
func (c *coverCache) pending() (book.Assets, map[string]uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	assets := book.Assets{Files: map[string][]byte{}}
	marks := map[string]uint64{}
	for editionID, entry := range c.entries {
		if !entry.dirty {
			continue
		}
		marks[editionID] = entry.version
		assets.Superseded = append(assets.Superseded, book.CoverPrefix(editionID))
		for name, data := range entry.files {
			assets.Files[book.CoverMember(editionID, name)] = data
		}
	}
	// An edition with no files and a dirty flag is a cover that was removed.
	// It still has to reach the writer, because its whole point is the
	// Superseded prefix that stops the passthrough carrying the old artwork
	// forward. Only a cache with nothing pending at all returns nothing.
	if len(marks) == 0 {
		return book.Assets{}, nil
	}
	return assets, marks
}

func (c *coverCache) settled(marks map[string]uint64) {
	if len(marks) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for editionID, version := range marks {
		if entry := c.entries[editionID]; entry != nil && entry.version == version {
			entry.dirty = false
		}
	}
}

// reset forgets everything. Opening or starting a book is a different book's
// covers.
func (c *coverCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = map[string]*coverEntry{}
}

// residentBytes and dropClean run under the lock.
func (c *coverCache) residentBytes() int {
	total := 0
	for _, entry := range c.entries {
		for _, data := range entry.files {
			total += len(data)
		}
	}
	return total
}

func (c *coverCache) dropClean() {
	for editionID, entry := range c.entries {
		if !entry.dirty {
			delete(c.entries, editionID)
		}
	}
}

// ── Attaching ──────────────────────────────────────────────────────────────

// AttachCoverDialog asks for a file and attaches it to an edition.
//
// The dialog returns a path, which is the only form this whole pipeline
// accepts. Dismissing it is not an error and does not produce one on screen.
func (a *App) AttachCoverDialog(editionID string, large bool) types.CoverResult {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Choose cover artwork",
		DefaultDirectory: a.getSettings().DefaultSaveDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "Cover artwork (*.jpg;*.jpeg;*.png;*.tif;*.tiff;*.webp;*.bmp)", Pattern: "*.jpg;*.jpeg;*.png;*.tif;*.tiff;*.webp;*.bmp"},
		},
	})
	if err != nil {
		return types.CoverResult{Error: err.Error()}
	}
	if path == "" {
		return types.CoverResult{Cancelled: true}
	}
	return a.AttachCover(editionID, path, large)
}

// AttachCover prepares the artwork at path and attaches it to one edition.
//
// It is wrapped against panics for the same reason the importers are: main.go
// binds App directly, and an unrecovered panic in a bound method takes the
// whole application down. A decoder handed a malformed file is exactly where
// one would come from, and losing the author's unsaved chapter because a TIFF
// was truncated is not a trade anybody would make.
func (a *App) AttachCover(editionID, sourcePath string, large bool) (result types.CoverResult) {
	defer func() {
		if r := recover(); r != nil {
			result = types.CoverResult{Error: fmt.Sprintf("that cover could not be read: %v", r)}
		}
	}()

	editionID = strings.TrimSpace(editionID)
	if editionID == "" {
		return types.CoverResult{Error: "choose an edition to attach this cover to"}
	}

	prepared, err := coverart.Prepare(sourcePath, coverart.Options{Large: large})
	if err != nil {
		return types.CoverResult{Error: err.Error()}
	}

	// The original is recorded, not copied. If it cannot be fingerprinted the
	// cover is still good; only the route back to the print-ready file is
	// lost, and the screen will say so when it asks.
	fingerprint, _ := coverart.TakeFingerprint(sourcePath)

	files := coverFiles{
		prepared.Cover.Name: prepared.Cover.Data,
		prepared.Thumb.Name: prepared.Thumb.Data,
	}
	record := &types.EditionCover{
		ID:                fmt.Sprintf("cov-%d", time.Now().UnixNano()),
		File:              prepared.Cover.Name,
		ThumbFile:         prepared.Thumb.Name,
		Width:             prepared.Cover.Width,
		Height:            prepared.Cover.Height,
		Bytes:             prepared.Cover.Bytes,
		ThumbWidth:        prepared.Thumb.Width,
		ThumbHeight:       prepared.Thumb.Height,
		ThumbBytes:        prepared.Thumb.Bytes,
		Encoding:          prepared.Cover.Encoding,
		Quality:           prepared.Cover.Quality,
		Greyscale:         prepared.Greyscale,
		ConvertedFromCMYK: prepared.ConvertedFromCMYK,
		FlattenedAlpha:    prepared.FlattenedAlpha,
		SourcePath:        fingerprint.Path,
		SourceChecksum:    fingerprint.Checksum,
		SourceBytes:       fingerprint.Bytes,
		SourceWidth:       prepared.SourceWidth,
		SourceHeight:      prepared.SourceHeight,
		SourceModified:    fingerprint.Modified,
		SourceFormat:      prepared.SourceFormat,
		Attached:          time.Now().Format(time.RFC3339),
		Notes:             prepared.Notes,
	}
	if record.Notes == nil {
		record.Notes = []string{}
	}
	if prepared.Large != nil {
		files[prepared.Large.Name] = prepared.Large.Data
		record.LargeFile = prepared.Large.Name
		record.LargeWidth = prepared.Large.Width
		record.LargeHeight = prepared.Large.Height
		record.LargeBytes = prepared.Large.Bytes
	}

	a.covers.put(editionID, files)
	return types.CoverResult{Success: true, Cover: record}
}

// RemoveCover drops an edition's artwork. The record is the frontend's to
// clear; this releases the bytes so the next save stops carrying them.
func (a *App) RemoveCover(editionID string) {
	a.covers.put(strings.TrimSpace(editionID), coverFiles{})
}

// CheckCoverSource answers whether the print-ready original is still where it
// was when the cover was made.
//
// It takes the recorded facts rather than an edition id, so the screen can ask
// about a record it is holding without a round trip through the book.
func (a *App) CheckCoverSource(sourcePath, checksum string) types.CoverSourceReport {
	report := coverart.CheckSource(coverart.Fingerprint{Path: sourcePath, Checksum: checksum})
	return types.CoverSourceReport{Status: report.Status, Message: report.Message, Path: report.Path}
}

// ── Export ─────────────────────────────────────────────────────────────────

// exportCover is the artwork an export should carry, or nil.
//
// The bytes are fetched here, in the application, and handed to the exporter
// as a parameter. They never travel on types.BookData and they never cross the
// bridge: the frontend asks for an export of a named format, and this reads
// the image for that format's edition straight out of the cache or the open
// project file.
//
// An export that names no format, or names one whose edition has no cover,
// gets nil and produces the file Draftline has always produced. A cover that
// cannot be read is nil too, for the same reason a missing font is: an author
// who asked for an ebook should get an ebook.
func (a *App) exportCover(bookData types.BookData, options types.ExportOptions) *export.CoverArt {
	formatID := strings.TrimSpace(options.FormatID)
	if formatID == "" || bookData.Editions == nil {
		return nil
	}
	edition, _, ok := bookData.Editions.FindFormat(formatID)
	if !ok || edition.Cover == nil || strings.TrimSpace(edition.Cover.File) == "" {
		return nil
	}
	data, err := a.coverBytes(edition.ID, edition.Cover.File)
	if err != nil || len(data) == 0 {
		return nil
	}
	return &export.CoverArt{
		Data:      data,
		MediaType: coverContentType(edition.Cover.File),
		FileName:  edition.Cover.File,
		Width:     edition.Cover.Width,
		Height:    edition.Cover.Height,
	}
}

// ── Display ────────────────────────────────────────────────────────────────

// editionAssetPrefix is the URL space cover art is served from. It is a
// sibling of /plugins/, installed by the same middleware in plugins.go.
const editionAssetPrefix = "/editions/"

// editionAssets serves an edition's images to the webview, same-origin.
//
// It answers from the cache first, and reads the project file on a miss, which
// is what a reopened book needs: the archive holds the covers, and nothing
// loads them until something asks to look at one.
func (a *App) editionAssets() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rest, ok := strings.CutPrefix(r.URL.Path, editionAssetPrefix)
		if !ok {
			http.NotFound(w, r)
			return
		}
		editionID, file, ok := strings.Cut(rest, "/")
		if !ok || !safeAssetSegment(editionID) || !safeAssetSegment(file) {
			http.NotFound(w, r)
			return
		}

		data, err := a.coverBytes(editionID, file)
		if err != nil || data == nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", coverContentType(file))
		// The URL carries the cover's identifier as a query, and that
		// identifier changes whenever the art does, so the bytes behind one
		// address never change and the webview may keep them.
		w.Header().Set("Cache-Control", "private, max-age=86400")
		http.ServeContent(w, r, file, time.Time{}, newByteSeeker(data))
	})
}

// coverBytes finds one image: the cache first, then the open project file.
func (a *App) coverBytes(editionID, file string) ([]byte, error) {
	if data, ok := a.covers.lookup(editionID, file); ok {
		return data, nil
	}
	archivePath := a.getCurrentFile()
	if archivePath == "" {
		return nil, nil
	}
	data, err := readArchiveMember(archivePath, book.CoverMember(editionID, file))
	if err != nil || data == nil {
		return nil, err
	}
	a.covers.cache(editionID, file, data)
	return data, nil
}

// readArchiveMember pulls one member out of a .draftline without opening the
// book. The size guard is the archive's own: a member claiming to be larger
// than ziputil allows is not read.
func readArchiveMember(archivePath, name string) ([]byte, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		if f.Name != name {
			continue
		}
		if f.UncompressedSize64 > ziputil.MaxEntrySize {
			return nil, fmt.Errorf("%s is larger than a project file may hold", name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return io.ReadAll(io.LimitReader(rc, ziputil.MaxEntrySize))
	}
	return nil, nil
}

func coverContentType(file string) string {
	switch strings.ToLower(path.Ext(file)) {
	case ".png":
		return "image/png"
	default:
		return "image/jpeg"
	}
}

// safeAssetSegment is the URL half of the same rule internal/book applies to
// the archive: one plain path segment, nothing that climbs out of it.
func safeAssetSegment(value string) bool {
	if value == "" || value == "." || value == ".." || len(value) > 64 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

// byteSeeker lets http.ServeContent do range requests over bytes already in
// memory without another copy.
type byteSeeker struct {
	data []byte
	pos  int64
}

func newByteSeeker(data []byte) *byteSeeker { return &byteSeeker{data: data} }

func (b *byteSeeker) Read(p []byte) (int, error) {
	if b.pos >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.pos:])
	b.pos += int64(n)
	return n, nil
}

func (b *byteSeeker) Seek(offset int64, whence int) (int64, error) {
	var next int64
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = b.pos + offset
	case io.SeekEnd:
		next = int64(len(b.data)) + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}
	if next < 0 {
		return 0, fmt.Errorf("negative position")
	}
	b.pos = next
	return next, nil
}
