package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// writeMinimalEPUB creates a tiny valid EPUB at path.
func writeMinimalEPUB(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	entries := map[string]string{
		"mimetype": "application/epub+zip",
		"META-INF/container.xml": `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Fixture Book</dc:title>
    <dc:creator>Test Author</dc:creator>
  </metadata>
  <manifest>
    <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="ch1"/></spine>
</package>`,
		"OEBPS/ch1.xhtml": `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Chapter One</title></head>
<body><h1>Chapter One</h1><p>It was a dark and stormy fixture.</p></body></html>`,
	}
	for name, content := range entries {
		e, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

// Regression test for the import overwrite bug: importing a book while a
// project was open left a.currentFile pointing at the previous project, so
// Ctrl+S wrote the imported book over it. A successful import must clear the
// session's current file so Save routes through Save As.
func TestImportEPUBClearsCurrentFile(t *testing.T) {
	dir := t.TempDir()
	epub := filepath.Join(dir, "fixture.epub")
	writeMinimalEPUB(t, epub)

	a := &App{currentFile: filepath.Join(dir, "previous-project.draftline")}
	res := a.ImportEPUB(epub)
	if !res.Success {
		t.Fatalf("import failed: %s", res.Error)
	}
	if len(res.Book.Body) == 0 {
		t.Fatal("imported book has no chapters")
	}
	if res.Book.FilePath != "" {
		t.Fatalf("imported book carries a file path: %q", res.Book.FilePath)
	}
	if a.currentFile != "" {
		t.Fatalf("currentFile not cleared after import: %q — Ctrl+S would overwrite the previous project", a.currentFile)
	}
}

// A failed import must leave the session untouched: the previous project is
// still the open book, so its save target must survive.
func TestFailedImportKeepsCurrentFile(t *testing.T) {
	prev := "C:/somewhere/previous-project.draftline"
	a := &App{currentFile: prev}
	res := a.ImportEPUB(filepath.Join(t.TempDir(), "does-not-exist.epub"))
	if res.Success {
		t.Fatal("import of missing file unexpectedly succeeded")
	}
	if a.currentFile != prev {
		t.Fatalf("failed import changed currentFile: %q", a.currentFile)
	}
}

func TestSplitImportedChaptersUsesInternalHeadings(t *testing.T) {
	body := `<div><h2>1</h2><p>Mara arrived.</p><h2>2</h2><p>Hanlon answered.</p><h2>3</h2><p>Ruiz waited.</p></div>`
	parts := splitImportedChapters("Novel", body)
	if len(parts) != 3 {
		t.Fatalf("expected 3 chapters, got %d: %+v", len(parts), parts)
	}
	if parts[0].Title != "1" || parts[1].Title != "2" || parts[2].Title != "3" {
		t.Fatalf("heading titles not preserved: %+v", parts)
	}
}

func TestClassifyImportedNonStorySection(t *testing.T) {
	for _, title := range []string{"Copyright", "ACKNOWLEDGMENTS", "Table of Contents", "Glossary"} {
		if got := classifyImportedSection(title, ""); got == "Chapter" {
			t.Fatalf("%q should not be classified as story prose", title)
		}
	}
}

func TestParseXHTMLPrefersVisibleSectionHeading(t *testing.T) {
	title, body := parseXHTMLContent(`<html><head><title>Book Title</title></head><body><h2>ACKNOWLEDGMENTS</h2><p>Thanks.</p></body></html>`)
	if title != "ACKNOWLEDGMENTS" {
		t.Fatalf("visible heading should win over repeated EPUB title, got %q", title)
	}
	if got := classifyImportedSection(title, body); got != "acknowledgments" {
		t.Fatalf("wrong semantic section type: %q", got)
	}
}
