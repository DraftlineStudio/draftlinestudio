package main

// Freezing and exporting, from the application's side.
//
// internal/book proves that a snapshot survives the archive. What has to be
// true here is the part that cannot be tested there: an export of a registered
// edition renders the words that edition went out with rather than the draft
// on screen, and none of those words ever touch the struct that crosses the
// Wails bridge.
//
// Every fixture in this file is invented. No sentence here comes from anybody's
// book.

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"draftline/internal/book"
	"draftline/internal/export"
	"draftline/internal/types"
)

func harbourBook() types.BookData {
	return types.BookData{
		Version: "2.2",
		Metadata: types.Metadata{
			Title: "The Salt Lantern", Author: "Nell Ardwick",
			Publisher: "Windlass and Co.", Language: "en-GB",
			CopyrightHolder: "Nell Ardwick",
		},
		Copyright: "<p>Set by hand.</p>",
		Body: []types.ChapterItem{
			{ID: "ch-1", Title: "The Tide Gate", Type: "chapter", Content: "<p>The gate stood open and the water came through it sideways.</p>"},
			{ID: "ch-2", Title: "Bell and Rope", Type: "chapter", Content: "<p>Ilsa counted the strokes and got a different number each time.</p>"},
		},
		Editions: &types.EditionIndex{
			Version: 1,
			Editions: []types.Edition{{
				ID: "ed-1", Label: "First edition", Year: "2026", Status: "Published",
				Formats: []types.EditionFormat{
					{ID: "fmt-ebook", Kind: types.EditionKindEbook, Format: "eBook", ISBN13: "978-1-9471345-0-8", EPUBVersion: "EPUB 3.3"},
					{ID: "fmt-paper", Kind: types.EditionKindPrint, Format: "Paperback", ISBN13: "978-1-9471345-1-5", PageCount: "310", Trim: "6 × 9 in (trade)"},
				},
			}},
		},
	}
}

// openedHarbour puts the book on disk, opens it, and hands back the app.
func openedHarbour(t *testing.T) (*App, string, types.BookData) {
	t.Helper()
	config := t.TempDir()
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CONFIG_HOME", config)

	path := filepath.Join(t.TempDir(), "harbour.draftline")
	b := harbourBook()
	if result := book.Write("", path, b, "test-version"); !result.Success {
		t.Fatal(result.Error)
	}
	app := &App{}
	if _, err := app.openBook(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.releaseBookLock)
	return app, path, b
}

// freeze runs the binding and applies its record to the book, which is what
// the store does on the other side of the bridge.
func freeze(t *testing.T, app *App, b types.BookData, formatID string) (types.BookData, types.SnapshotResult) {
	t.Helper()
	result := app.FreezeSnapshot(b, formatID)
	if !result.Success || result.Snapshot == nil {
		t.Fatalf("freezing %s failed: %s", formatID, result.Error)
	}
	index := *b.Editions
	editions := make([]types.Edition, len(index.Editions))
	copy(editions, index.Editions)
	for i := range editions {
		formats := make([]types.EditionFormat, len(editions[i].Formats))
		copy(formats, editions[i].Formats)
		for j := range formats {
			if formats[j].ID == formatID {
				formats[j].SnapshotID = result.Snapshot.ID
			}
		}
		editions[i].Formats = formats
	}
	index.Editions = editions
	if _, held := index.FindSnapshot(result.Snapshot.ID); !held {
		index.Snapshots = append(append([]types.EditionSnapshot{}, index.Snapshots...), *result.Snapshot)
	}
	b.Editions = &index
	return b, result
}

func snapshotMembers(t *testing.T, path string) []string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("cannot open %s: %v", filepath.Base(path), err)
	}
	defer func() { _ = r.Close() }()
	var names []string
	for _, file := range r.File {
		if strings.HasPrefix(file.Name, "editions/snapshots/") {
			names = append(names, file.Name)
		}
	}
	return names
}

// The milestone's first acceptance: the same unchanged text exported from two
// formats stores once, and both records point at the same words.
func TestTwoFormatsExportedFromTheSameTextShareOneStoredCopy(t *testing.T) {
	app, path, b := openedHarbour(t)

	b, first := freeze(t, app, b, "fmt-ebook")
	b, second := freeze(t, app, b, "fmt-paper")

	if first.Snapshot.ID != second.Snapshot.ID {
		t.Fatalf("the same text froze as %s and %s", first.Snapshot.ID, second.Snapshot.ID)
	}
	if !second.Reused {
		t.Fatal("the second freeze did not say the text was already frozen")
	}
	if len(b.Editions.Snapshots) != 1 {
		t.Fatalf("the catalogue holds %d frozen manuscripts, want 1", len(b.Editions.Snapshots))
	}
	if result := app.writeBook(b, path); !result.Success {
		t.Fatal(result.Error)
	}
	for _, name := range snapshotMembers(t, path) {
		if !strings.HasPrefix(name, "editions/snapshots/"+first.Snapshot.ID+"/") {
			t.Fatalf("a second copy of the same text is in the project file: %s", name)
		}
	}
}

// Write a second edition, then export the first edition's paperback: the first
// edition's text and the first edition's metadata come out, not the draft.
func TestExportingARegisteredEditionRendersTheTextItWentOutWith(t *testing.T) {
	app, path, b := openedHarbour(t)
	b, frozen := freeze(t, app, b, "fmt-paper")
	if result := app.writeBook(b, path); !result.Success {
		t.Fatal(result.Error)
	}

	// A year of revision, and a new title on the book record.
	b.Metadata.Title = "The Salt Lantern: Second Edition"
	b.Body = []types.ChapterItem{
		{ID: "ch-1", Title: "The Weir", Type: "chapter", Content: "<p>They had rebuilt the gate by then, and it held.</p>"},
	}
	if result := app.writeBook(b, path); !result.Success {
		t.Fatal(result.Error)
	}

	source, err := app.exportSource(b, types.ExportOptions{EditionID: "ed-1", FormatID: "fmt-paper"})
	if err != nil {
		t.Fatalf("exportSource: %v", err)
	}
	if source.Metadata.Title != "The Salt Lantern" {
		t.Fatalf("the export carries the title %q, want the one the ISBN went out with", source.Metadata.Title)
	}
	if len(source.Body) != 2 || source.Body[0].Title != "The Tide Gate" {
		t.Fatalf("the export carries %d chapters starting %q", len(source.Body), source.Body[0].Title)
	}
	if !strings.Contains(source.Body[0].Content, "sideways") {
		t.Fatalf("the frozen words did not come out: %q", source.Body[0].Content)
	}
	if source.Copyright != "<p>Set by hand.</p>" {
		t.Fatalf("the author's own copyright page did not come out: %q", source.Copyright)
	}
	// The publishing record itself is the live one — an ISBN corrected today
	// is the ISBN the file declares.
	if source.Editions == nil || source.Editions.Editions[0].Formats[1].SnapshotID != frozen.Snapshot.ID {
		t.Fatal("the export lost the publishing record")
	}
}

// A format with no frozen text exports the working draft. That is what the
// wizard says it will do, and it has to be what happens.
func TestAFormatWithNoFrozenTextExportsTheWorkingDraft(t *testing.T) {
	app, _, b := openedHarbour(t)
	b.Body[0].Content = "<p>Freshly typed this morning.</p>"

	source, err := app.exportSource(b, types.ExportOptions{EditionID: "ed-1", FormatID: "fmt-ebook"})
	if err != nil {
		t.Fatalf("exportSource: %v", err)
	}
	if source.Body[0].Content != "<p>Freshly typed this morning.</p>" {
		t.Fatalf("the draft did not come out: %q", source.Body[0].Content)
	}
	// So does an export attached to no edition at all.
	plain, err := app.exportSource(b, types.ExportOptions{})
	if err != nil {
		t.Fatalf("exportSource with no edition: %v", err)
	}
	if plain.Body[0].Content != "<p>Freshly typed this morning.</p>" {
		t.Fatal("an export with no edition did not get the draft")
	}
}

// Reconstructing a snapshot and exporting produces the same EPUB that was made
// at freeze time, apart from the stamp that says when the file was built.
func TestAnEPUBRebuiltFromAFrozenManuscriptMatchesTheOneMadeAtFreezeTime(t *testing.T) {
	app, path, b := openedHarbour(t)

	options := types.EPUBOptions{ExportOptions: types.ExportOptions{
		IncludeCopyright: true, IncludeFrontMatter: true, IncludeBackMatter: true,
		EditionID: "ed-1", FormatID: "fmt-ebook",
	}}

	atFreeze := filepath.Join(t.TempDir(), "at-freeze.epub")
	b, _ = freeze(t, app, b, "fmt-ebook")
	if result := export.EPUB(atFreeze, b, options, nil); !result.Success {
		t.Fatal(result.Error)
	}
	if result := app.writeBook(b, path); !result.Success {
		t.Fatal(result.Error)
	}

	// The draft moves on entirely.
	moved := b
	moved.Body = []types.ChapterItem{{ID: "ch-9", Title: "Nothing In Common", Type: "chapter", Content: "<p>A different book.</p>"}}
	moved.Metadata.Title = "Another Title Altogether"

	rebuilt := filepath.Join(t.TempDir(), "rebuilt.epub")
	source, err := app.exportSource(moved, options.ExportOptions)
	if err != nil {
		t.Fatalf("exportSource: %v", err)
	}
	if result := export.EPUB(rebuilt, source, options, nil); !result.Success {
		t.Fatal(result.Error)
	}

	want := epubMembers(t, atFreeze)
	got := epubMembers(t, rebuilt)
	if len(want) != len(got) {
		t.Fatalf("the rebuilt EPUB has %d members, the original %d", len(got), len(want))
	}
	// dcterms:modified is the one thing that legitimately differs: it records
	// when the file was built, not what is in it.
	stamp := regexp.MustCompile(`<meta property="dcterms:modified">[^<]*</meta>`)
	for name, data := range want {
		other, ok := got[name]
		if !ok {
			t.Fatalf("the rebuilt EPUB is missing %s", name)
		}
		a := stamp.ReplaceAllString(string(data), "STAMP")
		bb := stamp.ReplaceAllString(string(other), "STAMP")
		if a != bb {
			t.Fatalf("%s differs between the file made at freeze time and the one rebuilt from the snapshot", name)
		}
	}
}

func epubMembers(t *testing.T, path string) map[string][]byte {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("cannot open %s: %v", filepath.Base(path), err)
	}
	defer func() { _ = r.Close() }()
	out := map[string][]byte{}
	for _, file := range r.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("cannot read %s: %v", file.Name, err)
		}
		data := make([]byte, file.UncompressedSize64)
		if _, err := io.ReadFull(rc, data); err != nil {
			t.Fatalf("cannot read %s: %v", file.Name, err)
		}
		_ = rc.Close()
		out[file.Name] = data
	}
	return out
}

// The struct that crosses the bridge must be the same size with a whole book
// frozen as it is without one.
func TestFrozenTextIsNotOnTheStructThatCrossesTheBridge(t *testing.T) {
	app, _, b := openedHarbour(t)
	long := strings.Repeat("<p>The harbour wall ran three hundred yards and then stopped.</p>", 400)
	b.Body[0].Content = long
	b.Body[1].Content = long

	before, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	b, result := freeze(t, app, b, "fmt-ebook")
	after, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}

	if result.Snapshot.Bytes < int64(len(long)) {
		t.Fatalf("the snapshot records %d bytes for a manuscript of at least %d", result.Snapshot.Bytes, len(long))
	}
	// The catalogue record is a few hundred bytes; the manuscript is tens of
	// thousands. Anything close to the latter means the text came along.
	if grew := len(after) - len(before); grew > 1024 {
		t.Fatalf("freezing grew the bridge payload by %d bytes; the text is riding on it", grew)
	}
}

// The words are read back out of the project file after the session that froze
// them has let go of them.
func TestAFrozenManuscriptIsReadBackFromTheProjectFile(t *testing.T) {
	app, path, b := openedHarbour(t)
	b, frozen := freeze(t, app, b, "fmt-ebook")
	if result := app.writeBook(b, path); !result.Success {
		t.Fatal(result.Error)
	}
	// A reopened project: the cache knows nothing.
	app.snapshots.reset()

	source, err := app.exportSource(b, types.ExportOptions{EditionID: "ed-1", FormatID: "fmt-ebook"})
	if err != nil {
		t.Fatalf("exportSource after a reopen: %v", err)
	}
	if len(source.Body) != 2 || !strings.Contains(source.Body[1].Content, "different number") {
		t.Fatalf("the frozen text did not come back off disk: %+v", source.Body)
	}
	if _, held := b.Editions.FindSnapshot(frozen.Snapshot.ID); !held {
		t.Fatal("the catalogue lost the record")
	}
}

// An export that says it is giving you the book as it was, and then cannot
// find those words, must fail rather than quietly hand over today's draft.
func TestAnExportRefusesRatherThanSubstituteTheDraftForMissingFrozenText(t *testing.T) {
	app, path, b := openedHarbour(t)
	b, _ = freeze(t, app, b, "fmt-ebook")
	if result := app.writeBook(b, path); !result.Success {
		t.Fatal(result.Error)
	}
	app.snapshots.reset()

	// Point the format at words that are not there.
	b.Editions.Editions[0].Formats[0].SnapshotID = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	// Every exporter goes through this one resolution, and every one of them
	// turns its error into the message on screen without rewording it, so the
	// sentence here is the sentence the author reads.
	_, err := app.exportSource(b, types.ExportOptions{EditionID: "ed-1", FormatID: "fmt-ebook"})
	if err == nil {
		t.Fatal("an export with unreadable frozen text silently used the draft")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "frozen text of this edition") {
		t.Fatalf("the refusal does not say what is wrong: %v", err)
	}
}

// Freezing refuses the things that have no answer, rather than inventing one.
func TestFreezingRefusesWhatItCannotFreeze(t *testing.T) {
	app, _, b := openedHarbour(t)

	if result := app.FreezeSnapshot(b, ""); result.Success {
		t.Error("freezing with no format named succeeded")
	}
	if result := app.FreezeSnapshot(b, "fmt-nowhere"); result.Success {
		t.Error("freezing for a format this book does not have succeeded")
	}
	empty := harbourBook()
	empty.Body = nil
	if result := app.FreezeSnapshot(empty, "fmt-ebook"); result.Success {
		t.Error("an empty book was frozen")
	}
	noEditions := harbourBook()
	noEditions.Editions = nil
	if result := app.FreezeSnapshot(noEditions, "fmt-ebook"); result.Success {
		t.Error("a book with no publishing record froze text for a format")
	}
}

// Cover art and frozen manuscripts go to the writer together, and settling one
// after a save must not settle the other early.
func TestASaveCarriesBothCoverArtAndFrozenTextAndSettlesBoth(t *testing.T) {
	app, path, b := openedHarbour(t)
	art := writeTestArtwork(t, filepath.Join(t.TempDir(), "artwork.png"), 1600, 2560, 1.0)
	cover := app.AttachCover("ed-1", art, false)
	if !cover.Success || cover.Cover == nil {
		t.Fatalf("attaching a cover failed: %s", cover.Error)
	}
	b.Editions.Editions[0].Cover = cover.Cover
	b.Editions.Editions[0].CoverID = cover.Cover.ID
	b, frozen := freeze(t, app, b, "fmt-ebook")

	assets, _ := app.pendingAssets()
	if len(assets.Files) < frozen.Snapshot.Members+2 {
		t.Fatalf("the save was handed %d members, want a cover, a thumbnail and %d frozen pieces",
			len(assets.Files), frozen.Snapshot.Members)
	}
	if result := app.writeBook(b, path); !result.Success {
		t.Fatal(result.Error)
	}
	if got := len(snapshotMembers(t, path)); got != frozen.Snapshot.Members {
		t.Fatalf("the project file holds %d frozen pieces, want %d", got, frozen.Snapshot.Members)
	}
	if left, _ := app.pendingAssets(); len(left.Files) != 0 {
		t.Fatalf("a successful save left %d members still pending", len(left.Files))
	}

	// A second save must not lose either of them.
	if result := app.writeBook(b, path); !result.Success {
		t.Fatal(result.Error)
	}
	if got := len(snapshotMembers(t, path)); got != frozen.Snapshot.Members {
		t.Fatalf("the second save left %d frozen pieces, want %d", got, frozen.Snapshot.Members)
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		t.Fatalf("the project file is unreadable: %v", err)
	}
}

// A different project must never be handed the frozen words of the last one.
func TestLeavingAProjectForgetsItsFrozenText(t *testing.T) {
	app, _, b := openedHarbour(t)
	b, _ = freeze(t, app, b, "fmt-ebook")
	if assets, _ := app.pendingAssets(); len(assets.Files) == 0 {
		t.Fatal("nothing was pending after a freeze")
	}
	app.leaveOpenProject()
	if assets, _ := app.pendingAssets(); len(assets.Files) != 0 {
		t.Fatalf("%d members of another book's text are still waiting to be written", len(assets.Files))
	}
}

// An autosave is armed five seconds after every edit, and the wizard's own
// commits arm one. It can therefore fire while FreezeSnapshot is still
// crossing the bridge, carrying the publishing record as it stood BEFORE the
// stamp — and a save that does not know about a frozen manuscript does not
// write it, because the reaper cannot tell an unknown one from an abandoned
// one. What must not follow is the cache concluding from "the save succeeded"
// that the words are on disk: they are not, and they exist nowhere else. The
// ISBN would point at text that can never be exported again, and no screen
// would say so.
func TestAnInterleavedSaveDoesNotThrowAwayFrozenText(t *testing.T) {
	app, path, before := openedHarbour(t)
	after, result := freeze(t, app, before, "fmt-ebook")
	id := result.Snapshot.ID

	// The save that was already on its way, carrying the book as it was.
	if res := app.writeBook(before, path); !res.Success {
		t.Fatal(res.Error)
	}
	if names := snapshotMembers(t, path); len(names) != 0 {
		t.Fatalf("a save that did not know about the freeze wrote it anyway: %v", names)
	}
	if !app.snapshots.has(id) {
		t.Fatal("the frozen manuscript was forgotten by a save that never wrote it")
	}

	// The next save carries the record, and the words go in with it.
	if res := app.writeBook(after, path); !res.Success {
		t.Fatal(res.Error)
	}
	if len(snapshotMembers(t, path)) == 0 {
		t.Fatal("the following save did not write the frozen manuscript")
	}
	if app.snapshots.has(id) {
		t.Fatal("the frozen manuscript is still held in memory after being written")
	}

	// And it reads back: the ISBN still stands for words that exist.
	app.snapshots.reset()
	source, err := app.exportSource(after, types.ExportOptions{EditionID: "ed-1", FormatID: "fmt-ebook"})
	if err != nil {
		t.Fatalf("the frozen text cannot be read back: %v", err)
	}
	if len(source.Body) != len(before.Body) {
		t.Fatalf("the frozen text came back with %d chapters, want %d", len(source.Body), len(before.Body))
	}
}

// An export whose save dialog was dismissed wrote no file and recorded
// nothing, so the words it froze are let go of rather than carried by every
// save for the rest of the session.
func TestAFreezeThatNeverBecameAFileIsLetGoOf(t *testing.T) {
	app, path, b := openedHarbour(t)
	result := app.FreezeSnapshot(b, "fmt-ebook")
	if !result.Success {
		t.Fatal(result.Error)
	}
	id := result.Snapshot.ID
	if !app.snapshots.has(id) {
		t.Fatal("freezing did not hold the words")
	}

	app.DiscardSnapshot(id)
	if app.snapshots.has(id) {
		t.Fatal("the abandoned freeze is still held")
	}
	// The book never learned of it, so the save writes nothing under it.
	if res := app.writeBook(b, path); !res.Success {
		t.Fatal(res.Error)
	}
	if names := snapshotMembers(t, path); len(names) != 0 {
		t.Fatalf("an abandoned freeze reached the project file: %v", names)
	}
}

// Two formats frozen from the same words each hold their own claim on them.
// One export being abandoned must not take the other edition's text with it.
func TestAbandoningOneExportKeepsTheWordsAnotherFormatFroze(t *testing.T) {
	app, path, b := openedHarbour(t)
	b, first := freeze(t, app, b, "fmt-ebook")

	// The second format freezes the same unchanged text, then that export is
	// abandoned at the file picker.
	second := app.FreezeSnapshot(b, "fmt-paper")
	if !second.Success || !second.Reused {
		t.Fatalf("the same text did not come back as already frozen: %+v", second)
	}
	app.DiscardSnapshot(second.Snapshot.ID)

	if !app.snapshots.has(first.Snapshot.ID) {
		t.Fatal("abandoning the second export threw away the first edition's words")
	}
	if res := app.writeBook(b, path); !res.Success {
		t.Fatal(res.Error)
	}
	if len(snapshotMembers(t, path)) == 0 {
		t.Fatal("the first edition's frozen text never reached the project file")
	}
}
