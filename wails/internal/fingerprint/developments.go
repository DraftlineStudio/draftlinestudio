package fingerprint

// Narrative development synthesis (v5). Developments are story-level changes
// synthesized from clusters of frames and ledger transitions — they, not raw
// frames, are the future input to PlotWalker. Every rule is deterministic,
// every summary is a template around verbatim source text, and every
// development lists the mechanical basis it was synthesized from.

import (
	"fmt"
	"sort"

	"draftline/internal/types"
)

func synthesizeDevelopments(
	frames []types.NarrativeFrame,
	ledgers []types.StateLedger,
	identities []types.NarrativeEventIdentity,
) []types.NarrativeDevelopment {
	developments := []types.NarrativeDevelopment{}
	add := func(kind, summary, basis string, frame types.NarrativeFrame, confidence float64) {
		developments = append(developments, types.NarrativeDevelopment{
			ID: stableID("development", kind, frame.ID), Kind: kind, Summary: summary,
			Basis:    []string{basis},
			Entities: frame.Participants, FrameIDs: []string{frame.ID},
			EvidenceIDs: clone(frame.EvidenceIDs), ScopeID: frame.Scope.ID,
			ChapterIndex: frame.ChapterIndex, NarrativeOrder: frame.NarrativeOrder,
			Confidence: confidence,
		})
	}
	frameByID := map[string]types.NarrativeFrame{}
	for _, frame := range frames {
		frameByID[frame.ID] = frame
	}

	// Ledger-driven rules.
	goalsSeen := map[string]int{}
	mysterySeen := map[string]bool{}
	for _, ledger := range ledgers {
		switch ledger.Aspect {
		case "knowledge":
			chapters := map[int]bool{}
			for _, entry := range ledger.Entries {
				chapters[entry.ChapterIndex] = true
			}
			first := ledger.Entries[0]
			frame := frameByID[first.FrameID]
			if len(chapters) >= 2 && first.Operation == "set" && frame.Epistemic == "narration" {
				add("major_discovery",
					fmt.Sprintf("%s learns: “%s”", ledger.EntityName, first.Value),
					"knowledge referenced across multiple chapters; first acquisition", frame, .85)
			}
			// A negated or suspected fact opens a mystery; a later firm
			// acquisition of the same fact resolves it.
			var mysteryOpened *types.StateLedgerEntry
			for index := range ledger.Entries {
				entry := &ledger.Entries[index]
				entryFrame := frameByID[entry.FrameID]
				if mysteryOpened == nil && (entry.Operation == "clear" || entryFrame.Epistemic == "belief") {
					mysteryOpened = entry
					dedupeKey := fmt.Sprintf("%s\u0000%d", ledger.EntityName, entry.ChapterIndex)
					if mysterySeen[dedupeKey] {
						continue
					}
					mysterySeen[dedupeKey] = true
					add("mystery_created",
						fmt.Sprintf("Open question for %s: “%s”", ledger.EntityName, entry.Value),
						"unknown or merely suspected fact", entryFrame, .75)
					continue
				}
				if mysteryOpened != nil && entry.Operation == "set" && entryFrame.Epistemic == "narration" {
					add("mystery_resolved",
						fmt.Sprintf("%s now knows: “%s”", ledger.EntityName, entry.Value),
						"previously unknown or suspected fact firmly acquired", entryFrame, .8)
					mysteryOpened = nil
				}
			}
		case "goal":
			count := goalsSeen[ledger.EntityName]
			goalsSeen[ledger.EntityName] = count + 1
			if count >= 1 {
				first := ledger.Entries[0]
				add("goal_change",
					fmt.Sprintf("%s pursues a new goal: “%s”", ledger.EntityName, first.Value),
					"a later goal joins or replaces an earlier one", frameByID[first.FrameID], .75)
			}
		case "relationship":
			values := map[string]bool{}
			for _, entry := range ledger.Entries {
				values[entry.Value] = true
			}
			if len(values) >= 2 {
				last := ledger.Entries[len(ledger.Entries)-1]
				add("relationship_change",
					fmt.Sprintf("%s and %s: the relationship shifts to “%s”", ledger.EntityName, ledger.Qualifier, last.Value),
					"relationship ledger holds more than one value", frameByID[last.FrameID], .75)
			}
		case "obligation":
			first := ledger.Entries[0]
			add("new_obstacle",
				fmt.Sprintf("%s is now bound: “%s”", ledger.EntityName, first.Value),
				"an obligation opens", frameByID[first.FrameID], .7)
		case "life_status":
			// A same-scope flip in life status is a reversal.
			byScope := map[string][]types.StateLedgerEntry{}
			for _, entry := range ledger.Entries {
				byScope[entry.Scope.ID] = append(byScope[entry.Scope.ID], entry)
			}
			for _, entries := range byScope {
				for index := 1; index < len(entries); index++ {
					if entries[index].Value != entries[index-1].Value {
						add("reversal",
							fmt.Sprintf("%s is %s after being %s", ledger.EntityName, entries[index].Value, entries[index-1].Value),
							"life status flips within one narrative scope", frameByID[entries[index].FrameID], .85)
					}
				}
			}
		case "possession", "access":
			// A setup that pays off chapters later.
			firstChapter := ledger.Entries[0].ChapterIndex
			for _, entry := range ledger.Entries[1:] {
				if entry.ChapterIndex >= firstChapter+2 {
					add("payoff",
						fmt.Sprintf("“%s” matters again for %s", entry.Value, ledger.EntityName),
						"an item or access established chapters earlier is used", frameByID[entry.FrameID], .65)
					break
				}
			}
		}
	}

	// Identity-driven rules: mixed epistemics on one underlying event either
	// corroborate a claim (accounts agree) or reveal something (they don't).
	for _, identity := range identities {
		var claim, narration *types.NarrativeFrame
		for _, frameID := range identity.FrameIDs {
			frame := frameByID[frameID]
			switch frame.Epistemic {
			case "attributed_claim", "belief":
				if claim == nil {
					f := frame
					claim = &f
				}
			case "narration":
				if narration == nil {
					f := frame
					narration = &f
				}
			}
		}
		if claim == nil || narration == nil {
			continue
		}
		later := *narration
		if claim.NarrativeOrder > later.NarrativeOrder {
			later = *claim
		}
		if identity.Status == "consistent" {
			add("corroboration",
				fmt.Sprintf("An account is corroborated: “%s”", later.Detail),
				"a claim and narration describe the same event consistently", later, .8)
		} else {
			add("reveal",
				fmt.Sprintf("Accounts of the same event differ: “%s”", later.Detail),
				"claim and narration give conflicting versions of one event", later, .8)
		}
	}

	// Decisions are inherently story-moving.
	for _, frame := range frames {
		if frame.Type == types.FrameDecision && frame.Epistemic == "narration" && frame.Polarity == "asserted" {
			add("major_decision",
				fmt.Sprintf("%s decides: “%s”", subjectName(frame), frame.Detail),
				"an explicit decision by a canonical character", frame, .8)
		}
	}

	sort.SliceStable(developments, func(i, j int) bool {
		return developments[i].NarrativeOrder < developments[j].NarrativeOrder
	})
	return developments
}
