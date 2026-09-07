package fingerprint

import (
	"fmt"
	"sort"
	"strings"

	"draftline/internal/types"
)

// buildFingerprintInspections queries the complete manuscript-memory corpus.
// Narrative developments and display importance are intentionally irrelevant.
func buildFingerprintInspections(model *types.StoryFingerprint, records []types.EvidenceRecord) []types.FingerprintInspection {
	fingerprintByID := map[string]types.ManuscriptFingerprint{}
	fingerprintByEvidence := map[string][]types.ManuscriptFingerprint{}
	for _, fingerprint := range model.Fingerprints {
		fingerprintByID[fingerprint.ID] = fingerprint
		for _, evidenceID := range fingerprint.EvidenceIDs {
			fingerprintByEvidence[evidenceID] = append(fingerprintByEvidence[evidenceID], fingerprint)
		}
	}
	result := []types.FingerprintInspection{}
	for _, relation := range model.FingerprintRelations {
		if relation.Kind != "contradicts" && relation.Kind != "supersedes" {
			continue
		}
		left, leftOK := fingerprintByID[relation.FromID]
		right, rightOK := fingerprintByID[relation.ToID]
		if !leftOK || !rightOK {
			continue
		}
		assessment, severity := relationshipScopeAssessment(left, right)
		title := "Two manuscript assertions conflict"
		if relation.Kind == "supersedes" {
			title = "Later evidence may revise an earlier interpretation"
		}
		result = append(result, types.FingerprintInspection{
			ID: stableID("inspection", relation.ID), Kind: relation.Kind, Severity: severity, Title: title,
			Detail: relation.Explanation, ScopeAssessment: assessment,
			Sides:      []types.FingerprintInspectionSide{inspectionSide("Earlier", left), inspectionSide("Later", right)},
			Confidence: relation.Confidence, Status: "open",
		})
	}
	for _, identity := range model.EventIdentities {
		if identity.Status != "conflicted" {
			continue
		}
		sides := []types.FingerprintInspectionSide{}
		for _, property := range identity.Properties {
			spans := []types.NarrativeEvidenceSpan{}
			for _, fingerprintID := range property.FingerprintIDs {
				spans = appendUniqueSpans(spans, fingerprintByID[fingerprintID].EvidenceSpans...)
			}
			sides = append(sides, types.FingerprintInspectionSide{Label: property.Name + " = " + property.Value, FingerprintIDs: clone(property.FingerprintIDs), EvidenceSpans: spans})
		}
		result = append(result, types.FingerprintInspection{
			ID: stableID("inspection", identity.ID, "conflicting-properties"), Kind: "conflicting_event_accounts", Severity: "review",
			Title: "Accounts of the same event disagree", Detail: "Passages likely describing the same underlying " + identity.EventType + " event preserve incompatible properties.",
			ScopeAssessment: "The conflict is preserved as competing evidence. Attribution and reality scope determine whether it is a continuity error, testimony conflict, unreliable memory, or intentional mystery.",
			Sides:           sides, Confidence: identity.Confidence, Status: "open",
		})
	}
	result = append(result, stateHistoryInspections(model.StateHistories, fingerprintByID)...)
	result = append(result, openObligationInspections(model.Fingerprints, model.FingerprintRelations)...)
	for _, diagnostic := range model.Diagnostics {
		result = append(result, inspectionFromDiagnostic(diagnostic, fingerprintByEvidence, records))
	}
	return dedupeInspections(result)
}

func stateHistoryInspections(histories []types.FingerprintStateHistory, fingerprints map[string]types.ManuscriptFingerprint) []types.FingerprintInspection {
	result := []types.FingerprintInspection{}
	for _, history := range histories {
		if !stableConflictProperty(history.Property) || len(history.Entries) < 2 {
			continue
		}
		for leftIndex := 0; leftIndex < len(history.Entries); leftIndex++ {
			for rightIndex := leftIndex + 1; rightIndex < len(history.Entries); rightIndex++ {
				left, right := history.Entries[leftIndex], history.Entries[rightIndex]
				if normalizeSemantic(left.Value) == normalizeSemantic(right.Value) || !historyEntriesComparable(left, right) {
					continue
				}
				leftFingerprint, leftOK := fingerprints[left.FingerprintID]
				rightFingerprint, rightOK := fingerprints[right.FingerprintID]
				if !leftOK || !rightOK {
					continue
				}
				assessment, severity := relationshipScopeAssessment(leftFingerprint, rightFingerprint)
				result = append(result, types.FingerprintInspection{
					ID: stableID("inspection", history.ID, left.ID, right.ID), Kind: history.Property + "_conflict", Severity: severity,
					Title:           "An established " + strings.ReplaceAll(history.Property, "_", " ") + " conflicts",
					Detail:          fmt.Sprintf("%s is associated with %q and %q in the same reality scope.", history.EntityName, left.Value, right.Value),
					ScopeAssessment: assessment,
					Sides:           []types.FingerprintInspectionSide{inspectionSide("Earlier state", leftFingerprint), inspectionSide("Later state", rightFingerprint)},
					Confidence:      minFloat(left.Confidence, right.Confidence), Status: "open",
				})
				break
			}
		}
	}
	return result
}

func stableConflictProperty(property string) bool {
	switch property {
	case "attribute", "hair", "eye", "eyes", "name", "identity", "rank", "title", "life_status", "asserted_attribute":
		return true
	}
	return false
}

func historyEntriesComparable(left, right types.FingerprintStateEntry) bool {
	if left.Scope.ID != right.Scope.ID {
		return false
	}
	return left.EpistemicStatus != "uncertain_interpretation" && right.EpistemicStatus != "uncertain_interpretation"
}

func openObligationInspections(fingerprints []types.ManuscriptFingerprint, relations []types.ManuscriptFingerprintRelation) []types.FingerprintInspection {
	fulfilled := map[string]bool{}
	for _, relation := range relations {
		if relation.Kind == "fulfills" {
			fulfilled[relation.FromID] = true
		}
	}
	result := []types.FingerprintInspection{}
	for _, fingerprint := range fingerprints {
		if fingerprint.Kind != "commitment" || fulfilled[fingerprint.ID] {
			continue
		}
		result = append(result, types.FingerprintInspection{
			ID: stableID("inspection", fingerprint.ID, "open-obligation"), Kind: "open_obligation", Severity: "information",
			Title: "A promise or obligation remains open", Detail: fingerprint.Statement,
			ScopeAssessment: "No later corpus fingerprint has yet been linked as fulfillment. This may be pending, intentionally unresolved, or phrased too differently for deterministic matching.",
			Sides:           []types.FingerprintInspectionSide{inspectionSide("Obligation", fingerprint)}, Confidence: fingerprint.Confidence, Status: "open",
		})
	}
	return result
}

func inspectionFromDiagnostic(diagnostic types.FingerprintDiagnostic, byEvidence map[string][]types.ManuscriptFingerprint, records []types.EvidenceRecord) types.FingerprintInspection {
	recordByID := evidenceRecordMap(records)
	sides := []types.FingerprintInspectionSide{}
	for index, evidenceID := range diagnostic.EvidenceIDs {
		label := "Evidence"
		if len(diagnostic.EvidenceIDs) > 1 {
			label = fmt.Sprintf("Evidence %d", index+1)
		}
		fingerprintIDs := []string{}
		spans := []types.NarrativeEvidenceSpan{}
		for _, fingerprint := range byEvidence[evidenceID] {
			fingerprintIDs = appendUnique(fingerprintIDs, fingerprint.ID)
			spans = appendUniqueSpans(spans, fingerprint.EvidenceSpans...)
		}
		if len(spans) == 0 {
			if record, exists := recordByID[evidenceID]; exists {
				spans = append(spans, evidenceSpan(record))
			}
		}
		sides = append(sides, types.FingerprintInspectionSide{Label: label, FingerprintIDs: fingerprintIDs, EvidenceSpans: spans})
	}
	return types.FingerprintInspection{
		ID: stableID("inspection", diagnostic.ID), Kind: diagnostic.Kind, Severity: diagnostic.Severity, Title: diagnostic.Title,
		Detail: diagnostic.Detail, ScopeAssessment: "This deterministic inspection retains its source evidence; review reality scope and attribution before treating it as an author error.",
		Sides: sides, Confidence: diagnostic.Confidence, Status: nonEmpty(diagnostic.Status, "open"),
	}
}

func relationshipScopeAssessment(left, right types.ManuscriptFingerprint) (string, string) {
	if left.Scope.ID != right.Scope.ID || left.Scope.Kind != right.Scope.Kind {
		return "The assertions occur in different or uncertain reality scopes; preserve as a cross-scope discrepancy unless the author establishes that the scopes share one world state.", "information"
	}
	if !objectiveAssertionFingerprint(left) || !objectiveAssertionFingerprint(right) {
		return "At least one side is attributed, believed, inferred, deceptive, or otherwise non-objective. Treat this as conflicting story evidence, not automatically as an author mistake.", "review"
	}
	return "Both passages are presented as objective in the same reality scope, making this a likely continuity conflict unless nearby prose acknowledges the change.", "warning"
}

func objectiveAssertionFingerprint(item types.ManuscriptFingerprint) bool {
	return item.EpistemicStatus == "world_state_fact" || item.EpistemicStatus == "externally_corroborated_fact"
}

func inspectionSide(label string, fingerprint types.ManuscriptFingerprint) types.FingerprintInspectionSide {
	return types.FingerprintInspectionSide{Label: label, FingerprintIDs: []string{fingerprint.ID}, EvidenceSpans: append([]types.NarrativeEvidenceSpan(nil), fingerprint.EvidenceSpans...)}
}

func appendUniqueSpans(values []types.NarrativeEvidenceSpan, additions ...types.NarrativeEvidenceSpan) []types.NarrativeEvidenceSpan {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value.EvidenceID] = true
	}
	for _, addition := range additions {
		if !seen[addition.EvidenceID] {
			seen[addition.EvidenceID] = true
			values = append(values, addition)
		}
	}
	return values
}

func dedupeInspections(values []types.FingerprintInspection) []types.FingerprintInspection {
	seen := map[string]bool{}
	result := make([]types.FingerprintInspection, 0, len(values))
	for _, value := range values {
		if !seen[value.ID] {
			seen[value.ID] = true
			result = append(result, value)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		left, right := inspectionChapter(result[i]), inspectionChapter(result[j])
		if left != right {
			return left < right
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func inspectionChapter(value types.FingerprintInspection) int {
	chapter := int(^uint(0) >> 1)
	for _, side := range value.Sides {
		for _, span := range side.EvidenceSpans {
			if span.ChapterIndex < chapter {
				chapter = span.ChapterIndex
			}
		}
	}
	return chapter
}
