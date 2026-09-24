package types

import (
	"strconv"
	"strings"
)

// Editions: the publishing record of one book.
//
// A book is written once and published several times. Each publication is an
// edition, and an edition appears in several formats — an ebook, a paperback,
// a hardcover, an audiobook — each of which carries its own ISBN and its own
// specification. That is the shape a retailer, a distributor and a copyright
// page all assume, and it is the shape recorded here.
//
// Two rules run through the whole file:
//
//   - An ISBN is fixed once it is registered. Changing the trim size, the
//     cover or the publisher produces a NEW edition record. Nothing here
//     rewrites a published format for the author.
//   - Anything that can be computed is computed. The ISBN-10 form of an
//     ISBN-13 and the spine width of a printed book are derivations, not
//     stored truth, so they cannot go stale against the fields they come from.
//
// The records are small JSON and live in the archive as editions/index.json.
// Cover images and frozen manuscripts do not live here: they are bytes, and
// this struct crosses the Wails bridge as JSON every time the book is saved.

// Format kinds. The kind decides which specification a format has: page count
// and paper stock for print, an EPUB version and a layout for an ebook.
const (
	EditionKindEbook = "ebook"
	EditionKindPrint = "print"
	EditionKindAudio = "audio"
)

// EditionIndex is the whole publishing record for one book.
//
// Version is a hard gate, not a hint. A file written by a newer Draftline is
// refused rather than read loosely, because a save rebuilds the archive and a
// loose reading would be written back over the original.
type EditionIndex struct {
	Version  int       `json:"version"`
	Editions []Edition `json:"editions"`
	// Snapshots is the catalogue of frozen manuscripts this book holds — the
	// text each published ISBN actually went out with. The records are small;
	// the text they describe lives under editions/snapshots/ and never joins
	// this struct. See snapshot.go.
	Snapshots []EditionSnapshot `json:"snapshots,omitempty"`
}

// Edition is one publication of the book: the first edition, a revised second
// edition, a large-print reissue. Its formats share its copyright year, its
// cover and its revision note; they differ in ISBN and specification.
type Edition struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Year is the copyright year this edition establishes. The copyright page
	// prints it after the years of every edition it supersedes.
	Year   string `json:"year"`
	Status string `json:"status"`
	// CoverID names the cover art attached to this edition. Cover bytes are
	// not stored here; this is the identifier they are filed under, and it
	// mirrors Cover.ID whenever Cover is set.
	CoverID string `json:"cover_id,omitempty"`
	// Cover is the record of the attached artwork: its size, what was done to
	// it, and where the print-ready original was. Still no bytes - see
	// cover.go. Nil until a cover is attached.
	Cover *EditionCover `json:"cover,omitempty"`
	// PreviousEditionID is the edition this one supersedes. It is what makes
	// the copyright year list cumulative and what marks an edition as later,
	// which is the only condition under which a revision note is printed.
	PreviousEditionID string          `json:"previous_edition_id,omitempty"`
	RevisionNote      string          `json:"revision_note,omitempty"`
	Formats           []EditionFormat `json:"formats"`
}

// EditionFormat is one saleable object: one ISBN, one specification, one
// price. The field names are the ones the design brief uses; renaming them
// here would put the screen and the record out of step.
type EditionFormat struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	// Format is the printed word: eBook, Paperback, Hardcover, Large print.
	// The copyright page prints it lower-cased inside the ISBN line.
	Format string `json:"format,omitempty"`
	// ISBN13 is stored exactly as the author typed it, hyphens and all,
	// because an agency-issued ISBN arrives hyphenated and the author knows
	// where the groups fall. Draftline never regroups it.
	ISBN13 string `json:"isbn13,omitempty"`
	// Registration records where the number came from: an agency, a retailer,
	// or nowhere yet.
	Registration string `json:"registration,omitempty"`
	// EditionStatement is the wording printed on the title page.
	EditionStatement string `json:"edition_statement,omitempty"`
	PublicationDate  string `json:"publication_date,omitempty"`
	ListPrice        string `json:"list_price,omitempty"`
	Status           string `json:"status,omitempty"`
	Channels         string `json:"channels,omitempty"`

	// Print specification.
	Trim       string `json:"trim,omitempty"`
	PageCount  string `json:"page_count,omitempty"`
	PaperStock string `json:"paper_stock,omitempty"`
	Binding    string `json:"binding,omitempty"`
	Bleed      string `json:"bleed,omitempty"`
	Interior   string `json:"interior,omitempty"`
	Gutter     string `json:"gutter,omitempty"`

	// Ebook specification.
	EPUBVersion string `json:"epub_version,omitempty"`
	Layout      string `json:"layout,omitempty"`
	ASIN        string `json:"asin,omitempty"`
	DRM         string `json:"drm,omitempty"`

	// Advanced: the fields an author needs once and then forgets about.
	ImprintOfRecord string `json:"imprint_of_record,omitempty"`
	TerritoryRights string `json:"territory_rights,omitempty"`
	RightsNotice    string `json:"rights_notice,omitempty"`
	LCCN            string `json:"lccn,omitempty"`

	// Typesetting is "standard" or "custom", and it is stored rather than
	// worked out.
	//
	// Comparing the settings against the defaults to decide which one a format
	// is on cannot work: applying the standard and then reading it back gave
	// different answers, and a format whose standard happens to equal the
	// defaults could never be moved off it. It is a choice the author makes,
	// so it is a field they set. Empty means standard.
	Typesetting string `json:"typesetting,omitempty"`

	// ExportSettings is the export wizard's options, frozen onto the format so
	// that exporting this edition twice produces the same file. Nothing writes
	// or reads it yet; it is carried through a save untouched so that the
	// milestone which fills it in does not have to migrate anything.
	ExportSettings map[string]any `json:"export_settings,omitempty"`

	// Wrap is the print-ready wraparound artwork for this printed object.
	// Absent on an ebook or an audiobook, which have no wrap.
	Wrap *EditionWrap `json:"wrap,omitempty"`
	// SnapshotID names the frozen manuscript this format was exported from,
	// so that exporting the first edition after writing the second produces
	// the first edition's text. It is set when the format is first exported,
	// or by hand from the format panel, and it must name a record in the
	// index's own snapshot catalogue — see snapshot.go for why the reference
	// has to be stated rather than worked out from the archive.
	SnapshotID string `json:"snapshot_id,omitempty"`
}

// FindEdition returns the edition with the given id.
func (idx *EditionIndex) FindEdition(id string) (Edition, bool) {
	if idx == nil {
		return Edition{}, false
	}
	for _, edition := range idx.Editions {
		if edition.ID == id {
			return edition, true
		}
	}
	return Edition{}, false
}

// FindFormat returns a format and the edition it belongs to.
func (idx *EditionIndex) FindFormat(formatID string) (Edition, EditionFormat, bool) {
	if idx == nil {
		return Edition{}, EditionFormat{}, false
	}
	for _, edition := range idx.Editions {
		for _, format := range edition.Formats {
			if format.ID == formatID {
				return edition, format, true
			}
		}
	}
	return Edition{}, EditionFormat{}, false
}

// PriorYears is the copyright years established by the editions this one
// supersedes, oldest first. A first edition has none.
//
// The walk follows PreviousEditionID and stops at an edition it has already
// seen, so a record that points at itself — or at an ancestor, through a
// hand-edited file — yields a finite list instead of hanging the save.
func (idx *EditionIndex) PriorYears(editionID string) []string {
	if idx == nil {
		return nil
	}
	seen := map[string]bool{editionID: true}
	var years []string
	current, ok := idx.FindEdition(editionID)
	for ok && strings.TrimSpace(current.PreviousEditionID) != "" {
		if seen[current.PreviousEditionID] {
			break
		}
		seen[current.PreviousEditionID] = true
		current, ok = idx.FindEdition(current.PreviousEditionID)
		if !ok {
			break
		}
		if year := strings.TrimSpace(current.Year); year != "" {
			years = append([]string{year}, years...)
		}
	}
	return years
}

// DerivedISBN10 is the ISBN-10 form of this format's number, or "" when there
// is none. A 979 ISBN has no ISBN-10 at all: that is a fact about the number,
// not a failure, and the screen shows nothing rather than an error.
func (f EditionFormat) DerivedISBN10(hyphenate bool) string {
	ten := ISBN10(f.ISBN13)
	if ten == "" || !hyphenate {
		return ten
	}
	return ten[:1] + "-" + ten[1:9] + "-" + ten[9:]
}

// ── Spine width ────────────────────────────────────────────────────────────
//
// The spine of a perfect-bound printed book is its page count times the
// printer-published thickness of one page of its stock. Case-laminate covers
// have hinges, boards and wrap dimensions that must come from the printer's
// template, so Draftline deliberately gives them no made-up numeric spine.
//
// The stock table is in inches per page — the printer-published caliper
// figures for the stocks Draftline offers:
//
//	Cream, 55#         0.0025     the standard novel stock
//	White, 60#         0.002252   thinner than cream despite the weight
//	Groundwood, 45#    0.00235
//	Color paper        0.002347
const (
	spinePerPageCream55    = 0.0025
	spinePerPageWhite60    = 0.002252
	spinePerPageGroundwood = 0.00235
	spinePerPageColor      = 0.002347
)

// SpinePerPage is the thickness of one leaf of a paper stock, in inches.
func SpinePerPage(stock string) float64 {
	switch normalizeSpec(stock) {
	case "white,60#", "white60#", "white,50#", "white50#":
		return spinePerPageWhite60
	case "groundwood,45#", "groundwood45#":
		return spinePerPageGroundwood
	case "color", "colorpaper", "white,color":
		return spinePerPageColor
	default:
		return spinePerPageCream55
	}
}

// SpineWidthInches is the spine of a printed format, in inches. It returns 0
// for a format with no usable page count, and for anything that is not print,
// which is the honest answer: an ebook has no spine.
func SpineWidthInches(f EditionFormat) float64 {
	if f.Kind != EditionKindPrint {
		return 0
	}
	pages, err := strconv.Atoi(strings.TrimSpace(f.PageCount))
	if err != nil || pages <= 0 {
		return 0
	}
	if pages%2 != 0 {
		pages++
	}
	if binding := normalizeSpec(f.Binding); binding == "caselaminate" || binding == "clothwithjacket" || strings.Contains(strings.ToLower(f.Format), "hardcover") {
		return 0
	}
	caliper := SpinePerPage(f.PaperStock)
	if interior := strings.ToLower(f.Interior); strings.Contains(interior, "color") || strings.Contains(interior, "colour") {
		caliper = spinePerPageColor
	}
	return float64(pages) * caliper
}

// SpineWidthLabel is the spine as the screen shows it: three decimals and the
// unit, or "" when there is nothing to show.
func SpineWidthLabel(f EditionFormat) string {
	width := SpineWidthInches(f)
	if width <= 0 {
		return ""
	}
	return strconv.FormatFloat(width, 'f', 3, 64) + " in"
}

// normalizeSpec folds a spec label down to something a switch can match, so
// that "White, 60#" and "white 60 #" name the same stock.
func normalizeSpec(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if r == ' ' || r == '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
