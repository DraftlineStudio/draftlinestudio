package export

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"draftline/internal/types"
)

func TestDOCXUsesSharedDocumentRichStructure(t *testing.T) {
	book := types.BookData{
		Metadata: types.Metadata{Title: "Editable", Author: "A. Writer"},
		Body: []types.ChapterItem{{
			Title: "Chapter One", Subtitle: "A subtitle",
			Content: `<p style="text-align: center"><strong>Bold</strong> <em>italic</em> <u>underlined</u> H<sub>2</sub>O and x<sup>2</sup>. <a href="https://example.com?a=1&amp;b=2">Source</a> <a href="javascript:bad()">unsafe</a></p><blockquote><p>A quotation.</p></blockquote><hr/><ol><li>First</li><li>Second</li></ol>`,
		}},
	}
	doc, err := BuildDocument(book, types.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := renderDOCX(doc, types.DOCXOptions{})
	if err != nil {
		t.Fatal(err)
	}
	parts := readEPUBParts(t, data)
	for _, name := range []string{
		"[Content_Types].xml", "_rels/.rels", "docProps/core.xml", "docProps/app.xml",
		"word/document.xml", "word/styles.xml", "word/numbering.xml", "word/_rels/document.xml.rels",
	} {
		part, ok := parts[name]
		if !ok {
			t.Errorf("DOCX missing %s", name)
			continue
		}
		assertWellFormedXML(t, name, part)
	}
	document := string(parts["word/document.xml"])
	for _, want := range []string{
		"<w:b/>", "<w:i/>", `<w:u w:val="single"/>`, `w:val="subscript"`, `w:val="superscript"`,
		`<w:jc w:val="center"/>`, `<w:pStyle w:val="Quote"/>`, `<w:numId w:val="2"/>`, "⁂",
		`<w:hyperlink r:id="rIdLink1">`,
	} {
		if !strings.Contains(document, want) {
			t.Errorf("document XML missing %q: %s", want, document)
		}
	}
	rels := string(parts["word/_rels/document.xml.rels"])
	if !strings.Contains(rels, `Target="https://example.com?a=1&amp;b=2"`) || strings.Contains(rels, "javascript:") {
		t.Errorf("DOCX relationships did not preserve only safe links: %s", rels)
	}
}

// Track changes has to be in the file, not a promise on a screen. With
// w:trackChanges in word/settings.xml an editor who opens the document is
// marking it up from the first keystroke rather than silently rewriting it,
// which is the difference between getting edits back and getting a new file.
func TestDOCXTrackChangesReachesTheFile(t *testing.T) {
	book := types.BookData{
		Metadata: types.Metadata{Title: "Wide Water", Author: "A. Marsh"},
		Body: []types.ChapterItem{{Title: "Chapter One", Type: "chapter",
			Content: "<p>Before the break.</p><hr /><p>After the break.</p>"}},
	}

	on, err := DOCXBytes(book, types.DOCXOptions{TrackChanges: true})
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	settings := docxPartText(t, on, "word/settings.xml")
	if !strings.Contains(settings, "<w:trackChanges/>") {
		t.Errorf("settings.xml does not turn revision recording on:\n%s", settings)
	}

	off, err := DOCXBytes(book, types.DOCXOptions{})
	if err != nil {
		t.Fatalf("rendering without track changes: %v", err)
	}
	if strings.Contains(docxPartText(t, off, "word/settings.xml"), "trackChanges") {
		t.Error("revision recording was turned on for a document that did not ask for it")
	}
}

func TestDOCXManuscriptStyleAndSceneBreaks(t *testing.T) {
	book := types.BookData{
		Metadata: types.Metadata{Title: "Wide Water", Author: "A. Marsh"},
		Body: []types.ChapterItem{{Title: "Chapter One", Type: "chapter",
			Content: "<p>Before.</p><hr /><p>After.</p>"}},
	}

	manuscript, err := DOCXBytes(book, types.DOCXOptions{BodyStyle: "manuscript", HashSceneBreaks: true})
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	styles := docxPartText(t, manuscript, "word/styles.xml")
	if !strings.Contains(styles, "Courier New") || !strings.Contains(styles, `w:line="480"`) {
		t.Error("the manuscript style is not 12 pt Courier, double spaced")
	}
	if body := docxPartText(t, manuscript, "word/document.xml"); !strings.Contains(body, ">#<") {
		t.Error("the scene break is not a hash")
	}

	normal, err := DOCXBytes(book, types.DOCXOptions{})
	if err != nil {
		t.Fatalf("rendering the normal style: %v", err)
	}
	if strings.Contains(docxPartText(t, normal, "word/styles.xml"), `w:line="480"`) {
		t.Error("the normal style came out double spaced")
	}
	if body := docxPartText(t, normal, "word/document.xml"); !strings.Contains(body, "⁂") {
		t.Error("the default scene break is not an asterism")
	}
}

// Chapters start a new page unless the author asks for one continuous run.
func TestDOCXChapterBreak(t *testing.T) {
	book := types.BookData{
		Metadata: types.Metadata{Title: "Wide Water", Author: "A. Marsh"},
		Body: []types.ChapterItem{
			{Title: "Chapter One", Type: "chapter", Content: "<p>One.</p>"},
			{Title: "Chapter Two", Type: "chapter", Content: "<p>Two.</p>"},
		},
	}
	paged, err := DOCXBytes(book, types.DOCXOptions{})
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	if !strings.Contains(docxPartText(t, paged, "word/document.xml"), "<w:pageBreakBefore/>") {
		t.Error("chapters do not start a new page by default")
	}
	run, err := DOCXBytes(book, types.DOCXOptions{ChapterBreak: "run"})
	if err != nil {
		t.Fatalf("rendering a continuous run: %v", err)
	}
	if strings.Contains(docxPartText(t, run, "word/document.xml"), "<w:pageBreakBefore/>") {
		t.Error("a continuous run still broke the page before each chapter")
	}
}

func docxPartText(t *testing.T, data []byte, name string) string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("opening the docx: %v", err)
	}
	for _, f := range r.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("opening %s: %v", name, err)
		}
		defer func() { _ = rc.Close() }()
		body, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		return string(body)
	}
	t.Fatalf("%s is not in the docx", name)
	return ""
}
