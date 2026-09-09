package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"time"

	"draftline/internal/types"
)

// EPUB exports a standards-oriented, reflowable edition from the same
// renderer-neutral document used by Draftline's PDF editions.
func EPUB(path string, book types.BookData, options types.EPUBOptions) types.ExportResult {
	doc, err := BuildDocument(book, options.ExportOptions)
	if err != nil {
		return types.ExportResult{Success: false, Error: err.Error()}
	}
	data, err := renderEPUB(doc, book, normalizeEPUBOptions(options), time.Now().UTC())
	if err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to render EPUB: %v", err)}
	}
	if err := writeExportFile(path, data); err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}
	return types.ExportResult{Success: true, FilePath: path}
}

func renderEPUB(doc Document, book types.BookData, options types.EPUBOptions, modified time.Time) ([]byte, error) {
	var output bytes.Buffer
	zw := zip.NewWriter(&output)

	mimetype, err := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	if err != nil {
		return nil, err
	}
	if _, err := mimetype.Write([]byte("application/epub+zip")); err != nil {
		return nil, err
	}

	entries, fonts := buildEPUBEntries(doc, book, options, modified)
	for _, entry := range append(entries, fonts...) {
		writer, err := zw.CreateHeader(&zip.FileHeader{Name: entry.Name, Method: zip.Deflate})
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
