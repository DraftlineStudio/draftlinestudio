package export

import (
	"math"
	"strconv"
	"strings"

	"codeberg.org/go-pdf/fpdf"
)

func (r *publicationPDFRenderer) reserveTOC() {
	entries := 0
	for _, section := range r.doc.Sections {
		if section.Role == SectionBody || section.Role == SectionBack {
			entries++
		}
	}
	usable := r.spec.TrimHeight - r.spec.TopMargin - r.spec.BottomMargin - r.spec.LineHeight*3
	perPage := maxInt(1, int(usable/(r.spec.LineHeight*1.15)))
	pages := maxInt(1, int(math.Ceil(float64(entries)/float64(perPage))))
	for i := 0; i < pages; i++ {
		r.addPage(pageTOC, "Contents")
		r.tocPages = append(r.tocPages, r.pdf.PageNo())
	}
}

func (r *publicationPDFRenderer) fillTOC() {
	if len(r.tocPages) == 0 {
		return
	}
	// SetPage changes fpdf's active page. Restore the final manuscript page
	// after backfilling the reserved contents pages so Close() finalizes the
	// actual end of the book rather than the last TOC page.
	lastPage := r.pdf.PageCount()
	defer r.pdf.SetPage(lastPage)
	entryIndex := 0
	for pageIndex, page := range r.tocPages {
		r.pdf.SetPage(page)
		leftMargin, rightMargin := r.margins(page)
		left := r.trimX + leftMargin
		width := r.spec.TrimWidth - leftMargin - rightMargin
		y := r.trimY + r.spec.TopMargin
		if pageIndex == 0 {
			r.pdf.SetFont(r.spec.Font.ID, "B", r.spec.FontSize*1.65)
			title := "Contents"
			r.pdf.Text(left+(width-r.pdf.GetStringWidth(title))/2, y+r.spec.FontSize*1.65, title)
			y += r.spec.LineHeight * 3
		}
		bottom := r.trimY + r.spec.TrimHeight - r.spec.BottomMargin
		for entryIndex < len(r.toc) && y+r.spec.LineHeight <= bottom {
			entry := r.toc[entryIndex]
			r.pdf.SetFont(r.spec.Font.ID, "", r.spec.FontSize)
			pageText := strconv.Itoa(entry.Page)
			pageWidth := r.pdf.GetStringWidth(pageText)
			maxTitleWidth := width - pageWidth - r.spec.FontSize*2
			title := truncateToWidth(r.pdf, entry.Title, maxTitleWidth)
			r.pdf.Text(left, y+r.spec.FontSize, title)
			r.pdf.Text(left+width-pageWidth, y+r.spec.FontSize, pageText)
			y += r.spec.LineHeight * 1.15
			entryIndex++
		}
	}
}

func truncateToWidth(pdf *fpdf.Fpdf, text string, width float64) string {
	if pdf.GetStringWidth(text) <= width {
		return text
	}
	runes := []rune(text)
	for len(runes) > 0 && pdf.GetStringWidth(string(runes)+"\u2026") > width {
		runes = runes[:len(runes)-1]
	}
	return strings.TrimSpace(string(runes)) + "\u2026"
}
