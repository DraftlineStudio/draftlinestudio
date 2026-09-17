package export

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"codeberg.org/go-pdf/fpdf"
)

type pdfPageKind string

const (
	pageCover     pdfPageKind = "cover"
	pageTitle     pdfPageKind = "title"
	pageHalfTitle pdfPageKind = "half-title"
	pageTOC       pdfPageKind = "toc"
	pageSection   pdfPageKind = "section"
	pageBlank     pdfPageKind = "blank"
)

type pdfTOCEntry struct {
	Title string
	Page  int
}

type publicationPDFRenderer struct {
	pdf        *fpdf.Fpdf
	doc        Document
	spec       publicationPDFSpec
	pageWidth  float64
	pageHeight float64
	trimX      float64
	trimY      float64
	y          float64
	chapter    string
	// paragraphNumber counts paragraphs for a narration script, so a retake
	// can be asked for by number instead of by reading the line back.
	paragraphNumber int
	pageKind        pdfPageKind
	tocPages        []int
	toc             []pdfTOCEntry
}

func renderPublicationPDF(doc Document, spec publicationPDFSpec) ([]byte, error) {
	renderer := newPublicationPDFRenderer(doc, spec)
	if err := renderer.render(); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := renderer.pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func newPublicationPDFRenderer(doc Document, spec publicationPDFSpec) *publicationPDFRenderer {
	if spec.HeadingFont.ID == "" {
		spec.HeadingFont = spec.Font
	}
	if spec.FurnitureFont.ID == "" {
		spec.FurnitureFont = spec.Font
	}
	if spec.TitlePageFont.ID == "" {
		spec.TitlePageFont = spec.Font
	}
	if spec.CodeFont.ID == "" {
		spec.CodeFont = embeddedPDFFonts["ibmplexmono"]
	}
	markMargin := 0.0
	if spec.CropMarks {
		markMargin = 18
	}
	outside := spec.Bleed + markMargin
	pageWidth := spec.TrimWidth + 2*outside
	pageHeight := spec.TrimHeight + 2*outside
	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "pt",
		Size:           fpdf.SizeType{Wd: pageWidth, Ht: pageHeight},
	})
	fonts := []embeddedFontFamily{spec.Font, spec.HeadingFont, spec.FurnitureFont, spec.TitlePageFont}
	if documentUsesCode(doc) {
		fonts = append(fonts, spec.CodeFont)
	}
	registerPDFFonts(pdf, fonts...)
	pdf.SetTitle(doc.Title, true)
	pdf.SetAuthor(doc.Author, true)
	pdf.SetCreator("Draftline", true)
	pdf.SetProducer("Draftline", true)
	pdf.SetLang(doc.Language)
	pdf.SetDisplayMode("fullwidth", "continuous")
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetCompression(true)
	pdf.SetPageBox("trim", outside, outside, spec.TrimWidth, spec.TrimHeight)
	pdf.SetPageBox("crop", outside, outside, spec.TrimWidth, spec.TrimHeight)
	if spec.Bleed > 0 {
		pdf.SetPageBox("bleed", markMargin, markMargin, spec.TrimWidth+2*spec.Bleed, spec.TrimHeight+2*spec.Bleed)
	} else {
		pdf.SetPageBox("bleed", outside, outside, spec.TrimWidth, spec.TrimHeight)
	}
	return &publicationPDFRenderer{
		pdf: pdf, doc: doc, spec: spec,
		pageWidth: pageWidth, pageHeight: pageHeight,
		trimX: outside, trimY: outside,
	}
}

func (r *publicationPDFRenderer) render() error {
	r.renderCoverPage()
	if r.spec.GenerateHalfTitle {
		r.addPage(pageHalfTitle, "")
		r.centeredTextWithFont(r.doc.Title, r.spec.TitlePageFont, r.spec.FontSize*1.7, "B", r.trimY+r.spec.TrimHeight*0.42)
	}
	// The title page belongs on a recto. Trade convention runs half title,
	// blank, title page, copyright — so the half title opens the book on page
	// one, its verso is blank, and the title page is the first thing the
	// reader meets on opening the book flat. Draftline used to put the title
	// page straight after the half title, which lands it on the back of it:
	// a verso title page is the mark of a book nobody typeset.
	//
	// Only the print interior is arranged this way. A reading PDF is read one
	// page at a time on a screen, where there is no back of a sheet to land
	// on, and a blank page inserted into it is just a blank page.
	if r.spec.Print && r.pdf.PageNo()%2 == 1 {
		r.addPage(pageBlank, "")
	}
	if !r.spec.OmitTitlePage {
		r.addPage(pageTitle, "")
		r.renderTitlePage()
	}

	for i := range r.doc.Sections {
		section := r.doc.Sections[i]
		if section.Role == SectionCopyright || section.Role == SectionFront {
			r.renderSection(section, false)
		}
	}
	if r.spec.GenerateTOC {
		r.reserveTOC()
	}
	for i := range r.doc.Sections {
		section := r.doc.Sections[i]
		if section.Role == SectionBody || section.Role == SectionBack {
			r.renderSection(section, true)
		}
	}
	if r.spec.GenerateTOC {
		if err := r.fillTOC(); err != nil {
			return err
		}
	}
	return r.pdf.Error()
}

// renderCoverPage puts the edition's artwork on a page of its own, before
// everything else.
//
// The image is fitted inside the trim and centred rather than stretched to
// fill it: Draftline's cover derivative is 1600 by 2560, which is 1 to 1.6,
// and a 6 by 9 page is 1 to 1.5. Stretching would distort the artist's work to
// hide a band of white, which is the wrong trade.
//
// This is the same front-cover derivative the ebook carries, and it is not a
// print cover: no spine, no back, no bleed. The interior PDF a printer wants
// does not carry a cover at all, and the screen says so — this page is for the
// reading copy an author sends to a reviewer.
func (r *publicationPDFRenderer) renderCoverPage() {
	cover := r.spec.Cover
	if !cover.Usable() {
		return
	}
	name := "edition-cover"
	r.pdf.RegisterImageOptionsReader(name, fpdf.ImageOptions{ImageType: cover.ImageType()}, bytes.NewReader(cover.Data))
	if r.pdf.Err() {
		// A cover that will not decode must not cost the author the book. The
		// error is cleared and the export goes on without the page.
		r.pdf.ClearError()
		return
	}

	r.addPage(pageCover, "")
	width, height := r.spec.TrimWidth, r.spec.TrimHeight
	if cover.Width > 0 && cover.Height > 0 {
		scale := math.Min(width/float64(cover.Width), height/float64(cover.Height))
		width = float64(cover.Width) * scale
		height = float64(cover.Height) * scale
	}
	x := r.trimX + (r.spec.TrimWidth-width)/2
	y := r.trimY + (r.spec.TrimHeight-height)/2
	r.pdf.ImageOptions(name, x, y, width, height, false, fpdf.ImageOptions{ImageType: cover.ImageType()}, 0, "")
}

func (r *publicationPDFRenderer) addPage(kind pdfPageKind, chapter string) {
	r.pdf.AddPage()
	r.pageKind = kind
	r.chapter = chapter
	left, _ := r.margins(r.pdf.PageNo())
	r.y = r.trimY + r.spec.TopMargin
	r.pdf.SetXY(r.trimX+left, r.y)
	if r.spec.CropMarks {
		r.drawCropMarks()
	}
	if kind == pageSection || kind == pageTOC {
		r.drawFurniture()
	}
}

func (r *publicationPDFRenderer) addSectionPage() {
	r.addPage(pageSection, r.chapter)
}

func (r *publicationPDFRenderer) margins(page int) (float64, float64) {
	if !r.spec.MirroredMargins {
		return r.spec.GutterMargin, r.spec.OuterMargin
	}
	if page%2 == 1 {
		return r.spec.GutterMargin, r.spec.OuterMargin
	}
	return r.spec.OuterMargin, r.spec.GutterMargin
}

func (r *publicationPDFRenderer) bodyBounds() (left, width, bottom float64) {
	leftMargin, rightMargin := r.margins(r.pdf.PageNo())
	return r.trimX + leftMargin,
		r.spec.TrimWidth - leftMargin - rightMargin,
		r.trimY + r.spec.TrimHeight - r.spec.BottomMargin
}

func (r *publicationPDFRenderer) ensureSpace(height float64) {
	_, _, bottom := r.bodyBounds()
	if r.y+height <= bottom {
		return
	}
	r.addSectionPage()
}

func (r *publicationPDFRenderer) renderTitlePage() {
	y := r.trimY + r.spec.TrimHeight*0.38
	titleScale := 2.35
	if r.spec.TitlePageStyle == "minimal" {
		y = r.trimY + r.spec.TrimHeight*0.28
		titleScale = 2.0
	} else if r.spec.TitlePageStyle == "dramatic" {
		y = r.trimY + r.spec.TrimHeight*0.48
		titleScale = 3.0
	}
	r.centeredTextWithFont(r.doc.Title, r.spec.TitlePageFont, r.spec.FontSize*titleScale, "B", y)
	if r.spec.TitlePageShowAuthor && r.doc.Author != "" {
		r.centeredText("by "+r.doc.Author, r.spec.FontSize*1.2, "", y+r.spec.LineHeight*3)
	}
	if r.spec.TitlePageShowPublisher && r.doc.Publisher != "" {
		r.centeredText(r.doc.Publisher, r.spec.FontSize, "", r.trimY+r.spec.TrimHeight-r.spec.BottomMargin)
	}
}

func (r *publicationPDFRenderer) centeredText(text string, size float64, style string, baseline float64) {
	r.centeredTextWithFont(text, r.spec.Font, size, style, baseline)
}

func (r *publicationPDFRenderer) centeredTextWithFont(text string, font embeddedFontFamily, size float64, style string, baseline float64) {
	r.pdf.SetFont(font.ID, style, size)
	w := r.pdf.GetStringWidth(text)
	x := r.trimX + (r.spec.TrimWidth-w)/2
	r.pdf.Text(math.Max(r.trimX, x), baseline, text)
}

func (r *publicationPDFRenderer) renderSection(section DocumentSection, includeInTOC bool) {
	// If the current page is recto, the next page would be verso: insert a
	// genuine blank page so body chapters begin on the next recto.
	if r.spec.ChapterStartsRecto && section.Role == SectionBody && r.pdf.PageNo()%2 == 1 {
		r.addPage(pageBlank, "")
	}
	r.chapter = section.Title
	r.addPage(pageSection, section.Title)
	if includeInTOC {
		r.toc = append(r.toc, pdfTOCEntry{Title: section.Title, Page: r.pdf.PageNo()})
	}

	if section.Role == SectionBody && r.spec.Print {
		r.y = r.trimY + r.spec.TrimHeight*0.29
	}
	if section.Title != "" && section.Role != SectionCopyright {
		r.renderHeadingText(section.Title, r.spec.FontSize*1.75, "B", "center", r.spec.LineHeight*1.5)
	}
	if section.Subtitle != "" {
		r.renderHeadingText(section.Subtitle, r.spec.FontSize*1.08, "I", "center", r.spec.LineHeight)
	}
	if section.Title != "" && section.Role != SectionCopyright {
		r.y += r.spec.LineHeight
	}

	// A slate is the chapter on a page of its own: the cue a narrator records
	// a take against. The length under it is how a session is planned.
	if r.spec.SlatePage && section.Role == SectionBody && section.Title != "" {
		if r.spec.ChapterWordCount {
			r.renderHeadingText(sectionLengthLine(section), r.spec.FontSize*0.85, "", "center", r.spec.LineHeight)
		}
		r.addPage(pageSection, section.Title)
	}

	firstParagraph := true
	for _, block := range section.Blocks {
		switch block.Kind {
		case BlockSceneBreak:
			r.renderSceneBreak()
		case BlockHeading:
			size := r.spec.FontSize * math.Max(1.08, 1.55-float64(block.Level)*0.08)
			r.renderHeadingText(block.PlainText(), size, "B", blockAlignment(block, "left"), r.spec.LineHeight)
		case BlockParagraph:
			if r.spec.NumberParagraphs {
				r.paragraphNumber++
				r.drawParagraphNumber()
			}
			r.renderTextBlock(block, 0, firstParagraph && r.spec.DropCap)
			r.y += r.spec.ParagraphSpacing
			firstParagraph = false
		case BlockBlockquote:
			r.renderTextBlock(block, 0.35*pointsPerInch, false)
		case BlockListItem:
			r.renderListItem(block)
		case BlockCode:
			r.renderCodeBlock(block)
		}
	}
}

func blockAlignment(block DocumentBlock, fallback string) string {
	if block.Alignment != "" {
		return block.Alignment
	}
	return fallback
}

func (r *publicationPDFRenderer) renderHeadingText(text string, size float64, style, align string, after float64) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	r.ensureSpace(size + after)
	left, width, _ := r.bodyBounds()
	r.pdf.SetFont(r.spec.HeadingFont.ID, style, size)
	w := r.pdf.GetStringWidth(text)
	x := left
	if align == "center" {
		x = left + (width-w)/2
	} else if align == "right" {
		x = left + width - w
	}
	r.pdf.Text(math.Max(left, x), r.y+size, text)
	r.y += size + after
}

func (r *publicationPDFRenderer) renderSceneBreak() {
	style := strings.ToLower(strings.TrimSpace(r.spec.SceneBreakStyle))
	if style == "space" {
		// A blank line is the break. Nothing is drawn, which is the point.
		r.ensureSpace(r.spec.LineHeight * 2)
		r.y += r.spec.LineHeight * 1.8
		return
	}
	r.ensureSpace(r.spec.LineHeight * 2)
	r.y += r.spec.LineHeight * 0.35
	mark := "*  *  *"
	if style == "rule" {
		mark = "———"
	}
	// A narration script says the break out loud, because an asterism is
	// silent and a narrator cannot act on it.
	if style == "pause" {
		mark = "[PAUSE]"
	}
	// ASCII asterisks are intentionally used rather than U+2042. Not every
	// author-selected body face contains the asterism glyph, which produced a
	// visible .notdef box in otherwise valid PDFs.
	r.centeredText(mark, r.spec.FontSize, "", r.y+r.spec.FontSize)
	r.y += r.spec.LineHeight * 1.45
}

func (r *publicationPDFRenderer) renderListItem(block DocumentBlock) {
	prefix := "\u2022"
	if block.Ordered {
		prefix = strconv.Itoa(block.ListNumber) + "."
	}
	copyBlock := block
	copyBlock.Runs = append([]DocumentRun{{Text: prefix + " "}}, block.Runs...)
	inset := float64(maxInt(1, block.ListDepth)) * 0.22 * pointsPerInch
	r.renderTextBlock(copyBlock, inset, false)
}

type pdfToken struct {
	Text  string
	Run   DocumentRun
	Break bool
}

type pdfLine struct {
	Tokens []pdfToken
	Width  float64
	Limit  float64
	Inset  float64
}

func (r *publicationPDFRenderer) renderTextBlock(block DocumentBlock, inset float64, dropCap bool) {
	_, bodyWidth, _ := r.bodyBounds()
	width := bodyWidth - 2*inset
	if width <= r.spec.FontSize*8 {
		return
	}
	align := blockAlignment(block, r.spec.TextAlign)
	indent := r.spec.ParagraphIndent
	if block.Kind == BlockBlockquote || block.Kind == BlockCode || block.Kind == BlockListItem {
		indent = 0
	}

	runs := append([]DocumentRun(nil), block.Runs...)
	dropPrefix := ""
	dropText := ""
	dropWidth := 0.0
	dropLines := 0
	if dropCap {
		dropPrefix, dropText, runs = takeDropCap(runs)
		if strings.TrimSpace(dropText) != "" {
			dropLines = r.spec.DropCapLines
			dropSize := r.spec.FontSize * float64(dropLines) * 0.82
			r.pdf.SetFont(r.spec.Font.ID, "", r.spec.FontSize*1.05)
			prefixWidth := r.pdf.GetStringWidth(dropPrefix)
			r.pdf.SetFont(r.spec.Font.ID, "", dropSize)
			dropWidth = prefixWidth + r.pdf.GetStringWidth(dropText) + r.spec.FontSize*0.35
			indent = 0
		}
	}

	lines := r.wrapRuns(runs, width, indent, dropWidth, dropLines)
	if len(lines) == 0 {
		return
	}
	r.ensureSpace(r.spec.LineHeight)
	if dropText != "" {
		left, _, _ := r.bodyBounds()
		x := left + inset
		if dropPrefix != "" {
			r.pdf.SetFont(r.spec.Font.ID, "", r.spec.FontSize*1.05)
			r.pdf.Text(x, r.y+r.spec.FontSize*0.88, dropPrefix)
			x += r.pdf.GetStringWidth(dropPrefix)
		}
		dropSize := r.spec.FontSize * float64(dropLines) * 0.82
		r.pdf.SetFont(r.spec.Font.ID, "", dropSize)
		r.pdf.Text(x, r.y+dropSize*0.78, dropText)
	}

	for i, line := range lines {
		r.ensureSpace(r.spec.LineHeight)
		lineLeft, lineWidth, _ := r.bodyBounds()
		lineLeft += inset + line.Inset
		lineWidth -= 2*inset + line.Inset
		if line.Limit > 0 {
			lineWidth = line.Limit
		}
		r.renderLine(line, lineLeft, lineWidth, align, i == len(lines)-1)
		r.y += r.spec.LineHeight
	}
	if !r.spec.Print {
		r.y += r.spec.LineHeight * 0.35
	}
}

func (r *publicationPDFRenderer) wrapRuns(runs []DocumentRun, width, indent, dropWidth float64, dropLines int) []pdfLine {
	tokens := tokenizeRuns(runs)
	lines := make([]pdfLine, 0, 4)
	line := pdfLine{}
	lineNumber := 0

	lineGeometry := func() (limit, inset float64) {
		if lineNumber == 0 {
			inset += indent
		}
		if lineNumber < dropLines {
			inset += dropWidth
		}
		return math.Max(r.spec.FontSize*4, width-inset), inset
	}
	line.Limit, line.Inset = lineGeometry()

	flush := func(force bool) {
		trimLineSpaces(&line, r)
		if len(line.Tokens) > 0 || force {
			lines = append(lines, line)
		}
		lineNumber++
		limit, lineInset := lineGeometry()
		line = pdfLine{Limit: limit, Inset: lineInset}
	}

	for _, token := range tokens {
		if token.Break {
			flush(true)
			continue
		}
		if token.Text == " " && len(line.Tokens) == 0 {
			continue
		}
		widthOfToken := r.tokenWidth(token)
		if line.Width+widthOfToken <= line.Limit || (len(line.Tokens) == 0 && widthOfToken <= line.Limit) {
			line.Tokens = append(line.Tokens, token)
			line.Width += widthOfToken
			continue
		}
		if len(line.Tokens) > 0 {
			flush(false)
			if token.Text == " " {
				continue
			}
		}
		for r.tokenWidth(token) > line.Limit && utf8.RuneCountInString(token.Text) > 1 {
			fit, rest := r.splitToken(token, line.Limit)
			line.Tokens = append(line.Tokens, fit)
			line.Width += r.tokenWidth(fit)
			flush(false)
			token.Text = rest
		}
		if token.Text != "" {
			line.Tokens = append(line.Tokens, token)
			line.Width += r.tokenWidth(token)
		}
	}
	if len(line.Tokens) > 0 {
		flush(false)
	}
	return lines
}

func tokenizeRuns(runs []DocumentRun) []pdfToken {
	tokens := make([]pdfToken, 0, len(runs)*2)
	for _, run := range runs {
		if run.LineBreak {
			tokens = append(tokens, pdfToken{Break: true})
			continue
		}
		var word strings.Builder
		flushWord := func() {
			if word.Len() > 0 {
				tokens = append(tokens, pdfToken{Text: word.String(), Run: run})
				word.Reset()
			}
		}
		for _, char := range run.Text {
			if char == '\n' {
				flushWord()
				tokens = append(tokens, pdfToken{Break: true})
				continue
			}
			if unicode.IsSpace(char) {
				flushWord()
				if len(tokens) == 0 || tokens[len(tokens)-1].Text != " " {
					tokens = append(tokens, pdfToken{Text: " ", Run: run})
				}
				continue
			}
			word.WriteRune(char)
		}
		flushWord()
	}
	return tokens
}

func trimLineSpaces(line *pdfLine, renderer *publicationPDFRenderer) {
	for len(line.Tokens) > 0 && line.Tokens[len(line.Tokens)-1].Text == " " {
		line.Width -= renderer.tokenWidth(line.Tokens[len(line.Tokens)-1])
		line.Tokens = line.Tokens[:len(line.Tokens)-1]
	}
}

func (r *publicationPDFRenderer) tokenWidth(token pdfToken) float64 {
	size := r.spec.FontSize
	if token.Run.Superscript || token.Run.Subscript {
		size *= 0.7
	}
	r.pdf.SetFont(r.fontForRun(token.Run).ID, runStyle(token.Run), size)
	return r.pdf.GetStringWidth(token.Text)
}

func (r *publicationPDFRenderer) fontForRun(run DocumentRun) embeddedFontFamily {
	if run.Code {
		return r.spec.CodeFont
	}
	return r.spec.Font
}

func (r *publicationPDFRenderer) splitToken(token pdfToken, limit float64) (pdfToken, string) {
	runes := []rune(token.Text)
	cut := 1
	for i := 2; i <= len(runes); i++ {
		candidate := token
		candidate.Text = string(runes[:i])
		if r.tokenWidth(candidate) > limit {
			break
		}
		cut = i
	}
	fit := token
	fit.Text = string(runes[:cut])
	return fit, string(runes[cut:])
}

func (r *publicationPDFRenderer) renderLine(line pdfLine, left, width float64, align string, last bool) {
	x := alignedLineStart(left, width, line.Width, align)
	extraSpace := 0.0
	if align == "justify" && !last {
		spaces := 0
		for _, token := range line.Tokens {
			if token.Text == " " {
				spaces++
			}
		}
		if spaces > 0 && line.Width < width {
			extraSpace = (width - line.Width) / float64(spaces)
		}
	}
	baseline := r.y + r.spec.FontSize*0.82
	for _, token := range line.Tokens {
		size := r.spec.FontSize
		tokenBaseline := baseline
		if token.Run.Superscript || token.Run.Subscript {
			size *= 0.7
			if token.Run.Superscript {
				tokenBaseline -= r.spec.FontSize * 0.32
			} else {
				tokenBaseline += r.spec.FontSize * 0.18
			}
		}
		r.pdf.SetFont(r.fontForRun(token.Run).ID, runStyle(token.Run), size)
		href := safeExportHref(token.Run.Href)
		if href != "" {
			r.pdf.SetTextColor(35, 78, 120)
		}
		r.pdf.Text(x, tokenBaseline, token.Text)
		tokenWidth := r.pdf.GetStringWidth(token.Text)
		if href != "" {
			r.pdf.LinkString(x, r.y, tokenWidth, r.spec.LineHeight, href)
		}
		if href != "" {
			r.pdf.SetTextColor(0, 0, 0)
		}
		x += tokenWidth
		if token.Text == " " {
			x += extraSpace
		}
	}
}

func alignedLineStart(left, width, lineWidth float64, align string) float64 {
	if align == "center" {
		return left + math.Max(0, (width-lineWidth)/2)
	}
	if align == "right" {
		return left + math.Max(0, width-lineWidth)
	}
	return left
}

func runStyle(run DocumentRun) string {
	style := ""
	if run.Bold {
		style += "B"
	}
	if run.Italic {
		style += "I"
	}
	if run.Underline {
		style += "U"
	}
	if run.Strike {
		style += "S"
	}
	return style
}

func takeDropCap(runs []DocumentRun) (string, string, []DocumentRun) {
	copyRuns := append([]DocumentRun(nil), runs...)
	var prefix strings.Builder
	for i := range runs {
		if runs[i].LineBreak {
			return "", "", runs
		}
		trimmed := strings.TrimLeftFunc(runs[i].Text, unicode.IsSpace)
		if trimmed == "" {
			continue
		}
		for offset, char := range trimmed {
			if !unicode.IsLetter(char) && !unicode.IsNumber(char) {
				prefix.WriteRune(char)
				continue
			}
			_, size := utf8.DecodeRuneInString(trimmed[offset:])
			copyRuns[i].Text = trimmed[offset+size:]
			return prefix.String(), string(char), copyRuns
		}
		copyRuns[i].Text = ""
	}
	return "", "", runs
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// sectionLengthLine is how long a chapter is, for planning a session. 150
// words a minute is the working rate for audiobook narration.
func sectionLengthLine(section DocumentSection) string {
	words := 0
	for _, block := range section.Blocks {
		words += len(strings.Fields(block.PlainText()))
	}
	minutes := words / 150
	if minutes < 1 {
		minutes = 1
	}
	return fmt.Sprintf("%d words, about %d min", words, minutes)
}
