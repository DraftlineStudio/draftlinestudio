package export

import (
	"archive/zip"
	"bytes"
	"fmt"

	"draftline/internal/types"
)

// DOCX exports an editable Word edition from the shared publication document.
func DOCX(path string, book types.BookData, options types.ExportOptions) types.ExportResult {
	doc, err := BuildDocument(book, options)
	if err != nil {
		return types.ExportResult{Success: false, Error: err.Error()}
	}
	data, err := renderDOCX(doc)
	if err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to render DOCX: %v", err)}
	}
	if err := writeExportFile(path, data); err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}
	return types.ExportResult{Success: true, FilePath: path}
}

func renderDOCX(doc Document) ([]byte, error) {
	rels := collectDOCXLinks(doc)
	parts := buildDOCXParts(doc, rels)
	var output bytes.Buffer
	zw := zip.NewWriter(&output)
	for _, part := range parts {
		writer, err := zw.Create(part.Name)
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("create %s: %w", part.Name, err)
		}
		if _, err := writer.Write(part.Data); err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("write %s: %w", part.Name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("finalize archive: %w", err)
	}
	return output.Bytes(), nil
}
