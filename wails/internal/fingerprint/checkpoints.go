package fingerprint

import (
	"strings"

	"draftline/internal/types"
)

func evaluateCheckpoints(checkpoints []types.StoryCheckpoint, events []types.FingerprintEvent, records []types.EvidenceRecord) ([]types.StoryCheckpoint, []types.FingerprintDiagnostic) {
	recordByID := map[string]types.EvidenceRecord{}
	for _, record := range records {
		recordByID[record.ID] = record
	}
	diagnostics := []types.FingerprintDiagnostic{}
	for index := range checkpoints {
		checkpoint := &checkpoints[index]
		matched := map[string]bool{}
		for requirementIndex := range checkpoint.Requirements {
			requirement := &checkpoint.Requirements[requirementIndex]
			for _, event := range events {
				if requirementMatches(*requirement, event, recordByID) {
					requirement.Satisfied = true
					requirement.EvidenceID = event.EvidenceIDs[0]
					matched[event.ID] = true
					break
				}
			}
		}
		for _, negative := range checkpoint.NegativeConditions {
			for _, event := range events {
				if requirementMatches(negative, event, recordByID) && violatesWindow(*checkpoint, event) {
					diagnostics = append(diagnostics, checkpointDiagnostic(*checkpoint, event, "checkpoint_too_early", "A negative checkpoint condition appears before its allowed point."))
				}
			}
		}
		checkpoint.MatchedEventIDs = nil
		for _, event := range events {
			if matched[event.ID] {
				checkpoint.MatchedEventIDs = append(checkpoint.MatchedEventIDs, event.ID)
			}
		}
		satisfied := 0
		for _, requirement := range checkpoint.Requirements {
			if requirement.Satisfied {
				satisfied++
			}
		}
		if len(checkpoint.Requirements) > 0 && satisfied == len(checkpoint.Requirements) {
			if checkpoint.Status != "fulfilled" {
				checkpoint.Status = "proposed"
			}
		} else if satisfied > 0 {
			checkpoint.Status = "partial"
		} else if checkpoint.Status == "" {
			checkpoint.Status = "planned"
		}
	}
	return checkpoints, diagnostics
}

func requirementMatches(requirement types.CheckpointRequirement, event types.FingerprintEvent, records map[string]types.EvidenceRecord) bool {
	text := strings.ToLower(event.Summary + " " + eventSourceText(event, records))
	if requirement.EntityID != "" {
		found := false
		for _, id := range event.CharacterIDs {
			if id == requirement.EntityID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	for _, value := range []string{requirement.Text, requirement.Predicate, requirement.Object} {
		for token := range meaningfulTokens(value) {
			if !strings.Contains(text, token) {
				return false
			}
		}
	}
	return requirement.Text != "" || requirement.Predicate != "" || requirement.Object != "" || requirement.EntityID != ""
}

func violatesWindow(checkpoint types.StoryCheckpoint, event types.FingerprintEvent) bool {
	return checkpoint.AfterChapter != nil && event.ChapterIndex < *checkpoint.AfterChapter || checkpoint.BeforeChapter != nil && event.ChapterIndex > *checkpoint.BeforeChapter
}

func checkpointDiagnostic(checkpoint types.StoryCheckpoint, event types.FingerprintEvent, kind, detail string) types.FingerprintDiagnostic {
	return types.FingerprintDiagnostic{ID: stableID("diagnostic", checkpoint.ID, event.ID, kind), Kind: kind, Severity: "review", Title: checkpoint.Title, Detail: detail, EventIDs: []string{event.ID}, EvidenceIDs: clone(event.EvidenceIDs), ChapterIndices: []int{event.ChapterIndex}, Confidence: .9}
}
