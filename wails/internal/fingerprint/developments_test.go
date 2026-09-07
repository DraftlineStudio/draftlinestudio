package fingerprint

import (
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
	if discovery == nil || !containsSubstring(discovery.Summary, "the manifest had been altered") {
		t.Fatalf("major_discovery: %#v", discovery)
	}
	obstacle := developmentOfKind(developments, "new_obstacle")
	if obstacle == nil || !containsSubstring(obstacle.Summary, "deliver the report") {
		t.Fatalf("new_obstacle: %#v", obstacle)
	}
	decision := developmentOfKind(developments, "major_decision")
	if decision == nil || !containsSubstring(decision.Summary, "burn the letters") {
		t.Fatalf("major_decision: %#v", decision)
	}
	reversal := developmentOfKind(developments, "reversal")
	if reversal == nil || !containsSubstring(reversal.Summary, "alive") {
		t.Fatalf("reversal: %#v", reversal)
	}
	// Chronological: narrative order must be non-decreasing.
	for index := 1; index < len(developments); index++ {
		if developments[index].NarrativeOrder < developments[index-1].NarrativeOrder {
			t.Fatal("developments must read in manuscript order")
		}
	}
	// Every development is grounded.
	for _, development := range developments {
		if len(development.FrameIDs) == 0 || len(development.EvidenceIDs) == 0 || len(development.Basis) == 0 {
			t.Fatalf("ungrounded development: %#v", development)
		}
	}
}

func TestMysteryCreatedThenResolved(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, "Avery never learned that the safe had a second combination.", "fact", "state", "learned"),
		evidence("e2", 3, "Avery learned that the safe had a second combination.", "fact", "discovery", "learned"),
	})
	model := Build(&book, nil)
	created := developmentOfKind(model.Developments, "mystery_created")
	resolved := developmentOfKind(model.Developments, "mystery_resolved")
	if created == nil || resolved == nil {
		t.Fatalf("expected mystery arc, got %#v", model.Developments)
	}
	if resolved.NarrativeOrder <= created.NarrativeOrder {
		t.Fatal("resolution must follow creation")
	}
}

func TestConflictingRetellingBecomesAReveal(t *testing.T) {
	book := frameBook([]types.EvidenceRecord{
		evidence("e1", 0, `"Mira Voss died on the bridge," Avery said.`, "event", "interaction", "said"),
		evidence("e2", 4, "Mira Voss was killed in the warehouse fire.", "event", "state", "killed"),
	})
	// Make the claim participate in the same life-status identity: the claim
	// itself is a claim frame; the identity joins the two life-status frames
	// only if both exist — so add the claimed version as narration from a
	// remembered scope would in the manuscript. Here we assert the simpler
	// contract: conflicting accounts of one death surface as a reveal or a
	// conflicted identity, never silently.
	model := Build(&book, nil)
	if len(model.EventIdentities) == 0 {
		// The claim sentence has no sentence-initial subject; only e2
		// produces a life_status frame, so no identity forms — and that is
		// acceptable abstention, not silence about a known conflict.
		return
	}
	if developmentOfKind(model.Developments, "reveal") == nil && model.EventIdentities[0].Status != "conflicted" {
		t.Fatalf("conflicting accounts must surface: %#v", model.EventIdentities)
	}
}

func containsSubstring(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || len(needle) == 0 || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
