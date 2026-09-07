package fingerprint

import (
	"fmt"
	"strings"
	"testing"

	"draftline/internal/types"
)

func TestNarrativeBehavioralFixtures(t *testing.T) {
	t.Run("routine movement remains evidence", func(t *testing.T) {
		movement := fixtureRecord("move", 0, "Avery walked to the car.", "walked", "transition", "avery", "Avery")
		model := buildFixture(movement)
		assertEvidenceRetainedWithoutPromotion(t, model, "move")
	})

	t.Run("object acquisition promotes retrospectively when used", func(t *testing.T) {
		acquire := fixtureRecord("key-acquired", 0, "Avery picked up the brass key.", "picked", "state", "avery", "Avery")
		use := fixtureRecord("key-used", 1, "Avery used the brass key to unlock the archive.", "used", "interaction", "avery", "Avery")
		model := buildFixture(acquire, use)
		assertPromotedEvidence(t, model, "key-acquired")
		assertPromotedEvidence(t, model, "key-used")
		assertRelation(t, model, "enables")
	})

	t.Run("explicit lie remains an attributed deception", func(t *testing.T) {
		lie := fixtureRecord("lie", 0, `"The bridge is safe," Mira lied.`, "lied", "interaction", "mira", "Mira")
		model := buildFixture(lie)
		fingerprint := fingerprintForEvidence(t, model, "lie")
		if fingerprint.EpistemicStatus != "deliberate_deception" || fingerprint.Attribution.EntityName != "Mira" {
			t.Fatalf("lie was converted into world truth: %#v", fingerprint)
		}
	})

	t.Run("incorrect belief is superseded by later reveal", func(t *testing.T) {
		belief := fixtureRecord("belief", 0, "Mira believed the bridge was safe.", "believed", "knowledge_state", "mira", "Mira")
		reveal := fixtureRecord("reveal", 1, "The bridge was actually unstable.", "was", "state", "", "")
		model := buildFixture(belief, reveal)
		assertRelation(t, model, "supersedes")
		if fingerprintForEvidence(t, model, "belief").EpistemicStatus != "character_belief" {
			t.Fatal("belief lost its epistemic status")
		}
	})

	t.Run("contradictory accounts remain two attributed claims", func(t *testing.T) {
		first := fixtureRecord("account-a", 0, `"The vault is empty," Mira said.`, "said", "interaction", "mira", "Mira")
		second := fixtureRecord("account-b", 1, `"The vault is full," Oren said.`, "said", "interaction", "oren", "Oren")
		model := buildFixture(first, second)
		assertRelation(t, model, "contradicts")
		for _, id := range []string{"account-a", "account-b"} {
			if fingerprintForEvidence(t, model, id).EpistemicStatus != "attributed_claim" {
				t.Fatalf("%s became objective truth", id)
			}
		}
	})

	t.Run("deictic claims do not become contradictions", func(t *testing.T) {
		first := fixtureRecord("deictic-a", 0, `"This is the engine room," Mira said.`, "said", "interaction", "mira", "Mira")
		second := fixtureRecord("deictic-b", 1, `"This is the infirmary," Oren said.`, "said", "interaction", "oren", "Oren")
		model := buildFixture(first, second)
		if hasNarrativeRelation(model.NarrativeRelations, "contradicts") || len(model.NarrativeFingerprints) != 0 {
			t.Fatalf("unresolved deictic subjects created a false contradiction: %#v", model.NarrativeRelations)
		}
	})

	t.Run("existential claims do not share a fabricated subject", func(t *testing.T) {
		first := fixtureRecord("existential-a", 0, `"There is a sealed tunnel beneath the river," Mira said.`, "said", "interaction", "mira", "Mira")
		second := fixtureRecord("existential-b", 1, `"There was no pilot before you," Oren said.`, "said", "interaction", "oren", "Oren")
		model := buildFixture(first, second)
		if hasNarrativeRelation(model.NarrativeRelations, "contradicts") || len(model.NarrativeFingerprints) != 0 {
			t.Fatalf("existential clauses created a fabricated shared subject: %#v", model.NarrativeRelations)
		}
	})

	t.Run("conditional decision is not a character goal", func(t *testing.T) {
		conditional := fixtureRecord("conditional", 0, "The advocate would respond if the council decided to press charges.", "decided", "state", "advocate", "Advocate")
		model := buildFixture(conditional)
		assertEvidenceRetainedWithoutPromotion(t, model, "conditional")
	})

	t.Run("near injury does not establish injury", func(t *testing.T) {
		nearMiss := fixtureRecord("near-miss", 0, "The falling beam nearly killed Avery.", "killed", "state", "avery", "Avery")
		model := buildFixture(nearMiss)
		assertEvidenceRetainedWithoutPromotion(t, model, "near-miss")
	})

	t.Run("dream information retains scope when later useful", func(t *testing.T) {
		dream := fixtureRecord("dream", 0, "In a dream, Sela saw the brass symbol beside the sealed gate.", "saw", "state", "sela", "Sela")
		use := fixtureRecord("symbol-used", 2, "Sela used the brass symbol to decode the sealed gate.", "used", "discovery", "sela", "Sela")
		model := buildFixture(dream, use)
		fingerprint := fingerprintForEvidence(t, model, "dream")
		if fingerprint.Scope.Kind != "dream" {
			t.Fatalf("dream became current-reality fact: %#v", fingerprint.Scope)
		}
		assertRelation(t, model, "enables")
	})

	t.Run("flashback remains a scoped assertion", func(t *testing.T) {
		past := fixtureRecord("past", 0, "Years earlier, Toma buried the compass beneath the oak.", "buried", "time_reference", "toma", "Toma")
		model := buildFixture(past)
		assertion := assertionForEvidence(t, model, "past")
		if assertion.Scope.Kind != "flashback" {
			t.Fatalf("flashback scope lost: %#v", assertion.Scope)
		}
		assertEvidenceRetainedWithoutPromotion(t, model, "past")
	})

	t.Run("unresolved promise remains open without invented fulfillment", func(t *testing.T) {
		promise := fixtureRecord("promise-open", 0, "Avery promised to return the ledger before dawn.", "promised", "interaction", "avery", "Avery")
		model := buildFixture(promise)
		assertPromotedEvidence(t, model, "promise-open")
		if hasNarrativeRelation(model.NarrativeRelations, "fulfills") {
			t.Fatal("unresolved promise was silently fulfilled")
		}
	})

	t.Run("fulfilled promise links setup and payoff", func(t *testing.T) {
		promise := fixtureRecord("promise", 0, "Avery promised to return the ledger before dawn.", "promised", "interaction", "avery", "Avery")
		fulfillment := fixtureRecord("fulfilled", 2, "Avery returned the ledger to Niko before dawn.", "returned", "interaction", "avery", "Avery", "niko", "Niko")
		model := buildFixture(promise, fulfillment)
		assertRelation(t, model, "fulfills")
	})

	t.Run("consequential knowledge transfer is not world truth", func(t *testing.T) {
		warning := fixtureRecord("warning", 0, `"The west gate is trapped," Mira warned Oren.`, "warned", "knowledge_transfer", "mira", "Mira", "oren", "Oren")
		warning.KnowledgeStates = []types.EvidenceKnowledgeState{{State: "shared", CharacterIDs: []string{"mira"}, CharacterNames: []string{"Mira"}, CounterpartyIDs: []string{"oren"}, CounterpartyNames: []string{"Oren"}, Cue: "warned", Confidence: .88}}
		model := buildFixture(warning)
		fingerprint := fingerprintForEvidence(t, model, "warning")
		if fingerprint.EpistemicStatus != "attributed_claim" {
			t.Fatalf("knowledge transfer became world truth: %#v", fingerprint)
		}
	})

	t.Run("repeated information aggregates evidence", func(t *testing.T) {
		one := fixtureRecord("repeat-a", 0, "Mira discovered the archive was sealed from inside.", "discovered", "discovery", "mira", "Mira")
		one.NamedEntities = []types.EvidenceTerm{{Text: "archive", Label: "FAC"}}
		two := fixtureRecord("repeat-b", 3, "Mira discovered the archive was sealed from inside.", "discovered", "discovery", "mira", "Mira")
		two.NamedEntities = one.NamedEntities
		model := buildFixture(one, two)
		if len(model.NarrativeFingerprints) != 1 || len(model.NarrativeFingerprints[0].EvidenceIDs) != 2 {
			t.Fatalf("repetition created duplicate fingerprints: %#v", model.NarrativeFingerprints)
		}
	})

	t.Run("minor detail becomes setup after later dependency", func(t *testing.T) {
		detail := fixtureRecord("thread", 0, "Niko carried a length of red thread.", "carried", "state", "niko", "Niko")
		payoff := fixtureRecord("thread-used", 4, "Niko used the red thread to trace the hidden airflow.", "used", "discovery", "niko", "Niko")
		model := buildFixture(detail, payoff)
		if !hasPromotionReason(fingerprintForEvidence(t, model, "thread"), "retrospective_enables") {
			t.Fatalf("later dependency did not promote its setup: %#v", model.NarrativeFingerprints)
		}
	})

	t.Run("busy scene without persistent change stays below promotion", func(t *testing.T) {
		records := []types.EvidenceRecord{
			fixtureRecord("busy-a", 0, "Ivo ran across the plaza.", "ran", "transition", "ivo", "Ivo"),
			fixtureRecord("busy-b", 0, "Ivo ducked behind a bench.", "ducked", "transition", "ivo", "Ivo"),
			fixtureRecord("busy-c", 0, "Ivo shouted and waved toward the crowd.", "shouted", "interaction", "ivo", "Ivo"),
		}
		model := buildFixture(records...)
		if len(model.NarrativeFingerprints) != 0 {
			t.Fatalf("grammatical activity became narrative significance: %#v", model.NarrativeFingerprints)
		}
	})

	t.Run("large routine evidence remains sparse", func(t *testing.T) {
		records := make([]types.EvidenceRecord, 0, 1500)
		for index := 0; index < cap(records); index++ {
			records = append(records, fixtureRecord(fmt.Sprintf("routine-%d", index), index/50, fmt.Sprintf("Avery walked past marker %d.", index), "walked", "transition", "avery", "Avery"))
		}
		model := buildFixture(records...)
		if len(model.NarrativeFingerprints) != 0 || len(model.NarrativeRelations) != 0 {
			t.Fatalf("routine scale created narrative structure: %d fingerprints, %d relations", len(model.NarrativeFingerprints), len(model.NarrativeRelations))
		}
	})

	t.Run("quiet conversation may change relationship and goal", func(t *testing.T) {
		conversation := fixtureRecord("quiet", 0, "Mira agreed to trust Oren and continue the search.", "agreed", "interaction", "mira", "Mira", "oren", "Oren")
		model := buildFixture(conversation)
		fingerprint := fingerprintForEvidence(t, model, "quiet")
		if fingerprint.Kind != "commitment" {
			t.Fatalf("quiet durable change was missed: %#v", fingerprint)
		}
	})

	t.Run("diagnostic is textual and evidence-driven", func(t *testing.T) {
		promise := fixtureRecord("report", 0, "Avery promised to return the ledger.", "promised", "interaction", "avery", "Avery")
		model := buildFixture(promise)
		for _, expected := range []string{"NARRATIVE FINGERPRINT DIAGNOSTIC", "Promoted because:", "commitment_created", promise.Text} {
			if !strings.Contains(model.DiagnosticReport, expected) {
				t.Fatalf("diagnostic omitted %q:\n%s", expected, model.DiagnosticReport)
			}
		}
	})
}

func fixtureRecord(id string, chapter int, text, action, evidenceType string, characterPairs ...string) types.EvidenceRecord {
	result := types.EvidenceRecord{ID: id, Kind: "event", EvidenceType: evidenceType, ChapterID: "fixture-chapter", ChapterIndex: chapter, Section: "body", SectionIndex: chapter, ParagraphIndex: chapter, SentenceIndex: chapter, StartOffset: chapter * 100, EndOffset: chapter*100 + len(text), Text: text, Action: action, Confidence: .9, Status: "detected", Source: "auto"}
	for index := 0; index+1 < len(characterPairs); index += 2 {
		if characterPairs[index] == "" {
			continue
		}
		result.CharacterIDs = append(result.CharacterIDs, characterPairs[index])
		result.CharacterNames = append(result.CharacterNames, characterPairs[index+1])
	}
	return result
}

func buildFixture(records ...types.EvidenceRecord) *types.StoryFingerprint {
	book := testBook(records)
	return Build(&book, nil)
}

func assertionForEvidence(t *testing.T, model *types.StoryFingerprint, evidenceID string) types.StoryAssertion {
	t.Helper()
	for _, assertion := range model.Assertions {
		if containsString(assertion.EvidenceIDs, evidenceID) {
			return assertion
		}
	}
	t.Fatalf("missing assertion for %s", evidenceID)
	return types.StoryAssertion{}
}

func fingerprintForEvidence(t *testing.T, model *types.StoryFingerprint, evidenceID string) types.NarrativeFingerprint {
	t.Helper()
	for _, fingerprint := range model.NarrativeFingerprints {
		if containsString(fingerprint.EvidenceIDs, evidenceID) {
			return fingerprint
		}
	}
	t.Fatalf("missing narrative fingerprint for %s; assertions=%#v fingerprints=%#v", evidenceID, model.Assertions, model.NarrativeFingerprints)
	return types.NarrativeFingerprint{}
}

func assertPromotedEvidence(t *testing.T, model *types.StoryFingerprint, evidenceID string) {
	t.Helper()
	_ = fingerprintForEvidence(t, model, evidenceID)
}

func assertEvidenceRetainedWithoutPromotion(t *testing.T, model *types.StoryFingerprint, evidenceID string) {
	t.Helper()
	_ = assertionForEvidence(t, model, evidenceID)
	for _, fingerprint := range model.NarrativeFingerprints {
		if containsString(fingerprint.EvidenceIDs, evidenceID) {
			t.Fatalf("%s was promoted without narrative justification: %#v", evidenceID, fingerprint)
		}
	}
}

func assertRelation(t *testing.T, model *types.StoryFingerprint, kind string) {
	t.Helper()
	if !hasNarrativeRelation(model.NarrativeRelations, kind) {
		t.Fatalf("missing %s relation: %#v", kind, model.NarrativeRelations)
	}
}

func hasPromotionReason(fingerprint types.NarrativeFingerprint, code string) bool {
	for _, reason := range fingerprint.PromotionReasons {
		if reason.Code == code {
			return true
		}
	}
	return false
}
