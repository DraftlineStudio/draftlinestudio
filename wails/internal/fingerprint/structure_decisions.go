package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"draftline/internal/types"
)

func reconcileStructureDecisions(model *types.StoryFingerprint, records []types.EvidenceRecord) {
	if len(model.AuthorModel.StructureDecisions) == 0 {
		return
	}
	recordByID := make(map[string]types.EvidenceRecord, len(records))
	blockParts := map[string][]string{}
	for _, record := range records {
		recordByID[record.ID] = record
		key := structureBlockKey(record.ChapterID, record.ParagraphIndex)
		blockParts[key] = append(blockParts[key], normalizeFingerprintText(record.Text))
	}
	blockHashes := map[string]string{}
	for key, parts := range blockParts {
		blockHashes[key] = structureDependencyHash(parts)
	}

	for i := range model.AuthorModel.StructureDecisions {
		decision := &model.AuthorModel.StructureDecisions[i]
		decision.ChangedDependencyIDs = nil
		if len(decision.Dependencies) == 0 {
			present := 0
			for _, id := range decision.EvidenceIDs {
				if _, ok := recordByID[id]; ok {
					present++
				}
			}
			switch {
			case len(decision.EvidenceIDs) == 0:
				decision.Status = "conflict"
			case present == len(decision.EvidenceIDs):
				decision.Status = "active"
			case present == 0:
				decision.Status = "orphaned"
			default:
				decision.Status = "conflict"
			}
			continue
		}

		missing, changed := 0, 0
		for _, dependency := range decision.Dependencies {
			currentHash, exists := blockHashes[structureBlockKey(dependency.ChapterID, dependency.ParagraphIndex)]
			if !exists {
				missing++
				decision.ChangedDependencyIDs = appendUnique(decision.ChangedDependencyIDs, dependency.EvidenceID)
				continue
			}
			if dependency.ContentHash == "" || currentHash != dependency.ContentHash {
				changed++
				decision.ChangedDependencyIDs = appendUnique(decision.ChangedDependencyIDs, dependency.EvidenceID)
			}
		}
		switch {
		case changed > 0:
			decision.Status = "conflict"
		case missing == len(decision.Dependencies):
			decision.Status = "orphaned"
		case missing > 0:
			decision.Status = "conflict"
		default:
			decision.Status = "active"
		}
	}
}

func structureDecisionDiagnostics(decisions []types.StructureAuthorDecision) []types.FingerprintDiagnostic {
	result := []types.FingerprintDiagnostic{}
	for _, decision := range decisions {
		if decision.Status != "conflict" && decision.Status != "orphaned" {
			continue
		}
		detail := "The source evidence for an author structural decision no longer exists. Review or dismiss the decision."
		if decision.Status == "conflict" {
			detail = "Source evidence used by an author structural decision changed. The decision was not silently applied to the rewritten passage."
		}
		result = append(result, types.FingerprintDiagnostic{
			ID:   stableID("diagnostic", "structure-decision", decision.ID, decision.Status),
			Kind: "structure_decision_" + decision.Status, Severity: "review",
			Title: "Story-structure decision needs review", Detail: detail,
			EvidenceIDs: clone(decision.EvidenceIDs), Confidence: 1,
		})
	}
	return result
}

func structureBlockKey(chapterID string, paragraph int) string {
	return fmt.Sprintf("%s\x00%d", chapterID, paragraph)
}

func structureDependencyHash(parts []string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(hash[:10])
}
