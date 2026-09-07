package fingerprint

import (
	"testing"

	"draftline/internal/types"
)

func TestFingerprintCorpusRetainsRoutineContinuityMemory(t *testing.T) {
	movement := fixtureRecord("movement", 0, "Avery walked to the truck at the depot.", "walked", "transition", "avery", "Avery")
	movement.NamedEntities = []types.EvidenceTerm{{Text: "truck", Label: "PRODUCT"}, {Text: "depot", Label: "LOC"}}
	model := buildFixture(movement)
	if len(model.Fingerprints) != 1 {
		t.Fatalf("routine continuity memory was discarded: %#v", model.Fingerprints)
	}
	fingerprint := model.Fingerprints[0]
	if fingerprint.Statement == "" || !containsString(fingerprint.EvidenceIDs, "movement") || len(fingerprint.EvidenceSpans) != 1 {
		t.Fatalf("fingerprint lost semantic detail or provenance: %#v", fingerprint)
	}
	if len(model.NarrativeFingerprints) != 0 {
		t.Fatalf("legacy narrative promotion unexpectedly admitted routine motion: %#v", model.NarrativeFingerprints)
	}
}

func TestFingerprintCorpusPreservesEpistemicAndRealityScope(t *testing.T) {
	claim := fixtureRecord("claim", 0, `"The western door is unlocked," Mira claimed.`, "is", "interaction", "mira", "Mira")
	dream := fixtureRecord("dream", 1, "In a dream, Oren saw a red beacon beyond the ridge.", "saw", "state", "oren", "Oren")
	model := buildFixture(claim, dream)
	claimFingerprint := manuscriptFingerprintForEvidence(t, model, "claim")
	if claimFingerprint.EpistemicStatus != "attributed_claim" || claimFingerprint.Attribution.EntityName != "Mira" {
		t.Fatalf("claim became objective world truth: %#v", claimFingerprint)
	}
	dreamFingerprint := manuscriptFingerprintForEvidence(t, model, "dream")
	if dreamFingerprint.Scope.Kind != "dream" {
		t.Fatalf("dream memory leaked into current reality: %#v", dreamFingerprint.Scope)
	}
}

func TestRepeatedAssertionCollectsAllEvidenceWithoutDuplicateMemory(t *testing.T) {
	first := fixtureRecord("first", 0, "Avery carried the brass key.", "carried", "state", "avery", "Avery")
	second := fixtureRecord("second", 2, "Avery carried the brass key.", "carried", "state", "avery", "Avery")
	model := buildFixture(first, second)
	if len(model.Fingerprints) != 1 {
		t.Fatalf("equivalent memory was duplicated: %#v", model.Fingerprints)
	}
	if len(model.Fingerprints[0].EvidenceIDs) != 2 || len(model.Fingerprints[0].EvidenceSpans) != 2 {
		t.Fatalf("equivalent memory lost supporting passages: %#v", model.Fingerprints[0])
	}
}

func TestSameEventIdentityRetainsConflictingAccounts(t *testing.T) {
	first := fixtureRecord("account-one", 0, `"Avery fired first," Mira testified.`, "fired", "interaction", "avery", "Avery", "mira", "Mira")
	second := fixtureRecord("account-two", 1, `"Oren fired first," Sela testified.`, "fired", "interaction", "oren", "Oren", "sela", "Sela")
	model := buildFixture(first, second)
	if len(model.EventIdentities) != 1 {
		t.Fatalf("same underlying event was not resolved: %#v", model.EventIdentities)
	}
	identity := model.EventIdentities[0]
	if identity.Status != "conflicted" || len(identity.Properties) != 2 || len(identity.EvidenceIDs) != 2 {
		t.Fatalf("conflicting event properties were averaged or lost: %#v", identity)
	}
	for _, evidenceID := range []string{"account-one", "account-two"} {
		if manuscriptFingerprintForEvidence(t, model, evidenceID).EventIdentityID != identity.ID {
			t.Fatalf("%s was not linked to the shared event identity", evidenceID)
		}
	}
}

func manuscriptFingerprintForEvidence(t *testing.T, model *types.StoryFingerprint, evidenceID string) types.ManuscriptFingerprint {
	t.Helper()
	for _, fingerprint := range model.Fingerprints {
		if containsString(fingerprint.EvidenceIDs, evidenceID) {
			return fingerprint
		}
	}
	t.Fatalf("missing manuscript fingerprint for %s", evidenceID)
	return types.ManuscriptFingerprint{}
}
