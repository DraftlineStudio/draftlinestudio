package export

import (
	"fmt"

	"draftline/internal/types"
)

// PrintPDF exports a printer-oriented interior using the same document and
// layout engine as the reading PDF. The preset changes page construction and
// furniture; it does not fork manuscript interpretation.
func PrintPDF(path string, book types.BookData, options types.PrintPDFOptions) types.ExportResult {
	doc, err := BuildDocument(book, options.ExportOptions)
	if err != nil {
		return types.ExportResult{Success: false, Error: err.Error()}
	}
	data, err := renderPublicationPDF(doc, printPDFSpec(options))
	if err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to render print PDF: %v", err)}
	}
	if err := writeExportFile(path, data); err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}
	return types.ExportResult{Success: true, FilePath: path}
}

func generatePrintPDF(book types.BookData, options types.PrintPDFOptions) []byte {
	doc, err := BuildDocument(book, options.ExportOptions)
	if err != nil {
		return nil
	}
	data, _ := renderPublicationPDF(doc, printPDFSpec(options))
	return data
}
