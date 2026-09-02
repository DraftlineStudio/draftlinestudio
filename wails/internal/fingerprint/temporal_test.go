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
	if len(model.Events) != 2 {
		t.Fatalf("expected two events, got %d", len(model.Events))
	}
	if model.Events[0].NarrativeOrder >= model.Events[1].NarrativeOrder {
		t.Fatal("narrative order was not preserved")
	}
	if model.Events[0].StoryTime.DayOffset == nil || *model.Events[0].StoryTime.DayOffset != 5 {
		t.Fatalf("expected Friday anchor, got %#v", model.Events[0].StoryTime)
	}
	if model.Events[1].StoryTime.DayOffset == nil || *model.Events[1].StoryTime.DayOffset != 2 {
		t.Fatalf("expected Tuesday story time, got %#v", model.Events[1].StoryTime)
	}
	if model.Events[1].ContextID == "context-primary" {
		t.Fatal("expected past context for explicit ago cue")
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
	if model.Events[1].StoryTime.DayOffset == nil || *model.Events[1].StoryTime.DayOffset != 2 {
		t.Fatalf("expected Friday minus three days, got %#v", model.Events[1].StoryTime)
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
