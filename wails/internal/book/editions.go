package book

import (
	"fmt"
	"strings"

	"draftline/internal/types"
)

// editionsDataVersion is the version of editions/index.json this build reads
// and writes. It is checked through requireArchiveVersion, which refuses an
// unknown version outright: an author who registers editions in a newer
// Draftline and then opens the project here must not have that record read
// loosely and written back over.
const editionsDataVersion = 1

// editionsIndexFile is the archive member the publishing record lives in.
// Everything under editions/ is carried across a save byte for byte unless
// this file rewrites it — see preservedArchivePrefixes.
const editionsIndexFile = "editions/index.json"

// prepareEditionsData validates the publishing record and normalises it for
// storage and for the frontend.
//
// What it refuses, it refuses by name, because every one of these makes the
// record ambiguous rather than merely untidy:
//
//   - an unsupported version, which is the forward gate above;
//   - an edition or a format with no identifier, which nothing can select;
//   - two editions, or two formats anywhere in the book, sharing one
//     identifier, because the screen selects by identifier and would then be
//     editing whichever of them it happened to find first;
//   - a format whose kind is not one Draftline knows, because the kind decides
//     which specification the format has and an unknown one has none.
//
// What it normalises is whitespace and empty collections. It does not touch
// the substance of a field: an ISBN is stored exactly as the author typed it,
// hyphens and all.
func prepareEditionsData(index types.EditionIndex) (types.EditionIndex, error) {
	if err := requireArchiveVersion(editionsIndexFile, index.Version, editionsDataVersion); err != nil {
		return types.EditionIndex{}, err
	}
	if index.Editions == nil {
		index.Editions = []types.Edition{}
	}
	editionIDs := map[string]bool{}
	formatIDs := map[string]bool{}
	for i := range index.Editions {
		edition := &index.Editions[i]
		edition.ID = strings.TrimSpace(edition.ID)
		if edition.ID == "" {
			return types.EditionIndex{}, fmt.Errorf("%s holds an edition with no identifier", editionsIndexFile)
		}
		if editionIDs[edition.ID] {
			return types.EditionIndex{}, fmt.Errorf("%s holds two editions with the identifier %q", editionsIndexFile, edition.ID)
		}
		editionIDs[edition.ID] = true
		edition.Label = strings.TrimSpace(edition.Label)
		edition.Year = strings.TrimSpace(edition.Year)
		edition.Status = strings.TrimSpace(edition.Status)
		edition.CoverID = strings.TrimSpace(edition.CoverID)
		edition.PreviousEditionID = strings.TrimSpace(edition.PreviousEditionID)
		edition.RevisionNote = strings.TrimSpace(edition.RevisionNote)
		if edition.Formats == nil {
			edition.Formats = []types.EditionFormat{}
		}
		for j := range edition.Formats {
			format := &edition.Formats[j]
			format.ID = strings.TrimSpace(format.ID)
			if format.ID == "" {
				return types.EditionIndex{}, fmt.Errorf("%s holds a format of %q with no identifier", editionsIndexFile, edition.ID)
			}
			if formatIDs[format.ID] {
				return types.EditionIndex{}, fmt.Errorf("%s holds two formats with the identifier %q", editionsIndexFile, format.ID)
			}
			formatIDs[format.ID] = true
			format.Kind = strings.TrimSpace(format.Kind)
			switch format.Kind {
			case types.EditionKindEbook, types.EditionKindPrint, types.EditionKindAudio:
			default:
				return types.EditionIndex{}, fmt.Errorf("%s gives format %q the unknown kind %q", editionsIndexFile, format.ID, format.Kind)
			}
			trimFormatText(format)
		}
	}
	return index, nil
}

// trimFormatText strips the whitespace a form leaves on every text field. It
// is a long list because EditionFormat is a long record; there is no
// reflection here so that a field added later is a compile-time decision about
// whether it needs trimming, not a silent one.
func trimFormatText(format *types.EditionFormat) {
	for _, field := range []*string{
		&format.Format,
		&format.ISBN13,
		&format.Registration,
		&format.EditionStatement,
		&format.PublicationDate,
		&format.ListPrice,
		&format.Status,
		&format.Channels,
		&format.Trim,
		&format.PageCount,
		&format.PaperStock,
		&format.Binding,
		&format.Bleed,
		&format.Interior,
		&format.Gutter,
		&format.EPUBVersion,
		&format.Layout,
		&format.ASIN,
		&format.DRM,
		&format.ImprintOfRecord,
		&format.TerritoryRights,
		&format.RightsNotice,
		&format.LCCN,
		&format.SnapshotID,
	} {
		*field = strings.TrimSpace(*field)
	}
}
