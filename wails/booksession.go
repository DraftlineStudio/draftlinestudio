package main

// Opening, saving and closing the project.
//
// Every one of these is a Wails binding, so they are the backend half of what
// the writer means by "my book". writeBook is the funnel they all end at: it
// is where the OPEN project is read for preserved members, which is why a
// Save As carries the open book's chapter history to the new file rather than
// leaving it behind.

import (
	"strings"
	"time"

	"draftline/internal/book"
	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) writeBook(b types.BookData, path string) types.SaveResult {
	// The open project is what the save reads preserved members from — chapter
	// history above all. A Save As must carry the OPEN book's history to the new
	// file, and must not inherit the history of whatever project it is saving
	// over; reading them from the destination did both the wrong way round.
	source := a.getCurrentFile()
	// Saving to a NEW path (Save As, first save) claims that path's lock, so
	// two instances can't silently write over each other's book. Saving to
	// the already-current path keeps the lock it holds.
	// Cover art and frozen manuscripts ride beside the book, not on it. Only
	// what has changed since the last save is handed over; everything else
	// survives by passthrough, stored and never re-encoded.
	assets, settled := a.pendingAssets()
	if path != source {
		lock, err := a.claimBookLock(path)
		if err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		result := book.WriteArchive(source, path, b, AppVersion, nil, assets)
		if !result.Success {
			a.discardBookLockClaim(lock)
			return result
		}
		a.installBookLock(lock)
		a.claimDeviceLock(path, b.Metadata.BookID)
		a.setCurrentFile(path)
		settled()
		return result
	}
	result := book.WriteArchive(source, path, b, AppVersion, nil, assets)
	if result.Success {
		a.setCurrentFile(path)
		settled()
	}
	return result
}

// NewBook returns an empty types.BookData struct with defaults.
func (a *App) NewBook() types.BookData {
	now := time.Now().Format(time.RFC3339)
	a.leaveOpenProject()
	a.releaseBookLock()
	newBook := types.BookData{
		Version: "2.2",
		Metadata: types.Metadata{
			Title:    "Untitled",
			Created:  now,
			Modified: now,
			BookID:   types.NewBookID(),
		},
		Copyright:   "",
		FrontMatter: []types.ChapterItem{},
		Body: []types.ChapterItem{
			{Title: "Chapter 1", Type: "Chapter", Content: "<p></p>"},
		},
		BackMatter: []types.ChapterItem{},
		StoryBible: types.StoryBible{Characters: []types.Character{}},
	}
	book.EnsureBookChapterIDs(&newBook)
	return newBook
}

// OpenBookDialog shows the native file picker and opens the selected file.
func (a *App) OpenBookDialog() (types.BookData, error) {
	path, err := a.PickBookPath()
	if err != nil || path == "" {
		return types.BookData{}, nil
	}
	return a.openBook(path)
}

// PickBookPath shows the open dialog and returns only the chosen path, so
// the frontend can show loading feedback while the (separate) open call
// parses a large archive.
func (a *App) PickBookPath() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Open Draftline Project",
		DefaultDirectory: a.bookDialogDir(),
		Filters: []runtime.FileFilter{
			{DisplayName: "Draftline Files (*.draftline)", Pattern: "*.draftline"},
		},
	})
	if err != nil || path == "" {
		return "", nil
	}
	return path, nil
}

func (a *App) openBook(path string) (types.BookData, error) {
	// Lock before parsing: if another instance has this book, foreground it
	// and refuse the second copy (Word-style same-file semantics).
	lock, err := a.claimBookLock(path)
	if err != nil {
		return types.BookData{}, err
	}
	b, err := book.Open(path)
	if err != nil {
		a.discardBookLockClaim(lock)
		return types.BookData{}, err
	}
	a.installBookLock(lock)
	a.claimDeviceLock(path, b.Metadata.BookID)
	// A different book has different covers and a different publishing
	// history; both caches are per project.
	a.covers.reset()
	a.snapshots.reset()
	a.setCurrentFile(path)
	return b, nil
}

// SaveBook saves to the current file path, or invokes SaveBookAs if unsaved.
func (a *App) SaveBook(book types.BookData) types.SaveResult {
	current := a.getCurrentFile()
	if current == "" {
		return a.SaveBookAs(book)
	}
	return a.writeBook(book, current)
}

// SaveBookSnapshots atomically saves the current book and appends deduplicated
// chapter snapshots to the version history embedded in its archive.
func (a *App) SaveBookSnapshots(b types.BookData, snapshots []types.ChapterSnapshotRequest) types.SaveResult {
	current := a.getCurrentFile()
	if current == "" {
		return types.SaveResult{Success: false, Error: "save the project before creating version history"}
	}
	assets, settled := a.pendingAssets()
	result := book.WriteArchive(current, current, b, AppVersion, snapshots, assets)
	if result.Success {
		a.setCurrentFile(current)
		settled()
	}
	return result
}

// ListChapterHistory returns snapshot metadata for one stable chapter ID.
func (a *App) ListChapterHistory(chapterID string) ([]types.ChapterHistoryEntry, error) {
	return book.ListChapterHistory(a.getCurrentFile(), chapterID)
}

// GetChapterHistory returns one snapshot after resolving it through the
// archive index; callers cannot use the ID as an arbitrary archive path.
func (a *App) GetChapterHistory(snapshotID string) (types.ChapterHistorySnapshot, error) {
	return book.GetChapterHistorySnapshot(a.getCurrentFile(), snapshotID)
}

// SaveBookAs shows the native save dialog.
func (a *App) SaveBookAs(book types.BookData) types.SaveResult {
	defaultName := book.Metadata.Title
	if strings.TrimSpace(defaultName) == "" {
		defaultName = "Untitled"
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Save Draftline Project",
		DefaultFilename:  defaultName + ".draftline",
		DefaultDirectory: a.bookDialogDir(),
		Filters: []runtime.FileFilter{
			{DisplayName: "Draftline Files (*.draftline)", Pattern: "*.draftline"},
		},
	})
	if err != nil || path == "" {
		return types.SaveResult{Success: false, Error: "cancelled"}
	}
	if !strings.HasSuffix(strings.ToLower(path), ".draftline") {
		path += ".draftline"
	}
	// A copy is a different book.
	book.Metadata.BookID = types.NewBookID()
	return a.writeBook(book, path)
}

// GetCurrentFile returns the path of the currently open file.
func (a *App) GetCurrentFile() string {
	return a.getCurrentFile()
}
