package worktree

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Every fixture here is invented. No manuscript text goes into a test.

func writeArchive(t *testing.T, path string, members map[string]string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range members {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create %q: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("write %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}
}

func readArchive(t *testing.T, path string) map[string]string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer r.Close()
	out := map[string]string{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open member %q: %v", f.Name, err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			t.Fatalf("read member %q: %v", f.Name, err)
		}
		rc.Close()
		out[f.Name] = buf.String()
	}
	return out
}

func book() map[string]string {
	return map[string]string{
		"manifest.json":    `{"version":"2.2","body":[{"title":"Chapter 1","file":"body/000.html"}]}`,
		"body/000.html":    "<p>The first chapter, as it stands.</p>",
		"history/h1.html":  "<p>An older draft.</p>",
		"editions/cov.jpg": "\xff\xd8\xff\xe0binary",
	}
}

// fixture makes a temp archive and a temp worktree root.
func fixture(t *testing.T) (archive string, root string) {
	t.Helper()
	dir := t.TempDir()
	archive = filepath.Join(dir, "The Weather House.draftline")
	root = filepath.Join(dir, "appdata")
	writeArchive(t, archive, book())
	return archive, root
}

func TestDirForIsHiddenNamedAndPathUnique(t *testing.T) {
	root := "/data"
	a := DirFor(root, "/books/novel.draftline")
	b := DirFor(root, "/elsewhere/novel.draftline")

	if a == b {
		t.Fatal("two books with the same filename in different folders share a working copy")
	}
	base := filepath.Base(a)
	if !strings.HasPrefix(base, ".novel.draftline-") {
		t.Fatalf("working directory %q is not named for its book", base)
	}
	if DirFor(root, "/books/novel.draftline") != a {
		t.Fatal("DirFor is not stable for the same path")
	}
}

func TestOpenExtractsEveryMember(t *testing.T) {
	archive, root := fixture(t)
	tree, state, err := Open(archive, Options{Root: root, Autosave: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if state != StateExtracted {
		t.Fatalf("state = %v, want extracted", state)
	}
	members, err := tree.Members()
	if err != nil {
		t.Fatalf("Members: %v", err)
	}
	want := []string{"body/000.html", "editions/cov.jpg", "history/h1.html", "manifest.json"}
	if strings.Join(members, ",") != strings.Join(want, ",") {
		t.Fatalf("members = %v, want %v", members, want)
	}
	if tree.Dirty() {
		t.Fatal("a freshly extracted working copy is not dirty")
	}
}

func TestRepackPreservesEveryMemberIncludingOnesNobodyTouched(t *testing.T) {
	archive, root := fixture(t)
	tree, _, err := Open(archive, Options{Root: root, Autosave: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := tree.WriteMember("body/000.html", []byte("<p>Edited.</p>")); err != nil {
		t.Fatalf("WriteMember: %v", err)
	}
	if !tree.Dirty() {
		t.Fatal("a written member did not mark the working copy dirty")
	}
	if err := tree.Repack(); err != nil {
		t.Fatalf("Repack: %v", err)
	}

	got := readArchive(t, archive)
	if got["body/000.html"] != "<p>Edited.</p>" {
		t.Fatalf("chapter = %q, want the edit", got["body/000.html"])
	}
	// The whole point: the desktop's members survive a repack untouched.
	if got["history/h1.html"] != "<p>An older draft.</p>" {
		t.Fatalf("chapter history was lost: %q", got["history/h1.html"])
	}
	if got["editions/cov.jpg"] != "\xff\xd8\xff\xe0binary" {
		t.Fatalf("edition artwork was lost or altered: %q", got["editions/cov.jpg"])
	}
	if tree.Dirty() {
		t.Fatal("still dirty after a repack")
	}
}

func TestRepackIsANoOpWhenNothingChanged(t *testing.T) {
	archive, root := fixture(t)
	tree, _, err := Open(archive, Options{Root: root, Autosave: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	before, err := os.Stat(archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := tree.Repack(); err != nil {
		t.Fatalf("Repack: %v", err)
	}
	after, err := os.Stat(archive)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("a clean repack rewrote the archive; that is a pointless upload for every sync client watching")
	}
}

func TestRemovedMemberDoesNotComeBack(t *testing.T) {
	archive, root := fixture(t)
	tree, _, _ := Open(archive, Options{Root: root, Autosave: true})
	if err := tree.RemoveMember("body/000.html"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	if err := tree.Repack(); err != nil {
		t.Fatalf("Repack: %v", err)
	}
	if _, ok := readArchive(t, archive)["body/000.html"]; ok {
		t.Fatal("a deleted chapter survived the repack")
	}
}

func TestRepackRefusesToWriteABookWithNoManifest(t *testing.T) {
	archive, root := fixture(t)
	tree, _, _ := Open(archive, Options{Root: root, Autosave: true})
	if err := tree.RemoveMember("manifest.json"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	if err := tree.Repack(); err == nil {
		t.Fatal("Repack wrote a book with no manifest over a good archive")
	}
	// And the archive on disk is still the book it was.
	if _, ok := readArchive(t, archive)["manifest.json"]; !ok {
		t.Fatal("the archive lost its manifest anyway")
	}
}

func TestReopenReusesAnUnchangedWorkingCopy(t *testing.T) {
	archive, root := fixture(t)
	tree, _, _ := Open(archive, Options{Root: root, Autosave: true})
	if err := tree.WriteMember("body/000.html", []byte("<p>Edited.</p>")); err != nil {
		t.Fatal(err)
	}
	if err := tree.Repack(); err != nil {
		t.Fatal(err)
	}

	reopened, state, err := Open(archive, Options{Root: root, Autosave: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if state != StateReused {
		t.Fatalf("state = %v, want reused", state)
	}
	data, err := reopened.ReadMember("body/000.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "<p>Edited.</p>" {
		t.Fatalf("member = %q, want the edit", data)
	}
}

func TestCrashWithAutosaveOnRecoversTheWorkingCopy(t *testing.T) {
	archive, root := fixture(t)
	tree, _, _ := Open(archive, Options{Root: root, Autosave: true})
	// Written but never packed: the process died here.
	if err := tree.WriteMember("body/000.html", []byte("<p>Work that never reached the zip.</p>")); err != nil {
		t.Fatal(err)
	}

	recovered, state, err := Open(archive, Options{Root: root, Autosave: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if state != StateRecovered {
		t.Fatalf("state = %v, want recovered", state)
	}
	if !recovered.Dirty() {
		t.Fatal("a recovered working copy must still be dirty; it is ahead of the archive")
	}
	data, _ := recovered.ReadMember("body/000.html")
	if string(data) != "<p>Work that never reached the zip.</p>" {
		t.Fatalf("recovered member = %q; the unsaved work was lost", data)
	}
	if err := recovered.Repack(); err != nil {
		t.Fatal(err)
	}
	if readArchive(t, archive)["body/000.html"] != "<p>Work that never reached the zip.</p>" {
		t.Fatal("the recovered work did not reach the archive")
	}
}

func TestUnsavedWorkIsDiscardedWhenAutosaveWasOff(t *testing.T) {
	archive, root := fixture(t)
	tree, _, _ := Open(archive, Options{Root: root, Autosave: false})
	if err := tree.WriteMember("body/000.html", []byte("<p>Never saved, and they said no.</p>")); err != nil {
		t.Fatal(err)
	}

	reopened, state, err := Open(archive, Options{Root: root, Autosave: false})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if state != StateDiscarded {
		t.Fatalf("state = %v, want discarded", state)
	}
	data, _ := reopened.ReadMember("body/000.html")
	if string(data) != "<p>The first chapter, as it stands.</p>" {
		t.Fatalf("member = %q, want the archive's copy back", data)
	}
	if reopened.Dirty() {
		t.Fatal("a discarded working copy is clean")
	}
}

func TestDiscardRestoresTheArchivesCopy(t *testing.T) {
	archive, root := fixture(t)
	tree, _, _ := Open(archive, Options{Root: root, Autosave: false})
	if err := tree.WriteMember("body/000.html", []byte("<p>Thrown away.</p>")); err != nil {
		t.Fatal(err)
	}
	if err := tree.Discard(); err != nil {
		t.Fatalf("Discard: %v", err)
	}
	data, _ := tree.ReadMember("body/000.html")
	if string(data) != "<p>The first chapter, as it stands.</p>" {
		t.Fatalf("member = %q, want the archive's copy", data)
	}
	if tree.Dirty() {
		t.Fatal("still dirty after Discard")
	}
}

// conflicted builds a working copy holding unsaved work whose archive has been
// replaced underneath it, which is what a sync client bringing down a copy
// edited on another machine looks like.
func conflicted(t *testing.T) (archive, root string) {
	t.Helper()
	archive, root = fixture(t)
	tree, _, err := Open(archive, Options{Root: root, Autosave: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := tree.WriteMember("body/000.html", []byte("<p>Written on the phone.</p>")); err != nil {
		t.Fatal(err)
	}
	// Slept so the archive's modification time is measurably different; the
	// size alone would not move, both copies being one paragraph.
	time.Sleep(10 * time.Millisecond)
	changed := book()
	changed["body/000.html"] = "<p>Written on the laptop.</p>"
	writeArchive(t, archive, changed)
	return archive, root
}

func TestArchiveChangedUnderneathUnsavedWorkIsAConflict(t *testing.T) {
	archive, root := conflicted(t)
	tree, state, err := Open(archive, Options{Root: root, Autosave: true})
	if state != StateConflict || err == nil {
		t.Fatalf("state = %v, err = %v; want a conflict rather than a guess", state, err)
	}
	if tree != nil {
		t.Fatal("a conflicted Open handed back a usable tree")
	}
}

func TestConflictResolvedByKeepingTheWorkingCopy(t *testing.T) {
	archive, root := conflicted(t)
	tree, state, err := Open(archive, Options{Root: root, Autosave: true, Resolve: ResolveKeepWorkingCopy})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if state != StateRecovered {
		t.Fatalf("state = %v, want recovered", state)
	}
	data, _ := tree.ReadMember("body/000.html")
	if string(data) != "<p>Written on the phone.</p>" {
		t.Fatalf("member = %q, want the phone's copy kept", data)
	}
	// Resolving settles it: the next open is an ordinary recovery, not the
	// same question asked again.
	if _, state, err := Open(archive, Options{Root: root, Autosave: true}); err != nil || state != StateRecovered {
		t.Fatalf("second open: state = %v, err = %v; want a settled recovery", state, err)
	}
}

func TestConflictResolvedByTakingTheArchive(t *testing.T) {
	archive, root := conflicted(t)
	tree, state, err := Open(archive, Options{Root: root, Autosave: true, Resolve: ResolveTakeArchive})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if state != StateExtracted {
		t.Fatalf("state = %v, want extracted", state)
	}
	data, _ := tree.ReadMember("body/000.html")
	if string(data) != "<p>Written on the laptop.</p>" {
		t.Fatalf("member = %q, want the archive's copy taken", data)
	}
	if tree.Dirty() {
		t.Fatal("taking the archive leaves a clean working copy")
	}
}

func TestAChangedArchiveReplacesACleanWorkingCopy(t *testing.T) {
	archive, root := fixture(t)
	if _, _, err := Open(archive, Options{Root: root, Autosave: true}); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	changed := book()
	changed["body/000.html"] = "<p>Edited elsewhere.</p>"
	writeArchive(t, archive, changed)

	tree, state, err := Open(archive, Options{Root: root, Autosave: true})
	if err != nil {
		t.Fatal(err)
	}
	if state != StateExtracted {
		t.Fatalf("state = %v, want extracted", state)
	}
	data, _ := tree.ReadMember("body/000.html")
	if string(data) != "<p>Edited elsewhere.</p>" {
		t.Fatalf("member = %q, want the newer archive", data)
	}
}

func TestArchiveMembersCannotEscapeTheWorkingCopy(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "hostile.draftline")
	writeArchive(t, archive, map[string]string{
		"manifest.json":     `{"version":"2.2"}`,
		"../../escaped.txt": "should never be written",
	})

	if _, _, err := Open(archive, Options{Root: filepath.Join(dir, "appdata"), Autosave: true}); err == nil {
		t.Fatal("an archive member climbing out of the working copy was accepted")
	}
	if _, err := os.Stat(filepath.Join(dir, "escaped.txt")); err == nil {
		t.Fatal("a file was written outside the working copy")
	}
}

func TestWriteMemberCannotEscapeTheWorkingCopy(t *testing.T) {
	archive, root := fixture(t)
	tree, _, _ := Open(archive, Options{Root: root, Autosave: true})
	if err := tree.WriteMember("../escaped.txt", []byte("no")); err != nil {
		t.Fatalf("WriteMember should normalise, not fail: %v", err)
	}
	// Normalised into the tree, not out of it.
	if _, err := os.Stat(filepath.Join(root, "escaped.txt")); err == nil {
		t.Fatal("WriteMember wrote outside the working copy")
	}
}

func TestSetAutosaveSurvivesIntoTheNextSession(t *testing.T) {
	archive, root := fixture(t)
	tree, _, _ := Open(archive, Options{Root: root, Autosave: true})
	if err := tree.SetAutosave(false); err != nil {
		t.Fatalf("SetAutosave: %v", err)
	}
	if err := tree.WriteMember("body/000.html", []byte("<p>Unsaved.</p>")); err != nil {
		t.Fatal(err)
	}

	// Opening again in manual mode must honour the mode the work was done in,
	// not the mode this call happens to ask for.
	_, state, err := Open(archive, Options{Root: root, Autosave: true})
	if err != nil {
		t.Fatal(err)
	}
	if state != StateDiscarded {
		t.Fatalf("state = %v, want discarded: the work was done with autosave off", state)
	}
}
