package export

import (
	"reflect"
	"testing"

	"draftline/internal/types"
)

func TestParseDocumentHTMLPreservesStructureAndMarks(t *testing.T) {
	blocks, err := ParseDocumentHTML(`<h2 style="text-align: center">A &amp; B</h2>
<p><strong>Bold</strong> and <em>italic</em>, <u>underlined</u>, H<sub>2</sub>O and x<sup>2</sup>.<br/>Next line.</p>
<blockquote><p>A quoted <a href="https://example.test">link</a>.</p></blockquote>
<ol><li>First</li><li>Second<ul><li>Nested</li></ul></li></ol>
<hr><p>* * *</p><pre>  exact
  spacing</pre>`)
	if err != nil {
		t.Fatal(err)
	}

	kinds := make([]BlockKind, len(blocks))
	for i, block := range blocks {
		kinds[i] = block.Kind
	}
	wantKinds := []BlockKind{
		BlockHeading, BlockParagraph, BlockBlockquote,
		BlockListItem, BlockListItem, BlockListItem,
		BlockSceneBreak, BlockCode,
	}
	if !reflect.DeepEqual(kinds, wantKinds) {
		t.Fatalf("block kinds = %v, want %v", kinds, wantKinds)
	}
	if blocks[0].Level != 2 || blocks[0].Alignment != "center" || blocks[0].PlainText() != "A & B" {
		t.Fatalf("heading not preserved: %#v", blocks[0])
	}
	if got := blocks[1].PlainText(); got != "Bold and italic, underlined, H2O and x2.\nNext line." {
		t.Fatalf("paragraph text = %q", got)
	}
	if !blocks[1].Runs[0].Bold || !blocks[1].Runs[2].Italic || !blocks[1].Runs[4].Underline {
		t.Fatalf("inline marks not preserved: %#v", blocks[1].Runs)
	}
	if blocks[2].Kind != BlockBlockquote || blocks[2].Runs[1].Href != "https://example.test" {
		t.Fatalf("blockquote link not preserved: %#v", blocks[2])
	}
	if !blocks[3].Ordered || blocks[3].ListNumber != 1 || blocks[3].ListDepth != 1 {
		t.Fatalf("ordered list metadata not preserved: %#v", blocks[3])
	}
	if blocks[5].Ordered || blocks[5].ListDepth != 2 || blocks[5].PlainText() != "Nested" {
		t.Fatalf("nested list metadata not preserved: %#v", blocks[5])
	}
	if got := blocks[7].PlainText(); got != "  exact\n  spacing" {
		t.Fatalf("preformatted whitespace = %q", got)
	}
}

func TestBuildDocumentSelectsSectionsAndRetainsProvenance(t *testing.T) {
	book := types.BookData{
		Metadata:    types.Metadata{Title: "  A Book ", Author: " Author ", Publisher: " Press "},
		Copyright:   "<p>Copyright text</p>",
		FrontMatter: []types.ChapterItem{{ID: "dedication", Title: "Dedication", Content: "<p>For A.</p>"}},
		Body:        []types.ChapterItem{{ID: "chapter-1", Title: "Chapter One", Subtitle: "Arrival", Content: "<p>Body.</p>"}},
		BackMatter:  []types.ChapterItem{{ID: "about", Title: "About", Content: "<p>About.</p>"}},
	}

	doc, err := BuildDocument(book, types.ExportOptions{
		IncludeCopyright:   true,
		IncludeFrontMatter: false,
		IncludeBackMatter:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "A Book" || doc.Author != "Author" || doc.Publisher != "Press" {
		t.Fatalf("metadata not normalized: %#v", doc)
	}
	if len(doc.Sections) != 3 {
		t.Fatalf("sections = %d, want copyright + body + back matter", len(doc.Sections))
	}
	if doc.Sections[0].Role != SectionCopyright || doc.Sections[1].Role != SectionBody || doc.Sections[2].Role != SectionBack {
		t.Fatalf("section roles = %q, %q, %q", doc.Sections[0].Role, doc.Sections[1].Role, doc.Sections[2].Role)
	}
	if doc.Sections[1].ID != "chapter-1" || doc.Sections[1].SourceIndex != 0 || doc.Sections[1].Subtitle != "Arrival" {
		t.Fatalf("body provenance not retained: %#v", doc.Sections[1])
	}
}

func TestParseDocumentHTMLRetainsTextInsideUnknownContainers(t *testing.T) {
	blocks, err := ParseDocumentHTML(`<section><article><p>Text inside future editor markup.</p></article></section>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || blocks[0].PlainText() != "Text inside future editor markup." {
		t.Fatalf("unknown container lost content: %#v", blocks)
	}
}
