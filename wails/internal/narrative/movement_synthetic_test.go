package narrative

import (
	"testing"
)

func movementExtraction(t *testing.T, blocks ...string) (Document, Extraction) {
	t.Helper()
	d := document(blocks...)
	e, err := ExtractPresence(d, clausePeople)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InspectPresence(d, e); err != nil {
		t.Fatal(err)
	}
	return d, e
}

func TestMovementOriginsArePerParticipant(t *testing.T) {
	for _, blocks := range [][]string{
		{"Rhea stood in the garage. Rhea led them out the side exit."},
		{"Rhea stood in the garage.", "Rhea led them out the side exit."},
		{"Rhea stood in the garage.", " ", "Rhea led them out the side exit."},
	} {
		_, e := movementExtraction(t, blocks...)
		if len(e.Movements) != 1 {
			t.Fatalf("missing candidate: %+v", e)
		}
		m := e.Movements[0]
		leader, group := m.Participants[0], m.Participants[1]
		if leader.Origin == nil || leader.Origin.Place != "garage" || leader.Origin.Status != "inferred_adjacent_occupancy" || leader.Origin.Premise != e.Observations[0].ID || leader.Origin.Source != nil {
			t.Fatalf("unsupported leader binding: %+v", leader)
		}
		if group.Subject != "" || group.IdentityStatus != "unresolved" || group.Origin != nil || group.Source.Quote != "them" {
			t.Fatalf("invented group binding: %+v", group)
		}
		if m.RouteSource == nil || m.RouteSource.Quote != "the side exit" || len(m.PredicateSources) != 2 || m.PredicateSources[0].Quote != "led" || m.PredicateSources[1].Quote != "out" {
			t.Fatalf("lost route/predicate evidence: %+v", m)
		}
		for _, o := range e.Observations {
			if o.Kind == "departed" {
				t.Fatal("candidate promoted to accepted departure")
			}
		}
	}
	_, e := movementExtraction(t, "Rhea stood with Tomas in the garage.", "Rhea escorted Tomas out through the gate.")
	if len(e.Movements) != 1 {
		t.Fatal("missing named group")
	}
	for _, p := range e.Movements[0].Participants {
		if p.Origin == nil || p.Origin.Premise == "" || p.Origin.Place != "garage" {
			t.Fatalf("missing independent occupancy premise: %+v", p)
		}
	}
	if e.Movements[0].Participants[0].Origin.Premise == e.Movements[0].Participants[1].Origin.Premise {
		t.Fatal("leader premise reused for follower")
	}
}

func TestMovementDoesNotBridgeUnsupportedContext(t *testing.T) {
	for _, blocks := range [][]string{
		{"Tomas stood in the garage.", "Rhea led them out the side exit."},
		{"Rhea stood near the garage.", "Rhea led them out the side exit."},
		{"Rhea stood in the garage.", "Something happened.", "Rhea led them out the side exit."},
		{"Rhea stood in the garage, but went somewhere else.", "Rhea led them out the side exit."},
		{"Rhea remembered the garage.", "Rhea led them out the side exit."},
		{"Rhea stood in the garage.", "“This way.” Rhea led them out the side exit."},
		{"Rhea led them out the side exit.", "Rhea stood in the garage."},
	} {
		_, e := movementExtraction(t, blocks...)
		if len(e.Movements) != 1 {
			t.Fatalf("missing candidate for %v", blocks)
		}
		if e.Movements[0].Participants[0].Origin != nil {
			t.Fatalf("borrowed origin for %v", blocks)
		}
	}
	d := document("Rhea stood in the garage.", "Rhea led them out the side exit.")
	d.Blocks[1].Scene = "next"
	d = NewDocument(d.Blocks)
	e, err := ExtractPresence(d, clausePeople)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Movements) != 1 || e.Movements[0].Participants[0].Origin != nil {
		t.Fatal("origin crossed scene")
	}
}

func TestMovementRenamingAndRepeatedActionsRemainSeparate(t *testing.T) {
	for _, name := range []string{"Mara", "Orin", "Élodie"} {
		for _, place := range []string{"hangar", "library", "courtyard"} {
			d := document(name+" stood in the "+place+".", name+" led them out the side exit.", name+" stood in the "+place+".", name+" led them out the side exit.")
			e, err := ExtractPresence(d, []Person{{ID: "actor", Names: []string{name}}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := InspectPresence(d, e); err != nil {
				t.Fatal(err)
			}
			if len(e.Movements) != 2 || e.Movements[0].ID == e.Movements[1].ID {
				t.Fatalf("repeated movement merged: %+v", e.Movements)
			}
			for _, m := range e.Movements {
				p := m.Participants[0]
				if p.Subject != "actor" || p.Source.Quote != name || p.Origin == nil || p.Origin.Place != place {
					t.Fatalf("renamed binding failed: %+v", p)
				}
			}
			if e.Movements[0].Participants[0].Origin.Premise == e.Movements[1].Participants[0].Origin.Premise {
				t.Fatal("reused old origin instead of separate source chain")
			}
		}
	}
}

func TestMovementCandidateDoesNotSuppressMissingExit(t *testing.T) {
	d, e := movementExtraction(t, "Rhea stood with Tomas in the garage.", "Rhea led Tomas out the side exit.", "Rhea was alone in the garage.")
	withCandidates, err := InspectPresence(d, e)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Movements) != 1 || len(withCandidates) != 1 || withCandidates[0].Subject != "tomas" {
		t.Fatal("expected candidate and review-only missing transition")
	}
	e.Movements = nil
	withoutCandidates, err := InspectPresence(d, e)
	if err != nil || len(withoutCandidates) != len(withCandidates) {
		t.Fatal("candidate changed inspection state")
	}
}

func TestMovementExplicitOriginAndDialogueBoundaries(t *testing.T) {
	for _, text := range []string{
		"Rhea guided Tomas out of the hangar.",
		"“Go!” Rhea guided Tomas out of the hangar.",
		"\"Go!\" Rhea guided Tomas out of the hangar.",
	} {
		_, e := movementExtraction(t, text)
		if len(e.Movements) != 1 {
			t.Fatalf("missing %q", text)
		}
		m := e.Movements[0]
		if m.RouteSource != nil || m.PredicateSources[1].Quote != "out of" {
			t.Fatal("origin treated as route")
		}
		for _, p := range m.Participants {
			if p.Origin == nil || p.Origin.Status != "explicit" || p.Origin.Source.Quote != "the hangar" {
				t.Fatalf("lost explicit origin %+v", p)
			}
		}
	}
	for _, text := range []string{
		"“Rhea led them out the side exit.”",
		"“Go,” Rhea led them out the side exit.",
		"“Go! Rhea led them out the side exit.",
		"Rhea said Tomas led them out the side exit.",
		"Rhea would lead them out the side exit.",
		"Rhea never led them out the side exit.",
		"Rhea led them out of debt.",
		"Rhea led them out the side exit in her imagination.",
		"Rhea's portrait led them out the side exit.",
	} {
		_, e := movementExtraction(t, text)
		if len(e.Movements) != 0 {
			t.Fatalf("unsupported movement %q: %+v", text, e.Movements)
		}
	}
	_, e := movementExtraction(t, "“Let me tell you what happened.", "Rhea led them out the side exit.", "That was all.”", "Rhea led them out the side exit.")
	if len(e.Movements) != 0 {
		t.Fatal("unbalanced dialogue leaked into current motion")
	}
}

func TestMovementOriginEvidenceCannotBeForged(t *testing.T) {
	for _, mutate := range []func(*Extraction){
		func(e *Extraction) { e.Movements[0].Participants[0].Origin.Premise = "missing" },
		func(e *Extraction) { e.Movements[0].Participants[0].Origin.Place = "kitchen" },
		func(e *Extraction) { e.Movements[0].Participants[0].Subject = "tomas" },
		func(e *Extraction) { e.Movements[0].RouteSource.Quote = "garage" },
		func(e *Extraction) { e.Movements[0].Participants[1].Origin = e.Movements[0].Participants[0].Origin },
		func(e *Extraction) { e.Movements[0].Participants[0].Origin.Source = &e.Observations[0].Source },
	} {
		d, e := movementExtraction(t, "Rhea stood in the garage.", "Rhea led them out the side exit.")
		mutate(&e)
		if _, err := InspectPresence(d, e); err == nil {
			t.Fatal("forged evidence accepted")
		}
	}
}
