package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"draftline/internal/book"
	"draftline/internal/types"
)

func historyOf(t *testing.T, path string) map[string][]byte {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("cannot open %s: %v", filepath.Base(path), err)
	}
	defer func() { _ = r.Close() }()
	entries := map[string][]byte{}
	for _, file := range r.File {
		if !strings.HasPrefix(file.Name, "history/") {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("cannot open %q: %v", file.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("cannot read %q: %v", file.Name, err)
		}
		entries[file.Name] = data
	}
	return entries
}

func sortedNames(entries map[string][]byte) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func bookWithChapter(content string) types.BookData {
	return types.BookData{
		Version:  "2.0",
		Metadata: types.Metadata{Title: "Invented test book", Author: "Invented author"},
		Body: []types.ChapterItem{
			{ID: "ch-one", Title: "One", Type: "chapter", Content: content},
		},
	}
}

// Save As used to read chapter history from the file it was about to create,
// which does not exist yet, so the copy arrived with none of it. The history
// belongs to the book that is open, and must travel with it.
func TestSaveAsCarriesChapterHistoryToTheNewFile(t *testing.T) {
	config := t.TempDir()
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CONFIG_HOME", config)

	dir := t.TempDir()
	source := filepath.Join(dir, "source.draftline")
	b := bookWithChapter("<p>The lamp-lighter worked the west quay first.</p>")
	if result := book.Write("", source, b, "test-version"); !result.Success {
		t.Fatal(result.Error)
	}

	app := &App{}
	if _, err := app.openBook(source); err != nil {
		t.Fatal(err)
	}
	defer app.releaseBookLock()

	if result := app.SaveBookSnapshots(b, []types.ChapterSnapshotRequest{{
		ChapterID:    "ch-one",
		Section:      "body",
		ChapterTitle: "One",
		Content:      b.Body[0].Content,
		Reason:       "Writing session",
	}}); !result.Success {
		t.Fatal(result.Error)
	}

	sourceHistory := historyOf(t, source)
	if len(sourceHistory) < 2 {
		t.Fatalf("the source project has no chapter history to carry: %v", sortedNames(sourceHistory))
	}
	sourceBefore, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(dir, "copy.draftline")
	if result := app.writeBook(b, dest); !result.Success {
		t.Fatal(result.Error)
	}

	destHistory := historyOf(t, dest)
	if len(destHistory) != len(sourceHistory) {
		t.Fatalf("Save As wrote %v, wanted %v", sortedNames(destHistory), sortedNames(sourceHistory))
	}
	for name, want := range sourceHistory {
		got, ok := destHistory[name]
		if !ok {
			t.Fatalf("Save As dropped %q", name)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("Save As changed %q: %d bytes, wanted %d", name, len(got), len(want))
		}
	}

	sourceAfter, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sourceBefore, sourceAfter) {
		t.Fatal("Save As modified the project it was saving from")
	}
}

// Saving over another project replaces it. Its chapter history is that other
// book's, and must not be adopted as the open book's own.
func TestSaveAsOverAnotherProjectDoesNotInheritItsHistory(t *testing.T) {
	config := t.TempDir()
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CONFIG_HOME", config)

	dir := t.TempDir()
	other := filepath.Join(dir, "other.draftline")
	otherBook := bookWithChapter("<p>Quarry dust settled on the ledger.</p>")
	if result := book.WriteWithSnapshots("", other, otherBook, "test-version", []types.ChapterSnapshotRequest{{
		ChapterID:    "ch-one",
		Section:      "body",
		ChapterTitle: "One",
		Content:      otherBook.Body[0].Content,
		Reason:       "Writing session",
	}}); !result.Success {
		t.Fatal(result.Error)
	}
	if len(historyOf(t, other)) < 2 {
		t.Fatal("the other project was written without chapter history")
	}

	source := filepath.Join(dir, "source.draftline")
	b := bookWithChapter("<p>The lamp-lighter worked the west quay first.</p>")
	if result := book.Write("", source, b, "test-version"); !result.Success {
		t.Fatal(result.Error)
	}

	app := &App{}
	if _, err := app.openBook(source); err != nil {
		t.Fatal(err)
	}
	defer app.releaseBookLock()

	if result := app.writeBook(b, other); !result.Success {
		t.Fatal(result.Error)
	}

	written := historyOf(t, other)
	if len(written) != 0 {
		t.Fatalf("Save As inherited the overwritten project's chapter history: %v", sortedNames(written))
	}
	if app.getCurrentFile() != other {
		t.Fatalf("Save As left the current file at %q", app.getCurrentFile())
	}
}

// A project file that has gone unreadable — a dropped network share, a backup
// tool holding it open, a half-written copy — must not trap the text in memory.
// Save As is the escape: it leaves the damaged file alone, writes the book
// somewhere else, moves the author onto the new file, and says what it could
// not carry across.
func TestSaveAsRescuesABookWhoseProjectFileWentUnreadable(t *testing.T) {
	config := t.TempDir()
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CONFIG_HOME", config)

	dir := t.TempDir()
	source := filepath.Join(dir, "source.draftline")
	b := bookWithChapter("<p>The lamp-lighter worked the west quay first.</p>")
	if result := book.Write("", source, b, "test-version"); !result.Success {
		t.Fatal(result.Error)
	}

	app := &App{}
	if _, err := app.openBook(source); err != nil {
		t.Fatal(err)
	}
	defer app.releaseBookLock()

	intact, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, intact[:len(intact)/2], 0o600); err != nil {
		t.Fatal(err)
	}

	b.Body[0].Content = "<p>He had written another page before the file went bad.</p>"
	// An ordinary save writes over the damaged file itself, which is where the
	// unread history still is, so it still refuses rather than destroying it.
	if result := app.writeBook(b, source); result.Success {
		t.Fatal("saving over the damaged file destroyed what could not be read")
	}

	dest := filepath.Join(dir, "rescue.draftline")
	result := app.writeBook(b, dest)
	if !result.Success {
		t.Fatalf("the book could not be saved anywhere: %s", result.Error)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("the rescue carried nothing across without saying so")
	}
	if app.getCurrentFile() != dest {
		t.Fatalf("the author was left on the damaged file: %q", app.getCurrentFile())
	}

	reopened := &App{}
	rescued, err := reopened.openBook(dest)
	if err != nil {
		t.Fatalf("the rescue file does not open: %v", err)
	}
	defer reopened.releaseBookLock()
	if len(rescued.Body) != 1 || !strings.Contains(rescued.Body[0].Content, "another page") {
		t.Fatal("the rescue file does not hold the text that was in memory")
	}

	// The rescue file is a healthy project: saving into it from now on is
	// ordinary and silent.
	if result := reopened.SaveBook(rescued); !result.Success || len(result.Warnings) != 0 {
		t.Fatalf("saving the rescued project reported %v: %s", result.Warnings, result.Error)
	}
}
