package fingerprint

import (
	"fmt"
	"strings"

	"draftline/internal/types"
)

func buildNarrativeDevelopmentDiagnosticReport(model *types.StoryFingerprint) string {
	if model == nil {
		return "Narrative development analysis has not run.\n"
	}
	var b strings.Builder
	fmt.Fprintln(&b, "NARRATIVE DEVELOPMENT DIAGNOSTIC")
	fmt.Fprintf(&b, "Developments: %d\n\n", len(model.NarrativeDevelopments))
	if len(model.NarrativeDevelopments) == 0 {
		fmt.Fprintln(&b, "No contextual changes in the course of the story were established with sufficient evidence.")
		return b.String()
	}
	fingerprints := make(map[string]types.ManuscriptFingerprint, len(model.Fingerprints))
	for _, fingerprint := range model.Fingerprints {
		fingerprints[fingerprint.ID] = fingerprint
	}
	for index, development := range model.NarrativeDevelopments {
		fmt.Fprintf(&b, "%d. %s\n", index+1, development.Summary)
		fmt.Fprintf(&b, "   ID: %s\n   Kind: %s\n   Chapters: %d–%d\n   Scope: %s — %s\n   Confidence: %.2f\n",
			development.ID, development.Kind, development.ChapterStart+1, development.ChapterEnd+1, development.Scope.Kind, development.Scope.Label, development.Confidence)
		if development.Before != "" {
			fmt.Fprintf(&b, "   Before: %s\n", development.Before)
		}
		fmt.Fprintf(&b, "   After: %s\n", development.After)
		if development.AdvancedConcern != "" {
			fmt.Fprintf(&b, "   Advanced concern: %s\n", development.AdvancedConcern)
		}
		if len(development.DependencyIDs) > 0 {
			fmt.Fprintf(&b, "   Depends on developments: %s\n", strings.Join(development.DependencyIDs, ", "))
		}
		fmt.Fprintln(&b, "   Synthesized because:")
		for _, reason := range development.Reasons {
			fmt.Fprintf(&b, "     - %s: %s (%.2f)\n", reason.Code, reason.Explanation, reason.Confidence)
		}
		fmt.Fprintf(&b, "   Fingerprints: %s\n", strings.Join(development.FingerprintIDs, ", "))
		fmt.Fprintln(&b, "   Supporting evidence:")
		for _, fingerprintID := range development.FingerprintIDs {
			fingerprint, exists := fingerprints[fingerprintID]
			if !exists {
				continue
			}
			fmt.Fprintf(&b, "     - %s [%s; %s; confidence %.2f]\n", fingerprint.Statement, fingerprint.EpistemicStatus, fingerprint.Scope.Kind, fingerprint.Confidence)
			for _, span := range fingerprint.EvidenceSpans {
				fmt.Fprintf(&b, "       %s — chapter %d, paragraph %d, sentence %d: %q\n", span.EvidenceID, span.ChapterIndex+1, span.ParagraphIndex+1, span.SentenceIndex+1, span.Quote)
			}
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}
