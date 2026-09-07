package fingerprint

import (
	"fmt"
	"strings"

	"draftline/internal/types"
)

func buildCorpusDiagnosticReport(model *types.StoryFingerprint) string {
	if model == nil {
		return "Fingerprint corpus analysis has not run.\n"
	}
	var b strings.Builder
	fmt.Fprintln(&b, "FINGERPRINT CORPUS DIAGNOSTIC")
	fmt.Fprintf(&b, "Engine: %s (schema %d)\n", model.Engine, model.Version)
	fmt.Fprintf(&b, "Evidence atoms: %d\nAssertions: %d\nFingerprints: %d\nRelations: %d\nSame-event identities: %d\nState histories: %d\n\n",
		model.CorpusStats.EvidenceAtoms, model.CorpusStats.Assertions, model.CorpusStats.Fingerprints,
		model.CorpusStats.Relations, model.CorpusStats.EventIdentities, model.CorpusStats.StateHistories)
	if len(model.Fingerprints) == 0 {
		fmt.Fprintln(&b, "No manuscript fingerprints are available.")
		return b.String()
	}
	relations := corpusRelationsByFingerprint(model.FingerprintRelations)
	for index, fingerprint := range model.Fingerprints {
		fmt.Fprintf(&b, "%d. %s\n", index+1, fingerprint.Statement)
		fmt.Fprintf(&b, "   ID: %s\n   Kind: %s\n   Proposition: subject=%q predicate=%q object=%q polarity=%s\n",
			fingerprint.ID, fingerprint.Kind, fingerprint.Subject, fingerprint.Predicate, fingerprint.Object, fingerprint.Polarity)
		fmt.Fprintf(&b, "   Epistemic: %s\n   Attribution: %s", fingerprint.EpistemicStatus, fingerprint.Attribution.Kind)
		if fingerprint.Attribution.EntityName != "" {
			fmt.Fprintf(&b, " (%s)", fingerprint.Attribution.EntityName)
		}
		fmt.Fprintf(&b, "\n   Scope: %s — %s\n   Persistence: %s\n   Confidence: %.2f\n", fingerprint.Scope.Kind, fingerprint.Scope.Label, fingerprint.Persistence, fingerprint.Confidence)
		if fingerprint.EventIdentityID != "" {
			fmt.Fprintf(&b, "   Same-event identity: %s\n", fingerprint.EventIdentityID)
		}
		if fingerprint.StateChange != nil {
			fmt.Fprintf(&b, "   State: %s / %s", fingerprint.StateChange.StateKind, fingerprint.StateChange.Operation)
			if fingerprint.StateChange.Previous != "" {
				fmt.Fprintf(&b, " / previous=%q", fingerprint.StateChange.Previous)
			}
			fmt.Fprintf(&b, " / new=%q\n", fingerprint.StateChange.New)
		}
		fmt.Fprintln(&b, "   Evidence:")
		for _, span := range fingerprint.EvidenceSpans {
			fmt.Fprintf(&b, "     - %s — chapter %d, paragraph %d, sentence %d [%d:%d]: %q\n",
				span.EvidenceID, span.ChapterIndex+1, span.ParagraphIndex+1, span.SentenceIndex+1, span.StartOffset, span.EndOffset, span.Quote)
		}
		for _, relation := range relations[fingerprint.ID] {
			direction, other := "to", relation.ToID
			if relation.ToID == fingerprint.ID {
				direction, other = "from", relation.FromID
			}
			fmt.Fprintf(&b, "   Relation: %s %s %s — %s (%.2f)\n", relation.Kind, direction, other, relation.Explanation, relation.Confidence)
		}
		fmt.Fprintln(&b)
	}
	if len(model.EventIdentities) > 0 {
		fmt.Fprintln(&b, "SAME-EVENT IDENTITIES")
		for _, identity := range model.EventIdentities {
			fmt.Fprintf(&b, "- %s: type=%s status=%s confidence=%.2f fingerprints=%s\n", identity.ID, identity.EventType, identity.Status, identity.Confidence, strings.Join(identity.FingerprintIDs, ", "))
			for _, property := range identity.Properties {
				fmt.Fprintf(&b, "  property %s=%q (%s; evidence %s)\n", property.Name, property.Value, property.EpistemicStatus, strings.Join(property.EvidenceIDs, ", "))
			}
		}
	}
	if len(model.StateHistories) > 0 {
		fmt.Fprintln(&b, "STATE HISTORIES")
		for _, history := range model.StateHistories {
			fmt.Fprintf(&b, "- %s / %s", history.EntityName, history.Property)
			if history.Qualifier != "" {
				fmt.Fprintf(&b, " (%s)", history.Qualifier)
			}
			fmt.Fprintln(&b)
			for _, entry := range history.Entries {
				fmt.Fprintf(&b, "  chapter %d paragraph %d: %s → %q [%s; evidence %s]\n",
					entry.ChapterIndex+1, entry.ParagraphIndex+1, entry.Operation, entry.Value, entry.EpistemicStatus, strings.Join(entry.EvidenceIDs, ", "))
			}
		}
	}
	return b.String()
}

func corpusRelationsByFingerprint(values []types.ManuscriptFingerprintRelation) map[string][]types.ManuscriptFingerprintRelation {
	result := map[string][]types.ManuscriptFingerprintRelation{}
	for _, relation := range values {
		result[relation.FromID] = append(result[relation.FromID], relation)
		result[relation.ToID] = append(result[relation.ToID], relation)
	}
	return result
}
