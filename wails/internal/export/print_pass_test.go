package export

// The print pass: three things a printed book gets wrong if nobody has ever
// looked at the object rather than at the file.
//
// All three are checked against the PDF that actually comes out. fpdf writes
// one compressed content stream per page, in page order, so pageTexts below
// inflates them and reads back what is on each page — which is the only way to
// say "page one is the title page" without taking the renderer's word for it.
//
// Every fixture here is invented.

import (
	"bytes"
	"compress/zlib"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"draftline/internal/types"
)

var pdfStreamPattern = regexp.MustCompile(`(?s)stream\r?\n(.*?)endstream`)
var pdfShownTextPattern = regexp.MustCompile(`\((?:[^()\\]|\\.)*\)\s*Tj`)

// pageTexts returns what is drawn on each page of a PDF, in page order.
//
// fpdf draws text as UTF-16BE inside a PDF string, so every ASCII character
// arrives with a leading zero byte; those are stripped. A page content stream
// is recognised by fpdf's own prelude, which keeps font programs and other
// compressed objects out of the list.
func pageTexts(t *testing.T, data []byte) []string {
	t.Helper()
	var pages []string
	for _, match := range pdfStreamPattern.FindAllSubmatch(data, -1) {
		zr, err := zlib.NewReader(bytes.NewReader(match[1]))
		if err != nil {
			continue
		}
		body, err := io.ReadAll(zr)
		_ = zr.Close()
		if err != nil || !bytes.HasPrefix(body, []byte("0 J")) {
			continue
		}
		var text strings.Builder
		for _, run := range pdfShownTextPattern.FindAll(body, -1) {
			literal := run[1:bytes.LastIndexByte(run, ')')]
			text.Write(bytes.ReplaceAll(literal, []byte{0}, nil))
			text.WriteByte('\n')
		}
		pages = append(pages, strings.TrimSpace(text.String()))
	}
	return pages
}

func printPassChapter(n int) string {
	return "Chapter " + strconv.Itoa(n)
}

func printPassBook(chapters int) types.BookData {
	b := types.BookData{
		Metadata:  types.Metadata{Title: "Zephyr Quarter", Author: "Ines Okonkwo"},
		Copyright: "<p>Printed on invented paper.</p>",
	}
	for i := 1; i <= chapters; i++ {
		b.Body = append(b.Body, types.ChapterItem{
			Title: printPassChapter(i), Type: "chapter",
			Content: "<p>" + strings.Repeat("The quarter kept its own hours and its own weather. ", 12) + "</p>",
		})
	}
	return b
}

func basePrintOptions() types.PrintPDFOptions {
	return types.PrintPDFOptions{
		PDFOptions: types.PDFOptions{
			ExportOptions: types.ExportOptions{IncludeCopyright: true},
			FontSize:      10,
		},
		TrimSize:           "6x9",
		PageNumberPosition: "bottom-center",
		SkipKDPChecks:      true,
	}
}

// A title page on a verso is the mark of a book nobody typeset. Trade
// convention runs half title, blank, title page — so the title page is always
// on a recto, which in a PDF is an odd-numbered page.
func TestThePrintTitlePageTakesARecto(t *testing.T) {
	options := basePrintOptions()
	options.GenerateHalfTitle = true
	pages := pageTexts(t, generatePrintPDF(printPassBook(2), options))

	if len(pages) < 4 {
		t.Fatalf("the print PDF has only %d pages", len(pages))
	}
	if !strings.Contains(flatten(pages[0]), "Zephyr Quarter") {
		t.Fatalf("page one is not the half title: %q", pages[0])
	}
	if strings.TrimSpace(pages[1]) != "" {
		t.Fatalf("page two should be the half title's blank verso, and carries %q", pages[1])
	}
	if !strings.Contains(flatten(pages[2]), "Zephyr Quarter") || !strings.Contains(flatten(pages[2]), "Ines Okonkwo") {
		t.Fatalf("page three is not the title page: %q", pages[2])
	}
	if !strings.Contains(flatten(pages[3]), "invented paper") {
		t.Fatalf("the copyright page does not follow the title page: %q", pages[3])
	}
}

// With no half title the title page is page one, and stays page one: the blank
// exists to reach a recto, never for its own sake.
func TestPageOneOfAPrintPDFIsTheTitlePage(t *testing.T) {
	pages := pageTexts(t, generatePrintPDF(printPassBook(2), basePrintOptions()))
	if len(pages) < 2 {
		t.Fatalf("the print PDF has %d pages", len(pages))
	}
	if !strings.Contains(flatten(pages[0]), "Zephyr Quarter") || !strings.Contains(flatten(pages[0]), "Ines Okonkwo") {
		t.Fatalf("page one is not the title page: %q", pages[0])
	}
	if strings.TrimSpace(pages[1]) == "" {
		t.Fatal("a blank page was inserted where the title page was already on a recto")
	}
}

// A reading PDF is read on a screen, one page at a time. There is no back of a
// sheet for a title page to land on, so no blank is inserted there.
func TestAReadingPDFGetsNoBlankBeforeItsTitlePage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reading.pdf")
	result := PDF(path, printPassBook(2), types.PDFOptions{
		ExportOptions: types.ExportOptions{IncludeCopyright: true}, FontSize: 12,
	}, nil)
	if !result.Success {
		t.Fatal(result.Error)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pages := pageTexts(t, data)
	if len(pages) < 2 || strings.TrimSpace(pages[1]) == "" {
		t.Fatalf("a reading PDF gained a blank page: %q", pages)
	}
}

// The reading PDF had no page numbers at all: the position was never set and
// an unset position means print nothing. A reviewer cannot say where they are
// in a fixed-layout copy without them.
func TestAReadingPDFHasFolios(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reading.pdf")
	result := PDF(path, printPassBook(3), types.PDFOptions{
		ExportOptions: types.ExportOptions{IncludeCopyright: true}, FontSize: 12,
	}, nil)
	if !result.Success {
		t.Fatal(result.Error)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pages := pageTexts(t, data)
	if len(pages) < 4 {
		t.Fatalf("the reading PDF has only %d pages", len(pages))
	}
	numbered := 0
	for i, page := range pages {
		for _, line := range strings.Split(page, "\n") {
			if strings.TrimSpace(line) == strconv.Itoa(i+1) {
				numbered++
				break
			}
		}
	}
	if numbered < len(pages)-1 {
		t.Fatalf("only %d of %d pages carry their own number:\n%q", numbered, len(pages), pages)
	}
	// The title page is furniture-free by design; a folio on it would be a
	// page number on a page that has no number.
	for _, line := range strings.Split(pages[0], "\n") {
		if strings.TrimSpace(line) == "1" {
			t.Fatal("the title page carries a folio")
		}
	}
}

// A contents that outgrows the pages set aside for it used to stop writing,
// silently, and the missing chapters were invisible in a file on its way to a
// printer. It is an error now, and the error says how many were lost.
//
// The reservation and the fill share their arithmetic, so they will not
// disagree on their own; the shape of the failure is reached directly.
func TestAContentsThatWillNotFitIsAnErrorRatherThanAShortList(t *testing.T) {
	doc, err := BuildDocument(printPassBook(40), types.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	spec := printPDFSpec(basePrintOptions())
	spec.GenerateTOC = true
	r := newPublicationPDFRenderer(doc, spec)

	r.addPage(pageTOC, "Contents")
	r.tocPages = []int{r.pdf.PageNo()}
	for i := 1; i <= 40; i++ {
		r.toc = append(r.toc, pdfTOCEntry{Title: printPassChapter(i), Page: i + 4})
	}

	err = r.fillTOC()
	if err == nil {
		t.Fatal("forty contents entries fitted on one reserved page, which means they were dropped")
	}
	for _, want := range []string{"does not fit", "40 entries", "1 page "} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the error does not say %q: %s", want, err)
		}
	}
}

// The ordinary case, and the one the guard exists to protect: a forty-chapter
// contents comes out whole. Each title appears twice — once in the contents,
// once at the head of its chapter.
func TestAFortyChapterContentsListsEveryChapter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "long.pdf")
	options := basePrintOptions()
	options.GenerateTOC = true
	result := PrintPDF(path, printPassBook(40), options)
	if !result.Success {
		t.Fatalf("a forty-chapter book would not export: %s", result.Error)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(pageTexts(t, data), "\n")
	for i := 1; i <= 40; i++ {
		title := printPassChapter(i)
		// "Chapter 1" is a prefix of "Chapter 10"; count whole lines.
		seen := 0
		for _, line := range strings.Split(joined, "\n") {
			if strings.TrimSpace(line) == title {
				seen++
			}
		}
		if seen < 2 {
			t.Fatalf("%q appears on %d lines: it is missing from the contents or from the book", title, seen)
		}
	}
}

// flatten joins a page's text runs back into one string. fpdf emits a run per
// drawn fragment, so a wrapped sentence arrives as several of them; the runs
// carry their own spaces, so removing the separators restores the sentence.
func flatten(page string) string {
	return strings.ReplaceAll(page, "\n", "")
}
