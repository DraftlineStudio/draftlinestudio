package main

import (
	"os"
	"path/filepath"
	"testing"

	"draftline/internal/book"
	"draftline/internal/types"
)

func TestFailedOpenKeepsCurrentBookLock(t *testing.T) {
	config := t.TempDir()
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CONFIG_HOME", config)

	dir := t.TempDir()
	current := filepath.Join(dir, "current.draftline")
	b := types.BookData{
		Version:  "2.0",
		Metadata: types.Metadata{Title: "Invented test book"},
		Body:     []types.ChapterItem{{Title: "One", Type: "chapter", Content: "<p>The signal lamp dimmed.</p>"}},
	}
	if result := book.Write(current, current, b, "test-version"); !result.Success {
		t.Fatal(result.Error)
	}

	app := &App{}
	if _, err := app.openBook(current); err != nil {
		t.Fatal(err)
	}
	defer app.releaseBookLock()
	original := app.bookLock.lock

	bad := filepath.Join(dir, "damaged.draftline")
	if err := os.WriteFile(bad, []byte("not an archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := app.openBook(bad); err == nil {
		t.Fatal("damaged project unexpectedly opened")
	}
	if app.bookLock.lock != original || app.getCurrentFile() != current {
		t.Fatal("failed open replaced the current book or its lock")
	}
}

func TestFailedSaveAsKeepsCurrentBookLock(t *testing.T) {
	config := t.TempDir()
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CONFIG_HOME", config)

	dir := t.TempDir()
	current := filepath.Join(dir, "current.draftline")
	b := types.BookData{Version: "2.0", Metadata: types.Metadata{Title: "Invented test book"}}
	if result := book.Write(current, current, b, "test-version"); !result.Success {
		t.Fatal(result.Error)
	}

	app := &App{}
	if _, err := app.openBook(current); err != nil {
		t.Fatal(err)
	}
	defer app.releaseBookLock()
	original := app.bookLock.lock

	bad := filepath.Join(dir, "missing", "cannot-save.draftline")
	if result := app.writeBook(b, bad); result.Success {
		t.Fatal("Save As unexpectedly succeeded in a missing directory")
	}
	if app.bookLock.lock != original || app.getCurrentFile() != current {
		t.Fatal("failed Save As replaced the current book or its lock")
	}
}
