package fingerprint

import (
	"regexp"
	"strings"

	"draftline/internal/types"
)

var (
	negationRe  = regexp.MustCompile(`(?i)\b(?:not|never|no longer|didn't|doesn't|isn't|wasn't|couldn't|wouldn't|cannot|can't)\b`)
	beliefRe    = regexp.MustCompile(`(?i)\b(?:believed|believes|thought|thinks|suspected|suspects|theorized|guessed|assumed)\b`)
	memoryRe    = regexp.MustCompile(`(?i)\b(?:remembered|remembers|recalled|recalls|memory of)\b`)
	uncertainRe = regexp.MustCompile(`(?i)\b(?:maybe|perhaps|possibly|apparently|seemed|might|could have)\b`)
)

func buildAssertions(records []types.EvidenceRecord, contexts map[string]string) []types.StoryAssertion {
	result := make([]types.StoryAssertion, 0, len(records))
	for _, record := range records {
		subjectID, subject := firstCharacter(record)
		predicate := strings.TrimSpace(record.Action)
		if predicate == "" {
			predicate = record.EvidenceType
		}
		objectID, object := assertionObject(record, subject)
		posture, polarity := assertionPosture(record.Text, record.KnowledgeStates)
		result = append(result, types.StoryAssertion{
			ID: stableID("assertion", record.ID, subjectID, predicate, object), EvidenceIDs: []string{record.ID},
			SubjectID: subjectID, Subject: subject, Predicate: predicate, ObjectID: objectID, Object: object,
			Posture: posture, Polarity: polarity, ContextID: contexts[record.ID], Confidence: record.Confidence,
		})
		for _, knowledge := range record.KnowledgeStates {
			for index, id := range knowledge.CharacterIDs {
				name := id
				if index < len(knowledge.CharacterNames) {
					name = knowledge.CharacterNames[index]
				}
				result = append(result, types.StoryAssertion{
					ID: stableID("assertion", record.ID, id, knowledge.State, knowledge.Cue), EvidenceIDs: []string{record.ID},
					SubjectID: id, Subject: name, Predicate: knowledge.State, Object: knowledge.Cue,
					Posture: knowledgePosture(knowledge.State), Polarity: knowledgePolarity(knowledge.State), ContextID: contexts[record.ID], Confidence: knowledge.Confidence,
				})
			}
		}
	}
	return result
}

func firstCharacter(record types.EvidenceRecord) (string, string) {
	if len(record.CharacterIDs) == 0 {
		return "", ""
	}
	name := record.CharacterIDs[0]
	if len(record.CharacterNames) > 0 {
		name = record.CharacterNames[0]
	}
	return record.CharacterIDs[0], name
}

func assertionObject(record types.EvidenceRecord, subject string) (string, string) {
	for _, term := range record.NamedEntities {
		if strings.EqualFold(term.Text, subject) {
			continue
		}
		return "", term.Text
	}
	if len(record.CharacterIDs) > 1 {
		name := record.CharacterIDs[1]
		if len(record.CharacterNames) > 1 {
			name = record.CharacterNames[1]
		}
		return record.CharacterIDs[1], name
	}
	return "", ""
}

func assertionPosture(text string, knowledge []types.EvidenceKnowledgeState) (string, string) {
	polarity := "positive"
	if negationRe.MatchString(text) {
		polarity = "negative"
	}
	if uncertainRe.MatchString(text) {
		polarity = "uncertain"
	}
	lower := strings.ToLower(text)
	if dreamCueRe.MatchString(text) {
		return "dream", polarity
	}
	if memoryRe.MatchString(text) {
		return "memory", polarity
	}
	if beliefRe.MatchString(text) {
		return "belief", polarity
	}
	for _, state := range knowledge {
		if state.State == "suspects" || state.State == "believes" {
			return "belief", polarity
		}
	}
	if strings.ContainsAny(text, "\"“”") || strings.Contains(lower, " said") || strings.Contains(lower, " told ") {
		return "claim", polarity
	}
	return "fact", polarity
}

func knowledgePosture(state string) string {
	if state == "believes" || state == "suspects" {
		return "belief"
	}
	return "fact"
}

func knowledgePolarity(state string) string {
	if state == "does_not_know" || state == "withheld" {
		return "negative"
	}
	return "positive"
}
