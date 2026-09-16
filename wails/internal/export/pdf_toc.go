package export

import (
	"fmt"
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
	perPage := r.tocEntriesPerPage()
	pages := maxInt(1, int(math.Ceil(float64(entries)/float64(perPage))))
	for i := 0; i < pages; i++ {
		r.addPage(pageTOC, "Contents")
		r.tocPages = append(r.tocPages, r.pdf.PageNo())
	}
}

// tocEntriesPerPage is how many contents lines fit on the tightest reserved
// page — the first one, which gives three line-heights to the word "Contents".
// The reservation and the fill both go through it, so the number of pages set
// aside and the number of pages needed are worked out the same way.
func (r *publicationPDFRenderer) tocEntriesPerPage() int {
	usable := r.spec.TrimHeight - r.spec.TopMargin - r.spec.BottomMargin - r.spec.LineHeight*3
	return maxInt(1, int(usable/(r.spec.LineHeight*1.15)))
}

// tocOverflowError is what an oversubscribed contents produces instead of a
// short one.
//
// It is a guard, not a repair: the reservation and the fill both go through
// tocEntriesPerPage, and the first reserved page is the tightest of them, so a
// book cannot reach here by being long. What it stops is a later change to
// either arithmetic quietly reintroducing the older behaviour, which was to
// stop writing when the reserved pages ran out and leave the missing chapters
// invisible in a file on its way to a printer. Should the two ever fall out of
// step, an export fails with a number in it rather than succeeding with a
// short contents.
func tocOverflowError(remaining, listed, pages int) error {
	return fmt.Errorf(
		"the contents does not fit: %d of the %d entries could not be listed on the %d page%s set aside for it. Shorten the chapter titles, or turn the contents page off and let the book carry its own.",
		remaining, listed+remaining, pages, map[bool]string{true: "", false: "s"}[pages == 1])
}

func (r *publicationPDFRenderer) fillTOC() error {
	if len(r.tocPages) == 0 {
		return nil
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
			r.pdf.SetFont(r.spec.HeadingFont.ID, "B", r.spec.FontSize*1.65)
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
	if entryIndex < len(r.toc) {
		return tocOverflowError(len(r.toc)-entryIndex, entryIndex, len(r.tocPages))
	}
	return nil
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
