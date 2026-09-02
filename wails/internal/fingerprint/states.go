package fingerprint

import (
	"regexp"
	"strings"

	"draftline/internal/types"
)

var (
	attributeRe  = regexp.MustCompile(`(?i)\b(?:has|had|with)\s+(?:a\s+)?(blond|blonde|brown|black|red|gray|grey|white|blue|green|hazel)\s+(hair|eyes?)\b`)
	possessionRe = regexp.MustCompile(`(?i)\b(?:held|carried|carries|carrying|obtained|took|takes|has|had)\s+(?:a|an|the|his|her|their)?\s*([a-z][a-z0-9 -]{1,40})`)
	presenceRe   = regexp.MustCompile(`(?i)\b(?:sat|stood|waited|entered|arrived|walked|came|remained|stayed)\b`)
)

func buildStates(events []types.FingerprintEvent, records []types.EvidenceRecord, assertions []types.StoryAssertion) []types.StoryStateInterval {
	recordByID := map[string]types.EvidenceRecord{}
	eventByEvidence := map[string]types.FingerprintEvent{}
	for _, record := range records {
		recordByID[record.ID] = record
	}
	for _, event := range events {
		for _, id := range event.EvidenceIDs {
			eventByEvidence[id] = event
		}
	}
	result := []types.StoryStateInterval{}
	for _, event := range events {
		for _, evidenceID := range event.EvidenceIDs {
			record := recordByID[evidenceID]
			if presenceRe.MatchString(record.Text) || record.EvidenceType == "introduction" || record.EvidenceType == "transition" {
				for index, id := range record.CharacterIDs {
					name := id
					if index < len(record.CharacterNames) {
						name = record.CharacterNames[index]
					}
					result = appendState(result, state(id, name, "presence", locationValue(event), event, evidenceID, false, .72))
				}
			}
			if match := attributeRe.FindStringSubmatch(record.Text); len(match) == 3 && len(record.CharacterIDs) > 0 {
				id, name := firstCharacter(record)
				result = appendState(result, state(id, name, "attribute", strings.ToLower(match[1]+" "+match[2]), event, evidenceID, true, .84))
			}
			if match := possessionRe.FindStringSubmatch(record.Text); len(match) == 2 && len(record.CharacterIDs) > 0 {
				value := strings.TrimSpace(match[1])
				if len(strings.Fields(value)) <= 5 {
					id, name := firstCharacter(record)
					item := state(id, name, "possession", value, event, evidenceID, true, .7)
					item.Qualifier = "physical_custody"
					result = appendState(result, item)
				}
			}
			for _, knowledge := range record.KnowledgeStates {
				for index, id := range knowledge.CharacterIDs {
					name := id
					if index < len(knowledge.CharacterNames) {
						name = knowledge.CharacterNames[index]
					}
					kind := "knowledge"
					if knowledge.State == "believes" || knowledge.State == "suspects" {
						kind = "belief"
					}
					result = appendState(result, state(id, name, kind, knowledge.Cue, event, evidenceID, true, knowledge.Confidence))
				}
			}
		}
	}
	closeTransientStates(result, events)
	return result
}

func state(id, name, kind, value string, event types.FingerprintEvent, evidenceID string, persistent bool, confidence float64) types.StoryStateInterval {
	return types.StoryStateInterval{ID: stableID("state", id, kind, strings.ToLower(value), event.ContextID, evidenceID), EntityID: id, EntityName: name, Kind: kind, Value: value, ContextID: event.ContextID, StartEventID: event.ID, EvidenceIDs: []string{evidenceID}, Persistent: persistent, Confidence: confidence}
}

func appendState(states []types.StoryStateInterval, addition types.StoryStateInterval) []types.StoryStateInterval {
	for index := len(states) - 1; index >= 0; index-- {
		prior := &states[index]
		if prior.EntityID == addition.EntityID && prior.Kind == addition.Kind && prior.ContextID == addition.ContextID && strings.EqualFold(prior.Value, addition.Value) && prior.EndEventID == "" {
			prior.EvidenceIDs = appendUnique(prior.EvidenceIDs, addition.EvidenceIDs...)
			if addition.Confidence > prior.Confidence {
				prior.Confidence = addition.Confidence
			}
			return states
		}
	}
	return append(states, addition)
}

func closeTransientStates(states []types.StoryStateInterval, events []types.FingerprintEvent) {
	lastByChapter := map[int]string{}
	for _, event := range events {
		lastByChapter[event.ChapterIndex] = event.ID
	}
	for index := range states {
		if !states[index].Persistent {
			for _, event := range events {
				if event.ID == states[index].StartEventID {
					states[index].EndEventID = lastByChapter[event.ChapterIndex]
					break
				}
			}
		}
	}
}

func locationValue(event types.FingerprintEvent) string {
	if len(event.Locations) > 0 {
		return event.Locations[0].Text
	}
	return "present in passage"
}
