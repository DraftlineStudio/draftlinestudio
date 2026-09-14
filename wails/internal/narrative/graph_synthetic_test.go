package narrative

import (
	"reflect"
	"testing"
)

func graphObservation(d Document, index int, kind string) Observation {
	return Observation{ID: id("observation", d.Blocks[index].ID), Kind: kind, Mode: "current_narration", Source: d.Anchor(index, 0, len(d.Blocks[index].Text)), Rule: "test-supplied-semantic-observation"}
}

func graphNode(o Observation, activity, operation string) Development {
	return Development{ID: id("development", o.ID), Label: o.Source.Quote, EstablishedBy: o.ID, Changes: []ActivityChange{{ActivityID: activity, Operation: operation, Witness: o.ID}}}
}

// This tests the reduction contract with supplied semantic observations. It is
// deliberately NOT an end-to-end extraction test or a manuscript acceptance gate.
func TestGraphKeepsTwoActivitiesThroughConvergence(t *testing.T) {
	d := document("Nadia accepted the trespass call.", "Elias accepted the jumper call.", "Nadia and Elias agreed to continue both the trespass and jumper investigations together.", "They searched the structure.")
	obs := []Observation{}
	for i := range d.Blocks {
		obs = append(obs, graphObservation(d, i, "activity"))
	}
	a, b := graphNode(obs[0], "trespass-1", "initiate"), graphNode(obs[1], "jumper-1", "initiate")
	join := graphNode(obs[2], "jumper-1", "continue")
	join.Changes = append(join.Changes, ActivityChange{ActivityID: "trespass-1", Operation: "continue", Witness: obs[2].ID})
	search := graphNode(obs[3], "search-1", "initiate")
	relations := []Relation{
		{ID: "a-join", From: a.ID, To: join.ID, Kind: "joins", Witnesses: []string{obs[2].ID}, Rule: "test-supported-reunion"},
		{ID: "b-join", From: b.ID, To: join.ID, Kind: "joins", Witnesses: []string{obs[2].ID}, Rule: "test-supported-reunion"},
		{ID: "join-search", From: join.ID, To: search.ID, Kind: "branches", Witnesses: []string{obs[3].ID}, Rule: "test-explicit-search"},
	}
	g, err := BuildGraph(d, obs, []Development{search, join, b, a}, relations)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Threads) != 3 || len(g.Relations) != 3 {
		t.Fatalf("lost braid structure: %+v", g)
	}
	if g.Nodes[0].ID != a.ID || g.Nodes[3].ID != search.ID {
		t.Fatal("not manuscript order")
	}
	for _, thread := range g.Threads {
		if thread.Status != "outcome_not_observed" {
			t.Fatal("reunion manufactured completion")
		}
	}
	if len(g.Threads[0].NodeIDs) != 2 || len(g.Threads[1].NodeIDs) != 2 {
		t.Fatal("shared node lost a membership")
	}
}

func TestLaterDevelopmentStaysAtEstablishingPassage(t *testing.T) {
	d := document("The key was in her pocket.", "She used the key to open the door.")
	a, b := graphObservation(d, 0, "possession"), graphObservation(d, 1, "use")
	n := Development{ID: "open-door", Label: "Door opened", EstablishedBy: b.ID, SupportingObservations: []string{a.ID}}
	g, err := BuildGraph(d, []Observation{a, b}, []Development{n}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if g.Nodes[0].EstablishedBy != b.ID {
		t.Fatal("payoff moved to setup")
	}
	if len(g.Threads) != 0 || len(g.Relations) != 0 {
		t.Fatal("support was promoted into automatic plot relation")
	}
	n.EstablishedBy = a.ID
	n.SupportingObservations = []string{b.ID}
	if _, err := BuildGraph(d, []Observation{a, b}, []Development{n}, nil); err == nil {
		t.Fatal("future evidence moved development backward")
	}
}

func TestRepeatedWordsAndActorsDoNotCreateRelations(t *testing.T) {
	d := document("Nadia walked toward a greenhouse.", "Nadia walked toward a different building.")
	a, b := graphObservation(d, 0, "motion"), graphObservation(d, 1, "motion")
	a.Subject = "nadia"
	b.Subject = "nadia"
	nodes := []Development{{ID: "walk-1", Label: a.Source.Quote, EstablishedBy: a.ID}, {ID: "walk-2", Label: b.Source.Quote, EstablishedBy: b.ID}}
	g, err := BuildGraph(d, []Observation{a, b}, nodes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Nodes) != 2 || len(g.Relations) != 0 || len(g.Threads) != 0 {
		t.Fatal("lexical recurrence manufactured identity or thread")
	}
}

func TestReportedHistoryPreservesDisclosurePosition(t *testing.T) {
	d := document("Elias entered the room.", "A journal claimed that Elias Vale died at sea in 1972.")
	a, b := graphObservation(d, 0, "presence"), graphObservation(d, 1, "death-claim")
	b.Mode = "attributed_report"
	nodes := []Development{{ID: "arrival", Label: a.Source.Quote, EstablishedBy: a.ID}, {ID: "report", Label: b.Source.Quote, EstablishedBy: b.ID}}
	g, err := BuildGraph(d, []Observation{a, b}, nodes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if g.Nodes[1].ID != "report" || g.Observations[1].Mode != "attributed_report" {
		t.Fatal("report was moved or promoted to narration")
	}
}

func TestGraphRejectsUnsupportedWiring(t *testing.T) {
	d := document("Mara entered the room.", "Mara left the room.")
	a, b := graphObservation(d, 0, "arrival"), graphObservation(d, 1, "departure")
	n1, n2 := graphNode(a, "visit", "initiate"), graphNode(b, "visit", "complete")
	for _, tc := range []struct {
		name  string
		nodes []Development
		edges []Relation
	}{
		{"missing initiation", []Development{n2}, nil},
		{"missing witness", []Development{n1, n2}, []Relation{{ID: "bad", From: n1.ID, To: n2.ID, Kind: "continues", Rule: "rule"}}},
		{"unknown witness", []Development{n1, n2}, []Relation{{ID: "bad", From: n1.ID, To: n2.ID, Kind: "continues", Rule: "rule", Witnesses: []string{"absent"}}}},
		{"lexical relation", []Development{n1, n2}, []Relation{{ID: "bad", From: n1.ID, To: n2.ID, Kind: "word_overlap", Rule: "rule", Witnesses: []string{b.ID}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BuildGraph(d, []Observation{a, b}, tc.nodes, tc.edges); err == nil {
				t.Fatal("unsupported graph accepted")
			}
		})
	}
	g, err := BuildGraph(d, []Observation{a, b}, []Development{n1, n2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if g.Threads[0].Status != "completed" {
		t.Fatal("explicit completion lost")
	}
}

func TestInputBatchOrderingDoesNotChangeNodeOrThreadOrder(t *testing.T) {
	d := document("Mara entered the room.", "Mara left the room.")
	a, b := graphObservation(d, 0, "arrival"), graphObservation(d, 1, "departure")
	n1, n2 := graphNode(a, "visit", "initiate"), graphNode(b, "visit", "complete")
	x, err := BuildGraph(d, []Observation{a, b}, []Development{n1, n2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	y, err := BuildGraph(d, []Observation{b, a}, []Development{n2, n1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(x.Nodes, y.Nodes) || !reflect.DeepEqual(x.Threads, y.Threads) {
		t.Fatal("batch order changed semantic projection")
	}
}
