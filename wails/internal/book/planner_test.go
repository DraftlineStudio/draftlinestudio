package book

import (
	"archive/zip"
	"bytes"
	"encoding/json"
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
		Version: 1,
		Lanes: []types.PlannerLane{
			{ID: "main", Name: "Main plot", Kind: "main", Color: "#5aafe0"},
			{ID: "lane-1", Name: "Rhea", Kind: "character", Color: "#F472B6", CharacterID: "char-rhea"},
		},
		Cards: []types.PlannerCard{
			{ID: "c1", Title: "The tower light goes out", Synopsis: "Rhea sees the beacon fail.", Lines: []string{"main", "lane-1"}, Who: []string{"char-rhea"},
				Changes: "The town loses its warning.", ChapterID: "ch-planner", Link: &types.PlannerLink{ChapterID: "ch-planner", Scene: 1}, Status: "drafted"},
			{ID: "c2", Title: "Tomas returns", Lines: []string{"main"}, Who: []string{}, ChapterID: "", Status: "planned"},
		},
		Notes:        []types.PlannerNote{{ID: "n1", Title: "Loose threads", Body: "- who lit the beacon\n", Updated: "2026-09-15"}},
		Synopsis:     map[string]string{"ch-planner": "Edited paragraph."},
		BeatTemplate: "three-act",
		HiddenLanes:  []string{"lane-1"},
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
	if len(p.Cards) != 2 || p.Cards[0].Link == nil || p.Cards[0].Link.Scene != 1 || p.Cards[1].ChapterID != "" || p.Cards[1].Link != nil {
		t.Fatalf("cards: %+v", p.Cards)
	}
	if p.Synopsis["ch-planner"] != "Edited paragraph." || p.BeatTemplate != "three-act" || len(p.HiddenLanes) != 1 {
		t.Fatalf("settings: %+v", p)
	}
	if len(p.Notes) != 1 || p.Notes[0].Body != "- who lit the beacon\n" {
		t.Fatalf("notes: %+v", p.Notes)
	}
}

func TestPlannerCollectionsAreNormalizedOnOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "planner-empty.draftline")
	b := testBook()
	b.Body[0].ID = "ch-legacy"
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

// Names stored beside a card's people are read by position, so a list that
// does not line up with the people is dropped on open rather than naming
// the wrong person.
func TestPlannerMisalignedWhoNamesAreDroppedOnOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "planner-who.draftline")
	b := testBook()
	b.Body[0].ID = "ch-legacy"
	b.Planner = &types.PlannerData{Version: 1, Cards: []types.PlannerCard{
		{ID: "c1", Title: "Aligned", Lines: []string{"main"}, Who: []string{"char-rhea", "char-tomas"}, WhoNames: []string{"Rhea", "Tomas"}, Status: "planned"},
		{ID: "c2", Title: "Misaligned", Lines: []string{"main"}, Who: []string{"char-rhea", "char-tomas"}, WhoNames: []string{"Rhea"}, Status: "planned"},
		{ID: "c3", Title: "Unnamed", Lines: []string{"main"}, Who: []string{"char-rhea"}, Status: "planned"},
	}}
	if result := Write(path, b, "test-version"); !result.Success {
		t.Fatal(result.Error)
	}
	got, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	cards := got.Planner.Cards
	if len(cards) != 3 || len(cards[0].WhoNames) != 2 || cards[1].WhoNames != nil || cards[2].WhoNames != nil {
		t.Fatalf("who_names after open: %+v", cards)
	}
}

func TestPlannerSaveDoesNotMutateCaller(t *testing.T) {
	path := filepath.Join(t.TempDir(), "planner-copy.draftline")
	b := testBook()
	b.Body[0].ID = "ch-legacy"
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
	b.Body[0].ID = "ch-legacy"
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
	b.Body[0].ID = "ch-legacy"
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

// A planner.json written by a build that still had the experimental
// narrative engine carries keys this build has no field for: the Plot Walker
// switch, dismissals, and the writer's decisions about detected
// developments. Those keys are dropped on read, and the book still opens
// with every card, lane, and note the author made.
func TestArchiveWithLegacyNarrativeKeysStillOpens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.draftline")
	b := testBook()
	b.Body[0].ID = "ch-legacy"
	b.Planner = &types.PlannerData{
		Version: 1,
		Lanes:   []types.PlannerLane{{ID: "main", Name: "Main plot", Kind: "main", Color: "#5aafe0"}},
		Cards: []types.PlannerCard{{
			ID: "c1", Title: "The lantern goes out", Synopsis: "Rhea finds the lamp dark.",
			Lines: []string{"main"}, Who: []string{}, ChapterID: "ch-legacy",
			Link: &types.PlannerLink{ChapterID: "ch-legacy", Scene: 1}, Status: "drafted",
		}},
		Notes: []types.PlannerNote{},
	}
	if res := Write(path, b, "test-version"); !res.Success {
		t.Fatalf("Write failed: %s", res.Error)
	}

	// Put the retired keys back into the archive's planner.json by hand, the
	// way an older build left them.
	legacy := map[string]any{
		"version": 1,
		"lanes":   []any{map[string]any{"id": "main", "name": "Main plot", "kind": "main", "color": "#5aafe0", "source_strand_id": "str-1"}},
		"cards": []any{map[string]any{
			"id": "c1", "title": "The lantern goes out", "synopsis": "Rhea finds the lamp dark.",
			"lines": []any{"main"}, "who": []any{}, "chapter_id": "ch-legacy",
			"link": map[string]any{"chapter_id": "ch-legacy", "scene": 1}, "status": "drafted",
			"origin": "adopted", "dev_kind": "major_discovery",
			"evidence": []any{map[string]any{"source_id": "s", "revision": "r", "chapter_id": "ch-legacy", "scene": 1, "start": 0, "end": 4, "quote": "dark", "space": "chapter"}},
		}},
		"notes":       []any{},
		"plot_walker": true,
		"dismissed":   []any{"dev-9"},
		"decisions": []any{map[string]any{
			"id": "dec-1", "kind": "link", "card_id": "c1", "development_id": "dev-1", "signature": "sig-1",
		}},
	}
	encoded, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("encode legacy planner: %v", err)
	}
	rewriteArchiveEntry(t, path, "planner.json", encoded)

	got, err := Open(path)
	if err != nil {
		t.Fatalf("a book with legacy planner keys must still open: %v", err)
	}
	if got.Planner == nil {
		t.Fatal("the planner was dropped entirely")
	}
	if len(got.Planner.Cards) != 1 || got.Planner.Cards[0].ID != "c1" || got.Planner.Cards[0].Link == nil {
		t.Fatalf("the author's card did not survive: %+v", got.Planner.Cards)
	}
	if len(got.Planner.Lanes) != 1 || got.Planner.Lanes[0].ID != "main" {
		t.Fatalf("the author's lane did not survive: %+v", got.Planner.Lanes)
	}
	if got.Planner.Version != 1 {
		t.Fatalf("planner.json must stay at version 1, got %d", got.Planner.Version)
	}
}

// rewriteArchiveEntry replaces one member of a .draftline archive in place,
// copying every other entry through untouched.
func rewriteArchiveEntry(t *testing.T, path, name string, content []byte) {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, file := range reader.File {
		if file.Name == name {
			continue
		}
		if err := writer.Copy(file); err != nil {
			t.Fatalf("copy %s: %v", file.Name, err)
		}
	}
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if _, err := entry.Write(content); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	reader.Close()
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("replace archive: %v", err)
	}
}
