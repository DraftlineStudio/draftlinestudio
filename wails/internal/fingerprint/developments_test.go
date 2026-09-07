package fingerprint

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

func TestNarrativeDevelopmentSynthesizesQuietTransitionAcrossContext(t *testing.T) {
	doubt := fixtureRecord("doubt", 0, "Mira suspected Oren was hiding the records.", "suspected", "knowledge_state", "mira", "Mira", "oren", "Oren")
	shown := fixtureRecord("shown", 0, "Oren showed Mira the complete records.", "showed", "interaction", "oren", "Oren", "mira", "Mira")
	agreement := fixtureRecord("agreement", 0, "Mira agreed to trust Oren and continue the search.", "agreed", "interaction", "mira", "Mira", "oren", "Oren")
	doubt.ParagraphIndex, shown.ParagraphIndex, agreement.ParagraphIndex = 0, 1, 2
	doubt.StartOffset, shown.StartOffset, agreement.StartOffset = 0, 100, 200
	model := buildFixture(doubt, shown, agreement)
	development := findDevelopment(t, model.NarrativeDevelopments, "obligation_created")
	if len(development.FingerprintIDs) < 3 {
		t.Fatalf("development did not synthesize its surrounding context: %#v", development)
	}
	if development.Summary == agreement.Text || !strings.Contains(strings.ToLower(development.Summary), "obligation") {
		t.Fatalf("development is a copied prose fragment rather than normalized change: %#v", development)
	}
}

func TestNarrativeDevelopmentLinksSetupAndLaterUseAcrossChapters(t *testing.T) {
	setup := fixtureRecord("setup", 0, "Avery acquired the brass key to the archive.", "acquired", "state", "avery", "Avery")
	use := fixtureRecord("use", 4, "Avery used the brass key to unlock the archive door.", "used", "interaction", "avery", "Avery")
	model := buildFixture(setup, use)
	development := findDevelopment(t, model.NarrativeDevelopments, "causal_enablement")
	if development.ChapterStart != 0 || development.ChapterEnd != 4 || len(development.FingerprintIDs) != 2 {
		t.Fatalf("cross-chapter setup/payoff was not synthesized: %#v", development)
	}
}

func TestNarrativeDevelopmentCapturesObjectiveAndObstacle(t *testing.T) {
	goal := fixtureRecord("goal", 0, "Avery decided to investigate the eastern signal.", "decided", "state", "avery", "Avery")
	obstacle := fixtureRecord("obstacle", 1, "The flooded passage blocked the investigation team.", "blocked", "state", "team", "Investigation Team")
	model := buildFixture(goal, obstacle)
	if development := findDevelopment(t, model.NarrativeDevelopments, "objective_change"); !strings.Contains(strings.ToLower(development.Summary), "objective") {
		t.Fatalf("objective change lacks normalized meaning: %#v", development)
	}
	if development := findDevelopment(t, model.NarrativeDevelopments, "obstacle"); !strings.Contains(strings.ToLower(development.Summary), "obstructed") {
		t.Fatalf("obstacle lacks course-change meaning: %#v", development)
	}
}

func TestRepeatedAccountsOfConsequentialStateCreateRevelation(t *testing.T) {
	first := fixtureRecord("first", 0, `"Avery died during the bridge crossing," Mira testified.`, "died", "interaction", "avery", "Avery", "mira", "Mira")
	second := fixtureRecord("second", 2, `"Avery was killed during the bridge crossing," Oren reported.`, "killed", "interaction", "avery", "Avery", "oren", "Oren")
	model := buildFixture(first, second)
	development := findDevelopment(t, model.NarrativeDevelopments, "model_revelation")
	if len(development.FingerprintIDs) != 2 || !strings.Contains(strings.ToLower(development.After), "dead") {
		t.Fatalf("corroborating accounts did not synthesize a consequential revelation: %#v", development)
	}
}

func TestActionHeavyPassageWithoutStoryChangeCreatesNoDevelopment(t *testing.T) {
	records := []types.EvidenceRecord{
		fixtureRecord("run", 0, "Ivo ran across the plaza.", "ran", "transition", "ivo", "Ivo"),
		fixtureRecord("duck", 0, "Ivo ducked behind a bench.", "ducked", "transition", "ivo", "Ivo"),
		fixtureRecord("wave", 0, "Ivo waved toward the crowd.", "waved", "interaction", "ivo", "Ivo"),
	}
	model := buildFixture(records...)
	if len(model.Fingerprints) != 3 || len(model.NarrativeDevelopments) != 0 {
		t.Fatalf("activity was confused with story movement: fingerprints=%d developments=%#v", len(model.Fingerprints), model.NarrativeDevelopments)
	}
}

func TestConcreteFindingPromotesOnlyAfterLaterDependence(t *testing.T) {
	found := fixtureRecord("found", 0, "Avery found a brass token beneath the bench.", "found", "discovery", "avery", "Avery")
	if model := buildFixture(found); len(model.NarrativeDevelopments) != 0 {
		t.Fatalf("an isolated object interaction became a development: %#v", model.NarrativeDevelopments)
	}
	use := fixtureRecord("use", 2, "Avery presented the brass token to gain entry to the archive.", "presented", "interaction", "avery", "Avery")
	model := buildFixture(found, use)
	development := findDevelopment(t, model.NarrativeDevelopments, "knowledge_change")
	if !strings.Contains(strings.ToLower(development.AdvancedConcern), "brass token") || len(development.FingerprintIDs) == 0 {
		t.Fatalf("later dependence did not retrospectively promote the finding: %#v", development)
	}
}

func TestDevelopmentDiagnosticExplainsContextAndEvidence(t *testing.T) {
	goal := fixtureRecord("goal", 0, "Avery decided to investigate the eastern signal.", "decided", "state", "avery", "Avery")
	model := buildFixture(goal)
	for _, expected := range []string{"NARRATIVE DEVELOPMENT DIAGNOSTIC", "objective_change", "Before:", "After:", "Fingerprints:", "Supporting evidence:", `"Avery decided to investigate the eastern signal."`} {
		if !strings.Contains(model.DevelopmentDiagnostic, expected) {
			t.Fatalf("development report omitted %q:\n%s", expected, model.DevelopmentDiagnostic)
		}
	}
}

func findDevelopment(t *testing.T, developments []types.NarrativeDevelopment, kind string) types.NarrativeDevelopment {
	t.Helper()
	for _, development := range developments {
		if development.Kind == kind {
			return development
		}
	}
	t.Fatalf("missing %s development: %#v", kind, developments)
	return types.NarrativeDevelopment{}
}
