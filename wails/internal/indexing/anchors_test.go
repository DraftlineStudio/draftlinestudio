package indexing

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

// All prose in this file is invented for these tests.

func anchorTestBook() types.BookData {
	return types.BookData{
		Body: []types.ChapterItem{
			{ID: "ch-tower", Title: "Tower", Type: "chapter", Content: "<p>Rhea climbed the tower stair.</p><hr><p>Tomas waited in the lantern room.</p><p>***</p><p>Eloise rowed the skiff across the harbour.</p>"},
			{ID: "ch-harbour", Title: "Harbour", Type: "chapter", Content: "<p>The skiff was tied at the harbour wall.</p>"},
		},
	}
}

func TestContentHashMatchesEvidenceHash(t *testing.T) {
	book := anchorTestBook()
	evidence := AnalyzeEvidence(&book, nil)
	if evidence.ContentHash == "" || evidence.ContentHash != ContentHash(&book) {
		t.Fatalf("ContentHash %q differs from the evidence index's %q", ContentHash(&book), evidence.ContentHash)
	}
	edited := anchorTestBook()
	edited.Body[1].Content = "<p>The skiff was gone from the harbour wall.</p>"
	if ContentHash(&edited) == ContentHash(&book) {
		t.Fatal("an edit did not change the content hash")
	}
	retitled := anchorTestBook()
	retitled.Body[0].Title = "Stair"
	if ContentHash(&retitled) == ContentHash(&book) {
		t.Fatal("a chapter title change did not change the content hash")
	}
}

func TestLocateAnchorRejectsStaleRevisionQuoteMismatchAndLegacySpace(t *testing.T) {
	book := anchorTestBook()
	locator := NewLocator(&book)
	text, ok := locator.ChapterText("ch-tower")
	if !ok {
		t.Fatal("chapter text missing")
	}
	quote := "Tomas waited in the lantern room."
	start := strings.Index(text, quote)
	good := types.PlannerEvidence{Revision: locator.Revision(), ChapterID: "ch-tower", Scene: 2, Start: start, End: start + len(quote), Quote: quote, Space: types.PlannerEvidenceChapterSpace}
	if ok, reason := locator.Locate(good); !ok || reason != "located" {
		t.Fatalf("valid anchor rejected: %v %s", ok, reason)
	}

	legacy := good
	legacy.Space = ""
	legacy.BlockID = "ch-tower/block/1"
	if ok, reason := locator.Locate(legacy); ok || reason != "legacy_block_anchor" {
		t.Fatalf("legacy block anchor: %v %s", ok, reason)
	}

	missing := good
	missing.ChapterID = "ch-gone"
	if ok, reason := locator.Locate(missing); ok || reason != "chapter_missing" {
		t.Fatalf("missing chapter: %v %s", ok, reason)
	}

	wrongQuote := good
	wrongQuote.Quote = "Tomas waited in the cellar."
	if ok, reason := locator.Locate(wrongQuote); ok || reason != "quote_mismatch" {
		t.Fatalf("quote mismatch under the same revision: %v %s", ok, reason)
	}

	wrongScene := good
	wrongScene.Scene = 1
	if ok, reason := locator.Locate(wrongScene); ok || reason != "scene_mismatch" {
		t.Fatalf("scene mismatch: %v %s", ok, reason)
	}

	// The passage is edited: the old anchor is stale, not merely mismatched.
	edited := anchorTestBook()
	edited.Body[0].Content = strings.Replace(edited.Body[0].Content, "lantern room", "lamp room", 1)
	if ok, reason := LocateAnchor(&edited, good); ok || reason != "stale_revision" {
		t.Fatalf("edited passage: %v %s", ok, reason)
	}

	// An edit elsewhere leaves this passage where it was: the anchor
	// relocates and the caller may refresh its revision.
	elsewhere := anchorTestBook()
	elsewhere.Body[1].Content = "<p>The skiff was gone.</p>"
	if ok, reason := LocateAnchor(&elsewhere, good); !ok || reason != "relocated" {
		t.Fatalf("edit elsewhere: %v %s", ok, reason)
	}
}

func TestAnchorFromSpanRefusesUnlocatableQuote(t *testing.T) {
	book := anchorTestBook()
	locator := NewLocator(&book)
	text, _ := locator.ChapterText("ch-tower")
	quote := "Eloise rowed the skiff across the harbour."
	start := strings.Index(text, quote)
	span := types.NarrativeEvidenceSpan{ChapterID: "ch-tower", StartOffset: start, EndOffset: start + len(quote), Quote: quote}
	anchor, err := locator.AnchorFromSpan(span)
	if err != nil {
		t.Fatal(err)
	}
	if anchor.Space != types.PlannerEvidenceChapterSpace || anchor.BlockID != "" || anchor.Scene != 3 || anchor.Revision != locator.Revision() {
		t.Fatalf("anchor %+v", anchor)
	}
	if ok, reason := locator.Locate(anchor); !ok {
		t.Fatalf("emitted anchor does not locate: %s", reason)
	}
	stamped, err := AnchorFromSpan(&book, span, "rev-x")
	if err != nil || stamped.Revision != "rev-x" || stamped.Quote != quote {
		t.Fatalf("stamped anchor %+v (%v)", stamped, err)
	}

	for name, bad := range map[string]types.NarrativeEvidenceSpan{
		"shifted offsets": {ChapterID: "ch-tower", StartOffset: start + 1, EndOffset: start + 1 + len(quote), Quote: quote},
		"other text":      {ChapterID: "ch-tower", StartOffset: start, EndOffset: start + len(quote), Quote: "Eloise rowed the skiff across the river."},
		"empty quote":     {ChapterID: "ch-tower", StartOffset: start, EndOffset: start, Quote: ""},
		"unknown chapter": {ChapterID: "ch-gone", StartOffset: 0, EndOffset: len(quote), Quote: quote},
		"out of range":    {ChapterID: "ch-tower", StartOffset: start, EndOffset: len(text) + 10, Quote: quote},
	} {
		if _, err := locator.AnchorFromSpan(bad); err == nil {
			t.Fatalf("%s: span became an anchor", name)
		}
	}
}

func TestSceneAtAndSceneCountFollowEveryMarkerForm(t *testing.T) {
	html := "<p>one</p><hr><p>two</p><p>***</p><p>three</p><p>* * *</p><p>four</p><p>⁂</p><p>five</p><p>###</p><p>six</p><p>---</p><p>seven</p><p>~ ~ ~</p><p>eight</p><p>. . .</p><p>nine</p>"
	text := StripHTMLForAnalysis(html)
	if got := SceneCount(text); got != 9 {
		t.Fatalf("SceneCount = %d, want 9", got)
	}
	for scene, word := range []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine"} {
		at := strings.Index(text, word)
		if got := SceneAt(text, at); got != scene+1 {
			t.Fatalf("SceneAt(%q) = %d, want %d", word, got, scene+1)
		}
	}
	if SceneCount("") != 0 || SceneCount("  \n ") != 0 {
		t.Fatal("empty text has scenes")
	}
	if IsSceneBreakParagraph("*** and then") || IsSceneBreakParagraph("Rhea") || !IsSceneBreakParagraph("  ⁂ ") {
		t.Fatal("IsSceneBreakParagraph")
	}
}
