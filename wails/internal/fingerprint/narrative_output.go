package fingerprint

import (
	"fmt"
	"sort"
	"strings"

	"draftline/internal/types"
)

// DiagnosticReport rebuilds the semantic layers from persisted evidence and
// returns the plain-text quality-gate report. The supplied book is copied by
// callers that require immutability; Build changes only rebuildable analysis
// fields and never manuscript prose.
func DiagnosticReport(book *types.BookData) string {
	if book == nil || book.Analysis.Evidence == nil {
		return "Narrative fingerprint analysis has not run.\n"
	}
	return Build(book, nil).DiagnosticReport
}

// legacyEventsFromNarrative keeps existing non-visual consumers operational
// while making their input honest: every compatibility event corresponds to a
// promoted narrative fingerprint, never directly to an evidence atom.
func legacyEventsFromNarrative(book *types.BookData, fingerprints []types.NarrativeFingerprint, records []types.EvidenceRecord) []types.FingerprintEvent {
	recordByID := evidenceRecordMap(records)
	result := make([]types.FingerprintEvent, 0, len(fingerprints))
	for _, fingerprint := range fingerprints {
		if len(fingerprint.EvidenceIDs) == 0 {
			continue
		}
		record, exists := recordByID[fingerprint.EvidenceIDs[0]]
		if !exists {
			continue
		}
		locations, objects := eventTerms(record)
		characters, names := []string{}, []string{}
		for _, participant := range fingerprint.Participants {
			if participant.EntityID != "" {
				characters = appendUnique(characters, participant.EntityID)
			}
			if participant.EntityName != "" {
				names = appendUnique(names, participant.EntityName)
			}
		}
		result = append(result, types.FingerprintEvent{
			ID: fingerprint.ID, Summary: fingerprint.Summary, EvidenceIDs: clone(fingerprint.EvidenceIDs), AssertionIDs: clone(fingerprint.AssertionIDs),
			CharacterIDs: characters, CharacterNames: names, Locations: locations, Objects: objects, Kinds: []string{fingerprint.Kind},
			ContextID: fingerprint.Scope.ID, StoryTime: fingerprint.Temporal, ChapterID: record.ChapterID, ChapterIndex: record.ChapterIndex,
			ChapterTitle: chapterTitle(book, record), ParagraphIndex: record.ParagraphIndex, StartOffset: record.StartOffset,
			Importance: narrativeImportance(fingerprint), Confidence: fingerprint.Confidence, Status: fingerprint.Status,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ChapterIndex != result[j].ChapterIndex {
			return result[i].ChapterIndex < result[j].ChapterIndex
		}
		return result[i].StartOffset < result[j].StartOffset
	})
	for index := range result {
		result[index].NarrativeOrder = index
	}
	return result
}

func narrativeImportance(fingerprint types.NarrativeFingerprint) float64 {
	importance := .72
	for _, reason := range fingerprint.PromotionReasons {
		switch reason.Code {
		case "persistent_state_changed", "retrospective_contradicts", "retrospective_supersedes":
			importance = maxFloat(importance, .94)
		case "commitment_created", "goal_or_plan_changed", "retrospective_fulfills", "retrospective_enables":
			importance = maxFloat(importance, .88)
		case "relationship_changed", "consequential_knowledge_changed":
			importance = maxFloat(importance, .84)
		}
	}
	return importance
}

func narrativeDiagnostics(relations []types.NarrativeFingerprintRelation, fingerprints []types.NarrativeFingerprint) []types.FingerprintDiagnostic {
	byID := map[string]types.NarrativeFingerprint{}
	for _, fingerprint := range fingerprints {
		byID[fingerprint.ID] = fingerprint
	}
	result := []types.FingerprintDiagnostic{}
	for _, relation := range relations {
		if relation.Kind != "contradicts" && relation.Kind != "supersedes" {
			continue
		}
		left, right := byID[relation.FromID], byID[relation.ToID]
		kind := "narrative_claim_conflict"
		title := "Two story assertions conflict"
		if relation.Kind == "supersedes" {
			kind = "narrative_interpretation_superseded"
			title = "Later evidence corrects an earlier interpretation"
		}
		result = append(result, types.FingerprintDiagnostic{
			ID: stableID("diagnostic", relation.ID), Kind: kind, Severity: "review", Title: title,
			Detail:   left.Summary + " / " + right.Summary + ". " + relation.Explanation,
			EventIDs: []string{left.ID, right.ID}, EvidenceIDs: clone(relation.EvidenceIDs),
			ChapterIndices: fingerprintChapters(left, right), Confidence: relation.Confidence,
		})
	}
	return result
}

// buildNarrativeDiagnosticReport is deliberately textual. It is the semantic
// quality gate for this milestone and exposes exactly why every promotion and
// relationship exists before any visualization consumes the model.
func buildNarrativeDiagnosticReport(model *types.StoryFingerprint) string {
	if model == nil {
		return "Narrative fingerprint analysis has not run.\n"
	}
	var b strings.Builder
	fmt.Fprintln(&b, "NARRATIVE FINGERPRINT DIAGNOSTIC")
	fmt.Fprintf(&b, "Engine: %s (schema %d)\n", model.Engine, model.Version)
	fmt.Fprintf(&b, "Evidence atoms: %d\nAssertions: %d\nPromoted fingerprints: %d\nRetained as evidence only: %d\n\n",
		model.PromotionStats.EvidenceAtoms, model.PromotionStats.Assertions, model.PromotionStats.PromotedFingerprints, model.PromotionStats.RetainedAsEvidence)
	if len(model.NarrativeFingerprints) == 0 {
		fmt.Fprintln(&b, "No assertions met the precision-first narrative promotion threshold.")
		return b.String()
	}
	relationsByFingerprint := map[string][]types.NarrativeFingerprintRelation{}
	for _, relation := range model.NarrativeRelations {
		relationsByFingerprint[relation.FromID] = append(relationsByFingerprint[relation.FromID], relation)
		relationsByFingerprint[relation.ToID] = append(relationsByFingerprint[relation.ToID], relation)
	}
	for index, fingerprint := range model.NarrativeFingerprints {
		fmt.Fprintf(&b, "%d. %s\n", index+1, fingerprint.Summary)
		fmt.Fprintf(&b, "   ID: %s\n   Kind: %s\n   Epistemic: %s\n   Attribution: %s", fingerprint.ID, fingerprint.Kind, fingerprint.EpistemicStatus, fingerprint.Attribution.Kind)
		if fingerprint.Attribution.EntityName != "" {
			fmt.Fprintf(&b, " (%s)", fingerprint.Attribution.EntityName)
		}
		fmt.Fprintf(&b, "\n   Scope: %s — %s\n   Persistence: %s\n   Confidence: %.2f\n", fingerprint.Scope.Kind, fingerprint.Scope.Label, fingerprint.Persistence, fingerprint.Confidence)
		if fingerprint.StateChange != nil {
			fmt.Fprintf(&b, "   State: %s / %s", fingerprint.StateChange.StateKind, fingerprint.StateChange.Operation)
			if fingerprint.StateChange.Previous != "" {
				fmt.Fprintf(&b, " / previous=%q", fingerprint.StateChange.Previous)
			}
			fmt.Fprintf(&b, " / new=%q\n", fingerprint.StateChange.New)
		}
		fmt.Fprintln(&b, "   Promoted because:")
		for _, reason := range fingerprint.PromotionReasons {
			fmt.Fprintf(&b, "     - %s: %s (%.2f)\n", reason.Code, reason.Explanation, reason.Confidence)
		}
		fmt.Fprintln(&b, "   Evidence:")
		for _, span := range fingerprint.EvidenceSpans {
			fmt.Fprintf(&b, "     - chapter %d, paragraph %d, sentence %d [%d:%d]: %q\n", span.ChapterIndex+1, span.ParagraphIndex+1, span.SentenceIndex+1, span.StartOffset, span.EndOffset, span.Quote)
		}
		if relations := relationsByFingerprint[fingerprint.ID]; len(relations) > 0 {
			fmt.Fprintln(&b, "   Relationships:")
			for _, relation := range relations {
				direction, other := "to", relation.ToID
				if relation.ToID == fingerprint.ID {
					direction, other = "from", relation.FromID
				}
				fmt.Fprintf(&b, "     - %s %s %s: %s (%.2f)\n", relation.Kind, direction, other, relation.Explanation, relation.Confidence)
			}
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}

func fingerprintChapters(values ...types.NarrativeFingerprint) []int {
	result := []int{}
	for _, value := range values {
		for _, span := range value.EvidenceSpans {
			result = appendUniqueInt(result, span.ChapterIndex)
		}
	}
	return result
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
