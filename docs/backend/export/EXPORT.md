# Export Package

`internal/export/` handles multi-format book export (EPUB, DOCX, PDF, Print PDF). Rebuilt in 0.18 around a shared document model.

## Shared Document Model

All formats render from the same renderer-neutral document (`document.go`): `BuildDocument(book, options)` selects the requested archive sections and parses their HTML via `ParseDocumentHTML` into semantic blocks, which each format's renderer then lays out.

## Functions

### EPUB(path string, book types.BookData, options types.EPUBOptions) types.ExportResult
Exports a reflowable EPUB edition rendered from the shared document (`epub_render.go`), with optional font embedding (`epub_fonts.go`).

### DOCX(path string, book types.BookData, options types.ExportOptions) types.ExportResult
Exports to Microsoft Word format from the shared document (`docx_render.go`). Creates a ZIP archive with Office Open XML content.

### PDF(path string, book types.BookData, options types.PDFOptions) types.ExportResult
Exports a fixed-layout reading PDF with embedded Unicode fonts (`pdf_fonts.go`, subset per output — no machine-local font dependency).

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

`PDF` and `PrintPDF` share one pipeline: `BuildDocument(...)` → `renderPublicationPDF(doc, spec)`. The layout specs (page size, margins, furniture) live in `pdf_spec.go` (`readingPDFSpec` / `printPDFSpec`); the shared renderer in `pdf_renderer.go` handles pagination, the table of contents (`pdf_toc.go`), and code-block rendering (`pdf_code.go`). The print preset changes page construction only — it does not fork manuscript interpretation.
- PDF object generation and cross-reference tables
