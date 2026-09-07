package fingerprint

import (
	"sort"
	"strings"

	"draftline/internal/types"
)

type assertionRelation struct {
	from, to   string
	kind       string
	explain    string
	evidence   []string
	confidence float64
}

var semanticStopWords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true, "be": true, "been": true,
	"but": true, "by": true, "for": true, "from": true, "had": true, "has": true, "have": true, "he": true,
	"her": true, "hers": true, "him": true, "his": true, "i": true, "in": true, "is": true, "it": true,
	"its": true, "of": true, "on": true, "or": true, "she": true, "that": true, "the": true, "their": true,
	"them": true, "they": true, "this": true, "to": true, "was": true, "were": true, "with": true, "you": true,
	"didn": true, "doesn": true, "wasn": true, "couldn": true, "wouldn": true, "answer": true,
}

// inferAssertionRelations builds semantic relationships across the complete
// assertion corpus. These relationships support memory, state histories,
// developments, and inspections; they do not promote or discard assertions.
func inferAssertionRelations(assertions []types.StoryAssertion, records map[string]types.EvidenceRecord) []assertionRelation {
	result := []assertionRelation{}
	statePrior := map[string][]int{}
	commitmentByTerm := map[string][]int{}
	setupByTerm := map[string][]int{}
	uncertainByTerm := map[string][]int{}
	for index, right := range assertions {
		if key := comparableStateKey(right); key != "" {
			for _, earlier := range statePrior[key] {
				if relation, ok := contradictionRelation(assertions[earlier], right); ok {
					result = append(result, relation)
				}
			}
		}
		if completionCueRe.MatchString(right.Statement) {
			for _, earlier := range indexedCandidates(commitmentByTerm, relationTerms(right)) {
				if relation, ok := fulfillmentRelation(assertions[earlier], right); ok {
					result = append(result, relation)
				}
			}
		}
		if useCueRe.MatchString(right.Statement) || explicitDependencyLanguage(right.Statement) {
			for _, earlier := range indexedCandidates(setupByTerm, relationTerms(right)) {
				if relation, ok := enablingRelation(assertions[earlier], right); ok {
					result = append(result, relation)
				}
			}
		}
		if right.EpistemicStatus == "externally_corroborated_fact" {
			for _, earlier := range indexedCandidates(uncertainByTerm, relationTerms(right)) {
				if relation, ok := corroborationRelation(assertions[earlier], right); ok {
					result = append(result, relation)
				}
			}
		}

		if key := comparableStateKey(right); key != "" {
			statePrior[key] = append(statePrior[key], index)
		}
		if right.Kind == "commitment" {
			indexAssertionTerms(commitmentByTerm, index, right)
		}
		if isPotentialSetup(right) {
			indexAssertionTerms(setupByTerm, index, right)
		}
		switch right.EpistemicStatus {
		case "attributed_claim", "character_belief", "character_inference", "uncertain_interpretation":
			indexAssertionTerms(uncertainByTerm, index, right)
		}
	}
	return dedupeAssertionRelations(result)
}

func contradictionRelation(left, right types.StoryAssertion) (assertionRelation, bool) {
	if !compatibleScopes(left.Scope, right.Scope) {
		return assertionRelation{}, false
	}
	if !sameAssertionSubject(left, right) {
		return assertionRelation{}, false
	}
	leftState, rightState := stateComparisonKey(left), stateComparisonKey(right)
	if leftState == "" || leftState != rightState {
		return assertionRelation{}, false
	}
	differentValue := normalizeSemantic(assertionValue(left)) != normalizeSemantic(assertionValue(right))
	oppositePolarity := left.Polarity != right.Polarity && left.Polarity != "uncertain" && right.Polarity != "uncertain"
	if !differentValue && !oppositePolarity {
		return assertionRelation{}, false
	}
	kind := "contradicts"
	explain := "Later evidence establishes an incompatible value for the same subject and state."
	if correctiveLanguage(right.Statement) {
		kind = "supersedes"
		explain = "The later passage explicitly corrects or replaces the earlier interpretation."
	}
	return assertionRelation{from: left.ID, to: right.ID, kind: kind, explain: explain, evidence: appendUnique(clone(left.EvidenceIDs), right.EvidenceIDs...), confidence: .88}, true
}

func fulfillmentRelation(left, right types.StoryAssertion) (assertionRelation, bool) {
	if left.Kind != "commitment" || !completionCueRe.MatchString(right.Statement) {
		return assertionRelation{}, false
	}
	leftTerms, rightTerms := relationTerms(left), relationTerms(right)
	if sharedTermCount(leftTerms, rightTerms) < 2 || semanticOverlap(leftTerms, rightTerms) < .34 {
		return assertionRelation{}, false
	}
	return assertionRelation{from: left.ID, to: right.ID, kind: "fulfills", explain: "The later state change satisfies the earlier promise or obligation.", evidence: appendUnique(clone(left.EvidenceIDs), right.EvidenceIDs...), confidence: .84}, true
}

func enablingRelation(left, right types.StoryAssertion) (assertionRelation, bool) {
	isAcquiredObject := left.StateChange != nil && left.StateChange.StateKind == "possession" && left.StateChange.Operation == "acquire"
	isScopedInformation := left.Scope.Kind == "dream" || left.Scope.Kind == "vision" || left.Scope.Kind == "remembered"
	isExplicitCause := explicitDependencyLanguage(right.Statement) && (left.Persistence == "persistent" || left.Persistence == "conditional")
	if !isAcquiredObject && !isScopedInformation && !isExplicitCause {
		return assertionRelation{}, false
	}
	if !useCueRe.MatchString(right.Statement) && !explicitDependencyLanguage(right.Statement) {
		return assertionRelation{}, false
	}
	leftTerms, rightTerms := relationTerms(left), relationTerms(right)
	if sharedTermCount(leftTerms, rightTerms) < 2 || semanticOverlap(leftTerms, rightTerms) < .5 {
		return assertionRelation{}, false
	}
	return assertionRelation{from: left.ID, to: right.ID, kind: "enables", explain: "A retained earlier state or object is explicitly used by the later occurrence.", evidence: appendUnique(clone(left.EvidenceIDs), right.EvidenceIDs...), confidence: .86}, true
}

func corroborationRelation(left, right types.StoryAssertion) (assertionRelation, bool) {
	if left.EpistemicStatus != "attributed_claim" && left.EpistemicStatus != "character_belief" && left.EpistemicStatus != "character_inference" {
		return assertionRelation{}, false
	}
	leftTerms, rightTerms := relationTerms(left), relationTerms(right)
	if right.EpistemicStatus != "externally_corroborated_fact" || sharedTermCount(leftTerms, rightTerms) < 2 || semanticOverlap(leftTerms, rightTerms) < .5 {
		return assertionRelation{}, false
	}
	return assertionRelation{from: left.ID, to: right.ID, kind: "corroborates", explain: "Later external evidence supports an earlier attributed or uncertain proposition.", evidence: appendUnique(clone(left.EvidenceIDs), right.EvidenceIDs...), confidence: .9}, true
}

func evidenceSpan(record types.EvidenceRecord) types.NarrativeEvidenceSpan {
	return types.NarrativeEvidenceSpan{EvidenceID: record.ID, ChapterID: record.ChapterID, ChapterIndex: record.ChapterIndex, Section: record.Section, SectionIndex: record.SectionIndex, ParagraphIndex: record.ParagraphIndex, SentenceIndex: record.SentenceIndex, StartOffset: record.StartOffset, EndOffset: record.EndOffset, Quote: record.Text, Confidence: record.Confidence}
}

func evidenceRecordMap(records []types.EvidenceRecord) map[string]types.EvidenceRecord {
	result := make(map[string]types.EvidenceRecord, len(records))
	for _, record := range records {
		result[record.ID] = record
	}
	return result
}

func semanticTerms(value string) map[string]bool {
	result := map[string]bool{}
	for _, word := range strings.Fields(normalizeSemantic(value)) {
		if len(word) > 2 && !semanticStopWords[word] {
			result[word] = true
		}
	}
	return result
}

func relationTerms(assertion types.StoryAssertion) map[string]bool {
	return semanticTerms(assertion.Object + " " + stateValue(assertion))
}

func semanticOverlap(left, right map[string]bool) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	common := 0
	for value := range left {
		if right[value] {
			common++
		}
	}
	denominator := min(len(left), len(right))
	return float64(common) / float64(denominator)
}

func sharedTermCount(left, right map[string]bool) int {
	common := 0
	for value := range left {
		if right[value] {
			common++
		}
	}
	return common
}

func sameAssertionSubject(left, right types.StoryAssertion) bool {
	if left.SubjectID != "" && right.SubjectID != "" {
		return left.SubjectID == right.SubjectID
	}
	return normalizeSemantic(left.Subject) != "" && normalizeSemantic(left.Subject) == normalizeSemantic(right.Subject)
}

func stateComparisonKey(assertion types.StoryAssertion) string {
	if assertion.StateChange != nil {
		return normalizeSemantic(assertion.StateChange.StateKind)
	}
	if assertion.Kind == "claim" || assertion.Kind == "knowledge" || assertion.Kind == "state" {
		return normalizeSemantic(assertion.Predicate)
	}
	return ""
}

func comparableStateKey(assertion types.StoryAssertion) string {
	subject := assertion.SubjectID
	if subject == "" {
		subject = normalizeSemantic(assertion.Subject)
	}
	if subject == "" {
		return ""
	}
	if !stableAssertionSubject(assertion) {
		return ""
	}
	if assertion.StateChange != nil && assertion.Persistence == "persistent" {
		switch assertion.StateChange.StateKind {
		case "attribute", "hair", "eye", "eyes", "name", "rank", "title", "life_status", "physical_condition", "world_object_condition":
			return assertion.Scope.ID + "\x00" + subject + "\x00" + assertion.StateChange.StateKind
		}
	}
	if assertion.Predicate == "is" {
		switch assertion.EpistemicStatus {
		case "attributed_claim", "character_belief", "character_inference", "uncertain_interpretation":
			return assertion.Scope.ID + "\x00" + subject + "\x00is"
		case "world_state_fact", "externally_corroborated_fact":
			if correctiveLanguage(assertion.Statement) {
				return assertion.Scope.ID + "\x00" + subject + "\x00is"
			}
		}
	}
	return ""
}

func stableAssertionSubject(assertion types.StoryAssertion) bool {
	if assertion.SubjectID != "" {
		return true
	}
	words := strings.Fields(normalizeSemantic(assertion.Subject))
	if len(words) == 0 || len(words) > 8 {
		return false
	}
	switch words[0] {
	case "this", "that", "these", "those", "here", "there", "it", "he", "she", "they", "we", "you", "i", "something", "anything", "everything", "nothing", "someone", "anyone":
		return false
	}
	return true
}

func assertionValue(assertion types.StoryAssertion) string {
	if assertion.StateChange != nil {
		return assertion.StateChange.New
	}
	return assertion.Object
}

func stateValue(assertion types.StoryAssertion) string {
	if assertion.StateChange == nil {
		return ""
	}
	return assertion.StateChange.Previous + " " + assertion.StateChange.New
}

func compatibleScopes(left, right types.NarrativeRealityScope) bool {
	if left.ID == right.ID {
		return true
	}
	return left.Kind == "current" && right.Kind == "current"
}

func correctiveLanguage(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "actually") || strings.Contains(lower, "in fact") || strings.Contains(lower, "turned out") || strings.Contains(lower, "was wrong") || strings.Contains(lower, "instead")
}

func explicitDependencyLanguage(value string) bool {
	lower := strings.ToLower(value)
	for _, cue := range []string{"because of", "thanks to", "enabled", "allowed", "depended on", "therefore", "as a result"} {
		if strings.Contains(lower, cue) {
			return true
		}
	}
	return false
}

func isPotentialSetup(assertion types.StoryAssertion) bool {
	if len(relationTerms(assertion)) == 0 {
		return false
	}
	if assertion.StateChange != nil && assertion.StateChange.StateKind == "possession" && assertion.StateChange.Operation == "acquire" {
		return true
	}
	if assertion.Scope.Kind == "dream" || assertion.Scope.Kind == "vision" || assertion.Scope.Kind == "remembered" {
		return true
	}
	return assertion.Persistence == "persistent" || assertion.Persistence == "conditional"
}

func indexAssertionTerms(index map[string][]int, position int, assertion types.StoryAssertion) {
	for term := range relationTerms(assertion) {
		index[term] = append(index[term], position)
	}
}

func indexedCandidates(index map[string][]int, terms map[string]bool) []int {
	seen := map[int]bool{}
	result := []int{}
	for term := range terms {
		for _, position := range index[term] {
			if !seen[position] {
				seen[position] = true
				result = append(result, position)
			}
		}
	}
	sort.Ints(result)
	return result
}

func appendParticipant(values []types.NarrativeParticipant, addition types.NarrativeParticipant) []types.NarrativeParticipant {
	if addition.EntityName == "" {
		return values
	}
	for index := range values {
		if values[index].EntityID != "" && values[index].EntityID == addition.EntityID {
			if values[index].Role == "mentioned" && addition.Role != "mentioned" {
				values[index].Role = addition.Role
			}
			return values
		}
		if values[index].EntityID == "" && strings.EqualFold(values[index].EntityName, addition.EntityName) {
			return values
		}
	}
	return append(values, addition)
}

func dedupeAssertionRelations(values []assertionRelation) []assertionRelation {
	seen := map[string]bool{}
	result := make([]assertionRelation, 0, len(values))
	for _, value := range values {
		key := value.from + "\x00" + value.to + "\x00" + value.kind
		if !seen[key] {
			seen[key] = true
			result = append(result, value)
		}
	}
	return result
}

func spanOrder(spans []types.NarrativeEvidenceSpan) int {
	if len(spans) == 0 {
		return int(^uint(0) >> 1)
	}
	return spans[0].ChapterIndex*1_000_000 + spans[0].StartOffset
}
