package fingerprint

import (
	"testing"

	"draftline/internal/types"
)

func TestBuildSeparatesNarrativeOrderFromRelativeStoryTime(t *testing.T) {
	book := testBook([]types.EvidenceRecord{
		record("present", 0, 0, "It was Friday, and Hanlon had been in interrogation for three days."),
		record("past", 1, 0, "Three days ago, on Tuesday, Hanlon found the tunnel."),
	})
	model := Build(&book, nil)
	if len(model.Assertions) != 2 {
		t.Fatalf("expected two assertions, got %d", len(model.Assertions))
	}
	if model.Assertions[0].Temporal.DayOffset == nil || *model.Assertions[0].Temporal.DayOffset != 5 {
		t.Fatalf("expected Friday anchor, got %#v", model.Assertions[0].Temporal)
	}
	if model.Assertions[1].Temporal.DayOffset == nil || *model.Assertions[1].Temporal.DayOffset != 2 {
		t.Fatalf("expected Tuesday story time, got %#v", model.Assertions[1].Temporal)
	}
	if model.Assertions[1].Scope.Kind != "current" {
		t.Fatal("an incidental relative phrase must not relocate its containing scene")
	}
}

func TestBuildPreservesAuthorModelAndMarksDeletedCorrectionOrphaned(t *testing.T) {
	book := testBook([]types.EvidenceRecord{record("kept", 0, 0, "Hanlon found the tunnel on Sunday.")})
	book.Analysis.Fingerprint = &types.StoryFingerprint{AuthorModel: types.StoryAuthorModel{
		Canon:       []types.CanonRule{{ID: "canon-1", Subject: "Hanlon", Predicate: "hair", Object: "brown"}},
		Corrections: []types.FingerprintCorrection{{ID: "correction-1", TargetID: "deleted-event", Status: "active"}},
	}}
	model := Build(&book, nil)
	if len(model.AuthorModel.Canon) != 1 {
		t.Fatal("author canon was not preserved")
	}
	if model.AuthorModel.Corrections[0].Status != "orphaned" {
		t.Fatalf("expected orphaned correction, got %q", model.AuthorModel.Corrections[0].Status)
	}
}

func TestRelativeClaimAnchorsToConversationTimeAcrossPastContext(t *testing.T) {
	book := testBook([]types.EvidenceRecord{
		record("friday", 0, 0, "It was Friday in the interrogation room."),
		record("claim", 0, 2, `"Three days ago we found the tunnel," Ruiz said.`),
	})
	model := Build(&book, nil)
	var claim *types.StoryAssertion
	for index := range model.Assertions {
		if containsString(model.Assertions[index].EvidenceIDs, "claim") {
			claim = &model.Assertions[index]
			break
		}
	}
	if claim == nil || claim.Temporal.DayOffset == nil || *claim.Temporal.DayOffset != 2 {
		t.Fatalf("expected Friday minus three days, got %#v", claim)
	}
	var posture string
	for _, constraint := range model.TemporalConstraints {
		if constraint.FromEvidenceID == "claim" && constraint.Relation == "offset" {
			posture = constraint.Posture
		}
	}
	if posture != "claimed" {
		t.Fatalf("expected dialogue time to remain a claim, got %q", posture)
	}
}

func testBook(records []types.EvidenceRecord) types.BookData {
	return types.BookData{Body: []types.ChapterItem{{ID: "chapter-1", Title: "Chapter 1", Type: "chapter"}, {ID: "chapter-2", Title: "Chapter 2", Type: "chapter"}}, Analysis: types.AnalysisData{Evidence: &types.EvidenceData{ContentHash: "hash", Records: records}}}
}

func record(id string, chapter, paragraph int, text string) types.EvidenceRecord {
	return types.EvidenceRecord{ID: id, Kind: "event", EvidenceType: "discovery", ChapterID: "chapter-1", ChapterIndex: chapter, ParagraphIndex: paragraph, Text: text, TimeExpressions: []string{text}, Confidence: .9, Status: "detected", Source: "auto"}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
