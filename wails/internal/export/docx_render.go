package export

import (
	"fmt"
	"sort"
	"strings"
)

func collectDOCXLinks(doc Document) map[string]string {
	unique := map[string]struct{}{}
	for _, section := range doc.Sections {
		for _, block := range section.Blocks {
			for _, run := range block.Runs {
				if href := safeExportHref(run.Href); href != "" && !strings.HasPrefix(href, "#") {
					unique[href] = struct{}{}
				}
			}
		}
	}
	hrefs := make([]string, 0, len(unique))
	for href := range unique {
		hrefs = append(hrefs, href)
	}
	sort.Strings(hrefs)
	links := make(map[string]string, len(hrefs))
	for index, href := range hrefs {
		links[href] = fmt.Sprintf("rIdLink%d", index+1)
	}
	return links
}

func renderDOCXDocument(doc Document, links map[string]string) string {
	var out strings.Builder
	out.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:body>
`)
	writeDOCXParagraph(&out, "Title", "center", []DocumentRun{{Text: doc.Title}}, nil)
	if strings.TrimSpace(doc.Author) != "" {
		writeDOCXParagraph(&out, "Subtitle", "center", []DocumentRun{{Text: "by " + doc.Author}}, nil)
	}
	writeDOCXPageBreak(&out)
	for index, section := range doc.Sections {
		writeDOCXSection(&out, section, links)
		if index < len(doc.Sections)-1 {
			writeDOCXPageBreak(&out)
		}
	}
	out.WriteString(`    <w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>
  </w:body>
</w:document>`)
	return out.String()
}

func writeDOCXSection(out *strings.Builder, section DocumentSection, links map[string]string) {
	if strings.TrimSpace(section.Title) != "" {
		writeDOCXParagraph(out, "Heading1", "", []DocumentRun{{Text: section.Title}}, links)
	}
	if strings.TrimSpace(section.Subtitle) != "" {
		writeDOCXParagraph(out, "Subtitle", "center", []DocumentRun{{Text: section.Subtitle}}, links)
	}
	for _, block := range section.Blocks {
		style := "Normal"
		extra := ""
		switch block.Kind {
		case BlockHeading:
			style = "Heading2"
		case BlockBlockquote:
			style = "Quote"
		case BlockCode:
			style = "Code"
		case BlockSceneBreak:
			writeDOCXParagraph(out, "Normal", "center", []DocumentRun{{Text: "⁂"}}, links)
			continue
		case BlockListItem:
			numID := 1
			if block.Ordered {
				numID = 2
			}
			level := block.ListDepth - 1
			if level < 0 {
				level = 0
			}
			if level > 8 {
				level = 8
			}
			extra = fmt.Sprintf("<w:numPr><w:ilvl w:val=\"%d\"/><w:numId w:val=\"%d\"/></w:numPr>", level, numID)
		}
		writeDOCXParagraph(out, style, block.Alignment, block.Runs, links, extra)
	}
}

func writeDOCXParagraph(out *strings.Builder, style, alignment string, runs []DocumentRun, links map[string]string, extras ...string) {
	out.WriteString("    <w:p><w:pPr>")
	if style != "" {
		fmt.Fprintf(out, "<w:pStyle w:val=\"%s\"/>", style)
	}
	if alignment = normalizedAlignment(alignment, ""); alignment != "" {
		fmt.Fprintf(out, "<w:jc w:val=\"%s\"/>", alignment)
	}
	for _, extra := range extras {
		out.WriteString(extra)
	}
	out.WriteString("</w:pPr>")
	for _, run := range runs {
		writeDOCXRun(out, run, links)
	}
	out.WriteString("</w:p>\n")
}

func writeDOCXRun(out *strings.Builder, run DocumentRun, links map[string]string) {
	if run.LineBreak {
		out.WriteString("<w:r><w:br/></w:r>")
		return
	}
	if run.Text == "" {
		return
	}
	href := safeExportHref(run.Href)
	if id := links[href]; id != "" {
		fmt.Fprintf(out, "<w:hyperlink r:id=\"%s\">", id)
	}
	out.WriteString("<w:r><w:rPr>")
	if run.Bold {
		out.WriteString("<w:b/>")
	}
	if run.Italic {
		out.WriteString("<w:i/>")
	}
	if run.Underline {
		out.WriteString("<w:u w:val=\"single\"/>")
	}
	if run.Strike {
		out.WriteString("<w:strike/>")
	}
	if run.Superscript {
		out.WriteString("<w:vertAlign w:val=\"superscript\"/>")
	}
	if run.Subscript {
		out.WriteString("<w:vertAlign w:val=\"subscript\"/>")
	}
	if run.Code {
		out.WriteString("<w:rFonts w:ascii=\"Courier New\" w:hAnsi=\"Courier New\"/>")
	}
	out.WriteString("</w:rPr><w:t xml:space=\"preserve\">")
	out.WriteString(EscapeXML(run.Text))
	out.WriteString("</w:t></w:r>")
	if links[href] != "" {
		out.WriteString("</w:hyperlink>")
	}
}

func writeDOCXPageBreak(out *strings.Builder) {
	out.WriteString("    <w:p><w:r><w:br w:type=\"page\"/></w:r></w:p>\n")
}
