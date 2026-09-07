package fingerprint

// Canonical state ledgers (v5): per-entity histories folded from typed
// frames — the basis of continuity checking. A ledger never averages or
// discards conflicting values; every account remains an entry with its own
// scope, epistemic posture, and evidence.

import (
	"regexp"
	"sort"
	"strings"

	"draftline/internal/types"
)

var articleRe = regexp.MustCompile(`(?i)^(the|a|an|his|her|their|its|my|your|our)\s+`)
var nonKeyRe = regexp.MustCompile(`[^\p{L}\p{N} ]+`)

// qualifierKey normalizes a verbatim phrase into a stable ledger qualifier:
// lowercased, article-stripped, punctuation-free, capped at six words.
func qualifierKey(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	for {
		next := articleRe.ReplaceAllString(text, "")
		if next == text {
			break
		}
		text = next
	}
	text = nonKeyRe.ReplaceAllString(text, "")
	words := strings.Fields(text)
	if len(words) > 6 {
		words = words[:6]
	}
	return strings.Join(words, " ")
}

func subjectName(frame types.NarrativeFrame) string {
	for _, p := range frame.Participants {
		if p.Role == "subject" {
			return p.EntityName
		}
	}
	return ""
}

func participantNamed(frame types.NarrativeFrame, role string) string {
	for _, p := range frame.Participants {
		if p.Role == role {
			return p.EntityName
		}
	}
	return ""
}

type ledgerKey struct {
	entity    string
	aspect    string
	qualifier string
}

func buildLedgers(frames []types.NarrativeFrame) []types.StateLedger {
	ledgers := map[ledgerKey]*types.StateLedger{}
	order := []ledgerKey{}

	add := func(entity, aspect, qualifier string, frame types.NarrativeFrame, value, operation string) {
		if entity == "" || value == "" {
			return
		}
		key := ledgerKey{entity: entity, aspect: aspect, qualifier: qualifier}
		ledger := ledgers[key]
		if ledger == nil {
			ledger = &types.StateLedger{
				ID: stableID("ledger", entity, aspect, qualifier), EntityName: entity,
				Aspect: aspect, Qualifier: qualifier,
			}
			ledgers[key] = ledger
			order = append(order, key)
		}
		ledger.Entries = append(ledger.Entries, types.StateLedgerEntry{
			ID: stableID("entry", frame.ID, aspect, operation), FrameID: frame.ID,
			Value: value, Operation: operation,
			Epistemic: frame.Epistemic, Attribution: frame.Attribution,
			Scope: frame.Scope, Temporal: frame.Temporal,
			ChapterIndex: frame.ChapterIndex, ParagraphIndex: frame.ParagraphIndex,
			NarrativeOrder: frame.NarrativeOrder,
			EvidenceIDs:    clone(frame.EvidenceIDs), Confidence: frame.Confidence,
		})
	}

	for _, frame := range frames {
		subject := subjectName(frame)
		switch frame.Type {
		case types.FrameLocation:
			value := participantNamed(frame, "place")
			if value == "" {
				value = frame.Detail
			}
			operation := "set"
			if frame.Value == "clear" {
				operation = "clear"
			}
			add(subject, "location", "", frame, value, operation)
		case types.FramePossession:
			item := participantNamed(frame, "item")
			if item == "" {
				item = frame.Detail
			}
			operation := "set"
			if frame.Value == "relinquished" || frame.Polarity == "negated" {
				operation = "clear"
			}
			add(subject, "possession", qualifierKey(item), frame, item, operation)
		case types.FrameTransfer:
			item := participantNamed(frame, "item")
			source := participantNamed(frame, "source")
			recipient := participantNamed(frame, "recipient")
			add(source, "possession", qualifierKey(item), frame, item, "transfer_out")
			add(recipient, "possession", qualifierKey(item), frame, item, "transfer_in")
		case types.FrameInjury:
			operation := "set"
			if frame.Polarity == "negated" {
				operation = "clear"
			}
			add(subject, "condition", "", frame, frame.Detail, operation)
		case types.FrameLifeStatus:
			add(subject, "life_status", "", frame, frame.Value, "set")
		case types.FrameKnowledge:
			operation := "set"
			if frame.Polarity == "negated" {
				operation = "clear"
			}
			add(subject, "knowledge", qualifierKey(frame.Detail), frame, frame.Detail, operation)
		case types.FrameBelief:
			add(subject, "knowledge", qualifierKey(frame.Detail), frame, frame.Detail, "set")
		case types.FrameGoal:
			add(subject, "goal", qualifierKey(frame.Detail), frame, frame.Detail, "open")
		case types.FrameObligation:
			add(subject, "obligation", qualifierKey(frame.Detail), frame, frame.Detail, "open")
		case types.FrameRelationship:
			counterparty := participantNamed(frame, "counterparty")
			if counterparty == "" {
				continue // an unresolved counterparty stays a frame, not state
			}
			add(subject, "relationship", counterparty, frame, frame.Value, "set")
		case types.FrameAccess:
			operation := "set"
			if frame.Value == "revoked" || frame.Polarity == "negated" {
				operation = "clear"
			}
			add(subject, "access", qualifierKey(frame.Detail), frame, frame.Detail, operation)
		}
		// Timeline ledger: any frame with a solved story day contributes to
		// the per-scope chronology history.
		if frame.Temporal.DayOffset != nil {
			label := frame.Temporal.Label
			if label == "" {
				label = frame.Detail
			}
			add("story", "timeline", frame.Scope.ID, frame, label, "set")
		}
	}

	result := make([]types.StateLedger, 0, len(order))
	for _, key := range order {
		ledger := ledgers[key]
		sort.SliceStable(ledger.Entries, func(i, j int) bool {
			return ledger.Entries[i].NarrativeOrder < ledger.Entries[j].NarrativeOrder
		})
		result = append(result, *ledger)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].EntityName != result[j].EntityName {
			return result[i].EntityName < result[j].EntityName
		}
		if result[i].Aspect != result[j].Aspect {
			return result[i].Aspect < result[j].Aspect
		}
		return result[i].Qualifier < result[j].Qualifier
	})
	return result
}
