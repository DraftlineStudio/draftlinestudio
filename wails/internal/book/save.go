package book

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"draftline/internal/backup"
	"draftline/internal/fsutil"
	"draftline/internal/types"
	"draftline/internal/ziputil"
)

// archiveWriter writes the members of a .draftline, tracking every name it has
// written. Two code paths add members — data rebuilt from BookData and data
// carried over from the book being saved — and a ZIP will happily hold the same
// name twice, with readers disagreeing about which copy wins. A collision is an
// error here instead.
//
// It also keeps a running count of what has gone in. Until frozen manuscripts
// existed, a .draftline could not plausibly reach the archive limits in
// ziputil; a sixty-section novel frozen for six formats can, so a save that
// would exceed one has to say which part of the file is responsible rather
// than refuse a project that opened yesterday with a number.
type archiveWriter struct {
	zw      *zip.Writer
	written map[string]bool
	bytes   int64
	// snapshotEntries and snapshotBytes are the share of the above that
	// belongs to frozen manuscripts, and snapshots counts how many distinct
	// ones they belong to.
	snapshotEntries int
	snapshotBytes   int64
	snapshots       map[string]bool
}

func newArchiveWriter(zw *zip.Writer) *archiveWriter {
	return &archiveWriter{zw: zw, written: map[string]bool{}, snapshots: map[string]bool{}}
}

func (a *archiveWriter) claim(name string) error {
	if a.written[name] {
		return fmt.Errorf("archive entry %q would be written twice", name)
	}
	a.written[name] = true
	return nil
}

// account records what one written member costs.
func (a *archiveWriter) account(name string, size int64) {
	a.bytes += size
	if id, ok := snapshotFolder(name); ok {
		a.snapshotEntries++
		a.snapshotBytes += size
		a.snapshots[id] = true
	}
}

// addEntry writes text, deflated.
func (a *archiveWriter) addEntry(name, content string) error {
	return a.addDeflated(name, []byte(content))
}

// addDeflated writes a member compressed, which is what everything made of
// words wants.
func (a *archiveWriter) addDeflated(name string, data []byte) error {
	if err := a.claim(name); err != nil {
		return err
	}
	f, err := a.zw.Create(name)
	if err != nil {
		return err
	}
	a.account(name, int64(len(data)))
	_, err = f.Write(data)
	return err
}

// addAsset writes a member handed over in memory, compressed or not according
// to what it is made of.
//
// Cover art arrives as a JPEG or a PNG, which is compressed already: deflating
// it again costs processor time on every autosave and saves nothing. A frozen
// manuscript is HTML and JSON, which deflates to about a fifth of its size —
// and a project file holding several of them is copied whole into a rolling
// backup before every single save, so that fifth is paid over and over.
func (a *archiveWriter) addAsset(name string, data []byte) error {
	if arrivesCompressed(name) {
		return a.addBytes(name, data)
	}
	return a.addDeflated(name, data)
}

// arrivesCompressed reports whether a member's bytes were compressed before
// they reached this program.
func arrivesCompressed(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif":
		return true
	}
	return false
}

// addBytes writes a binary asset stored rather than deflated. Cover art and
// other already-compressed formats gain nothing from a second pass, and
// autosave runs five seconds after every edit.
func (a *archiveWriter) addBytes(name string, data []byte) error {
	if err := a.claim(name); err != nil {
		return err
	}
	f, err := a.zw.CreateHeader(&zip.FileHeader{
		Name:     name,
		Method:   zip.Store,
		Modified: time.Now(),
	})
	if err != nil {
		return err
	}
	a.account(name, int64(len(data)))
	_, err = f.Write(data)
	return err
}

// copyEntry carries one member over from the source archive without inflating
// or recompressing it.
//
// A name already written is skipped rather than refused. The collision guard is
// there to catch this program writing one name twice, but the source archive
// was not necessarily written by this program — the format is shared with the
// Python-era app — and a ZIP holding the same name twice still opens. Refusing
// it would turn a book that loads, and accepts edits, into a book that no save
// and no Save As can ever write again. A reader takes the first copy of a
// repeated name, so the save keeps the first copy too, and the file comes out
// of the save without the repeat.
func (a *archiveWriter) copyEntry(f *zip.File) error {
	if a.written[f.Name] {
		return nil
	}
	if err := a.claim(f.Name); err != nil {
		return err
	}
	a.account(f.Name, int64(f.UncompressedSize64))
	if err := a.zw.Copy(f); err != nil {
		return fmt.Errorf("cannot preserve archived entry %q: %w", f.Name, err)
	}
	return nil
}

// budget refuses a save that would produce a project file this build, or any
// other reader of the format, would then refuse to open.
//
// The check is here rather than only in ziputil so that the message names what
// filled the file. "archive exceeds safety limits" on a project that saved
// fine an hour ago tells an author nothing they can act on; the number of
// frozen editions in it, and the fact that dropping one releases the space,
// tells them exactly what to do.
func (a *archiveWriter) budget() error {
	if len(a.written) > ziputil.MaxEntries {
		return fmt.Errorf(
			"this project would hold %d pieces, more than the %d a Draftline file can contain%s",
			len(a.written), ziputil.MaxEntries, a.snapshotShare(int64(a.snapshotEntries), int64(len(a.written))))
	}
	if a.bytes > ziputil.MaxTotalSize {
		return fmt.Errorf(
			"this project would come to %s, more than the %s a Draftline file can hold%s",
			byteSize(a.bytes), byteSize(ziputil.MaxTotalSize), a.snapshotShare(a.snapshotBytes, a.bytes))
	}
	return nil
}

// snapshotShare is the sentence that names frozen manuscripts as the cause,
// and it is only added when they are actually most of the problem. A project
// that is simply enormous on its own should not be told to delete editions.
func (a *archiveWriter) snapshotShare(part, whole int64) string {
	if whole == 0 || part*2 < whole || len(a.snapshots) == 0 {
		return ". Remove some of what it holds and save again."
	}
	return fmt.Sprintf(
		". The %d frozen %s in it account for most of that — release one from a format on the Book & Editions screen and the space comes back on the next save.",
		len(a.snapshots), plural(len(a.snapshots), "manuscript", "manuscripts"))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func byteSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/float64(1<<20))
	default:
		return fmt.Sprintf("%.0f KB", float64(n)/float64(1<<10))
	}
}

// toleratedSourceFailure decides what an unreadable source means for this save.
//
// Saving over the source itself still refuses: the data that could not be read
// is in the file about to be overwritten, and overwriting it is exactly what
// would destroy it. Saving anywhere else leaves the source where it is, so the
// save carries nothing across, says so, and writes the book — an author whose
// project file has gone unreadable, because a network share dropped or a backup
// tool has it open, needs a route to disk more than a tidy archive.
func toleratedSourceFailure(err error, sourcePath, destPath string) (string, bool) {
	var unreadable sourceReadError
	if !errors.As(err, &unreadable) || sameArchiveFile(sourcePath, destPath) {
		return "", false
	}
	return fmt.Sprintf("%s could not be read, so its chapter history and anything else stored in it did not come across. That file still holds them.", filepath.Base(sourcePath)), true
}

// sameArchiveFile reports whether two paths name one file on disk, so that a
// save spelled differently from the open project is not mistaken for a Save As.
func sameArchiveFile(a, b string) bool {
	if strings.TrimSpace(a) == "" || strings.TrimSpace(b) == "" {
		return false
	}
	if a == b {
		return true
	}
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(ai, bi)
}

// Assets are binary archive members this save writes from memory rather than
// carrying across from the book it is saving.
//
// Cover art is the first of them. Bytes never reach BookData - see
// types/cover.go - so they arrive here beside it instead, and they only arrive
// at all when they have changed: an unchanged cover survives by the passthrough
// in history.go, stored and never re-encoded.
type Assets struct {
	// Files maps a full archive member name to its contents. Whether one is
	// deflated depends on what it holds; see addAsset.
	Files map[string][]byte
	// Superseded lists member-name prefixes this save replaces. A carried-over
	// member under one of them is dropped instead of preserved, which is what
	// lets a cover be replaced by one in a different format: attaching a flat
	// design as cover.png must not leave the old cover.jpg behind it.
	Superseded []string
	// Written, when it is not nil, is filled in with every member name this
	// save actually wrote.
	//
	// It exists because not everything handed over is written: a member filed
	// under an edition, or a frozen manuscript, that the saved book does not
	// name is skipped below. The caller is holding the only copy of those
	// bytes, and a successful save is not proof that its own copy is safe to
	// let go of — so it is told which names reached the file rather than left
	// to infer it.
	Written map[string]bool
}

// Write saves a BookData to a .draftline file at destPath. sourcePath is the
// project the book is open from, which a Save As carries chapter history and
// other preserved members over from; it is empty for a book never yet saved.
// appVersion should be the current application version string.
func Write(sourcePath, destPath string, book types.BookData, appVersion string) types.SaveResult {
	return WriteArchive(sourcePath, destPath, book, appVersion, nil, Assets{})
}

// WriteWithSnapshots saves the book while preserving its embedded chapter
// history and optionally appending changed chapter snapshots. History is read
// from sourcePath and written to destPath.
func WriteWithSnapshots(sourcePath, destPath string, book types.BookData, appVersion string, snapshots []types.ChapterSnapshotRequest) types.SaveResult {
	return WriteArchive(sourcePath, destPath, book, appVersion, snapshots, Assets{})
}

// WriteArchive is the whole save: the book, optional chapter snapshots, and
// any binary assets that changed since the last one.
func WriteArchive(sourcePath, destPath string, book types.BookData, appVersion string, snapshots []types.ChapterSnapshotRequest, assets Assets) types.SaveResult {
	// warnings record what the save could not do without failing outright. The
	// same unreadable source is reported once however many reads it defeats.
	var warnings []string
	warn := func(note string) {
		for _, existing := range warnings {
			if existing == note {
				return
			}
		}
		warnings = append(warnings, note)
	}

	history := historyArchive{Entries: []types.ChapterHistoryEntry{}, Contents: map[string][]byte{}}
	if len(snapshots) > 0 {
		loaded, err := loadHistory(sourcePath)
		if err != nil {
			note, tolerated := toleratedSourceFailure(err, sourcePath, destPath)
			if !tolerated {
				return types.SaveResult{Success: false, Error: err.Error()}
			}
			warn(note)
		} else {
			history = loaded
		}
		// The snapshots in hand are this session's writing and owe nothing to
		// the old file, so they are kept even when none of it could be read.
		addHistorySnapshots(&history, snapshots)
	}
	EnsureBookChapterIDs(&book)

	// Create backup of existing file before overwriting
	if err := backup.Create(destPath); err != nil {
		// Log but don't fail the save - backup is best-effort
		fmt.Printf("Backup warning: %v\n", err)
	}

	book.Metadata.Modified = time.Now().Format(time.RFC3339)
	// Keep the legacy single-ISBN field mirroring the list so Python-era
	// readers of the format still see an ISBN.
	book.Metadata.NormalizeISBNs()
	RefreshWordCount(&book)

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	type entry struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Subtitle string `json:"subtitle,omitempty"`
		Type     string `json:"type"`
		File     string `json:"file"`
	}
	type manifest struct {
		Version      string                    `json:"version"`
		AppVersion   string                    `json:"app_version"`
		Metadata     types.Metadata            `json:"metadata"`
		FrontMatter  []entry                   `json:"front_matter"`
		Body         []entry                   `json:"body"`
		BackMatter   []entry                   `json:"back_matter"`
		WritingGoals types.WritingGoals        `json:"writing_goals,omitempty"`
		StyleOptions types.WritingStyleOptions `json:"style_options,omitempty"`
		IsIndexed    bool                      `json:"is_indexed,omitempty"`
		LastIndexed  string                    `json:"last_indexed,omitempty"`
	}

	mf := manifest{
		Version:      "2.2",
		AppVersion:   appVersion,
		Metadata:     book.Metadata,
		FrontMatter:  []entry{},
		IsIndexed:    book.IsIndexed,
		LastIndexed:  book.LastIndexed,
		Body:         []entry{},
		BackMatter:   []entry{},
		WritingGoals: book.WritingGoals,
		StyleOptions: book.StyleOptions,
	}

	aw := newArchiveWriter(w)
	addEntry := aw.addEntry

	// The publishing record is written BEFORE the passthrough below, not after
	// it. Everything under editions/ is carried across a save, which is what
	// keeps cover art and frozen manuscripts alive; the index is the one
	// member under that prefix this save rebuilds, so it claims its own name
	// first and the passthrough then skips the copy in the old file. Written
	// after, it would collide with the carried-over copy and fail the save.
	//
	// liveEditions is nil until the publishing record has been read, and stays
	// nil for a book that has none. Nil means "do not reap"; see
	// orphanedEditionMember for why that distinction has to be kept.
	var liveEditions map[string]bool
	// liveSnapshots is the other reaping set: the frozen manuscripts at least
	// one format was exported from. Nil for the same reason, and the bytes of
	// anything not in it are simply not carried across this save.
	var liveSnapshots map[string]bool
	if book.Editions != nil {
		editions, err := prepareEditionsData(*book.Editions)
		if err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		liveEditions = liveEditionIDs(editions)
		liveSnapshots = liveSnapshotIDs(editions)
		editionsJSON, err := json.MarshalIndent(editions, "", "  ")
		if err != nil {
			return types.SaveResult{Success: false, Error: fmt.Sprintf("failed to encode editions: %v", err)}
		}
		if err := addEntry(editionsIndexFile, string(editionsJSON)); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	}

	// Binary assets go in before the passthrough, for the same reason the
	// editions index does: they claim their names first, so the passthrough
	// skips the copies in the old file instead of colliding with them.
	//
	// An asset filed under an edition this book does not have is not written
	// at all. The cover of an edition deleted a moment ago is the honest case;
	// a cover still held in memory from a project that was closed or replaced
	// is the dangerous one, because there is no screen in the app that could
	// ever show the author that another book's artwork is inside their file.
	for name, data := range assets.Files {
		if orphanedEditionMember(name, liveEditions) || orphanedSnapshotMember(name, liveSnapshots) {
			continue
		}
		if err := aw.addAsset(name, data); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		if assets.Written != nil {
			assets.Written[name] = true
		}
	}

	// Carry over everything this save does not rebuild. The snapshot branch is
	// rewriting chapter history from memory, so it preserves the rest.
	preserved := preservedArchivePrefixes
	if len(snapshots) > 0 {
		preserved = preservedPrefixesExcept(historyPrefix)
	}
	if err := copyPreservedEntriesExcept(aw, sourcePath, preserved, assets.Superseded, liveEditions, liveSnapshots); err != nil {
		note, tolerated := toleratedSourceFailure(err, sourcePath, destPath)
		if !tolerated {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		warn(note)
	}
	if len(snapshots) > 0 {
		for _, historyEntry := range history.Entries {
			content, ok := history.Contents[historyEntry.File]
			if !ok {
				return types.SaveResult{Success: false, Error: "chapter history content is missing"}
			}
			if err := addEntry(historyEntry.File, string(content)); err != nil {
				return types.SaveResult{Success: false, Error: err.Error()}
			}
		}
	}
	if len(snapshots) > 0 && len(history.Entries) > 0 {
		historyJSON, err := json.MarshalIndent(struct {
			Entries []types.ChapterHistoryEntry `json:"entries"`
		}{Entries: history.Entries}, "", "  ")
		if err != nil {
			return types.SaveResult{Success: false, Error: fmt.Sprintf("failed to encode chapter history: %v", err)}
		}
		if err := addEntry(historyIndexFile, string(historyJSON)); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	}

	if err := addEntry("copyright.html", book.Copyright); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}
	if book.StoryBible.Characters == nil {
		book.StoryBible.Characters = []types.Character{}
	}
	bibleJSON, _ := json.MarshalIndent(book.StoryBible, "", "  ")
	if err := addEntry("story_bible.json", string(bibleJSON)); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}

	// Analysis is stored separately from the author-owned story bible. It is
	// rebuildable, but persisting it keeps mention locations, relationships,
	// and manual merge/split decisions available after reopening a project.
	analysisJSON, err := json.MarshalIndent(book.Analysis, "", "  ")
	if err != nil {
		return types.SaveResult{Success: false, Error: fmt.Sprintf("failed to encode analysis: %v", err)}
	}
	if err := addEntry("analysis.json", string(analysisJSON)); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}

	// Save beat_sheet.json if there are beats
	if len(book.BeatSheet.Beats) > 0 {
		beatJSON, _ := json.MarshalIndent(book.BeatSheet, "", "  ")
		if err := addEntry("beat_sheet.json", string(beatJSON)); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	}

	// Save foreshadowing.json if there are items
	if len(book.ForeshadowingLedger.Items) > 0 {
		foreshadowJSON, _ := json.MarshalIndent(book.ForeshadowingLedger, "", "  ")
		if err := addEntry("foreshadowing.json", string(foreshadowJSON)); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	}

	// Save knowledge_matrix.json if there are secrets or entries
	if len(book.KnowledgeMatrix.Secrets) > 0 || len(book.KnowledgeMatrix.Entries) > 0 {
		matrixJSON, _ := json.MarshalIndent(book.KnowledgeMatrix, "", "  ")
		if err := addEntry("knowledge_matrix.json", string(matrixJSON)); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	}

	// Save read_aloud_cast.json if cast mode was ever configured
	if book.ReadAloudCast.CastMode || len(book.ReadAloudCast.Voices) > 0 {
		castJSON, _ := json.MarshalIndent(book.ReadAloudCast, "", "  ")
		if err := addEntry("read_aloud_cast.json", string(castJSON)); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	}

	// Save planner.json once the Planner has been used for this book.
	if book.Planner != nil {
		planner, err := preparePlannerData(*book.Planner)
		if err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		plannerJSON, err := json.MarshalIndent(planner, "", "  ")
		if err != nil {
			return types.SaveResult{Success: false, Error: fmt.Sprintf("failed to encode planner data: %v", err)}
		}
		if err := addEntry("planner.json", string(plannerJSON)); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	}

	for i, item := range book.FrontMatter {
		file := fmt.Sprintf("front_matter/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		mf.FrontMatter = append(mf.FrontMatter, entry{ID: item.ID, Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, File: file})
	}
	for i, item := range book.Body {
		file := fmt.Sprintf("body/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		mf.Body = append(mf.Body, entry{ID: item.ID, Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, File: file})
	}
	for i, item := range book.BackMatter {
		file := fmt.Sprintf("back_matter/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		mf.BackMatter = append(mf.BackMatter, entry{ID: item.ID, Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, File: file})
	}

	manifestBytes, _ := json.MarshalIndent(mf, "", "  ")
	if err := addEntry("manifest.json", string(manifestBytes)); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}

	// Asked before the archive is finalised, so that a project which has grown
	// past what the format can hold says what filled it rather than failing
	// the generic check below with a number.
	if err := aw.budget(); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}

	if err := w.Close(); err != nil {
		return types.SaveResult{Success: false, Error: fmt.Sprintf("failed to finalize archive: %v", err)}
	}

	// Validate the produced archive before letting it near the user's file.
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return types.SaveResult{Success: false, Error: fmt.Sprintf("internal error: produced archive is invalid: %v", err)}
	}
	if _, err := ziputil.ReadNamed(zr.File, "manifest.json", false); err != nil {
		return types.SaveResult{Success: false, Error: "internal error: produced archive is missing manifest.json"}
	}
	if err := ziputil.CheckArchive(zr.File); err != nil {
		return types.SaveResult{Success: false, Error: fmt.Sprintf("archive exceeds safety limits: %v", err)}
	}

	if err := fsutil.WriteFileAtomic(destPath, buf.Bytes(), 0644); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}

	return types.SaveResult{Success: true, FilePath: destPath, Warnings: warnings}
}
