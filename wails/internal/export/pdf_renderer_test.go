package export

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"draftline/internal/types"
)

func TestPDFSpecsHonorReadingAndPrintControls(t *testing.T) {
	reading := readingPDFSpec(types.PDFOptions{
		PageSize:        "a4",
		FontFamily:      "lato",
		FontSize:        14,
		LineHeight:      1.6,
		ParagraphIndent: "0.5",
		TextAlign:       "justify",
	})
	if reading.TrimWidth != 595.28 || reading.TrimHeight != 841.89 {
		t.Fatalf("A4 page ignored: %gx%g", reading.TrimWidth, reading.TrimHeight)
	}
	if reading.Font.ID != "Lato" || reading.FontSize != 14 || math.Abs(reading.LineHeight-22.4) > 0.0001 {
		t.Fatalf("reading typography ignored: font=%s size=%g line=%g", reading.Font.ID, reading.FontSize, reading.LineHeight)
	}
	if reading.ParagraphIndent != 36 || reading.TextAlign != "justify" {
		t.Fatalf("reading composition ignored: %#v", reading)
	}

	printSpec := printPDFSpec(types.PrintPDFOptions{
		PDFOptions: types.PDFOptions{
			FontFamily:      "merriweather",
			FontSize:        10,
			LineHeight:      1.3,
			ParagraphIndent: "0.3",
			TextAlign:       "left",
		},
		TrimSize:               "custom",
		CustomWidth:            "5.75",
		CustomHeight:           "8.25",
		Bleed:                  "0.125",
		GutterMargin:           "0.9",
		OuterMargin:            "0.6",
		TopMargin:              "0.7",
		BottomMargin:           "0.65",
		IncludeCropMarks:       true,
		ChapterStartsRecto:     true,
		DropCap:                true,
		DropCapLines:           4,
		RunningHeaders:         true,
		HeaderStyle:            "italic",
		PageNumberPosition:     "top-outside",
		GenerateHalfTitle:      true,
		GenerateTOC:            true,
		MirroredMargins:        true,
		HeadingFont:            "fantasy",
		FurnitureFont:          "modern",
		TitlePageFont:          "romance",
		TitlePageStyle:         "dramatic",
		TitlePageShowAuthor:    true,
		TitlePageShowPublisher: true,
	})
	if printSpec.TrimWidth != 414 || printSpec.TrimHeight != 594 || printSpec.Bleed != 9 {
		t.Fatalf("custom print page ignored: %#v", printSpec)
	}
	if math.Abs(printSpec.GutterMargin-64.8) > 0.0001 || math.Abs(printSpec.OuterMargin-43.2) > 0.0001 || math.Abs(printSpec.ParagraphIndent-21.6) > 0.0001 {
		t.Fatalf("print margins/indent ignored: gutter=%g outer=%g indent=%g", printSpec.GutterMargin, printSpec.OuterMargin, printSpec.ParagraphIndent)
	}
	if !printSpec.CropMarks || !printSpec.MirroredMargins || !printSpec.DropCap || printSpec.DropCapLines != 4 {
		t.Fatalf("print toggles ignored: %#v", printSpec)
	}
	if printSpec.HeadingFont.ID != "CinzelDecorative" || printSpec.FurnitureFont.ID != "Lato" || printSpec.TitlePageFont.ID != "GreatVibes" {
		t.Fatalf("display font controls ignored: heading=%s furniture=%s title=%s", printSpec.HeadingFont.ID, printSpec.FurnitureFont.ID, printSpec.TitlePageFont.ID)
	}
	if printSpec.TitlePageStyle != "dramatic" || !printSpec.TitlePageShowAuthor || !printSpec.TitlePageShowPublisher {
		t.Fatalf("title page controls ignored: %#v", printSpec)
	}
}

func TestPrintPDFSpecDefaultsToLeftAlignedTenPointBody(t *testing.T) {
	spec := printPDFSpec(types.PrintPDFOptions{})
	if spec.TextAlign != "left" {
		t.Fatalf("print alignment defaulted to %q, want left", spec.TextAlign)
	}
	if spec.FontSize != 10 {
		t.Fatalf("print type size defaulted to %g, want 10", spec.FontSize)
	}
	if spec.HeadingFont.ID != "EBGaramond" {
		t.Fatalf("default heading face = %q, want EBGaramond", spec.HeadingFont.ID)
	}
}

func TestDisplayFontPresetsResolveAndRender(t *testing.T) {
	want := map[string]string{
		"body": "Merriweather", "classic": "EBGaramond", "modern": "Lato",
		"romance": "GreatVibes", "scifi": "Orbitron", "fantasy": "CinzelDecorative",
	}
	body := resolvePDFFont("merriweather")
	for preset, familyID := range want {
		family := resolveDisplayFont(preset, body)
		if family.ID != familyID {
			t.Errorf("%s resolved to %s, want %s", preset, family.ID, familyID)
			continue
		}
		doc := Document{Title: "Display", Author: "Writer", Sections: []DocumentSection{{Title: "Chapter", Role: SectionBody, Blocks: []DocumentBlock{{Kind: BlockParagraph, Runs: []DocumentRun{{Text: "Body text."}}}}}}}
		spec := publicationPDFSpec{Print: true, TrimWidth: 360, TrimHeight: 576, GutterMargin: 54, OuterMargin: 45, TopMargin: 54, BottomMargin: 45, Font: body, HeadingFont: family, FurnitureFont: family, TitlePageFont: family, FontSize: 10, LineHeight: 14, TextAlign: "left", TitlePageStyle: "classic", TitlePageShowAuthor: true}
		data, err := renderPublicationPDF(doc, spec)
		if err != nil {
			t.Errorf("%s could not render: %v", preset, err)
			continue
		}
		assertParseablePDF(t, data)
	}
}

func TestPrintRendererCreatesTrueRectoStartsAndTOCLinks(t *testing.T) {
	doc := Document{
		Title:  "Test Edition",
		Author: "A. Writer",
		Sections: []DocumentSection{
			{ID: "one", Title: "Chapter One", Role: SectionBody, Blocks: []DocumentBlock{{Kind: BlockParagraph, Runs: []DocumentRun{{Text: "First chapter."}}}}},
			{ID: "two", Title: "Chapter Two", Role: SectionBody, Blocks: []DocumentBlock{{Kind: BlockParagraph, Runs: []DocumentRun{{Text: "Second chapter."}}}}},
		},
	}
	spec := publicationPDFSpec{
		Print: true, TrimWidth: 396, TrimHeight: 612,
		GutterMargin: 63, OuterMargin: 45, TopMargin: 54, BottomMargin: 45,
		Font: resolvePDFFont("merriweather"), FontSize: 11, LineHeight: 15.4,
		ParagraphIndent: 18, TextAlign: "justify", ChapterStartsRecto: true,
		GenerateTOC: true, PageNumberPosition: "bottom-center",
	}
	renderer := newPublicationPDFRenderer(doc, spec)
	if err := renderer.render(); err != nil {
		t.Fatal(err)
	}
	if renderer.pdf.PageNo() != renderer.pdf.PageCount() {
		t.Fatalf("active page %d, final page %d; TOC backfill did not restore the manuscript end", renderer.pdf.PageNo(), renderer.pdf.PageCount())
	}
	if len(renderer.tocPages) != 1 {
		t.Fatalf("TOC pages = %v", renderer.tocPages)
	}
	if len(renderer.toc) != 2 {
		t.Fatalf("TOC entries = %#v", renderer.toc)
	}
	for _, entry := range renderer.toc {
		if entry.Page%2 != 1 {
			t.Errorf("%q starts on verso page %d", entry.Title, entry.Page)
		}
	}
	if renderer.toc[1].Page <= renderer.toc[0].Page {
		t.Errorf("chapter page order is not increasing: %#v", renderer.toc)
	}
}

func TestPublicationPDFEmbedsUnicodeFontsAndPrintBoxes(t *testing.T) {
	doc := Document{
		Title:  "Curly \u201cQuotes\u201d \u2014 caf\u00e9",
		Author: "A. Writer",
		Sections: []DocumentSection{{
			Title: "Chapter One", Role: SectionBody,
			Blocks: []DocumentBlock{{Kind: BlockParagraph, Runs: []DocumentRun{
				{Text: "It\u2019s emphasized ", Italic: true},
				{Text: "and bold. ", Bold: true},
				{Text: "Reference", Href: "https://example.com"},
			}}},
		}},
	}
	spec := publicationPDFSpec{
		Print: true, TrimWidth: 432, TrimHeight: 648, Bleed: 9, CropMarks: true,
		GutterMargin: 63, OuterMargin: 45, TopMargin: 54, BottomMargin: 45,
		Font: resolvePDFFont("merriweather"), FontSize: 11, LineHeight: 15.4,
		TextAlign: "justify", PageNumberPosition: "bottom-center",
	}
	pdf, err := renderPublicationPDF(doc, spec)
	if err != nil {
		t.Fatal(err)
	}
	assertParseablePDF(t, pdf)
	if !bytes.Contains(pdf, []byte("/FontFile2")) {
		t.Fatal("PDF does not embed a TrueType font program")
	}
	if bytes.Contains(pdf, []byte("/Helvetica")) || bytes.Contains(pdf, []byte("/WinAnsiEncoding")) {
		t.Fatal("legacy built-in PDF font path is still present")
	}
	if !bytes.Contains(pdf, []byte("/URI (https://example.com)")) {
		t.Fatal("PDF did not preserve a safe authored hyperlink as an annotation")
	}
	for _, box := range []string{"/TrimBox", "/BleedBox", "/CropBox"} {
		if !bytes.Contains(pdf, []byte(box)) {
			t.Errorf("print PDF missing %s", box)
		}
	}
	// 6x9 trim + 0.125in bleed + 0.25in mark canvas on every edge.
	if !regexp.MustCompile(`/MediaBox\s*\[\s*0(?:\.00)?\s+0(?:\.00)?\s+486(?:\.00)?\s+702(?:\.00)?\s*\]`).Match(pdf) {
		t.Fatalf("unexpected media box; crop/bleed canvas was not applied")
	}
}

func TestReadingTypographyChangesPagination(t *testing.T) {
	paragraph := "<p>" + repeatedWords(4200) + "</p>"
	book := types.BookData{
		Metadata: types.Metadata{Title: "Pagination"},
		Body:     []types.ChapterItem{{Title: "Chapter", Content: paragraph}},
	}
	doc, err := BuildDocument(book, types.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	compact := newPublicationPDFRenderer(doc, readingPDFSpec(types.PDFOptions{PageSize: "letter", FontFamily: "lato", FontSize: 11, LineHeight: 1.3, ParagraphIndent: "0.2", TextAlign: "left"}))
	if err := compact.render(); err != nil {
		t.Fatal(err)
	}
	open := newPublicationPDFRenderer(doc, readingPDFSpec(types.PDFOptions{PageSize: "letter", FontFamily: "lato", FontSize: 14, LineHeight: 1.6, ParagraphIndent: "0.2", TextAlign: "left"}))
	if err := open.render(); err != nil {
		t.Fatal(err)
	}
	if open.pdf.PageNo() <= compact.pdf.PageNo() {
		t.Fatalf("larger/open typography pages=%d, compact pages=%d; controls did not affect layout", open.pdf.PageNo(), compact.pdf.PageNo())
	}
}

func TestWriteExportFileReplacesOnlyAfterCompleteWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "edition.pdf")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeExportFile(path, []byte("new complete export")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new complete export" {
		t.Fatalf("file content = %q", got)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".draftline-export-*.tmp"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary export files remain: %v (err %v)", matches, err)
	}
}

func repeatedWords(count int) string {
	var out bytes.Buffer
	for i := 0; i < count; i++ {
		out.WriteString("measured typography ")
	}
	return out.String()
}
