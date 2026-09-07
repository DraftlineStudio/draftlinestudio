package fingerprint

import (
	"testing"

	"draftline/internal/types"
)

func TestPresenceLedgerFindsKyleStillInRoomWhenHanlonIsAlone(t *testing.T) {
	kyle := record("kyle", 0, 0, "Kyle sat with Hanlon at the interrogation table.")
	kyle.CharacterIDs = []string{"kyle", "hanlon"}
	kyle.CharacterNames = []string{"Kyle", "Hanlon"}
	kyle.Action = "sat"
	kyle.EvidenceType = "introduction"
	alvarez := record("alvarez", 0, 1, "Alvarez entered, then left the room.")
	alvarez.CharacterIDs = []string{"alvarez"}
	alvarez.CharacterNames = []string{"Alvarez"}
	alvarez.Action = "left"
	alvarez.EvidenceType = "transition"
	alone := record("alone", 0, 2, "That left Hanlon alone with his thoughts.")
	alone.CharacterIDs = []string{"hanlon"}
	alone.CharacterNames = []string{"Hanlon"}
	alone.Action = "left"
	alone.EvidenceType = "state"
	book := testBook([]types.EvidenceRecord{kyle, alvarez, alone})
	model := Build(&book, nil)
	if !hasDiagnostic(model.Diagnostics, "unclosed_presence") {
		t.Fatalf("expected unclosed presence diagnostic, got %#v", model.Diagnostics)
	}
}

func TestContradictoryHairAndNamesAreFlagged(t *testing.T) {
	first := record("first", 0, 0, "Ruiz's first name was Carlos, and he had blonde hair.")
	first.CharacterIDs = []string{"ruiz"}
	first.CharacterNames = []string{"Ruiz"}
	second := record("second", 2, 0, "Ruiz's first name was Jose, and he had brown hair.")
	second.CharacterIDs = []string{"ruiz"}
	second.CharacterNames = []string{"Ruiz"}
	book := testBook([]types.EvidenceRecord{first, second})
	model := Build(&book, nil)
	if !hasDiagnostic(model.Diagnostics, "attribute_conflict") || !hasDiagnostic(model.Diagnostics, "identity_conflict") {
		t.Fatalf("expected attribute and identity conflicts, got %#v", model.Diagnostics)
	}
}

func TestNearDuplicateChaptersRequireRealityReview(t *testing.T) {
	content := ""
	for i := 0; i < 40; i++ {
		content += "Hanlon woke beneath the familiar logo and listened to the monitor beep. "
	}
	book := testBook(nil)
	book.Body = []types.ChapterItem{{ID: "prologue", Title: "Prologue", Content: content}, {ID: "epilogue", Title: "Epilogue", Content: content + " RESET COMPLETE"}}
	model := Build(&book, nil)
	if !hasDiagnostic(model.Diagnostics, "near_duplicate_chapter") {
		t.Fatalf("expected duplicate chapter diagnostic, got %#v", model.Diagnostics)
	}
}

func TestQueryAnswersWhenFromStoryChronology(t *testing.T) {
	discovery := record("tunnel", 2, 0, "On Tuesday, Hanlon discovered the underground tunnel beneath IBM.")
	discovery.CharacterIDs = []string{"hanlon"}
	discovery.CharacterNames = []string{"Hanlon"}
	discovery.Action = "discovered"
	discovery.NamedEntities = []types.EvidenceTerm{{Text: "IBM", Label: "ORG"}}
	book := testBook([]types.EvidenceRecord{discovery})
	book.Analysis.Fingerprint = Build(&book, nil)
	answer := Query(book, types.FingerprintQueryRequest{Query: "When did Hanlon discover the tunnel?"})
	if !answer.Success || len(answer.Events) == 0 || answer.Events[0].StoryTime.Label == "" {
		t.Fatalf("expected chronological evidence answer, got %#v", answer)
	}
}

func TestAuthorCorrectionOverridesTimeAndSurvivesByEvidence(t *testing.T) {
	discovery := record("tunnel", 2, 0, "Hanlon discovered the tunnel sometime in the past.")
	book := testBook([]types.EvidenceRecord{discovery})
	book.Analysis.Fingerprint = &types.StoryFingerprint{AuthorModel: types.StoryAuthorModel{Corrections: []types.FingerprintCorrection{{ID: "time-fix", Kind: "story_day", Value: "-90", EvidenceIDs: []string{"tunnel"}}}}}
	model := Build(&book, nil)
	if len(model.Assertions) == 0 || model.Assertions[0].Temporal.DayOffset == nil || *model.Assertions[0].Temporal.DayOffset != -90 {
		t.Fatalf("author chronology correction was not applied to the assertion: %#v", model.Assertions)
	}
	if model.AuthorModel.Corrections[0].Status != "active" {
		t.Fatalf("evidence-backed correction should remain active: %#v", model.AuthorModel.Corrections[0])
	}
}

func hasDiagnostic(values []types.FingerprintDiagnostic, kind string) bool {
	for _, value := range values {
		if value.Kind == kind {
			return true
		}
	}
	return false
}
