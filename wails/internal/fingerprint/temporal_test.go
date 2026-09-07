package fingerprint

import (
	"testing"

	"draftline/internal/types"
)

func TestSolveTemporalSeparatesNarrativeOrderFromRelativeStoryTime(t *testing.T) {
	book := testBook([]types.EvidenceRecord{
		record("present", 0, 0, "It was Friday, and Hanlon had been in interrogation for three days."),
		record("past", 1, 0, "Three days ago, on Tuesday, Hanlon found the tunnel."),
	})
	records := eligibleEvidence(book.Analysis.Evidence.Records)
	_, contextByEvidence := inferContexts(records, nil)
	constraints := inferTemporalConstraints(records, contextByEvidence)
	points, _ := solveTemporal(records, contextByEvidence, constraints)
	if points["present"].DayOffset == nil || *points["present"].DayOffset != 5 {
		t.Fatalf("expected Friday anchor, got %#v", points["present"])
	}
	if points["past"].DayOffset == nil || *points["past"].DayOffset != 2 {
		t.Fatalf("expected Tuesday story time, got %#v", points["past"])
	}
}

func TestBuildPreservesAuthorModel(t *testing.T) {
	book := testBook([]types.EvidenceRecord{record("kept", 0, 0, "Hanlon found the tunnel on Sunday.")})
	book.Analysis.Fingerprint = &types.StoryFingerprint{AuthorModel: types.StoryAuthorModel{
		Corrections: []types.FingerprintCorrection{{ID: "correction-1", TargetID: "deleted-event", Status: "active"}},
		VoiceNotes:  []types.CharacterVoiceNotes{{CharacterID: "entity-1", Dialect: "declared"}},
	}}
	model := Build(&book, nil)
	if len(model.AuthorModel.Corrections) != 1 || len(model.AuthorModel.VoiceNotes) != 1 {
		t.Fatal("author model was not preserved across rebuild")
	}
}

func TestRelativeClaimAnchorsToConversationTimeAcrossPastContext(t *testing.T) {
	book := testBook([]types.EvidenceRecord{
		record("friday", 0, 0, "It was Friday in the interrogation room."),
		record("claim", 0, 2, `"Three days ago we found the tunnel," Ruiz said.`),
	})
	records := eligibleEvidence(book.Analysis.Evidence.Records)
	_, contextByEvidence := inferContexts(records, nil)
	constraints := inferTemporalConstraints(records, contextByEvidence)
	points, _ := solveTemporal(records, contextByEvidence, constraints)
	if points["claim"].DayOffset == nil || *points["claim"].DayOffset != 2 {
		t.Fatalf("expected Friday minus three days, got %#v", points["claim"])
	}
	var posture string
	for _, constraint := range constraints {
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
