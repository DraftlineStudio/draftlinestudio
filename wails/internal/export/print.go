package export

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"draftline/internal/types"
)

// PrintPDF exports the book to print-ready PDF format at the specified path.
func PrintPDF(path string, book types.BookData, options types.PrintPDFOptions) types.ExportResult {
	// Generate print-ready PDF with custom trim size
	pdf := generatePrintPDF(book, options)

	err := os.WriteFile(path, pdf, 0644)
	if err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}

	return types.ExportResult{Success: true, FilePath: path}
}

// printPDFWriter extends pdfWriter with professional print features
type printPDFWriter struct {
	pageWidth       float64
	pageHeight      float64
	gutterMargin    float64 // Inside margin (toward spine)
	outerMargin     float64 // Outside margin
	topMargin       float64
	bottomMargin    float64
	fontSize        float64
	lineHeight      float64
	paragraphIndent float64
	currentY        float64
	currentPage     *strings.Builder
	pageContents    []string
	pageNumber      int // 1-indexed current page number
	mirroredMargins bool
	// Book metadata for headers
	bookTitle      string
	currentChapter string
	// Options
	runningHeaders     bool
	headerStyle        string
	pageNumberPosition string
	dropCap            bool
	dropCapLines       int
	chapterStartsRecto bool
	// TOC tracking
	tocEntries []tocEntry
}

type tocEntry struct {
	title   string
	pageNum int
}

func newPrintPDFWriter(opts types.PrintPDFOptions, bookTitle string) *printPDFWriter {
	// Calculate page dimensions from trim size (72 points = 1 inch)
	var pageWidth, pageHeight float64

	switch opts.TrimSize {
	case "5x8":
		pageWidth = 5.0 * 72
		pageHeight = 8.0 * 72
	case "5.25x8":
		pageWidth = 5.25 * 72
		pageHeight = 8.0 * 72
	case "5.5x8.5":
		pageWidth = 5.5 * 72
		pageHeight = 8.5 * 72
	case "6x9":
		pageWidth = 6.0 * 72
		pageHeight = 9.0 * 72
	case "custom":
		w, _ := strconv.ParseFloat(opts.CustomWidth, 64)
		h, _ := strconv.ParseFloat(opts.CustomHeight, 64)
		if w <= 0 {
			w = 5.5
		}
		if h <= 0 {
			h = 8.5
		}
		pageWidth = w * 72
		pageHeight = h * 72
	default:
		pageWidth = 5.5 * 72
		pageHeight = 8.5 * 72
	}

	// Parse margins
	gutterMargin := 0.875
	if g, err := strconv.ParseFloat(opts.GutterMargin, 64); err == nil && g > 0 {
		gutterMargin = g
	}
	outerMargin := 0.625
	if o, err := strconv.ParseFloat(opts.OuterMargin, 64); err == nil && o > 0 {
		outerMargin = o
	}
	topMargin := 0.75
	if t, err := strconv.ParseFloat(opts.TopMargin, 64); err == nil && t > 0 {
		topMargin = t
	}
	bottomMargin := 0.625
	if b, err := strconv.ParseFloat(opts.BottomMargin, 64); err == nil && b > 0 {
		bottomMargin = b
	}
	paragraphIndent := 0.25
	if pi, err := strconv.ParseFloat(opts.ParagraphIndent, 64); err == nil && pi >= 0 {
		paragraphIndent = pi
	}

	lineHeight := opts.LineHeight
	if lineHeight <= 0 {
		lineHeight = 1.4
	}

	dropCapLines := opts.DropCapLines
	if dropCapLines <= 0 {
		dropCapLines = 3
	}

	return &printPDFWriter{
		pageWidth:          pageWidth,
		pageHeight:         pageHeight,
		gutterMargin:       gutterMargin * 72,
		outerMargin:        outerMargin * 72,
		topMargin:          topMargin * 72,
		bottomMargin:       bottomMargin * 72,
		fontSize:           float64(opts.FontSize),
		lineHeight:         float64(opts.FontSize) * lineHeight,
		paragraphIndent:    paragraphIndent * 72,
		currentY:           pageHeight - topMargin*72,
		currentPage:        &strings.Builder{},
		pageNumber:         0,
		mirroredMargins:    opts.MirroredMargins,
		bookTitle:          bookTitle,
		runningHeaders:     opts.RunningHeaders,
		headerStyle:        opts.HeaderStyle,
		pageNumberPosition: opts.PageNumberPosition,
		dropCap:            opts.DropCap,
		dropCapLines:       dropCapLines,
		chapterStartsRecto: opts.ChapterStartsRecto,
		tocEntries:         []tocEntry{},
	}
}

// getMargins returns left and right margins for the current page (handles mirroring)
func (p *printPDFWriter) getMargins() (left, right float64) {
	if !p.mirroredMargins {
		// No mirroring - use average
		avg := (p.gutterMargin + p.outerMargin) / 2
		return avg, avg
	}
	// Mirrored margins: odd pages have gutter on LEFT, even pages have gutter on RIGHT
	if p.pageNumber%2 == 1 {
		// Odd page (recto/right page) - gutter is on left (spine side)
		return p.gutterMargin, p.outerMargin
	}
	// Even page (verso/left page) - gutter is on right (spine side)
	return p.outerMargin, p.gutterMargin
}

func (p *printPDFWriter) textWidth() float64 {
	left, right := p.getMargins()
	return p.pageWidth - left - right
}

func (p *printPDFWriter) charsPerLine() int {
	charWidth := p.fontSize * 0.52
	return int(p.textWidth() / charWidth)
}

func (p *printPDFWriter) newPage() {
	if p.currentPage.Len() > 0 {
		p.pageContents = append(p.pageContents, p.currentPage.String())
	}
	p.currentPage = &strings.Builder{}
	p.pageNumber++
	p.currentY = p.pageHeight - p.topMargin
}

// ensureRectoPage ensures the next content starts on an odd (right-hand) page
func (p *printPDFWriter) ensureRectoPage() {
	// If we're on an even page, add a blank page
	if p.pageNumber > 0 && p.pageNumber%2 == 0 {
		p.newPage() // Add blank verso page
	}
}

func (p *printPDFWriter) writeLine(text string, fontSize float64, bold bool) {
	if p.currentY-p.lineHeight < p.bottomMargin {
		p.newPage()
	}

	escaped := EscapePDFString(text)
	fontName := "/F1"
	if bold {
		fontName = "/F2"
	}

	left, _ := p.getMargins()
	p.currentPage.WriteString(fmt.Sprintf("BT\n%s %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
		fontName, fontSize, left, p.currentY, escaped))
	p.currentY -= p.lineHeight
}

func (p *printPDFWriter) writeLineCentered(text string, fontSize float64, bold bool) {
	if p.currentY-p.lineHeight < p.bottomMargin {
		p.newPage()
	}

	escaped := EscapePDFString(text)
	fontName := "/F1"
	if bold {
		fontName = "/F2"
	}

	// Estimate text width and center it
	textWidthPts := float64(len(text)) * fontSize * 0.52
	left, right := p.getMargins()
	availWidth := p.pageWidth - left - right
	xPos := left + (availWidth-textWidthPts)/2
	if xPos < left {
		xPos = left
	}

	p.currentPage.WriteString(fmt.Sprintf("BT\n%s %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
		fontName, fontSize, xPos, p.currentY, escaped))
	p.currentY -= p.lineHeight
}

func (p *printPDFWriter) writeChapterTitle(title string) {
	// Position title roughly 1/3 down the page for chapter openings
	targetY := p.pageHeight * 0.7
	if p.currentY > targetY {
		p.currentY = targetY
	}

	titleSize := p.fontSize * 1.8
	p.writeLineCentered(title, titleSize, true)
	p.currentY -= p.lineHeight * 2 // Extra space after chapter title
}

func (p *printPDFWriter) writeParagraph(text string, isFirst bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	charsPerLine := p.charsPerLine()
	words := strings.Fields(text)
	var line strings.Builder
	lineNum := 0

	left, _ := p.getMargins()
	indent := p.paragraphIndent

	// First paragraph after chapter title: no indent (Reedsy style)
	if isFirst {
		indent = 0
	}

	for _, word := range words {
		if line.Len() == 0 {
			line.WriteString(word)
		} else if line.Len()+1+len(word) <= charsPerLine {
			line.WriteString(" ")
			line.WriteString(word)
		} else {
			// Write the line
			if p.currentY-p.lineHeight < p.bottomMargin {
				p.newPage()
				left, _ = p.getMargins()
			}
			xPos := left
			if lineNum == 0 {
				xPos += indent
			}
			escaped := EscapePDFString(line.String())
			p.currentPage.WriteString(fmt.Sprintf("BT\n/F1 %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
				p.fontSize, xPos, p.currentY, escaped))
			p.currentY -= p.lineHeight
			lineNum++
			line.Reset()
			line.WriteString(word)
		}
	}

	if line.Len() > 0 {
		if p.currentY-p.lineHeight < p.bottomMargin {
			p.newPage()
			left, _ = p.getMargins()
		}
		xPos := left
		if lineNum == 0 {
			xPos += indent
		}
		escaped := EscapePDFString(line.String())
		p.currentPage.WriteString(fmt.Sprintf("BT\n/F1 %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
			p.fontSize, xPos, p.currentY, escaped))
		p.currentY -= p.lineHeight
	}

	// Paragraph spacing - small gap
	p.currentY -= p.lineHeight * 0.3
}

func (p *printPDFWriter) writeHalfTitlePage(title, author string) {
	p.newPage()

	// Position title in upper third of page
	p.currentY = p.pageHeight * 0.65

	// Author name in small caps (simulated with smaller font)
	if author != "" {
		authorSize := p.fontSize * 1.1
		p.writeLineCentered(strings.ToUpper(author), authorSize, false)
		p.currentY -= p.lineHeight
	}

	// Book title
	titleSize := p.fontSize * 2.0
	p.writeLineCentered(title, titleSize, true)
}

func (p *printPDFWriter) writeCopyrightPage(copyright string) {
	p.newPage()

	// Copyright page content - positioned near top
	p.currentY = p.pageHeight - p.topMargin - p.lineHeight*2

	content := HtmlToPlainParagraphs(copyright)
	paragraphs := strings.Split(content, "\n")
	for i, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para != "" {
			// Center copyright text
			p.writeLineCentered(para, p.fontSize*0.9, false)
			if i < len(paragraphs)-1 {
				p.currentY -= p.lineHeight * 0.5
			}
		}
	}
}

func (p *printPDFWriter) writeTOCPage() {
	if len(p.tocEntries) == 0 {
		return
	}

	p.newPage()

	// TOC header
	p.currentY = p.pageHeight * 0.75
	headerSize := p.fontSize * 1.5
	p.writeLineCentered("Contents", headerSize, true)
	p.currentY -= p.lineHeight * 2

	left, right := p.getMargins()
	availWidth := p.pageWidth - left - right

	for _, entry := range p.tocEntries {
		if p.currentY-p.lineHeight < p.bottomMargin {
			p.newPage()
			left, _ = p.getMargins()
		}

		// Write chapter title on left
		escaped := EscapePDFString(entry.title)
		p.currentPage.WriteString(fmt.Sprintf("BT\n/F1 %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
			p.fontSize, left, p.currentY, escaped))

		// Write page number on right
		pageStr := fmt.Sprintf("%d", entry.pageNum)
		pageWidth := float64(len(pageStr)) * p.fontSize * 0.52
		xPos := left + availWidth - pageWidth
		escapedPage := EscapePDFString(pageStr)
		p.currentPage.WriteString(fmt.Sprintf("BT\n/F1 %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
			p.fontSize, xPos, p.currentY, escapedPage))

		p.currentY -= p.lineHeight * 1.2
	}
}

func (p *printPDFWriter) writeChapter(ch types.ChapterItem, _ bool) {
	// Record page for TOC (before potentially adding blank page)
	startPage := p.pageNumber + 1

	// Start chapter on new page
	if p.chapterStartsRecto {
		p.newPage()
		p.ensureRectoPage()
	} else {
		p.newPage()
	}

	// Update current chapter for running headers
	p.currentChapter = ch.Title

	// Add to TOC
	p.tocEntries = append(p.tocEntries, tocEntry{
		title:   ch.Title,
		pageNum: p.pageNumber,
	})
	_ = startPage // may use later for roman numerals

	// Chapter title
	p.writeChapterTitle(ch.Title)

	// Subtitle if present
	if ch.Subtitle != "" {
		subtitleSize := p.fontSize * 1.2
		p.writeLineCentered(ch.Subtitle, subtitleSize, false)
		p.currentY -= p.lineHeight
	}

	// Content - split into paragraphs
	content := HtmlToPlainParagraphs(ch.Content)
	paragraphs := strings.Split(content, "\n")
	isFirst := true
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para != "" {
			p.writeParagraph(para, isFirst)
			isFirst = false
		}
	}
}

func (p *printPDFWriter) build() []byte {
	// Finalize current page
	if p.currentPage.Len() > 0 {
		p.pageContents = append(p.pageContents, p.currentPage.String())
	}

	if len(p.pageContents) == 0 {
		p.pageContents = append(p.pageContents, "")
	}

	numPages := len(p.pageContents)
	pageObjIDs := make([]int, numPages)
	nextObjID := 5

	for i := 0; i < numPages; i++ {
		pageObjIDs[i] = nextObjID
		nextObjID += 2
	}

	// Collect all object bodies in ID order so we can record true byte
	// offsets as each object is written (required for a valid xref table).
	var objects []string

	// Object 1: Catalog
	objects = append(objects, "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Object 2: Pages
	var pagesObj strings.Builder
	pagesObj.WriteString("2 0 obj\n<< /Type /Pages /Kids [")
	for i, id := range pageObjIDs {
		if i > 0 {
			pagesObj.WriteString(" ")
		}
		pagesObj.WriteString(fmt.Sprintf("%d 0 R", id))
	}
	pagesObj.WriteString(fmt.Sprintf("] /Count %d >>\nendobj\n", numPages))
	objects = append(objects, pagesObj.String())

	// Object 3: Font (Helvetica)
	objects = append(objects, "3 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	// Object 4: Bold Font (Helvetica-Bold)
	objects = append(objects, "4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>\nendobj\n")

	// Pages and content streams
	for i, content := range p.pageContents {
		pageObjID := pageObjIDs[i]
		contentObjID := pageObjID + 1
		pageNum := i + 1

		// Add running header and page number to content
		finalContent := content

		// Page number at bottom center
		if p.pageNumberPosition != "" {
			pageNumStr := fmt.Sprintf("%d", pageNum)
			pageNumEscaped := EscapePDFString(pageNumStr)
			var pageNumX, pageNumY float64

			left, right := p.gutterMargin, p.outerMargin
			if p.mirroredMargins && pageNum%2 == 0 {
				left, right = p.outerMargin, p.gutterMargin
			}

			switch p.pageNumberPosition {
			case "bottom-center":
				pageNumX = p.pageWidth / 2
				pageNumY = p.bottomMargin / 2
			case "bottom-outside":
				if pageNum%2 == 1 {
					pageNumX = p.pageWidth - right - 10
				} else {
					pageNumX = left + 10
				}
				pageNumY = p.bottomMargin / 2
			case "top-outside":
				if pageNum%2 == 1 {
					pageNumX = p.pageWidth - right - 10
				} else {
					pageNumX = left + 10
				}
				pageNumY = p.pageHeight - p.topMargin/2
			default:
				pageNumX = p.pageWidth / 2
				pageNumY = p.bottomMargin / 2
			}

			finalContent += fmt.Sprintf("BT\n/F1 %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
				p.fontSize*0.9, pageNumX, pageNumY, pageNumEscaped)
		}

		// Running headers
		if p.runningHeaders && pageNum > 1 {
			headerY := p.pageHeight - p.topMargin/2
			headerFontSize := p.fontSize * 0.85

			var headerText string
			if pageNum%2 == 0 {
				// Even page (verso): book title
				headerText = p.bookTitle
			} else {
				// Odd page (recto): chapter title
				headerText = p.currentChapter
			}

			if p.headerStyle == "smallcaps" {
				headerText = strings.ToUpper(headerText)
				headerFontSize = p.fontSize * 0.75
			}

			headerEscaped := EscapePDFString(headerText)
			headerWidth := float64(len(headerText)) * headerFontSize * 0.52
			headerX := (p.pageWidth - headerWidth) / 2

			fontName := "/F1"
			if p.headerStyle == "italic" {
				// Note: We'd need an italic font, using regular for now
				fontName = "/F1"
			}

			finalContent += fmt.Sprintf("BT\n%s %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
				fontName, headerFontSize, headerX, headerY, headerEscaped)
		}

		// Page object
		objects = append(objects, fmt.Sprintf("%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Contents %d 0 R /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> >>\nendobj\n",
			pageObjID, p.pageWidth, p.pageHeight, contentObjID))

		// Content stream
		objects = append(objects, fmt.Sprintf("%d 0 obj\n<< /Length %d >>\nstream\n%sendstream\nendobj\n",
			contentObjID, len(finalContent), finalContent))
	}

	// Write header and all objects, recording the true byte offset of each
	// object so the xref table points exactly at its "N 0 obj" marker.
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(objects))
	for i, obj := range objects {
		offsets[i] = buf.Len()
		buf.WriteString(obj)
	}

	// Xref and trailer with accurate offsets
	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString(fmt.Sprintf("0 %d\n", nextObjID))
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}
	buf.WriteString("trailer\n")
	buf.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", nextObjID))
	buf.WriteString("startxref\n")
	buf.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	buf.WriteString("%%EOF\n")

	return buf.Bytes()
}

// generatePrintPDF creates a print-ready PDF with custom trim size and professional formatting
func generatePrintPDF(book types.BookData, options types.PrintPDFOptions) []byte {
	// Create print PDF writer with all options
	pdf := newPrintPDFWriter(options, book.Metadata.Title)

	// === FRONT MATTER ===

	// Half-title page (optional) - recto
	if options.GenerateHalfTitle {
		pdf.writeHalfTitlePage(book.Metadata.Title, book.Metadata.Author)
	}

	// Copyright page - verso (even page, back of half-title)
	if options.IncludeCopyright && book.Copyright != "" {
		pdf.writeCopyrightPage(book.Copyright)
	}

	// User's front matter chapters
	if options.IncludeFrontMatter {
		for i, ch := range book.FrontMatter {
			pdf.writeChapter(ch, i == 0)
		}
	}

	// === BODY CHAPTERS ===
	isFirstBody := true
	for _, ch := range book.Body {
		pdf.writeChapter(ch, isFirstBody)
		isFirstBody = false
	}

	// === BACK MATTER ===
	if options.IncludeBackMatter {
		for _, ch := range book.BackMatter {
			pdf.writeChapter(ch, false)
		}
	}

	// Build and return the PDF
	// Note: TOC generation would require a two-pass approach since we need page numbers
	// For now, TOC is tracked but not inserted (would require rewriting page order)
	return pdf.build()
}
