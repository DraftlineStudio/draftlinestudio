package fingerprint

// Hard acceptance tests for the v5 manuscript-memory engine (spec item 9):
// a synthetic manuscript with planted contradictions runs through the FULL
// pipeline; the inspections must catch each plant, the frames must be
// well-formed, and deliberate simulation divergence must never be flattened
// into an ordinary continuity error. All fixtures are synthetic — no
// manuscript-specific names, literals, or thresholds live in production code.

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

// plantedBook builds the acceptance manuscript. Chapter indexes matter:
// context inference is per chapter, and the simulation chapter carries its
// own scope.
func plantedBook() types.BookData {
	records := []types.EvidenceRecord{
		// ch0 — setup: possession, route north, countdown at three days.
		evidence("p1", 0, "Avery picked up the knife from the counter.", "event", "interaction", "picked"),
		evidence("p2", 0, "Avery drove north to Millbrook.", "event", "transition", "drove"),
		evidence("p3", 0, "Avery realized that three days remained until the exchange.", "fact", "discovery", "realized"),
		// ch1 — knowledge before acquisition: Mira already knows the code...
		evidence("p4", 1, "Mira knew that the vault code had changed.", "fact", "knowledge_change", "knew"),
		// ch2 — inventory conflict: the knife re-appears in other hands.
		evidence("p5", 2, "Mira picked up the knife from the table.", "event", "interaction", "picked"),
		// ch2 — route direction contradiction: same trip, opposite heading.
		evidence("p6", 2, "Avery drove south to Millbrook.", "event", "transition", "drove"),
		// ch3 — countdown reversal: five days after three days.
		evidence("p7", 3, "Avery realized that five days remained until the exchange.", "fact", "discovery", "realized"),
		// ch3 — ...and only now learns the code she knew in ch1.
		evidence("p8", 3, "Mira learned that the vault code had changed.", "fact", "discovery", "learned"),
		// ch4 — repeated event, told differently (same scope).
		evidence("p9", 4, "Avery found the tunnel beneath the rail yard.", "event", "discovery", "found"),
		evidence("p10", 4, "Avery found the tunnel behind the flood wall, according to the report.", "event", "discovery", "found"),
		// ch5 — simulation scope: Avery dies inside the simulation.
		evidence("p11", 5, "The simulation rendered the plaza in perfect detail.", "fact", "state", "rendered"),
		evidence("p12", 5, "Avery Cole was killed when the plaza collapsed.", "event", "state", "killed"),
		// ch6 — primary reality: Avery is alive. Divergence, not an error.
		evidence("p13", 6, "Avery was alive and unhurt at the checkpoint.", "fact", "state", "was"),
	}
	return frameBook(records)
}

func inspectionsOfKind(inspections []types.NarrativeInspection, kind string) []types.NarrativeInspection {
	result := []types.NarrativeInspection{}
	for _, inspection := range inspections {
		if inspection.Kind == kind {
			result = append(result, inspection)
		}
	}
	return result
}

func TestAcceptancePlantedContradictionsAreCaught(t *testing.T) {
	book := plantedBook()
	model := Build(&book, nil)

	if hits := inspectionsOfKind(model.Inspections, "impossible_possession"); len(hits) == 0 {
		t.Fatal("planted inventory conflict (knife changes hands without hand-off) was not caught")
	} else if hits[0].Severity != "error" {
		t.Fatalf("inventory conflict severity: %#v", hits[0])
	}

	routeCaught := false
	for _, inspection := range inspectionsOfKind(model.Inspections, "contradictory_state") {
		if strings.Contains(inspection.Title, "direction") || strings.Contains(inspection.Detail, "north") {
			routeCaught = true
		}
	}
	if !routeCaught {
		t.Fatal("planted route-direction contradiction (north vs south to the same place) was not caught")
	}

	if hits := inspectionsOfKind(model.Inspections, "timeline_conflict"); len(hits) == 0 {
		t.Fatal("planted countdown reversal (three days, then five days remaining) was not caught")
	}

	if hits := inspectionsOfKind(model.Inspections, "knowledge_before_acquisition"); len(hits) == 0 {
		t.Fatal("planted knowledge-before-acquisition (knows the code before learning it) was not caught")
	}

	if hits := inspectionsOfKind(model.Inspections, "repeated_event_conflict"); len(hits) == 0 {
		t.Fatal("planted repeated-event difference (tunnel found in two places) was not caught")
	}
}

func TestAcceptanceSimulationDivergenceIsNotAnError(t *testing.T) {
	book := plantedBook()
	model := Build(&book, nil)

	// The simulation death vs primary-reality survival must surface as
	// cross-scope divergence (informational), never as a same-scope error.
	for _, inspection := range model.Inspections {
		mentionsLife := false
		for _, side := range inspection.Sides {
			for _, span := range side.EvidenceSpans {
				if span.EvidenceID == "p12" || span.EvidenceID == "p13" {
					mentionsLife = true
				}
			}
		}
		if !mentionsLife {
			continue
		}
		if inspection.Severity == "error" {
			t.Fatalf("simulation divergence flattened into an error: %#v", inspection)
		}
		if inspection.ScopeAssessment == "same_scope_likely_error" {
			t.Fatalf("simulation divergence misjudged as same-scope: %#v", inspection)
		}
	}

	// The simulation chapter's frames must actually carry a non-current scope.
	simScoped := false
	for _, frame := range model.Frames {
		for _, id := range frame.EvidenceIDs {
			if id == "p12" && frame.Scope.Kind == "simulation" {
				simScoped = true
			}
		}
	}
	if !simScoped {
		t.Fatal("the simulation chapter's frames did not carry the simulation scope")
	}
}

func TestAcceptanceFrameWellFormedness(t *testing.T) {
	book := plantedBook()
	model := Build(&book, nil)
	byID := map[string]types.EvidenceRecord{}
	for _, record := range book.Analysis.Evidence.Records {
		byID[record.ID] = record
	}
	validTypes := map[string]bool{
		types.FrameEvent: true, types.FrameLocation: true, types.FramePossession: true,
		types.FrameTransfer: true, types.FrameInjury: true, types.FrameLifeStatus: true,
		types.FrameKnowledge: true, types.FrameBelief: true, types.FrameClaim: true,
		types.FrameGoal: true, types.FrameDecision: true, types.FrameObligation: true,
		types.FrameRelationship: true, types.FrameTime: true, types.FrameIdentity: true,
		types.FrameAccess: true, types.FrameCausal: true,
	}
	for _, frame := range model.Frames {
		if !validTypes[frame.Type] {
			t.Fatalf("frame outside the constrained vocabulary: %q", frame.Type)
		}
		if len(frame.EvidenceIDs) == 0 || len(frame.EvidenceSpans) == 0 {
			t.Fatalf("unsourced frame: %#v", frame)
		}
		if frame.Detail != "" && !strings.Contains(byID[frame.EvidenceIDs[0]].Text, frame.Detail) {
			t.Fatalf("non-verbatim Detail %q for %q", frame.Detail, byID[frame.EvidenceIDs[0]].Text)
		}
		// A subject participant, when present, must be a canonical name.
		for _, p := range frame.Participants {
			if p.Role == "subject" && p.EntityName != "Avery Cole" && p.EntityName != "Mira Voss" && p.EntityName != "Dane" {
				t.Fatalf("non-canonical subject %q", p.EntityName)
			}
		}
	}
	if model.CorpusStats.Frames != len(model.Frames) || model.CorpusStats.Inspections != len(model.Inspections) {
		t.Fatal("corpus stats out of sync")
	}
}

func TestAcceptanceReportsAreReadableAndComplete(t *testing.T) {
	book := plantedBook()
	model := Build(&book, nil)
	if !strings.Contains(model.FrameDiagnostic, "MANUSCRIPT MEMORY — FRAMES") ||
		!strings.Contains(model.DevelopmentDiagnostic, "NARRATIVE DEVELOPMENTS") ||
		!strings.Contains(model.InspectionDiagnostic, "CONTINUITY INSPECTIONS") {
		t.Fatal("the three text diagnostics must render")
	}
	if !strings.Contains(model.InspectionDiagnostic, "Scope assessment") {
		t.Fatal("inspection report must show scope assessments")
	}
	for _, inspection := range model.Inspections {
		if len(inspection.Sides) == 0 {
			t.Fatalf("inspection without sides: %#v", inspection)
		}
		for _, side := range inspection.Sides {
			if len(side.EvidenceSpans) == 0 {
				t.Fatalf("inspection side without evidence: %#v", inspection)
			}
		}
	}
}
