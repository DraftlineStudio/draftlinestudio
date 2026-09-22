package export

import (
	"fmt"

	"draftline/internal/types"
)

// PDF exports a fixed-layout reading edition using the shared publication
// document and embedded Unicode fonts.
//
// cover is the edition's artwork, or nil. fpdf embeds PNG, JPEG and GIF only,
// which is one more reason a Draftline cover is a JPEG unless a PNG is
// genuinely smaller.
func PDF(path string, book types.BookData, options types.PDFOptions, cover *CoverArt) types.ExportResult {
	data, err := PDFBytes(book, options, cover)
	if err != nil {
		return types.ExportResult{Success: false, Error: err.Error()}
	}
	if err := writeExportFile(path, data); err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}
	return types.ExportResult{Success: true, FilePath: path}
}
