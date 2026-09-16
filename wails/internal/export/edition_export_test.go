package export

// Exporting from a registered edition.
//
// Everything here is checked against the file that comes out, not against
// which function was called: the identifier a reading device reads, the
// manifest item a cover lives in, the order of the spine, the lines on the
// copyright page, and the page count of a PDF. The artwork is drawn from
// arithmetic so that no image file has to live in the repository.

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"draftline/internal/types"
)

// ── Fixtures ───────────────────────────────────────────────────────────────

// editionBook is a small invented book with a publishing record on it: a first
// edition of 2026 and a second edition of 2030 that supersedes it, each with
// an ebook and a paperback. The names, the imprint and the ISBNs are invented;
// the ISBNs carry correct check digits so that they normalise the way a real
// one would.
func editionBook() types.BookData {
	return types.BookData{
		Metadata: types.Metadata{
			Title:           "The Quiet Ledger",
			Subtitle:        "A Harrowgate Novel",
			Author:          "Ilse Marchetti",
			Publisher:       "Bellwether House",
			Imprint:         "Bellwether House",
			CopyrightHolder: "Ilse Marchetti",
			Language:        "en-GB",
		},
		Copyright: "<p>No part of this book may be reproduced without permission.</p>",
		Body: []types.ChapterItem{
			{ID: "b1", Title: "The Crossing", Type: "chapter", Content: "<p>The ferry left before anyone had counted the crates.</p>"},
			{ID: "b2", Title: "The Ledger", Type: "chapter", Content: "<p>Every column balanced, which was the trouble.</p>"},
		},
		Editions: &types.EditionIndex{
			Version: 1,
			Editions: []types.Edition{
				{
					ID: "ed-1", Label: "First edition", Year: "2026", Status: "Published",
					Formats: []types.EditionFormat{
						{
							ID: "fmt-1", Kind: types.EditionKindEbook, Format: "eBook",
							ISBN13: "978-1-9471345-1-5", EPUBVersion: "EPUB 3.3",
							PublicationDate: "2026-04-14", Status: "Published",
							ImprintOfRecord: "Bellwether House", RightsNotice: "All rights reserved",
						},
					},
				},
				{
					ID: "ed-2", Label: "Second edition", Year: "2030", Status: "In progress",
					PreviousEditionID: "ed-1", RevisionNote: "Reset from the 2026 text.",
					Formats: []types.EditionFormat{
						{
							ID: "fmt-2", Kind: types.EditionKindEbook, Format: "eBook",
							ISBN13: "978-1-9471345-2-2", EPUBVersion: "EPUB 2.0.1",
							EditionStatement: "Second edition, revised",
							PublicationDate:  "2030-09-08", Status: "Registered",
							ImprintOfRecord: "Harrowgate Press", RightsNotice: "CC BY-NC-ND 4.0",
						},
						{
							ID: "fmt-3", Kind: types.EditionKindPrint, Format: "Paperback",
							ISBN13: "978-1-9471345-3-9", Trim: "6 × 9 in (trade)",
							PageCount: "412", PaperStock: "Cream, 55#", Binding: "Perfect bound",
							PublicationDate: "2030-09-08", Status: "Registered", Gutter: "0.9",
						},
					},
				},
			},
		},
	}
}

func editionOptions(formatID string) types.ExportOptions {
	options := defaultExportOptions()
	options.FormatID = formatID
	return options
}

// inventedArtwork draws a cover from arithmetic: a vertical wash with a
// lighter band across it, at the proportions of Draftline's own cover
// derivative. No image file is read from anywhere.
func inventedArtwork(t *testing.T, width, height int) *CoverArt {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		shade := uint8(40 + (180 * y / height))
		for x := 0; x < width; x++ {
			r := shade
			if y > height/3 && y < height/3+height/12 {
				r = 230
			}
			img.Set(x, y, color.RGBA{R: r, G: shade / 2, B: 255 - shade, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 88}); err != nil {
		t.Fatalf("encode invented artwork: %v", err)
	}
	return &CoverArt{
		Data: buf.Bytes(), MediaType: "image/jpeg", FileName: "cover.jpg",
		Width: width, Height: height,
	}
}

// epubParts reads a written .epub back into a map of member name to bytes.
func epubParts(t *testing.T, path string) map[string]string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer func() { _ = zr.Close() }()
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		_ = rc.Close()
		out[f.Name] = buf.String()
	}
	return out
}

func exportEPUBTo(t *testing.T, book types.BookData, formatID string, cover *CoverArt) map[string]string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "book.epub")
	res := EPUB(path, book, types.EPUBOptions{ExportOptions: editionOptions(formatID)}, cover)
	if !res.Success {
		t.Fatalf("EPUB export failed: %s", res.Error)
	}
	return epubParts(t, path)
}

// ── The payoff ─────────────────────────────────────────────────────────────

// The whole point of the milestone: pick the registered ebook and the file
// that comes out is that edition's file.
func TestRegisteredEbookExportsAsThatEdition(t *testing.T) {
	book := editionBook()
	cover := inventedArtwork(t, 800, 1280)
	parts := exportEPUBTo(t, book, "fmt-1", cover)

	opf := parts["OEBPS/content.opf"]
	if opf == "" {
		t.Fatal("no package document in the exported file")
	}

	// The identifier is this format's ISBN, written plainly.
	if !strings.Contains(opf, `<dc:identifier id="uid">urn:isbn:9781947134515</dc:identifier>`) {
		t.Errorf("the package does not identify itself by the edition's ISBN:\n%s", opf)
	}
	// The cover reaches the file three ways, because three generations of
	// reading system look for it in three places.
	if !strings.Contains(opf, `properties="cover-image"`) {
		t.Errorf("no cover-image item in the manifest:\n%s", opf)
	}
	if !strings.Contains(opf, `<meta name="cover" content="cover-image"/>`) {
		t.Errorf("no EPUB 2 cover meta:\n%s", opf)
	}
	if _, ok := parts["OEBPS/images/cover.jpg"]; !ok {
		t.Errorf("the artwork itself is not in the package: %v", memberNames(parts))
	}
	// And the cover page is the first thing opened.
	spine := between(t, opf, "<spine", "</spine>")
	first := regexp.MustCompile(`idref="([^"]+)"`).FindStringSubmatch(spine)
	if first == nil || first[1] != epubCoverPageID {
		t.Errorf("the cover page is not first in the spine:\n%s", spine)
	}
	if page := parts["OEBPS/text/"+epubCoverPageDoc]; !strings.Contains(page, `src="../images/cover.jpg"`) {
		t.Errorf("the cover page does not show the artwork:\n%s", page)
	}

	// The copyright page is generated from the record, and says what this
	// object is.
	copyrightPage := parts["OEBPS/text/section-0001.xhtml"]
	for _, want := range []string{
		"The Quiet Ledger",
		"Copyright © 2026 by Ilse Marchetti",
		"All rights reserved.",
		"First edition, April 2026",
		"Published by Bellwether House",
		"ISBN 978-1-9471345-1-5 (ebook)",
	} {
		if !strings.Contains(copyrightPage, want) {
			t.Errorf("the copyright page does not carry %q:\n%s", want, copyrightPage)
		}
	}
	// What the author wrote on their own copyright page is kept, underneath.
	if !strings.Contains(copyrightPage, "No part of this book may be reproduced") {
		t.Errorf("the author's own copyright text was dropped:\n%s", copyrightPage)
	}

	// The record's own publication date and rights are declared.
	if !strings.Contains(opf, `<dc:date>2026-04-14</dc:date>`) {
		t.Errorf("no publication date from the edition:\n%s", opf)
	}
	if !strings.Contains(opf, `<dc:rights>All rights reserved.</dc:rights>`) {
		t.Errorf("no rights line from the edition:\n%s", opf)
	}
}

// A second edition carries its own ISBN, its own imprint, both copyright
// years, and its revision note — none of which are facts about the book.
func TestASecondEditionExportsItsOwnFacts(t *testing.T) {
	parts := exportEPUBTo(t, editionBook(), "fmt-2", nil)
	opf := parts["OEBPS/content.opf"]

	if !strings.Contains(opf, "urn:isbn:9781947134522") {
		t.Errorf("the second edition does not carry its own ISBN:\n%s", opf)
	}
	if !strings.Contains(opf, "<dc:publisher>Harrowgate Press</dc:publisher>") {
		t.Errorf("the format's own imprint of record is not declared:\n%s", opf)
	}
	if !strings.Contains(opf, "<dc:rights>CC BY-NC-ND 4.0.</dc:rights>") {
		t.Errorf("the second edition's rights notice is not declared:\n%s", opf)
	}

	page := parts["OEBPS/text/section-0001.xhtml"]
	for _, want := range []string{
		"Copyright © 2026, 2030 by Ilse Marchetti",
		"Second edition, September 2030",
		"Published by Harrowgate Press",
		"Reset from the 2026 text.",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the copyright page does not carry %q:\n%s", want, page)
		}
	}
}

// ── The identifier ─────────────────────────────────────────────────────────

// An ISBN is a fact, so it does not change between exports. Two exports of the
// same edition made on different days are the same book to a reading device.
func TestTwoExportsOfOneEditionShareTheirIdentifier(t *testing.T) {
	book := editionBook()
	first := exportEPUBTo(t, book, "fmt-1", nil)["OEBPS/content.opf"]
	second := exportEPUBTo(t, book, "fmt-1", nil)["OEBPS/content.opf"]
	one, two := identifierOf(t, first), identifierOf(t, second)
	if one != two {
		t.Errorf("the same edition exported twice identifies itself differently: %q then %q", one, two)
	}
	if one != "urn:isbn:9781947134515" {
		t.Errorf("identifier = %q", one)
	}
}

// A format with no ISBN yet falls back to a random identifier. Two of them
// made in the same instant must still be two books, and both must be real
// version-4 UUIDs — the old clock-derived value was neither.
func TestFormatsWithoutAnISBNGetDistinctVersion4Identifiers(t *testing.T) {
	book := editionBook()
	book.Editions.Editions[0].Formats[0].ISBN13 = ""

	seen := map[string]bool{}
	for i := 0; i < 8; i++ {
		id := identifierOf(t, exportEPUBTo(t, book, "fmt-1", nil)["OEBPS/content.opf"])
		value, ok := strings.CutPrefix(id, "urn:uuid:")
		if !ok {
			t.Fatalf("a format with no ISBN should fall back to a UUID, got %q", id)
		}
		assertRFC4122Version4(t, value)
		if seen[value] {
			t.Fatalf("two exports issued together carry the same identifier %q", value)
		}
		seen[value] = true
	}
}

// assertRFC4122Version4 checks the two nibbles that make a UUID conformant:
// the version nibble must be 4 and the variant nibble must be 8, 9, a or b.
func assertRFC4122Version4(t *testing.T, value string) {
	t.Helper()
	fields := strings.Split(value, "-")
	if len(fields) != 5 || len(fields[0]) != 8 || len(fields[1]) != 4 || len(fields[2]) != 4 || len(fields[3]) != 4 || len(fields[4]) != 12 {
		t.Fatalf("%q is not shaped like a UUID", value)
	}
	if fields[2][0] != '4' {
		t.Errorf("%q does not declare version 4", value)
	}
	switch fields[3][0] {
	case '8', '9', 'a', 'b', 'A', 'B':
	default:
		t.Errorf("%q does not carry the RFC 4122 variant bits", value)
	}
}

// ── The declared EPUB ──────────────────────────────────────────────────────

// The format's Specification field decides which EPUB comes out. EPUB 2.0.1
// means an NCX, a guide, and none of EPUB 3's refinement metadata — a package
// that declares 2.0 and then uses EPUB 3 constructs is refused by the
// validators and misread by the readers it was chosen for.
func TestAnEPUB2FormatProducesAnEPUB2Package(t *testing.T) {
	parts := exportEPUBTo(t, editionBook(), "fmt-2", inventedArtwork(t, 600, 960))
	opf := parts["OEBPS/content.opf"]

	if !strings.Contains(opf, `<package xmlns="http://www.idpf.org/2007/opf" version="2.0"`) {
		t.Errorf("the package does not declare version 2.0:\n%s", opf)
	}
	if _, ok := parts["OEBPS/"+epubNCXName]; !ok {
		t.Errorf("an EPUB 2 package has no navigation without an NCX: %v", memberNames(parts))
	}
	if !strings.Contains(opf, `<spine toc="ncx">`) {
		t.Errorf("the spine does not point at the NCX:\n%s", opf)
	}
	if !strings.Contains(opf, `<reference type="cover"`) {
		t.Errorf("an EPUB 2 reader finds the cover through the guide, which is missing:\n%s", opf)
	}
	if strings.Contains(opf, "properties=") {
		t.Errorf("EPUB 2 manifests have no properties attribute:\n%s", opf)
	}
	if strings.Contains(opf, "refines=") || strings.Contains(opf, "dcterms:modified") {
		t.Errorf("EPUB 3 refinement metadata in an EPUB 2 package:\n%s", opf)
	}
	if ncx := parts["OEBPS/"+epubNCXName]; !strings.Contains(ncx, "urn:isbn:9781947134522") {
		t.Errorf("the NCX does not carry the package identifier:\n%s", ncx)
	}
}

// A package document is only half of an EPUB. The content documents inside it
// are the other half, and an EPUB 2 package wrapped around XHTML5 fails
// EPUBCheck on every single file — which would make the file an author chose
// EPUB 2.0.1 to produce the one a legacy retailer refuses.
//
// OPS 2.0.1 content is XHTML 1.1: no HTML5 doctype, no <meta charset>, no
// <section>, no <nav>, no epub: namespace, and none of the elements XHTML 1.1
// dropped.
func TestEPUB2ContentDocumentsAreXHTML11(t *testing.T) {
	book := editionBook()
	// Some struck-through text, so the run renderer is exercised too.
	book.Body[0].Content = `<p>The ferry left <s>early</s> before anyone counted.</p><hr/><p>Then it did not.</p>`
	parts := exportEPUBTo(t, book, "fmt-2", inventedArtwork(t, 600, 960))

	html5 := []string{"<!DOCTYPE html>", "<meta charset", "<section ", "<nav ", "epub:type", "xmlns:epub", "aria-hidden", "<s>"}
	for name, body := range parts {
		if !strings.HasSuffix(name, ".xhtml") {
			continue
		}
		for _, construct := range html5 {
			if strings.Contains(body, construct) {
				t.Errorf("%s is not an OPS 2.0.1 document: it contains %q\n%s", name, construct, body)
			}
		}
		if !strings.Contains(body, `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN"`) {
			t.Errorf("%s does not declare the XHTML 1.1 document type:\n%s", name, body)
		}
		if !strings.Contains(body, `<meta http-equiv="Content-Type"`) {
			t.Errorf("%s states its encoding the HTML5 way:\n%s", name, body)
		}
	}
	// The markup the EPUB 3 file expresses with epub:type survives as a class,
	// so one stylesheet still dresses both.
	if !anyPartContains(parts, `<div class="section chapter">`) {
		t.Errorf("no chapter kept its role: %v", memberNames(parts))
	}
	if !anyPartContains(parts, `<span class="strike">`) {
		t.Errorf("struck text was not rewritten for XHTML 1.1: %v", memberNames(parts))
	}
	if css := parts["OEBPS/styles/book.css"]; !strings.Contains(css, ".strike") || !strings.Contains(css, "div.section") {
		t.Errorf("the stylesheet does not dress the EPUB 2 markup:\n%s", css)
	}
	if nav := parts["OEBPS/nav.xhtml"]; !strings.Contains(nav, `<div class="toc" id="toc">`) {
		t.Errorf("the contents page still uses <nav>:\n%s", nav)
	}
}

// And the EPUB 3 file keeps every one of those constructs, because there they
// are what the specification asks for.
func TestEPUB3ContentDocumentsStayXHTML5(t *testing.T) {
	parts := exportEPUBTo(t, editionBook(), "fmt-1", inventedArtwork(t, 600, 960))
	for _, want := range []string{"<!DOCTYPE html>", "<meta charset", `<section epub:type="chapter">`, "xmlns:epub"} {
		if !anyPartContains(parts, want) {
			t.Errorf("no EPUB 3 content document carries %q: %v", want, memberNames(parts))
		}
	}
	if nav := parts["OEBPS/nav.xhtml"]; !strings.Contains(nav, `<nav epub:type="toc" id="toc">`) {
		t.Errorf("the EPUB 3 navigation document is not a nav:\n%s", nav)
	}
	if cover := parts["OEBPS/text/"+epubCoverPageDoc]; !strings.Contains(cover, `epub:type="cover"`) {
		t.Errorf("the EPUB 3 cover page is not marked as one:\n%s", cover)
	}
}

// dc:date is constrained to W3CDTF and the record's publication date is free
// text, because "Spring 2027" is a real answer to give a contract. The file
// declares what it is allowed to declare and leaves out what it is not, rather
// than shipping a package no validator will pass.
func TestOnlyADateTheSpecificationAllowsIsDeclared(t *testing.T) {
	cases := []struct {
		record string
		want   string
	}{
		{"2026-04-14", "2026-04-14"},
		{"2026-04", "2026-04"},
		{"2026", "2026"},
		{"Spring 2027", ""},
		{"14/04/2026", ""},
		{"2026-13-01", ""},
		{"", ""},
	}
	for _, tc := range cases {
		book := editionBook()
		book.Editions.Editions[0].Formats[0].PublicationDate = tc.record
		opf := exportEPUBTo(t, book, "fmt-1", nil)["OEBPS/content.opf"]
		if tc.want == "" {
			if strings.Contains(opf, "<dc:date>") {
				t.Errorf("%q reached dc:date, which only takes a W3CDTF value:\n%s", tc.record, opf)
			}
			continue
		}
		if !strings.Contains(opf, "<dc:date>"+tc.want+"</dc:date>") {
			t.Errorf("%q did not become <dc:date>%s</dc:date>:\n%s", tc.record, tc.want, opf)
		}
	}
}

// EPUB 3.3 and EPUB 3.0 both declare version="3.0", because that attribute has
// only ever had two legal values. The revision the author chose is recorded
// where a revision belongs.
func TestAnEPUB3FormatDeclaresThreePointZeroAndNamesItsRevision(t *testing.T) {
	book := editionBook()
	parts := exportEPUBTo(t, book, "fmt-1", nil)
	opf := parts["OEBPS/content.opf"]
	if !strings.Contains(opf, `version="3.0"`) {
		t.Errorf("an EPUB 3 package declares version 3.0:\n%s", opf)
	}
	if !strings.Contains(opf, "<meta property=\"dcterms:conformsTo\">https://www.w3.org/TR/epub-33/</meta>") {
		t.Errorf("the chosen revision is not recorded:\n%s", opf)
	}

	book.Editions.Editions[0].Formats[0].EPUBVersion = "EPUB 3.0"
	opf = exportEPUBTo(t, book, "fmt-1", nil)["OEBPS/content.opf"]
	if !strings.Contains(opf, "https://www.w3.org/TR/epub-30/") {
		t.Errorf("choosing EPUB 3.0 changed nothing in the file:\n%s", opf)
	}
	if !strings.Contains(opf, `version="3.0"`) {
		t.Errorf("an EPUB 3.0 package declares version 3.0:\n%s", opf)
	}
}

// The edition statement is a title of the publication, not of the work, and
// EPUB 3's own vocabulary has a word for that.
func TestTheEditionStatementIsDeclaredAsAnEditionTitle(t *testing.T) {
	book := editionBook()
	book.Editions.Editions[0].Formats[0].EditionStatement = "First edition"
	opf := exportEPUBTo(t, book, "fmt-1", nil)["OEBPS/content.opf"]
	if !strings.Contains(opf, `<dc:title id="edition-statement">First edition</dc:title>`) {
		t.Errorf("the edition statement is not declared:\n%s", opf)
	}
	if !strings.Contains(opf, `<meta refines="#edition-statement" property="title-type">edition</meta>`) {
		t.Errorf("the edition title is not typed as an edition:\n%s", opf)
	}
}

// ── The package stays well formed ──────────────────────────────────────────

// Whatever an edition asks for, the package document has to hold together:
// one unique identifier, no empty Dublin Core, and a spine that refers only to
// things the manifest declares.
func TestEditionPackagesStayInternallyConsistent(t *testing.T) {
	book := editionBook()
	for _, tc := range []struct {
		name     string
		formatID string
		cover    *CoverArt
	}{
		{"first edition ebook with a cover", "fmt-1", inventedArtwork(t, 500, 800)},
		{"second edition ebook, EPUB 2, no cover", "fmt-2", nil},
		{"no edition at all", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opf := exportEPUBTo(t, book, tc.formatID, tc.cover)["OEBPS/content.opf"]

			if n := strings.Count(opf, `id="uid"`); n != 1 {
				t.Errorf("%d elements claim to be the unique identifier", n)
			}
			if m := regexp.MustCompile(`<dc:[a-zA-Z]+[^>]*>\s*</dc:[a-zA-Z]+>`).FindString(opf); m != "" {
				t.Errorf("empty Dublin Core element %q", m)
			}

			declared := map[string]bool{}
			for _, m := range regexp.MustCompile(`<item id="([^"]+)"`).FindAllStringSubmatch(opf, -1) {
				declared[m[1]] = true
			}
			refs := regexp.MustCompile(`<itemref idref="([^"]+)"`).FindAllStringSubmatch(opf, -1)
			if len(refs) == 0 {
				t.Fatal("the spine is empty")
			}
			for _, m := range refs {
				if !declared[m[1]] {
					t.Errorf("the spine refers to %q, which the manifest does not declare", m[1])
				}
			}
		})
	}
}

// A book with no publishing record at all exports exactly what it did before:
// the identifier comes from the book's own list, and no edition facts appear.
func TestABookWithNoEditionsExportsAsItAlwaysDid(t *testing.T) {
	book := editionBook()
	book.Editions = nil
	book.Metadata.ISBNs = []types.ISBNEntry{{Format: "ebook", Value: "978-1-9471345-9-1"}}

	parts := exportEPUBTo(t, book, "", nil)
	opf := parts["OEBPS/content.opf"]
	if !strings.Contains(opf, "urn:isbn:9781947134591") {
		t.Errorf("the book's own ISBN is no longer used:\n%s", opf)
	}
	if strings.Contains(opf, "<dc:rights>") || strings.Contains(opf, "<dc:date>") {
		t.Errorf("edition facts appeared on a book with no editions:\n%s", opf)
	}
	page := parts["OEBPS/text/section-0001.xhtml"]
	if strings.Contains(page, "All rights reserved") {
		t.Errorf("a copyright page was generated for a book with no editions:\n%s", page)
	}
	if !strings.Contains(page, "No part of this book may be reproduced") {
		t.Errorf("the author's own copyright page is missing:\n%s", page)
	}
}

// ── The trim ───────────────────────────────────────────────────────────────

// A trim now arrives from a stored record rather than being typed this
// session, so a number nobody looked at has to be refused rather than
// typeset. "99" is a ninety-nine inch page.
func TestACustomTrimIsBoundedAtBothEnds(t *testing.T) {
	for _, tc := range []struct {
		width, height string
		accepted      bool
	}{
		{"6", "9", true},
		{"5.5", "8.5", true},
		{"3", "12", true},
		{"99", "9", false},
		{"6", "99", false},
		{"0.2", "9", false},
		{"", "9", false},
		{"six", "9", false},
		{"-6", "9", false},
	} {
		name := fmt.Sprintf("%s by %s", tc.width, tc.height)
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "print.pdf")
			options := types.PrintPDFOptions{
				PDFOptions:   types.PDFOptions{ExportOptions: defaultExportOptions(), FontSize: 11},
				TrimSize:     "custom",
				CustomWidth:  tc.width,
				CustomHeight: tc.height,
			}
			res := PrintPDF(path, editionBook(), options)
			if res.Success != tc.accepted {
				t.Fatalf("success = %v, want %v (%s)", res.Success, tc.accepted, res.Error)
			}
			if tc.accepted {
				return
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("a refused trim still wrote a file")
			}
			if !strings.Contains(res.Error, "3") || !strings.Contains(res.Error, "12") {
				t.Errorf("the refusal does not say what a usable trim is: %q", res.Error)
			}
		})
	}
}

// A named trim is Draftline's own and is never refused.
func TestANamedTrimIsNeverRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "print.pdf")
	options := types.PrintPDFOptions{
		PDFOptions:   types.PDFOptions{ExportOptions: defaultExportOptions(), FontSize: 11},
		TrimSize:     "6x9",
		CustomWidth:  "99",
		CustomHeight: "99",
	}
	if res := PrintPDF(path, editionBook(), options); !res.Success {
		t.Fatalf("a named trim was refused: %s", res.Error)
	}
}

// ── The cover in a PDF ─────────────────────────────────────────────────────

// A reading copy carries the cover: one more page than the same export
// without it, and the artwork embedded rather than described.
func TestTheReadingPDFCarriesTheCoverOnItsOwnPage(t *testing.T) {
	book := editionBook()
	options := types.PDFOptions{ExportOptions: editionOptions("fmt-1"), PageSize: "6x9", FontSize: 11}

	dir := t.TempDir()
	bare := filepath.Join(dir, "bare.pdf")
	if res := PDF(bare, book, options, nil); !res.Success {
		t.Fatalf("export without a cover failed: %s", res.Error)
	}
	withCover := filepath.Join(dir, "cover.pdf")
	if res := PDF(withCover, book, options, inventedArtwork(t, 800, 1280)); !res.Success {
		t.Fatalf("export with a cover failed: %s", res.Error)
	}

	before, after := pdfPageCount(t, bare), pdfPageCount(t, withCover)
	if after != before+1 {
		t.Errorf("a cover added %d pages, want exactly 1", after-before)
	}
	data, err := os.ReadFile(withCover)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("/Subtype /Image")) {
		t.Errorf("no image is embedded in the exported PDF")
	}
	assertParseablePDF(t, data)
}

// The print interior never carries the cover. A print-on-demand service wants
// the interior alone; a cover bound into it becomes page one of the block, and
// Draftline's derivative is smaller than the trim it would be printed at.
func TestThePrintInteriorNeverCarriesTheCover(t *testing.T) {
	book := editionBook()
	options := types.PrintPDFOptions{
		PDFOptions: types.PDFOptions{ExportOptions: editionOptions("fmt-3"), FontSize: 11},
		TrimSize:   "6x9",
	}
	path := filepath.Join(t.TempDir(), "print.pdf")
	if res := PrintPDF(path, book, options); !res.Success {
		t.Fatalf("print export failed: %s", res.Error)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("/Subtype /Image")) {
		t.Errorf("the print interior has an image in it")
	}
}

// A cover that will not decode costs the author a cover page, never the book.
func TestUnreadableArtworkDoesNotStopTheExport(t *testing.T) {
	book := editionBook()
	options := types.PDFOptions{ExportOptions: editionOptions("fmt-1"), PageSize: "6x9", FontSize: 11}
	broken := &CoverArt{Data: []byte("this is not a JPEG"), MediaType: "image/jpeg", FileName: "cover.jpg", Width: 800, Height: 1280}
	path := filepath.Join(t.TempDir(), "broken.pdf")
	res := PDF(path, book, options, broken)
	if !res.Success {
		t.Fatalf("a bad cover stopped the export: %s", res.Error)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	assertParseablePDF(t, data)
}

// ── The selection itself ───────────────────────────────────────────────────

// A format that has gone — deleted between opening the wizard and finishing
// it — exports the book rather than failing. The author asked for a file.
func TestAMissingFormatFallsBackToTheBook(t *testing.T) {
	book := editionBook()
	doc, err := BuildDocument(book, editionOptions("fmt-does-not-exist"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Edition != nil {
		t.Errorf("a format that is not in the record was resolved anyway: %#v", doc.Edition)
	}
}

// Cover art is recorded on the edition, so the derived facts about a printed
// object come out of the record rather than being typed twice.
func TestTheSelectionCarriesTheDerivedPrintFacts(t *testing.T) {
	doc, err := BuildDocument(editionBook(), editionOptions("fmt-3"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Edition == nil {
		t.Fatal("the paperback was not resolved")
	}
	if doc.Edition.Spine != "1.037 in" {
		t.Errorf("spine = %q, want 1.037 in", doc.Edition.Spine)
	}
	if doc.Edition.Trim != "6 × 9 in (trade)" {
		t.Errorf("trim = %q", doc.Edition.Trim)
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func memberNames(parts map[string]string) []string {
	names := make([]string, 0, len(parts))
	for name := range parts {
		names = append(names, name)
	}
	return names
}

func between(t *testing.T, s, open, close string) string {
	t.Helper()
	start := strings.Index(s, open)
	end := strings.Index(s, close)
	if start < 0 || end < start {
		t.Fatalf("no %q ... %q in:\n%s", open, close, s)
	}
	return s[start : end+len(close)]
}

var pdfPageMarker = regexp.MustCompile(`/Type\s*/Page[^s]`)

func pdfPageCount(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return len(pdfPageMarker.FindAll(data, -1))
}

// anyPartContains says whether any content document in the package carries a
// construct. Which numbered section a chapter lands in depends on how much
// front matter precedes it, and that is not what these tests are about.
func anyPartContains(parts map[string]string, want string) bool {
	for name, body := range parts {
		if strings.HasSuffix(name, ".xhtml") && strings.Contains(body, want) {
			return true
		}
	}
	return false
}
