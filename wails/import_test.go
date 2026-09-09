package main

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"draftline/internal/export"
	"draftline/internal/types"
)

// writeEPUB creates an EPUB at path from the given zip entries.
func writeEPUB(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
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

// epubManifestSpec describes one manifest+spine item for buildEPUBOPF.
type epubManifestSpec struct {
	id, href, mediaType, properties string
}

// buildEPUBOPF renders a minimal OPF for the given items, all of which are
// also placed on the spine in order.
func buildEPUBOPF(items []epubManifestSpec) string {
	manifest := ""
	spine := ""
	for _, it := range items {
		props := ""
		if it.properties != "" {
			props = ` properties="` + it.properties + `"`
		}
		manifest += `    <item id="` + it.id + `" href="` + it.href + `" media-type="` + it.mediaType + `"` + props + "/>\n"
		spine += `<itemref idref="` + it.id + `"/>`
	}
	return `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Fixture Book</dc:title>
    <dc:creator>Test Author</dc:creator>
  </metadata>
  <manifest>
` + manifest + `  </manifest>
  <spine>` + spine + `</spine>
</package>`
}

const epubContainerXML = `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`

// writeMinimalEPUB creates a tiny valid EPUB at path.
func writeMinimalEPUB(t *testing.T, path string) {
	t.Helper()
	writeEPUB(t, path, map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": epubContainerXML,
		"OEBPS/content.opf": buildEPUBOPF([]epubManifestSpec{
			{id: "ch1", href: "ch1.xhtml", mediaType: "application/xhtml+xml"},
		}),
		"OEBPS/ch1.xhtml": `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Chapter One</title></head>
<body><h1>Chapter One</h1><p>It was a dark and stormy fixture.</p></body></html>`,
	})
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

// A panic anywhere inside an importer must surface as a failed result, not
// kill the process — main.go binds App with no recover of its own.
func TestImportRecoversFromPanic(t *testing.T) {
	res := safeImport("EPUB", func() types.ImportResult { panic("boom") })
	if res.Success {
		t.Fatal("panicking import reported success")
	}
	if !strings.Contains(res.Error, "boom") {
		t.Fatalf("panic message lost: %q", res.Error)
	}
}

// EPUB 3 nav documents and SVG cover pages are machine/graphic resources, not
// chapters; they used to import as garbage chapters.
func TestImportSkipsNavAndSVGSpineItems(t *testing.T) {
	dir := t.TempDir()
	epub := filepath.Join(dir, "fixture.epub")
	writeEPUB(t, epub, map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": epubContainerXML,
		"OEBPS/content.opf": buildEPUBOPF([]epubManifestSpec{
			{id: "cover", href: "cover.svg", mediaType: "image/svg+xml"},
			{id: "nav", href: "nav.xhtml", mediaType: "application/xhtml+xml", properties: "nav"},
			{id: "ch1", href: "ch1.xhtml", mediaType: "application/xhtml+xml"},
		}),
		"OEBPS/cover.svg": `<svg xmlns="http://www.w3.org/2000/svg"><text>Cover Art</text></svg>`,
		"OEBPS/nav.xhtml": `<html xmlns="http://www.w3.org/1999/xhtml"><body><nav epub:type="toc"><ol><li><a href="ch1.xhtml">One</a></li></ol></nav></body></html>`,
		"OEBPS/ch1.xhtml": `<html xmlns="http://www.w3.org/1999/xhtml"><body><h1>One</h1><p>Real prose.</p></body></html>`,
	})

	a := &App{}
	res := a.ImportEPUB(epub)
	if !res.Success {
		t.Fatalf("import failed: %s", res.Error)
	}
	if len(res.Book.Body) != 1 {
		t.Fatalf("expected only the real chapter, got %d: %+v", len(res.Book.Body), res.Book.Body)
	}
	if res.Book.Body[0].Title != "One" {
		t.Fatalf("wrong chapter imported: %+v", res.Book.Body[0])
	}
}

// Oversized spine documents are skipped with a warning instead of being
// pushed through the sanitizer and the JSON bridge.
func TestImportSkipsOversizedSpineDoc(t *testing.T) {
	dir := t.TempDir()
	epub := filepath.Join(dir, "fixture.epub")
	// Number sequences keep the deflate ratio well under ziputil's 200:1
	// zip-bomb guard while still exceeding the per-document import cap.
	var filler strings.Builder
	for i := 0; filler.Len() <= maxSpineDocBytes; i++ {
		fmt.Fprintf(&filler, "%d ", i*7919)
	}
	huge := "<html><body><h1>Huge</h1><p>" + filler.String() + "</p></body></html>"
	writeEPUB(t, epub, map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": epubContainerXML,
		"OEBPS/content.opf": buildEPUBOPF([]epubManifestSpec{
			{id: "huge", href: "huge.xhtml", mediaType: "application/xhtml+xml"},
			{id: "ch1", href: "ch1.xhtml", mediaType: "application/xhtml+xml"},
		}),
		"OEBPS/huge.xhtml": huge,
		"OEBPS/ch1.xhtml":  `<html><body><h1>One</h1><p>Real prose.</p></body></html>`,
	})

	a := &App{}
	res := a.ImportEPUB(epub)
	if !res.Success {
		t.Fatalf("import failed: %s", res.Error)
	}
	if len(res.Book.Body) != 1 {
		t.Fatalf("oversized doc was not skipped, got %d chapters", len(res.Book.Body))
	}
	if len(res.Warnings) == 0 || !strings.Contains(res.Warnings[0], "huge.xhtml") {
		t.Fatalf("expected a warning naming the skipped document, got %v", res.Warnings)
	}
}

// Conservative chaptering: a spine document is one chapter unless it clearly
// contains several (multiple h1s, or multiple h2s with no h1). The chapter
// heading becomes the title and leaves the content — the app renders titles
// itself and the exporter re-adds an <h1>.
func TestAssembleChaptersConservativeSplit(t *testing.T) {
	doc := func(body string) []importedBlock {
		t.Helper()
		_, blocks, _, err := parseSpineDoc([]byte("<body>" + body + "</body>"))
		if err != nil {
			t.Fatal(err)
		}
		return blocks
	}

	t.Run("h1 with scene headings stays one chapter", func(t *testing.T) {
		parts, _ := assembleChapters("Doc", doc(`<h1>The Crossing</h1><p>Mara arrived.</p><h3>Later</h3><p>Night fell.</p>`))
		if len(parts) != 1 {
			t.Fatalf("expected 1 chapter, got %d: %+v", len(parts), parts)
		}
		if parts[0].Title != "The Crossing" {
			t.Fatalf("wrong title: %q", parts[0].Title)
		}
		if strings.Contains(parts[0].Content, "The Crossing") {
			t.Fatalf("title heading left in content: %s", parts[0].Content)
		}
		if !strings.Contains(parts[0].Content, "<h3>Later</h3>") {
			t.Fatalf("scene heading lost: %s", parts[0].Content)
		}
	})

	t.Run("one h1 with h2 part headers stays one chapter", func(t *testing.T) {
		parts, _ := assembleChapters("Doc", doc(`<h1>One</h1><p>Start.</p><h2>Part A</h2><p>Middle.</p><h2>Part B</h2><p>End.</p>`))
		if len(parts) != 1 {
			t.Fatalf("expected 1 chapter, got %d", len(parts))
		}
		if !strings.Contains(parts[0].Content, "<h2>Part A</h2>") || !strings.Contains(parts[0].Content, "<h2>Part B</h2>") {
			t.Fatalf("h2 part headers lost: %s", parts[0].Content)
		}
	})

	t.Run("multiple h1s split", func(t *testing.T) {
		parts, _ := assembleChapters("Doc", doc(`<h1>1</h1><p>Mara arrived.</p><h1>2</h1><p>Hanlon answered.</p><h1>3</h1><p>Ruiz waited.</p>`))
		if len(parts) != 3 {
			t.Fatalf("expected 3 chapters, got %d: %+v", len(parts), parts)
		}
		if parts[0].Title != "1" || parts[1].Title != "2" || parts[2].Title != "3" {
			t.Fatalf("heading titles not preserved: %+v", parts)
		}
		if strings.Contains(parts[1].Content, "<h1>") {
			t.Fatalf("split heading left in content: %s", parts[1].Content)
		}
	})

	t.Run("multiple h2s split when no h1 exists", func(t *testing.T) {
		parts, _ := assembleChapters("Doc", doc(`<h2>1</h2><p>Mara arrived.</p><h2>2</h2><p>Hanlon answered.</p>`))
		if len(parts) != 2 {
			t.Fatalf("expected 2 chapters, got %d", len(parts))
		}
	})

	t.Run("prefix content merges into first chapter", func(t *testing.T) {
		parts, _ := assembleChapters("Doc", doc(`<p>An epigraph.</p><h1>One</h1><p>Prose.</p><h1>Two</h1><p>More.</p>`))
		if len(parts) != 2 {
			t.Fatalf("expected 2 chapters, got %d: %+v", len(parts), parts)
		}
		if parts[0].Title != "One" || !strings.Contains(parts[0].Content, "An epigraph.") {
			t.Fatalf("prefix did not merge into first chapter: %+v", parts[0])
		}
	})

	t.Run("leading h3 is not a title", func(t *testing.T) {
		parts, _ := assembleChapters("Doc Title", doc(`<h3>Scene</h3><p>Prose.</p>`))
		if parts[0].Title != "Doc Title" {
			t.Fatalf("h3 stole the title: %q", parts[0].Title)
		}
		if !strings.Contains(parts[0].Content, "<h3>Scene</h3>") {
			t.Fatalf("leading h3 lost: %s", parts[0].Content)
		}
	})
}

// Export → import must not stack titles: the exporter writes <h1>{title}</h1>
// and the importer must take it back out of the content.
func TestImportExportRoundTripNoDoubleTitle(t *testing.T) {
	dir := t.TempDir()
	epub := filepath.Join(dir, "roundtrip.epub")
	book := types.BookData{
		Version:  "2.0",
		Metadata: types.Metadata{Title: "Round Trip", Author: "Tester"},
		Body: []types.ChapterItem{
			{Title: "The Crossing", Type: "Chapter", Content: "<p>Mara arrived at dusk.</p>"},
			{Title: "The Ledger", Type: "Chapter", Content: "<p>Hanlon counted twice.</p>"},
		},
	}
	if res := export.EPUB(epub, book, types.EPUBOptions{}); !res.Success {
		t.Fatalf("export failed: %s", res.Error)
	}

	a := &App{}
	res := a.ImportEPUB(epub)
	if !res.Success {
		t.Fatalf("import failed: %s", res.Error)
	}
	if len(res.Book.Body) != 2 {
		t.Fatalf("expected 2 chapters, got %d: %+v", len(res.Book.Body), res.Book.Body)
	}
	for i, want := range []string{"The Crossing", "The Ledger"} {
		ch := res.Book.Body[i]
		if ch.Title != want {
			t.Fatalf("chapter %d title = %q, want %q", i, ch.Title, want)
		}
		if strings.Contains(ch.Content, "<h1>") {
			t.Fatalf("chapter %d content still carries its title heading: %s", i, ch.Content)
		}
	}
}

func TestRouteImportedSection(t *testing.T) {
	cases := []struct {
		title     string
		plainText string
		route     sectionRoute
		typeLabel string
	}{
		{"Cover", "", routeSkip, ""},
		{"Table of Contents", "", routeSkip, ""},
		{"Copyright", "", routeCopyright, ""},
		{"Untitled", "© 2026 Example House. All rights reserved.", routeCopyright, ""},
		{"Title Page", "", routeFront, "Title Page"},
		{"Dedication", "", routeFront, "Dedication"},
		{"Epigraph", "", routeFront, "Epigraph"},
		{"PROLOGUE", "", routeFront, "Prologue"},
		{"Also by Jane Doe", "", routeFront, "Also By"},
		{"ACKNOWLEDGMENTS", "", routeBack, "Acknowledgments"},
		{"Untitled", "Acknowledgements go to everyone.", routeBack, "Acknowledgments"},
		{"About the Author", "", routeBack, "About the Author"},
		{"Epilogue", "", routeBack, "Epilogue"},
		{"Glossary", "", routeBack, "Glossary"},
		{"The Crossing", "Mara arrived at dusk.", routeBody, "Chapter"},
		{"Chapter 7", "", routeBody, "Chapter"},
	}
	for _, tc := range cases {
		route, typeLabel := routeImportedSection(tc.title, tc.plainText)
		if route != tc.route || typeLabel != tc.typeLabel {
			t.Fatalf("routeImportedSection(%q, %q) = (%v, %q), want (%v, %q)",
				tc.title, tc.plainText, route, typeLabel, tc.route, tc.typeLabel)
		}
	}
}

// EPUBs that repeat the book title in every spine item's <title> (and carry
// no visible headings) must fall back to "Chapter N", not name every chapter
// after the book.
func TestImportIgnoresRepeatedBookTitle(t *testing.T) {
	dir := t.TempDir()
	epub := filepath.Join(dir, "fixture.epub")
	writeEPUB(t, epub, map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": epubContainerXML,
		"OEBPS/content.opf": buildEPUBOPF([]epubManifestSpec{
			{id: "ch1", href: "ch1.xhtml", mediaType: "application/xhtml+xml"},
			{id: "ch2", href: "ch2.xhtml", mediaType: "application/xhtml+xml"},
		}),
		"OEBPS/ch1.xhtml": `<html><head><title>Fixture Book</title></head><body><p>First chapter prose.</p></body></html>`,
		"OEBPS/ch2.xhtml": `<html><head><title>Fixture Book</title></head><body><p>Second chapter prose.</p></body></html>`,
	})

	a := &App{}
	res := a.ImportEPUB(epub)
	if !res.Success {
		t.Fatalf("import failed: %s", res.Error)
	}
	if len(res.Book.Body) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(res.Book.Body))
	}
	if res.Book.Body[0].Title != "Chapter 1" || res.Book.Body[1].Title != "Chapter 2" {
		t.Fatalf("repeated book title leaked into chapter names: %q, %q",
			res.Book.Body[0].Title, res.Book.Body[1].Title)
	}
}

// End-to-end routing: sections land in the right BookData destination.
func TestImportRoutesSections(t *testing.T) {
	dir := t.TempDir()
	epub := filepath.Join(dir, "fixture.epub")
	writeEPUB(t, epub, map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": epubContainerXML,
		"OEBPS/content.opf": buildEPUBOPF([]epubManifestSpec{
			{id: "copy", href: "copyright.xhtml", mediaType: "application/xhtml+xml"},
			{id: "ded", href: "dedication.xhtml", mediaType: "application/xhtml+xml"},
			{id: "ch1", href: "ch1.xhtml", mediaType: "application/xhtml+xml"},
			{id: "ch2", href: "ch2.xhtml", mediaType: "application/xhtml+xml"},
			{id: "ack", href: "ack.xhtml", mediaType: "application/xhtml+xml"},
		}),
		"OEBPS/copyright.xhtml":  `<html><body><h2>Copyright</h2><p>© 2026 Example House. All rights reserved.</p></body></html>`,
		"OEBPS/dedication.xhtml": `<html><body><h2>Dedication</h2><p>For the fixtures.</p></body></html>`,
		"OEBPS/ch1.xhtml":        `<html><body><h1>The Crossing</h1><p>Mara arrived at dusk.</p></body></html>`,
		"OEBPS/ch2.xhtml":        `<html><body><h1>The Ledger</h1><p>Hanlon counted twice.</p></body></html>`,
		"OEBPS/ack.xhtml":        `<html><body><h2>Acknowledgments</h2><p>Thanks, everyone.</p></body></html>`,
	})

	a := &App{}
	res := a.ImportEPUB(epub)
	if !res.Success {
		t.Fatalf("import failed: %s", res.Error)
	}
	if !strings.Contains(res.Book.Copyright, "All rights reserved") {
		t.Fatalf("copyright page did not fill the Copyright field: %q", res.Book.Copyright)
	}
	if len(res.Book.FrontMatter) != 1 || res.Book.FrontMatter[0].Type != "Dedication" {
		t.Fatalf("dedication not routed to front matter: %+v", res.Book.FrontMatter)
	}
	if len(res.Book.Body) != 2 || res.Book.Body[0].Title != "The Crossing" || res.Book.Body[1].Title != "The Ledger" {
		t.Fatalf("body chapters wrong: %+v", res.Book.Body)
	}
	for _, ch := range res.Book.Body {
		if ch.Type != "Chapter" {
			t.Fatalf("body chapter has wrong type: %+v", ch)
		}
	}
	if len(res.Book.BackMatter) != 1 || res.Book.BackMatter[0].Type != "Acknowledgments" {
		t.Fatalf("acknowledgments not routed to back matter: %+v", res.Book.BackMatter)
	}
}

func TestParseSpineDocPrefersVisibleSectionHeading(t *testing.T) {
	title, _, _, err := parseSpineDoc([]byte(`<html><head><title>Book Title</title></head><body><h2>ACKNOWLEDGMENTS</h2><p>Thanks.</p></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	if title != "ACKNOWLEDGMENTS" {
		t.Fatalf("visible heading should win over repeated EPUB title, got %q", title)
	}
	if route, _ := routeImportedSection(title, "Thanks."); route != routeBack {
		t.Fatalf("acknowledgments heading not routed to back matter: %v", route)
	}
}
