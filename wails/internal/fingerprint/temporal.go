package fingerprint

import (
	"regexp"
	"strconv"
	"strings"

	"draftline/internal/types"
)

var (
	weekdayRe  = regexp.MustCompile(`(?i)\b(sunday|monday|tuesday|wednesday|thursday|friday|saturday)\b`)
	relativeRe = regexp.MustCompile(`(?i)\b(one|two|three|four|five|six|seven|eight|nine|ten|\d+)\s+(day|week|month|year)s?\s+(ago|earlier|before|later|after)\b`)
	// Relative phrases such as "two years ago" are temporal evidence, not a
	// license to relocate the containing scene or every later paragraph.
	pastCueRe       = regexp.MustCompile(`(?i)\b(?:flashback|years? earlier|months? earlier|days? earlier|back then)\b`)
	dreamCueRe      = regexp.MustCompile(`(?i)\b(?:dream|dreamed|dreaming|nightmare|dream world)\b`)
	wakeCueRe       = regexp.MustCompile(`(?i)\b(?:awoke|woke up|woke)\b`)
	simulationCueRe = regexp.MustCompile(`(?i)\b(?:simulation|sim reset|reset complete|rendering|buffer_overflow)\b`)
)

func inferContexts(records []types.EvidenceRecord, authored []types.StoryContext) ([]types.StoryContext, map[string]string) {
	primary := types.StoryContext{ID: "context-primary", Kind: "primary", Label: "Primary story time", Confidence: 1, Source: "inferred"}
	result := []types.StoryContext{primary}
	byID := map[string]bool{primary.ID: true}
	for _, context := range authored {
		if context.ID == "" {
			context.ID = stableID("context", context.Kind, context.Label)
		}
		context.Source = "author"
		context.Confidence = 1
		if !byID[context.ID] {
			result = append(result, context)
			byID[context.ID] = true
		}
	}
	contextByEvidence := map[string]string{}
	chapterContext := map[int]string{}
	for _, record := range records {
		contextID := chapterContext[record.ChapterIndex]
		if contextID == "" {
			contextID = primary.ID
		}
		kind, label, confidence := contextCue(record.Text)
		if kind != "" {
			contextID = stableID("context", strconv.Itoa(record.ChapterIndex), kind)
			if !byID[contextID] {
				result = append(result, types.StoryContext{ID: contextID, Kind: kind, Label: label, ParentID: primary.ID, Confidence: confidence, Source: "inferred"})
				byID[contextID] = true
			}
			chapterContext[record.ChapterIndex] = contextID
		}
		if wakeCueRe.MatchString(record.Text) && kind == "" {
			contextID = primary.ID
			chapterContext[record.ChapterIndex] = primary.ID
		}
		contextByEvidence[record.ID] = contextID
		for index := range result {
			if result[index].ID == contextID {
				result[index].EvidenceIDs = append(result[index].EvidenceIDs, record.ID)
			}
		}
	}
	return result, contextByEvidence
}

func contextCue(text string) (string, string, float64) {
	switch {
	case simulationCueRe.MatchString(text):
		return "simulation", "Simulation", .9
	case dreamCueRe.MatchString(text):
		return "dream", "Possible dream", .72
	case pastCueRe.MatchString(text):
		return "past", "Earlier story time", .78
	default:
		return "", "", 0
	}
}

func inferTemporalConstraints(records []types.EvidenceRecord, contexts map[string]string) []types.TemporalConstraint {
	result := []types.TemporalConstraint{}
	lastByContext := map[string]string{}
	for _, record := range records {
		contextID := contexts[record.ID]
		if previous := lastByContext[contextID]; previous != "" {
			result = append(result, types.TemporalConstraint{ID: stableID("time", previous, record.ID, "before"), FromEvidenceID: previous, ToEvidenceID: record.ID, Relation: "before", Label: "Narrative order within this time context", Posture: "inferred", Confidence: .55, Source: "auto"})
		}
		lastByContext[contextID] = record.ID
		for _, expression := range record.TimeExpressions {
			if days, ok := relativeDays(expression); ok {
				offset := days
				result = append(result, types.TemporalConstraint{ID: stableID("time", record.ID, expression), FromEvidenceID: record.ID, Relation: "offset", OffsetDays: &offset, Label: expression, Posture: temporalPosture(record.Text), Confidence: .8, Source: "auto"})
			}
		}
		if match := weekdayRe.FindString(record.Text); match != "" {
			result = append(result, types.TemporalConstraint{ID: stableID("time", record.ID, strings.ToLower(match)), FromEvidenceID: record.ID, Relation: "same", Label: match, Posture: temporalPosture(record.Text), Confidence: .9, Source: "auto"})
		}
	}
	return result
}

func solveTemporal(records []types.EvidenceRecord, contexts map[string]string, constraints []types.TemporalConstraint) (map[string]types.StoryTime, []types.FingerprintDiagnostic) {
	points := map[string]types.StoryTime{}
	diagnostics := []types.FingerprintDiagnostic{}
	anchors := map[string]float64{}
	var lastNarrativeAnchor *float64
	for _, record := range records {
		contextID := contexts[record.ID]
		point := types.StoryTime{ContextID: contextID, Precision: "unknown", Confidence: .35}
		if weekday := weekdayRe.FindString(record.Text); weekday != "" {
			day := weekdayNumber(weekday)
			point.DayOffset = &day
			point.EarliestDay = &day
			point.LatestDay = &day
			point.Label = strings.Title(strings.ToLower(weekday))
			point.Precision = "day"
			point.Confidence = .9
			if prior, ok := anchors[contextID]; ok && prior != day {
				// A second weekday is valid; keep it as a new anchor rather than forcing manuscript order.
			}
			anchors[contextID] = day
			anchorCopy := day
			lastNarrativeAnchor = &anchorCopy
		} else {
			for _, expression := range record.TimeExpressions {
				if days, ok := relativeDays(expression); ok {
					point.Label = expression
					point.Precision = "relative"
					point.Confidence = .72
					if anchor, exists := anchors[contextID]; exists {
						value := anchor + days
						point.DayOffset = &value
						point.EarliestDay = &value
						point.LatestDay = &value
					} else if lastNarrativeAnchor != nil {
						value := *lastNarrativeAnchor + days
						point.DayOffset = &value
						point.EarliestDay = &value
						point.LatestDay = &value
					}
					break
				}
			}
		}
		points[record.ID] = point
	}
	return points, diagnostics
}

func relativeDays(text string) (float64, bool) {
	match := relativeRe.FindStringSubmatch(text)
	if len(match) != 4 {
		return 0, false
	}
	count := wordNumber(match[1])
	multiplier := map[string]float64{"day": 1, "week": 7, "month": 30, "year": 365}[strings.ToLower(match[2])]
	offset := float64(count) * multiplier
	direction := strings.ToLower(match[3])
	if direction == "ago" || direction == "earlier" || direction == "before" {
		offset = -offset
	}
	return offset, true
}

func wordNumber(value string) int {
	if number, err := strconv.Atoi(value); err == nil {
		return number
	}
	return map[string]int{"one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10}[strings.ToLower(value)]
}

func weekdayNumber(value string) float64 {
	return map[string]float64{"sunday": 0, "monday": 1, "tuesday": 2, "wednesday": 3, "thursday": 4, "friday": 5, "saturday": 6}[strings.ToLower(value)]
}

func temporalPosture(text string) string {
	lower := strings.ToLower(text)
	if strings.ContainsAny(text, "\"“”") {
		return "claimed"
	}
	if strings.Contains(lower, "remember") || strings.Contains(lower, "recall") {
		return "remembered"
	}
	return "narrated"
}
