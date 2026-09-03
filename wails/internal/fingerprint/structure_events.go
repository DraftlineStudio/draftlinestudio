package fingerprint

import (
	"fmt"
	"math"
	"strings"

	"draftline/internal/types"
)

const (
	significantOccurrenceThreshold = .55
	supportingEvidenceThreshold    = .38
)

func significantDisposition(event types.SignificantStoryEvent, records map[string]types.EvidenceRecord, decisions []types.StructureAuthorDecision) (bool, types.StructureDecision) {
	for _, decision := range matchingStructureDecisions(event, decisions) {
		if decision.Action == "irrelevant" {
			return false, structureDecision("aggregate_creation", "excluded", "author-irrelevant", "The author marked this inference as irrelevant noise.", 1, 1, event.Confidence, nil)
		}
		if decision.Action == "confirm" || decision.Action == "correct" {
			return true, structureDecision("aggregate_creation", "created", "author-decision", "The author confirmed or corrected this narrative occurrence.", 1, 1, 1, nil)
		}
	}

	signals := []types.StructureSignal{}
	add := func(code, detail string, weight float64) {
		signals = append(signals, types.StructureSignal{Code: code, Detail: detail, Weight: weight, EvidenceIDs: clone(event.EvidenceIDs)})
	}
	if len(event.ObligationThreadIDs) > 0 {
		add("obligation-change", "The occurrence opens or resolves a tracked narrative obligation.", .9)
	}
	if len(event.StateIDs) > 0 {
		add("persistent-state-change", "The occurrence starts a tracked persistent state.", .85)
	}
	if event.SalienceReasons.Contradiction > 0 {
		add("contradiction", "The occurrence participates in conflicting evidence.", .9)
	}
	for _, kind := range event.Kinds {
		switch kind {
		case "discovery", "knowledge_change", "knowledge_state", "knowledge_transfer":
			add("knowledge-event", "The source records a discovery or change in knowledge.", .8)
		case "transition":
			add("transition-event", "The source records a temporal or spatial transition.", .75)
		case "introduction":
			add("introduction-event", "The source introduces a participant or story element.", .68)
		case "interaction":
			add("interaction-event", "The source records an interaction between participants.", .62)
		}
	}
	if event.SalienceReasons.Decision > 0 {
		add("decision-language", "The source contains language associated with a decision or commitment.", .72)
	}
	for _, id := range event.EvidenceIDs {
		record, ok := records[id]
		if !ok || record.Kind != "event" || strings.TrimSpace(record.Action) == "" {
			continue
		}
		add("explicit-action", fmt.Sprintf("The source extracts the action %q.", record.Action), .6)
		break
	}

	core := len(signals) > 0
	outcome, rule, summary := "supporting", "supporting-atomic-evidence", "The atomic fact remains lossless evidence but does not independently establish a significant occurrence."
	if core {
		outcome, rule, summary = "created", "narrative-nucleus", "The evidence establishes an action, discovery, decision, transition, interaction, obligation, contradiction, or persistent state change."
	}
	return core, structureDecision("aggregate_creation", outcome, rule, summary, 0, 0, event.Confidence, signals)
}

func significantMembershipDecision(existing, candidate types.SignificantStoryEvent, records map[string]types.EvidenceRecord, supporting bool) types.StructureDecision {
	threshold := significantOccurrenceThreshold
	kind := "occurrence-membership"
	if supporting {
		threshold = supportingEvidenceThreshold
		kind = "supporting-evidence-membership"
	}
	if existing.ChapterID != candidate.ChapterID {
		return rejectedMembership(kind, "chapter-discontinuity", "The source passages occur in different manuscript chapters.", candidate, threshold)
	}
	if existing.ContextID != candidate.ContextID {
		return rejectedMembership(kind, "context-discontinuity", "The source passages belong to different temporal or reality contexts.", candidate, threshold)
	}
	if incompatibleStoryDays(existing.StoryTime, candidate.StoryTime) {
		return rejectedMembership(kind, "story-time-conflict", "Resolved story-time placements put the occurrences on different days.", candidate, threshold)
	}

	signals := []types.StructureSignal{}
	score := 0.0
	add := func(code, detail string, weight float64, evidence ...string) {
		score += weight
		signals = append(signals, types.StructureSignal{Code: code, Detail: detail, Weight: weight, EvidenceIDs: appendUnique([]string{}, evidence...)})
	}
	distance := sourceParagraphDistance(existing, candidate)
	switch {
	case distance == 0:
		add("same-paragraph", "The facts occur in the same source paragraph.", .22, candidate.EvidenceIDs...)
	case distance == 1:
		add("adjacent-paragraph", "The facts occur in adjacent source paragraphs.", .12, candidate.EvidenceIDs...)
	case distance <= 3:
		add("nearby-prose", "The facts occur within the same local passage.", .04, candidate.EvidenceIDs...)
	}
	if stringOverlap(existing.CharacterIDs, candidate.CharacterIDs) {
		add("shared-participant", "The facts share an identified participant.", .18, candidate.EvidenceIDs...)
	}
	if termOverlap(existing.Locations, candidate.Locations) {
		add("shared-location", "The facts share an extracted location.", .15, candidate.EvidenceIDs...)
	}
	if termOverlap(existing.Objects, candidate.Objects) {
		add("shared-object", "The facts concern the same extracted object or institution.", .15, candidate.EvidenceIDs...)
	}
	if stringOverlap(existing.StateIDs, candidate.StateIDs) {
		add("shared-state", "The facts participate in the same persistent state change.", .28, candidate.EvidenceIDs...)
	}
	if stringOverlap(existing.ObligationThreadIDs, candidate.ObligationThreadIDs) {
		add("shared-obligation", "The facts participate in the same narrative obligation.", .35, candidate.EvidenceIDs...)
	}
	if complementaryKinds(existing.Kinds, candidate.Kinds) {
		add("compatible-event-kinds", "The event kinds can describe different facets of one occurrence.", .12, candidate.EvidenceIDs...)
	}
	existingActions, candidateActions := significantActions(existing, records), significantActions(candidate, records)
	if stringOverlap(existingActions, candidateActions) {
		add("shared-action", "The facts extract the same normalized action.", .32, candidate.EvidenceIDs...)
	} else if !supporting && len(existingActions) > 0 && len(candidateActions) > 0 {
		add("independent-actions", "The facts contain different explicit actions and may be separate occurrences.", -.28, candidate.EvidenceIDs...)
	}
	if sameResolvedStoryTime(existing.StoryTime, candidate.StoryTime) {
		add("same-story-time", "The facts share a resolved story-time placement.", .1, candidate.EvidenceIDs...)
	}

	outcome, rule, summary := "boundary", "insufficient-occurrence-evidence", "The available evidence does not establish that both candidates describe the same occurrence."
	if score >= threshold {
		outcome, rule, summary = "included", "occurrence-evidence", "Independent continuity signals support membership in the same narrative occurrence."
	}
	confidence := math.Min(existing.Confidence, candidate.Confidence)
	decision := structureDecision(kind, outcome, rule, summary, score, threshold, confidence, signals)
	decision.ChildID = first(candidate.FingerprintEventIDs)
	decision.CandidateID = first(candidate.FingerprintEventIDs)
	return decision
}

func rejectedMembership(kind, rule, summary string, candidate types.SignificantStoryEvent, threshold float64) types.StructureDecision {
	decision := structureDecision(kind, "boundary", rule, summary, 0, threshold, candidate.Confidence, []types.StructureSignal{{Code: rule, Detail: summary, EvidenceIDs: clone(candidate.EvidenceIDs)}})
	decision.ChildID = first(candidate.FingerprintEventIDs)
	decision.CandidateID = first(candidate.FingerprintEventIDs)
	return decision
}

func structureDecision(kind, outcome, rule, summary string, score, threshold, confidence float64, signals []types.StructureSignal) types.StructureDecision {
	return types.StructureDecision{Kind: kind, Outcome: outcome, Rule: rule, Summary: summary, Score: score, Threshold: threshold, Confidence: clamp01(confidence), Signals: signals}
}

func seedMembershipDecision(event types.SignificantStoryEvent) types.StructureDecision {
	decision := structureDecision("occurrence-membership", "included", "aggregate-seed", "This fingerprint event establishes the initial narrative nucleus.", 1, 1, event.Confidence, nil)
	decision.ChildID = first(event.FingerprintEventIDs)
	return decision
}

func structureEvidenceRef(record types.EvidenceRecord) types.StructureEvidenceRef {
	return types.StructureEvidenceRef{EvidenceID: record.ID, ChapterID: record.ChapterID, ChapterIndex: record.ChapterIndex, Section: record.Section, SectionIndex: record.SectionIndex, ParagraphIndex: record.ParagraphIndex, SentenceIndex: record.SentenceIndex, StartOffset: record.StartOffset, EndOffset: record.EndOffset, Quotation: record.Text, Confidence: record.Confidence, Status: record.Status, Source: record.Source}
}

func appendEvidenceRefs(existing, additions []types.StructureEvidenceRef) []types.StructureEvidenceRef {
	seen := map[string]bool{}
	for _, ref := range existing {
		seen[ref.EvidenceID] = true
	}
	for _, ref := range additions {
		if ref.EvidenceID != "" && !seen[ref.EvidenceID] {
			existing = append(existing, ref)
			seen[ref.EvidenceID] = true
		}
	}
	return existing
}

func matchingStructureDecisions(event types.SignificantStoryEvent, decisions []types.StructureAuthorDecision) []types.StructureAuthorDecision {
	result := []types.StructureAuthorDecision{}
	for _, decision := range decisions {
		if decision.Status != "active" || (decision.TargetType != "" && decision.TargetType != "significant_event") {
			continue
		}
		if decision.TargetID != "" && contains(event.FingerprintEventIDs, decision.TargetID) || stringOverlap(event.EvidenceIDs, decision.EvidenceIDs) {
			result = append(result, decision)
		}
	}
	return result
}

func applyStructureAuthorDecisions(event *types.SignificantStoryEvent, decisions []types.StructureAuthorDecision) {
	for _, decision := range matchingStructureDecisions(*event, decisions) {
		event.AuthorDecisionIDs = appendUnique(event.AuthorDecisionIDs, decision.ID)
		switch decision.Action {
		case "confirm":
			event.InterpretationStatus = "confirmed"
		case "correct":
			event.InterpretationStatus = "corrected"
			if decision.Field == "summary" && strings.TrimSpace(decision.Value) != "" {
				event.AuthorSummary = strings.TrimSpace(decision.Value)
				event.Summary = event.AuthorSummary
			}
		case "unresolved":
			event.InterpretationStatus = "unresolved"
		case "intentional_ambiguity":
			event.InterpretationStatus = "intentional_ambiguity"
		}
	}
}

func applyFingerprintCorrectionLinks(event *types.SignificantStoryEvent, corrections []types.FingerprintCorrection) {
	for _, correction := range corrections {
		if correction.Status != "active" {
			continue
		}
		if contains(event.FingerprintEventIDs, correction.TargetID) || stringOverlap(event.EvidenceIDs, correction.EvidenceIDs) {
			event.CorrectionIDs = appendUnique(event.CorrectionIDs, correction.ID)
		}
	}
}

func temporalPlacementDecision(event types.FingerprintEvent) types.StructureDecision {
	signals := []types.StructureSignal{}
	outcome, rule, summary, confidence := "unresolved", "missing-temporal-context", "The manuscript does not establish a resolved story-time placement for this occurrence.", event.StoryTime.Confidence
	if event.StoryTime.DayOffset != nil {
		outcome, rule, summary = "placed", "resolved-day-offset", "The fingerprint provides a resolved relative story-day placement."
		signals = append(signals, types.StructureSignal{Code: "day-offset", Detail: event.StoryTime.Label, EvidenceIDs: clone(event.EvidenceIDs)})
	} else if event.StoryTime.Precision == "relative" || event.StoryTime.EarliestDay != nil || event.StoryTime.LatestDay != nil {
		outcome, rule, summary = "relative", "relative-temporal-placement", "The evidence establishes relative or bounded order without an exact story day."
	} else if event.ContextID != "" {
		outcome, rule, summary = "context-only", "temporal-context-only", "The occurrence has a reality or discourse context but no resolved chronology."
	}
	return structureDecision("temporal_placement", outcome, rule, summary, 0, 0, confidence, signals)
}

func interpretationStatus(event types.FingerprintEvent, contradiction bool) string {
	if contradiction {
		return "conflicting_evidence"
	}
	if event.Confidence < .6 {
		return "low_confidence"
	}
	if event.ContextID == "" {
		return "missing_context"
	}
	return "inferred"
}

func interpretationSignals(event types.FingerprintEvent, contradiction bool) []types.StructureSignal {
	result := []types.StructureSignal{}
	if contradiction {
		result = append(result, types.StructureSignal{Code: "conflicting-evidence", Detail: "A current fingerprint diagnostic identifies conflicting evidence for this occurrence.", EvidenceIDs: clone(event.EvidenceIDs)})
	}
	if event.Confidence < .6 {
		result = append(result, types.StructureSignal{Code: "low-confidence-extraction", Detail: "The source extraction is below the engine's review-confidence boundary.", Weight: event.Confidence, EvidenceIDs: clone(event.EvidenceIDs)})
	}
	if event.ContextID == "" {
		result = append(result, types.StructureSignal{Code: "missing-context", Detail: "No temporal or reality context could be established from the available prose.", EvidenceIDs: clone(event.EvidenceIDs)})
	}
	if event.StoryTime.Precision == "unknown" && event.StoryTime.DayOffset == nil {
		result = append(result, types.StructureSignal{Code: "unresolved-story-time", Detail: "The manuscript does not currently establish an exact or bounded story-time placement.", EvidenceIDs: clone(event.EvidenceIDs)})
	}
	return result
}

func mergeInterpretationStatus(left, right string) string {
	priority := map[string]int{"": 0, "inferred": 1, "missing_context": 2, "low_confidence": 3, "unresolved": 4, "intentional_ambiguity": 5, "conflicting_evidence": 6, "confirmed": 7, "corrected": 8}
	if priority[right] > priority[left] {
		return right
	}
	return left
}

func significantActions(event types.SignificantStoryEvent, records map[string]types.EvidenceRecord) []string {
	result := []string{}
	for _, id := range event.EvidenceIDs {
		action := strings.ToLower(normalizeAction(records[id].Action))
		if action != "" {
			result = appendUnique(result, action)
		}
	}
	return result
}

func sourceParagraphDistance(left, right types.SignificantStoryEvent) int {
	if left.ChapterID != right.ChapterID {
		return math.MaxInt
	}
	if left.ParagraphEnd < right.ParagraphStart {
		return right.ParagraphStart - left.ParagraphEnd
	}
	if right.ParagraphEnd < left.ParagraphStart {
		return left.ParagraphStart - right.ParagraphEnd
	}
	return 0
}

func incompatibleStoryDays(left, right types.StoryTime) bool {
	return left.DayOffset != nil && right.DayOffset != nil && *left.DayOffset != *right.DayOffset
}

func sameResolvedStoryTime(left, right types.StoryTime) bool {
	return left.DayOffset != nil && right.DayOffset != nil && *left.DayOffset == *right.DayOffset
}

func salienceSignals(event types.SignificantStoryEvent) []types.StructureSignal {
	result := []types.StructureSignal{}
	add := func(code, detail string, weight float64) {
		if weight != 0 {
			result = append(result, types.StructureSignal{Code: code, Detail: detail, Weight: weight, EvidenceIDs: clone(event.EvidenceIDs)})
		}
	}
	r := event.SalienceReasons
	add("base-importance", "Maximum source-event importance contribution.", r.Base)
	add("persistent-state", "Persistent state-change contribution.", r.StateChange)
	add("movement", "Movement or transition contribution.", r.Movement)
	add("decision", "Decision-language contribution.", r.Decision)
	add("discovery", "Discovery contribution.", r.Discovery)
	add("temporal-change", "Temporal-placement contribution.", r.TemporalChange)
	add("thread-change", "Narrative-obligation contribution.", r.ThreadChange)
	add("later-references", "Later exact-term references contribution.", r.LaterReferences)
	add("contradiction", "Conflicting-evidence contribution.", r.Contradiction)
	add("description-only", "Descriptive-only penalty.", -r.DescriptionOnly)
	add("low-confidence", "Low-confidence extraction penalty.", -r.LowConfidence)
	return result
}
