package main

// EPUB spine-document sanitizer. Parses real (often malformed) XHTML with the
// tolerant x/net/html parser and re-emits only the editor's dialect: p, h1-h3,
// blockquote, ul/ol/li, pre>code, hr, br, strong/em/u/s/sub/sup/code, and
// text-align styles on paragraphs and headings. Everything else is unwrapped
// to its text or dropped (scripts, styles, images, navigation). TipTap would
// otherwise discard the unknown markup silently on first load and make the
// loss permanent on the next save.

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"regexp"
	"strings"
	"unicode/utf8"

	xhtml "golang.org/x/net/html"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"

	"draftline/internal/types"
)

// importedBlock is one top-level block of sanitized chapter content.
type importedBlock struct {
	level int    // 1-3 for headings (after h4-h6 → h3 mapping); 0 otherwise
	html  string // sanitized markup, e.g. "<p>…</p>", "<h2>…</h2>", "<hr>"
	text  string // trimmed plain text ("" for <hr>)
}

// ── Encoding ────────────────────────────────────────────────────────────────

var declaredEncodingRe = regexp.MustCompile(`(?i)(?:encoding=["']([A-Za-z0-9._-]+)["']|charset=["']?([A-Za-z0-9._-]+))`)

// decodeToUTF8 normalizes raw spine-document bytes to valid UTF-8: BOM sniff,
// then declared xml/meta encoding, then UTF-8 validity, then a Windows-1252
// fallback (a superset of Latin-1 that never fails and salvages mojibake).
func decodeToUTF8(raw []byte) []byte {
	switch {
	case bytes.HasPrefix(raw, []byte{0xFF, 0xFE}):
		return decodeWith(unicode.UTF16(unicode.LittleEndian, unicode.UseBOM), raw)
	case bytes.HasPrefix(raw, []byte{0xFE, 0xFF}):
		return decodeWith(unicode.UTF16(unicode.BigEndian, unicode.UseBOM), raw)
	case bytes.HasPrefix(raw, []byte{0xEF, 0xBB, 0xBF}):
		return raw[3:]
	}

	head := raw
	if len(head) > 1024 {
		head = head[:1024]
	}
	if m := declaredEncodingRe.FindSubmatch(head); m != nil {
		name := string(m[1])
		if name == "" {
			name = string(m[2])
		}
		switch strings.ToLower(name) {
		case "iso-8859-1", "iso8859-1", "latin-1", "latin1", "windows-1252", "cp-1252", "cp1252":
			return decodeWith(charmap.Windows1252, raw)
		case "utf-16", "utf-16le":
			return decodeWith(unicode.UTF16(unicode.LittleEndian, unicode.UseBOM), raw)
		case "utf-16be":
			return decodeWith(unicode.UTF16(unicode.BigEndian, unicode.UseBOM), raw)
		}
	}

	if utf8.Valid(raw) {
		return raw
	}
	return decodeWith(charmap.Windows1252, raw)
}

func decodeWith(enc encoding.Encoding, raw []byte) []byte {
	decoded, err := enc.NewDecoder().Bytes(raw)
	if err != nil {
		return raw
	}
	return decoded
}

// ── Sanitizer ───────────────────────────────────────────────────────────────

// Elements removed together with their entire subtree.
var droppedElements = map[string]bool{
	"script": true, "style": true, "head": true, "title": true, "nav": true,
	"svg": true, "img": true, "image": true, "picture": true, "source": true,
	"video": true, "audio": true, "iframe": true, "object": true, "embed": true,
	"canvas": true, "map": true, "template": true, "link": true, "meta": true,
	"base": true, "noscript": true,
}

// Inline elements kept as editor marks (source tag → emitted tag).
var inlineMarkFor = map[string]string{
	"strong": "strong", "b": "strong",
	"em": "em", "i": "em", "cite": "em", "q": "em",
	"u": "u", "s": "s", "strike": "s", "del": "s",
	"sub": "sub", "sup": "sup", "code": "code",
}

// Block containers that are transparent: they contribute paragraph boundaries
// but no markup of their own.
var transparentContainers = map[string]bool{
	"div": true, "section": true, "article": true, "aside": true, "main": true,
	"header": true, "footer": true, "figure": true, "figcaption": true,
	"center": true, "dl": true, "dt": true, "dd": true, "table": true,
	"tbody": true, "thead": true, "tfoot": true, "tr": true, "form": true,
	"details": true, "summary": true, "body": true, "html": true,
	"address": true, "caption": true, "hgroup": true, "colgroup": true,
}

var wsRunRe = regexp.MustCompile(`\s+`)
var textAlignRe = regexp.MustCompile(`(?i)text-align\s*:\s*(left|right|center|justify)`)

type sanitizer struct {
	blocks        []importedBlock
	para          strings.Builder
	paraText      strings.Builder
	imagesDropped int
}

// parseSpineDoc decodes, parses, and sanitizes one spine document.
// The doc title prefers the first h1/h2 block over the <title> element, which
// EPUBs commonly repeat verbatim across every file.
func parseSpineDoc(raw []byte) (title string, blocks []importedBlock, imagesDropped int, err error) {
	doc, parseErr := xhtml.Parse(bytes.NewReader(decodeToUTF8(raw)))
	if parseErr != nil {
		return "", nil, 0, fmt.Errorf("failed to parse document: %v", parseErr)
	}

	bodyNode := findElement(doc, "body")
	titleNode := findElement(doc, "title")

	s := &sanitizer{}
	if bodyNode != nil {
		for child := bodyNode.FirstChild; child != nil; child = child.NextSibling {
			s.walkBlock(child)
		}
	}
	s.flushPara()

	for _, b := range s.blocks {
		if (b.level == 1 || b.level == 2) && b.text != "" {
			title = b.text
			break
		}
	}
	if title == "" && titleNode != nil {
		title = collapseWS(nodeText(titleNode, false))
		title = strings.TrimSpace(title)
	}

	return title, s.blocks, s.imagesDropped, nil
}

func findElement(n *xhtml.Node, tag string) *xhtml.Node {
	if n.Type == xhtml.ElementNode && n.Data == tag {
		return n
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := findElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}

// walkBlock processes one node in block context.
func (s *sanitizer) walkBlock(n *xhtml.Node) {
	switch n.Type {
	case xhtml.TextNode:
		s.appendInlineText(&s.para, n.Data)
	case xhtml.ElementNode:
		tag := n.Data
		switch {
		case droppedElements[tag]:
			s.countDropped(n)
		case tag == "h1" || tag == "h2" || tag == "h3" || tag == "h4" || tag == "h5" || tag == "h6":
			s.flushPara()
			level := min(int(tag[1]-'0'), 3)
			inner, text := s.renderInlineChildren(n)
			if text != "" {
				s.blocks = append(s.blocks, importedBlock{
					level: level,
					html:  fmt.Sprintf("<h%d%s>%s</h%d>", level, alignAttr(n), inner, level),
					text:  text,
				})
			}
		case tag == "p":
			s.flushPara()
			inner, text := s.renderInlineChildren(n)
			if text != "" {
				s.blocks = append(s.blocks, importedBlock{
					html: "<p" + alignAttr(n) + ">" + inner + "</p>",
					text: text,
				})
			}
		case tag == "blockquote":
			s.flushPara()
			nested := &sanitizer{}
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				nested.walkBlock(child)
			}
			nested.flushPara()
			s.imagesDropped += nested.imagesDropped
			if len(nested.blocks) > 0 {
				var htmlParts, textParts []string
				for _, b := range nested.blocks {
					htmlParts = append(htmlParts, b.html)
					if b.text != "" {
						textParts = append(textParts, b.text)
					}
				}
				s.blocks = append(s.blocks, importedBlock{
					html: "<blockquote>" + strings.Join(htmlParts, "") + "</blockquote>",
					text: strings.Join(textParts, " "),
				})
			}
		case tag == "ul" || tag == "ol":
			s.flushPara()
			listHTML, listText := s.renderList(n)
			if strings.TrimSpace(listText) != "" {
				s.blocks = append(s.blocks, importedBlock{html: listHTML, text: strings.TrimSpace(listText)})
			}
		case tag == "hr":
			s.flushPara()
			s.blocks = append(s.blocks, importedBlock{html: "<hr>"})
		case tag == "pre":
			s.flushPara()
			text := nodeText(n, true)
			if strings.TrimSpace(text) != "" {
				s.blocks = append(s.blocks, importedBlock{
					html: "<pre><code>" + stdhtml.EscapeString(text) + "</code></pre>",
					text: strings.TrimSpace(collapseWS(text)),
				})
			}
		case tag == "td" || tag == "th" || tag == "li":
			// Stray cells/items outside their list/table context.
			s.flushPara()
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				s.walkBlock(child)
			}
			s.flushPara()
		case tag == "br":
			s.para.WriteString("<br>")
		case transparentContainers[tag]:
			s.flushPara()
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				s.walkBlock(child)
			}
			s.flushPara()
		default:
			// Inline element (a, span, em, font, …) in block context: render
			// into the running paragraph.
			s.renderInlineNode(n, &s.para, &s.paraText)
		}
	default:
		// Comments, doctypes, and any future NodeType are dropped on
		// purpose — a sanitizer keeps only text and known elements.
	}
}

// renderList renders a ul/ol with its li children; nested lists recurse.
func (s *sanitizer) renderList(n *xhtml.Node) (string, string) {
	var sb, tb strings.Builder
	sb.WriteString("<" + n.Data + ">")
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != xhtml.ElementNode || child.Data != "li" {
			continue
		}
		sb.WriteString("<li>")
		var itemHTML, itemText strings.Builder
		for grand := child.FirstChild; grand != nil; grand = grand.NextSibling {
			if grand.Type == xhtml.ElementNode && (grand.Data == "ul" || grand.Data == "ol") {
				nestedHTML, nestedText := s.renderList(grand)
				itemHTML.WriteString(nestedHTML)
				itemText.WriteString(" " + nestedText)
				continue
			}
			s.renderInlineNode(grand, &itemHTML, &itemText)
		}
		sb.WriteString(strings.TrimSpace(itemHTML.String()))
		sb.WriteString("</li>")
		tb.WriteString(" " + itemText.String())
	}
	sb.WriteString("</" + n.Data + ">")
	return sb.String(), strings.TrimSpace(collapseWS(tb.String()))
}

// renderInlineChildren renders an element's children in inline context and
// returns (html, trimmed plain text).
func (s *sanitizer) renderInlineChildren(n *xhtml.Node) (string, string) {
	var sb, tb strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		s.renderInlineNode(child, &sb, &tb)
	}
	return strings.TrimSpace(sb.String()), strings.TrimSpace(collapseWS(tb.String()))
}

// renderInlineNode renders one node in inline context into sb (markup) and
// tb (plain text).
func (s *sanitizer) renderInlineNode(n *xhtml.Node, sb, tb *strings.Builder) {
	switch n.Type {
	case xhtml.TextNode:
		s.appendInlineTextTo(sb, tb, n.Data)
	case xhtml.ElementNode:
		tag := n.Data
		if droppedElements[tag] {
			s.countDropped(n)
			return
		}
		if tag == "br" {
			sb.WriteString("<br>")
			return
		}
		if mark, ok := inlineMarkFor[tag]; ok {
			var innerSB, innerTB strings.Builder
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				s.renderInlineNode(child, &innerSB, &innerTB)
			}
			inner := innerSB.String()
			if strings.TrimSpace(collapseWS(innerTB.String())) == "" && !strings.Contains(inner, "<br>") {
				// Keep whitespace-only runs so word boundaries survive.
				sb.WriteString(inner)
				tb.WriteString(innerTB.String())
				return
			}
			sb.WriteString("<" + mark + ">" + inner + "</" + mark + ">")
			tb.WriteString(innerTB.String())
			return
		}
		// Everything else (a, span, font, small, abbr, unknown) unwraps.
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			s.renderInlineNode(child, sb, tb)
		}
	default:
		// Comments, doctypes, and any future NodeType are dropped on
		// purpose — a sanitizer keeps only text and known elements.
	}
}

// appendInlineText appends collapsed text into the running paragraph buffers.
func (s *sanitizer) appendInlineText(_ *strings.Builder, text string) {
	s.appendInlineTextTo(&s.para, &s.paraText, text)
}

func (s *sanitizer) appendInlineTextTo(sb, tb *strings.Builder, text string) {
	collapsed := collapseWS(text)
	if collapsed == "" {
		return
	}
	if collapsed == " " && (sb.Len() == 0 || strings.HasSuffix(sb.String(), " ")) {
		return
	}
	sb.WriteString(stdhtml.EscapeString(collapsed))
	tb.WriteString(collapsed)
}

func (s *sanitizer) flushPara() {
	htmlContent := strings.TrimSpace(s.para.String())
	text := strings.TrimSpace(collapseWS(s.paraText.String()))
	s.para.Reset()
	s.paraText.Reset()
	if text == "" {
		return
	}
	s.blocks = append(s.blocks, importedBlock{html: "<p>" + htmlContent + "</p>", text: text})
}

// countDropped tallies dropped visual content so the import can warn about it.
func (s *sanitizer) countDropped(n *xhtml.Node) {
	switch n.Data {
	case "img", "image", "picture", "svg":
		s.imagesDropped++
	}
}

// alignAttr preserves a source text-align style (the only style the editor's
// TextAlign extension understands) on paragraphs and headings.
func alignAttr(n *xhtml.Node) string {
	for _, attr := range n.Attr {
		if attr.Key != "style" {
			continue
		}
		if m := textAlignRe.FindStringSubmatch(attr.Val); m != nil {
			return ` style="text-align: ` + strings.ToLower(m[1]) + `"`
		}
	}
	return ""
}

// nodeText returns the concatenated text of a subtree. When raw is true,
// whitespace is preserved exactly (for <pre>).
func nodeText(n *xhtml.Node, raw bool) string {
	var sb strings.Builder
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.TextNode {
			sb.WriteString(node.Data)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		walk(child)
	}
	if raw {
		return sb.String()
	}
	return sb.String()
}

// collapseWS reduces every whitespace run to a single space, preserving
// leading/trailing boundary spaces.
func collapseWS(s string) string {
	return wsRunRe.ReplaceAllString(s, " ")
}

// joinBlocks renders sanitized blocks back to chapter HTML.
func joinBlocks(blocks []importedBlock) string {
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		parts = append(parts, b.html)
	}
	return strings.Join(parts, "\n")
}

// ── Chaptering ──────────────────────────────────────────────────────────────

const (
	// maxImportedChapterHTML truncates a single chapter's sanitized HTML so no
	// chapter can stall TipTap's synchronous setContent.
	maxImportedChapterHTML = 2 << 20
	// maxImportedBookHTML caps the whole imported book crossing the bridge.
	maxImportedBookHTML = 40 << 20
	// maxImportedChapters caps chapter explosion from pathological documents.
	maxImportedChapters = 500
)

// assembleChapters converts one spine document's sanitized blocks into
// chapters. A spine document is one chapter by default; it splits only at
// multiple h1s, or at multiple h2s when it has no h1 — books that use h2/h3
// for part titles and scene headings keep them inline instead of being
// shattered into bogus chapters. The chapter-opening heading becomes the
// title and is removed from content: the app renders titles itself and the
// EPUB exporter re-adds an <h1>, so leaving it in doubles the title on every
// round trip.
func assembleChapters(docTitle string, blocks []importedBlock) ([]types.ChapterItem, []string) {
	h1s, h2s := 0, 0
	for _, b := range blocks {
		switch b.level {
		case 1:
			h1s++
		case 2:
			h2s++
		}
	}
	splitLevel := 0
	if h1s >= 2 {
		splitLevel = 1
	} else if h1s == 0 && h2s >= 2 {
		splitLevel = 2
	}

	var chapters []types.ChapterItem
	var warnings []string
	emit := func(title string, content []importedBlock) {
		html, truncated := renderChapterHTML(content)
		if truncated {
			warnings = append(warnings, fmt.Sprintf("Chapter %q was truncated during import (over %d MB)", title, maxImportedChapterHTML>>20))
		}
		chapters = append(chapters, types.ChapterItem{Title: title, Content: html})
	}

	if splitLevel == 0 {
		title := docTitle
		content := blocks
		// A document-leading h1/h2 is the chapter title; a leading h3 is a
		// scene heading and stays in the content.
		if len(blocks) > 0 && (blocks[0].level == 1 || blocks[0].level == 2) {
			title = blocks[0].text
			content = blocks[1:]
		}
		emit(title, content)
		return chapters, warnings
	}

	// Blocks before the first split heading stay accumulated so they land at
	// the top of the first chapter instead of becoming a spurious chapter.
	var current []importedBlock
	currentTitle := docTitle
	started := false
	for _, b := range blocks {
		if b.level == splitLevel {
			if started {
				emit(currentTitle, current)
				current = nil
			}
			started = true
			currentTitle = b.text
			continue
		}
		current = append(current, b)
	}
	emit(currentTitle, current)
	return chapters, warnings
}

// renderChapterHTML joins blocks into chapter HTML, truncating at a block
// boundary once the per-chapter cap is exceeded.
func renderChapterHTML(blocks []importedBlock) (string, bool) {
	var sb strings.Builder
	truncated := false
	for _, b := range blocks {
		if sb.Len()+len(b.html) > maxImportedChapterHTML {
			truncated = true
			break
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(b.html)
	}
	if truncated {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("<p>[Content truncated on import]</p>")
	}
	if sb.Len() == 0 {
		return "<p></p>", truncated
	}
	return sb.String(), truncated
}
