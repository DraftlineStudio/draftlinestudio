package fingerprint

import (
	"sort"
	"strings"

	"draftline/internal/types"
)

func buildFingerprintStateHistories(fingerprints []types.ManuscriptFingerprint, records []types.EvidenceRecord) []types.FingerprintStateHistory {
	recordByID := evidenceRecordMap(records)
	histories := map[string]*types.FingerprintStateHistory{}
	for _, fingerprint := range fingerprints {
		if fingerprint.StateChange != nil {
			change := fingerprint.StateChange
			entityID, entityName := change.EntityID, change.EntityName
			if entityName == "" {
				entityID, entityName = fingerprint.SubjectID, fingerprint.Subject
			}
			appendHistoryEntry(histories, fingerprint, entityID, entityName, change.StateKind, stateQualifier(change.StateKind, change.Operation), change.New, change.Operation)
		}
		if fingerprint.Kind == "knowledge" && fingerprint.Subject != "" && fingerprint.Object != "" {
			property := "knowledge"
			if fingerprint.EpistemicStatus == "character_belief" || fingerprint.EpistemicStatus == "character_inference" {
				property = "belief"
			}
			appendHistoryEntry(histories, fingerprint, fingerprint.SubjectID, fingerprint.Subject, property, "", fingerprint.Object, fingerprint.Predicate)
		}
		if fingerprint.Kind == "claim" && fingerprint.Subject != "" && fingerprint.Predicate == "is" && fingerprint.Object != "" {
			appendHistoryEntry(histories, fingerprint, fingerprint.SubjectID, fingerprint.Subject, "asserted_attribute", "", fingerprint.Object, "establish")
		}
		for _, evidenceID := range fingerprint.EvidenceIDs {
			record, exists := recordByID[evidenceID]
			if !exists || !locationStateEvidence(record) {
				continue
			}
			locations, _ := eventTerms(record)
			for _, location := range locations {
				operation := "present"
				if leaveCueRe.MatchString(record.Text) {
					operation = "depart"
				} else if record.EvidenceType == "transition" {
					operation = "arrive"
				}
				for _, participant := range fingerprint.Participants {
					if participant.Role == "source" {
						continue
					}
					appendHistoryEntry(histories, fingerprint, participant.EntityID, participant.EntityName, "location", "physical_presence", location.Text, operation)
				}
			}
		}
	}
	result := make([]types.FingerprintStateHistory, 0, len(histories))
	for _, history := range histories {
		sort.SliceStable(history.Entries, func(i, j int) bool {
			if history.Entries[i].ChapterIndex != history.Entries[j].ChapterIndex {
				return history.Entries[i].ChapterIndex < history.Entries[j].ChapterIndex
			}
			return history.Entries[i].ParagraphIndex < history.Entries[j].ParagraphIndex
		})
		result = append(result, *history)
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := normalizeSemantic(result[i].EntityName + " " + result[i].Property + " " + result[i].ScopeID)
		right := normalizeSemantic(result[j].EntityName + " " + result[j].Property + " " + result[j].ScopeID)
		return left < right
	})
	return result
}

func appendHistoryEntry(histories map[string]*types.FingerprintStateHistory, fingerprint types.ManuscriptFingerprint, entityID, entityName, property, qualifier, value, operation string) {
	entityName, property, value = strings.TrimSpace(entityName), strings.TrimSpace(property), strings.TrimSpace(value)
	if entityName == "" || property == "" || value == "" || !stableHistoryEntity(entityID, entityName) {
		return
	}
	keyEntity := entityID
	if keyEntity == "" {
		keyEntity = normalizeSemantic(entityName)
	}
	key := strings.Join([]string{keyEntity, property, qualifier, fingerprint.Scope.ID}, "\x00")
	history := histories[key]
	if history == nil {
		history = &types.FingerprintStateHistory{ID: stableID("state-history", key), EntityID: entityID, EntityName: entityName, Property: property, Qualifier: qualifier, ScopeID: fingerprint.Scope.ID, Entries: []types.FingerprintStateEntry{}}
		histories[key] = history
	}
	chapter, paragraph := 0, 0
	if len(fingerprint.EvidenceSpans) > 0 {
		chapter, paragraph = fingerprint.EvidenceSpans[0].ChapterIndex, fingerprint.EvidenceSpans[0].ParagraphIndex
	}
	entry := types.FingerprintStateEntry{
		ID: stableID("state-entry", history.ID, fingerprint.ID, normalizeSemantic(value), operation), FingerprintID: fingerprint.ID,
		Value: value, Operation: nonEmpty(operation, "establish"), EpistemicStatus: fingerprint.EpistemicStatus,
		Attribution: fingerprint.Attribution, Scope: fingerprint.Scope, Temporal: fingerprint.Temporal, EvidenceIDs: clone(fingerprint.EvidenceIDs),
		ChapterIndex: chapter, ParagraphIndex: paragraph, Confidence: fingerprint.Confidence,
	}
	for index := range history.Entries {
		prior := &history.Entries[index]
		if normalizeSemantic(prior.Value) == normalizeSemantic(entry.Value) && prior.Operation == entry.Operation {
			prior.EvidenceIDs = appendUnique(prior.EvidenceIDs, entry.EvidenceIDs...)
			if entry.Confidence > prior.Confidence {
				prior.Confidence = entry.Confidence
			}
			return
		}
	}
	history.Entries = append(history.Entries, entry)
}

func stableHistoryEntity(entityID, entityName string) bool {
	if entityID != "" {
		return true
	}
	words := strings.Fields(normalizeSemantic(entityName))
	if len(words) == 0 || len(words) > 8 {
		return false
	}
	switch words[0] {
	case "this", "that", "these", "those", "here", "there", "it", "he", "she", "they", "we", "you", "i", "someone", "something":
		return false
	}
	return true
}

func locationStateEvidence(record types.EvidenceRecord) bool {
	if record.EvidenceType != "transition" && record.EvidenceType != "introduction" && record.EvidenceType != "state" {
		return false
	}
	locations, _ := eventTerms(record)
	return len(locations) > 0 && (presenceRe.MatchString(record.Text) || leaveCueRe.MatchString(record.Text) || record.EvidenceType == "transition")
}

func stateQualifier(kind, operation string) string {
	if kind == "possession" {
		return "physical_custody"
	}
	return ""
}
