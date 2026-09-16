package export

// What goes around the text block: the running head, the folio, the crop
// marks a printer trims to, and the paragraph numbers a narration script
// carries in its gutter.
//
// None of it is part of the manuscript. All of it is drawn after a page's
// text is laid out, from the spec rather than from the document, which is why
// it lives apart from the renderer that sets the words.

import (
	"strconv"
	"strings"
)

// drawFurniture puts the running head and the folio on the page.
//
// The two can share a line. With folios set top-outside they sit on the same
// baseline at the same outside edge, so the head is indented inside the folio
// by its own width plus a space — otherwise the page number is printed on top
// of the author's name, which is what happened before this measured anything.
func (r *publicationPDFRenderer) drawFurniture() {
	page := r.pdf.PageNo()
	leftMargin, rightMargin := r.margins(page)
	left := r.trimX + leftMargin
	width := r.spec.TrimWidth - leftMargin - rightMargin
	recto := page%2 == 1
	headerY := r.trimY + r.spec.TopMargin*0.48
	furnitureSize := r.spec.FontSize * 0.72

	// The folio first, because the head has to know how much room it left.
	folio := ""
	folioWidth := 0.0
	if r.spec.PageNumberPosition != "" {
		r.pdf.SetFont(r.spec.FurnitureFont.ID, "", furnitureSize)
		folio = strconv.Itoa(page)
		folioWidth = r.pdf.GetStringWidth(folio)
	}
	sharesTheLine := folio != "" && r.spec.PageNumberPosition == "top-outside"

	if r.spec.RunningHeaders && r.pageKind == pageSection {
		header := r.runningHead(page)
		style := ""
		if strings.EqualFold(r.spec.HeaderStyle, "italic") {
			style = "I"
		}
		if strings.EqualFold(r.spec.HeaderStyle, "smallcaps") {
			header = strings.ToUpper(header)
		}
		if header != "" {
			r.pdf.SetFont(r.spec.FurnitureFont.ID, style, furnitureSize)
			inset := 0.0
			if sharesTheLine {
				inset = folioWidth + r.pdf.GetStringWidth("  ")
			}
			w := r.pdf.GetStringWidth(header)
			x := left + inset
			if recto {
				x = left + width - inset - w
			}
			r.pdf.Text(x, headerY, header)
		}
	}

	if folio == "" {
		return
	}
	r.pdf.SetFont(r.spec.FurnitureFont.ID, "", furnitureSize)
	x := left + (width-folioWidth)/2
	y := r.trimY + r.spec.TrimHeight - r.spec.BottomMargin*0.4
	if r.spec.PageNumberPosition == "bottom-outside" || r.spec.PageNumberPosition == "top-outside" {
		if recto {
			x = r.trimX + r.spec.TrimWidth - rightMargin - folioWidth
		} else {
			x = r.trimX + leftMargin
		}
	}
	if r.spec.PageNumberPosition == "top-outside" {
		y = headerY
	}
	r.pdf.Text(x, y, folio)
}

// runningHead is what the header says on this page.
//
// A verso and a recto carry different things, which is the whole point of a
// running head: a reader who opens the book in the middle can see whose book
// it is on one side and where they are on the other.
func (r *publicationPDFRenderer) runningHead(page int) string {
	verso := page%2 == 0
	switch strings.ToLower(strings.TrimSpace(r.spec.HeaderContent)) {
	case "chapter":
		return r.chapter
	case "title-chapter":
		if verso {
			return r.doc.Title
		}
		return r.chapter
	default:
		// author-title, and anything a later build writes that this one does
		// not know: the author's own name is never the wrong thing to print.
		if verso {
			return r.doc.Author
		}
		return r.doc.Title
	}
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

// drawParagraphNumber puts the count in the gutter beside the paragraph about
// to be written. It is drawn rather than laid out so it cannot push the text.
func (r *publicationPDFRenderer) drawParagraphNumber() {
	page := r.pdf.PageNo()
	leftMargin, _ := r.margins(page)
	label := strconv.Itoa(r.paragraphNumber)
	r.pdf.SetFont(r.spec.FurnitureFont.ID, "", r.spec.FontSize*0.62)
	w := r.pdf.GetStringWidth(label)
	r.pdf.Text(r.trimX+leftMargin-w-6, r.y+r.spec.FontSize, label)
	r.pdf.SetFont(r.spec.Font.ID, "", r.spec.FontSize)
}
