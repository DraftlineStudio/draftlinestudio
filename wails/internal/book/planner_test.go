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

// planner.json rides inside the archive and must survive a save/open round
// trip exactly, including the empty "Later" chapter position and links.
func TestPlannerRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "planner.draftline")
	book := testBook()
	book.Body[0].ID = "ch-planner"
	book.Planner = &types.PlannerData{
		Version:  1,
		SourceID: "book-source",
		Lanes: []types.PlannerLane{
			{ID: "main", Name: "Main plot", Kind: "main", Color: "#5aafe0"},
			{ID: "lane-1", Name: "Rhea", Kind: "character", Color: "#F472B6", CharacterID: "char-rhea"},
		},
		Cards: []types.PlannerCard{
			{ID: "c1", Title: "The tower light goes out", Synopsis: "Rhea sees the beacon fail.", Lines: []string{"main", "lane-1"}, Who: []string{"char-rhea"},
				Changes: "The town loses its warning.", ChapterID: "ch-planner", Link: &types.PlannerLink{ChapterID: "ch-planner", Scene: 1}, Status: "drafted", Origin: "manual",
				Evidence: []types.PlannerEvidence{{SourceID: "book-source", Revision: "revision-1", ChapterID: "ch-planner", Scene: 1, BlockID: "ch-planner/block/0", Start: 0, End: 12, Quote: "Invented text"}}},
			{ID: "c2", Title: "Tomas returns", Lines: []string{"main"}, Who: []string{}, ChapterID: "", Status: "planned", Origin: "outline"},
		},
		Notes:        []types.PlannerNote{{ID: "n1", Title: "Loose threads", Body: "- who lit the beacon\n", Updated: "2026-09-15"}},
		Synopsis:     map[string]string{"ch-planner": "Edited paragraph."},
		BeatTemplate: "three-act",
		HiddenLanes:  []string{"lane-1"},
		Dismissed:    []string{"det-9"},
		PlotWalker:   true,
	}
	if res := Write(path, book, "test-version"); !res.Success {
		t.Fatalf("write: %s", res.Error)
	}
	got, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Planner == nil {
		t.Fatal("planner.json was not read back")
	}
	p := got.Planner
	if len(p.Lanes) != 2 || p.Lanes[1].CharacterID != "char-rhea" {
		t.Fatalf("lanes: %+v", p.Lanes)
	}
	if len(p.Cards) != 2 || p.Cards[0].Link == nil || p.Cards[0].Link.Scene != 1 || len(p.Cards[0].Evidence) != 1 || p.Cards[0].Evidence[0].Revision != "revision-1" || p.Cards[1].ChapterID != "" || p.Cards[1].Link != nil {
		t.Fatalf("cards: %+v", p.Cards)
	}
	if p.SourceID != "book-source" || p.Synopsis["ch-planner"] != "Edited paragraph." || p.BeatTemplate != "three-act" || !p.PlotWalker || len(p.HiddenLanes) != 1 || len(p.Dismissed) != 1 {
		t.Fatalf("settings: %+v", p)
	}
	if len(p.Notes) != 1 || p.Notes[0].Body != "- who lit the beacon\n" {
		t.Fatalf("notes: %+v", p.Notes)
	}
}

func TestPlannerCollectionsAreNormalizedOnOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "planner-empty.draftline")
	b := testBook()
	b.Planner = &types.PlannerData{Version: 1}
	if result := Write(path, b, "test-version"); !result.Success {
		t.Fatal(result.Error)
	}
	got, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Planner == nil || got.Planner.Lanes == nil || got.Planner.Cards == nil || got.Planner.Notes == nil {
		t.Fatalf("required Planner collections were not normalized: %+v", got.Planner)
	}
}

func TestPlannerSaveDoesNotMutateCaller(t *testing.T) {
	path := filepath.Join(t.TempDir(), "planner-copy.draftline")
	b := testBook()
	b.Planner = &types.PlannerData{Version: 1}
	if result := Write(path, b, "test-version"); !result.Success {
		t.Fatal(result.Error)
	}
	if b.Planner.Lanes != nil || b.Planner.Cards != nil || b.Planner.Notes != nil {
		t.Fatalf("save mutated the caller's Planner collections: %+v", b.Planner)
	}
}

func TestMalformedPlannerBlocksOpenRatherThanBeingDropped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "planner-malformed.draftline")
	b := testBook()
	b.Planner = &types.PlannerData{Version: 1}
	if result := Write(path, b, "test-version"); !result.Success {
		t.Fatal(result.Error)
	}
	replacePlannerEntry(t, path, []byte(`{"version":`))
	if _, err := Open(path); err == nil || !strings.Contains(err.Error(), "planner data could not be parsed") {
		t.Fatalf("malformed author data was silently accepted: %v", err)
	}
}

func TestUnsupportedPlannerVersionCannotBeSaved(t *testing.T) {
	b := testBook()
	b.Planner = &types.PlannerData{Version: 2}
	result := Write(filepath.Join(t.TempDir(), "planner-newer.draftline"), b, "test-version")
	if result.Success || !strings.Contains(result.Error, "version 2") {
		t.Fatalf("unsupported Planner save was accepted: %+v", result)
	}
}

func replacePlannerEntry(t *testing.T, path string, replacement []byte) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range reader.File {
		header := file.FileHeader
		destination, err := writer.CreateHeader(&header)
		if err != nil {
			t.Fatal(err)
		}
		if file.Name == "planner.json" {
			if _, err := destination.Write(replacement); err != nil {
				t.Fatal(err)
			}
			continue
		}
		source, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		_, copyErr := io.Copy(destination, source)
		closeErr := source.Close()
		if copyErr != nil || closeErr != nil {
			t.Fatalf("copy archive member: %v / %v", copyErr, closeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, output.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A book that never opened the Planner writes no planner.json.
func TestPlannerAbsentWhenUnused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain.draftline")
	if res := Write(path, testBook(), "test-version"); !res.Success {
		t.Fatalf("write: %s", res.Error)
	}
	got, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Planner != nil {
		t.Fatal("planner.json written for a book that never used the Planner")
	}
}
