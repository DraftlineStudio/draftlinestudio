package fingerprint

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

func TestBuildConsolidatesAdjacentEvidenceIntoOneEvent(t *testing.T) {
	first := record("maps-1", 0, 3, "Hanlon asked Gary for the building maps.")
	first.CharacterIDs = []string{"hanlon", "gary"}
	first.CharacterNames = []string{"Hanlon", "Gary"}
	first.Action = "asked"
	second := record("maps-2", 0, 3, "Gary handed Hanlon the maps.")
	second.CharacterIDs = []string{"gary", "hanlon"}
	second.CharacterNames = []string{"Gary", "Hanlon"}
	second.Action = "handed"
	second.NamedEntities = []types.EvidenceTerm{{Text: "maps", Label: "PRODUCT"}}
	book := testBook([]types.EvidenceRecord{first, second})
	model := Build(&book, nil)
	if len(model.Events) != 1 {
		t.Fatalf("expected one consolidated event, got %d", len(model.Events))
	}
	if len(model.Events[0].EvidenceIDs) != 2 {
		t.Fatalf("expected both sources, got %v", model.Events[0].EvidenceIDs)
	}
	if !strings.Contains(strings.ToLower(model.Events[0].Summary), "hanlon") {
		t.Fatalf("expected mechanical summary, got %q", model.Events[0].Summary)
	}
}

func TestAssertionsKeepClaimsBeliefsAndNegationDistinct(t *testing.T) {
	claim := record("claim", 0, 0, `"Ruiz never knew about IBM," Hanlon said.`)
	claim.CharacterIDs = []string{"hanlon"}
	claim.CharacterNames = []string{"Hanlon"}
	claim.Action = "knew"
	claim.NamedEntities = []types.EvidenceTerm{{Text: "IBM", Label: "ORG"}}
	book := testBook([]types.EvidenceRecord{claim})
	model := Build(&book, nil)
	if len(model.Assertions) == 0 {
		t.Fatal("expected an assertion")
	}
	if model.Assertions[0].Posture != "claim" || model.Assertions[0].Polarity != "negative" {
		t.Fatalf("truth posture lost: %#v", model.Assertions[0])
	}
}

func TestPersistentAttributeAndChapterBoundedPresence(t *testing.T) {
	appearance := record("appearance", 0, 0, "Ruiz entered with brown hair and carried a pistol.")
	appearance.EvidenceType = "transition"
	appearance.CharacterIDs = []string{"ruiz"}
	appearance.CharacterNames = []string{"Ruiz"}
	book := testBook([]types.EvidenceRecord{appearance})
	model := Build(&book, nil)
	var attribute, presence *types.StoryStateInterval
	for index := range model.States {
		item := &model.States[index]
		if item.Kind == "attribute" {
			attribute = item
		}
		if item.Kind == "presence" {
			presence = item
		}
	}
	if attribute == nil || !attribute.Persistent {
		t.Fatalf("expected persistent attribute, got %#v", attribute)
	}
	if presence == nil || presence.EndEventID == "" || presence.Persistent {
		t.Fatalf("expected chapter-bounded presence, got %#v", presence)
	}
}
