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
		if !safeArchiveSegment(edition.ID) {
			return types.EditionIndex{}, fmt.Errorf(
				"%s gives an edition the identifier %q, which cannot be a folder name inside the project file",
				editionsIndexFile, edition.ID)
		}
		editionIDs[edition.ID] = true
		edition.Label = strings.TrimSpace(edition.Label)
		edition.Year = strings.TrimSpace(edition.Year)
		edition.Status = strings.TrimSpace(edition.Status)
		edition.CoverID = strings.TrimSpace(edition.CoverID)
		if err := prepareCoverRecord(edition); err != nil {
			return types.EditionIndex{}, err
		}
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

// ── Cover art ──────────────────────────────────────────────────────────────

// CoverPrefix is the folder one edition's binary assets live in. Everything
// under editions/ survives a save by passthrough; this is the part of it that
// holds images.
func CoverPrefix(editionID string) string {
	return editionsPrefix + editionID + "/"
}

// CoverMember is the full archive member name of one of an edition's images.
func CoverMember(editionID, file string) string {
	return CoverPrefix(editionID) + file
}

// editionFolder names the edition an archive member is filed under, for
// members that live in an edition's own folder.
//
// editions/index.json belongs to no edition and answers false, which is what
// keeps the publishing record itself from being read as an orphan and swept
// away by the reaping below.
func editionFolder(name string) (string, bool) {
	rest, ok := strings.CutPrefix(name, editionsPrefix)
	if !ok {
		return "", false
	}
	id, _, ok := strings.Cut(rest, "/")
	if !ok || id == "" {
		return "", false
	}
	return id, true
}

// orphanedEditionMember reports whether a member belongs to an edition the
// book no longer has.
//
// Everything under editions/ survives a save by passthrough, byte for byte,
// which is what keeps cover art alive through the five-second autosave. The
// same mechanism means that deleting an edition would otherwise leave its
// artwork in the project file for ever: nothing on screen refers to it, no
// screen can see it, and every later save and every backup copies it again. An
// author who reworks an edition - and the screen tells them a changed trim or
// publisher means a NEW edition record, so that is the expected way to work -
// would leave a megabyte behind each time.
//
// live is the set of edition identifiers the save is writing. A nil set means
// this save does not know them and must not reap: a book saved with no
// publishing record at all does not rewrite editions/index.json either, so the
// old index is still carried across and the editions it names are still real.
func orphanedEditionMember(name string, live map[string]bool) bool {
	if live == nil {
		return false
	}
	id, ok := editionFolder(name)
	return ok && !live[id]
}

// liveEditionIDs is the set of editions a prepared publishing record names.
func liveEditionIDs(index types.EditionIndex) map[string]bool {
	live := map[string]bool{}
	for _, edition := range index.Editions {
		live[edition.ID] = true
	}
	return live
}

// prepareCoverRecord tidies an edition's cover record and refuses one that
// could not be filed.
//
// The file names matter more than they look. They become an archive member
// name and a URL path inside the running app, so a name with a slash or a
// parent-directory step in it would reach outside the edition's own folder.
// Draftline writes these names itself, but the project file is a ZIP anyone
// can edit, and a value read out of one is not a value this code wrote.
func prepareCoverRecord(edition *types.Edition) error {
	cover := edition.Cover
	if cover == nil {
		return nil
	}
	cover.ID = strings.TrimSpace(cover.ID)
	cover.File = strings.TrimSpace(cover.File)
	cover.ThumbFile = strings.TrimSpace(cover.ThumbFile)
	cover.LargeFile = strings.TrimSpace(cover.LargeFile)
	cover.Encoding = strings.TrimSpace(cover.Encoding)
	cover.SourcePath = strings.TrimSpace(cover.SourcePath)
	cover.SourceChecksum = strings.TrimSpace(cover.SourceChecksum)

	if cover.File == "" || cover.ThumbFile == "" {
		return fmt.Errorf("%s gives edition %q a cover with no image filed under it", editionsIndexFile, edition.ID)
	}
	for _, name := range []string{cover.File, cover.ThumbFile, cover.LargeFile} {
		if name == "" {
			continue
		}
		if !safeArchiveSegment(name) {
			return fmt.Errorf("%s files edition %q's cover under the name %q, which is not a plain file name", editionsIndexFile, edition.ID, name)
		}
	}
	if cover.Notes == nil {
		cover.Notes = []string{}
	}
	// The identifier on the record and the identifier on the edition are the
	// same fact written twice; the cover's own is the one that was made when
	// the art was attached.
	if cover.ID != "" {
		edition.CoverID = cover.ID
	}
	return nil
}

// safeArchiveSegment reports whether a string can be one path segment inside
// the archive and inside a same-origin asset URL.
func safeArchiveSegment(value string) bool {
	if value == "" || value == "." || value == ".." || len(value) > 64 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}
