package book

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"draftline/internal/types"
	"draftline/internal/ziputil"
)

// A short invented book. Every name here is made up; nothing in this file
// comes from anybody's manuscript.
func snapshotFixtureBook() types.BookData {
	return types.BookData{
		Version: "2.2",
		Metadata: types.Metadata{
			Title:           "The Salt Lantern",
			Subtitle:        "A Harbour Novel",
			Author:          "Nell Ardwick",
			Publisher:       "Windlass and Co.",
			Imprint:         "Windlass Press",
			Language:        "en-GB",
			CopyrightHolder: "Nell Ardwick",
		},
		Copyright: "<p>Set in Quorn by hand.</p>",
		FrontMatter: []types.ChapterItem{
			{ID: "fm-1", Title: "Dedication", Type: "dedication", Content: "<p>For the harbourmaster.</p>"},
		},
		Body: []types.ChapterItem{
			{ID: "ch-1", Title: "The Tide Gate", Type: "chapter", Content: "<p>The gate stood open and the water came through it sideways.</p>"},
			{ID: "ch-2", Title: "Bell and Rope", Type: "chapter", Content: "<p>Ilsa counted the strokes and got a different number each time.</p>"},
		},
		BackMatter: []types.ChapterItem{
			{ID: "bm-1", Title: "A Note on the Harbour", Type: "afterword", Content: "<p>No such harbour exists.</p>"},
		},
		StoryBible: types.StoryBible{Characters: []types.Character{}},
	}
}

func freezeInto(t *testing.T, index *types.EditionIndex, formatID string, b types.BookData) (Snapshot, Assets) {
	t.Helper()
	snapshot := BuildSnapshot(b, time.Date(2026, 4, 14, 9, 0, 0, 0, time.UTC))
	for i := range index.Editions {
		for j := range index.Editions[i].Formats {
			if index.Editions[i].Formats[j].ID == formatID {
				index.Editions[i].Formats[j].SnapshotID = snapshot.Record.ID
			}
		}
	}
	if _, ok := index.FindSnapshot(snapshot.Record.ID); !ok {
		index.Snapshots = append(index.Snapshots, snapshot.Record)
	}
	return snapshot, Assets{Files: snapshot.Files}
}

func twoFormatEditions() *types.EditionIndex {
	return &types.EditionIndex{
		Version: 1,
		Editions: []types.Edition{{
			ID: "ed-1", Label: "First edition", Year: "2026", Status: "Published",
			Formats: []types.EditionFormat{
				{ID: "fmt-ebook", Kind: types.EditionKindEbook, Format: "eBook", ISBN13: "978-1-9471345-0-8"},
				{ID: "fmt-paper", Kind: types.EditionKindPrint, Format: "Paperback", ISBN13: "978-1-9471345-1-5", PageCount: "310"},
			},
		}},
	}
}

// membersUnder reads every archive member under a prefix and CLOSES the
// archive before returning. The close matters on Windows: the next save
// renames a temp file over this one, and an open reader makes that fail.
func membersUnder(t *testing.T, path, prefix string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	eachMember(t, path, func(file *zip.File) {
		if !strings.HasPrefix(file.Name, prefix) {
			return
		}
		data, err := ziputil.ReadEntry(file)
		if err != nil {
			t.Fatalf("cannot read %s: %v", file.Name, err)
		}
		out[file.Name] = data
	})
	return out
}

func eachMember(t *testing.T, path string, visit func(*zip.File)) {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("cannot open %s: %v", filepath.Base(path), err)
	}
	defer func() { _ = r.Close() }()
	for _, file := range r.File {
		visit(file)
	}
}

// The same unchanged text frozen for two formats is one stored copy that both
// records point at. This is the whole reason a snapshot is addressed by its
// contents rather than filed under the edition that froze it.
func TestTheSameTextFrozenTwiceIsStoredOnce(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "harbour.draftline")

	b := snapshotFixtureBook()
	index := twoFormatEditions()
	first, assets := freezeInto(t, index, "fmt-ebook", b)
	second, more := freezeInto(t, index, "fmt-paper", b)

	if first.Record.ID != second.Record.ID {
		t.Fatalf("the same text produced two identifiers: %s and %s", first.Record.ID, second.Record.ID)
	}
	for name, data := range more.Files {
		assets.Files[name] = data
	}
	if len(index.Snapshots) != 1 {
		t.Fatalf("the catalogue holds %d frozen manuscripts, want 1", len(index.Snapshots))
	}

	b.Editions = index
	if res := WriteArchive("", path, b, "test", nil, assets); !res.Success {
		t.Fatalf("save failed: %s", res.Error)
	}

	stored := membersUnder(t, path, snapshotsPrefix)
	if len(stored) != first.Record.Members {
		t.Fatalf("the project file holds %d snapshot members, want the %d of one snapshot", len(stored), first.Record.Members)
	}
	for name := range stored {
		if !strings.HasPrefix(name, SnapshotPrefix(first.Record.ID)) {
			t.Fatalf("a member outside the one frozen manuscript: %s", name)
		}
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	for _, format := range reopened.Editions.Editions[0].Formats {
		if format.SnapshotID != first.Record.ID {
			t.Fatalf("format %s points at %q, want %q", format.ID, format.SnapshotID, first.Record.ID)
		}
	}
}

// Freezing is deterministic, and the stamp on a save is not part of what makes
// one text different from another: an autosave between two exports must not
// make the same words look like a different book.
func TestTheSaveStampDoesNotChangeWhatTextIsFrozen(t *testing.T) {
	a := snapshotFixtureBook()
	a.Metadata.Modified = "2026-04-14T09:00:00Z"
	b := snapshotFixtureBook()
	b.Metadata.Modified = "2031-11-02T23:14:59Z"

	first := BuildSnapshot(a, time.Now())
	second := BuildSnapshot(b, time.Now().Add(72*time.Hour))
	if first.Record.ID != second.Record.ID {
		t.Fatalf("the save stamp changed the frozen identity: %s vs %s", first.Record.ID, second.Record.ID)
	}
	for name, data := range first.Files {
		if string(second.Files[name]) != string(data) {
			t.Fatalf("%s differs between two freezes of the same text", name)
		}
	}
}

// Edit one chapter and freeze again: two snapshots, both intact, and the first
// still byte for byte what it was.
func TestEditingAChapterFreezesASecondManuscriptAndLeavesTheFirstAlone(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "harbour.draftline")

	b := snapshotFixtureBook()
	index := twoFormatEditions()
	first, assets := freezeInto(t, index, "fmt-ebook", b)
	b.Editions = index
	if res := WriteArchive("", path, b, "test", nil, assets); !res.Success {
		t.Fatalf("first save failed: %s", res.Error)
	}
	before := membersUnder(t, path, SnapshotPrefix(first.Record.ID))

	revised := snapshotFixtureBook()
	revised.Body[1].Content = "<p>Ilsa counted the strokes and this time the number held.</p>"
	second, revisedAssets := freezeInto(t, index, "fmt-paper", revised)
	if second.Record.ID == first.Record.ID {
		t.Fatal("a changed chapter produced the same frozen identifier")
	}
	revised.Editions = index
	if res := WriteArchive(path, path, revised, "test", nil, revisedAssets); !res.Success {
		t.Fatalf("second save failed: %s", res.Error)
	}

	after := membersUnder(t, path, SnapshotPrefix(first.Record.ID))
	if len(after) != len(before) {
		t.Fatalf("the first frozen manuscript now has %d members, had %d", len(after), len(before))
	}
	for name, data := range before {
		if string(after[name]) != string(data) {
			t.Fatalf("%s changed under the first frozen manuscript", name)
		}
	}
	if len(membersUnder(t, path, SnapshotPrefix(second.Record.ID))) != second.Record.Members {
		t.Fatal("the second frozen manuscript did not reach the project file whole")
	}
}

// What comes back out is the book that went in: the order, the titles, the
// subtitles, the author's own copyright page and the metadata as it stood.
func TestAFrozenManuscriptComesBackAsTheBookThatWentIn(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "harbour.draftline")

	b := snapshotFixtureBook()
	b.Body[0].Subtitle = "in which the water arrives"
	index := twoFormatEditions()
	frozen, assets := freezeInto(t, index, "fmt-ebook", b)
	b.Editions = index
	if res := WriteArchive("", path, b, "test", nil, assets); !res.Success {
		t.Fatalf("save failed: %s", res.Error)
	}

	// The draft moves on, hard.
	moved := snapshotFixtureBook()
	moved.Metadata.Title = "The Salt Lantern: Revised"
	moved.Body = []types.ChapterItem{{ID: "ch-9", Title: "A Different Book", Type: "chapter", Content: "<p>Nothing of the old one survives.</p>"}}
	moved.Editions = index
	if res := WriteArchive(path, path, moved, "test", nil, Assets{}); !res.Success {
		t.Fatalf("second save failed: %s", res.Error)
	}

	back, err := ReadSnapshot(path, frozen.Record.ID)
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	if back.Metadata.Title != "The Salt Lantern" {
		t.Fatalf("frozen title is %q, want the one that was frozen", back.Metadata.Title)
	}
	if back.Metadata.Subtitle != b.Metadata.Subtitle || back.Metadata.Language != "en-GB" {
		t.Fatalf("frozen metadata came back wrong: %+v", back.Metadata)
	}
	if back.Copyright != b.Copyright {
		t.Fatalf("the author's own copyright page did not come back: %q", back.Copyright)
	}
	if len(back.Body) != 2 || back.Body[0].Title != "The Tide Gate" || back.Body[1].Title != "Bell and Rope" {
		t.Fatalf("section order or titles came back wrong: %+v", back.Body)
	}
	if back.Body[0].Subtitle != "in which the water arrives" {
		t.Fatalf("a section subtitle was not frozen: %q", back.Body[0].Subtitle)
	}
	if back.Body[1].Content != b.Body[1].Content {
		t.Fatalf("frozen text came back as %q", back.Body[1].Content)
	}
	if len(back.FrontMatter) != 1 || back.FrontMatter[0].Title != "Dedication" {
		t.Fatalf("front matter came back wrong: %+v", back.FrontMatter)
	}
	if len(back.BackMatter) != 1 || back.BackMatter[0].Type != "afterword" {
		t.Fatalf("back matter came back wrong: %+v", back.BackMatter)
	}
}

// Deleting a format that shares its words with another leaves the words.
// Deleting the last one that points at them lets them go, and does not take
// anything else with them.
func TestFrozenTextGoesOnlyWhenNoFormatIsPublishedFromIt(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "harbour.draftline")

	shared := snapshotFixtureBook()
	other := snapshotFixtureBook()
	other.Body[0].Content = "<p>A different opening entirely.</p>"

	index := twoFormatEditions()
	index.Editions[0].Formats = append(index.Editions[0].Formats,
		types.EditionFormat{ID: "fmt-hard", Kind: types.EditionKindPrint, Format: "Hardcover", PageCount: "330"})

	sharedSnap, assets := freezeInto(t, index, "fmt-ebook", shared)
	_, more := freezeInto(t, index, "fmt-paper", shared)
	otherSnap, evenMore := freezeInto(t, index, "fmt-hard", other)
	for _, extra := range []Assets{more, evenMore} {
		for name, data := range extra.Files {
			assets.Files[name] = data
		}
	}

	b := shared
	b.Editions = index
	if res := WriteArchive("", path, b, "test", nil, assets); !res.Success {
		t.Fatalf("save failed: %s", res.Error)
	}

	// Drop one of the two formats published from the shared words.
	index.Editions[0].Formats = index.Editions[0].Formats[1:]
	b.Editions = index
	if res := WriteArchive(path, path, b, "test", nil, Assets{}); !res.Success {
		t.Fatalf("save after dropping one format failed: %s", res.Error)
	}
	if len(membersUnder(t, path, SnapshotPrefix(sharedSnap.Record.ID))) == 0 {
		t.Fatal("dropping one of two formats destroyed the text the other was published from")
	}

	// Drop the last one.
	index.Editions[0].Formats = index.Editions[0].Formats[1:]
	b.Editions = index
	if res := WriteArchive(path, path, b, "test", nil, Assets{}); !res.Success {
		t.Fatalf("save after dropping the last format failed: %s", res.Error)
	}
	if left := membersUnder(t, path, SnapshotPrefix(sharedSnap.Record.ID)); len(left) != 0 {
		t.Fatalf("unreferenced frozen text stayed in the project file: %v", left)
	}
	if len(membersUnder(t, path, SnapshotPrefix(otherSnap.Record.ID))) != otherSnap.Record.Members {
		t.Fatal("dropping one frozen manuscript took another with it")
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if len(reopened.Editions.Snapshots) != 1 || reopened.Editions.Snapshots[0].ID != otherSnap.Record.ID {
		t.Fatalf("the catalogue came back as %+v", reopened.Editions.Snapshots)
	}
}

// The one bookkeeping mistake that must never be allowed to decide a deletion.
func TestASaveRefusesToLoseTextAFormatIsStillPublishedFrom(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "harbour.draftline")

	b := snapshotFixtureBook()
	index := twoFormatEditions()
	frozen, assets := freezeInto(t, index, "fmt-ebook", b)
	b.Editions = index
	if res := WriteArchive("", path, b, "test", nil, assets); !res.Success {
		t.Fatalf("save failed: %s", res.Error)
	}

	// The catalogue record goes but the format still names it.
	index.Snapshots = nil
	b.Editions = index
	res := WriteArchive(path, path, b, "test", nil, Assets{})
	if res.Success {
		t.Fatal("the save accepted a format published from a manuscript the book no longer lists")
	}
	for _, want := range []string{editionsIndexFile, frozen.Record.ID, "will not drop text"} {
		if !strings.Contains(res.Error, want) {
			t.Fatalf("the refusal does not name %q: %s", want, res.Error)
		}
	}
	if len(membersUnder(t, path, SnapshotPrefix(frozen.Record.ID))) == 0 {
		t.Fatal("the refused save destroyed the frozen text anyway")
	}
}

// A hundred autosaves, which is roughly eight minutes of writing.
func TestASnapshotBearingProjectSurvivesAHundredSaves(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "harbour.draftline")

	b := snapshotFixtureBook()
	index := twoFormatEditions()
	frozen, assets := freezeInto(t, index, "fmt-ebook", b)
	b.Editions = index
	if res := WriteArchive("", path, b, "test", nil, assets); !res.Success {
		t.Fatalf("first save failed: %s", res.Error)
	}
	want := membersUnder(t, path, SnapshotPrefix(frozen.Record.ID))
	if len(want) == 0 {
		t.Fatal("nothing was frozen into the project file")
	}

	for i := 0; i < 100; i++ {
		b.Body[0].Content = fmt.Sprintf("<p>The gate stood open, %d.</p>", i)
		var snapshots []types.ChapterSnapshotRequest
		if i%2 == 1 {
			snapshots = []types.ChapterSnapshotRequest{{ChapterID: "ch-1", Section: "body", ChapterTitle: "The Tide Gate", Content: b.Body[0].Content}}
		}
		if res := WriteArchive(path, path, b, "test", snapshots, Assets{}); !res.Success {
			t.Fatalf("save %d failed: %s", i, res.Error)
		}
	}

	got := membersUnder(t, path, SnapshotPrefix(frozen.Record.ID))
	if len(got) != len(want) {
		t.Fatalf("after a hundred saves the frozen manuscript has %d members, had %d", len(got), len(want))
	}
	for name, data := range want {
		if string(got[name]) != string(data) {
			t.Fatalf("%s changed over a hundred saves", name)
		}
	}
	var files []*zip.File
	eachMember(t, path, func(file *zip.File) { files = append(files, file) })
	if err := ziputil.CheckArchive(files); err != nil {
		t.Fatalf("the project file left the archive limits: %v", err)
	}
	if _, err := Open(path); err != nil {
		t.Fatalf("the project no longer opens: %v", err)
	}
}

// Frozen manuscripts are the first thing that makes the archive's own limits
// reachable, so exceeding one has to say what filled the file.
func TestAProjectTooFullToSaveNamesTheFrozenManuscriptsInIt(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "huge.draftline")

	// A book with more sections than the archive can hold pieces, frozen once.
	b := snapshotFixtureBook()
	b.Body = nil
	for i := 0; i < ziputil.MaxEntries+50; i++ {
		b.Body = append(b.Body, types.ChapterItem{
			ID: fmt.Sprintf("ch-%d", i), Title: fmt.Sprintf("Section %d", i),
			Type: "chapter", Content: "<p>A line.</p>",
		})
	}
	index := twoFormatEditions()
	snapshot := BuildSnapshot(b, time.Now())
	index.Editions[0].Formats[0].SnapshotID = snapshot.Record.ID
	index.Snapshots = []types.EditionSnapshot{snapshot.Record}

	// Only the frozen manuscript goes in, so the book itself stays small.
	small := snapshotFixtureBook()
	small.Editions = index
	res := WriteArchive("", path, small, "test", nil, Assets{Files: snapshot.Files})
	if res.Success {
		t.Fatal("a project past the archive's own limits saved anyway")
	}
	if !strings.Contains(res.Error, "frozen") {
		t.Fatalf("the refusal does not mention frozen manuscripts: %s", res.Error)
	}
	if !strings.Contains(res.Error, "Book & Editions") {
		t.Fatalf("the refusal does not say where to go: %s", res.Error)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("the refused save left a file behind")
	}
}

// A snapshot written by a later Draftline is refused by name, never coerced.
func TestAFrozenManuscriptFromTheFutureIsRefusedByName(t *testing.T) {
	future, _ := json.Marshal(map[string]any{"version": 2, "metadata": map[string]string{"title": "Later"}})
	_, err := DecodeSnapshot("abc123", func(name string) ([]byte, error) {
		if name == snapshotIndexName {
			return future, nil
		}
		return nil, fmt.Errorf("no such member")
	})
	if err == nil {
		t.Fatal("a version this build does not know was read anyway")
	}
	for _, want := range []string{"version 2", "version 1", snapshotIndexName} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the refusal does not name %q: %s", want, err)
		}
	}
}

// "snapshots" is the folder frozen manuscripts live in, so it cannot also be
// an edition: an edition claiming it would file its cover among them.
func TestAnEditionCannotBeCalledSnapshots(t *testing.T) {
	index := types.EditionIndex{
		Version:  1,
		Editions: []types.Edition{{ID: snapshotsSegment, Label: "Odd", Formats: []types.EditionFormat{}}},
	}
	if _, err := prepareEditionsData(index); err == nil {
		t.Fatal("an edition called \"snapshots\" was accepted")
	}
}

// The reaping must not mistake the shared snapshots folder for an edition's
// own folder, or the first save after registering an edition would sweep every
// published edition's text away.
func TestTheEditionReaperLeavesFrozenManuscriptsAlone(t *testing.T) {
	if _, ok := editionFolder(snapshotsPrefix + "abc/index.json"); ok {
		t.Fatal("a frozen manuscript is being read as an edition's own folder")
	}
	live := map[string]bool{"ed-1": true}
	if orphanedEditionMember(snapshotsPrefix+"abc/000.html", live) {
		t.Fatal("the edition reaper would drop a frozen manuscript")
	}
	if !orphanedSnapshotMember(snapshotsPrefix+"abc/000.html", map[string]bool{"def": true}) {
		t.Fatal("an unreferenced frozen manuscript is not being reaped")
	}
	if orphanedSnapshotMember(snapshotsPrefix+"abc/000.html", nil) {
		t.Fatal("a save that does not know the record reaped anyway")
	}
}

// Frozen text is deflated, and cover art is not.
//
// A frozen manuscript is HTML. Nothing compressed it on the way in — the same
// words are already in the same file, under body/, at about a fifth of this
// size — so storing it uncompressed multiplies what an edition costs the
// project file, every rolling backup of it, and every save. A JPEG is the
// other way round: it arrives compressed and a second pass is work for
// nothing. The member decides, not the code path.
func TestFrozenTextIsDeflatedAndCoverArtIsNot(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "harbour.draftline")

	b := snapshotFixtureBook()
	index := twoFormatEditions()
	frozen, assets := freezeInto(t, index, "fmt-ebook", b)
	index.Editions[0].Cover = &types.EditionCover{File: "cover.jpg", ThumbFile: "cover_thumb.jpg", Width: 1600, Height: 2560}
	// Bytes a JPEG could plausibly be: incompressible, so a deflate pass would
	// be measurable work and no saving.
	jpeg := make([]byte, 4096)
	for i := range jpeg {
		jpeg[i] = byte(i*7 + i/3)
	}
	assets.Files[CoverMember("ed-1", "cover.jpg")] = jpeg
	assets.Files[CoverMember("ed-1", "cover_thumb.jpg")] = jpeg[:512]
	b.Editions = index
	if res := WriteArchive("", path, b, "test", nil, assets); !res.Success {
		t.Fatalf("save failed: %s", res.Error)
	}

	var text, packed uint64
	eachMember(t, path, func(file *zip.File) {
		switch {
		case strings.HasPrefix(file.Name, SnapshotPrefix(frozen.Record.ID)):
			if file.Method != zip.Deflate {
				t.Fatalf("%s was stored uncompressed", file.Name)
			}
			text += file.UncompressedSize64
			packed += file.CompressedSize64
		case file.Name == CoverMember("ed-1", "cover.jpg"):
			if file.Method != zip.Store {
				t.Fatalf("cover art was deflated a second time")
			}
			if file.CompressedSize64 != file.UncompressedSize64 {
				t.Fatalf("cover art is %d stored bytes for %d", file.CompressedSize64, file.UncompressedSize64)
			}
		}
	})
	if text == 0 {
		t.Fatal("no frozen text reached the archive")
	}
	if packed >= text {
		t.Fatalf("the frozen manuscript packed to %d bytes from %d, which is no saving", packed, text)
	}
}
