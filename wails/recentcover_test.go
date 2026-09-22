package main

import (
	"bytes"
	"image"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"draftline/internal/types"
)

// The start screen shows a book, not a coloured rectangle. The art is read out
// of the archive before the project is open, because a recent project is only
// a path until someone clicks it.
func TestTheStartScreenServesARecentProjectsCover(t *testing.T) {
	app, path := openedProject(t)
	cover := attachTestCover(t, app, "ed-1", 1800, 2880, 1.0, false)
	if result := app.writeBook(bookWithEdition(cover), path); !result.Success {
		t.Fatal(result.Error)
	}
	if err := app.AddRecentProject(types.RecentProject{Type: "book", Path: path, Name: "The Lantern", LastOpened: "2026-09-18T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}

	res := fetch(t, app.recentCovers(), "/recent-cover/"+recentCoverKey(path))
	if res.Code != http.StatusOK {
		t.Fatalf("the recent project's cover came back %d", res.Code)
	}
	if got := res.Body.Len(); got != cover.ThumbBytes {
		t.Errorf("served %d bytes, the record says %d", got, cover.ThumbBytes)
	}
	if _, _, err := image.Decode(bytes.NewReader(res.Body.Bytes())); err != nil {
		t.Errorf("what was served is not an image: %v", err)
	}
}

// A book with no artwork is the ordinary case, not a fault. The start screen
// keeps its colour when this 404s.
func TestARecentProjectWithNoArtworkIsNotFound(t *testing.T) {
	app, path := openedProject(t)
	if result := app.writeBook(bookWithEdition(nil), path); !result.Success {
		t.Fatal(result.Error)
	}
	if err := app.AddRecentProject(types.RecentProject{Type: "book", Path: path, Name: "The Lantern", LastOpened: "2026-09-18T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}

	if res := fetch(t, app.recentCovers(), "/recent-cover/"+recentCoverKey(path)); res.Code != http.StatusNotFound {
		t.Errorf("a book with no cover answered %d, want 404", res.Code)
	}
}

// Only a key this process issued is accepted. A handler that took a path
// would read any file on the machine on request, which is not a trade worth
// making for a thumbnail.
func TestTheRecentCoverHandlerTakesOnlyAKeyItIssued(t *testing.T) {
	app, path := openedProject(t)
	cover := attachTestCover(t, app, "ed-1", 1800, 2880, 1.0, false)
	if result := app.writeBook(bookWithEdition(cover), path); !result.Success {
		t.Fatal(result.Error)
	}
	if err := app.AddRecentProject(types.RecentProject{Type: "book", Path: path, Name: "The Lantern", LastOpened: "2026-09-18T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}

	server := app.recentCovers()
	for _, target := range []string{
		"/recent-cover/" + path,          // the archive by name
		"/recent-cover/../../etc/passwd", // a traversal
		"/recent-cover/0",                // the old position-based address
		"/recent-cover/",                 // nothing at all
		"/recent-cover/deadbeefdeadbeef", // a key belonging to nothing
	} {
		if res := fetch(t, server, target); res.Code != http.StatusNotFound {
			t.Errorf("%q answered %d, want 404", target, res.Code)
		}
	}
}

// The newest edition's art is the one shown: a reissue with new artwork should
// replace the first edition's on the shelf, not sit behind it.
func TestTheNewestEditionsArtworkIsTheOneShown(t *testing.T) {
	app, path := openedProject(t)
	first := attachTestCover(t, app, "ed-1", 1800, 2880, 1.0, false)
	second := attachTestCover(t, app, "ed-2", 1600, 2560, 0.4, false)

	book := bookWithEdition(first)
	book.Editions.Editions = append(book.Editions.Editions, types.Edition{
		ID:                "ed-2",
		Label:             "Second edition",
		Year:              "2027",
		Status:            "Draft",
		Cover:             second,
		CoverID:           second.ID,
		PreviousEditionID: "ed-1",
		Formats:           []types.EditionFormat{{ID: "fmt-2", Kind: types.EditionKindEbook, Format: "eBook"}},
	})
	if result := app.writeBook(book, path); !result.Success {
		t.Fatal(result.Error)
	}
	if err := app.AddRecentProject(types.RecentProject{Type: "book", Path: path, Name: "The Lantern", LastOpened: "2026-09-18T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}

	res := fetch(t, app.recentCovers(), "/recent-cover/"+recentCoverKey(path))
	if res.Code != http.StatusOK {
		t.Fatalf("the cover came back %d", res.Code)
	}
	if got := res.Body.Len(); got != second.ThumbBytes {
		t.Errorf("served %d bytes; the second edition's thumbnail is %d and the first is %d",
			got, second.ThumbBytes, first.ThumbBytes)
	}
}

// The bug that position-based addressing caused, pinned.
//
// Opening a book moves it to the front of the recents list. Addressed by
// position, every other book then asked for the art of whichever project had
// taken its place, so a shelf of books with no covers all wore the cover of
// the one just opened. A key belongs to a path, so reordering cannot do it.
func TestReorderingTheRecentsDoesNotMoveArtworkBetweenBooks(t *testing.T) {
	app, withArt := openedProject(t)
	cover := attachTestCover(t, app, "ed-1", 1800, 2880, 1.0, false)
	if result := app.writeBook(bookWithEdition(cover), withArt); !result.Success {
		t.Fatal(result.Error)
	}

	// A second book, with no artwork at all.
	bare := filepath.Join(t.TempDir(), "No Cover.draftline")
	if result := app.writeBook(bookWithEdition(nil), bare); !result.Success {
		t.Fatal(result.Error)
	}

	for _, p := range []string{withArt, bare} {
		if err := app.AddRecentProject(types.RecentProject{Type: "book", Path: p, Name: "x", LastOpened: "2026-09-18T00:00:00Z"}); err != nil {
			t.Fatal(err)
		}
	}
	// `bare` was added last, so it is now first: the exact reordering that
	// broke position-based addressing.
	recents := app.GetRecentProjects()
	if len(recents) < 2 || recents[0].Path != bare {
		t.Fatalf("expected the bare book at the front, got %+v", recents)
	}

	server := app.recentCovers()
	if res := fetch(t, server, "/recent-cover/"+recentCoverKey(bare)); res.Code != http.StatusNotFound {
		t.Errorf("the book with no artwork answered %d, want 404 -- it is wearing another book's cover", res.Code)
	}
	if res := fetch(t, server, "/recent-cover/"+recentCoverKey(withArt)); res.Code != http.StatusOK {
		t.Errorf("the book with artwork answered %d after the list reordered", res.Code)
	}
}

// Replacing a book's artwork changes the bytes behind an address that does not
// change, so the cache has to notice on its own. It stamps what it holds with
// the archive's size and modification time; a rewritten archive stops matching.
func TestNewArtworkReplacesTheCachedThumbnail(t *testing.T) {
	app, path := openedProject(t)
	first := attachTestCover(t, app, "ed-1", 1800, 2880, 1.0, false)
	if result := app.writeBook(bookWithEdition(first), path); !result.Success {
		t.Fatal(result.Error)
	}
	if err := app.AddRecentProject(types.RecentProject{Type: "book", Path: path, Name: "The Lantern", LastOpened: "2026-09-18T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}

	url := "/recent-cover/" + recentCoverKey(path)
	if res := fetch(t, app.recentCovers(), url); res.Code != http.StatusOK || res.Body.Len() != first.ThumbBytes {
		t.Fatalf("first read came back %d with %d bytes, want 200 with %d", res.Code, res.Body.Len(), first.ThumbBytes)
	}

	second := attachTestCover(t, app, "ed-1", 1600, 2560, 0.4, false)
	if second.ThumbBytes == first.ThumbBytes {
		t.Skip("the two test thumbnails are the same size; this test cannot tell them apart")
	}
	if result := app.writeBook(bookWithEdition(second), path); !result.Success {
		t.Fatal(result.Error)
	}
	// Archive timestamps have coarse resolution on some filesystems; make the
	// rewrite unambiguously newer than the read that cached the first one.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}

	res := fetch(t, app.recentCovers(), url)
	if res.Code != http.StatusOK {
		t.Fatalf("after replacing the artwork the cover came back %d", res.Code)
	}
	if got := res.Body.Len(); got != second.ThumbBytes {
		t.Errorf("served %d bytes, still the old thumbnail; the new one is %d", got, second.ThumbBytes)
	}
}
