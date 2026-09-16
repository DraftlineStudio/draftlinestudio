package export

import (
	"fmt"

	"draftline/internal/types"
)

// PrintPDF exports a printer-oriented interior using the same document and
// layout engine as the reading PDF. The preset changes page construction and
// furniture; it does not fork manuscript interpretation.
//
// It takes no cover, and that is deliberate. A print-on-demand service wants
// the interior alone and the wrap as a separate file; a cover bound into the
// interior becomes page one of the printed block. Draftline's cover derivative
// is a 1600 by 2560 front cover with no spine, no back and no bleed — 5.33 by
// 8.53 inches at 300 dpi, smaller than the 6 by 9 page this exporter sets —
// so putting it here would be printing a downsized cover at print size, which
// is the one thing the cover pipeline exists to refuse. The reading PDF, which
// is a file an author sends to a reviewer rather than to a printer, does carry
// it.
func PrintPDF(path string, book types.BookData, options types.PrintPDFOptions) types.ExportResult {
	// The trim is checked before anything is rendered, because a trim can now
	// arrive from a stored edition record rather than being typed in this
	// session, and a page ninety-nine inches tall is a file nobody can print
	// and nobody would notice until a printer refused it.
	data, err := PrintPDFBytes(book, options)
	if err != nil {
		return types.ExportResult{Success: false, Error: err.Error()}
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
