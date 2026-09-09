package export

import "strings"

func documentUsesCode(doc Document) bool {
	for _, section := range doc.Sections {
		for _, block := range section.Blocks {
			if block.Kind == BlockCode {
				return true
			}
			for _, run := range block.Runs {
				if run.Code {
					return true
				}
			}
		}
	}
	return false
}

func (r *publicationPDFRenderer) renderCodeBlock(block DocumentBlock) {
	left, bodyWidth, _ := r.bodyBounds()
	outerInset := 0.18 * pointsPerInch
	padding := 0.12 * pointsPerInch
	boxLeft := left + outerInset
	boxWidth := bodyWidth - 2*outerInset
	contentLeft := boxLeft + padding
	contentWidth := boxWidth - 2*padding
	if contentWidth <= r.spec.FontSize*8 {
		return
	}

	fontSize := r.spec.FontSize * 0.86
	if fontSize < 7.5 {
		fontSize = 7.5
	}
	lineHeight := r.spec.LineHeight * 0.9
	if lineHeight < fontSize*1.3 {
		lineHeight = fontSize * 1.3
	}
	r.pdf.SetFont(r.spec.CodeFont.ID, "", fontSize)
	lines := r.wrapCodeText(block.PlainText(), contentWidth)
	if len(lines) == 0 {
		return
	}

	drawPadding := func() {
		r.pdf.SetFillColor(244, 244, 242)
		r.pdf.Rect(boxLeft, r.y, boxWidth, padding, "F")
		r.y += padding
	}
	r.ensureSpace(padding + lineHeight + padding)
	drawPadding()
	align := normalizedAlignment(block.Alignment, "left")
	for _, line := range lines {
		previousPage := r.pdf.PageNo()
		r.ensureSpace(lineHeight + padding)
		if r.pdf.PageNo() != previousPage {
			r.ensureSpace(padding + lineHeight + padding)
			drawPadding()
		}
		r.pdf.SetFillColor(244, 244, 242)
		r.pdf.Rect(boxLeft, r.y, boxWidth, lineHeight, "F")
		r.pdf.SetFont(r.spec.CodeFont.ID, "", fontSize)
		textWidth := r.pdf.GetStringWidth(line)
		x := contentLeft
		if align == "center" {
			x += (contentWidth - textWidth) / 2
		} else if align == "right" {
			x += contentWidth - textWidth
		}
		if x < contentLeft {
			x = contentLeft
		}
		r.pdf.Text(x, r.y+fontSize, line)
		r.y += lineHeight
	}
	drawPadding()
	r.y += r.spec.LineHeight * 0.35
}

func (r *publicationPDFRenderer) wrapCodeText(text string, width float64) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "\t", "    ")
	sourceLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(sourceLines))
	for _, source := range sourceLines {
		if source == "" {
			lines = append(lines, "")
			continue
		}
		var current strings.Builder
		for _, char := range source {
			candidate := current.String() + string(char)
			if current.Len() > 0 && r.pdf.GetStringWidth(candidate) > width {
				lines = append(lines, current.String())
				current.Reset()
			}
			current.WriteRune(char)
		}
		lines = append(lines, current.String())
	}
	return lines
}
