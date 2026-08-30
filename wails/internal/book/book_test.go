package book

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"draftline/internal/types"
)

func testBook() types.BookData {
	return types.BookData{
		Version: "2.0",
		Metadata: types.Metadata{
			Title:  "Test Book",
			Author: "Tester",
		},
		Copyright: "<p>© test</p>",
		Body: []types.ChapterItem{
			{Title: "Chapter One", Type: "chapter", Content: "<p>Hello — “world”.</p>"},
			{Title: "Chapter Two", Type: "chapter", Content: "<p>More text.</p>"},
		},
		FrontMatter: []types.ChapterItem{{Title: "Dedication", Type: "dedication", Content: "<p>For x.</p>"}},
		BackMatter:  []types.ChapterItem{},
		StoryBible:  types.StoryBible{Characters: []types.Character{}},
	}
}

// redirect backups away from the real user config dir
func isolateConfigDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
}

func TestWriteOpenRoundTrip(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "book.draftline")

	res := Write(path, testBook(), "test-version")
	if !res.Success {
		t.Fatalf("Write failed: %s", res.Error)
	}

	// The output must be a valid zip containing manifest.json.
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("output is not a valid zip: %v", err)
	}
	if _, err := ReadZipEntry(r, "manifest.json"); err != nil {
		t.Fatalf("manifest.json missing: %v", err)
	}
	_ = r.Close()

	got, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got.Metadata.Title != "Test Book" || len(got.Body) != 2 {
		t.Fatalf("round-trip mismatch: title=%q body=%d", got.Metadata.Title, len(got.Body))
	}
	if got.Body[0].Content != "<p>Hello — “world”.</p>" {
		t.Fatalf("content mismatch: %q", got.Body[0].Content)
	}
}

func TestWriteReplacesExistingFileAtomically(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "book.draftline")

	if res := Write(path, testBook(), "v1"); !res.Success {
		t.Fatalf("first write: %s", res.Error)
	}
	b := testBook()
	b.Metadata.Title = "Second Save"
	if res := Write(path, b, "v1"); !res.Success {
		t.Fatalf("second write: %s", res.Error)
	}
	got, err := Open(path)
	if err != nil {
		t.Fatalf("Open after overwrite: %v", err)
	}
	if got.Metadata.Title != "Second Save" {
		t.Fatalf("expected updated title, got %q", got.Metadata.Title)
	}
	// No stray temp files next to the book.
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if e.Name() != filepath.Base(path) {
			t.Fatalf("unexpected file left beside book: %s", e.Name())
		}
	}
}

func TestOpenRejectsZipBomb(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "bomb.draftline")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	mw, _ := w.Create("manifest.json")
	_, _ = mw.Write([]byte(`{"version":"2.0"}`))
	bw, _ := w.Create("body/000.html")
	_, _ = bw.Write(make([]byte, 20<<20)) // 20MB zeros: extreme compression ratio
	_ = w.Close()
	_ = f.Close()

	if _, err := Open(path); err == nil {
		t.Fatal("expected zip bomb to be rejected")
	}
}
