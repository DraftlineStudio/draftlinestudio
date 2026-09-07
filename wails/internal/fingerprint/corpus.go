package fingerprint

import (
	"sort"
	"strings"

	"draftline/internal/types"
)

// buildFingerprintCorpus turns every normalized assertion into durable
// manuscript memory. There is deliberately no narrative-importance gate here.
func buildFingerprintCorpus(assertions []types.StoryAssertion, records []types.EvidenceRecord) ([]types.ManuscriptFingerprint, []types.ManuscriptFingerprintRelation, []types.ManuscriptEventIdentity) {
	recordByID := evidenceRecordMap(records)
	fingerprints := make([]types.ManuscriptFingerprint, 0, len(assertions))
	assertionToFingerprint := make(map[string]string, len(assertions))
	for _, assertion := range assertions {
		item := manuscriptFingerprintFromAssertion(assertion, recordByID)
		fingerprints = append(fingerprints, item)
		assertionToFingerprint[assertion.ID] = item.ID
	}
	relations := corpusRelations(inferAssertionRelations(assertions, recordByID), assertionToFingerprint)
	identities, sameEventRelations := resolveEventIdentities(fingerprints, recordByID)
	relations = append(relations, sameEventRelations...)
	sort.SliceStable(fingerprints, func(i, j int) bool {
		return spanOrder(fingerprints[i].EvidenceSpans) < spanOrder(fingerprints[j].EvidenceSpans)
	})
	return fingerprints, dedupeCorpusRelations(relations), identities
}

func manuscriptFingerprintFromAssertion(assertion types.StoryAssertion, records map[string]types.EvidenceRecord) types.ManuscriptFingerprint {
	spans := make([]types.NarrativeEvidenceSpan, 0, len(assertion.EvidenceIDs))
	participants := []types.NarrativeParticipant{}
	for _, id := range assertion.EvidenceIDs {
		record, exists := records[id]
		if !exists {
			continue
		}
		spans = append(spans, evidenceSpan(record))
		for index, name := range record.CharacterNames {
			entityID := ""
			if index < len(record.CharacterIDs) {
				entityID = record.CharacterIDs[index]
			}
			role := "mentioned"
			if (assertion.SubjectID != "" && assertion.SubjectID == entityID) || (assertion.SubjectID == "" && strings.EqualFold(assertion.Subject, name)) {
				role = "subject"
			}
			participants = appendParticipant(participants, types.NarrativeParticipant{EntityID: entityID, EntityName: name, Role: role})
		}
	}
	if assertion.Attribution.Kind == "character" {
		participants = appendParticipant(participants, types.NarrativeParticipant{EntityID: assertion.Attribution.EntityID, EntityName: assertion.Attribution.EntityName, Role: "source"})
	}
	id := stableID("fingerprint", assertion.SemanticKey, assertion.Scope.ID, assertion.EpistemicStatus, assertion.Attribution.EntityID)
	return types.ManuscriptFingerprint{
		ID: id, Kind: assertion.Kind, Statement: normalizedMemoryStatement(assertion), SemanticKey: assertion.SemanticKey,
		SubjectID: assertion.SubjectID, Subject: assertion.Subject, Predicate: assertion.Predicate, ObjectID: assertion.ObjectID, Object: assertion.Object,
		Polarity: assertion.Polarity, AssertionIDs: []string{assertion.ID}, EvidenceIDs: clone(assertion.EvidenceIDs), EvidenceSpans: spans,
		Participants: participants, StateChange: assertion.StateChange, EpistemicStatus: assertion.EpistemicStatus, Attribution: assertion.Attribution,
		Scope: assertion.Scope, Persistence: assertion.Persistence, Temporal: assertion.Temporal, Confidence: assertion.Confidence, Status: assertion.Status,
	}
}

func normalizedMemoryStatement(assertion types.StoryAssertion) string {
	subject := strings.TrimSpace(assertion.Subject)
	predicate := normalizeAction(assertion.Predicate)
	object := strings.TrimSpace(assertion.Object)
	if assertion.Attribution.Kind == "character" && assertion.Attribution.EntityName != "" {
		claim := strings.TrimSpace(strings.Join(nonEmptyParts(subject, predicate, object), " "))
		if claim == "" {
			claim = strings.TrimSpace(assertion.Statement)
		}
		return strings.TrimSpace(assertion.Attribution.EntityName + " " + nonEmpty(assertion.Attribution.Cue, "states") + " that " + claim)
	}
	if assertion.StateChange != nil && subject != "" {
		change := assertion.StateChange
		if change.Previous != "" {
			return strings.TrimSpace(subject + " " + change.StateKind + " changes from " + change.Previous + " to " + change.New)
		}
		if change.New != "" {
			return strings.TrimSpace(subject + " " + normalizeAction(change.Operation) + " " + change.StateKind + ": " + change.New)
		}
	}
	result := strings.TrimSpace(strings.Join(nonEmptyParts(subject, predicate, object), " "))
	if result == "" {
		return strings.TrimSpace(assertion.Statement)
	}
	return result
}

func nonEmptyParts(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func corpusRelations(values []assertionRelation, assertionToFingerprint map[string]string) []types.ManuscriptFingerprintRelation {
	result := make([]types.ManuscriptFingerprintRelation, 0, len(values))
	for _, value := range values {
		from, fromOK := assertionToFingerprint[value.from]
		to, toOK := assertionToFingerprint[value.to]
		if !fromOK || !toOK || from == to {
			continue
		}
		result = append(result, types.ManuscriptFingerprintRelation{
			ID: stableID("fingerprint-relation", from, to, value.kind), FromID: from, ToID: to, Kind: value.kind,
			Explanation: value.explain, EvidenceIDs: clone(value.evidence), Confidence: value.confidence,
		})
	}
	return result
}

func resolveEventIdentities(fingerprints []types.ManuscriptFingerprint, records map[string]types.EvidenceRecord) ([]types.ManuscriptEventIdentity, []types.ManuscriptFingerprintRelation) {
	parent := make([]int, len(fingerprints))
	for index := range parent {
		parent[index] = index
	}
	find := func(index int) int { return index }
	var root func(int) int
	root = func(index int) int {
		if parent[index] != index {
			parent[index] = root(parent[index])
		}
		return parent[index]
	}
	find = root
	union := func(left, right int) {
		left, right = find(left), find(right)
		if left != right {
			parent[right] = left
		}
	}

	byType := map[string][]int{}
	for index, item := range fingerprints {
		eventType := fingerprintEventType(item, records)
		if eventType == "" || !identityEligible(item) {
			continue
		}
		key := item.Scope.ID + "\x00" + eventType
		for _, prior := range byType[key] {
			if sameUnderlyingEvent(fingerprints[prior], item, eventType) {
				union(prior, index)
			}
		}
		byType[key] = append(byType[key], index)
	}

	groups := map[int][]int{}
	for index := range fingerprints {
		groups[find(index)] = append(groups[find(index)], index)
	}
	identities := []types.ManuscriptEventIdentity{}
	relations := []types.ManuscriptFingerprintRelation{}
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		identity := makeEventIdentity(group, fingerprints, records)
		identities = append(identities, identity)
		for _, index := range group {
			fingerprints[index].EventIdentityID = identity.ID
		}
		for index := 1; index < len(group); index++ {
			left, right := fingerprints[group[0]], fingerprints[group[index]]
			relations = append(relations, types.ManuscriptFingerprintRelation{
				ID: stableID("fingerprint-relation", left.ID, right.ID, "same-event"), FromID: left.ID, ToID: right.ID, Kind: "same_event",
				Explanation: "The passages likely describe the same underlying event based on event type, semantic detail, scope, participants, and nearby narrative context.",
				EvidenceIDs: appendUnique(clone(left.EvidenceIDs), right.EvidenceIDs...), Confidence: identity.Confidence,
			})
		}
	}
	sort.SliceStable(identities, func(i, j int) bool { return identities[i].ID < identities[j].ID })
	return identities, relations
}

func identityEligible(item types.ManuscriptFingerprint) bool {
	if item.EpistemicStatus != "world_state_fact" && item.EpistemicStatus != "externally_corroborated_fact" {
		return true
	}
	return item.Scope.Kind != "current" || item.Temporal.Precision == "exact" || item.Temporal.Precision == "day"
}

func fingerprintEventType(item types.ManuscriptFingerprint, records map[string]types.EvidenceRecord) string {
	if item.StateChange != nil && item.StateChange.StateKind != "" {
		return normalizeSemantic(item.StateChange.StateKind)
	}
	for _, evidenceID := range item.EvidenceIDs {
		action := normalizeSemantic(records[evidenceID].Action)
		if action != "" && !reportingAction(action) {
			return action
		}
	}
	predicate := normalizeSemantic(item.Predicate)
	if predicate != "" && predicate != "state" && predicate != "is" && predicate != "claims" && !reportingAction(predicate) {
		return predicate
	}
	return ""
}

func reportingAction(action string) bool {
	switch action {
	case "said", "says", "told", "claimed", "claims", "reported", "reports", "explained", "warned", "asked", "replied", "answered":
		return true
	}
	return false
}

func sameUnderlyingEvent(left, right types.ManuscriptFingerprint, eventType string) bool {
	if !compatibleScopes(left.Scope, right.Scope) {
		return false
	}
	leftChapter, rightChapter := firstFingerprintChapter(left), firstFingerprintChapter(right)
	if absInt(leftChapter-rightChapter) > 3 && !compatibleExactTime(left.Temporal, right.Temporal) {
		return false
	}
	leftTerms := identityTerms(left, eventType)
	rightTerms := identityTerms(right, eventType)
	shared := sharedTermCount(leftTerms, rightTerms)
	if shared < 2 {
		return false
	}
	score := .35 + .2 + minFloat(.2, semanticOverlap(leftTerms, rightTerms)*.2)
	if participantOverlap(left.Participants, right.Participants) {
		score += .15
	}
	if absInt(leftChapter-rightChapter) <= 1 || compatibleExactTime(left.Temporal, right.Temporal) {
		score += .15
	}
	return score >= .75
}

func identityTerms(item types.ManuscriptFingerprint, eventType string) map[string]bool {
	terms := semanticTerms(item.Subject + " " + item.Object + " " + item.Statement)
	delete(terms, eventType)
	for _, participant := range item.Participants {
		for term := range semanticTerms(participant.EntityName) {
			delete(terms, term)
		}
	}
	return terms
}

func participantOverlap(left, right []types.NarrativeParticipant) bool {
	ids := map[string]bool{}
	names := map[string]bool{}
	for _, participant := range left {
		if participant.EntityID != "" {
			ids[participant.EntityID] = true
		}
		if participant.EntityName != "" {
			names[normalizeSemantic(participant.EntityName)] = true
		}
	}
	for _, participant := range right {
		if participant.EntityID != "" && ids[participant.EntityID] {
			return true
		}
		if participant.EntityName != "" && names[normalizeSemantic(participant.EntityName)] {
			return true
		}
	}
	return false
}

func compatibleExactTime(left, right types.StoryTime) bool {
	if left.DayOffset == nil || right.DayOffset == nil {
		return false
	}
	return *left.DayOffset == *right.DayOffset
}

func makeEventIdentity(group []int, fingerprints []types.ManuscriptFingerprint, records map[string]types.EvidenceRecord) types.ManuscriptEventIdentity {
	first := fingerprints[group[0]]
	eventType := fingerprintEventType(first, records)
	identity := types.ManuscriptEventIdentity{
		EventType: eventType, ScopeIDs: []string{}, Temporal: first.Temporal, Status: "resolved", Confidence: .78,
	}
	for _, index := range group {
		item := fingerprints[index]
		identity.FingerprintIDs = append(identity.FingerprintIDs, item.ID)
		identity.EvidenceIDs = appendUnique(identity.EvidenceIDs, item.EvidenceIDs...)
		identity.ScopeIDs = appendUnique(identity.ScopeIDs, item.Scope.ID)
		for _, participant := range item.Participants {
			identity.Participants = appendParticipant(identity.Participants, participant)
		}
		if property, ok := eventProperty(item, eventType); ok {
			identity.Properties = append(identity.Properties, property)
		}
	}
	identity.ID = stableID("event-identity", eventType, strings.Join(identity.EvidenceIDs, "|"))
	if eventPropertiesConflict(identity.Properties) {
		identity.Status = "conflicted"
	}
	return identity
}

func eventProperty(item types.ManuscriptFingerprint, eventType string) (types.ManuscriptEventProperty, bool) {
	name, value := "", ""
	if item.StateChange != nil {
		name, value = item.StateChange.StateKind, item.StateChange.New
	} else if actor := actorBeforeAction(item.Object, eventType); actor != "" {
		name, value = "actor", actor
	}
	if name == "" || value == "" {
		return types.ManuscriptEventProperty{}, false
	}
	return types.ManuscriptEventProperty{Name: name, Value: value, FingerprintIDs: []string{item.ID}, EvidenceIDs: clone(item.EvidenceIDs), EpistemicStatus: item.EpistemicStatus, Attribution: item.Attribution, Confidence: item.Confidence}, true
}

func actorBeforeAction(value, action string) string {
	words := strings.Fields(strings.TrimSpace(value))
	for index, word := range words {
		if normalizeSemantic(word) == action && index > 0 && index <= 4 {
			return strings.Join(words[:index], " ")
		}
	}
	return ""
}

func eventPropertiesConflict(properties []types.ManuscriptEventProperty) bool {
	byName := map[string]map[string]bool{}
	for _, property := range properties {
		if byName[property.Name] == nil {
			byName[property.Name] = map[string]bool{}
		}
		byName[property.Name][normalizeSemantic(property.Value)] = true
	}
	for _, values := range byName {
		if len(values) > 1 {
			return true
		}
	}
	return false
}

func firstFingerprintChapter(item types.ManuscriptFingerprint) int {
	if len(item.EvidenceSpans) == 0 {
		return 0
	}
	return item.EvidenceSpans[0].ChapterIndex
}

func dedupeCorpusRelations(values []types.ManuscriptFingerprintRelation) []types.ManuscriptFingerprintRelation {
	seen := map[string]bool{}
	result := make([]types.ManuscriptFingerprintRelation, 0, len(values))
	for _, value := range values {
		if !seen[value.ID] {
			seen[value.ID] = true
			result = append(result, value)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
