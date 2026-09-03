package fingerprint

import (
	"testing"

	"draftline/internal/types"
)

func TestStructureAggregatesWithoutLosingEvidence(t *testing.T) {
	opened := record("wake-1", 0, 0, "Daniel opened his eyes beneath the hospital ceiling.")
	opened.CharacterIDs, opened.CharacterNames = []string{"daniel"}, []string{"Daniel"}
	opened.Action = "opened"
	opened.NamedEntities = []types.EvidenceTerm{{Text: "Northwestern Memorial Hospital", Label: "FAC"}}
	recognized := record("wake-2", 0, 0, "Daniel recognized the Northwestern Memorial Hospital logo.")
	recognized.EvidenceType = "discovery"
	recognized.CharacterIDs, recognized.CharacterNames = []string{"daniel"}, []string{"Daniel"}
	recognized.Action = "recognized"
	recognized.NamedEntities = []types.EvidenceTerm{{Text: "Northwestern Memorial Hospital", Label: "FAC"}}

	book := testBook([]types.EvidenceRecord{opened, recognized})
	model := Build(&book, nil)
	if model.Structure == nil {
		t.Fatal("expected story structure")
	}
	if got := len(model.Structure.SignificantEvents); got != 1 {
		t.Fatalf("expected one significant event, got %d", got)
	}
	if got := len(model.Structure.SignificantEvents[0].EvidenceIDs); got != 2 {
		t.Fatalf("aggregate lost evidence: %#v", model.Structure.SignificantEvents[0].EvidenceIDs)
	}
	if len(model.Structure.Scenes) != 1 || len(model.Structure.Sequences) != 1 || len(model.Structure.NarrativeThreads) != 1 || len(model.Structure.Arcs) != 1 {
		t.Fatalf("incomplete hierarchy: %#v", model.Structure)
	}
	for _, id := range model.Structure.SignificantEvents[0].EvidenceIDs {
		if len(model.Structure.Cache.EvidenceDependents[id]) == 0 {
			t.Fatalf("missing reverse dependency for %s", id)
		}
	}
}

func TestStructureKeepsTemporalContextsInSeparateScenes(t *testing.T) {
	first := record("present", 0, 0, "Hanlon sat in the interrogation room on Friday.")
	first.TimeExpressions = []string{"Friday"}
	second := record("memory", 0, 1, "Three days ago Hanlon entered the tunnel.")
	second.EvidenceType = "transition"
	second.TimeExpressions = []string{"three days ago"}
	for _, item := range []*types.EvidenceRecord{&first, &second} {
		item.CharacterIDs, item.CharacterNames = []string{"hanlon"}, []string{"Hanlon"}
	}
	book := testBook([]types.EvidenceRecord{first, second})
	model := Build(&book, nil)
	if len(model.Structure.Scenes) < 2 {
		t.Fatalf("temporal transition was flattened into one scene: %#v", model.Structure.Scenes)
	}
}
