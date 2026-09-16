package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"time"

	"draftline/internal/types"
)

// EPUB exports a standards-oriented, reflowable edition from the same
// renderer-neutral document used by Draftline's PDF editions.
//
// cover is the artwork of the edition being exported, or nil. It is a
// parameter rather than a field on book because cover bytes never go on
// types.BookData: that struct crosses the Wails bridge as JSON on every save.
func EPUB(path string, book types.BookData, options types.EPUBOptions, cover *CoverArt) types.ExportResult {
	data, err := EPUBBytes(book, options, cover)
	if err != nil {
		return types.ExportResult{Success: false, Error: err.Error()}
	}
	if err := writeExportFile(path, data); err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}
	return types.ExportResult{Success: true, FilePath: path}
}

func renderEPUB(doc Document, book types.BookData, options types.EPUBOptions, cover *CoverArt, modified time.Time) ([]byte, error) {
	var output bytes.Buffer
	zw := zip.NewWriter(&output)

	mimetype, err := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	if err != nil {
		return nil, err
	}
	if _, err := mimetype.Write([]byte("application/epub+zip")); err != nil {
		return nil, err
	}

	entries, fonts := buildEPUBEntries(doc, book, options, cover, modified)
	for _, entry := range append(entries, fonts...) {
		// Cover artwork is already a compressed JPEG or PNG. Deflating it
		// again costs time and gains nothing, exactly as in the project
		// archive; everything else in an EPUB is text and compresses well.
		method := zip.Deflate
		if isPreCompressedEPUBEntry(entry.Name) {
			method = zip.Store
		}
		writer, err := zw.CreateHeader(&zip.FileHeader{Name: entry.Name, Method: method})
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("create %s: %w", entry.Name, err)
		}
		if _, err := writer.Write(entry.Data); err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("write %s: %w", entry.Name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("finalize archive: %w", err)
	}
	return output.Bytes(), nil
}

func isPreCompressedEPUBEntry(name string) bool {
	return strings.HasPrefix(name, "OEBPS/images/")
}
