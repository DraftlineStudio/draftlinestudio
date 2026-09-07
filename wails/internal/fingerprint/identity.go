package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"draftline/internal/types"
)

func stableID(kind string, parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return kind + "-" + hex.EncodeToString(hash[:8])
}

func applyEvidenceContextCorrections(corrections []types.FingerprintCorrection, contexts map[string]string) {
	for _, correction := range corrections {
		if correction.Kind != "context" || correction.Value == "" {
			continue
		}
		for _, evidenceID := range correction.EvidenceIDs {
			if _, exists := contexts[evidenceID]; exists {
				contexts[evidenceID] = correction.Value
			}
		}
	}
}

func applyEventCorrections(events []types.FingerprintEvent, corrections []types.FingerprintCorrection) {
	for _, correction := range corrections {
		for index := range events {
			event := &events[index]
			if correction.TargetID != event.ID && evidenceSimilarity(correction.EvidenceIDs, event.EvidenceIDs) == 0 {
				continue
			}
			switch correction.Kind {
			case "context":
				if correction.Value != "" {
					event.ContextID = correction.Value
					event.StoryTime.ContextID = correction.Value
				}
			case "story_day":
				if value, err := strconv.ParseFloat(correction.Value, 64); err == nil {
					event.StoryTime.DayOffset = &value
					event.StoryTime.EarliestDay = &value
					event.StoryTime.LatestDay = &value
					event.StoryTime.Precision = "exact"
					event.StoryTime.Confidence = 1
				}
			case "summary":
				if correction.Value != "" {
					event.AuthorSummary = correction.Value
					event.Summary = correction.Value
				}
			case "importance":
				if value, err := strconv.ParseFloat(correction.Value, 64); err == nil {
					if value < 0 {
						value = 0
					}
					if value > 1 {
						value = 1
					}
					event.Importance = value
				}
			}
		}
	}
}

func applyAssertionCorrections(assertions []types.StoryAssertion, corrections []types.FingerprintCorrection) {
	for _, correction := range corrections {
		for index := range assertions {
			assertion := &assertions[index]
			if correction.TargetID != assertion.ID && evidenceSimilarity(correction.EvidenceIDs, assertion.EvidenceIDs) == 0 {
				continue
			}
			switch correction.Kind {
			case "context":
				if correction.Value != "" {
					assertion.ContextID = correction.Value
					assertion.Scope.ID = correction.Value
					assertion.Scope.Kind = "uncertain"
					assertion.Scope.Label = "Author-corrected context"
					assertion.Scope.Confidence = 1
				}
			case "story_day":
				if value, err := strconv.ParseFloat(correction.Value, 64); err == nil {
					assertion.Temporal.DayOffset = &value
					assertion.Temporal.EarliestDay = &value
					assertion.Temporal.LatestDay = &value
					assertion.Temporal.Precision = "exact"
					assertion.Temporal.Confidence = 1
				}
			}
		}
	}
}

func applyManuscriptFingerprintCorrections(fingerprints []types.ManuscriptFingerprint, corrections []types.FingerprintCorrection) {
	for _, correction := range corrections {
		for index := range fingerprints {
			fingerprint := &fingerprints[index]
			if correction.TargetID != fingerprint.ID && evidenceSimilarity(correction.EvidenceIDs, fingerprint.EvidenceIDs) == 0 {
				continue
			}
			switch correction.Kind {
			case "context":
				if correction.Value != "" {
					fingerprint.Scope.ID = correction.Value
					fingerprint.Scope.Kind = "uncertain"
					fingerprint.Scope.Label = "Author-corrected context"
					fingerprint.Scope.Confidence = 1
				}
			case "story_day":
				if value, err := strconv.ParseFloat(correction.Value, 64); err == nil {
					fingerprint.Temporal.DayOffset = &value
					fingerprint.Temporal.EarliestDay = &value
					fingerprint.Temporal.LatestDay = &value
					fingerprint.Temporal.Precision = "exact"
					fingerprint.Temporal.Confidence = 1
				}
			case "summary":
				if correction.Value != "" {
					fingerprint.Statement = correction.Value
				}
			}
		}
	}
}

func reconcileCorrections(model *types.StoryFingerprint) {
	known := map[string]bool{}
	for _, event := range model.Events {
		known[event.ID] = true
		for _, evidenceID := range event.EvidenceIDs {
			known[evidenceID] = true
		}
	}
	for _, assertion := range model.Assertions {
		known[assertion.ID] = true
		for _, evidenceID := range assertion.EvidenceIDs {
			known[evidenceID] = true
		}
	}
	for _, fingerprint := range model.Fingerprints {
		known[fingerprint.ID] = true
	}
	for _, development := range model.NarrativeDevelopments {
		known[development.ID] = true
	}
	for _, identity := range model.EventIdentities {
		known[identity.ID] = true
	}
	for _, context := range model.Contexts {
		known[context.ID] = true
	}
	for index := range model.AuthorModel.Corrections {
		correction := &model.AuthorModel.Corrections[index]
		active := known[correction.TargetID]
		for _, evidenceID := range correction.EvidenceIDs {
			active = active || known[evidenceID]
		}
		if active {
			correction.Status = "active"
		} else {
			correction.Status = "orphaned"
		}
	}
}
