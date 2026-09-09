package export

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
	"time"

	"draftline/internal/types"
)

func TestEPUBUsesSharedDocumentAndWellFormedXHTML(t *testing.T) {
	book := types.BookData{
		Metadata:    types.Metadata{Title: "A & B", Author: "Writer <One>", Publisher: "Press & Co"},
		Copyright:   "<p>Copyright © 2026</p>",
		FrontMatter: []types.ChapterItem{{Title: "Foreword", Content: "<p>Opening note.</p>"}},
		Body: []types.ChapterItem{{
			Title: "Chapter & One", Subtitle: "A beginning",
			Content: `<p class="editor-only">A <strong>bold</strong> and <em>curly—quoted</em> line.<br/>Next line with <a href="https://example.com?a=1&amp;b=2">a link</a> and <a href="javascript:alert(1)">unsafe link text</a>.</p><hr/><ul><li>First</li><li>Second</li></ul>`,
		}},
		BackMatter: []types.ChapterItem{{Title: "Notes", Content: "<p>Closing note.</p>"}},
	}
	doc, err := BuildDocument(book, types.ExportOptions{IncludeCopyright: true, IncludeFrontMatter: true, IncludeBackMatter: true})
	if err != nil {
		t.Fatal(err)
	}
	data, err := renderEPUB(doc, book, normalizeEPUBOptions(types.EPUBOptions{
		FontFamily: "reader", ParagraphStyle: "indented", TextAlign: "reader",
		ChapterStyle: "classic", SceneBreakStyle: "asterism",
	}), time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	parts := readEPUBParts(t, data)
	for _, required := range []string{
		"mimetype", "META-INF/container.xml", "OEBPS/content.opf", "OEBPS/nav.xhtml",
		"OEBPS/styles/book.css", "OEBPS/text/section-0001.xhtml", "OEBPS/text/section-0002.xhtml",
		"OEBPS/text/section-0003.xhtml", "OEBPS/text/section-0004.xhtml",
	} {
		if _, ok := parts[required]; !ok {
			t.Errorf("EPUB is missing %s", required)
		}
	}
	for name, part := range parts {
		if strings.HasSuffix(name, ".xhtml") || strings.HasSuffix(name, ".opf") || strings.HasSuffix(name, ".xml") {
			assertWellFormedXML(t, name, part)
		}
	}
	body := string(parts["OEBPS/text/section-0003.xhtml"])
	for _, want := range []string{"<strong>bold</strong>", "<em>curly—quoted</em>", "<br/>", "A beginning", "⁂", "<ul>", "<li>First</li>"} {
		if !strings.Contains(body, want) {
			t.Errorf("chapter XHTML missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "editor-only") || strings.Contains(body, "javascript:") {
		t.Fatalf("editor markup or unsafe link escaped the normalized document:\n%s", body)
	}
	if !strings.Contains(body, `href="https://example.com?a=1&amp;b=2"`) {
		t.Errorf("safe link was not preserved: %s", body)
	}
}

func TestEPUBOptionsChangeResponsiveStylesAndEmbeddedFonts(t *testing.T) {
	book := types.BookData{Metadata: types.Metadata{Title: "Styles"}, Body: []types.ChapterItem{{Title: "One", Content: "<p>Text.</p><hr/><p>More.</p>"}}}
	doc, err := BuildDocument(book, types.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	options := normalizeEPUBOptions(types.EPUBOptions{
		FontFamily: "lato", ParagraphStyle: "spaced", TextAlign: "justify",
		ChapterStyle: "minimal", SceneBreakStyle: "rule",
	})
	data, err := renderEPUB(doc, book, options, time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	parts := readEPUBParts(t, data)
	css := string(parts["OEBPS/styles/book.css"])
	for _, want := range []string{"text-indent: 0", "text-align: justify", "font-family: 'Lato'", "lato-regular.ttf"} {
		if !strings.Contains(css, want) {
			t.Errorf("responsive CSS missing %q:\n%s", want, css)
		}
	}
	for _, name := range []string{
		"OEBPS/fonts/lato-regular.ttf", "OEBPS/fonts/lato-bold.ttf",
		"OEBPS/fonts/lato-italic.ttf", "OEBPS/fonts/lato-bold-italic.ttf",
	} {
		if len(parts[name]) < 100_000 {
			t.Errorf("embedded font %s missing or implausibly small", name)
		}
	}
	opf := string(parts["OEBPS/content.opf"])
	if !strings.Contains(opf, `media-type="font/ttf"`) {
		t.Errorf("embedded fonts are absent from manifest: %s", opf)
	}
	body := string(parts["OEBPS/text/section-0001.xhtml"])
	if !strings.Contains(body, `class="body-matter minimal"`) || !strings.Contains(body, `scene-rule`) {
		t.Errorf("chapter/scene styles did not reach XHTML: %s", body)
	}
}

func TestEPUBReaderTypographyDoesNotEmbedFonts(t *testing.T) {
	options := normalizeEPUBOptions(types.EPUBOptions{})
	if options.FontFamily != "reader" || options.ParagraphStyle != "indented" || options.ChapterStyle != "classic" || options.SceneBreakStyle != "asterism" {
		t.Fatalf("unexpected accessible defaults: %+v", options)
	}
	book := types.BookData{Metadata: types.Metadata{Title: "Reader"}, Body: []types.ChapterItem{{Title: "One", Content: "<p>Text.</p>"}}}
	doc, err := BuildDocument(book, types.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := renderEPUB(doc, book, options, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	parts := readEPUBParts(t, data)
	for name := range parts {
		if strings.HasPrefix(name, "OEBPS/fonts/") {
			t.Fatalf("reader-controlled EPUB unexpectedly embeds %s", name)
		}
	}
	if strings.Contains(string(parts["OEBPS/styles/book.css"]), "@font-face") {
		t.Fatal("reader-controlled EPUB still contains an embedded font declaration")
	}
}

func readEPUBParts(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(zr.File) == 0 || zr.File[0].Name != "mimetype" || zr.File[0].Method != zip.Store {
		t.Fatal("EPUB mimetype is not the first uncompressed entry")
	}
	parts := make(map[string][]byte, len(zr.File))
	for _, file := range zr.File {
		r, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			t.Fatal(err)
		}
		parts[file.Name] = content
	}
	return parts
}

func assertWellFormedXML(t *testing.T, name string, data []byte) {
	t.Helper()
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		if _, err := decoder.Token(); err != nil {
			if err == io.EOF {
				return
			}
			t.Fatalf("%s is not well-formed XML: %v", name, err)
		}
	}
}
