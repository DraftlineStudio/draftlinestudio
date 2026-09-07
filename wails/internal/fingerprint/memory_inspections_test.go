package fingerprint

import (
	"testing"

	"draftline/internal/types"
)

func TestPossessionHistoryRemainsSeparateFromNarrativeDevelopment(t *testing.T) {
	acquired := fixtureRecord("acquired", 0, "Avery acquired the brass key.", "acquired", "state", "avery", "Avery")
	carried := fixtureRecord("carried", 1, "Avery carried the brass key.", "carried", "state", "avery", "Avery")
	surrendered := fixtureRecord("surrendered", 2, "Avery surrendered the brass key.", "surrendered", "state", "avery", "Avery")
	reacquired := fixtureRecord("reacquired", 3, "Avery received the brass key again.", "received", "state", "avery", "Avery")
	model := buildFixture(acquired, carried, surrendered, reacquired)
	history := findStateHistory(t, model.StateHistories, "avery", "possession")
	if len(history.Entries) != 4 {
		t.Fatalf("possession transitions were lost: %#v", history)
	}
	want := []string{"acquire", "carry", "relinquish", "acquire"}
	for index, operation := range want {
		if history.Entries[index].Operation != operation {
			t.Fatalf("entry %d operation = %q, want %q: %#v", index, history.Entries[index].Operation, operation, history)
		}
	}
	if len(model.NarrativeDevelopments) != 0 {
		t.Fatalf("state history leaked into developments before synthesis: %#v", model.NarrativeDevelopments)
	}
}

func TestLocationAndKnowledgeHistoriesRetainMundaneMemory(t *testing.T) {
	arrived := fixtureRecord("arrived", 0, "Avery arrived at the north depot.", "arrived", "transition", "avery", "Avery")
	arrived.NamedEntities = []types.EvidenceTerm{{Text: "north depot", Label: "LOC"}}
	learned := fixtureRecord("learned", 1, "Avery learned that the western gate was locked.", "learned", "knowledge_change", "avery", "Avery")
	learned.KnowledgeStates = []types.EvidenceKnowledgeState{{State: "learned", CharacterIDs: []string{"avery"}, CharacterNames: []string{"Avery"}, Cue: "learned", Confidence: .9}}
	model := buildFixture(arrived, learned)
	if len(findStateHistory(t, model.StateHistories, "avery", "location").Entries) != 1 {
		t.Fatal("location history missing")
	}
	if len(findStateHistory(t, model.StateHistories, "avery", "knowledge").Entries) == 0 {
		t.Fatal("knowledge history missing")
	}
}

func TestInspectionsUseFingerprintEvidenceFromBothSides(t *testing.T) {
	first := fixtureRecord("hair-one", 0, "Avery had blonde hair.", "had", "state", "avery", "Avery")
	second := fixtureRecord("hair-two", 2, "Avery had brown hair.", "had", "state", "avery", "Avery")
	model := buildFixture(first, second)
	inspection := findInspection(t, model.Inspections, "attribute_conflict")
	if len(inspection.Sides) < 2 || len(inspection.Sides[0].EvidenceSpans) == 0 || len(inspection.Sides[1].EvidenceSpans) == 0 {
		t.Fatalf("inspection did not preserve both sources: %#v", inspection)
	}
}

func TestConflictingEventAccountsAreReviewEvidenceNotAutomaticAuthorError(t *testing.T) {
	first := fixtureRecord("account-one", 0, `"Avery fired first," Mira testified.`, "fired", "interaction", "avery", "Avery", "mira", "Mira")
	second := fixtureRecord("account-two", 1, `"Oren fired first," Sela testified.`, "fired", "interaction", "oren", "Oren", "sela", "Sela")
	model := buildFixture(first, second)
	inspection := findInspection(t, model.Inspections, "conflicting_event_accounts")
	if inspection.Severity == "error" || len(inspection.Sides) != 2 {
		t.Fatalf("competing testimony was flattened into author error: %#v", inspection)
	}
}

func TestDifferentRealityScopesDoNotCreateStateConflict(t *testing.T) {
	current := fixtureRecord("current", 0, "Avery had blonde hair.", "had", "state", "avery", "Avery")
	dream := fixtureRecord("dream-hair", 1, "In a dream, Avery had black hair.", "had", "state", "avery", "Avery")
	model := buildFixture(current, dream)
	for _, inspection := range model.Inspections {
		if inspection.Kind == "attribute_conflict" {
			t.Fatalf("incompatible realities were treated as one state: %#v", inspection)
		}
	}
}

func TestUnfulfilledPromiseRemainsInspectableWithoutInventedResolution(t *testing.T) {
	promise := fixtureRecord("promise", 0, "Avery promised to return the ledger.", "promised", "interaction", "avery", "Avery")
	model := buildFixture(promise)
	inspection := findInspection(t, model.Inspections, "open_obligation")
	if len(inspection.Sides) != 1 || len(inspection.Sides[0].EvidenceSpans) != 1 {
		t.Fatalf("open obligation lost its source: %#v", inspection)
	}
}

func findStateHistory(t *testing.T, histories []types.FingerprintStateHistory, entityID, property string) types.FingerprintStateHistory {
	t.Helper()
	for _, history := range histories {
		if history.EntityID == entityID && history.Property == property {
			return history
		}
	}
	t.Fatalf("missing %s history for %s: %#v", property, entityID, histories)
	return types.FingerprintStateHistory{}
}

func findInspection(t *testing.T, inspections []types.FingerprintInspection, kind string) types.FingerprintInspection {
	t.Helper()
	for _, inspection := range inspections {
		if inspection.Kind == kind {
			return inspection
		}
	}
	t.Fatalf("missing %s inspection: %#v", kind, inspections)
	return types.FingerprintInspection{}
}
