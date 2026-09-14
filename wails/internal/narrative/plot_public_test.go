package narrative

import (
	"strings"
	"testing"
)

func TestPlotActivityProjectionIsIndependentOfMovement(t *testing.T) {
	d := document(`“We have a disturbance at 24 West Maple.”`, `“I’ll take it.”`, `“We have a fire at 18 North Pine.”`, `“Responding on foot. Unit B is still handling the disturbance call at 24 W Maple.”`, `“Requesting a thorough search of the building.”`, `They found nothing.`)
	p, err := AnalyzePlot(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Graph.Threads) != 3 || len(p.Graph.Relations) != 3 {
		t.Fatalf("lost candidate strands: %+v", p.Graph)
	}
	for _, e := range p.Graph.Relations {
		if !strings.HasPrefix(e.Rule, "candidate_") {
			t.Fatal("unqualified candidate edge")
		}
	}
	for _, thread := range p.Graph.Threads {
		if thread.Status != "outcome_not_observed" {
			t.Fatal("invented completion")
		}
	}
	last := p.Graph.Nodes[len(p.Graph.Nodes)-1]
	if len(last.Changes) != 0 {
		t.Fatal("unknown search participants attached by recency")
	}
	for _, o := range p.Graph.Observations {
		if o.Subject != "" {
			t.Fatal("unattributed statement assigned to character")
		}
	}
}

func TestPlotDoesNotUseUnscopedCommitmentsOrAmbiguousAddress(t *testing.T) {
	d := document(`“I’ll take it.”`, `“We have a disturbance at 24 West Maple.”`, `“I’ll take it.”`, `“We have a disturbance at 24 West Maple.”`, `“I’ll take it.”`, `“Unit B is still handling the disturbance call at 24 W Maple.”`, `If they found nothing, they would leave.`, `“Requesting a word search book.”`)
	p, err := AnalyzePlot(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Graph.Threads) != 2 || len(p.Graph.Relations) != 2 {
		t.Fatalf("unsupported thread/link: %+v", p.Graph)
	}
	for _, c := range p.Claims {
		if c.Observation.Kind == "ongoing_activity_report" && c.LinkStatus != "" {
			t.Fatal("greedy ambiguous task match")
		}
		if c.Observation.Kind == "negative_finding" {
			t.Fatal("hypothetical outcome asserted")
		}
	}
}

func TestPlotAndAttributeHTMLKeepsContextAndEscapes(t *testing.T) {
	d := document(`A blue keypad blinked. The light glowed white.`, `The LED changed from white to red.`, `“I'm forty-two.”`, `Rhea was 33 years old. Rhea was born in 1991.`, `“Requesting a search of <script>alert(1)</script>.”`)
	p, err := AnalyzePlot(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Attributes) != 6 {
		t.Fatalf("attribute mentions: %+v", p.Attributes)
	}
	for _, a := range p.Attributes {
		if a.AssertionStatus != "mention_only_requires_context_interpretation" {
			t.Fatal("mention promoted to fact")
		}
		for _, source := range []Anchor{a.Source, a.SubjectSource, a.ValueSource, a.ContextSource} {
			if _, err := d.Position(source); err != nil {
				t.Fatal(err)
			}
		}
	}
	page, err := RenderPlotHTML(d, p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(page, "<script>") || !strings.Contains(page, "&lt;script&gt;") {
		t.Fatal("HTML injection or lost source")
	}
	p.Attributes[0].Source.Quote = "forged"
	if _, err := RenderPlotHTML(d, p); err == nil {
		t.Fatal("forged attribute source accepted")
	}
}
