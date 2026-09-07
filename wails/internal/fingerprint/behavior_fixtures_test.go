package fingerprint

import (
	"testing"

	"draftline/internal/types"
)

func TestManuscriptMemoryBehavioralFixtures(t *testing.T) {
	t.Run("routine movement remains queryable memory", func(t *testing.T) {
		model := buildFixture(fixtureRecord("move", 0, "Avery walked to the car.", "walked", "transition", "avery", "Avery"))
		_ = manuscriptFingerprintForEvidence(t, model, "move")
	})

	t.Run("object acquisition later enables use", func(t *testing.T) {
		acquire := fixtureRecord("key-acquired", 0, "Avery picked up the brass key.", "picked", "state", "avery", "Avery")
		use := fixtureRecord("key-used", 1, "Avery used the brass key to unlock the archive.", "used", "interaction", "avery", "Avery")
		model := buildFixture(acquire, use)
		assertCorpusRelation(t, model, "enables")
	})

	t.Run("explicit lie remains attributed deception", func(t *testing.T) {
		lie := fixtureRecord("lie", 0, `"The bridge is safe," Mira lied.`, "lied", "interaction", "mira", "Mira")
		fingerprint := manuscriptFingerprintForEvidence(t, buildFixture(lie), "lie")
		if fingerprint.EpistemicStatus != "deliberate_deception" || fingerprint.Attribution.EntityName != "Mira" {
			t.Fatalf("lie became world truth: %#v", fingerprint)
		}
	})

	t.Run("incorrect belief can be superseded", func(t *testing.T) {
		belief := fixtureRecord("belief", 0, "Mira believed the bridge was safe.", "believed", "knowledge_state", "mira", "Mira")
		reveal := fixtureRecord("reveal", 1, "The bridge was actually unstable.", "was", "state")
		model := buildFixture(belief, reveal)
		assertCorpusRelation(t, model, "supersedes")
	})

	t.Run("deictic and existential clauses do not fabricate contradictions", func(t *testing.T) {
		first := fixtureRecord("deictic", 0, `"This is the engine room," Mira said.`, "said", "interaction", "mira", "Mira")
		second := fixtureRecord("existential", 1, `"There was no pilot before you," Oren said.`, "said", "interaction", "oren", "Oren")
		model := buildFixture(first, second)
		if hasCorpusRelation(model.FingerprintRelations, "contradicts") {
			t.Fatalf("unresolved referents created a contradiction: %#v", model.FingerprintRelations)
		}
	})

	t.Run("conditional decision is memory but not a goal", func(t *testing.T) {
		conditional := fixtureRecord("conditional", 0, "The advocate would respond if the council decided to press charges.", "decided", "state", "advocate", "Advocate")
		fingerprint := manuscriptFingerprintForEvidence(t, buildFixture(conditional), "conditional")
		if fingerprint.Kind == "goal" {
			t.Fatalf("conditional possibility became an adopted goal: %#v", fingerprint)
		}
	})

	t.Run("near injury does not establish physical injury", func(t *testing.T) {
		nearMiss := fixtureRecord("near-miss", 0, "The falling beam nearly killed Avery.", "killed", "state", "avery", "Avery")
		model := buildFixture(nearMiss)
		for _, history := range model.StateHistories {
			if history.Property == "life_status" || history.Property == "physical_condition" {
				t.Fatalf("near miss became a persistent condition: %#v", history)
			}
		}
	})

	t.Run("dream and flashback retain separate scopes", func(t *testing.T) {
		dream := fixtureRecord("dream", 0, "In a dream, Sela saw a brass symbol beside the gate.", "saw", "state", "sela", "Sela")
		past := fixtureRecord("past", 1, "Years earlier, Toma buried the compass beneath the oak.", "buried", "time_reference", "toma", "Toma")
		model := buildFixture(dream, past)
		if manuscriptFingerprintForEvidence(t, model, "dream").Scope.Kind != "dream" || manuscriptFingerprintForEvidence(t, model, "past").Scope.Kind != "flashback" {
			t.Fatalf("reality scopes were flattened: %#v", model.Fingerprints)
		}
	})

	t.Run("fulfilled and unresolved promises remain distinguishable", func(t *testing.T) {
		open := fixtureRecord("open", 0, "Avery promised to return the ledger before dawn.", "promised", "interaction", "avery", "Avery")
		fulfilled := fixtureRecord("fulfilled", 2, "Avery returned the ledger to Niko before dawn.", "returned", "interaction", "avery", "Avery", "niko", "Niko")
		model := buildFixture(open, fulfilled)
		assertCorpusRelation(t, model, "fulfills")
		for _, inspection := range model.Inspections {
			if inspection.Kind == "open_obligation" {
				t.Fatalf("fulfilled promise remained open: %#v", inspection)
			}
		}
	})

	t.Run("knowledge transfer remains an attributed claim", func(t *testing.T) {
		warning := fixtureRecord("warning", 0, `"The west gate is trapped," Mira warned Oren.`, "warned", "knowledge_transfer", "mira", "Mira", "oren", "Oren")
		warning.KnowledgeStates = []types.EvidenceKnowledgeState{{State: "shared", CharacterIDs: []string{"mira"}, CharacterNames: []string{"Mira"}, CounterpartyIDs: []string{"oren"}, CounterpartyNames: []string{"Oren"}, Cue: "warned", Confidence: .88}}
		fingerprint := manuscriptFingerprintForEvidence(t, buildFixture(warning), "warning")
		if fingerprint.EpistemicStatus != "attributed_claim" {
			t.Fatalf("transferred information became world truth: %#v", fingerprint)
		}
	})

	t.Run("quiet conversation can establish durable commitment", func(t *testing.T) {
		conversation := fixtureRecord("quiet", 0, "Mira agreed to trust Oren and continue the search.", "agreed", "interaction", "mira", "Mira", "oren", "Oren")
		if fingerprint := manuscriptFingerprintForEvidence(t, buildFixture(conversation), "quiet"); fingerprint.Kind != "commitment" {
			t.Fatalf("quiet durable change was missed: %#v", fingerprint)
		}
	})
}

func fixtureRecord(id string, chapter int, text, action, evidenceType string, characterPairs ...string) types.EvidenceRecord {
	result := types.EvidenceRecord{ID: id, Kind: "event", EvidenceType: evidenceType, ChapterID: "fixture-chapter", ChapterIndex: chapter, Section: "body", SectionIndex: chapter, ParagraphIndex: chapter, SentenceIndex: chapter, StartOffset: chapter * 100, EndOffset: chapter*100 + len(text), Text: text, Action: action, Confidence: .9, Status: "detected", Source: "auto"}
	for index := 0; index+1 < len(characterPairs); index += 2 {
		if characterPairs[index] != "" {
			result.CharacterIDs = append(result.CharacterIDs, characterPairs[index])
			result.CharacterNames = append(result.CharacterNames, characterPairs[index+1])
		}
	}
	return result
}

func buildFixture(records ...types.EvidenceRecord) *types.StoryFingerprint {
	book := testBook(records)
	return Build(&book, nil)
}

func hasCorpusRelation(values []types.ManuscriptFingerprintRelation, kind string) bool {
	for _, value := range values {
		if value.Kind == kind {
			return true
		}
	}
	return false
}

func assertCorpusRelation(t *testing.T, model *types.StoryFingerprint, kind string) {
	t.Helper()
	if !hasCorpusRelation(model.FingerprintRelations, kind) {
		t.Fatalf("missing %s relationship: %#v", kind, model.FingerprintRelations)
	}
}
