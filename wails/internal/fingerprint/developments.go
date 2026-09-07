package fingerprint

import (
	"regexp"
	"sort"
	"strings"

	"draftline/internal/types"
)

var (
	obstacleDevelopmentRe = regexp.MustCompile(`(?i)\b(blocked|prevented|trapped|denied|refused|lost access|cut off)\b`)
	threatDevelopmentRe   = regexp.MustCompile(`(?i)\b(threatened|attacked|ambushed|hunted|issued (?:an )?ultimatum|pursued|began (?:a )?siege|invaded)\b`)
	resultDiscoveryRe     = regexp.MustCompile(`(?i)\b(discovered|uncovered|learned|learnt|realized|determined|revealed|found)\b`)
	incidentalDiscoveryRe = regexp.MustCompile(`(?i)\b(found (?:himself|herself|themselves|it hard|no |a position)|trying to find|tried to find|needed to find|wishing .* find|couldn['’]t find|didn['’]t find)\b`)
)

type developmentCandidate struct {
	kind, before, after, concern string
	fingerprintIDs, evidenceIDs  []string
	participants                 []types.NarrativeParticipant
	scope                        types.NarrativeRealityScope
	temporal                     types.StoryTime
	chapterStart, chapterEnd     int
	paragraphStart, paragraphEnd int
	reasons                      []types.NarrativeDevelopmentReason
	confidence                   float64
	fromRelation                 bool
}

func synthesizeNarrativeDevelopments(fingerprints []types.ManuscriptFingerprint, relations []types.ManuscriptFingerprintRelation, histories []types.FingerprintStateHistory, records []types.EvidenceRecord) []types.NarrativeDevelopment {
	byID := make(map[string]types.ManuscriptFingerprint, len(fingerprints))
	for _, fingerprint := range fingerprints {
		byID[fingerprint.ID] = fingerprint
	}
	relationMembers := map[string]bool{}
	candidates := []developmentCandidate{}
	for _, relation := range relations {
		left, leftOK := byID[relation.FromID]
		right, rightOK := byID[relation.ToID]
		if !leftOK || !rightOK || !developmentRelationForFingerprints(relation.Kind, left, right) {
			continue
		}
		relationMembers[left.ID], relationMembers[right.ID] = true, true
		candidates = append(candidates, candidateFromRelation(relation, left, right))
	}
	recordByID := evidenceRecordMap(records)
	for _, fingerprint := range fingerprints {
		if candidate, ok := candidateFromFingerprint(fingerprint, fingerprints, histories, recordByID); ok {
			if relationMembers[fingerprint.ID] {
				candidate.reasons = append(candidate.reasons, types.NarrativeDevelopmentReason{
					Code: "also_independently_meaningful", Explanation: "The state transition is meaningful independently of its relationship to another fingerprint.",
					FingerprintIDs: []string{fingerprint.ID}, EvidenceIDs: clone(fingerprint.EvidenceIDs), Confidence: fingerprint.Confidence,
				})
			}
			candidate = attachDevelopmentContext(candidate, fingerprints)
			candidates = append(candidates, candidate)
		}
	}
	candidates = mergeDevelopmentCandidates(candidates)
	result := make([]types.NarrativeDevelopment, 0, len(candidates))
	fingerprintToDevelopment := map[string]string{}
	for _, candidate := range candidates {
		development := developmentFromCandidate(candidate, byID)
		result = append(result, development)
		for _, fingerprintID := range development.FingerprintIDs {
			fingerprintToDevelopment[fingerprintID] = development.ID
		}
	}
	for _, relation := range relations {
		from, fromOK := fingerprintToDevelopment[relation.FromID]
		to, toOK := fingerprintToDevelopment[relation.ToID]
		if fromOK && toOK && from != to {
			for index := range result {
				if result[index].ID == to {
					result[index].DependencyIDs = appendUnique(result[index].DependencyIDs, from)
				}
			}
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ChapterStart != result[j].ChapterStart {
			return result[i].ChapterStart < result[j].ChapterStart
		}
		return firstDevelopmentOffset(result[i], byID) < firstDevelopmentOffset(result[j], byID)
	})
	return result
}

func developmentRelation(kind string) bool {
	switch kind {
	case "enables", "fulfills", "contradicts", "supersedes", "corroborates":
		return true
	}
	return false
}

func developmentRelationForFingerprints(kind string, left, right types.ManuscriptFingerprint) bool {
	if developmentRelation(kind) {
		return true
	}
	if kind != "same_event" || left.StateChange == nil || right.StateChange == nil {
		return false
	}
	if left.StateChange.StateKind != right.StateChange.StateKind || normalizeSemantic(left.StateChange.New) != normalizeSemantic(right.StateChange.New) {
		return false
	}
	switch left.StateChange.StateKind {
	case "life_status", "physical_condition", "identity", "world_object_condition", "freedom_or_presence":
		return true
	}
	return false
}

func candidateFromRelation(relation types.ManuscriptFingerprintRelation, left, right types.ManuscriptFingerprint) developmentCandidate {
	kind, before, after := relation.Kind, left.Statement, right.Statement
	switch relation.Kind {
	case "enables":
		kind, before, after = "causal_enablement", "Required capability or information was not yet applied", conciseFingerprintState(left)+" enables "+conciseFingerprintState(right)
	case "fulfills":
		kind, before, after = "obligation_fulfilled", "Obligation remained open: "+conciseFingerprintState(left), "Obligation fulfilled: "+conciseFingerprintState(right)
	case "contradicts":
		kind, after = "model_conflict", conciseFingerprintState(right)+" conflicts with "+conciseFingerprintState(left)
	case "supersedes":
		kind, after = "model_revision", conciseFingerprintState(right)+" revises "+conciseFingerprintState(left)
	case "corroborates":
		kind, after = "theory_corroborated", conciseFingerprintState(right)+" supports "+conciseFingerprintState(left)
	case "same_event":
		kind = "model_revelation"
		before = "The consequential event or state was not established by multiple passages"
		change := right.StateChange
		subject := developmentSubject(right, mergeParticipants(left.Participants, right.Participants))
		after = strings.TrimSpace(subject + " " + strings.ReplaceAll(change.StateKind, "_", " ") + ": " + change.New)
	}
	startChapter, startParagraph := fingerprintPosition(left)
	endChapter, endParagraph := fingerprintPosition(right)
	if endChapter < startChapter || (endChapter == startChapter && endParagraph < startParagraph) {
		startChapter, endChapter = endChapter, startChapter
		startParagraph, endParagraph = endParagraph, startParagraph
	}
	return developmentCandidate{
		kind: kind, before: before, after: after, concern: sharedConcern(left, right),
		fingerprintIDs: []string{left.ID, right.ID}, evidenceIDs: appendUnique(clone(left.EvidenceIDs), right.EvidenceIDs...),
		participants: mergeParticipants(left.Participants, right.Participants), scope: right.Scope, temporal: right.Temporal,
		chapterStart: startChapter, chapterEnd: endChapter, paragraphStart: startParagraph, paragraphEnd: endParagraph,
		reasons:    []types.NarrativeDevelopmentReason{{Code: "fingerprint_" + relation.Kind, Explanation: relation.Explanation, FingerprintIDs: []string{left.ID, right.ID}, EvidenceIDs: clone(relation.EvidenceIDs), Confidence: relation.Confidence}},
		confidence: relation.Confidence, fromRelation: true,
	}
}

func candidateFromFingerprint(fingerprint types.ManuscriptFingerprint, all []types.ManuscriptFingerprint, histories []types.FingerprintStateHistory, records map[string]types.EvidenceRecord) (developmentCandidate, bool) {
	kind, before, after, concern, reason := "", "", "", "", ""
	if fingerprint.StateChange != nil {
		change := fingerprint.StateChange
		switch change.StateKind {
		case "goal":
			if stableHistoryEntity(change.EntityID, change.EntityName) && meaningfulDevelopmentObject(change.New) {
				kind, after, concern, reason = "objective_change", conciseFingerprintState(fingerprint), change.New, "A character adopts, abandons, or materially changes an intended course of action."
			}
		case "commitment":
			kind, after, concern, reason = "obligation_created", conciseFingerprintState(fingerprint), change.New, "The passage creates a promise, deadline, agreement, or obligation that can affect later action."
		case "relationship":
			if stableHistoryEntity(change.EntityID, change.EntityName) && distinctDevelopmentParticipants(fingerprint.Participants) >= 2 {
				kind, after, concern, reason = "relationship_change", conciseFingerprintState(fingerprint), change.New, "The relationship between story entities materially changes."
			}
		case "life_status", "physical_condition", "world_object_condition", "freedom_or_presence":
			if objectiveAssertionFingerprint(fingerprint) && (stableHistoryEntity(change.EntityID, change.EntityName) || consequentialAnonymousState(change.StateKind)) {
				kind, after, concern, reason = "persistent_state_change", conciseFingerprintState(fingerprint), change.StateKind, "A durable character or world condition changes what later action is possible."
			}
		}
		if kind != "" {
			before = priorStateValue(fingerprint, change.StateKind, histories)
			if before == "" {
				before = unstatedPriorState(change.StateKind)
			}
		}
	}
	if kind == "" {
		if discovered, explanation, ok := contextualKnowledgeChange(fingerprint, all, records); ok {
			actor := developmentSubject(fingerprint, fingerprint.Participants)
			kind, before, after, concern, reason = "knowledge_change", "This information was not established for the affected character or reader", strings.TrimSpace(actor+" knows: "+discovered), discovered, explanation
		}
	}
	if kind == "" && objectiveAssertionFingerprint(fingerprint) && obstacleDevelopmentRe.MatchString(fingerprint.Statement) {
		kind, before, after, concern, reason = "obstacle", "The prior course remained available", conciseFingerprintState(fingerprint), fingerprint.Object, "An obstacle frustrates or blocks an existing course of action."
	}
	if kind == "" && objectiveAssertionFingerprint(fingerprint) && threatDevelopmentRe.MatchString(fingerprint.Statement) {
		kind, before, after, concern, reason = "threat_change", "The threat was absent, latent, or less immediate", conciseFingerprintState(fingerprint), fingerprint.Object, "A threat appears, becomes explicit, or escalates."
	}
	if kind == "" {
		return developmentCandidate{}, false
	}
	chapter, paragraph := fingerprintPosition(fingerprint)
	return developmentCandidate{
		kind: kind, before: before, after: after, concern: concern, fingerprintIDs: []string{fingerprint.ID}, evidenceIDs: clone(fingerprint.EvidenceIDs),
		participants: append([]types.NarrativeParticipant(nil), fingerprint.Participants...), scope: fingerprint.Scope, temporal: fingerprint.Temporal,
		chapterStart: chapter, chapterEnd: chapter, paragraphStart: paragraph, paragraphEnd: paragraph,
		reasons:    []types.NarrativeDevelopmentReason{{Code: kind, Explanation: reason, FingerprintIDs: []string{fingerprint.ID}, EvidenceIDs: clone(fingerprint.EvidenceIDs), Confidence: fingerprint.Confidence}},
		confidence: fingerprint.Confidence,
	}, true
}

func consequentialAnonymousState(kind string) bool {
	switch kind {
	case "life_status", "world_object_condition", "freedom_or_presence":
		return true
	}
	return false
}

func unstatedPriorState(kind string) string {
	switch kind {
	case "goal":
		return "No prior version of this objective is established in the current scope"
	case "commitment":
		return "No matching obligation is established before this point"
	case "relationship":
		return "The prior relationship state is not explicitly established"
	default:
		return "The prior state is not explicitly established"
	}
}

func contextualKnowledgeChange(fingerprint types.ManuscriptFingerprint, all []types.ManuscriptFingerprint, records map[string]types.EvidenceRecord) (string, string, bool) {
	if routineLogisticsRe.MatchString(fingerprint.Statement) || (fingerprint.StateChange != nil && fingerprint.StateChange.StateKind == "possession") {
		return "", "", false
	}
	for _, evidenceID := range fingerprint.EvidenceIDs {
		record, exists := records[evidenceID]
		if !exists || record.Confidence < .58 || incidentalDiscoveryRe.MatchString(record.Text) {
			continue
		}
		if record.EvidenceType == "knowledge_transfer" && informativeKnowledgePayload(fingerprint, record) {
			return developmentDiscoveryObject(fingerprint, record), "Information passes between story participants and changes what another participant can know or act upon.", true
		}
		match := resultDiscoveryRe.FindStringIndex(record.Text)
		if match == nil || (record.EvidenceType != "discovery" && fingerprint.Kind != "knowledge") {
			continue
		}
		result := cleanClause(record.Text[match[1]:])
		if !meaningfulDevelopmentObject(result) {
			continue
		}
		if normalizeSemantic(record.Text[match[0]:match[1]]) == "found" {
			if !concreteDiscoverySupport(record, result) || !laterFingerprintDependsOn(fingerprint, result, all) {
				continue
			}
			return developmentDiscoveryObject(fingerprint, record), "A concrete finding gains retrospective significance because later manuscript knowledge refers to the same result.", true
		}
		return developmentDiscoveryObject(fingerprint, record), "A source-backed discovery or realization changes the available story model; nearby fingerprints are retained as its context.", true
	}
	return "", "", false
}

func distinctDevelopmentParticipants(participants []types.NarrativeParticipant) int {
	seen := map[string]bool{}
	for _, participant := range participants {
		key := participant.EntityID
		if key == "" {
			key = normalizeSemantic(participant.EntityName)
		}
		if key != "" {
			seen[key] = true
		}
	}
	return len(seen)
}

func informativeKnowledgePayload(fingerprint types.ManuscriptFingerprint, record types.EvidenceRecord) bool {
	payload := developmentDiscoveryObject(fingerprint, record)
	terms := semanticTerms(payload)
	for _, participant := range fingerprint.Participants {
		for term := range semanticTerms(participant.EntityName) {
			delete(terms, term)
		}
	}
	for _, term := range []string{"everything", "anything", "nothing", "truth", "story", "thing", "things", "him", "her", "them"} {
		delete(terms, term)
	}
	return len(terms) >= 3
}

func laterFingerprintDependsOn(source types.ManuscriptFingerprint, result string, all []types.ManuscriptFingerprint) bool {
	sourceChapter, sourceParagraph := fingerprintPosition(source)
	terms := semanticTerms(result)
	for _, participant := range source.Participants {
		for term := range semanticTerms(participant.EntityName) {
			delete(terms, term)
		}
	}
	if len(terms) < 2 {
		return false
	}
	for _, candidate := range all {
		if candidate.ID == source.ID || candidate.Scope.ID != source.Scope.ID {
			continue
		}
		chapter, paragraph := fingerprintPosition(candidate)
		if chapter < sourceChapter || (chapter == sourceChapter && paragraph <= sourceParagraph) {
			continue
		}
		if sharedTermCount(terms, semanticTerms(candidate.Statement)) >= 2 {
			return true
		}
	}
	return false
}

func meaningfulDevelopmentObject(values ...string) bool {
	for _, value := range values {
		terms := semanticTerms(value)
		delete(terms, "find")
		delete(terms, "found")
		if len(terms) >= 2 {
			return true
		}
	}
	return false
}

func concreteDiscoverySupport(record types.EvidenceRecord, result string) bool {
	for _, term := range record.NamedEntities {
		if term.Label != "PERSON" && strings.TrimSpace(term.Text) != "" {
			return true
		}
	}
	// A concrete result stated with enough semantic detail remains usable even
	// when the local entity tagger did not recognize an ordinary object.
	return len(semanticTerms(result)) >= 4
}

func developmentDiscoveryObject(fingerprint types.ManuscriptFingerprint, record types.EvidenceRecord) string {
	if object := strings.TrimSpace(fingerprint.Object); meaningfulDevelopmentObject(object) {
		return normalizeDevelopmentText(object)
	}
	if match := resultDiscoveryRe.FindStringIndex(record.Text); match != nil {
		return normalizeDevelopmentText(cleanClause(record.Text[match[1]:]))
	}
	return normalizeDevelopmentText(record.Text)
}

func normalizeDevelopmentText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	return strings.Trim(strings.TrimSpace(value), "—–-:;, .!?\"'“”")
}

func attachDevelopmentContext(candidate developmentCandidate, fingerprints []types.ManuscriptFingerprint) developmentCandidate {
	for _, fingerprint := range fingerprints {
		if contains(candidate.fingerprintIDs, fingerprint.ID) || fingerprint.Scope.ID != candidate.scope.ID {
			continue
		}
		chapter, paragraph := fingerprintPosition(fingerprint)
		if chapter < candidate.chapterStart || chapter > candidate.chapterEnd || paragraph < candidate.paragraphStart-2 || paragraph > candidate.paragraphEnd+2 {
			continue
		}
		if !participantOverlap(candidate.participants, fingerprint.Participants) && semanticOverlap(semanticTerms(candidate.concern+" "+candidate.after), semanticTerms(fingerprint.Statement)) < .34 {
			continue
		}
		candidate.fingerprintIDs = appendUnique(candidate.fingerprintIDs, fingerprint.ID)
		candidate.evidenceIDs = appendUnique(candidate.evidenceIDs, fingerprint.EvidenceIDs...)
		candidate.participants = mergeParticipants(candidate.participants, fingerprint.Participants)
		if paragraph < candidate.paragraphStart {
			candidate.paragraphStart = paragraph
		}
		if paragraph > candidate.paragraphEnd {
			candidate.paragraphEnd = paragraph
		}
		candidate.reasons = append(candidate.reasons, types.NarrativeDevelopmentReason{
			Code: "context_support", Explanation: "A nearby fingerprint shares affected entities or subject matter and supplies context for the transition.",
			FingerprintIDs: []string{fingerprint.ID}, EvidenceIDs: clone(fingerprint.EvidenceIDs), Confidence: fingerprint.Confidence,
		})
	}
	return candidate
}

func mergeDevelopmentCandidates(candidates []developmentCandidate) []developmentCandidate {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].chapterStart != candidates[j].chapterStart {
			return candidates[i].chapterStart < candidates[j].chapterStart
		}
		return candidates[i].paragraphStart < candidates[j].paragraphStart
	})
	result := []developmentCandidate{}
	for _, candidate := range candidates {
		merged := false
		for index := len(result) - 1; index >= 0 && index >= len(result)-4; index-- {
			if candidatesDescribeSameTransition(result[index], candidate) {
				result[index] = mergeDevelopmentCandidate(result[index], candidate)
				merged = true
				break
			}
		}
		if !merged {
			result = append(result, candidate)
		}
	}
	return result
}

func candidatesDescribeSameTransition(left, right developmentCandidate) bool {
	if left.fromRelation || right.fromRelation || left.kind != right.kind || left.scope.ID != right.scope.ID || left.chapterEnd != right.chapterStart {
		return false
	}
	if right.paragraphStart-left.paragraphEnd > 3 {
		return false
	}
	return participantOverlap(left.participants, right.participants) || semanticOverlap(semanticTerms(left.concern+" "+left.after), semanticTerms(right.concern+" "+right.after)) >= .4
}

func mergeDevelopmentCandidate(left, right developmentCandidate) developmentCandidate {
	left.fingerprintIDs = appendUnique(left.fingerprintIDs, right.fingerprintIDs...)
	left.evidenceIDs = appendUnique(left.evidenceIDs, right.evidenceIDs...)
	left.participants = mergeParticipants(left.participants, right.participants)
	left.reasons = append(left.reasons, right.reasons...)
	left.after = right.after
	if left.before == "" {
		left.before = right.before
	}
	if left.concern == "" {
		left.concern = right.concern
	}
	left.chapterEnd, left.paragraphEnd = right.chapterEnd, right.paragraphEnd
	left.confidence = minFloat(left.confidence, right.confidence)
	return left
}

func developmentFromCandidate(candidate developmentCandidate, fingerprints map[string]types.ManuscriptFingerprint) types.NarrativeDevelopment {
	summary := developmentSummary(candidate, fingerprints)
	idParts := append([]string{candidate.kind, candidate.scope.ID}, candidate.fingerprintIDs...)
	return types.NarrativeDevelopment{
		ID: stableID("development", idParts...), Kind: candidate.kind, Summary: summary, Before: candidate.before, After: candidate.after,
		AffectedEntities: candidate.participants, AdvancedConcern: candidate.concern, FingerprintIDs: clone(candidate.fingerprintIDs), EvidenceIDs: clone(candidate.evidenceIDs),
		Scope: candidate.scope, Temporal: candidate.temporal, ChapterStart: candidate.chapterStart, ChapterEnd: candidate.chapterEnd,
		Reasons: candidate.reasons, Confidence: candidate.confidence, Status: "active",
	}
}

func developmentSummary(candidate developmentCandidate, fingerprints map[string]types.ManuscriptFingerprint) string {
	if candidate.fromRelation {
		return candidate.after
	}
	primary := types.ManuscriptFingerprint{}
	for _, id := range candidate.fingerprintIDs {
		if value, exists := fingerprints[id]; exists && value.StateChange != nil {
			primary = value
			break
		}
	}
	if primary.ID == "" && len(candidate.fingerprintIDs) > 0 {
		primary = fingerprints[candidate.fingerprintIDs[0]]
	}
	subject := developmentSubject(primary, candidate.participants)
	switch candidate.kind {
	case "objective_change":
		return strings.TrimSpace(subject + " adopts or changes objective: " + candidate.concern)
	case "obligation_created":
		return strings.TrimSpace(subject + " creates an obligation: " + candidate.concern)
	case "relationship_change":
		return strings.TrimSpace(subject + " changes a relationship: " + candidate.concern)
	case "knowledge_change":
		return strings.TrimSpace(subject + " learns or establishes: " + candidate.concern)
	case "obstacle":
		return "An existing course is obstructed: " + conciseFingerprintState(primary)
	case "threat_change":
		return "A threat appears or escalates: " + conciseFingerprintState(primary)
	case "persistent_state_change":
		if subject == "The story" {
			subject = "An unresolved entity"
		}
		return strings.TrimSpace(subject + " undergoes a consequential state change: " + conciseFingerprintState(primary))
	}
	return candidate.after
}

func developmentSubject(primary types.ManuscriptFingerprint, participants []types.NarrativeParticipant) string {
	if primary.StateChange != nil && stableHistoryEntity(primary.StateChange.EntityID, primary.StateChange.EntityName) {
		return primary.StateChange.EntityName
	}
	if primary.SubjectID != "" && stableHistoryEntity(primary.SubjectID, primary.Subject) {
		return primary.Subject
	}
	for _, participant := range participants {
		if participant.Role == "subject" && participant.EntityID != "" && stableHistoryEntity(participant.EntityID, participant.EntityName) {
			return participant.EntityName
		}
	}
	return "The story"
}

func conciseFingerprintState(item types.ManuscriptFingerprint) string {
	if item.StateChange != nil {
		subject := nonEmpty(item.StateChange.EntityName, item.Subject)
		return strings.TrimSpace(subject + " " + normalizeAction(item.StateChange.Operation) + " " + item.StateChange.StateKind + ": " + item.StateChange.New)
	}
	return strings.TrimSpace(strings.Join(nonEmptyParts(item.Subject, item.Predicate, item.Object), " "))
}

func priorStateValue(fingerprint types.ManuscriptFingerprint, property string, histories []types.FingerprintStateHistory) string {
	for _, history := range histories {
		if history.Property != property || history.ScopeID != fingerprint.Scope.ID || (fingerprint.SubjectID != "" && history.EntityID != fingerprint.SubjectID) {
			continue
		}
		for index, entry := range history.Entries {
			if entry.FingerprintID == fingerprint.ID && index > 0 {
				return history.Entries[index-1].Value
			}
		}
	}
	return ""
}

func sharedConcern(left, right types.ManuscriptFingerprint) string {
	shared := []string{}
	leftTerms, rightTerms := semanticTerms(left.Object+" "+stateValueFromMemory(left)), semanticTerms(right.Object+" "+stateValueFromMemory(right))
	for term := range leftTerms {
		if rightTerms[term] {
			shared = append(shared, term)
		}
	}
	sort.Strings(shared)
	return strings.Join(shared, " ")
}

func stateValueFromMemory(item types.ManuscriptFingerprint) string {
	if item.StateChange != nil {
		return item.StateChange.New
	}
	return ""
}

func fingerprintPosition(item types.ManuscriptFingerprint) (int, int) {
	if len(item.EvidenceSpans) == 0 {
		return 0, 0
	}
	return item.EvidenceSpans[0].ChapterIndex, item.EvidenceSpans[0].ParagraphIndex
}

func mergeParticipants(left, right []types.NarrativeParticipant) []types.NarrativeParticipant {
	result := append([]types.NarrativeParticipant(nil), left...)
	for _, participant := range right {
		result = appendParticipant(result, participant)
	}
	return result
}

func firstParticipantName(values []types.NarrativeParticipant) string {
	for _, participant := range values {
		if participant.Role != "source" && participant.EntityName != "" {
			return participant.EntityName
		}
	}
	return "The story"
}

func firstDevelopmentOffset(development types.NarrativeDevelopment, fingerprints map[string]types.ManuscriptFingerprint) int {
	result := int(^uint(0) >> 1)
	for _, id := range development.FingerprintIDs {
		for _, span := range fingerprints[id].EvidenceSpans {
			if span.StartOffset < result {
				result = span.StartOffset
			}
		}
	}
	return result
}
