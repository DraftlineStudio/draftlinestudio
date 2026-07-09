# Export Package

`internal/export/` handles multi-format book export (EPUB, DOCX, PDF, Print PDF).

## Functions

### EPUB(path string, book types.BookData, options types.ExportOptions) types.ExportResult
Exports to EPUB 3.0 format. Creates a valid ZIP archive with XHTML content, CSS styling, and required metadata files.

### DOCX(path string, book types.BookData, options types.ExportOptions) types.ExportResult
Exports to Microsoft Word format. Creates a ZIP archive with Office Open XML content.

### PDF(path string, book types.BookData, options types.PDFOptions) types.ExportResult
Exports to standard PDF format (Letter size, 8.5x11"). Generates raw PDF with Helvetica font, proper pagination, and chapter breaks.

### PrintPDF(path string, book types.BookData, options types.PrintPDFOptions) types.ExportResult
Exports to print-ready PDF with professional formatting:
- Configurable trim sizes (5x8, 5.25x8, 5.5x8.5, 6x9)
- Bleed area (0.125")
- Crop marks
- Mirrored margins for binding
- Running headers with author/title
- Chapter drop caps
- Table of contents

## Helper Functions

### HtmlToPlainParagraphs(html string) string
Strips HTML tags and returns plain text with paragraph breaks.

### EscapePDFString(s string) string
Escapes special characters for PDF string literals.

### EscapeXML(s string) string
Escapes special characters for XML/XHTML content.

### GenerateUUID() string
Generates a random UUID for EPUB identifiers.

## PDF Internals

Both PDF functions use custom `pdfWriter` / `printPDFWriter` structs that handle:
- Multi-page layout with automatic page breaks
- Text wrapping and word-wrap
- Font sizing and line height calculation
- PDF object generation and cross-reference tables
