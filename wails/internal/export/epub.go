package export

import (
	"archive/zip"
	"fmt"
	"os"
	"time"

	"draftline/internal/types"
)

// EPUB exports the book to EPUB format at the specified path.
func EPUB(path string, book types.BookData, options types.ExportOptions) types.ExportResult {
	// Create EPUB file (it's a ZIP archive)
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

	// Write mimetype (must be first, uncompressed)
	mimetypeWriter, err := zipWriter.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	if err != nil {
		_ = zipWriter.Close()
		_ = file.Close()
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to create mimetype: %v", err)}
	}
	if _, err := mimetypeWriter.Write([]byte("application/epub+zip")); err != nil {
		_ = zipWriter.Close()
		_ = file.Close()
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write mimetype: %v", err)}
	}

	// Write META-INF/container.xml
	if err := writeZipFile("META-INF/container.xml", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)); err != nil {
		_ = zipWriter.Close()
		_ = file.Close()
		return types.ExportResult{Success: false, Error: err.Error()}
	}

	// Collect chapters to include
	chapters := make([]types.ChapterItem, 0)
	chapterIDs := make([]string, 0)

	if options.IncludeCopyright && book.Copyright != "" {
		chapters = append(chapters, types.ChapterItem{Title: "Copyright", Type: "Copyright", Content: book.Copyright})
		chapterIDs = append(chapterIDs, "copyright")
	}
	if options.IncludeFrontMatter {
		for i, ch := range book.FrontMatter {
			chapters = append(chapters, ch)
			chapterIDs = append(chapterIDs, fmt.Sprintf("front%d", i))
		}
	}
	for i, ch := range book.Body {
		chapters = append(chapters, ch)
		chapterIDs = append(chapterIDs, fmt.Sprintf("chapter%d", i))
	}
	if options.IncludeBackMatter {
		for i, ch := range book.BackMatter {
			chapters = append(chapters, ch)
			chapterIDs = append(chapterIDs, fmt.Sprintf("back%d", i))
		}
	}

	// Write content.opf
	var manifestItems, spineItems string
	for _, id := range chapterIDs {
		manifestItems += fmt.Sprintf(`    <item id="%s" href="%s.xhtml" media-type="application/xhtml+xml"/>
`, id, id)
		spineItems += fmt.Sprintf(`    <itemref idref="%s"/>
`, id)
	}

	opfContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="uid">urn:uuid:%s</dc:identifier>
    <dc:title>%s</dc:title>
    <dc:creator>%s</dc:creator>
    <dc:publisher>%s</dc:publisher>
    <dc:language>en</dc:language>
    <meta property="dcterms:modified">%s</meta>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
%s  </manifest>
  <spine>
%s  </spine>
</package>`,
		GenerateUUID(),
		EscapeXML(book.Metadata.Title),
		EscapeXML(book.Metadata.Author),
		EscapeXML(book.Metadata.Publisher),
		time.Now().Format("2006-01-02T15:04:05Z"),
		manifestItems,
		spineItems)

	if err := writeZipFile("OEBPS/content.opf", []byte(opfContent)); err != nil {
		_ = zipWriter.Close()
		_ = file.Close()
		return types.ExportResult{Success: false, Error: err.Error()}
	}

	// Write nav.xhtml (table of contents)
	var tocItems string
	for i, ch := range chapters {
		tocItems += fmt.Sprintf(`      <li><a href="%s.xhtml">%s</a></li>
`, chapterIDs[i], EscapeXML(ch.Title))
	}

	navContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head>
  <title>Table of Contents</title>
</head>
<body>
  <nav epub:type="toc">
    <h1>Table of Contents</h1>
    <ol>
%s    </ol>
  </nav>
</body>
</html>`, tocItems)

	if err := writeZipFile("OEBPS/nav.xhtml", []byte(navContent)); err != nil {
		_ = zipWriter.Close()
		_ = file.Close()
		return types.ExportResult{Success: false, Error: err.Error()}
	}

	// Write each chapter
	for i, ch := range chapters {
		content := ch.Content
		if content == "" {
			content = "<p></p>"
		}
		chapterContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>%s</title>
</head>
<body>
  <h1>%s</h1>
  %s
</body>
</html>`, EscapeXML(ch.Title), EscapeXML(ch.Title), content)

		if err := writeZipFile(fmt.Sprintf("OEBPS/%s.xhtml", chapterIDs[i]), []byte(chapterContent)); err != nil {
			_ = zipWriter.Close()
			_ = file.Close()
			return types.ExportResult{Success: false, Error: err.Error()}
		}
	}

	// Close zip writer (flushes all data) - must check error
	if err := zipWriter.Close(); err != nil {
		_ = file.Close()
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to finalize EPUB: %v", err)}
	}

	// Close file
	if err := file.Close(); err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to close file: %v", err)}
	}

	return types.ExportResult{Success: true, FilePath: path}
}
