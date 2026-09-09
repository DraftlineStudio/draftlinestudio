package export

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"codeberg.org/go-pdf/fpdf"
	"draftline/internal/types"
)

const pointsPerInch = 72.0

type publicationPDFSpec struct {
	Print              bool
	TrimWidth          float64
	TrimHeight         float64
	Bleed              float64
	CropMarks          bool
	GutterMargin       float64
	OuterMargin        float64
	TopMargin          float64
	BottomMargin       float64
	MirroredMargins    bool
	Font               embeddedFontFamily
	FontSize           float64
	LineHeight         float64
	ParagraphIndent    float64
	TextAlign          string
	ChapterStartsRecto bool
	DropCap            bool
	DropCapLines       int
	RunningHeaders     bool
	HeaderStyle        string
	PageNumberPosition string
	GenerateHalfTitle  bool
	GenerateTOC        bool
}

func readingPDFSpec(options types.PDFOptions) publicationPDFSpec {
	w, h := readingPageSize(options.PageSize)
	fontSize := float64(options.FontSize)
	if fontSize <= 0 {
		fontSize = 12
	}
	lineHeight := options.LineHeight
	if lineHeight <= 0 {
		lineHeight = 1.5
	}
	return publicationPDFSpec{
		TrimWidth:       w,
		TrimHeight:      h,
		GutterMargin:    pointsPerInch,
		OuterMargin:     pointsPerInch,
		TopMargin:       pointsPerInch,
		BottomMargin:    pointsPerInch,
		Font:            resolvePDFFont(options.FontFamily),
		FontSize:        fontSize,
		LineHeight:      fontSize * lineHeight,
		ParagraphIndent: parseInches(options.ParagraphIndent, 0.25),
		TextAlign:       normalizedAlignment(options.TextAlign, "left"),
	}
}

func printPDFSpec(options types.PrintPDFOptions) publicationPDFSpec {
	w, h := trimPageSize(options.TrimSize, options.CustomWidth, options.CustomHeight)
	fontSize := float64(options.FontSize)
	if fontSize <= 0 {
		fontSize = 11
	}
	lineHeight := options.LineHeight
	if lineHeight <= 0 {
		lineHeight = 1.4
	}
	dropLines := options.DropCapLines
	if dropLines < 2 || dropLines > 4 {
		dropLines = 3
	}
	return publicationPDFSpec{
		Print:              true,
		TrimWidth:          w,
		TrimHeight:         h,
		Bleed:              parseInches(options.Bleed, 0),
		CropMarks:          options.IncludeCropMarks,
		GutterMargin:       parseInches(options.GutterMargin, 0.875),
		OuterMargin:        parseInches(options.OuterMargin, 0.625),
		TopMargin:          parseInches(options.TopMargin, 0.75),
		BottomMargin:       parseInches(options.BottomMargin, 0.625),
		MirroredMargins:    options.MirroredMargins,
		Font:               resolvePDFFont(options.FontFamily),
		FontSize:           fontSize,
		LineHeight:         fontSize * lineHeight,
		ParagraphIndent:    parseInches(options.ParagraphIndent, 0.25),
		TextAlign:          normalizedAlignment(options.TextAlign, "justify"),
		ChapterStartsRecto: options.ChapterStartsRecto,
		DropCap:            options.DropCap,
		DropCapLines:       dropLines,
		RunningHeaders:     options.RunningHeaders,
		HeaderStyle:        options.HeaderStyle,
		PageNumberPosition: normalizedPageNumberPosition(options.PageNumberPosition),
		GenerateHalfTitle:  options.GenerateHalfTitle,
		GenerateTOC:        options.GenerateTOC,
	}
}

func readingPageSize(name string) (float64, float64) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "a4":
		return 595.28, 841.89
	case "5x8":
		return 5 * pointsPerInch, 8 * pointsPerInch
	case "5.5x8.5":
		return 5.5 * pointsPerInch, 8.5 * pointsPerInch
	case "6x9":
		return 6 * pointsPerInch, 9 * pointsPerInch
	default:
		return 8.5 * pointsPerInch, 11 * pointsPerInch
	}
}

func trimPageSize(name, customWidth, customHeight string) (float64, float64) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "5x8":
		return 5 * pointsPerInch, 8 * pointsPerInch
	case "5.25x8":
		return 5.25 * pointsPerInch, 8 * pointsPerInch
	case "6x9":
		return 6 * pointsPerInch, 9 * pointsPerInch
	case "custom":
		return parseInches(customWidth, 5.5), parseInches(customHeight, 8.5)
	default:
		return 5.5 * pointsPerInch, 8.5 * pointsPerInch
	}
}

func parseInches(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || parsed < 0 {
		parsed = fallback
	}
	return parsed * pointsPerInch
}

func normalizedAlignment(value, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "left", "center", "right", "justify":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return fallback
	}
}

func normalizedPageNumberPosition(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bottom-outside", "top-outside":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "bottom-center"
	}
}

type pdfPageKind string

const (
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
	pageKind   pdfPageKind
	tocPages   []int
	toc        []pdfTOCEntry
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
	registerPDFFonts(pdf)
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
	if r.spec.GenerateHalfTitle {
		r.addPage(pageHalfTitle, "")
		r.centeredText(r.doc.Title, r.spec.FontSize*1.7, "B", r.trimY+r.spec.TrimHeight*0.42)
	}
	r.addPage(pageTitle, "")
	r.renderTitlePage()

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
		r.fillTOC()
	}
	return r.pdf.Error()
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
	r.centeredText(r.doc.Title, r.spec.FontSize*2.35, "B", y)
	if r.doc.Author != "" {
		r.centeredText("by "+r.doc.Author, r.spec.FontSize*1.2, "", y+r.spec.LineHeight*3)
	}
	if r.doc.Publisher != "" {
		r.centeredText(r.doc.Publisher, r.spec.FontSize, "", r.trimY+r.spec.TrimHeight-r.spec.BottomMargin)
	}
}

func (r *publicationPDFRenderer) centeredText(text string, size float64, style string, baseline float64) {
	r.pdf.SetFont(r.spec.Font.ID, style, size)
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

	firstParagraph := true
	for _, block := range section.Blocks {
		switch block.Kind {
		case BlockSceneBreak:
			r.renderSceneBreak()
		case BlockHeading:
			size := r.spec.FontSize * math.Max(1.08, 1.55-float64(block.Level)*0.08)
			r.renderHeadingText(block.PlainText(), size, "B", blockAlignment(block, "left"), r.spec.LineHeight)
		case BlockParagraph:
			r.renderTextBlock(block, 0, firstParagraph && r.spec.DropCap)
			firstParagraph = false
		case BlockBlockquote:
			r.renderTextBlock(block, 0.35*pointsPerInch, false)
		case BlockListItem:
			r.renderListItem(block)
		case BlockCode:
			r.renderTextBlock(block, 0.25*pointsPerInch, false)
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
	r.pdf.SetFont(r.spec.Font.ID, style, size)
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
	r.ensureSpace(r.spec.LineHeight * 2)
	r.y += r.spec.LineHeight * 0.35
	r.centeredText("\u2042", r.spec.FontSize*1.05, "", r.y+r.spec.FontSize)
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
	dropText := ""
	dropWidth := 0.0
	dropLines := 0
	if dropCap {
		dropText, runs = takeFirstRune(runs)
		if strings.TrimSpace(dropText) != "" {
			dropLines = r.spec.DropCapLines
			dropSize := r.spec.FontSize * float64(dropLines) * 0.82
			r.pdf.SetFont(r.spec.Font.ID, "", dropSize)
			dropWidth = r.pdf.GetStringWidth(dropText) + r.spec.FontSize*0.35
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
		dropSize := r.spec.FontSize * float64(dropLines) * 0.82
		r.pdf.SetFont(r.spec.Font.ID, "", dropSize)
		r.pdf.Text(left+inset, r.y+dropSize*0.78, dropText)
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
	r.pdf.SetFont(r.spec.Font.ID, runStyle(token.Run), size)
	return r.pdf.GetStringWidth(token.Text)
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
	x := left
	if align == "center" {
		x += math.Max(0, (width-line.Width)/2)
	} else if align == "right" {
		x += math.Max(0, width-line.Width)
	}
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
		r.pdf.SetFont(r.spec.Font.ID, runStyle(token.Run), size)
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

func takeFirstRune(runs []DocumentRun) (string, []DocumentRun) {
	for i := range runs {
		trimmed := strings.TrimLeftFunc(runs[i].Text, unicode.IsSpace)
		if trimmed == "" {
			continue
		}
		first, size := utf8.DecodeRuneInString(trimmed)
		copyRuns := append([]DocumentRun(nil), runs...)
		copyRuns[i].Text = trimmed[size:]
		return string(first), copyRuns
	}
	return "", runs
}

func (r *publicationPDFRenderer) drawFurniture() {
	page := r.pdf.PageNo()
	leftMargin, rightMargin := r.margins(page)
	left := r.trimX + leftMargin
	width := r.spec.TrimWidth - leftMargin - rightMargin
	if r.spec.RunningHeaders && r.pageKind == pageSection {
		header := r.chapter
		if page%2 == 0 {
			header = r.doc.Title
		}
		style := ""
		if strings.EqualFold(r.spec.HeaderStyle, "italic") {
			style = "I"
		}
		if strings.EqualFold(r.spec.HeaderStyle, "smallcaps") {
			header = strings.ToUpper(header)
		}
		r.pdf.SetFont(r.spec.Font.ID, style, r.spec.FontSize*0.72)
		w := r.pdf.GetStringWidth(header)
		x := left
		if page%2 == 1 {
			x = left + width - w
		}
		r.pdf.Text(x, r.trimY+r.spec.TopMargin*0.48, header)
	}
	if r.spec.PageNumberPosition == "" {
		return
	}
	r.pdf.SetFont(r.spec.Font.ID, "", r.spec.FontSize*0.72)
	text := strconv.Itoa(page)
	w := r.pdf.GetStringWidth(text)
	x := left + (width-w)/2
	y := r.trimY + r.spec.TrimHeight - r.spec.BottomMargin*0.4
	if r.spec.PageNumberPosition == "bottom-outside" || r.spec.PageNumberPosition == "top-outside" {
		if page%2 == 1 {
			x = r.trimX + r.spec.TrimWidth - rightMargin - w
		} else {
			x = r.trimX + leftMargin
		}
	}
	if r.spec.PageNumberPosition == "top-outside" {
		y = r.trimY + r.spec.TopMargin*0.48
	}
	r.pdf.Text(x, y, text)
}

func (r *publicationPDFRenderer) drawCropMarks() {
	const length = 12.0
	const gap = 4.0
	x1 := r.trimX
	x2 := r.trimX + r.spec.TrimWidth
	y1 := r.trimY
	y2 := r.trimY + r.spec.TrimHeight
	r.pdf.SetDrawColor(0, 0, 0)
	r.pdf.SetLineWidth(0.25)
	for _, x := range []float64{x1, x2} {
		r.pdf.Line(x, y1-gap, x, y1-gap-length)
		r.pdf.Line(x, y2+gap, x, y2+gap+length)
	}
	for _, y := range []float64{y1, y2} {
		r.pdf.Line(x1-gap, y, x1-gap-length, y)
		r.pdf.Line(x2+gap, y, x2+gap+length, y)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
