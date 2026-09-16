package main

// What has to be true of a cover once it is attached.
//
// The invariants here are the expensive ones to get wrong: artwork that is
// destroyed by the next autosave, artwork that rides the Wails bridge as
// base64 on every keystroke, and artwork that is deflated over and over for no
// gain. None of those shows up as a crash; all three show up as a project that
// is slow, or as art that is quietly gone.

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"draftline/internal/book"
	"draftline/internal/types"
)

// ── Invented artwork ───────────────────────────────────────────────────────

// paintTestArtwork draws a cover from arithmetic. No file on disk anywhere is
// read; nothing here is anybody's picture.
func paintTestArtwork(w, h int, tint float64) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		v := float64(y) / float64(h-1)
		for x := 0; x < w; x++ {
			u := float64(x) / float64(w-1)
			base := 0.08 + 0.7*v
			r := base * (0.7 + 0.4*u) * tint
			g := base * (0.9 + 0.1*math.Sin(u*5))
			b := base * (1.2 - 0.3*u)
			if y > h/3 && y < h/3+h/20 && x > w/8 && x < w*7/8 {
				r, g, b = 0.93, 0.9, 0.82
			}
			img.SetNRGBA(x, y, color.NRGBA{R: clamp255(r), G: clamp255(g), B: clamp255(b), A: 255})
		}
	}
	return img
}

func clamp255(v float64) uint8 {
	n := int(v*255 + 0.5)
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return uint8(n)
}

func writeTestArtwork(t *testing.T, path string, w, h int, tint float64) string {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, paintTestArtwork(w, h, tint)); err != nil {
		t.Fatalf("encoding invented artwork: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("writing invented artwork: %v", err)
	}
	return path
}

// ── Reading the archive ────────────────────────────────────────────────────

type archiveEntry struct {
	data   []byte
	method uint16
}

func editionEntries(t *testing.T, path string) map[string]archiveEntry {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("cannot open %s: %v", filepath.Base(path), err)
	}
	defer func() { _ = r.Close() }()
	out := map[string]archiveEntry{}
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "editions/") || strings.HasSuffix(f.Name, ".json") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("cannot open %q: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("cannot read %q: %v", f.Name, err)
		}
		out[f.Name] = archiveEntry{data: data, method: f.Method}
	}
	return out
}

// bookWithEdition is the book the screen would hand back after attaching a
// cover: the record on the edition, and no bytes anywhere on it.
func bookWithEdition(cover *types.EditionCover) types.BookData {
	b := bookWithChapter("<p>The pilot boat came about in the channel.</p>")
	b.Editions = &types.EditionIndex{
		Version: 1,
		Editions: []types.Edition{{
			ID:      "ed-1",
			Label:   "First edition",
			Year:    "2026",
			Status:  "Draft",
			Cover:   cover,
			Formats: []types.EditionFormat{{ID: "fmt-1", Kind: types.EditionKindEbook, Format: "eBook"}},
		}},
	}
	if cover != nil {
		b.Editions.Editions[0].CoverID = cover.ID
	}
	return b
}

// openedProject builds a project on disk, opens it, and returns the app and
// the path.
func openedProject(t *testing.T) (*App, string) {
	t.Helper()
	config := t.TempDir()
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CONFIG_HOME", config)

	dir := t.TempDir()
	path := filepath.Join(dir, "project.draftline")
	if result := book.Write("", path, bookWithEdition(nil), "test-version"); !result.Success {
		t.Fatal(result.Error)
	}
	app := &App{}
	if _, err := app.openBook(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.releaseBookLock)
	return app, path
}

func attachTestCover(t *testing.T, app *App, editionID string, w, h int, tint float64, large bool) *types.EditionCover {
	t.Helper()
	art := writeTestArtwork(t, filepath.Join(t.TempDir(), "artwork.png"), w, h, tint)
	result := app.AttachCover(editionID, art, large)
	if !result.Success || result.Cover == nil {
		t.Fatalf("attaching a cover failed: %s", result.Error)
	}
	return result.Cover
}

// ── The invariants ─────────────────────────────────────────────────────────

// The archive is rebuilt from scratch on every save and an autosave fires five
// seconds after a keystroke. Cover art that is not explicitly carried across
// is destroyed by the next one, so ten saves in a row is the shape of the bug
// this is guarding against: the art is there after the first save and gone
// after the second.
func TestCoverArtSurvivesTenAutosavesUntouched(t *testing.T) {
	app, path := openedProject(t)
	cover := attachTestCover(t, app, "ed-1", 2000, 3200, 1.0, false)
	b := bookWithEdition(cover)

	if result := app.writeBook(b, path); !result.Success {
		t.Fatal(result.Error)
	}
	first := editionEntries(t, path)
	if len(first) != 2 {
		t.Fatalf("the save wrote %d cover members, wanted the cover and its thumbnail: %v", len(first), namesOf(first))
	}
	for name, entry := range first {
		if entry.method != zip.Store {
			t.Errorf("%s was deflated; an already-compressed image gains nothing and costs CPU on every autosave", name)
		}
	}

	for i := 0; i < 10; i++ {
		if result := app.writeBook(b, path); !result.Success {
			t.Fatalf("autosave %d failed: %s", i+1, result.Error)
		}
	}

	after := editionEntries(t, path)
	if len(after) != len(first) {
		t.Fatalf("after ten autosaves the archive holds %v, not %v", namesOf(after), namesOf(first))
	}
	for name, want := range first {
		got, ok := after[name]
		if !ok {
			t.Errorf("%s did not survive ten autosaves", name)
			continue
		}
		if !bytes.Equal(got.data, want.data) {
			t.Errorf("%s was re-encoded by an autosave: %d bytes became %d", name, len(want.data), len(got.data))
		}
		if got.method != zip.Store {
			t.Errorf("%s came back deflated after an autosave", name)
		}
	}
}

// The one thing that must never happen: image data on the struct that crosses
// the Wails bridge as JSON every time the author pauses.
func TestTheSavedBookCarriesNoImageData(t *testing.T) {
	app, path := openedProject(t)
	cover := attachTestCover(t, app, "ed-1", 2000, 3200, 1.0, false)
	b := bookWithEdition(cover)
	if result := app.writeBook(b, path); !result.Success {
		t.Fatal(result.Error)
	}

	payload, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if cover.Bytes < 10_000 {
		t.Fatalf("the invented cover came out at %d bytes, too small for this test to mean anything", cover.Bytes)
	}
	if len(payload) > 20_000 {
		t.Errorf("the book crossing the bridge is %d bytes with a %d-byte cover attached; image data has got onto BookData",
			len(payload), cover.Bytes)
	}
	// The JPEG start-of-image marker, base64-encoded, is how a JPEG smuggled
	// into JSON would look.
	for _, marker := range []string{"/9j/", "iVBORw0KGgo", "\\u00ff\\u00d8"} {
		if bytes.Contains(payload, []byte(marker)) {
			t.Errorf("the save payload contains encoded image data (%q)", marker)
		}
	}

	// And what IS on it is the record: enough to draw the panel and find the
	// original again.
	var round types.BookData
	if err := json.Unmarshal(payload, &round); err != nil {
		t.Fatal(err)
	}
	got := round.Editions.Editions[0].Cover
	if got == nil || got.File != cover.File || got.Width != cover.Width || got.SourceChecksum != cover.SourceChecksum {
		t.Errorf("the cover record did not survive the bridge: %+v", got)
	}
}

// A cover replaced by artwork that encodes differently must not leave the old
// file behind. The passthrough would otherwise carry cover.jpg forward for
// ever beside a new cover.png.
func TestReplacingACoverLeavesNothingOfTheOldOne(t *testing.T) {
	app, path := openedProject(t)

	first := attachTestCover(t, app, "ed-1", 2000, 3200, 1.0, true)
	if result := app.writeBook(bookWithEdition(first), path); !result.Success {
		t.Fatal(result.Error)
	}
	if got := len(editionEntries(t, path)); got != 3 {
		t.Fatalf("the first save wrote %d members, wanted cover, thumbnail and the larger copy", got)
	}

	// The second cover is attached without the larger copy, so the old
	// cover_large member has nothing replacing it by name.
	second := attachTestCover(t, app, "ed-1", 1800, 2880, 0.4, false)
	if result := app.writeBook(bookWithEdition(second), path); !result.Success {
		t.Fatal(result.Error)
	}

	after := editionEntries(t, path)
	if len(after) != 2 {
		t.Fatalf("after replacing the cover the archive holds %v; the old artwork was not dropped", namesOf(after))
	}
	for name := range after {
		if strings.Contains(name, "cover_large") {
			t.Errorf("%s belongs to the replaced cover and is still in the project", name)
		}
	}
	if got := after[book.CoverMember("ed-1", second.File)]; len(got.data) != second.Bytes {
		t.Errorf("the archived cover is %d bytes, the attached one %d", len(got.data), second.Bytes)
	}
}

// Removing a cover releases it from the project rather than leaving orphaned
// artwork in every future save.
func TestRemovingACoverTakesItOutOfTheProject(t *testing.T) {
	app, path := openedProject(t)
	cover := attachTestCover(t, app, "ed-1", 1600, 2560, 1.0, false)
	if result := app.writeBook(bookWithEdition(cover), path); !result.Success {
		t.Fatal(result.Error)
	}
	if len(editionEntries(t, path)) == 0 {
		t.Fatal("the cover was not saved in the first place")
	}

	app.RemoveCover("ed-1")
	if result := app.writeBook(bookWithEdition(nil), path); !result.Success {
		t.Fatal(result.Error)
	}
	if got := editionEntries(t, path); len(got) != 0 {
		t.Errorf("removing the cover left %v in the project", namesOf(got))
	}
}

// ── Showing it ─────────────────────────────────────────────────────────────

// The panel and the rail fetch the cover from this process over an ordinary
// same-origin URL. Nothing is base64-encoded and nothing crosses the bridge.
func TestTheCoverIsServedFromASameOriginURL(t *testing.T) {
	app, path := openedProject(t)
	attached := attachTestCover(t, app, "ed-1", 1800, 2880, 1.0, false)
	if result := app.writeBook(bookWithEdition(attached), path); !result.Success {
		t.Fatal(result.Error)
	}

	server := app.editionAssets()

	res := fetch(t, server, "/editions/ed-1/"+attached.ThumbFile)
	if res.Code != http.StatusOK {
		t.Fatalf("the thumbnail came back %d", res.Code)
	}
	if got := res.Body.Len(); got != attached.ThumbBytes {
		t.Errorf("the thumbnail served %d bytes, the record says %d", got, attached.ThumbBytes)
	}
	if got := res.Header().Get("Content-Type"); got != "image/jpeg" && got != "image/png" {
		t.Errorf("the thumbnail was served as %q", got)
	}
	if _, _, err := image.Decode(bytes.NewReader(res.Body.Bytes())); err != nil {
		t.Errorf("what was served is not an image: %v", err)
	}

	if full := fetch(t, server, "/editions/ed-1/"+attached.File); full.Code != http.StatusOK {
		t.Errorf("the full cover came back %d", full.Code)
	}
}

// The cache is emptied when a project is opened, so a reopened book has to
// find its covers in the archive. That read-through is what keeps every
// edition's artwork out of memory until something looks at it.
func TestAReopenedProjectStillShowsItsCovers(t *testing.T) {
	app, path := openedProject(t)
	cover := attachTestCover(t, app, "ed-1", 1800, 2880, 1.0, false)
	if result := app.writeBook(bookWithEdition(cover), path); !result.Success {
		t.Fatal(result.Error)
	}

	app.covers.reset()
	if _, ok := app.covers.lookup("ed-1", cover.ThumbFile); ok {
		t.Fatal("the cache still holds the cover after being reset")
	}

	res := fetch(t, app.editionAssets(), "/editions/ed-1/"+cover.ThumbFile)
	if res.Code != http.StatusOK {
		t.Fatalf("a reopened project served its thumbnail as %d", res.Code)
	}
	if res.Body.Len() != cover.ThumbBytes {
		t.Errorf("the archived thumbnail is %d bytes, the record says %d", res.Body.Len(), cover.ThumbBytes)
	}
	if _, ok := app.covers.lookup("ed-1", cover.ThumbFile); !ok {
		t.Error("the bytes read out of the archive were not kept for the next request")
	}
}

// The asset route is reachable from a webview, so it answers only for plain
// names inside one edition's folder.
func TestTheAssetRouteRefusesAnythingButAnEditionsOwnFiles(t *testing.T) {
	app, path := openedProject(t)
	cover := attachTestCover(t, app, "ed-1", 1600, 2560, 1.0, false)
	if result := app.writeBook(bookWithEdition(cover), path); !result.Success {
		t.Fatal(result.Error)
	}
	server := app.editionAssets()

	for _, probe := range []string{
		"/editions/",
		"/editions/ed-1",
		"/editions/ed-1/",
		"/editions/../manifest.json",
		"/editions/ed-1/../../manifest.json",
		"/editions/ed-1/index.json",
		"/editions/ed-2/cover.jpg",
		"/editions/ed-1/nothing.jpg",
	} {
		if res := fetch(t, server, probe); res.Code == http.StatusOK {
			t.Errorf("%s was served, and should not have been", probe)
		}
	}
}

// ── Refusals ───────────────────────────────────────────────────────────────

func TestAttachingRefusesWhatIsNotArtwork(t *testing.T) {
	app, _ := openedProject(t)
	dir := t.TempDir()

	notes := filepath.Join(dir, "outline.txt")
	if err := os.WriteFile(notes, []byte("three chapters and a funeral"), 0o644); err != nil {
		t.Fatal(err)
	}

	if result := app.AttachCover("ed-1", notes, false); result.Success {
		t.Error("a text file was accepted as cover artwork")
	} else if !strings.Contains(result.Error, "JPEG, PNG, TIFF") {
		t.Errorf("the refusal does not say what Draftline reads: %s", result.Error)
	}

	small := writeTestArtwork(t, filepath.Join(dir, "small.png"), 300, 480, 1.0)
	if result := app.AttachCover("ed-1", small, false); result.Success {
		t.Error("artwork below the retailer floor was accepted")
	}

	if result := app.AttachCover("", small, false); result.Success {
		t.Error("a cover was attached to no edition at all")
	}
}

// An edition identifier becomes a folder name in the archive and a segment of
// a URL. The project file is a ZIP anybody can edit, so a name that could
// climb out of that folder is refused when the book is saved.
func TestAnEditionThatCouldNotBeAFolderIsRefused(t *testing.T) {
	_, path := openedProject(t)
	b := bookWithEdition(nil)
	b.Editions.Editions[0].ID = "../../etc"

	result := book.Write("", path, b, "test-version")
	if result.Success {
		t.Fatal("an edition whose identifier is a path was saved")
	}
	if !strings.Contains(result.Error, "folder name") {
		t.Errorf("the refusal does not explain itself: %s", result.Error)
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func fetch(t *testing.T, handler http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, target, nil))
	return res
}

func namesOf(entries map[string]archiveEntry) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	return names
}

// ── Whose cover it is ──────────────────────────────────────────────────────

// An unsaved cover belongs to the project it was attached to, and to no other.
//
// The cache is the one thing the backend holds that is not on BookData, so it
// does not travel with the book across the bridge and no screen in the app
// could ever show that it is still there. Every path that stops working on the
// open book has to empty it: starting a new book, opening another, importing
// one, and going back to the launch screen. Miss one and the next project
// saved gets another book's artwork inside it, invisible, and kept there for
// ever by the editions passthrough.
func TestLeavingAProjectLetsGoOfItsUnsavedCover(t *testing.T) {
	for _, leave := range []struct {
		name string
		do   func(app *App)
	}{
		{"closing the book", func(app *App) { app.CloseBookFile() }},
		{"starting a new book", func(app *App) { app.NewBook() }},
		// ImportEPUB and ImportDOCX both end by calling leaveOpenProject for
		// exactly this reason; calling it directly is what they do, without
		// needing an EPUB on disk to do it with.
		{"importing another book", func(app *App) { app.leaveOpenProject() }},
	} {
		t.Run(leave.name, func(t *testing.T) {
			app, _ := openedProject(t)
			attachTestCover(t, app, "ed-1", 1600, 2560, 1.0, false)
			if _, ok := app.covers.lookup("ed-1", "cover.jpg"); !ok {
				t.Fatal("the cover was not attached in the first place")
			}

			leave.do(app)
			if _, ok := app.covers.lookup("ed-1", "cover.jpg"); ok {
				t.Error("the cover of the project that was left is still held")
			}

			// The next book saved must come out with nothing of it in.
			next := filepath.Join(t.TempDir(), "next.draftline")
			if result := app.writeBook(bookWithEdition(nil), next); !result.Success {
				t.Fatal(result.Error)
			}
			if got := editionEntries(t, next); len(got) != 0 {
				t.Errorf("the next book saved carries %v from the project before it", namesOf(got))
			}
		})
	}
}
