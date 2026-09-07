package fingerprint

import (
	"testing"

	"draftline/internal/types"
)

func TestSameDeathToldTwiceBecomesOneConflictedIdentity(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Mira Voss was killed in the warehouse fire.", "event", "state", "killed"),
		evidence("e2", 5, "Mira Voss had died on the bridge years earlier.", "event", "state", "died"),
	})
	model := Build(&book, nil)
	if len(model.EventIdentities) != 1 {
		t.Fatalf("expected one identity, got %#v", model.EventIdentities)
	}
	identity := model.EventIdentities[0]
	if identity.EventClass != types.FrameLifeStatus || len(identity.FrameIDs) != 2 {
		t.Fatalf("identity shape: %#v", identity)
	}
	if identity.Status != "conflicted" || len(identity.Properties[0].Values) != 2 {
		t.Fatalf("conflicting accounts must be preserved as conflicting values: %#v", identity)
	}
}

func TestDifferentSubjectsNeverMerge(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Mira Voss was killed in the warehouse fire.", "event", "state", "killed"),
		evidence("e2", 2, "Avery Cole was killed in the warehouse fire.", "event", "state", "killed"),
	})
	model := Build(&book, nil)
	if len(model.EventIdentities) != 0 {
		t.Fatalf("distinct victims must not merge on shared vocabulary: %#v", model.EventIdentities)
	}
}

func TestSharedVocabularyAloneNeverMerges(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery searched the warehouse office for the missing ledger.", "event", "interaction", "searched"),
		evidence("e2", 3, "Mira searched the warehouse office for the missing ledger.", "event", "interaction", "searched"),
	})
	model := Build(&book, nil)
	for _, identity := range model.EventIdentities {
		names := map[string]bool{}
		for _, p := range identity.Participants {
			names[p.EntityName] = true
		}
		if names["Avery Cole"] && names["Mira Voss"] {
			t.Fatalf("vocabulary-only merge across subjects: %#v", identity)
		}
	}
}

func TestRetoldEventMergesOnSubjectAndActionHead(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 1, "Avery found the tunnel beneath the rail yard.", "event", "discovery", "found"),
		evidence("e2", 6, "Avery found the tunnel that same night, or so the report claimed.", "event", "discovery", "found"),
	})
	model := Build(&book, nil)
	if len(model.EventIdentities) != 1 {
		t.Fatalf("expected the retelling to join one identity, got %#v", model.EventIdentities)
	}
	if len(model.EventIdentities[0].Properties[0].Values) != 2 {
		t.Fatalf("both accounts must be preserved: %#v", model.EventIdentities[0])
	}
}
