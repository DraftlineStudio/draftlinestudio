package export

import (
	"fmt"
	"strings"
	"unicode"

	"draftline/internal/types"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Document is the renderer-neutral representation of a selected book edition.
// Exporters consume this model instead of independently flattening TipTap HTML.
// Source IDs and section roles survive conversion so layout and navigation can
// be traced back to the manuscript without interpreting presentation markup.
type Document struct {
	Title    string
	Subtitle string
	Author   string
	// Publisher is the imprint the book appears under when the book names
	// one, and the publisher otherwise.
	Publisher string
	// Language is a BCP 47 tag; never empty by the time a renderer sees it.
	Language string
	// Catalogue fields a package format can declare. Each is empty when the
	// book does not carry it, and a renderer must leave the element out
	// rather than emit an empty one.
	SeriesName   string
	SeriesNumber string
	Description  string
	Subjects     []string
	Contributors string
	// Edition is the registered format this export was made against, or nil
	// when the author exported from scratch. When it is set, the identifier,
	// the imprint, the rights line, the date and the copyright page all come
	// from it rather than from the book, so that everything in the file agrees
	// with the ISBN printed on its cover.
	Edition *DocumentEdition
	// EPUBProfile is the package version an export made from no edition
	// asked for. An edition export leaves it nil and the record decides.
	EPUBProfile *epubProfile
	Sections    []DocumentSection
}

// SectionRole describes where a section came from in the Draftline archive.
type SectionRole string

const (
	SectionCopyright SectionRole = "copyright"
	SectionFront     SectionRole = "front-matter"
	SectionBody      SectionRole = "body"
	SectionBack      SectionRole = "back-matter"
)

// DocumentSection is one authored or generated section in reading order.
type DocumentSection struct {
	ID          string
	Title       string
	Subtitle    string
	Role        SectionRole
	SourceIndex int
	Blocks      []DocumentBlock
}

// BlockKind is a semantic block retained from the editor document.
type BlockKind string

const (
	BlockParagraph  BlockKind = "paragraph"
	BlockHeading    BlockKind = "heading"
	BlockBlockquote BlockKind = "blockquote"
	BlockListItem   BlockKind = "list-item"
	BlockCode       BlockKind = "code"
	BlockSceneBreak BlockKind = "scene-break"
)

// DocumentBlock preserves structure that affects readable book layout.
type DocumentBlock struct {
	Kind       BlockKind
	Level      int
	Alignment  string
	Ordered    bool
	ListDepth  int
	ListNumber int
	Runs       []DocumentRun
}

// DocumentRun is an inline text span with the marks supported by Draftline's
// editor. LineBreak is a semantic <br>, not an exporter-invented wrap.
type DocumentRun struct {
	Text        string
	Bold        bool
	Italic      bool
	Underline   bool
	Strike      bool
	Superscript bool
	Subscript   bool
	Code        bool
	Href        string
	LineBreak   bool
}

// PlainText returns the authored text of a block without discarding explicit
// line breaks. It is useful to renderers that cannot represent inline marks.
func (b DocumentBlock) PlainText() string {
	var out strings.Builder
	for _, run := range b.Runs {
		if run.LineBreak {
			out.WriteByte('\n')
		} else {
			out.WriteString(run.Text)
		}
	}
	return out.String()
}

// BuildDocument selects the requested archive sections and parses their
// documentSubjects are the book's subject headings, in the order the author
// listed them, with blanks left out.
func documentSubjects(m types.Metadata) []string {
	out := []string{}
	for _, value := range []string{m.BISAC1, m.BISAC2} {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// documentPublisher is the imprint a book appears under when it names one,
// and the publisher that owns the imprint otherwise. A title page prints the
// imprint; the publisher is the company behind it.
func documentPublisher(m types.Metadata) string {
	if imprint := strings.TrimSpace(m.Imprint); imprint != "" {
		return imprint
	}
	return strings.TrimSpace(m.Publisher)
}

// documentLanguage is the book's language tag, or "en" when the book carries
// none. Every book written before the field existed is English by default,
// which is what the exporters declared unconditionally until now.
func documentLanguage(tag string) string {
	if trimmed := strings.TrimSpace(tag); trimmed != "" {
		return trimmed
	}
	return "en"
}

// editor HTML into a single renderer-neutral document.
func BuildDocument(book types.BookData, options types.ExportOptions) (Document, error) {
	doc := Document{
		Title:        strings.TrimSpace(book.Metadata.Title),
		Subtitle:     strings.TrimSpace(book.Metadata.Subtitle),
		Author:       strings.TrimSpace(book.Metadata.Author),
		Publisher:    documentPublisher(book.Metadata),
		Language:     documentLanguage(book.Metadata.Language),
		SeriesName:   strings.TrimSpace(book.Metadata.SeriesName),
		SeriesNumber: strings.TrimSpace(book.Metadata.SeriesNumber),
		Description:  strings.TrimSpace(book.Metadata.ShortDescription),
		Subjects:     documentSubjects(book.Metadata),
		Contributors: strings.TrimSpace(book.Metadata.Contributors),
		Edition:      selectEdition(book, options),
	}
	// A format's own imprint of record outranks the book's, because that is
	// what the copyright page of this particular object says.
	if doc.Edition != nil && doc.Edition.Imprint != "" {
		doc.Publisher = doc.Edition.Imprint
	}

	appendSection := func(ch types.ChapterItem, role SectionRole, sourceIndex int) error {
		blocks, err := ParseDocumentHTML(ch.Content)
		if err != nil {
			return fmt.Errorf("parse %s %q: %w", role, ch.Title, err)
		}
		doc.Sections = append(doc.Sections, DocumentSection{
			ID:          ch.ID,
			Title:       strings.TrimSpace(ch.Title),
			Subtitle:    strings.TrimSpace(ch.Subtitle),
			Role:        role,
			SourceIndex: sourceIndex,
			Blocks:      blocks,
		})
		return nil
	}

	if options.IncludeCopyright {
		blocks, err := copyrightSectionBlocks(doc.Edition, book.Copyright)
		if err != nil {
			return Document{}, err
		}
		if len(blocks) > 0 {
			doc.Sections = append(doc.Sections, DocumentSection{
				ID: "copyright", Title: "Copyright", Role: SectionCopyright, Blocks: blocks,
			})
		}
	}
	if options.IncludeFrontMatter {
		for i, ch := range book.FrontMatter {
			if err := appendSection(ch, SectionFront, i); err != nil {
				return Document{}, err
			}
		}
	}
	for i, ch := range book.Body {
		if err := appendSection(ch, SectionBody, i); err != nil {
			return Document{}, err
		}
	}
	if options.IncludeBackMatter {
		for i, ch := range book.BackMatter {
			if err := appendSection(ch, SectionBack, i); err != nil {
				return Document{}, err
			}
		}
	}

	return doc, nil
}

// copyrightSectionBlocks builds the copyright page of the exported book.
//
// Exporting from a registered edition generates it: the title, the cumulative
// copyright years, the rights line, the edition and its month, the imprint and
// the ISBN of the format in hand all live in the record already, and typing
// them a second time is how a book ends up with the paperback's ISBN on the
// ebook's copyright page.
//
// Whatever the author wrote on their own copyright page then follows
// underneath, because a copyright page carries more than the notice —
// permissions, a designer's credit, a disclaimer — and an export that silently
// dropped it would be throwing away work. An export that names no edition is
// the author's page alone, exactly as before.
func copyrightSectionBlocks(edition *DocumentEdition, authored string) ([]DocumentBlock, error) {
	var blocks []DocumentBlock
	if edition != nil {
		blocks = append(blocks, generatedCopyrightBlocks(edition.Copyright)...)
	}
	if strings.TrimSpace(authored) == "" {
		return blocks, nil
	}
	parsed, err := ParseDocumentHTML(authored)
	if err != nil {
		return nil, fmt.Errorf("parse copyright page: %w", err)
	}
	return append(blocks, parsed...), nil
}

// generatedCopyrightBlocks turns the generated lines into paragraphs. The
// blank line the generator uses to separate the notice from the publication
// details is not a paragraph and is left out; paragraph spacing does that job
// in every renderer Draftline has.
func generatedCopyrightBlocks(lines []string) []DocumentBlock {
	blocks := make([]DocumentBlock, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		blocks = append(blocks, DocumentBlock{
			Kind: BlockParagraph,
			Runs: []DocumentRun{{Text: line}},
		})
	}
	return blocks
}

// ParseDocumentHTML converts a TipTap HTML fragment to semantic export blocks.
// Unknown container elements are traversed so their text is retained.
func ParseDocumentHTML(fragment string) ([]DocumentBlock, error) {
	root := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := html.ParseFragment(strings.NewReader(fragment), root)
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		root.AppendChild(node)
	}

	parser := documentParser{}
	parser.walkBlocks(root, blockContext{})
	return parser.blocks, nil
}

type blockContext struct {
	blockquote bool
	ordered    bool
	listDepth  int
	listNumber int
}

type documentParser struct {
	blocks []DocumentBlock
}

func (p *documentParser) walkBlocks(node *html.Node, ctx blockContext) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}

		tag := strings.ToLower(child.Data)
		switch tag {
		case "p", "div":
			kind := BlockParagraph
			if ctx.blockquote {
				kind = BlockBlockquote
			}
			p.appendTextBlock(child, kind, 0, ctx)
		case "h1", "h2", "h3", "h4", "h5", "h6":
			p.appendTextBlock(child, BlockHeading, int(tag[1]-'0'), ctx)
		case "blockquote":
			next := ctx
			next.blockquote = true
			p.walkBlocks(child, next)
		case "ul", "ol":
			next := ctx
			next.ordered = tag == "ol"
			next.listDepth++
			next.listNumber = 0
			p.walkList(child, next)
		case "li":
			next := ctx
			next.listNumber++
			p.appendTextBlock(child, BlockListItem, 0, next)
		case "pre":
			p.appendTextBlock(child, BlockCode, 0, ctx)
		case "hr":
			p.blocks = append(p.blocks, DocumentBlock{Kind: BlockSceneBreak})
		default:
			p.walkBlocks(child, ctx)
		}
	}
}

func (p *documentParser) walkList(list *html.Node, ctx blockContext) {
	number := 0
	for child := list.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode || strings.ToLower(child.Data) != "li" {
			continue
		}
		number++
		itemCtx := ctx
		itemCtx.listNumber = number
		p.appendTextBlock(child, BlockListItem, 0, itemCtx)
		for nested := child.FirstChild; nested != nil; nested = nested.NextSibling {
			if nested.Type != html.ElementNode {
				continue
			}
			tag := strings.ToLower(nested.Data)
			if tag == "ul" || tag == "ol" {
				next := ctx
				next.ordered = tag == "ol"
				next.listDepth++
				p.walkList(nested, next)
			}
		}
	}
}

func (p *documentParser) appendTextBlock(node *html.Node, kind BlockKind, level int, ctx blockContext) {
	block := DocumentBlock{
		Kind:       kind,
		Level:      level,
		Alignment:  nodeAlignment(node),
		Ordered:    ctx.ordered,
		ListDepth:  ctx.listDepth,
		ListNumber: ctx.listNumber,
	}
	collectRuns(node, runMarks{}, &block.Runs, kind == BlockCode, true)
	block.Runs = normalizeRuns(block.Runs, kind == BlockCode)

	if kind == BlockParagraph && separatorText(block.PlainText()) {
		block = DocumentBlock{Kind: BlockSceneBreak}
	}
	if kind == BlockSceneBreak || strings.TrimSpace(block.PlainText()) != "" || hasLineBreak(block.Runs) {
		p.blocks = append(p.blocks, block)
	}
}

type runMarks struct {
	bold, italic, underline, strike bool
	superscript, subscript, code    bool
	href                            string
}

func collectRuns(node *html.Node, marks runMarks, runs *[]DocumentRun, preserveSpace, skipNestedLists bool) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.TextNode {
			text := child.Data
			if !preserveSpace {
				text = collapseHTMLSpace(text)
			}
			if text != "" {
				*runs = append(*runs, DocumentRun{
					Text: text, Bold: marks.bold, Italic: marks.italic,
					Underline: marks.underline, Strike: marks.strike,
					Superscript: marks.superscript, Subscript: marks.subscript,
					Code: marks.code, Href: marks.href,
				})
			}
			continue
		}
		if child.Type != html.ElementNode {
			continue
		}

		tag := strings.ToLower(child.Data)
		if skipNestedLists && (tag == "ul" || tag == "ol") {
			continue
		}
		if tag == "br" {
			*runs = append(*runs, DocumentRun{LineBreak: true})
			continue
		}

		next := marks
		switch tag {
		case "strong", "b":
			next.bold = true
		case "em", "i":
			next.italic = true
		case "u":
			next.underline = true
		case "s", "strike", "del":
			next.strike = true
		case "sup":
			next.superscript = true
		case "sub":
			next.subscript = true
		case "code":
			next.code = true
		case "a":
			next.href = attribute(child, "href")
		case "span":
			applyInlineStyle(attribute(child, "style"), &next)
		}
		collectRuns(child, next, runs, preserveSpace || tag == "pre", skipNestedLists)
	}
}

func normalizeRuns(runs []DocumentRun, preserveSpace bool) []DocumentRun {
	out := make([]DocumentRun, 0, len(runs))
	for _, run := range runs {
		if !preserveSpace && !run.LineBreak {
			run.Text = strings.ReplaceAll(run.Text, "\u00a0", " ")
		}
		if run.Text == "" && !run.LineBreak {
			continue
		}
		if len(out) > 0 && sameRunStyle(out[len(out)-1], run) && !run.LineBreak && !out[len(out)-1].LineBreak {
			out[len(out)-1].Text += run.Text
			continue
		}
		out = append(out, run)
	}
	if !preserveSpace && len(out) > 0 {
		out[0].Text = strings.TrimLeftFunc(out[0].Text, unicode.IsSpace)
		last := len(out) - 1
		out[last].Text = strings.TrimRightFunc(out[last].Text, unicode.IsSpace)
		if out[0].Text == "" && !out[0].LineBreak {
			out = out[1:]
		}
		if len(out) > 0 && out[len(out)-1].Text == "" && !out[len(out)-1].LineBreak {
			out = out[:len(out)-1]
		}
	}
	return out
}

func sameRunStyle(a, b DocumentRun) bool {
	return a.Bold == b.Bold && a.Italic == b.Italic && a.Underline == b.Underline &&
		a.Strike == b.Strike && a.Superscript == b.Superscript &&
		a.Subscript == b.Subscript && a.Code == b.Code && a.Href == b.Href
}

func collapseHTMLSpace(value string) string {
	value = strings.ReplaceAll(value, "\u00a0", " ")
	var out strings.Builder
	space := false
	for _, r := range value {
		if unicode.IsSpace(r) {
			if !space {
				out.WriteByte(' ')
				space = true
			}
			continue
		}
		space = false
		out.WriteRune(r)
	}
	return out.String()
}

func hasLineBreak(runs []DocumentRun) bool {
	for _, run := range runs {
		if run.LineBreak {
			return true
		}
	}
	return false
}

func separatorText(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	count := 0
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		switch r {
		case '*', '#', '~', '_', '-', '\u2022', '\u00b7', '\u203b', '\u2042':
			count++
		default:
			return false
		}
	}
	return count >= 3
}

func nodeAlignment(node *html.Node) string {
	value := strings.ToLower(strings.TrimSpace(attribute(node, "align")))
	if value == "left" || value == "center" || value == "right" || value == "justify" {
		return value
	}
	for _, declaration := range strings.Split(attribute(node, "style"), ";") {
		pair := strings.SplitN(declaration, ":", 2)
		if len(pair) != 2 || strings.TrimSpace(strings.ToLower(pair[0])) != "text-align" {
			continue
		}
		value = strings.ToLower(strings.TrimSpace(pair[1]))
		if value == "left" || value == "center" || value == "right" || value == "justify" {
			return value
		}
	}
	return ""
}

func applyInlineStyle(style string, marks *runMarks) {
	for _, declaration := range strings.Split(style, ";") {
		pair := strings.SplitN(declaration, ":", 2)
		if len(pair) != 2 {
			continue
		}
		name := strings.TrimSpace(strings.ToLower(pair[0]))
		value := strings.TrimSpace(strings.ToLower(pair[1]))
		switch name {
		case "font-weight":
			marks.bold = value == "bold" || value == "bolder" || value >= "600"
		case "font-style":
			marks.italic = value == "italic" || value == "oblique"
		case "text-decoration", "text-decoration-line":
			marks.underline = marks.underline || strings.Contains(value, "underline")
			marks.strike = marks.strike || strings.Contains(value, "line-through")
		case "vertical-align":
			marks.superscript = value == "super"
			marks.subscript = value == "sub"
		}
	}
}

func attribute(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}
