package book

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"draftline/internal/types"
)

// injectStoredEntry rewrites an archive with one extra stored member, standing
// in for data that lives in the file and is not rebuilt from BookData on save.
func injectStoredEntry(t *testing.T, path, name string, data []byte) {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("cannot open fixture archive: %v", err)
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, file := range r.File {
		if err := w.Copy(file); err != nil {
			t.Fatalf("cannot copy fixture entry %q: %v", file.Name, err)
		}
	}
	_ = r.Close()
	f, err := w.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
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

func archiveEntries(t *testing.T, path string) map[string]*zip.File {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("cannot open archive %s: %v", filepath.Base(path), err)
	}
	t.Cleanup(func() { _ = r.Close() })
	entries := map[string]*zip.File{}
	for _, file := range r.File {
		entries[file.Name] = file
	}
	return entries
}

func entryBytes(t *testing.T, file *zip.File) []byte {
	t.Helper()
	rc, err := file.Open()
	if err != nil {
		t.Fatalf("cannot open entry %q: %v", file.Name, err)
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("cannot read entry %q: %v", file.Name, err)
	}
	return data
}

func writingSnapshot() types.ChapterSnapshotRequest {
	return types.ChapterSnapshotRequest{
		ChapterID:    "ch-one",
		Section:      "body",
		ChapterTitle: "Chapter One",
		Content:      "<p>The harbour bell rang twice and Mirren counted both.</p>",
		Reason:       "Writing session",
	}
}

// A member the save path does not rebuild must still be in the file after an
// afternoon of autosaves, byte for byte and still stored rather than deflated.
// Before the passthrough became a prefix list, the first save five seconds
// later destroyed it.
func TestPreservedEntrySurvivesRepeatedSaves(t *testing.T) {
	for _, withHistory := range []bool{false, true} {
		name := "without chapter snapshots"
		if withHistory {
			name = "with chapter snapshots"
		}
		t.Run(name, func(t *testing.T) {
			isolateConfigDir(t)
			path := filepath.Join(t.TempDir(), "book.draftline")
			b := testBook()
			b.Body[0].ID = "ch-one"
			if res := Write("", path, b, "v1"); !res.Success {
				t.Fatalf("Write failed: %s", res.Error)
			}

			probe := bytes.Repeat([]byte("probe-bytes-"), 512)
			injectStoredEntry(t, path, "editions/probe.bin", probe)

			for i := 0; i < 100; i++ {
				var res types.SaveResult
				if withHistory {
					res = WriteWithSnapshots(path, path, b, "v1", []types.ChapterSnapshotRequest{writingSnapshot()})
				} else {
					res = Write(path, path, b, "v1")
				}
				if !res.Success {
					t.Fatalf("save %d failed: %s", i+1, res.Error)
				}
			}

			entries := archiveEntries(t, path)
			file := entries["editions/probe.bin"]
			if file == nil {
				t.Fatal("archived data outside the rebuilt set was destroyed by saving")
			}
			if got := entryBytes(t, file); !bytes.Equal(got, probe) {
				t.Fatalf("preserved entry changed after 100 saves: %d bytes, wanted %d", len(got), len(probe))
			}
			if file.Method != zip.Store {
				t.Fatalf("preserved entry was recompressed: method %d", file.Method)
			}
			if file.CompressedSize64 != file.UncompressedSize64 {
				t.Fatalf("stored entry reports %d compressed of %d uncompressed", file.CompressedSize64, file.UncompressedSize64)
			}
			if withHistory && entries[historyIndexFile] == nil {
				t.Fatal("chapter history was lost while preserving other data")
			}
		})
	}
}

// Binary assets are stored, not deflated: an autosave every five seconds must
// not spend its time recompressing a JPEG.
func TestAddBytesStoresAssetsWhileTextIsDeflated(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	aw := newArchiveWriter(w)

	asset := bytes.Repeat([]byte("cover"), 4096)
	if err := aw.addBytes("editions/e1/cover.jpg", asset); err != nil {
		t.Fatalf("addBytes failed: %v", err)
	}
	if err := aw.addEntry("editions/index.json", strings.Repeat(`{"edition":1}`, 4096)); err != nil {
		t.Fatalf("addEntry failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("cannot finalize archive: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("produced archive is invalid: %v", err)
	}
	entries := map[string]*zip.File{}
	for _, f := range r.File {
		entries[f.Name] = f
	}

	cover := entries["editions/e1/cover.jpg"]
	if cover == nil {
		t.Fatal("cover entry is missing")
	}
	if cover.Method != zip.Store || cover.CompressedSize64 != uint64(len(asset)) {
		t.Fatalf("cover was deflated: method %d, %d bytes of %d", cover.Method, cover.CompressedSize64, len(asset))
	}
	if !bytes.Equal(entryBytes(t, cover), asset) {
		t.Fatal("stored asset did not round-trip")
	}

	index := entries["editions/index.json"]
	if index == nil {
		t.Fatal("index entry is missing")
	}
	if index.Method != zip.Deflate || index.CompressedSize64 >= index.UncompressedSize64 {
		t.Fatalf("text entry was not deflated: method %d, %d of %d", index.Method, index.CompressedSize64, index.UncompressedSize64)
	}
}

// A ZIP can hold one name twice, and readers disagree about which copy wins.
// Two writers into one archive must collide loudly instead.
func TestDuplicateArchiveEntryIsRefused(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	aw := newArchiveWriter(w)

	if err := aw.addEntry("editions/index.json", "{}"); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	err := aw.addBytes("editions/index.json", []byte("{}"))
	if err == nil {
		t.Fatal("a duplicate archive entry was written twice")
	}
	if !strings.Contains(err.Error(), "editions/index.json") {
		t.Fatalf("duplicate error does not name the entry: %v", err)
	}
	if err := aw.addEntry("editions/index.json", "{}"); err == nil {
		t.Fatal("a duplicate text entry was written twice")
	}
	if err := w.Close(); err != nil {
		t.Fatalf("cannot finalize archive: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("produced archive is invalid: %v", err)
	}
	if len(r.File) != 1 {
		t.Fatalf("archive holds %d entries, wanted 1", len(r.File))
	}
}

// A save must not carry preserved data over from a project it is not saving.
func TestPreservedEntriesComeFromTheSourceArchive(t *testing.T) {
	isolateConfigDir(t)
	dir := t.TempDir()
	other := filepath.Join(dir, "other.draftline")
	if res := Write("", other, testBook(), "v1"); !res.Success {
		t.Fatalf("Write failed: %s", res.Error)
	}
	injectStoredEntry(t, other, "editions/probe.bin", []byte("not this book"))

	if res := Write("", other, testBook(), "v1"); !res.Success {
		t.Fatalf("Write failed: %s", res.Error)
	}
	if archiveEntries(t, other)["editions/probe.bin"] != nil {
		t.Fatal("a save with no source inherited the destination's archived data")
	}
}

// Data written by a newer Draftline is refused by name rather than coerced into
// the shape this build expects and written back lossily.
func TestRequireArchiveVersionRefusesUnknownVersions(t *testing.T) {
	if err := requireArchiveVersion("editions data", 1, 1); err != nil {
		t.Fatalf("the supported version was refused: %v", err)
	}
	err := requireArchiveVersion("editions data", 2, 1)
	if err == nil {
		t.Fatal("data from a newer Draftline was accepted")
	}
	if !strings.Contains(err.Error(), "editions data") || !strings.Contains(err.Error(), "version 2") {
		t.Fatalf("refusal does not say what it refused: %v", err)
	}
	if err := requireArchiveVersion("editions data", 0, 1); err == nil {
		t.Fatal("unversioned data was accepted")
	}
}

// A source archive that cannot be read must not trap the author's text in
// memory. Saving over the same file still refuses, because the unreadable
// history is still in that file and overwriting it would destroy it; saving to
// a different file carries nothing across and says so, leaving the original
// where it is.
func TestSaveAsWritesEvenWhenTheSourceArchiveCannotBeRead(t *testing.T) {
	isolateConfigDir(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "source.draftline")
	if res := Write("", source, testBook(), "v1"); !res.Success {
		t.Fatalf("Write failed: %s", res.Error)
	}
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("cannot read fixture: %v", err)
	}
	if err := os.WriteFile(source, original[:len(original)/2], 0o600); err != nil {
		t.Fatalf("cannot truncate fixture: %v", err)
	}

	dest := filepath.Join(dir, "rescue.draftline")
	res := Write(source, dest, testBook(), "v1")
	if !res.Success {
		t.Fatalf("Save As to a fresh path was refused because the source is damaged: %s", res.Error)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("the rescued save carried nothing across and did not say so")
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("no rescue file was written: %v", err)
	}
	if archiveEntries(t, dest)["manifest.json"] == nil {
		t.Fatal("the rescue file is not a readable project")
	}

	if res := Write(source, source, testBook(), "v1"); res.Success {
		t.Fatal("saving over the damaged file itself destroyed what could not be read")
	}
}

// An archive that holds one preserved name twice still opens, so it must still
// save. Keeping the first copy is what every reader of the file already does.
func TestDuplicatePreservedMemberDoesNotBlockTheSave(t *testing.T) {
	for _, name := range []string{"editions/probe.bin", historyIndexFile} {
		t.Run(name, func(t *testing.T) {
			isolateConfigDir(t)
			dir := t.TempDir()
			path := filepath.Join(dir, "book.draftline")
			if res := Write("", path, testBook(), "v1"); !res.Success {
				t.Fatalf("Write failed: %s", res.Error)
			}
			injectStoredEntry(t, path, name, []byte("first copy"))
			injectStoredEntry(t, path, name, []byte("second copy"))

			if res := Write(path, path, testBook(), "v1"); !res.Success {
				t.Fatalf("a book that opens could not be saved: %s", res.Error)
			}
			rescue := filepath.Join(dir, "rescue.draftline")
			if res := Write(path, rescue, testBook(), "v1"); !res.Success {
				t.Fatalf("Save As was refused as well, leaving no route to disk: %s", res.Error)
			}

			r, err := zip.OpenReader(path)
			if err != nil {
				t.Fatalf("cannot open saved archive: %v", err)
			}
			defer func() { _ = r.Close() }()
			var copies int
			var kept []byte
			for _, f := range r.File {
				if f.Name == name {
					copies++
					if copies == 1 {
						kept = entryBytes(t, f)
					}
				}
			}
			if copies != 1 {
				t.Fatalf("the saved archive holds %d copies of the duplicated name, wanted 1", copies)
			}
			if string(kept) != "first copy" {
				t.Fatalf("the save kept %q, but a reader takes the first copy", kept)
			}
		})
	}
}

// The rescue still keeps this session's writing: snapshots taken now owe
// nothing to the file that cannot be read.
func TestRescueSaveKeepsTheSnapshotsTakenNow(t *testing.T) {
	isolateConfigDir(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "source.draftline")
	b := testBook()
	b.Body[0].ID = "ch-one"
	if res := WriteWithSnapshots("", source, b, "v1", []types.ChapterSnapshotRequest{writingSnapshot()}); !res.Success {
		t.Fatalf("Write failed: %s", res.Error)
	}
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("cannot read fixture: %v", err)
	}
	if err := os.WriteFile(source, original[:len(original)/2], 0o600); err != nil {
		t.Fatalf("cannot truncate fixture: %v", err)
	}

	fresh := writingSnapshot()
	fresh.Content = "<p>Mirren wrote the last of it while the tide went out.</p>"
	dest := filepath.Join(dir, "rescue.draftline")
	res := WriteWithSnapshots(source, dest, b, "v1", []types.ChapterSnapshotRequest{fresh})
	if !res.Success {
		t.Fatalf("a snapshot save to a fresh path was refused: %s", res.Error)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("the rescue reported %d warnings, wanted one: %v", len(res.Warnings), res.Warnings)
	}
	entries := archiveEntries(t, dest)
	if entries[historyIndexFile] == nil {
		t.Fatal("the snapshot taken during the rescue was not written")
	}
}
