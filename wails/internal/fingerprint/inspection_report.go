package fingerprint

import (
	"fmt"
	"strings"

	"draftline/internal/types"
)

func buildInspectionDiagnosticReport(model *types.StoryFingerprint) string {
	if model == nil {
		return "Fingerprint inspection analysis has not run.\n"
	}
	var b strings.Builder
	fmt.Fprintln(&b, "FINGERPRINT INSPECTION DIAGNOSTIC")
	fmt.Fprintf(&b, "Inspections: %d\n\n", len(model.Inspections))
	if len(model.Inspections) == 0 {
		fmt.Fprintln(&b, "No concrete corpus conflicts or unresolved obligations were identified.")
		return b.String()
	}
	for index, inspection := range model.Inspections {
		fmt.Fprintf(&b, "%d. [%s] %s\n", index+1, inspection.Kind, inspection.Title)
		fmt.Fprintf(&b, "   Severity: %s\n   Confidence: %.2f\n   Detail: %s\n   Scope assessment: %s\n",
			inspection.Severity, inspection.Confidence, inspection.Detail, inspection.ScopeAssessment)
		for _, side := range inspection.Sides {
			fmt.Fprintf(&b, "   %s:\n", side.Label)
			for _, span := range side.EvidenceSpans {
				fmt.Fprintf(&b, "     - %s — chapter %d, paragraph %d, sentence %d [%d:%d]: %q\n",
					span.EvidenceID, span.ChapterIndex+1, span.ParagraphIndex+1, span.SentenceIndex+1, span.StartOffset, span.EndOffset, span.Quote)
			}
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}
