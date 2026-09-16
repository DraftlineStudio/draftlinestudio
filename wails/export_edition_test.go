package main

// The bridge half of exporting from an edition.
//
// The exporter itself is tested in internal/export. What has to be true here
// is the thing that cannot be tested there: the artwork reaches the exporter
// without ever being on types.BookData, which is the struct that crosses the
// Wails bridge as JSON on every autosave. A cover of a megabyte on that struct
// would be a megabyte base64-encoded through the webview every five seconds.

import (
	"encoding/json"
	"strings"
	"testing"

	"draftline/internal/types"
)

func TestTheExporterIsHandedTheEditionsCoverBytes(t *testing.T) {
	app, _ := openedProject(t)
	cover := attachTestCover(t, app, "ed-1", 2000, 3200, 1.0, false)
	b := bookWithEdition(cover)

	art := app.exportCover(b, types.ExportOptions{EditionID: "ed-1", FormatID: "fmt-1"})
	if art == nil {
		t.Fatal("the export was handed no cover for a format whose edition has one")
	}
	if len(art.Data) < 1000 {
		t.Errorf("the cover handed over is %d bytes, which is not an image", len(art.Data))
	}
	if art.FileName != cover.File {
		t.Errorf("file name = %q, want %q", art.FileName, cover.File)
	}
	if art.MediaType != "image/jpeg" && art.MediaType != "image/png" {
		t.Errorf("media type = %q", art.MediaType)
	}
	if art.Width != cover.Width || art.Height != cover.Height {
		t.Errorf("size = %dx%d, want %dx%d", art.Width, art.Height, cover.Width, cover.Height)
	}
}

// The same book, serialised the way the bridge serialises it, must carry none
// of those bytes.
func TestTheCoverHandedToAnExportIsNotOnTheBookThatCrossedTheBridge(t *testing.T) {
	app, _ := openedProject(t)
	cover := attachTestCover(t, app, "ed-1", 2000, 3200, 1.0, false)
	b := bookWithEdition(cover)

	art := app.exportCover(b, types.ExportOptions{EditionID: "ed-1", FormatID: "fmt-1"})
	if art == nil {
		t.Fatal("no cover was handed over")
	}

	encoded, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > 20<<10 {
		t.Errorf("the book crossing the bridge is %d bytes with a %d byte cover attached", len(encoded), len(art.Data))
	}
	for _, marker := range []string{"/9j/", "iVBORw0", "base64"} {
		if strings.Contains(string(encoded), marker) {
			t.Errorf("image data (%q) is on the struct that crosses the bridge", marker)
		}
	}
}

// An export that names no format, or names one whose edition has no artwork,
// gets nothing — and produces the file Draftline always produced.
func TestAnExportWithNoEditionIsHandedNoCover(t *testing.T) {
	app, _ := openedProject(t)
	cover := attachTestCover(t, app, "ed-1", 1200, 1920, 1.0, false)

	for _, tc := range []struct {
		name    string
		book    types.BookData
		options types.ExportOptions
	}{
		{"no format named", bookWithEdition(cover), types.ExportOptions{}},
		{"a format the book does not hold", bookWithEdition(cover), types.ExportOptions{FormatID: "fmt-gone"}},
		{"an edition with no cover", bookWithEdition(nil), types.ExportOptions{FormatID: "fmt-1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if art := app.exportCover(tc.book, tc.options); art != nil {
				t.Errorf("a cover was handed over anyway: %d bytes of %s", len(art.Data), art.FileName)
			}
		})
	}
}

// A cover that was saved and then dropped from the cache — which is what a
// reopened project looks like — is still found, because the asset reader falls
// through to the project file.
func TestAReopenedProjectStillHandsTheCoverToAnExport(t *testing.T) {
	app, path := openedProject(t)
	cover := attachTestCover(t, app, "ed-1", 1600, 2560, 1.0, false)
	b := bookWithEdition(cover)
	if result := app.writeBook(b, path); !result.Success {
		t.Fatal(result.Error)
	}

	reopened := &App{}
	if _, err := reopened.openBook(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.releaseBookLock)

	art := reopened.exportCover(b, types.ExportOptions{FormatID: "fmt-1"})
	if art == nil || len(art.Data) == 0 {
		t.Fatal("a reopened project handed the export no cover")
	}
}
