package main

import (
	"bytes"
	"image"
	"net/http"
	"testing"

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

	res := fetch(t, app.recentCovers(), "/recent-cover/0")
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

	if res := fetch(t, app.recentCovers(), "/recent-cover/0"); res.Code != http.StatusNotFound {
		t.Errorf("a book with no cover answered %d, want 404", res.Code)
	}
}

// Only a position is accepted. A handler that took a path would read any file
// on the machine on request, which is not a trade worth making for a thumbnail.
func TestTheRecentCoverHandlerTakesOnlyAPosition(t *testing.T) {
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
		"/recent-cover/" + path,             // the archive by name
		"/recent-cover/../../etc/passwd",    // a traversal
		"/recent-cover/-1",                  // before the list
		"/recent-cover/99",                  // past the list
		"/recent-cover/",                    // nothing at all
		"/recent-cover/0x0",                 // not a number
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

	res := fetch(t, app.recentCovers(), "/recent-cover/0")
	if res.Code != http.StatusOK {
		t.Fatalf("the cover came back %d", res.Code)
	}
	if got := res.Body.Len(); got != second.ThumbBytes {
		t.Errorf("served %d bytes; the second edition's thumbnail is %d and the first is %d",
			got, second.ThumbBytes, first.ThumbBytes)
	}
}
