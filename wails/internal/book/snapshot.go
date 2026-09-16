package book

// Freezing a manuscript.
//
// A snapshot is the book as it stood at one moment, filed in the archive under
// the hash of its own contents:
//
//	editions/snapshots/<sha256>/index.json      order, titles, metadata
//	editions/snapshots/<sha256>/copyright.html  the author's own copyright page
//	editions/snapshots/<sha256>/000.html        section text, in manuscript order
//
// Content addressing is the whole reason the same text exported as a paperback
// and as an ebook is stored once: both format records name the same folder.
// Per-edition folders could not do that, and a sixty-section novel frozen for
// six formats is three hundred and sixty archive members that did not need to
// exist.
//
// The hash is taken over the ordered content and nothing else. In particular
// it is not taken over the moment of freezing, or over the metadata's Modified
// stamp, which is a fact about the last save and not about the text: if it
// counted, exporting twice in one afternoon would freeze two copies of an
// identical book.
//
// The members are written through the byte path from 02645 — stored, not
// deflated, because HTML deflates once on the way in and there is nothing to
// gain from doing it again — and they survive a save by the editions/ prefix
// in preservedArchivePrefixes.

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"sort"
	"strings"
	"time"

	"draftline/internal/types"
	"draftline/internal/ziputil"
)

const (
	// snapshotsPrefix is the folder every frozen manuscript lives under. It
	// sits inside editions/ so that the passthrough already carries it.
	snapshotsPrefix = editionsPrefix + "snapshots/"
	// snapshotsSegment is the one edition identifier that is not available to
	// an author, because it is this folder's name.
	snapshotsSegment = "snapshots"

	snapshotIndexName     = "index.json"
	snapshotCopyrightName = "copyright.html"
	snapshotDataVersion   = 1

	// maxSnapshotSections bounds what one frozen manuscript may hold. It is
	// the same order as the chapter-history limit and exists for the same
	// reason: the file being read is a ZIP anybody can edit.
	maxSnapshotSections = 2000
)

// snapshotEntry is one section inside a frozen manuscript. It carries the
// title and the subtitle as well as the text, because the order and the titles
// are part of what "as it was" means — a chapter renamed after publication did
// not have that name in the book that went out.
type snapshotEntry struct {
	ID       string `json:"id,omitempty"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Type     string `json:"type"`
	File     string `json:"file"`
}

// snapshotManifest is index.json inside one snapshot folder.
type snapshotManifest struct {
	Version int `json:"version"`
	// Metadata is the book record as it stood, so an export of this snapshot
	// prints the title, the author and the publisher of the book that went
	// out rather than the one on screen today.
	Metadata types.Metadata `json:"metadata"`
	// Copyright is the member holding the author's own copyright page, or ""
	// when the book had none.
	Copyright   string          `json:"copyright,omitempty"`
	FrontMatter []snapshotEntry `json:"front_matter"`
	Body        []snapshotEntry `json:"body"`
	BackMatter  []snapshotEntry `json:"back_matter"`
}

// Snapshot is a frozen manuscript on its way into the archive: the catalogue
// record that goes in the publishing index, and the members that hold the text.
type Snapshot struct {
	Record types.EditionSnapshot
	// Files maps a full archive member name to its bytes.
	Files map[string][]byte
}

// SnapshotPrefix is the folder one frozen manuscript's members live under.
func SnapshotPrefix(id string) string { return snapshotsPrefix + id + "/" }

// SnapshotMember is the full archive member name of one file inside a snapshot.
func SnapshotMember(id, name string) string { return SnapshotPrefix(id) + name }

// BuildSnapshot freezes a book.
//
// The result is deterministic: the same sections, in the same order, with the
// same titles and the same metadata, always produce the same identifier and
// byte-identical members. `frozen` is the moment to record, and it is the one
// thing in the record that the identifier does not depend on.
func BuildSnapshot(b types.BookData, frozen time.Time) Snapshot {
	// Word count is recomputed rather than trusted: metadata.word_count is
	// filled in by whichever save last ran, and a snapshot's count has to
	// describe the text in the snapshot.
	counted := b
	RefreshWordCount(&counted)

	meta := counted.Metadata
	// Modified says when the project file was last written. It is not a fact
	// about the frozen text and must not change its identity.
	meta.Modified = ""
	meta.NormalizeISBNs()

	manifest := snapshotManifest{
		Version:     snapshotDataVersion,
		Metadata:    meta,
		FrontMatter: []snapshotEntry{},
		Body:        []snapshotEntry{},
		BackMatter:  []snapshotEntry{},
	}
	files := map[string][]byte{}

	if strings.TrimSpace(b.Copyright) != "" {
		manifest.Copyright = snapshotCopyrightName
		files[snapshotCopyrightName] = []byte(b.Copyright)
	}

	next := 0
	freeze := func(items []types.ChapterItem) []snapshotEntry {
		out := make([]snapshotEntry, 0, len(items))
		for _, item := range items {
			name := fmt.Sprintf("%03d.html", next)
			next++
			files[name] = []byte(item.Content)
			out = append(out, snapshotEntry{
				ID: item.ID, Title: item.Title, Subtitle: item.Subtitle,
				Type: item.Type, File: name,
			})
		}
		return out
	}
	manifest.FrontMatter = freeze(b.FrontMatter)
	manifest.Body = freeze(b.Body)
	manifest.BackMatter = freeze(b.BackMatter)

	indexJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		// snapshotManifest holds only strings, ints and slices of them; there
		// is no value of it that cannot be encoded. Failing loudly here beats
		// filing a book under the hash of an empty document.
		panic(fmt.Sprintf("snapshot manifest cannot be encoded: %v", err))
	}
	files[snapshotIndexName] = indexJSON

	id := snapshotDigest(files)

	members := map[string][]byte{}
	var bytes int64
	for name, data := range files {
		members[SnapshotMember(id, name)] = data
		bytes += int64(len(data))
	}

	return Snapshot{
		Record: types.EditionSnapshot{
			ID:        id,
			Frozen:    frozen.Format(time.RFC3339),
			Title:     strings.TrimSpace(meta.Title),
			WordCount: meta.WordCount,
			Sections:  len(manifest.FrontMatter) + len(manifest.Body) + len(manifest.BackMatter),
			Members:   len(members),
			Bytes:     bytes,
		},
		Files: members,
	}
}

// snapshotDigest hashes a snapshot's files.
//
// Every name and every body goes in length-prefixed, so that no rearrangement
// of the same bytes across different members can produce the same digest — a
// chapter ending where the next one begins is exactly the sort of thing a
// plain concatenation would confuse.
func snapshotDigest(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	sum := sha256.New()
	writeChunk(sum, []byte("draftline/edition-snapshot/1"))
	for _, name := range names {
		writeChunk(sum, []byte(name))
		writeChunk(sum, files[name])
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func writeChunk(h hash.Hash, data []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(data)))
	_, _ = h.Write(length[:])
	_, _ = h.Write(data)
}

// DecodeSnapshot rebuilds a book from one frozen manuscript.
//
// fetch reads a member by its name inside the snapshot folder. The caller
// decides where those bytes come from — the open project file, or a snapshot
// frozen a moment ago and not saved yet — so that this stays the one place
// that knows the shape of the thing.
//
// The result is a plain types.BookData. That is the point: every exporter
// already takes one, so exporting a frozen edition is the same export path
// handed a different book, not a second path that has to be kept in step.
func DecodeSnapshot(id string, fetch func(name string) ([]byte, error)) (types.BookData, error) {
	if !safeArchiveSegment(id) {
		return types.BookData{}, fmt.Errorf("%q does not name a frozen manuscript in this project", id)
	}
	indexData, err := fetch(snapshotIndexName)
	if err != nil {
		return types.BookData{}, fmt.Errorf("the frozen text of this edition could not be read: %w", err)
	}
	var manifest snapshotManifest
	if err := json.Unmarshal(indexData, &manifest); err != nil {
		return types.BookData{}, fmt.Errorf("the frozen text of this edition could not be read: %w", err)
	}
	if err := requireArchiveVersion(SnapshotMember(id, snapshotIndexName), manifest.Version, snapshotDataVersion); err != nil {
		return types.BookData{}, err
	}

	total := len(manifest.FrontMatter) + len(manifest.Body) + len(manifest.BackMatter)
	if total > maxSnapshotSections {
		return types.BookData{}, fmt.Errorf("the frozen text of this edition lists %d sections (limit %d)", total, maxSnapshotSections)
	}

	book := types.BookData{
		Version:     "2.2",
		Metadata:    manifest.Metadata,
		FrontMatter: []types.ChapterItem{},
		Body:        []types.ChapterItem{},
		BackMatter:  []types.ChapterItem{},
	}
	if name := strings.TrimSpace(manifest.Copyright); name != "" {
		if !safeArchiveSegment(name) {
			return types.BookData{}, fmt.Errorf("the frozen text of this edition names an unreadable file %q", name)
		}
		data, err := fetch(name)
		if err != nil {
			return types.BookData{}, fmt.Errorf("the frozen copyright page of this edition could not be read: %w", err)
		}
		book.Copyright = string(data)
	}

	load := func(entries []snapshotEntry) ([]types.ChapterItem, error) {
		items := make([]types.ChapterItem, 0, len(entries))
		for _, entry := range entries {
			if !safeArchiveSegment(entry.File) {
				return nil, fmt.Errorf("the frozen text of this edition names an unreadable file %q", entry.File)
			}
			data, err := fetch(entry.File)
			if err != nil {
				return nil, fmt.Errorf("the frozen text of this edition is missing %q: %w", entry.Title, err)
			}
			items = append(items, types.ChapterItem{
				ID: entry.ID, Title: entry.Title, Subtitle: entry.Subtitle,
				Type: entry.Type, Content: string(data),
			})
		}
		return items, nil
	}
	if book.FrontMatter, err = load(manifest.FrontMatter); err != nil {
		return types.BookData{}, err
	}
	if book.Body, err = load(manifest.Body); err != nil {
		return types.BookData{}, err
	}
	if book.BackMatter, err = load(manifest.BackMatter); err != nil {
		return types.BookData{}, err
	}
	return book, nil
}

// ReadSnapshot rebuilds a frozen manuscript out of a project file.
//
// The archive is opened once and every member of the snapshot read from that
// one reader, which matters: a sixty-chapter book is sixty-two members, and
// opening the ZIP for each of them would read its central directory sixty-two
// times for a file the author is waiting on.
func ReadSnapshot(archivePath, id string) (types.BookData, error) {
	if strings.TrimSpace(archivePath) == "" {
		return types.BookData{}, fmt.Errorf("no project is open")
	}
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return types.BookData{}, fmt.Errorf("the frozen text of this edition could not be read: %w", err)
	}
	defer func() { _ = r.Close() }()
	if err := ziputil.CheckArchive(r.File); err != nil {
		return types.BookData{}, err
	}
	prefix := SnapshotPrefix(id)
	return DecodeSnapshot(id, func(name string) ([]byte, error) {
		return ziputil.ReadNamed(r.File, prefix+name, false)
	})
}

// ── Retention ──────────────────────────────────────────────────────────────

// prepareSnapshotCatalogue settles which frozen manuscripts this save keeps.
//
// The rule has two halves, and both failures it guards against are bad in
// different ways. A catalogue record nothing points at is dropped, and its
// bytes with it: a novel's worth of HTML that no ISBN refers to would
// otherwise be copied into the project file and into every rolling backup on
// every autosave, for the life of the book. A format that still points at a
// record the catalogue has lost is refused outright, because the alternative
// is dropping the text a published ISBN stands for on the strength of a
// bookkeeping mistake.
func prepareSnapshotCatalogue(index *types.EditionIndex) error {
	referenced := index.ReferencedSnapshots()
	catalogued := map[string]bool{}

	kept := make([]types.EditionSnapshot, 0, len(index.Snapshots))
	for _, snapshot := range index.Snapshots {
		snapshot.ID = strings.TrimSpace(snapshot.ID)
		if snapshot.ID == "" {
			return fmt.Errorf("%s holds a frozen manuscript with no identifier", editionsIndexFile)
		}
		if !safeArchiveSegment(snapshot.ID) {
			return fmt.Errorf(
				"%s files a frozen manuscript under %q, which cannot be a folder name inside the project file",
				editionsIndexFile, snapshot.ID)
		}
		if catalogued[snapshot.ID] {
			return fmt.Errorf("%s holds two frozen manuscripts with the identifier %q", editionsIndexFile, snapshot.ID)
		}
		catalogued[snapshot.ID] = true
		snapshot.Frozen = strings.TrimSpace(snapshot.Frozen)
		snapshot.Title = strings.TrimSpace(snapshot.Title)
		if !referenced[snapshot.ID] {
			// Nothing was exported from it any more. The bytes go with it,
			// below, by simply not being carried across.
			continue
		}
		kept = append(kept, snapshot)
	}

	for id := range referenced {
		if !catalogued[id] {
			return fmt.Errorf(
				"%s says a format was exported from the frozen manuscript %q, which is not in this book's list of them. Draftline will not drop text a published ISBN refers to.",
				editionsIndexFile, id)
		}
	}
	index.Snapshots = kept
	return nil
}

// liveSnapshotIDs is the set of frozen manuscripts whose bytes this save keeps.
func liveSnapshotIDs(index types.EditionIndex) map[string]bool {
	live := map[string]bool{}
	for _, snapshot := range index.Snapshots {
		live[snapshot.ID] = true
	}
	return live
}

// snapshotFolder names the frozen manuscript an archive member belongs to.
func snapshotFolder(name string) (string, bool) {
	rest, ok := strings.CutPrefix(name, snapshotsPrefix)
	if !ok {
		return "", false
	}
	id, _, ok := strings.Cut(rest, "/")
	if !ok || id == "" {
		return "", false
	}
	return id, true
}

// orphanedSnapshotMember reports whether a member belongs to a frozen
// manuscript no format refers to any more.
//
// A nil set means this save does not know the publishing record and must not
// reap — the same rule, and the same reason, as orphanedEditionMember: a book
// saved with no record in hand does not rewrite editions/index.json either, so
// everything the old index named is still real.
func orphanedSnapshotMember(name string, live map[string]bool) bool {
	if live == nil {
		return false
	}
	id, ok := snapshotFolder(name)
	return ok && !live[id]
}
