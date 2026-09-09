package export

import (
	"fmt"

	"draftline/internal/types"
)

// PDF exports a fixed-layout reading edition using the shared publication
// document and embedded Unicode fonts.
func PDF(path string, book types.BookData, options types.PDFOptions) types.ExportResult {
	doc, err := BuildDocument(book, options.ExportOptions)
	if err != nil {
		return types.ExportResult{Success: false, Error: err.Error()}
	}
	data, err := renderPublicationPDF(doc, readingPDFSpec(options))
	if err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to render PDF: %v", err)}
	}
	if err := writeExportFile(path, data); err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}
	return types.ExportResult{Success: true, FilePath: path}
}

// generatePDF remains package-private test support while the exported path
// returns renderer errors to the application.
func generatePDF(book types.BookData, options types.PDFOptions) []byte {
	doc, err := BuildDocument(book, options.ExportOptions)
	if err != nil {
		return nil
	}
	data, _ := renderPublicationPDF(doc, readingPDFSpec(options))
	return data
}
