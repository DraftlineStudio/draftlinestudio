package main

import (
	"strings"
	"testing"
	"unicode/utf8"

	"draftline/internal/book"
	"draftline/internal/types"
)

// Test prose is invented for this test; it is not taken from any manuscript.
func plannerTestBook() types.BookData {
	result := types.BookData{
		Version:  "2.0",
		Metadata: types.Metadata{Created: "2026-09-15T00:00:00Z"},
		Body: []types.ChapterItem{
			{ID: "ch-one", Title: "One", Type: "chapter", Content: "<p>Rhea searched the tower for the missing lamp.</p><p>She found the lamp in the cellar.</p><hr><p>Tomas repaired the radio before dawn.</p>"},
			{ID: "ch-two", Title: "Two", Type: "chapter", Content: "<p>Rhea started looking for the courier.</p>"},
		},
		StoryBible: types.StoryBible{Characters: []types.Character{
			{ID: "char-rhea", Name: "Rhea", DetectionStatus: "accepted", IsAutoDetected: true},
			{ID: "char-tomas", Name: "Tomas"},
			{ID: "char-ghost", Name: "Ghost", DetectionStatus: "rejected"},
		}},
	}
	book.EnsureBookChapterIDs(&result)
	return result
}

func TestPlannerPositionResolvesChapterAndScene(t *testing.T) {
	book := plannerTestBook()
	cases := []struct {
		block, scene string
		wantChapter  string
		wantScene    int
	}{
		{"ch-one/block/3", "ch-one/scene/1", "ch-one", 2},
		{"ch-two/block/0", "ch-two/scene/0", "ch-two", 1},
		// Chapters without stable IDs are keyed by body index by the engine.
		{"body/001/block/0", "body/001/scene/0", "ch-two", 1},
	}
	for _, c := range cases {
		chapter, scene, err := plannerPosition(book, c.block, c.scene)
		if err != nil {
			t.Fatal(err)
		}
		if chapter != c.wantChapter || scene != c.wantScene {
			t.Fatalf("plannerPosition(%q, %q) = %q, %d; want %q, %d", c.block, c.scene, chapter, scene, c.wantChapter, c.wantScene)
		}
	}
	for _, c := range []struct{ block, scene string }{
		{"unknown/block/0", "unknown/scene/4"},
		{"ch-one/block/0", "ch-two/scene/0"},
		{"ch-one", "ch-one/scene/0"},
		{"ch-one/block/0", "ch-one/scene/nope"},
	} {
		if _, _, err := plannerPosition(book, c.block, c.scene); err == nil {
			t.Fatalf("accepted invalid position %q / %q", c.block, c.scene)
		}
	}
}

func TestPlannerSourceIdentitySurvivesRenameAndSaveAs(t *testing.T) {
	b := plannerTestBook()
	first, err := plannerSourceID(b)
	if err != nil {
		t.Fatal(err)
	}
	b.Metadata.Title = "Renamed"
	b.FilePath = "D:/elsewhere/copy.draftline"
	second, err := plannerSourceID(b)
	if err != nil || first != second {
		t.Fatalf("source identity changed across rename/save-as: %q -> %q (%v)", first, second, err)
	}
	b.Planner = &types.PlannerData{SourceID: "planner-source"}
	if got, err := plannerSourceID(b); err != nil || got != "planner-source" {
		t.Fatalf("persisted source identity ignored: %q (%v)", got, err)
	}
}

func TestPlannerTitleShortensLongSentences(t *testing.T) {
	long := strings.Repeat("word ", 30)
	title := plannerTitle(long)
	if utf8.RuneCountInString(title) > 72 || !strings.HasSuffix(title, "…") || !utf8.ValidString(title) {
		t.Fatalf("title %q", title)
	}
	unicodeTitle := plannerTitle(strings.Repeat("界", 80))
	if utf8.RuneCountInString(unicodeTitle) != 70 || !utf8.ValidString(unicodeTitle) {
		t.Fatalf("unicode title was not shortened safely: %q", unicodeTitle)
	}
	if got := plannerTitle("Short."); got != "Short." {
		t.Fatalf("short title changed: %q", got)
	}
}

// The binding must run end to end over a small book without error and hand
// back positions the Planner can place; how many cards the engine proposes
// is the engine's business, not this test's.
func TestPlannerDetectCardsMapsProposalsOntoChapters(t *testing.T) {
	app := &App{}
	result := app.PlannerDetectCards(plannerTestBook())
	if result.Error != "" {
		t.Fatalf("detect: %s", result.Error)
	}
	if result.Cards == nil {
		t.Fatal("cards must be an empty slice, not null, for the frontend")
	}
	if result.SourceID == "" || result.Revision == "" || result.Limitations == nil {
		t.Fatalf("detection lost identity, revision, or limitations: %+v", result)
	}
	for _, c := range result.Cards {
		if c.ChapterID != "ch-one" && c.ChapterID != "ch-two" {
			t.Fatalf("card %q positioned at unknown chapter %q", c.Title, c.ChapterID)
		}
		if c.Scene < 1 || len(c.Evidence) == 0 || c.Kind != "development" || c.Status != "detected_candidate" || c.Support != "candidate" || c.SelectionRule == "" {
			t.Fatalf("card %+v", c)
		}
		for _, evidence := range c.Evidence {
			if evidence.SourceID != result.SourceID || evidence.Revision != result.Revision || evidence.ChapterID != c.ChapterID || evidence.Scene != c.Scene || evidence.Quote == "" {
				t.Fatalf("card evidence is not revision-bound: %+v", evidence)
			}
		}
	}
	if empty := app.PlannerDetectCards(types.BookData{}); empty.Error != "" || len(empty.Cards) != 0 {
		t.Fatalf("empty book: %+v", empty)
	}
	withoutIdentity := types.BookData{Body: []types.ChapterItem{{Content: "<p>Invented text.</p>"}}}
	failed := app.PlannerDetectCards(withoutIdentity)
	if failed.Error == "" || failed.Cards == nil || failed.Limitations == nil {
		t.Fatalf("failed detection must still return frontend-safe collections: %+v", failed)
	}
}
