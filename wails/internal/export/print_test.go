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
