package fingerprint

// Continuity inspections (v5): deterministic queries over the frame corpus
// and state ledgers. Narrative scope decides how a conflict is judged —
// the same facts diverging across a dream, simulation, or flashback are
// reported as scope divergence (informational), never flattened into
// ordinary continuity errors; only same-scope conflicts read as likely
// mistakes.

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"draftline/internal/types"
)

func buildInspections(
	frames []types.NarrativeFrame,
	ledgers []types.StateLedger,
	identities []types.NarrativeEventIdentity,
	scopes map[string]types.NarrativeRealityScope,
) []types.NarrativeInspection {
	frameByID := map[string]types.NarrativeFrame{}
	for _, frame := range frames {
		frameByID[frame.ID] = frame
	}
	inspections := []types.NarrativeInspection{}
	inspections = append(inspections, deathThenActivity(frames, ledgers, frameByID)...)
	inspections = append(inspections, impossibleLocations(ledgers, frameByID)...)
	inspections = append(inspections, impossiblePossession(ledgers, frameByID)...)
	inspections = append(inspections, knowledgeBeforeAcquisition(ledgers, frameByID)...)
	inspections = append(inspections, repeatedEventConflicts(identities, frameByID)...)
	inspections = append(inspections, countdownReversals(frames)...)
	inspections = append(inspections, backwardStoryTime(ledgers, frameByID)...)
	inspections = append(inspections, unresolvedObligations(ledgers, frameByID)...)
	inspections = append(inspections, relinquishedWithoutHolding(ledgers, frameByID)...)
	sort.SliceStable(inspections, func(i, j int) bool {
		rank := map[string]int{"error": 0, "warning": 1, "info": 2}
		if rank[inspections[i].Severity] != rank[inspections[j].Severity] {
			return rank[inspections[i].Severity] < rank[inspections[j].Severity]
		}
		return inspections[i].ID < inspections[j].ID
	})
	return inspections
}

func quoteOf(frame types.NarrativeFrame) string {
	if len(frame.EvidenceSpans) > 0 {
		return frame.EvidenceSpans[0].Quote
	}
	return frame.Detail
}

func sideFromEntries(label string, entries []types.StateLedgerEntry, frameByID map[string]types.NarrativeFrame) types.NarrativeInspectionSide {
	side := types.NarrativeInspectionSide{Label: label}
	for _, entry := range entries {
		if frame, ok := frameByID[entry.FrameID]; ok {
			side.FrameIDs = append(side.FrameIDs, frame.ID)
			side.EvidenceSpans = append(side.EvidenceSpans, frame.EvidenceSpans...)
		}
	}
	return side
}

func sideFromFrames(label string, members []types.NarrativeFrame) types.NarrativeInspectionSide {
	side := types.NarrativeInspectionSide{Label: label}
	for _, frame := range members {
		side.FrameIDs = append(side.FrameIDs, frame.ID)
		side.EvidenceSpans = append(side.EvidenceSpans, frame.EvidenceSpans...)
	}
	return side
}

// deathThenActivity flags a character acting within the same scope after
// narration declared them dead there.
func deathThenActivity(frames []types.NarrativeFrame, ledgers []types.StateLedger, frameByID map[string]types.NarrativeFrame) []types.NarrativeInspection {
	inspections := []types.NarrativeInspection{}
	for _, ledger := range ledgers {
		if ledger.Aspect != "life_status" {
			continue
		}
		for _, entry := range ledger.Entries {
			if entry.Value != "dead" || entry.Epistemic != "narration" {
				continue
			}
			for _, frame := range frames {
				if frame.NarrativeOrder <= entry.NarrativeOrder || frame.Scope.ID != entry.Scope.ID {
					continue
				}
				if subjectName(frame) != ledger.EntityName {
					continue
				}
				switch frame.Type {
				case types.FrameLocation, types.FramePossession, types.FrameTransfer, types.FrameDecision, types.FrameGoal, types.FrameEvent:
					deadFrame := frameByID[entry.FrameID]
					inspections = append(inspections, types.NarrativeInspection{
						ID: stableID("inspection", "contradictory_state", entry.ID, frame.ID), Kind: "contradictory_state",
						Severity: "error", ScopeAssessment: "same_scope_likely_error",
						Title:  fmt.Sprintf("%s acts after dying", ledger.EntityName),
						Detail: fmt.Sprintf("Narration establishes %s as dead, yet the same narrative reality later shows them active.", ledger.EntityName),
						Sides: []types.NarrativeInspectionSide{
							sideFromFrames("Declared dead", []types.NarrativeFrame{deadFrame}),
							sideFromFrames("Later activity", []types.NarrativeFrame{frame}),
						},
						Confidence: .85,
					})
					goto nextEntry
				}
			}
		nextEntry:
		}
	}
	return inspections
}

var oppositeDirections = [][2]string{{"north", "south"}, {"east", "west"}, {"uphill", "downhill"}, {"upstream", "downstream"}, {"left", "right"}}

// impossibleLocations flags one entity in two places at once within a scope,
// and contradictory route directions to the same place.
func impossibleLocations(ledgers []types.StateLedger, frameByID map[string]types.NarrativeFrame) []types.NarrativeInspection {
	inspections := []types.NarrativeInspection{}
	for _, ledger := range ledgers {
		if ledger.Aspect != "location" {
			continue
		}
		for i := 0; i < len(ledger.Entries); i++ {
			for j := i + 1; j < len(ledger.Entries); j++ {
				a, b := ledger.Entries[i], ledger.Entries[j]
				if a.Operation != "set" || b.Operation != "set" {
					continue
				}
				sameScope := a.Scope.ID == b.Scope.ID
				assessment := "same_scope_likely_error"
				severity := "error"
				if !sameScope {
					assessment = "cross_scope_divergence"
					severity = "info"
				}
				// Two different places at the same solved story time.
				if a.Value != b.Value && a.Temporal.DayOffset != nil && b.Temporal.DayOffset != nil && *a.Temporal.DayOffset == *b.Temporal.DayOffset && sameScope {
					inspections = append(inspections, types.NarrativeInspection{
						ID: stableID("inspection", "impossible_location", a.ID, b.ID), Kind: "impossible_location",
						Severity: severity, ScopeAssessment: assessment,
						Title:  fmt.Sprintf("%s is in two places at the same story time", ledger.EntityName),
						Detail: fmt.Sprintf("“%s” and “%s” are both placed on the same story day.", a.Value, b.Value),
						Sides: []types.NarrativeInspectionSide{
							sideFromEntries("First placement", []types.StateLedgerEntry{a}, frameByID),
							sideFromEntries("Second placement", []types.StateLedgerEntry{b}, frameByID),
						},
						Confidence: .8,
					})
					continue
				}
				// Same destination reached in opposite directions.
				if a.Value == b.Value {
					lowerA := strings.ToLower(quoteOf(frameByID[a.FrameID]))
					lowerB := strings.ToLower(quoteOf(frameByID[b.FrameID]))
					for _, pair := range oppositeDirections {
						if (strings.Contains(lowerA, pair[0]) && strings.Contains(lowerB, pair[1])) ||
							(strings.Contains(lowerA, pair[1]) && strings.Contains(lowerB, pair[0])) {
							inspections = append(inspections, types.NarrativeInspection{
								ID: stableID("inspection", "route_direction", a.ID, b.ID), Kind: "contradictory_state",
								Severity: severity, ScopeAssessment: assessment,
								Title:  fmt.Sprintf("Route to “%s” changes direction", a.Value),
								Detail: fmt.Sprintf("The same destination is reached %s in one account and %s in another.", pair[0], pair[1]),
								Sides: []types.NarrativeInspectionSide{
									sideFromEntries("First route", []types.StateLedgerEntry{a}, frameByID),
									sideFromEntries("Second route", []types.StateLedgerEntry{b}, frameByID),
								},
								Confidence: .8,
							})
						}
					}
				}
			}
		}
	}
	return inspections
}

// impossiblePossession walks each item's merged possession timeline: a new
// holder appearing while another still holds, without a transfer or
// relinquishment between, is flagged within one scope.
func impossiblePossession(ledgers []types.StateLedger, frameByID map[string]types.NarrativeFrame) []types.NarrativeInspection {
	type holderEvent struct {
		entity string
		entry  types.StateLedgerEntry
	}
	byItem := map[string][]holderEvent{}
	for _, ledger := range ledgers {
		if ledger.Aspect != "possession" || ledger.Qualifier == "" {
			continue
		}
		for _, entry := range ledger.Entries {
			byItem[ledger.Qualifier] = append(byItem[ledger.Qualifier], holderEvent{entity: ledger.EntityName, entry: entry})
		}
	}
	inspections := []types.NarrativeInspection{}
	for item, events := range byItem {
		sort.SliceStable(events, func(i, j int) bool { return events[i].entry.NarrativeOrder < events[j].entry.NarrativeOrder })
		holder := ""
		var holderEntry types.StateLedgerEntry
		for _, event := range events {
			switch event.entry.Operation {
			case "set", "transfer_in":
				if holder != "" && holder != event.entity && holderEntry.Scope.ID == event.entry.Scope.ID {
					inspections = append(inspections, types.NarrativeInspection{
						ID: stableID("inspection", "impossible_possession", holderEntry.ID, event.entry.ID), Kind: "impossible_possession",
						Severity: "error", ScopeAssessment: "same_scope_likely_error",
						Title:  fmt.Sprintf("“%s” changes hands without a hand-off", item),
						Detail: fmt.Sprintf("%s still holds “%s” when %s acquires it; no transfer or relinquishment appears between.", holder, item, event.entity),
						Sides: []types.NarrativeInspectionSide{
							sideFromEntries(holder+" holds it", []types.StateLedgerEntry{holderEntry}, frameByID),
							sideFromEntries(event.entity+" acquires it", []types.StateLedgerEntry{event.entry}, frameByID),
						},
						Confidence: .8,
					})
				}
				holder = event.entity
				holderEntry = event.entry
			case "clear", "transfer_out":
				if holder == event.entity {
					holder = ""
				}
			}
		}
	}
	return inspections
}

// knowledgeBeforeAcquisition flags a character firmly knowing a fact before
// the manuscript shows them learning it.
func knowledgeBeforeAcquisition(ledgers []types.StateLedger, frameByID map[string]types.NarrativeFrame) []types.NarrativeInspection {
	inspections := []types.NarrativeInspection{}
	for _, ledger := range ledgers {
		if ledger.Aspect != "knowledge" {
			continue
		}
		var knowsEarly *types.StateLedgerEntry
		for index := range ledger.Entries {
			entry := &ledger.Entries[index]
			frame := frameByID[entry.FrameID]
			if knowsEarly == nil && entry.Operation == "set" && frame.Value == "knows" {
				knowsEarly = entry
				continue
			}
			if knowsEarly != nil && entry.Operation == "set" && frame.Value == "learned" && entry.Scope.ID == knowsEarly.Scope.ID {
				inspections = append(inspections, types.NarrativeInspection{
					ID: stableID("inspection", "knowledge_before_acquisition", knowsEarly.ID, entry.ID), Kind: "knowledge_before_acquisition",
					Severity: "warning", ScopeAssessment: "same_scope_likely_error",
					Title:  fmt.Sprintf("%s knows “%s” before learning it", ledger.EntityName, knowsEarly.Value),
					Detail: "The character references this knowledge before the scene where they acquire it.",
					Sides: []types.NarrativeInspectionSide{
						sideFromEntries("Knows it here", []types.StateLedgerEntry{*knowsEarly}, frameByID),
						sideFromEntries("Learns it later", []types.StateLedgerEntry{*entry}, frameByID),
					},
					Confidence: .75,
				})
				knowsEarly = nil
			}
		}
	}
	return inspections
}

// repeatedEventConflicts projects conflicted event identities, judged by
// scope: same-scope conflicts are likely errors; divergence across dreams,
// simulations, or retellings is informational by design.
func repeatedEventConflicts(identities []types.NarrativeEventIdentity, frameByID map[string]types.NarrativeFrame) []types.NarrativeInspection {
	inspections := []types.NarrativeInspection{}
	for _, identity := range identities {
		if identity.Status != "conflicted" {
			continue
		}
		assessment := "same_scope_likely_error"
		severity := "warning"
		if len(identity.ScopeIDs) > 1 {
			assessment = "cross_scope_divergence"
			severity = "info"
		} else {
			epistemics := map[string]bool{}
			for _, frameID := range identity.FrameIDs {
				epistemics[frameByID[frameID].Epistemic] = true
			}
			if len(epistemics) > 1 {
				assessment = "attributed_account_difference"
				severity = "info"
			}
		}
		sides := []types.NarrativeInspectionSide{}
		for _, property := range identity.Properties {
			for _, value := range property.Values {
				members := []types.NarrativeFrame{}
				for _, frameID := range value.FrameIDs {
					members = append(members, frameByID[frameID])
				}
				sides = append(sides, sideFromFrames("Account: "+value.Value, members))
			}
		}
		subject := ""
		if len(identity.Participants) > 0 {
			subject = identity.Participants[0].EntityName
		}
		inspections = append(inspections, types.NarrativeInspection{
			ID: stableID("inspection", "repeated_event_conflict", identity.ID), Kind: "repeated_event_conflict",
			Severity: severity, ScopeAssessment: assessment,
			Title:  fmt.Sprintf("Accounts of the same %s event differ for %s", identity.EventClass, subject),
			Detail: "The manuscript describes one underlying event more than once with differing details.",
			Sides:  sides, Confidence: identity.Confidence,
		})
	}
	return inspections
}

var countdownRe = regexp.MustCompile(`(?i)\b(one|two|three|four|five|six|seven|eight|nine|ten|\d+)\s+(days?|hours?|weeks?)\s+(left|remaining|remained until|until|to go)\b`)

func countdownNumber(word string) float64 {
	words := map[string]float64{"one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10}
	if value, ok := words[strings.ToLower(word)]; ok {
		return value
	}
	if value, err := strconv.ParseFloat(word, 64); err == nil {
		return value
	}
	return -1
}

// countdownReversals flags a countdown that moves the wrong way within one
// scope ("three days left" followed later by "five days left").
func countdownReversals(frames []types.NarrativeFrame) []types.NarrativeInspection {
	type countdown struct {
		frame types.NarrativeFrame
		value float64
		unit  string
	}
	perScope := map[string][]countdown{}
	seenEvidence := map[string]bool{}
	for _, frame := range frames {
		for _, span := range frame.EvidenceSpans {
			if seenEvidence[span.EvidenceID] {
				continue
			}
			match := countdownRe.FindStringSubmatch(span.Quote)
			if match == nil {
				continue
			}
			seenEvidence[span.EvidenceID] = true
			value := countdownNumber(match[1])
			if value < 0 {
				continue
			}
			unit := strings.TrimSuffix(strings.ToLower(match[2]), "s")
			perScope[frame.Scope.ID+"\x00"+unit] = append(perScope[frame.Scope.ID+"\x00"+unit], countdown{frame: frame, value: value, unit: unit})
		}
	}
	inspections := []types.NarrativeInspection{}
	for _, series := range perScope {
		sort.SliceStable(series, func(i, j int) bool { return series[i].frame.NarrativeOrder < series[j].frame.NarrativeOrder })
		for index := 1; index < len(series); index++ {
			if series[index].value > series[index-1].value {
				inspections = append(inspections, types.NarrativeInspection{
					ID: stableID("inspection", "countdown", series[index-1].frame.ID, series[index].frame.ID), Kind: "timeline_conflict",
					Severity: "error", ScopeAssessment: "same_scope_likely_error",
					Title:  "A countdown runs backward",
					Detail: fmt.Sprintf("%.0f %ss remain, then later %.0f %ss remain within the same narrative reality.", series[index-1].value, series[index].unit, series[index].value, series[index].unit),
					Sides: []types.NarrativeInspectionSide{
						sideFromFrames("Earlier count", []types.NarrativeFrame{series[index-1].frame}),
						sideFromFrames("Later count", []types.NarrativeFrame{series[index].frame}),
					},
					Confidence: .85,
				})
			}
		}
	}
	return inspections
}

// backwardStoryTime flags solved story days that regress within one scope's
// timeline ledger while the narration claims continuous present action.
func backwardStoryTime(ledgers []types.StateLedger, frameByID map[string]types.NarrativeFrame) []types.NarrativeInspection {
	inspections := []types.NarrativeInspection{}
	for _, ledger := range ledgers {
		if ledger.Aspect != "timeline" {
			continue
		}
		for index := 1; index < len(ledger.Entries); index++ {
			previous, current := ledger.Entries[index-1], ledger.Entries[index]
			if previous.Temporal.DayOffset == nil || current.Temporal.DayOffset == nil {
				continue
			}
			if *current.Temporal.DayOffset < *previous.Temporal.DayOffset && current.Scope.ID == previous.Scope.ID && current.Scope.Kind == "current" {
				inspections = append(inspections, types.NarrativeInspection{
					ID: stableID("inspection", "backward_time", previous.ID, current.ID), Kind: "timeline_conflict",
					Severity: "warning", ScopeAssessment: "same_scope_likely_error",
					Title:  "Story time moves backward in the primary reality",
					Detail: fmt.Sprintf("Story day %.0f is followed by story day %.0f without a scope change.", *previous.Temporal.DayOffset, *current.Temporal.DayOffset),
					Sides: []types.NarrativeInspectionSide{
						sideFromEntries("Earlier day", []types.StateLedgerEntry{previous}, frameByID),
						sideFromEntries("Later, earlier-dated day", []types.StateLedgerEntry{current}, frameByID),
					},
					Confidence: .7,
				})
			}
		}
	}
	return inspections
}

// unresolvedObligations lists obligations opened and never closed — open
// story debts, informational by design.
func unresolvedObligations(ledgers []types.StateLedger, frameByID map[string]types.NarrativeFrame) []types.NarrativeInspection {
	inspections := []types.NarrativeInspection{}
	for _, ledger := range ledgers {
		if ledger.Aspect != "obligation" {
			continue
		}
		closed := false
		for _, entry := range ledger.Entries {
			if entry.Operation == "close" {
				closed = true
			}
		}
		if closed {
			continue
		}
		first := ledger.Entries[0]
		inspections = append(inspections, types.NarrativeInspection{
			ID: stableID("inspection", "unresolved_obligation", ledger.ID), Kind: "unresolved_obligation",
			Severity: "info", ScopeAssessment: "same_scope_likely_error",
			Title:  fmt.Sprintf("%s’s obligation is never resolved", ledger.EntityName),
			Detail: fmt.Sprintf("“%s” is opened and the manuscript never shows it fulfilled or released.", first.Value),
			Sides: []types.NarrativeInspectionSide{
				sideFromEntries("Opened here", []types.StateLedgerEntry{first}, frameByID),
			},
			Confidence: .65,
		})
	}
	return inspections
}

// relinquishedWithoutHolding flags giving away or dropping an item the
// character was never shown to hold — a causal prerequisite failure.
func relinquishedWithoutHolding(ledgers []types.StateLedger, frameByID map[string]types.NarrativeFrame) []types.NarrativeInspection {
	inspections := []types.NarrativeInspection{}
	for _, ledger := range ledgers {
		if ledger.Aspect != "possession" {
			continue
		}
		held := false
		for _, entry := range ledger.Entries {
			switch entry.Operation {
			case "set", "transfer_in":
				held = true
			case "clear", "transfer_out":
				if !held {
					inspections = append(inspections, types.NarrativeInspection{
						ID: stableID("inspection", "prerequisite", entry.ID), Kind: "causal_prerequisite_failure",
						Severity: "warning", ScopeAssessment: "same_scope_likely_error",
						Title:  fmt.Sprintf("%s parts with “%s” without ever holding it", ledger.EntityName, entry.Value),
						Detail: "The item leaves the character's possession before the manuscript shows them acquiring it.",
						Sides: []types.NarrativeInspectionSide{
							sideFromEntries("Relinquished here", []types.StateLedgerEntry{entry}, frameByID),
						},
						Confidence: .7,
					})
				}
				held = false
			}
		}
	}
	return inspections
}
