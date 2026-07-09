package export

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"draftline/internal/types"
)

// PDF exports the book to PDF format at the specified path.
func PDF(path string, book types.BookData, options types.PDFOptions) types.ExportResult {
	// Generate PDF content
	pdf := generatePDF(book, options)

	err := os.WriteFile(path, pdf, 0644)
	if err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}

	return types.ExportResult{Success: true, FilePath: path}
}

// pdfWriter handles multi-page PDF generation with proper text layout
type pdfWriter struct {
	pageWidth    float64
	pageHeight   float64
	marginLeft   float64
	marginRight  float64
	marginTop    float64
	marginBottom float64
	fontSize     float64
	lineHeight   float64
	currentY     float64
	currentPage  *strings.Builder
	pageContents []string
}

func newPDFWriter(width, height, marginLeft, marginRight, marginTop, marginBottom float64, fontSize int) *pdfWriter {
	return &pdfWriter{
		pageWidth:    width,
		pageHeight:   height,
		marginLeft:   marginLeft,
		marginRight:  marginRight,
		marginTop:    marginTop,
		marginBottom: marginBottom,
		fontSize:     float64(fontSize),
		lineHeight:   float64(fontSize) * 1.5,
		currentY:     height - marginTop,
		currentPage:  &strings.Builder{},
	}
}

func (p *pdfWriter) textWidth() float64 {
	return p.pageWidth - p.marginLeft - p.marginRight
}

func (p *pdfWriter) charsPerLine() int {
	// Approximate characters per line (Helvetica average char width ~ 0.52 * fontSize)
	charWidth := p.fontSize * 0.52
	return int(p.textWidth() / charWidth)
}

func (p *pdfWriter) newPage() {
	if p.currentPage.Len() > 0 {
		p.pageContents = append(p.pageContents, p.currentPage.String())
	}
	p.currentPage = &strings.Builder{}
	p.currentY = p.pageHeight - p.marginTop
}

func (p *pdfWriter) writeLine(text string, fontSize float64, bold bool) {
	if p.currentY-p.lineHeight < p.marginBottom {
		p.newPage()
	}

	escaped := EscapePDFString(text)
	fontName := "/F1"
	if bold {
		fontName = "/F2"
	}

	p.currentPage.WriteString(fmt.Sprintf("BT\n%s %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
		fontName, fontSize, p.marginLeft, p.currentY, escaped))
	p.currentY -= p.lineHeight
}

func (p *pdfWriter) writeTitle(text string) {
	titleSize := p.fontSize * 1.8
	p.currentY -= p.lineHeight // Extra space before title
	p.writeLine(text, titleSize, true)
	p.currentY -= p.lineHeight * 0.5 // Extra space after title
}

func (p *pdfWriter) writeSubtitle(text string) {
	subtitleSize := p.fontSize * 1.2
	p.writeLine(text, subtitleSize, false)
	p.currentY -= p.lineHeight * 0.3
}

func (p *pdfWriter) writeParagraph(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	// Word wrap
	charsPerLine := p.charsPerLine()
	words := strings.Fields(text)
	var line strings.Builder

	for _, word := range words {
		if line.Len() == 0 {
			line.WriteString(word)
		} else if line.Len()+1+len(word) <= charsPerLine {
			line.WriteString(" ")
			line.WriteString(word)
		} else {
			p.writeLine(line.String(), p.fontSize, false)
			line.Reset()
			line.WriteString(word)
		}
	}

	if line.Len() > 0 {
		p.writeLine(line.String(), p.fontSize, false)
	}

	// Paragraph spacing
	p.currentY -= p.lineHeight * 0.5
}

func (p *pdfWriter) writeChapter(ch types.ChapterItem) {
	// Start chapter on new page
	p.newPage()

	// Chapter title
	p.writeTitle(ch.Title)

	// Subtitle if present
	if ch.Subtitle != "" {
		p.writeSubtitle(ch.Subtitle)
	}

	p.currentY -= p.lineHeight // Space before content

	// Content - split into paragraphs
	content := HtmlToPlainParagraphs(ch.Content)
	paragraphs := strings.Split(content, "\n")
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para != "" {
			p.writeParagraph(para)
		}
	}
}

func (p *pdfWriter) build() []byte {
	// Finalize current page
	if p.currentPage.Len() > 0 {
		p.pageContents = append(p.pageContents, p.currentPage.String())
	}

	if len(p.pageContents) == 0 {
		p.pageContents = append(p.pageContents, "")
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	numPages := len(p.pageContents)
	pageObjIDs := make([]int, numPages)
	nextObjID := 5

	for i := 0; i < numPages; i++ {
		pageObjIDs[i] = nextObjID
		nextObjID += 2
	}

	var kidsBuilder strings.Builder
	for i, id := range pageObjIDs {
		if i > 0 {
			kidsBuilder.WriteString(" ")
		}
		kidsBuilder.WriteString(fmt.Sprintf("%d 0 R", id))
	}

	var objects []string
	var offsets []int

	objects = append(objects, "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	objects = append(objects, fmt.Sprintf("2 0 obj\n<< /Type /Pages /Kids [%s] /Count %d >>\nendobj\n",
		kidsBuilder.String(), numPages))
	objects = append(objects, "3 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>\nendobj\n")
	objects = append(objects, "4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>\nendobj\n")

	for i, content := range p.pageContents {
		pageID := pageObjIDs[i]
		contentID := pageID + 1

		pageObj := fmt.Sprintf("%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Contents %d 0 R /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> >>\nendobj\n",
			pageID, p.pageWidth, p.pageHeight, contentID)
		objects = append(objects, pageObj)

		contentObj := fmt.Sprintf("%d 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n",
			contentID, len(content), content)
		objects = append(objects, contentObj)
	}

	currentOffset := buf.Len()
	for _, obj := range objects {
		offsets = append(offsets, currentOffset)
		buf.WriteString(obj)
		currentOffset = buf.Len()
	}

	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString(fmt.Sprintf("0 %d\n", len(objects)+1))
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}

	buf.WriteString("trailer\n")
	buf.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	buf.WriteString("startxref\n")
	buf.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	buf.WriteString("%%EOF\n")

	return buf.Bytes()
}

// generatePDF creates a properly formatted multi-page PDF (Letter size 8.5x11)
func generatePDF(book types.BookData, options types.PDFOptions) []byte {
	// Letter size: 8.5 x 11 inches = 612 x 792 points
	pageWidth := 612.0
	pageHeight := 792.0

	// 1 inch margins
	marginLeft := 72.0
	marginRight := 72.0
	marginTop := 72.0
	marginBottom := 72.0

	pdf := newPDFWriter(pageWidth, pageHeight, marginLeft, marginRight, marginTop, marginBottom, options.FontSize)

	// Title page
	pdf.currentY = pageHeight/2 + 50
	titleSize := float64(options.FontSize) * 2.5
	pdf.writeLine(book.Metadata.Title, titleSize, true)
	pdf.currentY -= pdf.lineHeight * 2
	if book.Metadata.Author != "" {
		pdf.writeLine("by "+book.Metadata.Author, float64(options.FontSize)*1.3, false)
	}
	if book.Metadata.Publisher != "" {
		pdf.currentY -= pdf.lineHeight
		pdf.writeLine(book.Metadata.Publisher, float64(options.FontSize), false)
	}

	// Copyright page
	if options.IncludeCopyright && book.Copyright != "" {
		pdf.newPage()
		content := HtmlToPlainParagraphs(book.Copyright)
		for _, para := range strings.Split(content, "\n") {
			para = strings.TrimSpace(para)
			if para != "" {
				pdf.writeParagraph(para)
			}
		}
	}

	// Front matter
	if options.IncludeFrontMatter {
		for _, ch := range book.FrontMatter {
			pdf.writeChapter(ch)
		}
	}

	// Body chapters
	for _, ch := range book.Body {
		pdf.writeChapter(ch)
	}

	// Back matter
	if options.IncludeBackMatter {
		for _, ch := range book.BackMatter {
			pdf.writeChapter(ch)
		}
	}

	return pdf.build()
}
