package export

import (
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
	data, err := renderDOCX(doc)
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
