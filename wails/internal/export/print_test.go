package export

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"draftline/internal/types"
)

// TestPrintPDFXrefOffsets verifies that generatePrintPDF emits a structurally
// valid cross-reference table: startxref points at the "xref" keyword and every
// non-free xref entry offset points exactly at that object's "N 0 obj" marker.
func TestPrintPDFXrefOffsets(t *testing.T) {
	book := types.BookData{
		Metadata: types.Metadata{
			Title:  "Test Book",
			Author: "Test Author",
		},
		Copyright: "<p>Copyright 2026 Test Author</p>",
		Body: []types.ChapterItem{
			{Title: "Chapter One", Type: "chapter", Content: "<p>The quick brown fox jumps over the lazy dog. " +
				strings.Repeat("More words to force wrapping and multiple lines of content. ", 40) + "</p>"},
			{Title: "Chapter Two", Type: "chapter", Content: "<p>A second chapter with additional text to span pages. " +
				strings.Repeat("Filler sentence for pagination purposes here. ", 60) + "</p>"},
		},
	}
	opts := types.PrintPDFOptions{
		PDFOptions: types.PDFOptions{
			ExportOptions: types.ExportOptions{IncludeCopyright: true},
			FontSize:      12,
		},
		TrimSize:           "6x9",
		PageNumberPosition: "bottom-center",
		RunningHeaders:     true,
		GenerateHalfTitle:  true,
		SkipKDPChecks:      true,
	}

	pdf := generatePrintPDF(book, opts)

	// Locate startxref value.
	idx := bytes.LastIndex(pdf, []byte("startxref"))
	if idx < 0 {
		t.Fatal("no startxref keyword in PDF")
	}
	rest := string(pdf[idx+len("startxref"):])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		t.Fatal("no value after startxref")
	}
	xrefOffset, err := strconv.Atoi(fields[0])
	if err != nil {
		t.Fatalf("startxref value not an integer: %q", fields[0])
	}

	// startxref must point at the "xref" keyword.
	if xrefOffset < 0 || xrefOffset+4 > len(pdf) {
		t.Fatalf("startxref offset %d out of bounds (len %d)", xrefOffset, len(pdf))
	}
	if got := string(pdf[xrefOffset : xrefOffset+4]); got != "xref" {
		t.Fatalf("startxref offset %d points at %q, want \"xref\"", xrefOffset, got)
	}

	// Parse the xref subsection header "0 N".
	xrefBody := pdf[xrefOffset+len("xref\n"):]
	lines := strings.SplitN(string(xrefBody), "\n", 3)
	if len(lines) < 2 {
		t.Fatal("malformed xref section")
	}
	header := strings.Fields(lines[0])
	if len(header) != 2 || header[0] != "0" {
		t.Fatalf("unexpected xref subsection header: %q", lines[0])
	}
	count, err := strconv.Atoi(header[1])
	if err != nil {
		t.Fatalf("xref count not an integer: %q", header[1])
	}

	// Collect the offset entries. First entry is the free object (65535 f).
	entryRe := regexp.MustCompile(`(?m)^(\d{10}) (\d{5}) ([fn]) $`)
	entries := entryRe.FindAllStringSubmatch(string(xrefBody), -1)
	if len(entries) != count {
		t.Fatalf("found %d xref entries, header declares %d", len(entries), count)
	}
	if entries[0][3] != "f" {
		t.Fatalf("first xref entry should be free, got type %q", entries[0][3])
	}

	// Each in-use entry's offset must point at "<id> 0 obj".
	for objID := 1; objID < count; objID++ {
		entry := entries[objID]
		if entry[3] != "n" {
			t.Errorf("object %d xref entry not in-use: %q", objID, entry[0])
			continue
		}
		off, err := strconv.Atoi(entry[1])
		if err != nil {
			t.Errorf("object %d offset not integer: %q", objID, entry[1])
			continue
		}
		marker := fmt.Sprintf("%d 0 obj", objID)
		if off < 0 || off+len(marker) > len(pdf) {
			t.Errorf("object %d offset %d out of bounds", objID, off)
			continue
		}
		if got := string(pdf[off : off+len(marker)]); got != marker {
			t.Errorf("object %d offset %d points at %q, want %q", objID, off, got, marker)
		}
	}
}

// A running head and a top-outside folio share a line. The head is indented
// inside the folio so the page number is not printed over the author's name.
func TestTopOutsideFoliosRenderBesideTheRunningHead(t *testing.T) {
	book := types.BookData{
		Metadata: types.Metadata{Title: "Wide Water", Author: "A. Marsh"},
		Body: []types.ChapterItem{
			{Title: "Chapter One", Type: "chapter", Content: "<p>" +
				strings.Repeat("Words enough to run past the foot of the page and onto the next. ", 60) + "</p>"},
		},
	}
	options := types.PrintPDFOptions{
		PDFOptions: types.PDFOptions{
			ExportOptions: types.ExportOptions{IncludeCopyright: true},
			FontSize:      11,
		},
		TrimSize:       "6x9",
		RunningHeaders: true,
		HeaderContent:  "author-title",
		SkipKDPChecks:  true,
	}

	// Both positions have to render, because they are a toggle and neither may
	// depend on the other having been chosen.
	for _, position := range []string{"top-outside", "bottom-center", "bottom-outside"} {
		options.PageNumberPosition = position
		data, err := PrintPDFBytes(book, options)
		if err != nil {
			t.Fatalf("rendering with %s folios: %v", position, err)
		}
		if len(data) == 0 {
			t.Fatalf("%s folios produced no PDF", position)
		}
	}
}

// The scene-break mark is a preference a printed page honours, not an ebook
// setting borrowed for one.
func TestPrintSceneBreakStylesAllRender(t *testing.T) {
	book := types.BookData{
		Metadata: types.Metadata{Title: "Wide Water", Author: "A. Marsh"},
		Body: []types.ChapterItem{
			{Title: "Chapter One", Type: "chapter",
				Content: "<p>Before the break.</p><hr /><p>After the break.</p>"},
		},
	}
	for _, style := range []string{"asterism", "rule", "space", ""} {
		options := types.PrintPDFOptions{
			PDFOptions:      types.PDFOptions{FontSize: 11},
			TrimSize:        "6x9",
			SceneBreakStyle: style,
			SkipKDPChecks:   true,
		}
		if _, err := PrintPDFBytes(book, options); err != nil {
			t.Fatalf("rendering with the %q scene break: %v", style, err)
		}
	}
}

// A reading copy's folios and watermark are choices, not decoration. Both used
// to be switches on a screen that reached nothing.
func TestReadingCopyFoliosAndWatermark(t *testing.T) {
	book := types.BookData{
		Metadata: types.Metadata{Title: "Wide Water", Author: "A. Marsh"},
		Body: []types.ChapterItem{{Title: "Chapter One", Type: "chapter", Content: "<p>" +
			strings.Repeat("Words enough to run onto a second page. ", 60) + "</p>"}},
	}
	base := types.PDFOptions{PageSize: "letter", FontFamily: "merriweather", FontSize: 12, LineHeight: 1.5}

	plain, err := PDFBytes(book, base, nil)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	if readingFolioPosition(base) != "bottom-center" {
		t.Error("a reading copy does not carry folios by default")
	}

	hidden := base
	hidden.HideFolios = true
	if readingFolioPosition(hidden) != "" {
		t.Error("turning folios off left them on")
	}
	noFolios, err := PDFBytes(book, hidden, nil)
	if err != nil {
		t.Fatalf("rendering without folios: %v", err)
	}
	if string(noFolios) == string(plain) {
		t.Error("turning folios off changed nothing in the file")
	}

	marked := base
	marked.DraftWatermark = true
	stamped, err := PDFBytes(book, marked, nil)
	if err != nil {
		t.Fatalf("rendering with a watermark: %v", err)
	}
	if string(stamped) == string(plain) {
		t.Error("the draft watermark changed nothing in the file")
	}
}
