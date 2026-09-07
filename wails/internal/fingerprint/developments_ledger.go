package fingerprint

// Identity- and ledger-driven development rules (v5): corroboration,
// disconfirmation, and cross-reality revelation from event identities;
// recurring discoveries, relationship shifts, and setup/payoff from the
// state ledgers.

import (
	"fmt"
	"sort"

	"draftline/internal/types"
)

// ── identity-driven rules ────────────────────────────────────────────────────

// identityRules turn multiple accounts of one underlying event into
// corroboration, disconfirmation, or cross-reality revelation. Same-scope
// narration-vs-narration conflicts stay with the continuity inspections —
// a probable authoring error is not a story development.
func (s *developmentSynthesizer) identityRules(identities []types.NarrativeEventIdentity, locator *sceneLocator) {
	for _, identity := range identities {
		members := []types.NarrativeFrame{}
		for _, frameID := range identity.FrameIDs {
			if frame, exists := s.frameByID[frameID]; exists {
				members = append(members, frame)
			}
		}
		if len(members) < 2 {
			continue
		}
		sort.SliceStable(members, func(i, j int) bool { return members[i].NarrativeOrder < members[j].NarrativeOrder })
		later := members[len(members)-1]
		unit := s.unitOf(later, locator)
		epistemics := map[string]bool{}
		scopes := map[string]bool{}
		for _, member := range members {
			epistemics[member.Epistemic] = true
			scopes[member.Scope.ID] = true
		}
		mixedEpistemics := len(epistemics) > 1
		// The development happens where the later account appears — emit
		// records the earliest supporting order, so pin it afterwards.
		place := func(development *types.NarrativeDevelopment) *types.NarrativeDevelopment {
			if development != nil {
				development.NarrativeOrder = later.NarrativeOrder
			}
			return development
		}
		switch identity.Status {
		case "consistent":
			if mixedEpistemics {
				place(s.emit("corroboration",
					fmt.Sprintf("An account is corroborated: “%s”", later.Detail),
					unit, members,
					[]string{"a claim and narration describe the same event consistently"}, .8))
			}
		case "conflicted":
			if len(scopes) > 1 {
				development := place(s.emit("revelation",
					fmt.Sprintf("Accounts of the same event diverge across realities: “%s”", later.Detail),
					unit, members,
					[]string{"one underlying event carries different details in different narrative scopes"}, .75))
				if development != nil {
					s.identityBeforeAfter(development, identity)
				}
			} else if mixedEpistemics {
				development := place(s.emit("disconfirmation",
					fmt.Sprintf("An account is contradicted: “%s”", later.Detail),
					unit, members,
					[]string{"a claim and narration give conflicting versions of one event"}, .8))
				if development != nil {
					s.identityBeforeAfter(development, identity)
				}
			}
		}
	}
}

func (s *developmentSynthesizer) identityBeforeAfter(development *types.NarrativeDevelopment, identity types.NarrativeEventIdentity) {
	for _, property := range identity.Properties {
		if len(property.Values) >= 2 {
			development.Before = fmt.Sprintf("account: “%s”", property.Values[0].Value)
			development.After = fmt.Sprintf("account: “%s”", property.Values[len(property.Values)-1].Value)
			return
		}
	}
}

func (s *developmentSynthesizer) unitOf(frame types.NarrativeFrame, locator *sceneLocator) unitRef {
	offset := 0
	if len(frame.EvidenceSpans) > 0 {
		offset = frame.EvidenceSpans[0].StartOffset
	}
	return unitRef{chapter: frame.ChapterIndex, scene: locator.sceneOf(frame.ChapterIndex, offset)}
}

// ── ledger-driven rules ──────────────────────────────────────────────────────

func (s *developmentSynthesizer) ledgerRules(ledgers []types.StateLedger, locator *sceneLocator) {
	for _, ledger := range ledgers {
		switch ledger.Aspect {
		case "knowledge":
			s.recurringDiscoveryRule(ledger, locator)
		case "relationship":
			s.relationshipRule(ledger, locator)
		case "possession", "access":
			s.setupPayoffRule(ledger, locator)
		}
	}
}

// recurringDiscoveryRule marks a firm acquisition whose fact the manuscript
// keeps returning to across chapters — a discovery that matters.
func (s *developmentSynthesizer) recurringDiscoveryRule(ledger types.StateLedger, locator *sceneLocator) {
	chapters := map[int]bool{}
	for _, entry := range ledger.Entries {
		chapters[entry.ChapterIndex] = true
	}
	if len(chapters) < 2 {
		return
	}
	first := ledger.Entries[0]
	frame, exists := s.frameByID[first.FrameID]
	if !exists || first.Operation != "set" || frame.Epistemic != "narration" {
		return
	}
	// Only real acquisitions count — a recalled attempt or a withheld fact
	// spanning chapters is not a discovery.
	if frame.Value != "learned" && frame.Value != "knows" {
		return
	}
	if len(contentWords(frame.Detail, ledger.EntityName)) < 2 {
		return // "Hanlon tried to remember." carries nothing discovered
	}
	s.emit("major_discovery",
		fmt.Sprintf("%s learns: “%s”", ledger.EntityName, first.Value),
		s.unitOf(frame, locator), []types.NarrativeFrame{frame},
		[]string{"this knowledge is referenced across multiple chapters; first acquisition"}, .85)
}

func (s *developmentSynthesizer) relationshipRule(ledger types.StateLedger, locator *sceneLocator) {
	for index := 1; index < len(ledger.Entries); index++ {
		previous, current := ledger.Entries[index-1], ledger.Entries[index]
		if previous.Value == current.Value {
			continue
		}
		frames := []types.NarrativeFrame{}
		if frame, exists := s.frameByID[previous.FrameID]; exists {
			frames = append(frames, frame)
		}
		currentFrame, exists := s.frameByID[current.FrameID]
		if !exists {
			continue
		}
		frames = append(frames, currentFrame)
		development := s.emit("relationship_change",
			fmt.Sprintf("%s and %s: the relationship shifts from “%s” to “%s”", ledger.EntityName, ledger.Qualifier, previous.Value, current.Value),
			s.unitOf(currentFrame, locator), frames,
			[]string{"the relationship ledger records a different value than before"}, .75)
		if development != nil {
			development.Before = previous.Value
			development.After = current.Value
		}
	}
}

// setupPayoffRule: an item or access established early matters again chapters
// later — the establishment is the setup, the reuse is the payoff.
func (s *developmentSynthesizer) setupPayoffRule(ledger types.StateLedger, locator *sceneLocator) {
	firstChapter := ledger.Entries[0].ChapterIndex
	firstFrame, exists := s.frameByID[ledger.Entries[0].FrameID]
	if !exists {
		return
	}
	for _, entry := range ledger.Entries[1:] {
		if entry.ChapterIndex < firstChapter+2 {
			continue
		}
		payoffFrame, found := s.frameByID[entry.FrameID]
		if !found {
			break
		}
		setup := s.emit("setup",
			fmt.Sprintf("Established: %s has “%s”", ledger.EntityName, ledger.Entries[0].Value),
			s.unitOf(firstFrame, locator), []types.NarrativeFrame{firstFrame},
			[]string{"this item or access is used again chapters later"}, .65)
		payoff := s.emit("payoff",
			fmt.Sprintf("“%s” matters again for %s", entry.Value, ledger.EntityName),
			s.unitOf(payoffFrame, locator), []types.NarrativeFrame{payoffFrame},
			[]string{"an item or access established chapters earlier is used"}, .65)
		if payoff != nil && setup != nil {
			s.linkPredecessor(payoff, setup.ID)
		}
		break
	}
}
