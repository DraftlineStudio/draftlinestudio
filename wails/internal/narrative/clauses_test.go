package narrative

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/jdkato/prose/v3"
)

var clausePeople = []Person{{"rhea", []string{"Rhea"}}, {"tomas", []string{"Tomas"}}}

func clauseFacts(t *testing.T, text string) (Document, Extraction) {
	t.Helper()
	d := document(text)
	e, err := ExtractPresence(d, clausePeople)
	if err != nil {
		t.Fatal(err)
	}
	return d, e
}

func TestClauseArgumentCombinations(t *testing.T) {
	for _, c := range []struct {
		name, text string
		want       []string
	}{
		{"fronted-and-coordinated", "In the conservatory, Rhea and Tomas waited.", []string{"present/rhea/conservatory", "present/tomas/conservatory"}},
		{"companion-and-relative-location", "Rhea waited beside Tomas beneath the balcony.", []string{"present/rhea/beneath the balcony", "present/tomas/beneath the balcony"}},
		{"explicit-exception", "Tomas was alone inside the hangar, except for Rhea.", []string{"present/rhea/hangar", "present/tomas/hangar"}},
		{"plural-not-individually-alone", "Rhea and Tomas stood alone in the hall.", []string{"present/rhea/hall", "present/tomas/hall"}},
		{"exclusive-subject", "Only Rhea remained in the observatory.", []string{"present/rhea/observatory", "alone/rhea/observatory"}},
		{"unknown-exception", "Rhea sat alone in the library, except for a porter.", []string{"present/rhea/library"}},
		{"relative-clause-not-an-exit", "Rhea sat at the desk that Tomas left in the office.", []string{"present/rhea/desk"}},
		{"possessor-not-occupant", "Rhea stood in Tomas’s office.", []string{"present/rhea/Tomas’s office"}},
		{"adjective-list", "Standing in the dim, ancient library, Rhea waited.", []string{"present/rhea/dim, ancient library"}},
		{"qualified-adjective-list", "In the dim, ancient library of a similarly dim, ancient town, Rhea waited.", []string{"present/rhea/dim, ancient library of a similarly dim, ancient town"}},
		{"degree-modifier", "Rhea stood in the very quiet library.", []string{"present/rhea/very quiet library"}},
		{"comma-named-clause-boundary", "Rhea stood in the room, angry Tomas waited outside.", []string{"present/rhea/room"}},
		{"comma-common-clause-boundary", "Rhea stood in the room, angry strangers waited outside.", []string{"present/rhea/room"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			d, e := clauseFacts(t, c.text)
			got := []string{}
			for _, o := range e.Observations {
				got = append(got, o.Kind+"/"+o.Subject+"/"+o.Place)
				if o.PredicateSource == nil || o.PlaceSource == nil {
					t.Fatal("missing predicate/location binding")
				}
				for _, a := range []Anchor{o.Source, o.SubjectSource, *o.PredicateSource, *o.PlaceSource} {
					if _, err := d.Position(a); err != nil {
						t.Fatal(err)
					}
				}
			}
			sort.Strings(got)
			sort.Strings(c.want)
			if !reflect.DeepEqual(got, c.want) {
				tagged, err := prose.NewDocument(c.text, prose.WithExtraction(false), prose.WithSegmentation(false))
				if err == nil {
					t.Logf("tokens: %+v", tagged.Tokens())
				}
				t.Fatalf("want %v, got %v", c.want, got)
			}
		})
	}
}

func TestNonphysicalAndNegatedPredicatesDoNotCreatePresence(t *testing.T) {
	for _, text := range []string{
		"Rhea stood in silence.", "Rhea was in the mood to sing.",
		"Rhea was not in the library.", "Rhea left no room for doubt.",
		"Rhea’s portrait stood in the library.", "Rhea thought Tomas stood in the hall.",
		"Rhea planned to enter the library.", "Rhea walked toward the library.",
	} {
		t.Run(text, func(t *testing.T) {
			_, e := clauseFacts(t, text)
			if len(e.Observations) != 0 {
				t.Fatalf("unsupported physical state: %+v", e.Observations)
			}
		})
	}
}

func TestPronounBindingIsLocalAndEvidenceBacked(t *testing.T) {
	d, e := clauseFacts(t, "Rhea rose. She left the observatory.")
	if len(e.Observations) != 1 {
		t.Fatalf("expected one departure, got %+v", e)
	}
	o := e.Observations[0]
	if o.Subject != "rhea" || o.Kind != "departed" || o.SubjectSource.Quote != "She" || o.BindingStatus != "inferred_local_subject" {
		t.Fatalf("bad binding %+v", o)
	}
	if len(o.Antecedents) != 1 || o.Antecedents[0].Quote != "Rhea rose." {
		t.Fatalf("missing antecedent clause: %+v", o.Antecedents)
	}
	if _, err := InspectPresence(d, e); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"Rhea greeted Tomas. She left the observatory.",
		"Rhea rose. Tomas blinked. She left the observatory.",
		"Rhea rose. A stranger smiled. She left the observatory.",
	} {
		_, e := clauseFacts(t, text)
		for _, o := range e.Observations {
			if o.Kind == "departed" {
				t.Fatalf("ambiguous pronoun guessed in %q: %+v", text, o)
			}
		}
	}
	for _, splitScene := range []bool{false, true} {
		d := document("Rhea rose.", "She left the observatory.")
		if splitScene {
			d.Blocks[1].Scene = "second"
			d = NewDocument(d.Blocks)
		}
		e, err := ExtractPresence(d, clausePeople)
		if err != nil {
			t.Fatal(err)
		}
		if len(e.Observations) != 0 {
			t.Fatal("pronoun shortcut crossed a block/scope boundary")
		}
	}
}

func TestPredicateAndAntecedentAnchorsCannotBeForged(t *testing.T) {
	_, motion := clauseFacts(t, "Rhea slipped out of the observatory.")
	if len(motion.Observations) != 1 || motion.Observations[0].PredicateSource.Quote != "slipped out of" || motion.Observations[0].SpatialRelation != "from" {
		t.Fatalf("lost departure construction: %+v", motion)
	}
	d, e := clauseFacts(t, "Rhea rose. She left the observatory.")
	e.Observations[0].PredicateSource.Quote = "entered"
	if _, err := InspectPresence(d, e); err == nil {
		t.Fatal("forged predicate accepted")
	}
	d, e = clauseFacts(t, "Rhea rose. She left the observatory.")
	e.Observations[0].Antecedents = []Anchor{e.Observations[0].Source}
	if _, err := InspectPresence(d, e); err == nil {
		t.Fatal("future/self antecedent accepted")
	}
}

func TestLocationIsNotBorrowedFromAnotherClause(t *testing.T) {
	_, e := clauseFacts(t, "Rhea stood near the door, but Tomas sat in the kitchen.")
	if len(e.Observations) == 0 {
		t.Fatal("lost supported first clause")
	}
	for _, o := range e.Observations {
		if o.Subject == "rhea" && strings.Contains(o.Place, "kitchen") {
			t.Fatal("borrowed second clause location")
		}
	}
	if len(e.UnparsedBlocks) != 1 {
		t.Fatal("unsupported continuation lost from coverage")
	}
}

func TestDescriptiveLocationsSurviveRenaming(t *testing.T) {
	for _, name := range []string{"Mara", "Orin", "Élodie"} {
		for _, place := range []string{"library", "hangar", "courtyard"} {
			text := "Standing in the dim, ancient " + place + ", " + name + " waited."
			d := document(text)
			e, err := ExtractPresence(d, []Person{{ID: "actor", Names: []string{name}}})
			if err != nil {
				t.Fatal(err)
			}
			if len(e.Observations) != 1 {
				t.Fatalf("%q: expected one observation, got %+v", text, e)
			}
			o := e.Observations[0]
			if o.Subject != "actor" || o.Place != "dim, ancient "+place || o.PlaceSource == nil || o.PlaceSource.Quote != "in the dim, ancient "+place || o.SubjectSource.Quote != name {
				t.Fatalf("%q: incorrect field binding %+v", text, o)
			}
			if _, err := InspectPresence(d, e); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// A future discourse resolver must justify the origin and actor links. The
// last mentioned location, a route through a door, and an unresolved group
// pronoun cannot by themselves establish who departed which room.
func TestGroupMotionDoesNotBorrowLastMentionedPlace(t *testing.T) {
	for _, blocks := range [][]string{
		{"Tomas stood in the garage.", "Rhea led them out the side exit."},
		{"Rhea remembered the garage.", "Rhea led them out the side exit."},
		{"Rhea stood in the garage.", "Tomas led them out the side exit."},
	} {
		d := document(blocks...)
		e, err := ExtractPresence(d, clausePeople)
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range e.Observations {
			if o.Kind == "departed" {
				t.Fatalf("invented origin/participant link for %v: %+v", blocks, o)
			}
		}
	}
}
