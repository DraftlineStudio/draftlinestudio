package export

import (
	"archive/zip"
	"fmt"
	"os"
	"strings"

	"draftline/internal/types"
)

// DOCX exports the book to DOCX format at the specified path.
func DOCX(path string, book types.BookData, options types.ExportOptions) types.ExportResult {
	// Create DOCX file (it's a ZIP archive with XML content)
	file, err := os.Create(path)
	if err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to create file: %v", err)}
	}

	zipWriter := zip.NewWriter(file)

	// Helper to write a file to the zip archive
	writeZipFile := func(name string, content []byte) error {
		w, err := zipWriter.Create(name)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", name, err)
		}
		if _, err := w.Write(content); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
		return nil
	}

	// Write [Content_Types].xml
	if err := writeZipFile("[Content_Types].xml", []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`)); err != nil {
		_ = zipWriter.Close()
		_ = file.Close()
		return types.ExportResult{Success: false, Error: err.Error()}
	}

	// Write _rels/.rels
	if err := writeZipFile("_rels/.rels", []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`)); err != nil {
		_ = zipWriter.Close()
		_ = file.Close()
		return types.ExportResult{Success: false, Error: err.Error()}
	}

	// Write word/_rels/document.xml.rels
	if err := writeZipFile("word/_rels/document.xml.rels", []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`)); err != nil {
		_ = zipWriter.Close()
		_ = file.Close()
		return types.ExportResult{Success: false, Error: err.Error()}
	}

	// Write word/styles.xml
	if err := writeZipFile("word/styles.xml", []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="Heading 1"/>
    <w:pPr><w:spacing w:before="480" w:after="240"/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="48"/></w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Normal">
    <w:name w:val="Normal"/>
    <w:pPr><w:spacing w:after="200" w:line="276" w:lineRule="auto"/></w:pPr>
    <w:rPr><w:sz w:val="24"/></w:rPr>
  </w:style>
</w:styles>`)); err != nil {
		_ = zipWriter.Close()
		_ = file.Close()
		return types.ExportResult{Success: false, Error: err.Error()}
	}

	// Build document content
	var docContent strings.Builder
	docContent.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
`)

	// Title page
	docContent.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="72"/></w:rPr><w:t>%s</w:t></w:r></w:p>
`, EscapeXML(book.Metadata.Title)))
	if book.Metadata.Author != "" {
		docContent.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:sz w:val="36"/></w:rPr><w:t>by %s</w:t></w:r></w:p>
`, EscapeXML(book.Metadata.Author)))
	}
	docContent.WriteString(`    <w:p><w:r><w:br w:type="page"/></w:r></w:p>
`)

	// Copyright
	if options.IncludeCopyright && book.Copyright != "" {
		text := HtmlToPlainParagraphs(book.Copyright)
		for _, para := range strings.Split(text, "\n") {
			if strings.TrimSpace(para) != "" {
				docContent.WriteString(fmt.Sprintf(`    <w:p><w:r><w:t>%s</w:t></w:r></w:p>
`, EscapeXML(para)))
			}
		}
		docContent.WriteString(`    <w:p><w:r><w:br w:type="page"/></w:r></w:p>
`)
	}

	// Front matter
	if options.IncludeFrontMatter {
		for _, ch := range book.FrontMatter {
			writeDocxChapter(&docContent, ch)
		}
	}

	// Body
	for _, ch := range book.Body {
		writeDocxChapter(&docContent, ch)
	}

	// Back matter
	if options.IncludeBackMatter {
		for _, ch := range book.BackMatter {
			writeDocxChapter(&docContent, ch)
		}
	}

	docContent.WriteString(`  </w:body>
</w:document>`)

	// Write the main document
	if err := writeZipFile("word/document.xml", []byte(docContent.String())); err != nil {
		_ = zipWriter.Close()
		_ = file.Close()
		return types.ExportResult{Success: false, Error: err.Error()}
	}

	// Close zip writer (flushes all data) - must check error
	if err := zipWriter.Close(); err != nil {
		_ = file.Close()
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to finalize DOCX: %v", err)}
	}

	// Close file
	if err := file.Close(); err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to close file: %v", err)}
	}

	return types.ExportResult{Success: true, FilePath: path}
}

func writeDocxChapter(sb *strings.Builder, ch types.ChapterItem) {
	// Chapter title
	sb.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>%s</w:t></w:r></w:p>
`, EscapeXML(ch.Title)))

	// Subtitle if present
	if ch.Subtitle != "" {
		sb.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:i/></w:rPr><w:t>%s</w:t></w:r></w:p>
`, EscapeXML(ch.Subtitle)))
	}

	// Content
	text := HtmlToPlainParagraphs(ch.Content)
	for _, para := range strings.Split(text, "\n") {
		if strings.TrimSpace(para) != "" {
			sb.WriteString(fmt.Sprintf(`    <w:p><w:r><w:t>%s</w:t></w:r></w:p>
`, EscapeXML(para)))
		}
	}

	// Page break after chapter
	sb.WriteString(`    <w:p><w:r><w:br w:type="page"/></w:r></w:p>
`)
}
