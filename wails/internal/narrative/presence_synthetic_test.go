package narrative

import (
	"encoding/json"
	"strings"
	"testing"
)

var testPeople = []Person{{"elias", []string{"Elias", "Elias Vale"}}, {"mara", []string{"Mara"}}, {"tomas", []string{"Tomas"}}, {"nadia", []string{"Nadia"}}}

func document(texts ...string) Document {
	blocks := []Block{}
	for i, text := range texts {
		blocks = append(blocks, Block{ID: id("block", string(rune('a'+i))), Scene: "room-scene", Text: text})
	}
	return NewDocument(blocks)
}

func inspect(t *testing.T, d Document) (Extraction, []Inspection) {
	t.Helper()
	e, err := ExtractPresence(d, testPeople)
	if err != nil {
		t.Fatal(err)
	}
	w, err := InspectPresence(d, e)
	if err != nil {
		t.Fatal(err)
	}
	return e, w
}

func TestMaraWitnessesAndResultativeLeft(t *testing.T) {
	d := document("Mara sat with Elias at the interrogation table.", "Tomas entered the room. Tomas left the room.", "That left Elias alone with his thoughts.")
	e, w := inspect(t, d)
	if len(w) != 1 || w[0].Subject != "mara" || w[0].Kind != "unclosed_presence" {
		t.Fatalf("wrong finding: %+v", w)
	}
	if len(w[0].Sources) != 2 || w[0].Sources[0].Quote != d.Blocks[0].Text || w[0].Sources[1].Quote != d.Blocks[2].Text {
		t.Fatalf("both witnesses required: %+v", w)
	}
	for _, o := range e.Observations {
		if o.Subject == "elias" && o.Kind == "departed" {
			t.Fatal("resultative left became a departure")
		}
	}
	if w[0].Coverage != "bounded_grammar_not_proof_of_absence" {
		t.Fatal("absence must remain qualified")
	}
}

func TestPresenceControls(t *testing.T) {
	for _, tc := range []struct {
		name   string
		middle string
		split  bool
	}{
		{"explicit exit", "Mara left the interrogation room.", false},
		{"local room reference", "Mara left the room.", false},
		{"scene break", "Time passed.", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := document("Mara sat with Elias in the interrogation room.", tc.middle, "Elias sat alone in the interrogation room.")
			if tc.split {
				d.Blocks[2].Scene = "another-scene"
				d = NewDocument(d.Blocks)
			}
			_, w := inspect(t, d)
			if len(w) != 0 {
				t.Fatalf("false warning %+v", w)
			}
		})
	}
}

func TestUnknownTransitionIsExposedNotCalledImpossible(t *testing.T) {
	d := document("Mara sat with Elias in the interrogation room.", "He slipped away unnoticed.", "Elias sat alone in the interrogation room.")
	_, w := inspect(t, d)
	if len(w) != 1 || len(w[0].UnparsedBlocks) != 1 || w[0].UnparsedBlocks[0] != d.Blocks[1].ID {
		t.Fatalf("missing coverage gap %+v", w)
	}
}

func TestLeavingDifferentPlaceDoesNotClearPresence(t *testing.T) {
	_, w := inspect(t, document("Mara sat with Elias in the interrogation room.", "Mara left the kitchen.", "Elias sat alone in the interrogation room."))
	if len(w) != 1 {
		t.Fatalf("unrelated exit cleared occupancy: %+v", w)
	}
}

func TestAmbiguousAliasAbstains(t *testing.T) {
	d := document("Mara sat with Elias in the interrogation room.")
	people := append(append([]Person(nil), testPeople...), Person{"other-mara", []string{"Mara"}})
	e, err := ExtractPresence(d, people)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Observations) != 0 {
		t.Fatalf("ambiguous alias used: %+v", e)
	}
}

func TestUnsupportedScopesDoNotEstablishPresence(t *testing.T) {
	for _, source := range []string{
		`“Mara sat with Elias in the interrogation room,” Tomas said.`,
		"Elias remembered that Mara sat in the interrogation room.",
		"Mara stood in the interrogation room yesterday.",
		"If Mara stood in the interrogation room, Elias would see him.",
	} {
		t.Run(source, func(t *testing.T) {
			e, w := inspect(t, document(source, "Elias sat alone in the interrogation room."))
			for _, o := range e.Observations {
				if o.Subject == "mara" {
					t.Fatal("non-current claim became occupancy")
				}
			}
			if len(w) != 0 {
				t.Fatal(w)
			}
		})
	}
}

// These are extractor exclusion tests, NOT claims that the other semantic
// categories have been solved. They must stay out until a capable extractor
// with role/scope provenance is implemented and evaluated.
func TestSyntheticUnsupportedClaimsAreNotPromoted(t *testing.T) {
	for _, text := range []string{
		"A neon sign bled purple reflections onto wet tiles.",
		"Elias considered the possibility that everyone distrusted his account.",
		"A record alleged that Elias died during a voyage in 1972.",
	} {
		e, _ := inspect(t, document(text))
		if len(e.Observations) != 0 {
			t.Fatal("unsupported claim became occupancy")
		}
	}
}

func TestSamePassageAloneWithMara(t *testing.T) {
	// Synthetic lowercase alias with contradictory exclusivity.
	d := document("Elias sat alone in the workshop with mara.")
	_, w := inspect(t, d)
	if len(w) != 1 || w[0].Subject != "mara" || w[0].Kind != "conflicting_copresence" {
		t.Fatalf("expected immediate conflict, not invented missing exit: %+v", w)
	}
}

func TestAnchorsRejectStaleAndAlteredEvidence(t *testing.T) {
	d := document("Mara entered the room.")
	e, _ := inspect(t, d)
	e.Observations[0].Source.Quote = "Mara left the room."
	if _, err := InspectPresence(d, e); err == nil {
		t.Fatal("forged quote accepted")
	}
	d2 := document("Mara entered the kitchen.")
	if _, err := d2.Position(d.Anchor(0, 0, len(d.Blocks[0].Text))); err == nil {
		t.Fatal("stale anchor accepted")
	}
}

func TestUTF8OffsetsAndDeterminism(t *testing.T) {
	d := document("Rain fell. Mara entered the room.", "Elias stood alone in the room.")
	e, w := inspect(t, d)
	for _, o := range e.Observations {
		if _, err := d.Position(o.Source); err != nil {
			t.Fatal(err)
		}
	}
	first, _ := json.Marshal(w)
	_, again := inspect(t, d)
	second, _ := json.Marshal(again)
	if string(first) != string(second) {
		t.Fatal("nondeterministic inspections")
	}
	u := document("Élodie entered the room.")
	ue, err := ExtractPresence(u, []Person{{"elodie", []string{"Élodie"}}})
	if err != nil || len(ue.Observations) != 1 {
		t.Fatalf("unicode name: %+v %v", ue, err)
	}
	if !strings.HasPrefix(ue.Observations[0].SubjectSource.Quote, "É") {
		t.Fatal("broken UTF-8 subject span")
	}
}
