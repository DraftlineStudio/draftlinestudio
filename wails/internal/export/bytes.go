package export

import (
	"fmt"
	"strings"
	"time"

	"draftline/internal/types"
)

// The four exporters, as bytes.
//
// Every exporter already built its file in memory and handed it to
// writeExportFile at the end; these are that same work with the last step left
// off. A distribution bundle needs the bytes rather than a path, because the
// whole edition — several interiors and the artwork that goes with them — is
// written once, into one archive, and a half-built bundle must never reach
// disk. The path-taking functions above are these plus a write.

func EPUBBytes(book types.BookData, options types.EPUBOptions, cover *CoverArt) ([]byte, error) {
	doc, err := BuildDocument(book, options.ExportOptions)
	if err != nil {
		return nil, err
	}
	// A version chosen in the wizard belongs to an export with no edition
	// behind it; where there is an edition, its record has already said.
	if version := strings.TrimSpace(options.Version); version != "" && doc.Edition == nil {
		profile := epubProfileFor(version)
		doc.EPUBProfile = &profile
	}
	data, err := renderEPUB(doc, book, normalizeEPUBOptions(options), cover, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to render EPUB: %w", err)
	}
	return data, nil
}

func DOCXBytes(book types.BookData, options types.DOCXOptions) ([]byte, error) {
	doc, err := BuildDocument(book, options.ExportOptions)
	if err != nil {
		return nil, err
	}
	data, err := renderDOCX(doc, options)
	if err != nil {
		return nil, fmt.Errorf("failed to render DOCX: %w", err)
	}
	return data, nil
}

func PDFBytes(book types.BookData, options types.PDFOptions, cover *CoverArt) ([]byte, error) {
	doc, err := BuildDocument(book, options.ExportOptions)
	if err != nil {
		return nil, err
	}
	spec := readingPDFSpec(options)
	spec.Cover = cover
	data, err := renderPublicationPDF(doc, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to render PDF: %w", err)
	}
	return data, nil
}

func PrintPDFBytes(book types.BookData, options types.PrintPDFOptions) ([]byte, error) {
	// The trim is checked before anything is rendered, because a trim can now
	// arrive from a stored edition record rather than being typed in this
	// session, and a page ninety-nine inches tall is a file nobody can print
	// and nobody would notice until a printer refused it.
	if err := validateTrim(options); err != nil {
		return nil, err
	}
	doc, err := BuildDocument(book, options.ExportOptions)
	if err != nil {
		return nil, err
	}
	spec := printPDFSpec(options)
	if err := validateKDPPreflight(doc, options, spec); err != nil {
		return nil, err
	}
	data, err := renderPublicationPDFChecked(doc, spec, func(pages int) error {
		return validateKDPPageCount(doc, options, pages)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render print PDF: %w", err)
	}
	return data, nil
}
