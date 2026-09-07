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

// promoteNarrativeFingerprints applies precision-first narrative promotion.
// It starts with independently defensible state changes, then revisits all
// assertions for later contradiction, fulfillment, corroboration and causal
// dependence. Unpromoted assertions and their evidence remain intact.
func promoteNarrativeFingerprints(assertions []types.StoryAssertion, records []types.EvidenceRecord) ([]types.NarrativeFingerprint, []types.NarrativeFingerprintRelation) {
	recordByID := evidenceRecordMap(records)
	reasons := map[string][]types.NarrativePromotionReason{}
	for _, assertion := range assertions {
		reasons[assertion.ID] = initialPromotionReasons(assertion, recordByID)
	}
	relations := inferAssertionRelations(assertions, recordByID)
	for _, relation := range relations {
		switch relation.kind {
		case "contradicts", "supersedes", "corroborates", "fulfills", "enables", "setup_for":
			reasons[relation.from] = appendPromotionReason(reasons[relation.from], relationReason(relation, relation.from))
			reasons[relation.to] = appendPromotionReason(reasons[relation.to], relationReason(relation, relation.to))
		}
	}

	fingerprints := make([]types.NarrativeFingerprint, 0, len(reasons))
	assertionToFingerprint := map[string]string{}
	for _, assertion := range assertions {
		if len(reasons[assertion.ID]) == 0 {
			continue
		}
		fingerprint := fingerprintFromAssertion(assertion, reasons[assertion.ID], recordByID)
		fingerprints = append(fingerprints, fingerprint)
		assertionToFingerprint[assertion.ID] = fingerprint.ID
	}

	resultRelations := make([]types.NarrativeFingerprintRelation, 0, len(relations))
	seen := map[string]bool{}
	for _, relation := range relations {
		from, fromOK := assertionToFingerprint[relation.from]
		to, toOK := assertionToFingerprint[relation.to]
		if !fromOK || !toOK || from == to {
			continue
		}
		key := from + "\x00" + to + "\x00" + relation.kind
		if seen[key] {
			continue
		}
		seen[key] = true
		resultRelations = append(resultRelations, types.NarrativeFingerprintRelation{
			ID: stableID("narrative-relation", key), FromID: from, ToID: to, Kind: relation.kind,
			Explanation: relation.explain, EvidenceIDs: clone(relation.evidence), Confidence: relation.confidence,
		})
		if relation.kind == "contradicts" {
			markFingerprintStatus(fingerprints, from, "contradicted")
		}
		if relation.kind == "supersedes" {
			markFingerprintStatus(fingerprints, from, "superseded")
		}
	}

	sort.SliceStable(fingerprints, func(i, j int) bool {
		return spanOrder(fingerprints[i].EvidenceSpans) < spanOrder(fingerprints[j].EvidenceSpans)
	})
	sort.SliceStable(resultRelations, func(i, j int) bool {
		if resultRelations[i].FromID == resultRelations[j].FromID {
			return resultRelations[i].ToID < resultRelations[j].ToID
		}
		return resultRelations[i].FromID < resultRelations[j].FromID
	})
	return fingerprints, resultRelations
}

func initialPromotionReasons(assertion types.StoryAssertion, records map[string]types.EvidenceRecord) []types.NarrativePromotionReason {
	result := []types.NarrativePromotionReason{}
	for _, id := range assertion.EvidenceIDs {
		record := records[id]
		if record.Pinned || record.Source == "author" || record.Status == "confirmed" {
			result = appendPromotionReason(result, promotionReason("author_confirmed", "The author explicitly confirmed or pinned the supporting evidence.", assertion, .99))
			break
		}
	}
	switch assertion.Kind {
	case "commitment":
		if meaningfulClause(assertion.Object) {
			result = appendPromotionReason(result, promotionReason("commitment_created", "The passage establishes a durable promise, agreement, or obligation.", assertion, .92))
		}
	case "goal":
		if decisiveGoal(assertion) {
			result = appendPromotionReason(result, promotionReason("goal_or_plan_changed", "The passage establishes or changes a character's intended course of action.", assertion, .88))
		}
	case "relationship":
		result = appendPromotionReason(result, promotionReason("relationship_changed", "The passage explicitly changes a relationship between story entities.", assertion, .9))
	case "state":
		if assertion.StateChange != nil && assertion.StateChange.Operation == "change" && assertion.Persistence == "persistent" && objectiveAssertion(assertion) && stableStateSubject(assertion.StateChange) {
			result = appendPromotionReason(result, promotionReason("persistent_state_changed", "The passage changes a persistent character or world state.", assertion, .91))
		}
	case "knowledge":
		if consequentialKnowledge(assertion, records) {
			result = appendPromotionReason(result, promotionReason("consequential_knowledge_changed", "The passage explicitly establishes a substantive discovery or knowledge change.", assertion, .84))
		}
	case "claim":
		if consequentialKnowledgeTransfer(assertion, records) {
			result = appendPromotionReason(result, promotionReason("consequential_knowledge_transfer", "A character deliberately transfers substantive information to another character without making it objective world truth.", assertion, .86))
		}
	}
	if assertion.EpistemicStatus == "deliberate_deception" && strings.TrimSpace(assertion.Object) != "" {
		result = appendPromotionReason(result, promotionReason("identifiable_deception", "The manuscript explicitly identifies an attributed proposition as deception.", assertion, .95))
	}
	return result
}

func consequentialKnowledgeTransfer(assertion types.StoryAssertion, records map[string]types.EvidenceRecord) bool {
	if assertion.EpistemicStatus != "attributed_claim" || len(semanticTerms(assertion.Subject+" "+assertion.Object)) < 2 {
		return false
	}
	for _, id := range assertion.EvidenceIDs {
		record := records[id]
		for _, knowledge := range record.KnowledgeStates {
			if knowledge.State == "shared" && len(knowledge.CounterpartyIDs) > 0 {
				return true
			}
		}
		cue := strings.ToLower(assertion.Attribution.Cue)
		if len(record.CharacterIDs) >= 2 && (cue == "warned" || cue == "confessed" || cue == "admitted") {
			return true
		}
	}
	return false
}

func decisiveGoal(assertion types.StoryAssertion) bool {
	if !meaningfulClause(assertion.Object) {
		return false
	}
	switch assertion.Predicate {
	case "decides", "plans", "ordered", "orders":
		return true
	default:
		return false
	}
}

func consequentialKnowledge(assertion types.StoryAssertion, records map[string]types.EvidenceRecord) bool {
	if assertion.Predicate != "discovers" || !meaningfulClause(assertion.Object) {
		return false
	}
	if len(semanticTerms(assertion.Object)) < 2 || routineLogisticsRe.MatchString(assertion.Statement) {
		return false
	}
	for _, id := range assertion.EvidenceIDs {
		record := records[id]
		if len(nonParticipantTerms(record, assertion.Subject)) > 0 {
			return true
		}
	}
	words := semanticTerms(assertion.Object)
	return len(words) >= 4 && containsStructuralRelation(assertion.Object)
}

func objectiveAssertion(assertion types.StoryAssertion) bool {
	return assertion.EpistemicStatus == "world_state_fact" || assertion.EpistemicStatus == "externally_corroborated_fact"
}

func stableStateSubject(change *types.NarrativeStateChange) bool {
	if change == nil {
		return false
	}
	if change.EntityID != "" {
		return true
	}
	words := strings.Fields(normalizeSemantic(change.EntityName))
	if len(words) == 0 {
		return false
	}
	switch words[0] {
	case "he", "she", "they", "it", "you", "i", "we", "this", "that", "someone", "something":
		return false
	}
	return true
}

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

func fingerprintFromAssertion(assertion types.StoryAssertion, reasons []types.NarrativePromotionReason, records map[string]types.EvidenceRecord) types.NarrativeFingerprint {
	spans := make([]types.NarrativeEvidenceSpan, 0, len(assertion.EvidenceIDs))
	participants := []types.NarrativeParticipant{}
	for _, id := range assertion.EvidenceIDs {
		if record, exists := records[id]; exists {
			spans = append(spans, evidenceSpan(record))
			for index, name := range record.CharacterNames {
				entityID := ""
				if index < len(record.CharacterIDs) {
					entityID = record.CharacterIDs[index]
				}
				role := "mentioned"
				if entityID != "" && entityID == assertion.SubjectID {
					role = "subject"
				}
				participants = appendParticipant(participants, types.NarrativeParticipant{EntityID: entityID, EntityName: name, Role: role})
			}
		}
	}
	if assertion.Attribution.Kind == "character" {
		participants = appendParticipant(participants, types.NarrativeParticipant{EntityID: assertion.Attribution.EntityID, EntityName: assertion.Attribution.EntityName, Role: "source"})
	}
	confidence := assertion.Confidence
	for _, reason := range reasons {
		if reason.Confidence < confidence {
			confidence = reason.Confidence
		}
	}
	return types.NarrativeFingerprint{
		ID:   stableID("narrative-fingerprint", assertion.SemanticKey, assertion.Scope.ID, assertion.EpistemicStatus, assertion.Attribution.EntityID),
		Kind: assertion.Kind, Summary: assertion.Statement, SemanticKey: assertion.SemanticKey,
		AssertionIDs: []string{assertion.ID}, EvidenceIDs: clone(assertion.EvidenceIDs), EvidenceSpans: spans,
		Participants: participants, StateChange: assertion.StateChange, EpistemicStatus: assertion.EpistemicStatus,
		Attribution: assertion.Attribution, Scope: assertion.Scope, Persistence: assertion.Persistence, Temporal: assertion.Temporal,
		PromotionReasons: reasons, Confidence: confidence, Status: assertion.Status,
	}
}

func evidenceSpan(record types.EvidenceRecord) types.NarrativeEvidenceSpan {
	return types.NarrativeEvidenceSpan{EvidenceID: record.ID, ChapterID: record.ChapterID, ChapterIndex: record.ChapterIndex, Section: record.Section, SectionIndex: record.SectionIndex, ParagraphIndex: record.ParagraphIndex, SentenceIndex: record.SentenceIndex, StartOffset: record.StartOffset, EndOffset: record.EndOffset, Quote: record.Text, Confidence: record.Confidence}
}

func promotionReason(code, explanation string, assertion types.StoryAssertion, confidence float64) types.NarrativePromotionReason {
	return types.NarrativePromotionReason{Code: code, Explanation: explanation, AssertionIDs: []string{assertion.ID}, EvidenceIDs: clone(assertion.EvidenceIDs), Confidence: confidence}
}

func relationReason(relation assertionRelation, assertionID string) types.NarrativePromotionReason {
	other := relation.to
	if assertionID == relation.to {
		other = relation.from
	}
	return types.NarrativePromotionReason{Code: "retrospective_" + relation.kind, Explanation: relation.explain, AssertionIDs: []string{assertionID, other}, EvidenceIDs: clone(relation.evidence), Confidence: relation.confidence}
}

func appendPromotionReason(values []types.NarrativePromotionReason, addition types.NarrativePromotionReason) []types.NarrativePromotionReason {
	for _, value := range values {
		if value.Code == addition.Code && strings.Join(value.AssertionIDs, "|") == strings.Join(addition.AssertionIDs, "|") {
			return values
		}
	}
	return append(values, addition)
}

func evidenceRecordMap(records []types.EvidenceRecord) map[string]types.EvidenceRecord {
	result := make(map[string]types.EvidenceRecord, len(records))
	for _, record := range records {
		result[record.ID] = record
	}
	return result
}

func meaningfulClause(value string) bool { return len(semanticTerms(value)) >= 2 }

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

func containsStructuralRelation(value string) bool {
	lower := strings.ToLower(value)
	for _, cue := range []string{" was ", " is ", " were ", " are ", " had ", " has ", " caused ", " requires ", " required ", " beneath ", " inside ", " behind "} {
		if strings.Contains(" "+lower+" ", cue) {
			return true
		}
	}
	return false
}

func nonParticipantTerms(record types.EvidenceRecord, subject string) []types.EvidenceTerm {
	result := []types.EvidenceTerm{}
	for _, term := range record.NamedEntities {
		if !strings.EqualFold(term.Text, subject) {
			result = append(result, term)
		}
	}
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

func markFingerprintStatus(values []types.NarrativeFingerprint, id, status string) {
	for index := range values {
		if values[index].ID == id {
			values[index].Status = status
			return
		}
	}
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
