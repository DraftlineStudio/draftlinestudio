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
	maxHistoryPerChapter = 50
	maxHistoryTotal      = 1000
)

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

func loadHistory(path string) (historyArchive, error) {
	state := historyArchive{Entries: []types.ChapterHistoryEntry{}, Contents: map[string][]byte{}}
	if strings.TrimSpace(path) == "" {
		return state, nil
	}
	r, err := zip.OpenReader(path)
	if os.IsNotExist(err) {
		return state, nil // A first save has no archive to preserve.
	}
	if err != nil {
		return state, fmt.Errorf("cannot preserve chapter history: %w", err)
	}
	defer func() { _ = r.Close() }()
	if err := ziputil.CheckArchive(r.File); err != nil {
		return state, fmt.Errorf("cannot preserve chapter history: %w", err)
	}
	entries, err := readHistoryIndex(r.File)
	if err != nil {
		return state, err
	}
	state.Entries = entries
	var total int64
	for _, entry := range state.Entries {
		content, err := ziputil.ReadNamed(r.File, entry.File, false)
		if err != nil {
			return state, fmt.Errorf("cannot read chapter history snapshot %q: %w", entry.ID, err)
		}
		total += int64(len(content))
		if total > ziputil.MaxTotalSize {
			return state, fmt.Errorf("chapter history expands beyond %d bytes", ziputil.MaxTotalSize)
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

// copyHistoryEntries preserves unchanged history without inflating it into
// memory or recompressing it. This is the hot path for ordinary autosaves.
func copyHistoryEntries(w *zip.Writer, path string) error {
	r, err := zip.OpenReader(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot preserve chapter history: %w", err)
	}
	defer func() { _ = r.Close() }()
	if err := ziputil.CheckArchive(r.File); err != nil {
		return fmt.Errorf("cannot preserve chapter history: %w", err)
	}
	for _, file := range r.File {
		if file.Name != historyIndexFile && !strings.HasPrefix(file.Name, "history/snapshots/") {
			continue
		}
		if err := w.Copy(file); err != nil {
			return fmt.Errorf("cannot preserve chapter history entry: %w", err)
		}
	}
	return nil
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
