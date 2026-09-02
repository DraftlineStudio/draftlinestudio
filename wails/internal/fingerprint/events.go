package fingerprint

import (
	"sort"
	"strings"

	"draftline/internal/types"
)

func consolidateEvents(seeds []types.FingerprintEvent, records []types.EvidenceRecord, assertions []types.StoryAssertion, prior *types.StoryFingerprint) []types.FingerprintEvent {
	recordByID := map[string]types.EvidenceRecord{}
	assertionsByEvidence := map[string][]types.StoryAssertion{}
	for _, record := range records {
		recordByID[record.ID] = record
	}
	for _, assertion := range assertions {
		for _, id := range assertion.EvidenceIDs {
			assertionsByEvidence[id] = append(assertionsByEvidence[id], assertion)
		}
	}
	result := []types.FingerprintEvent{}
	for _, seed := range seeds {
		if len(result) > 0 && shouldMergeEvent(result[len(result)-1], seed, recordByID) {
			mergeEvent(&result[len(result)-1], seed, recordByID, assertionsByEvidence)
			continue
		}
		for _, assertion := range assertionsByEvidence[seed.EvidenceIDs[0]] {
			seed.AssertionIDs = appendUnique(seed.AssertionIDs, assertion.ID)
		}
		seed.Locations, seed.Objects = eventTerms(recordByID[seed.EvidenceIDs[0]])
		result = append(result, seed)
	}
	for index := range result {
		event := &result[index]
		event.ID = compositeEventID(*event, recordByID)
		event.Summary = summarizeEvent(*event, recordByID, assertionsByEvidence)
		event.NarrativeOrder = index
		preservePriorEvent(event, prior, recordByID)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].NarrativeOrder < result[j].NarrativeOrder })
	return result
}

func shouldMergeEvent(left, right types.FingerprintEvent, records map[string]types.EvidenceRecord) bool {
	if left.ChapterIndex != right.ChapterIndex || left.ContextID != right.ContextID {
		return false
	}
	if right.ParagraphIndex-left.ParagraphIndex > 1 {
		return false
	}
	if left.ParagraphIndex == right.ParagraphIndex && overlap(left.CharacterIDs, right.CharacterIDs) {
		return true
	}
	leftRecord := records[left.EvidenceIDs[len(left.EvidenceIDs)-1]]
	rightRecord := records[right.EvidenceIDs[0]]
	if leftRecord.EvidenceType == rightRecord.EvidenceType && leftRecord.Action != "" && strings.EqualFold(leftRecord.Action, rightRecord.Action) && overlap(left.CharacterIDs, right.CharacterIDs) {
		return true
	}
	return false
}

func mergeEvent(target *types.FingerprintEvent, addition types.FingerprintEvent, records map[string]types.EvidenceRecord, assertions map[string][]types.StoryAssertion) {
	target.EvidenceIDs = appendUnique(target.EvidenceIDs, addition.EvidenceIDs...)
	target.CharacterIDs = appendUnique(target.CharacterIDs, addition.CharacterIDs...)
	target.CharacterNames = appendUnique(target.CharacterNames, addition.CharacterNames...)
	target.Kinds = appendUnique(target.Kinds, addition.Kinds...)
	for _, evidenceID := range addition.EvidenceIDs {
		for _, assertion := range assertions[evidenceID] {
			target.AssertionIDs = appendUnique(target.AssertionIDs, assertion.ID)
		}
	}
	locations, objects := eventTerms(records[addition.EvidenceIDs[0]])
	target.Locations = appendUniqueTerms(target.Locations, locations)
	target.Objects = appendUniqueTerms(target.Objects, objects)
	if addition.Importance > target.Importance {
		target.Importance = addition.Importance
	}
	if addition.Confidence < target.Confidence {
		target.Confidence = addition.Confidence
	}
}

func summarizeEvent(event types.FingerprintEvent, records map[string]types.EvidenceRecord, assertions map[string][]types.StoryAssertion) string {
	for _, evidenceID := range event.EvidenceIDs {
		for _, assertion := range assertions[evidenceID] {
			if assertion.Subject != "" && assertion.Predicate != "" && assertion.Predicate != "state" {
				summary := assertion.Subject + " " + normalizeAction(assertion.Predicate)
				if assertion.Object != "" {
					summary += " " + assertion.Object
				}
				return strings.TrimSpace(summary)
			}
		}
	}
	if len(event.EvidenceIDs) == 1 {
		return records[event.EvidenceIDs[0]].Text
	}
	first := records[event.EvidenceIDs[0]]
	if len(event.CharacterNames) > 0 {
		return event.CharacterNames[0] + " participates in a " + first.EvidenceType + " sequence"
	}
	return "A " + first.EvidenceType + " sequence unfolds"
}

func normalizeAction(action string) string {
	return strings.Trim(strings.Join(strings.Fields(action), " "), " ,.;:")
}

func eventTerms(record types.EvidenceRecord) ([]types.EvidenceTerm, []types.EvidenceTerm) {
	locations, objects := []types.EvidenceTerm{}, []types.EvidenceTerm{}
	for _, term := range record.NamedEntities {
		switch strings.ToUpper(term.Label) {
		case "GPE", "LOC", "FAC":
			locations = appendUniqueTerms(locations, []types.EvidenceTerm{term})
		case "PERSON", "PER", "DATE", "TIME", "CARDINAL", "ORDINAL":
		default:
			objects = appendUniqueTerms(objects, []types.EvidenceTerm{term})
		}
	}
	return locations, objects
}

func compositeEventID(event types.FingerprintEvent, records map[string]types.EvidenceRecord) string {
	parts := []string{event.ContextID, event.ChapterID}
	for _, id := range event.EvidenceIDs {
		record := records[id]
		parts = append(parts, strings.ToLower(strings.Join(record.CharacterNames, "|")), strings.ToLower(record.Action), normalizeFingerprintText(record.Text))
	}
	return stableID("event", parts...)
}

func preservePriorEvent(event *types.FingerprintEvent, prior *types.StoryFingerprint, records map[string]types.EvidenceRecord) {
	if prior == nil {
		return
	}
	for _, old := range prior.Events {
		if old.ID == event.ID || evidenceSimilarity(old.EvidenceIDs, event.EvidenceIDs) >= .6 {
			if old.AuthorSummary != "" {
				event.AuthorSummary = old.AuthorSummary
				event.Summary = old.AuthorSummary
			}
			return
		}
	}
}

func evidenceSimilarity(left, right []string) float64 {
	set := map[string]bool{}
	for _, value := range left {
		set[value] = true
	}
	common := 0
	for _, value := range right {
		if set[value] {
			common++
		}
	}
	denominator := len(left) + len(right) - common
	if denominator == 0 {
		return 0
	}
	return float64(common) / float64(denominator)
}

func normalizeFingerprintText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

func overlap(left, right []string) bool {
	set := map[string]bool{}
	for _, value := range left {
		set[value] = true
	}
	for _, value := range right {
		if set[value] {
			return true
		}
	}
	return false
}

func appendUnique(values []string, additions ...string) []string {
	set := map[string]bool{}
	for _, value := range values {
		set[value] = true
	}
	for _, value := range additions {
		if value != "" && !set[value] {
			values = append(values, value)
			set[value] = true
		}
	}
	return values
}

func appendUniqueTerms(values []types.EvidenceTerm, additions []types.EvidenceTerm) []types.EvidenceTerm {
	set := map[string]bool{}
	for _, value := range values {
		set[strings.ToLower(value.Label+"\x00"+value.Text)] = true
	}
	for _, value := range additions {
		key := strings.ToLower(value.Label + "\x00" + value.Text)
		if value.Text != "" && !set[key] {
			values = append(values, value)
			set[key] = true
		}
	}
	return values
}
