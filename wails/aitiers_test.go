package main

import "testing"

// A ladder must skip ids the endpoint does not serve, which is what keeps a
// retired model from becoming a 404 in front of the writer.
func TestLadderSkipsModelsTheEndpointDoesNotServe(t *testing.T) {
	a := &App{}
	seedCatalogue("openai", map[string]bool{"gpt-5.5": true, "gpt-4o": true})
	defer clearCatalogue()

	if got := a.resolveTierModel(openAILadder, "key", rewriteRequestProfile("copy_edit")); got != "gpt-5.5" {
		t.Fatalf("lite tier did not fall through to an available model: %q", got)
	}
	if got := a.resolveTierModel(openAILadder, "key", standardAIRequest); got != "gpt-5.5" {
		t.Fatalf("high tier did not fall through to an available model: %q", got)
	}
}

// An unreachable endpoint must still yield a model rather than an empty one.
func TestLadderFallsBackWhenTheCatalogueIsUnavailable(t *testing.T) {
	a := &App{}
	seedCatalogue("openai", nil)
	defer clearCatalogue()

	if got := a.resolveTierModel(openAILadder, "key", standardAIRequest); got != openAILadder.tiers[tierHigh][0] {
		t.Fatalf("expected the first preference, got %q", got)
	}
}

// Every tier needs at least one candidate, or a task silently sends no model.
func TestEveryLadderTierHasCandidates(t *testing.T) {
	for _, l := range []modelLadder{claudeLadder, openAILadder} {
		for _, tier := range []aiTier{tierLite, tierMedium, tierHigh} {
			if len(l.tiers[tier]) == 0 {
				t.Fatalf("%s ladder has no %s candidates", l.name, tier)
			}
		}
	}
}
