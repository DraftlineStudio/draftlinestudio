package narrative

import "testing"

func TestPassiveMovementKeepsRolesAndOriginsSeparate(t *testing.T) {
	for _, text := range []string{
		"Rhea was led from the hall.",
		"Rhea was escorted out of the hall.",
		"Rhea let herself be guided from the hall.",
		"Rhea was led from the hall by Tomas.",
	} {
		d, e := movementExtraction(t, text)
		if len(e.Movements) != 1 {
			t.Fatalf("missing %q", text)
		}
		m := e.Movements[0]
		leader, follower := m.Participants[0], m.Participants[1]
		if follower.Subject != "rhea" || follower.Origin == nil || follower.Origin.Place != "hall" || leader.Origin != nil {
			t.Fatalf("incorrect passive roles: %+v", m)
		}
		if leader.Subject == "" && (leader.IdentityStatus != "unexpressed" || leader.Source.Quote != "led" && leader.Source.Quote != "escorted" && leader.Source.Quote != "guided") {
			t.Fatal("omitted agent presented as an explicit mention")
		}
		if leader.Subject != "" && leader.Subject != "tomas" {
			t.Fatal("invented agent identity")
		}
		for _, a := range m.PredicateSources {
			if _, err := d.Position(a); err != nil {
				t.Fatal(err)
			}
		}
	}
	_, e := movementExtraction(t, "Rhea and Tomas were led from the hall.")
	if len(e.Movements) != 1 || len(e.Movements[0].Participants) != 3 {
		t.Fatal("lost coordinated passive followers")
	}
	_, e = movementExtraction(t, "She was led from the hall.")
	if len(e.Movements) != 1 || e.Movements[0].Participants[1].Subject != "" || e.Movements[0].Participants[1].IdentityStatus != "unresolved" {
		t.Fatal("guessed pronoun identity")
	}
}

func TestMovementVoiceAndOrderRenaming(t *testing.T) {
	for _, name := range []string{"Mara", "Orin", "Élodie"} {
		for _, place := range []string{"hangar", "library", "courtyard"} {
			for _, text := range []string{
				name + " was led from the " + place + ".",
				"They ran from the " + place + ", led by " + name + ".",
				"They marched from the " + place + ", three in a line, guided by " + name + ".",
			} {
				d := document(text)
				e, err := ExtractPresence(d, []Person{{ID: "actor", Names: []string{name}}})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := InspectPresence(d, e); err != nil {
					t.Fatal(err)
				}
				if len(e.Movements) != 1 {
					t.Fatalf("missing %q", text)
				}
				found := false
				for _, p := range e.Movements[0].Participants {
					if p.Subject == "actor" {
						found = true
						if p.Source.Quote != name || p.Origin == nil || p.Origin.Place != place {
							t.Fatalf("incorrect renamed binding: %+v", p)
						}
					}
				}
				if !found {
					t.Fatal("lost explicit identity")
				}
			}
		}
	}
}

func TestMovementVoiceAndOrderDoNotMineOtherClauses(t *testing.T) {
	for _, text := range []string{
		"Rhea was not led from the hall.",
		"Rhea wanted to be guided from the hall.",
		"Rhea imagined Tomas was led from the hall.",
		"Rhea said Tomas was led from the hall.",
		"Rhea was led from despair.",
		"Rhea’s portrait was led from the hall.",
		"They ran from the hall, Tomas watched, led by Rhea.",
		"They ran from the hall, three people watched, led by Rhea.",
		"They ran from the hall, followed a sign led by Rhea.",
		"They let herself be guided from the hall.",
	} {
		_, e := movementExtraction(t, text)
		if len(e.Movements) != 0 {
			t.Fatalf("unsupported candidate %q: %+v", text, e.Movements)
		}
	}
}

func TestUnexpressedAgentCannotAcquireIdentityOrOrigin(t *testing.T) {
	for _, mutate := range []func(*MovementParticipant){
		func(p *MovementParticipant) { p.Subject = "tomas" },
		func(p *MovementParticipant) {
			p.Origin = &MovementOrigin{Place: "hall", Status: "explicit", Source: &p.Source}
		},
		func(p *MovementParticipant) { p.IdentityStatus = "explicit" },
	} {
		d, e := movementExtraction(t, "Rhea was led from the hall.")
		mutate(&e.Movements[0].Participants[0])
		if _, err := InspectPresence(d, e); err == nil {
			t.Fatal("fabricated passive-agent evidence accepted")
		}
	}
}
