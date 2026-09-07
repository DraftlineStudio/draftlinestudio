package fingerprint

import (
	"testing"

	"draftline/internal/types"
)

func TestHigherLevelStructureWaitsForValidatedNarrativeFingerprints(t *testing.T) {
	wake := record("wake", 0, 0, "Avery opened their eyes beneath the clinic ceiling.")
	wake.CharacterIDs, wake.CharacterNames = []string{"avery"}, []string{"Avery"}
	wake.Action = "opened"
	wake.NamedEntities = []types.EvidenceTerm{{Text: "Harbor Clinic", Label: "FAC"}}
	book := testBook([]types.EvidenceRecord{wake})
	model := Build(&book, nil)
	if model.Structure != nil || len(model.Threads) != 0 {
		t.Fatalf("unvalidated hierarchy must not consume evidence atoms: structure=%#v threads=%#v", model.Structure, model.Threads)
	}
	if len(model.Assertions) == 0 {
		t.Fatal("the underlying evidence assertion was lost")
	}
}

func TestIncidentalRelativePhraseDoesNotCreateARealityScope(t *testing.T) {
	incidental := record("incidental", 0, 0, "Avery still wore the coat purchased two years ago.")
	incidental.TimeExpressions = []string{"two years ago"}
	incidental.CharacterIDs, incidental.CharacterNames = []string{"avery"}, []string{"Avery"}
	book := testBook([]types.EvidenceRecord{incidental})
	model := Build(&book, nil)
	if len(model.Assertions) == 0 || model.Assertions[0].Scope.Kind != "current" {
		t.Fatalf("incidental history relocated the containing scene: %#v", model.Assertions)
	}
}
