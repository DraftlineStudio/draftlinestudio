package indexing

import (
	"strconv"
	"strings"
	"testing"

	"draftline/internal/types"
)

// A "***" marker satisfies both the spaced and the run pattern; it is one
// scene break, not two.
func TestSceneBreakMarkerCountsOnce(t *testing.T) {
	text := StripHTML("<p>First scene.</p><p>***</p><p>Second scene.</p><hr><p>Third scene.</p>")
	breaks := 0
	for _, scene := range DetectScenes(text, 0) {
		if scene.SceneType == "scene_break" {
			breaks++
		}
	}
	if breaks != 2 {
		t.Fatalf("expected 2 scene breaks, got %d", breaks)
	}
	if got := len(SceneBreakOffsets(text)); got != 2 {
		t.Fatalf("expected 2 distinct break offsets, got %d", got)
	}
}

// Every marker paragraph opens a scene, including one at the top of a
// chapter and each of two in a row: the writer's marker is the rule, so
// the Planner, the story analysis, and the manuscript memory all count the
// same scenes.
func TestLeadingAndConsecutiveMarkersEachOpenAScene(t *testing.T) {
	text := StripHTMLForAnalysis("<p>***</p><p>First.</p><p>***</p><p>* * *</p><p>Second.</p>")
	if got := SceneCount(text); got != 4 {
		t.Fatalf("SceneCount = %d, want 4 (an empty first scene, First, an empty scene, Second)", got)
	}
	if got := SceneAt(text, strings.Index(text, "First.")); got != 2 {
		t.Fatalf("First. is in scene %d, want 2", got)
	}
	if got := SceneAt(text, strings.Index(text, "Second.")); got != 4 {
		t.Fatalf("Second. is in scene %d, want 4", got)
	}
	if got := SceneCount(StripHTMLForAnalysis("<p>Only prose.</p>")); got != 1 {
		t.Fatalf("SceneCount without markers = %d, want 1", got)
	}
}

// The first sentence after a scene break must be indexed without the
// marker attached, and its offsets must still locate the text — whether the
// segmenter glued the marker to the front of that sentence or, when the
// paragraph before the marker had no terminal punctuation, left it in the
// middle of one.
func TestEvidenceSentenceAfterSceneBreakDropsTheMarker(t *testing.T) {
	for _, before := range []string{"<p>Rhea walked into the tower.</p>", "<p>Rhea walked into the tower</p>"} {
		for _, marker := range []string{"<hr>", "<p>***</p>", "<p>- - -</p>"} {
			assertSentenceAfterBreakIndexed(t, before+marker+"<p>Rhea left the tower at dawn.</p>", marker)
		}
	}
}

func assertSentenceAfterBreakIndexed(t *testing.T, content, marker string) {
	t.Helper()
	{
		text := StripHTMLForAnalysis(content)
		book := types.BookData{
			Body:     []types.ChapterItem{{ID: "chapter-one", Title: "Chapter One", Type: "chapter", Content: content}},
			Analysis: types.AnalysisData{EntityResolution: &types.EntityData{Version: 1}},
		}
		entity := types.EntityRecord{ID: "rhea", Canonical: "Rhea", Aliases: []string{"Rhea"}, DetectionStatus: "accepted"}
		for offset := strings.Index(text, "Rhea"); offset >= 0; {
			mentionID := "mention-" + strconv.Itoa(offset)
			book.Analysis.EntityResolution.Mentions = append(book.Analysis.EntityResolution.Mentions, types.MentionRecord{ID: mentionID, Text: "Rhea", Chapter: 0, CharOffset: offset})
			entity.MentionIDs = append(entity.MentionIDs, mentionID)
			next := strings.Index(text[offset+4:], "Rhea")
			if next < 0 {
				break
			}
			offset += 4 + next
		}
		book.Analysis.EntityResolution.Entities = []types.EntityRecord{entity}
		records := AnalyzeEvidence(&book, nil).Records
		found, foundBefore := false, false
		for _, record := range records {
			if strings.ContainsAny(record.Text, "*-\n") {
				t.Fatalf("%s: record text carries the scene marker: %q", marker, record.Text)
			}
			if text[record.StartOffset:record.EndOffset] != record.Text {
				t.Fatalf("%s: record offsets do not locate its text", marker)
			}
			if strings.HasPrefix(record.Text, "Rhea left") {
				found = true
			}
			if strings.HasPrefix(record.Text, "Rhea walked") {
				foundBefore = true
			}
		}
		if !found || !foundBefore {
			t.Fatalf("%s: both sentences around the break must be indexed (after=%v before=%v) in %q", marker, found, foundBefore, content)
		}
	}
}

func TestSplitAtSceneBreaksLocatesEveryPiece(t *testing.T) {
	text := "Rhea walked into the tower\n\n***\n\nRhea left the tower at dawn.\n\n- - -\n\nTomas waited."
	pieces := splitAtSceneBreaks(text, 100)
	if len(pieces) != 3 {
		t.Fatalf("expected 3 prose pieces, got %d: %+v", len(pieces), pieces)
	}
	for _, piece := range pieces {
		if text[piece.start-100:piece.end-100] != piece.text {
			t.Fatalf("piece offsets do not locate its text: %+v", piece)
		}
	}
	if pieces[0].text != "Rhea walked into the tower" || pieces[2].text != "Tomas waited." {
		t.Fatalf("unexpected pieces: %+v", pieces)
	}
}

// Every marker form this rule accepts is mirrored by the Planner, which
// counts scenes itself before the first analysis
// (frontend/src/components/planner/plannerModel.ts BREAK_MARKERS). The two
// lists are one list; a form added here is added there.
func TestEverySceneBreakMarkerFormCountsOneBreak(t *testing.T) {
	for _, marker := range []string{"* * *", "***", "****", "⁂", "# # #", "###", "####", "- - -", "---", "----", "~ ~ ~", ". . ."} {
		text := "Rhea waited.\n\n" + marker + "\n\nTomas answered."
		if got := SceneCount(text); got != 2 {
			t.Fatalf("marker %q counted %d scenes, want 2", marker, got)
		}
	}
	if got := SceneCount("Rhea waited.\n\n*** and then\n\nTomas answered."); got != 1 {
		t.Fatalf("a paragraph that is not only a marker counted %d scenes, want 1", got)
	}
}
