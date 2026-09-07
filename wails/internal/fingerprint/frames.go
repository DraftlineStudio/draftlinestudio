package fingerprint

// Frame extraction (v5). Stubbed by the architectural reset; the typed
// extractors land in the next builds. The contract they must honor:
//   - constrained frame types only (types.Frame*)
//   - Detail is ALWAYS a verbatim slice of the evidence text
//   - participants are canonical entity names, anchored in the sentence
//   - abstain (record the reason) instead of guessing a slot

import "draftline/internal/types"

func extractFrames(
	book *types.BookData,
	records []types.EvidenceRecord,
	contextByEvidence map[string]string,
	points map[string]types.StoryTime,
	scopes map[string]types.NarrativeRealityScope,
) []types.NarrativeFrame {
	return []types.NarrativeFrame{}
}

func buildLedgers(frames []types.NarrativeFrame) []types.StateLedger {
	return nil
}

func resolveEventIdentities(frames []types.NarrativeFrame) []types.NarrativeEventIdentity {
	return nil
}

func synthesizeDevelopments(
	frames []types.NarrativeFrame,
	ledgers []types.StateLedger,
	identities []types.NarrativeEventIdentity,
) []types.NarrativeDevelopment {
	return nil
}

func buildInspections(
	frames []types.NarrativeFrame,
	ledgers []types.StateLedger,
	identities []types.NarrativeEventIdentity,
	scopes map[string]types.NarrativeRealityScope,
) []types.NarrativeInspection {
	return nil
}
