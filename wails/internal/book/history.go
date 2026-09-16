package book

import (
	"archive/zip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"draftline/internal/types"
	"draftline/internal/ziputil"
)

const (
	historyIndexFile     = "history/index.json"
	historyPrefix        = "history/"
	editionsPrefix       = "editions/"
	maxHistoryPerChapter = 50
	maxHistoryTotal      = 1000
)

// preservedArchivePrefixes lists the archive members a save carries over from
// the book it is saving. Every save rebuilds the .draftline from scratch, so a
// member under no prefix in this list is destroyed by the next autosave, five
// seconds after the next keystroke. Anything written into the archive that the
// app does not rebuild from BookData belongs here.
var preservedArchivePrefixes = []string{historyPrefix, editionsPrefix}

// preservedPrefixesExcept returns the passthrough list without one prefix, for
// the save branch that is already rewriting that prefix from memory.
func preservedPrefixesExcept(skip string) []string {
	kept := make([]string, 0, len(preservedArchivePrefixes))
	for _, prefix := range preservedArchivePrefixes {
		if prefix != skip {
			kept = append(kept, prefix)
		}
	}
	return kept
}

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

type historyArchive struct {
	Entries  []types.ChapterHistoryEntry `json:"entries"`
	Contents map[string][]byte           `json:"-"`
}

func newArchiveID(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err == nil {
		return prefix + hex.EncodeToString(b)
	}
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
}

func ensureChapterIDs(items []types.ChapterItem, seen map[string]bool) {
	for i := range items {
		id := strings.TrimSpace(items[i].ID)
		if id == "" || seen[id] {
			items[i].ID = newArchiveID("ch-")
		}
		seen[items[i].ID] = true
	}
}

func EnsureBookChapterIDs(book *types.BookData) {
	seen := map[string]bool{}
	ensureChapterIDs(book.FrontMatter, seen)
	ensureChapterIDs(book.Body, seen)
	ensureChapterIDs(book.BackMatter, seen)
}

func contentDigest(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func historyWordCount(content string) int {
	plain := html.UnescapeString(htmlTagPattern.ReplaceAllString(content, " "))
	return len(strings.FieldsFunc(plain, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsNumber(r) || r == '\'' || r == '’')
	}))
}

// loadHistory reads the chapter history of the book being saved FROM. During a
// Save As that is the open project, not the file about to be written.
func loadHistory(sourcePath string) (historyArchive, error) {
	state := historyArchive{Entries: []types.ChapterHistoryEntry{}, Contents: map[string][]byte{}}
	if strings.TrimSpace(sourcePath) == "" {
		return state, nil
	}
	r, err := zip.OpenReader(sourcePath)
	if os.IsNotExist(err) {
		return state, nil // A first save has no archive to preserve.
	}
	if err != nil {
		return state, unreadableSource(fmt.Errorf("cannot preserve chapter history: %w", err))
	}
	defer func() { _ = r.Close() }()
	if err := ziputil.CheckArchive(r.File); err != nil {
		return state, unreadableSource(fmt.Errorf("cannot preserve chapter history: %w", err))
	}
	entries, err := readHistoryIndex(r.File)
	if err != nil {
		return state, unreadableSource(err)
	}
	state.Entries = entries
	var total int64
	for _, entry := range state.Entries {
		content, err := ziputil.ReadNamed(r.File, entry.File, false)
		if err != nil {
			return state, unreadableSource(fmt.Errorf("cannot read chapter history snapshot %q: %w", entry.ID, err))
		}
		total += int64(len(content))
		if total > ziputil.MaxTotalSize {
			return state, unreadableSource(fmt.Errorf("chapter history expands beyond %d bytes", ziputil.MaxTotalSize))
		}
		state.Contents[entry.File] = content
	}
	return state, nil
}

func readHistoryIndex(files []*zip.File) ([]types.ChapterHistoryEntry, error) {
	data, err := ziputil.ReadNamed(files, historyIndexFile, false)
	if errors.Is(err, ziputil.ErrEntryNotFound) {
		return []types.ChapterHistoryEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read chapter history index: %w", err)
	}
	var index struct {
		Entries []types.ChapterHistoryEntry `json:"entries"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("cannot parse chapter history index: %w", err)
	}
	if len(index.Entries) > maxHistoryTotal {
		return nil, fmt.Errorf("chapter history has %d entries (limit %d)", len(index.Entries), maxHistoryTotal)
	}
	for _, entry := range index.Entries {
		if !strings.HasPrefix(entry.File, "history/snapshots/") || !strings.HasSuffix(entry.File, ".html") {
			return nil, fmt.Errorf("chapter history contains an invalid snapshot path")
		}
	}
	return index.Entries, nil
}

// sourceReadError marks a failure to read the project a save is reading
// preserved data OUT of, as opposed to a failure to write the file it is
// saving INTO. The distinction decides whether a save may carry on: during a
// Save As the source is left untouched, so whatever could not be read is still
// sitting in it, and refusing the save would strand the author's text in memory
// with no route to disk at all. The original message is kept unchanged, so a
// save that still refuses says exactly what it always said.
type sourceReadError struct{ err error }

func (e sourceReadError) Error() string { return e.err.Error() }
func (e sourceReadError) Unwrap() error { return e.err }

func unreadableSource(err error) error { return sourceReadError{err: err} }

// copyPreservedEntries carries members of the source archive into the archive
// being written, byte for byte and without inflating or recompressing them.
// This is the hot path for ordinary autosaves.
//
// sourcePath is the book being saved FROM, which during a Save As is not the
// file being written. A member whose name starts with none of the given
// prefixes is not copied, and because every save rebuilds the archive from
// scratch, not copying a member destroys it.
func copyPreservedEntries(w *archiveWriter, sourcePath string, prefixes []string) error {
	return copyPreservedEntriesExcept(w, sourcePath, prefixes, nil, nil, nil)
}

// copyPreservedEntriesExcept is the same, with a list of prefixes this save is
// replacing outright and the set of editions it still has.
//
// The exception exists because a replaced member does not always keep its
// name. Cover art is kept as a JPEG or, for flat artwork, as a PNG; attaching
// new art of the other kind writes cover.png beside a cover.jpg the passthrough
// would otherwise carry forward for ever. Naming the prefix drops the whole
// old set, and the assets written before this call have already claimed the
// names that survive.
//
// liveEditions is the other half of the same problem, for the case where it is
// not the artwork that went but the edition: see orphanedEditionMember.
// liveSnapshots is the same thing again for frozen manuscripts, which outlive
// any one edition and are kept by reference count instead.
func copyPreservedEntriesExcept(w *archiveWriter, sourcePath string, prefixes, superseded []string, liveEditions, liveSnapshots map[string]bool) error {
	if strings.TrimSpace(sourcePath) == "" || len(prefixes) == 0 {
		return nil
	}
	r, err := zip.OpenReader(sourcePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return unreadableSource(fmt.Errorf("cannot preserve archived data: %w", err))
	}
	defer func() { _ = r.Close() }()
	if err := ziputil.CheckArchive(r.File); err != nil {
		return unreadableSource(fmt.Errorf("cannot preserve archived data: %w", err))
	}
	for _, file := range r.File {
		if !hasAnyPrefix(file.Name, prefixes) || hasAnyPrefix(file.Name, superseded) {
			continue
		}
		if orphanedEditionMember(file.Name, liveEditions) || orphanedSnapshotMember(file.Name, liveSnapshots) {
			continue
		}
		if err := w.copyEntry(file); err != nil {
			return err
		}
	}
	return nil
}

func hasAnyPrefix(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func addHistorySnapshots(state *historyArchive, requests []types.ChapterSnapshotRequest) {
	now := time.Now()
	for offset, request := range requests {
		if strings.TrimSpace(request.ChapterID) == "" {
			continue
		}
		hash := contentDigest(request.Content)
		latest := -1
		for i := range state.Entries {
			if state.Entries[i].ChapterID == request.ChapterID &&
				(latest < 0 || state.Entries[i].CreatedAt > state.Entries[latest].CreatedAt) {
				latest = i
			}
		}
		if latest >= 0 && state.Entries[latest].ContentHash == hash {
			continue
		}
		id := newArchiveID("snap-")
		file := "history/snapshots/" + id + ".html"
		reason := strings.TrimSpace(request.Reason)
		if reason == "" {
			reason = "Writing session"
		}
		state.Entries = append(state.Entries, types.ChapterHistoryEntry{
			ID: id, ChapterID: request.ChapterID, Section: request.Section,
			ChapterTitle: request.ChapterTitle,
			CreatedAt:    now.Add(time.Duration(offset) * time.Nanosecond).Format(time.RFC3339Nano),
			Reason:       reason,
			WordCount:    historyWordCount(request.Content),
			ContentHash:  hash,
			File:         file,
		})
		state.Contents[file] = []byte(request.Content)
	}
	pruneHistory(state)
}

func pruneHistory(state *historyArchive) {
	sort.SliceStable(state.Entries, func(i, j int) bool { return state.Entries[i].CreatedAt > state.Entries[j].CreatedAt })
	kept := make([]types.ChapterHistoryEntry, 0, len(state.Entries))
	perChapter := map[string]int{}
	for _, entry := range state.Entries {
		if len(kept) >= maxHistoryTotal || perChapter[entry.ChapterID] >= maxHistoryPerChapter {
			delete(state.Contents, entry.File)
			continue
		}
		perChapter[entry.ChapterID]++
		kept = append(kept, entry)
	}
	state.Entries = kept
}

func ListChapterHistory(path, chapterID string) ([]types.ChapterHistoryEntry, error) {
	if strings.TrimSpace(path) == "" {
		return []types.ChapterHistoryEntry{}, nil
	}
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	if err := ziputil.CheckArchive(r.File); err != nil {
		return nil, err
	}
	all, err := readHistoryIndex(r.File)
	if err != nil {
		return nil, err
	}
	entries := make([]types.ChapterHistoryEntry, 0)
	for _, entry := range all {
		if entry.ChapterID == chapterID {
			entries = append(entries, entry)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].CreatedAt > entries[j].CreatedAt })
	return entries, nil
}

func GetChapterHistorySnapshot(path, snapshotID string) (types.ChapterHistorySnapshot, error) {
	if strings.TrimSpace(path) == "" {
		return types.ChapterHistorySnapshot{}, fmt.Errorf("no project is open")
	}
	r, err := zip.OpenReader(path)
	if err != nil {
		return types.ChapterHistorySnapshot{}, err
	}
	defer func() { _ = r.Close() }()
	if err := ziputil.CheckArchive(r.File); err != nil {
		return types.ChapterHistorySnapshot{}, err
	}
	entries, err := readHistoryIndex(r.File)
	if err != nil {
		return types.ChapterHistorySnapshot{}, err
	}
	for _, entry := range entries {
		if entry.ID == snapshotID {
			content, err := ziputil.ReadNamed(r.File, entry.File, false)
			if err != nil {
				return types.ChapterHistorySnapshot{}, fmt.Errorf("cannot read chapter history snapshot: %w", err)
			}
			return types.ChapterHistorySnapshot{Entry: entry, Content: string(content)}, nil
		}
	}
	return types.ChapterHistorySnapshot{}, fmt.Errorf("chapter history snapshot not found")
}
