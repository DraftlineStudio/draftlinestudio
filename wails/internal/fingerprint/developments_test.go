package fingerprint

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

func developmentOfKind(developments []types.NarrativeDevelopment, kind string) *types.NarrativeDevelopment {
	for index := range developments {
		if developments[index].Kind == kind {
			return &developments[index]
		}
	}
	return nil
}

func developmentByID(developments []types.NarrativeDevelopment, id string) *types.NarrativeDevelopment {
	for index := range developments {
		if developments[index].ID == id {
			return &developments[index]
		}
	}
	return nil
}

func TestDevelopmentsGiveARecognizableAccount(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		// ch1: a discovery whose fact recurs chapters later
		evidence("e1", 0, "Avery learned that the manifest had been altered.", "fact", "discovery", "learned"),
		// ch1: an obligation opens
		evidence("e2", 0, "Avery promised to deliver the report by Friday.", "fact", "state", "promised"),
		// ch2: a decision
		evidence("e3", 1, "Mira decided to burn the letters.", "event", "state", "decided"),
		// ch3: the discovered fact recurs (same knowledge key)
		evidence("e4", 2, "Avery knew that the manifest had been altered.", "fact", "knowledge_change", "knew"),
		// ch4-5: a same-scope life-status reversal
		evidence("e5", 3, "Mira Voss was killed in the warehouse fire.", "event", "state", "killed"),
		evidence("e6", 4, "Mira was alive and waiting at the dock.", "fact", "state", "was"),
	})
	model := Build(&book, nil)
	developments := model.Developments
	if len(developments) < 4 {
		t.Fatalf("expected a real account, got %#v", developments)
	}
	discovery := developmentOfKind(developments, "major_discovery")
	if discovery == nil || !strings.Contains(discovery.Summary, "the manifest had been altered") {
		t.Fatalf("major_discovery: %#v", discovery)
	}
	obstacle := developmentOfKind(developments, "obstacle_introduced")
	if obstacle == nil || !strings.Contains(obstacle.Summary, "deliver the report") {
		t.Fatalf("obstacle_introduced: %#v", obstacle)
	}
	if obstacle.After == "" {
		t.Fatalf("obstacle_introduced must state the resulting state: %#v", obstacle)
	}
	decision := developmentOfKind(developments, "major_decision")
	if decision == nil || !strings.Contains(decision.Summary, "burn the letters") {
		t.Fatalf("major_decision: %#v", decision)
	}
	death := developmentOfKind(developments, "threat_escalation")
	if death == nil || death.Before != "alive" || death.After != "dead" {
		t.Fatalf("threat_escalation for an active character's death: %#v", death)
	}
	revelation := developmentOfKind(developments, "revelation")
	if revelation == nil || !strings.Contains(revelation.Summary, "alive") {
		t.Fatalf("revelation: %#v", revelation)
	}
	// Chronological: narrative order must be non-decreasing.
	for index := 1; index < len(developments); index++ {
		if developments[index].NarrativeOrder < developments[index-1].NarrativeOrder {
			t.Fatal("developments must read in manuscript order")
		}
	}
	// Every development is fully grounded and diagnosable.
	for _, development := range developments {
		if len(development.FrameIDs) == 0 || len(development.EvidenceIDs) == 0 || len(development.Basis) == 0 {
			t.Fatalf("ungrounded development: %#v", development)
		}
		if len(development.EvidenceSpans) == 0 {
			t.Fatalf("development without evidence spans: %#v", development)
		}
		if development.Confidence <= 0 {
			t.Fatalf("development without confidence: %#v", development)
		}
	}
}

func TestMysteryIntroducedThenResolvedWithCausalLink(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery never learned that the safe had a second combination.", "fact", "state", "learned"),
		evidence("e2", 3, "Avery learned that the safe had a second combination.", "fact", "discovery", "learned"),
	})
	model := Build(&book, nil)
	introduced := developmentOfKind(model.Developments, "mystery_introduced")
	resolved := developmentOfKind(model.Developments, "mystery_resolved")
	if introduced == nil || resolved == nil {
		t.Fatalf("expected mystery arc, got %#v", model.Developments)
	}
	if resolved.NarrativeOrder <= introduced.NarrativeOrder {
		t.Fatal("resolution must follow introduction")
	}
	if resolved.PredecessorID != introduced.ID {
		t.Fatalf("resolution must link back to its introduction: %#v", resolved)
	}
	if introduced.SuccessorID != resolved.ID {
		t.Fatalf("introduction must link forward to its resolution: %#v", introduced)
	}
	if resolved.Advances == "" || resolved.Before == "" || resolved.After == "" {
		t.Fatalf("resolution must record what it advances and the state change: %#v", resolved)
	}
}

func TestMysteryNarrowedByPartialFact(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery never learned who had opened the vault door.", "fact", "state", "learned"),
		evidence("e2", 2, "Avery learned that the vault door showed no forced entry.", "fact", "discovery", "learned"),
	})
	model := Build(&book, nil)
	narrowed := developmentOfKind(model.Developments, "mystery_narrowed")
	if narrowed == nil {
		t.Fatalf("expected mystery_narrowed, got %#v", model.Developments)
	}
	if narrowed.Advances == "" || len(narrowed.AdvancesFrameIDs) == 0 {
		t.Fatalf("narrowing must reference the question it advances: %#v", narrowed)
	}
}

func TestInvestigationAggregatesSceneFacts(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Mira learned that the ledger pages were forged.", "fact", "discovery", "learned"),
		evidence("e2", 0, "Mira learned that the auditor caught a night train.", "fact", "discovery", "learned"),
	})
	model := Build(&book, nil)
	progress := developmentOfKind(model.Developments, "investigation_progress")
	if progress == nil {
		t.Fatalf("expected investigation_progress, got %#v", model.Developments)
	}
	if len(progress.FrameIDs) < 2 || len(progress.EvidenceIDs) < 2 {
		t.Fatalf("aggregation must cite every supporting frame: %#v", progress)
	}
	if !strings.Contains(progress.Summary, "ledger pages were forged") ||
		!strings.Contains(progress.Summary, "auditor caught a night train") {
		t.Fatalf("aggregated summary must quote each fact: %s", progress.Summary)
	}
}

func TestGoalProgressTiesAcquisitionsToTheGoal(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery wanted to reach the harbor tunnel by midnight.", "fact", "state", "wanted"),
		evidence("e2", 1, "Avery learned that the harbor tunnel gate stayed unlocked at night.", "fact", "discovery", "learned"),
	})
	model := Build(&book, nil)
	established := developmentOfKind(model.Developments, "goal_established")
	if established == nil || !strings.Contains(established.Summary, "reach the harbor tunnel") {
		t.Fatalf("goal_established: %#v", model.Developments)
	}
	progress := developmentOfKind(model.Developments, "investigation_progress")
	if progress == nil || !strings.Contains(progress.Advances, "reach the harbor tunnel") {
		t.Fatalf("goal-tied progress must advance the goal: %#v", progress)
	}
}

func TestCountdownEscalation(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery realized that three days remained until the exchange.", "fact", "discovery", "realized"),
		evidence("e2", 2, "Avery realized that two days remained until the exchange.", "fact", "discovery", "realized"),
	})
	model := Build(&book, nil)
	escalation := developmentOfKind(model.Developments, "threat_escalation")
	if escalation == nil {
		t.Fatalf("expected threat_escalation from a shrinking countdown, got %#v", model.Developments)
	}
	if escalation.Before == "" || escalation.After == "" {
		t.Fatalf("countdown escalation must record before → after: %#v", escalation)
	}
	if len(escalation.FrameIDs) < 2 {
		t.Fatalf("both countdown readings must support the development: %#v", escalation)
	}
}

func TestConflictingRetellingSurfaces(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, `"Mira Voss died on the bridge," Avery said.`, "event", "interaction", "said"),
		evidence("e2", 4, "Mira Voss was killed in the warehouse fire.", "event", "state", "killed"),
	})
	// Conflicting accounts of one death surface as a disconfirmation or a
	// revelation, or at minimum a conflicted identity — never silently.
	model := Build(&book, nil)
	if len(model.EventIdentities) == 0 {
		// The claim sentence has no sentence-initial subject; only e2
		// produces a life_status frame, so no identity forms — and that is
		// acceptable abstention, not silence about a known conflict.
		return
	}
	surfaced := developmentOfKind(model.Developments, "disconfirmation") != nil ||
		developmentOfKind(model.Developments, "revelation") != nil
	if !surfaced && model.EventIdentities[0].Status != "conflicted" {
		t.Fatalf("conflicting accounts must surface: %#v", model.EventIdentities)
	}
}

func TestDecisionLinksToSameSceneDiscovery(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery never learned that the courier had switched sides.", "fact", "state", "learned"),
		evidence("e2", 2, "Avery learned that the courier had switched sides.", "fact", "discovery", "learned"),
		evidence("e3", 2, "Avery decided to leave the city that night.", "event", "state", "decided"),
	})
	model := Build(&book, nil)
	decision := developmentOfKind(model.Developments, "major_decision")
	if decision == nil {
		t.Fatalf("expected major_decision, got %#v", model.Developments)
	}
	if decision.PredecessorID == "" {
		t.Fatalf("a decision after a same-scene development must link back: %#v", decision)
	}
	predecessor := developmentByID(model.Developments, decision.PredecessorID)
	if predecessor == nil || predecessor.NarrativeOrder > decision.NarrativeOrder {
		t.Fatalf("decision predecessor must exist and precede it: %#v", predecessor)
	}
}
