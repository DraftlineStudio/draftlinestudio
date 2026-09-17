package export

// Exporting from a registered edition.
//
// Until now an export knew only the book: a title, an author, and whichever
// ISBN happened to be first in the book's identifier list. An author who
// publishes the same book twice has two ISBNs, two copyright years and two
// covers, and the file that goes to a retailer has to carry the right one of
// each. The publishing record already holds all of it (internal/types/
// editions.go); this file is how an exporter reads it.
//
// Two rules:
//
//   - Nothing here decides anything the record does not say. An edition with
//     no ISBN produces no ISBN, not a placeholder; a format with no cover
//     produces no cover page.
//   - Bytes arrive as a parameter, never on types.BookData. Cover art is the
//     one thing an exporter needs that is too big to cross the Wails bridge,
//     so the application reads it out of the project file and hands it to the
//     exporter directly. See wails/cover.go.

import (
	"strings"
	"time"

	"draftline/internal/types"
)

// CoverArt is the artwork an exporter puts on the front of the book. It is the
// derivative Draftline made when the cover was attached — 1600 x 2560, sRGB,
// front cover only — not the print-ready original, which stays on the author's
// disk.
type CoverArt struct {
	Data []byte
	// MediaType is "image/jpeg" or "image/png"; FileName is the name the
	// artwork is filed under inside the exported package.
	MediaType string
	FileName  string
	Width     int
	Height    int
}

// Usable reports whether there is artwork to put on a cover page. A record
// with no bytes behind it is not a cover.
func (c *CoverArt) Usable() bool { return c != nil && len(c.Data) > 0 }

// ImageType is the three-letter word fpdf wants for an embedded image. fpdf
// accepts PNG, JPG and GIF only, which is one more reason Draftline keeps a
// cover as a JPEG unless a PNG is genuinely smaller.
func (c *CoverArt) ImageType() string {
	if c != nil && c.MediaType == "image/png" {
		return "PNG"
	}
	return "JPG"
}

// DocumentEdition is what one selected format of one edition tells an
// exporter. It is flattened out of the records deliberately: a renderer should
// not have to walk an edition index to find out what to print.
type DocumentEdition struct {
	EditionID string
	FormatID  string
	// Label is the edition's own words — "First edition" — and Statement is
	// the longer title-page wording when the format carries one.
	Label     string
	Statement string
	// ISBN is stored as the author typed it, hyphens and all. Identifier is
	// the same number as a URN, with the hyphens taken out, which is the form
	// a package document declares. Identifier is empty when there is no ISBN.
	ISBN       string
	Identifier string
	Imprint    string
	// Rights is the rights line as a sentence, and Date the publication date
	// as the record holds it.
	Rights string
	Date   string
	// EPUB is the package profile this format asks for.
	EPUB epubProfile
	// Copyright is the generated copyright page, line by line.
	Copyright []string
	// Trim, PageCount and Spine describe the printed object. Spine is derived
	// from the other two and the stock, never stored.
	Trim      string
	PageCount string
	Spine     string
	HasCover  bool
}

// selectEdition finds the format an export was asked for and flattens it.
//
// It returns nil for an export that named no edition, and for one that named
// a format the book no longer holds — a record deleted between opening the
// wizard and finishing it. The export then goes ahead from the book alone
// rather than failing,
// because the author asked for a file, not for a lecture about a record.
func selectEdition(book types.BookData, options types.ExportOptions) *DocumentEdition {
	formatID := strings.TrimSpace(options.FormatID)
	if formatID == "" || book.Editions == nil {
		return nil
	}
	edition, format, ok := book.Editions.FindFormat(formatID)
	if !ok {
		return nil
	}

	selected := &DocumentEdition{
		EditionID: edition.ID,
		FormatID:  format.ID,
		Label:     strings.TrimSpace(edition.Label),
		Statement: strings.TrimSpace(format.EditionStatement),
		ISBN:      strings.TrimSpace(format.ISBN13),
		Imprint:   strings.TrimSpace(format.ImprintOfRecord),
		Rights:    types.RightsSentence(format),
		Date:      w3cdtfDate(format.PublicationDate),
		EPUB:      epubProfileFor(format.EPUBVersion),
		Copyright: types.CopyrightLines(book.Metadata, edition, format, book.Editions.PriorYears(edition.ID)),
		Trim:      strings.TrimSpace(format.Trim),
		PageCount: strings.TrimSpace(format.PageCount),
		Spine:     types.SpineWidthLabel(format),
		HasCover:  edition.Cover != nil,
	}
	if digits := types.NormalizeISBN(format.ISBN13); digits != "" {
		selected.Identifier = "urn:isbn:" + digits
	}
	return selected
}

// w3cdtfDate is the publication date in the only form a package document may
// state it in.
//
// dc:date is constrained to W3CDTF — a year, a year and month, a calendar day,
// or a full timestamp — and a reading system or a validator will reject
// anything else. The Editions screen's publication date is a free text field,
// because an author with a contract that says "Spring 2027" should be able to
// write that down. So the record keeps what was typed and the file declares
// only what it is allowed to declare: a date it cannot parse produces no
// dc:date at all, which is a book with no stated publication date rather than
// a book no shop will accept.
func w3cdtfDate(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	for _, layout := range []string{"2006", "2006-01", "2006-01-02", "2006-01-02T15:04:05Z07:00", "2006-01-02T15:04Z07:00"} {
		if _, err := time.Parse(layout, text); err == nil {
			return text
		}
	}
	return ""
}

// ── The EPUB profile ───────────────────────────────────────────────────────

// epubProfile is which EPUB an export writes.
//
// PackageVersion is the value of the package element's version attribute, and
// it has only ever had two legal values: "2.0" and "3.0". EPUB 3.3 is a
// revision of EPUB 3 and declares itself 3.0 like every other 3.x — a package
// claiming version="3.3" is refused by EPUBCheck. The revision the author
// chose is therefore recorded where a revision belongs, in a dcterms:conformsTo
// metadatum naming the specification, so the file says which EPUB it was
// written to without lying about which EPUB it is.
type epubProfile struct {
	PackageVersion string
	Revision       string
	// Three is false only for EPUB 2.0.1, where the navigation document is an
	// NCX rather than a nav document, the manifest carries no properties, and
	// the refinement metadata of EPUB 3 is not allowed.
	Three bool
	// ConformsTo is the specification URL, empty for EPUB 2.
	ConformsTo string
}

const (
	epubRevision33  = "3.3"
	epubRevision30  = "3.0"
	epubRevision201 = "2.0.1"
)

// defaultEPUBProfile is what every export produced before an edition could ask
// for anything else, and what an export with no edition still produces.
func defaultEPUBProfile() epubProfile {
	return epubProfile{PackageVersion: "3.0", Revision: epubRevision33, Three: true, ConformsTo: "https://www.w3.org/TR/epub-33/"}
}

// epubProfileFor reads the Specification field of an ebook format. The words
// are the design brief's — "EPUB 3.3", "EPUB 3.0", "EPUB 2.0.1" — and an
// unrecognised value falls back to the current revision rather than refusing,
// because a record written by a later Draftline is not an error.
func epubProfileFor(name string) epubProfile {
	digits := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(name), "EPUB"))
	digits = strings.TrimSpace(digits)
	switch digits {
	case epubRevision201, "2.0", "2":
		return epubProfile{PackageVersion: "2.0", Revision: epubRevision201, Three: false}
	case epubRevision30, "3":
		return epubProfile{PackageVersion: "3.0", Revision: epubRevision30, Three: true, ConformsTo: "https://www.w3.org/TR/epub-30/"}
	default:
		return defaultEPUBProfile()
	}
}

// documentEPUBProfile is the profile a document was built for.
func documentEPUBProfile(doc Document) epubProfile {
	// An export made from no edition has no record to read a version out of,
	// so the wizard's own answer is carried on the document instead. It is
	// checked first: an author who picked a version meant it.
	if doc.EPUBProfile != nil {
		return *doc.EPUBProfile
	}
	if doc.Edition != nil && doc.Edition.EPUB.PackageVersion != "" {
		return doc.Edition.EPUB
	}
	return defaultEPUBProfile()
}
