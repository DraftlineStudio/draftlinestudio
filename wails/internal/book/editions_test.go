package book

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"draftline/internal/types"
)

// twoEditionsFiveFormats is the record the milestone is measured against: two
// editions with five formats between them. Every name, imprint and number is
// invented; the ISBNs pass their own check digits.
func twoEditionsFiveFormats() *types.EditionIndex {
	return &types.EditionIndex{
		Version: 1,
		Editions: []types.Edition{
			{
				ID: "ed-1", Label: "First edition", Year: "2026", Status: "Published",
				RevisionNote: "Original release.",
				Formats: []types.EditionFormat{
					{
						ID: "ed-1-ebook", Kind: types.EditionKindEbook, Format: "eBook",
						ISBN13: "978-1-9471345-0-8", Registration: "Registered — agency",
						EditionStatement: "First edition", PublicationDate: "2026-04-14",
						ListPrice: "6.99", Status: "Published", Channels: "Three storefronts",
						EPUBVersion: "EPUB 3.3", Layout: "Reflowable", ASIN: "B0INVENTED1", DRM: "None",
						ImprintOfRecord: "Windlass and Co.", TerritoryRights: "World",
						RightsNotice: "All rights reserved", LCCN: "2026901447",
					},
					{
						ID: "ed-1-paperback", Kind: types.EditionKindPrint, Format: "Paperback",
						ISBN13: "978-1-9471345-1-5", Registration: "Registered — agency",
						EditionStatement: "First edition", PublicationDate: "2026-04-14",
						ListPrice: "15.99", Status: "Published", Channels: "Two print channels",
						Trim: "6 × 9 in (trade)", PageCount: "412", PaperStock: "Cream, 55#",
						Binding: "Perfect bound", Bleed: "No bleed", Interior: "Black and white",
						Gutter: "0.875 in", ImprintOfRecord: "Windlass and Co.",
						TerritoryRights: "World", RightsNotice: "All rights reserved",
					},
				},
			},
			{
				ID: "ed-2", Label: "Second edition", Year: "2030", Status: "In progress",
				PreviousEditionID: "ed-1",
				RevisionNote:      "Revised and reset. Cover art by Petra Kemball.",
				Formats: []types.EditionFormat{
					{
						ID: "ed-2-ebook", Kind: types.EditionKindEbook, Format: "eBook",
						ISBN13: "979-8-1234567-1-2", Registration: "Registered — agency",
						EditionStatement: "Second edition, revised", PublicationDate: "2030-09-08",
						ListPrice: "7.99", Status: "Registered", EPUBVersion: "EPUB 3.3",
						Layout: "Reflowable", ImprintOfRecord: "Windlass and Co.",
					},
					{
						ID: "ed-2-paperback", Kind: types.EditionKindPrint, Format: "Paperback",
						ISBN13: "978-1-9471345-4-6", PublicationDate: "2030-09-08",
						ListPrice: "17.99", Status: "Registered", Trim: "6 × 9 in (trade)",
						PageCount: "428", PaperStock: "Cream, 55#", Binding: "Perfect bound",
						ImprintOfRecord: "Windlass and Co.",
					},
					{
						ID: "ed-2-hardcover", Kind: types.EditionKindPrint, Format: "Hardcover",
						Status: "Draft", Registration: "Not yet assigned", Trim: "6 × 9 in (trade)",
						PageCount: "428", PaperStock: "White, 60#", Binding: "Case laminate",
						ListPrice: "28.00", ImprintOfRecord: "Windlass and Co.",
					},
				},
			},
		},
	}
}

// Register two editions with five formats between them, let autosave fire ten
// times, close and reopen the project: every field comes back in order.
func TestThePublishingRecordSurvivesRepeatedSavesAndReopening(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "book.draftline")
	b := testBook()
	b.Body[0].ID = "ch-one"
	b.Editions = twoEditionsFiveFormats()

	if res := Write("", path, b, "v1"); !res.Success {
		t.Fatalf("first save failed: %s", res.Error)
	}
	for i := 0; i < 10; i++ {
		// Every other autosave carries a chapter snapshot, because that is the
		// branch of the writer that rewrites history from memory and preserves
		// the rest of the archive on a different prefix list.
		var res types.SaveResult
		if i%2 == 0 {
			res = WriteWithSnapshots(path, path, b, "v1", []types.ChapterSnapshotRequest{writingSnapshot()})
		} else {
			res = Write(path, path, b, "v1")
		}
		if !res.Success {
			t.Fatalf("autosave %d failed: %s", i+1, res.Error)
		}
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopening the project failed: %v", err)
	}
	if reopened.Editions == nil {
		t.Fatal("the publishing record did not come back at all")
	}
	want := twoEditionsFiveFormats()
	gotJSON, _ := json.MarshalIndent(reopened.Editions, "", "  ")
	wantJSON, _ := json.MarshalIndent(want, "", "  ")
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("the publishing record changed across ten saves and a reopen:\ngot\n%s\nwant\n%s", gotJSON, wantJSON)
	}

	// Order is part of the record: the rail lists editions and their formats
	// in the order the author made them.
	if reopened.Editions.Editions[0].ID != "ed-1" || reopened.Editions.Editions[1].ID != "ed-2" {
		t.Fatalf("editions came back out of order: %q, %q", reopened.Editions.Editions[0].ID, reopened.Editions.Editions[1].ID)
	}
	formats := reopened.Editions.Editions[1].Formats
	if len(formats) != 3 || formats[0].ID != "ed-2-ebook" || formats[2].ID != "ed-2-hardcover" {
		t.Fatalf("formats came back out of order: %+v", formats)
	}
}

// A book that never opened the screen has no editions member, and saving it
// does not invent one.
func TestABookWithNoEditionsWritesNoEditionsMember(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "book.draftline")
	if res := Write("", path, testBook(), "v1"); !res.Success {
		t.Fatalf("save failed: %s", res.Error)
	}
	if _, ok := archiveEntries(t, path)[editionsIndexFile]; ok {
		t.Fatalf("a book with no editions still wrote %s", editionsIndexFile)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Editions != nil {
		t.Fatal("a book with no editions came back with a publishing record")
	}
}

// The index is the one member under editions/ that a save rebuilds. It must
// replace the copy in the old file rather than colliding with it, and the save
// must not end up holding the name twice.
func TestSavingRewritesTheIndexRatherThanCollidingWithTheOldOne(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "book.draftline")
	b := testBook()
	b.Editions = twoEditionsFiveFormats()
	if res := Write("", path, b, "v1"); !res.Success {
		t.Fatalf("first save failed: %s", res.Error)
	}

	b.Editions.Editions[1].Formats[2].ISBN13 = "978-1-9471345-3-9"
	b.Editions.Editions[1].Formats[2].Status = "Registered"
	if res := Write(path, path, b, "v1"); !res.Success {
		t.Fatalf("second save failed: %s", res.Error)
	}

	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	seen := 0
	for _, file := range r.File {
		if file.Name == editionsIndexFile {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("the archive holds %s %d times", editionsIndexFile, seen)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Editions.Editions[1].Formats[2].ISBN13; got != "978-1-9471345-3-9" {
		t.Fatalf("the newly registered ISBN was lost: %q", got)
	}
}

// A record written by a newer Draftline is refused by name, and the project on
// disk is left exactly as it was.
func TestAnIndexFromTheFutureRefusesToOpenAndLeavesTheFileAlone(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "book.draftline")
	b := testBook()
	b.Editions = twoEditionsFiveFormats()
	if res := Write("", path, b, "v1"); !res.Success {
		t.Fatalf("save failed: %s", res.Error)
	}

	future := `{"version":2,"editions":[{"id":"ed-9","label":"Ninth edition","year":"2044","formats":[]}]}`
	replaceArchiveEntry(t, path, editionsIndexFile, []byte(future))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Open(path)
	if err == nil {
		t.Fatal("a version this build does not know was read as if it were current")
	}
	message := err.Error()
	for _, fragment := range []string{editionsIndexFile, "version 2", "version 1"} {
		if !strings.Contains(message, fragment) {
			t.Errorf("the refusal does not mention %q: %s", fragment, message)
		}
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("refusing to read the record still changed the project on disk")
	}
}

// Saving a record this build does not understand fails the save rather than
// writing a lossy version of it over the author's file.
func TestSavingAnUnsupportedVersionFailsInsteadOfRewritingIt(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "book.draftline")
	b := testBook()
	b.Editions = &types.EditionIndex{Version: 7, Editions: []types.Edition{}}
	res := Write("", path, b, "v1")
	if res.Success {
		t.Fatal("a save wrote an editions version it cannot read")
	}
	if !strings.Contains(res.Error, editionsIndexFile) {
		t.Fatalf("the failure does not name the file: %s", res.Error)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("the failed save still produced a project file")
	}
}

func TestAnAmbiguousRecordIsRefusedByName(t *testing.T) {
	cases := []struct {
		name  string
		index types.EditionIndex
		want  string
	}{
		{
			name: "an edition with no identifier",
			index: types.EditionIndex{Version: 1, Editions: []types.Edition{
				{Label: "First edition"},
			}},
			want: "no identifier",
		},
		{
			name: "two editions sharing an identifier",
			index: types.EditionIndex{Version: 1, Editions: []types.Edition{
				{ID: "ed-1", Label: "First edition"},
				{ID: "ed-1", Label: "Also the first edition"},
			}},
			want: `two editions with the identifier "ed-1"`,
		},
		{
			name: "two formats sharing an identifier, in different editions",
			index: types.EditionIndex{Version: 1, Editions: []types.Edition{
				{ID: "ed-1", Formats: []types.EditionFormat{{ID: "pb", Kind: types.EditionKindPrint}}},
				{ID: "ed-2", Formats: []types.EditionFormat{{ID: "pb", Kind: types.EditionKindPrint}}},
			}},
			want: `two formats with the identifier "pb"`,
		},
		{
			name: "a format whose kind decides nothing",
			index: types.EditionIndex{Version: 1, Editions: []types.Edition{
				{ID: "ed-1", Formats: []types.EditionFormat{{ID: "pb", Kind: "vinyl"}}},
			}},
			want: `unknown kind "vinyl"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := prepareEditionsData(tc.index)
			if err == nil {
				t.Fatal("accepted a record nothing can select from")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refusal does not say %q: %s", tc.want, err)
			}
		})
	}
}

func TestPreparingARecordTrimsTypingAndFillsInEmptyLists(t *testing.T) {
	prepared, err := prepareEditionsData(types.EditionIndex{Version: 1, Editions: []types.Edition{
		{ID: "  ed-1  ", Label: " First edition ", Year: " 2026 ", Formats: []types.EditionFormat{
			{ID: " ed-1-pb ", Kind: " print ", Format: " Paperback ", ISBN13: " 978-1-9471345-1-5 ", LCCN: " 2026901447 "},
		}},
		{ID: "ed-2"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	first := prepared.Editions[0]
	if first.ID != "ed-1" || first.Label != "First edition" || first.Year != "2026" {
		t.Fatalf("edition was not trimmed: %+v", first)
	}
	format := first.Formats[0]
	if format.ID != "ed-1-pb" || format.Kind != "print" || format.Format != "Paperback" ||
		format.ISBN13 != "978-1-9471345-1-5" || format.LCCN != "2026901447" {
		t.Fatalf("format was not trimmed: %+v", format)
	}
	// An edition with no formats gets an empty list, not a null, so the
	// frontend never has to guard a map over it.
	if prepared.Editions[1].Formats == nil {
		t.Fatal("an edition with no formats came back with a null format list")
	}
}

// replaceArchiveEntry rewrites one member of an archive in place, standing in
// for a project file written by a different version of Draftline.
func replaceArchiveEntry(t *testing.T, path, name string, data []byte) {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("cannot open fixture archive: %v", err)
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, file := range r.File {
		if file.Name == name {
			continue
		}
		if err := w.Copy(file); err != nil {
			t.Fatalf("cannot copy fixture entry %q: %v", file.Name, err)
		}
	}
	_ = r.Close()
	f, err := w.Create(name)
	if err != nil {
		t.Fatalf("cannot add fixture entry: %v", err)
	}
	if _, err := f.Write(data); err != nil {
		t.Fatalf("cannot write fixture entry: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("cannot finalize fixture archive: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("cannot replace fixture archive: %v", err)
	}
}
