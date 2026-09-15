package fingerprint

// Support classification for development synthesis (v5): which frames may
// complete or advance a current stake, and the discourse label every emitted
// development carries so a consumer never reads a recollection, a claim, or
// a flashback as a settled current fact.

import "draftline/internal/types"

// recalledValue reports whether a knowledge frame is a recollection rather
// than an acquisition: the character remembers, or tries to remember,
// something they already knew. A recollection never answers, narrows, or
// advances a current stake — it may be wrong, and it happened earlier.
func recalledValue(frame types.NarrativeFrame) bool {
	return frame.Value == "recalled" || frame.Value == "attempts_to_recall"
}

// supportsCurrentStake reports whether a frame may complete or advance a
// current goal or question: it must be asserted in the current reality
// scope (not a flashback, dream, hypothetical, or unclassified scope) and
// must not be a recollection. Scope kind "current" is the primary story
// time; every other kind, including the abstention default "uncertain",
// abstains.
func supportsCurrentStake(frame types.NarrativeFrame) bool {
	if frame.Scope.Kind != "current" || frame.Polarity == "negated" {
		return false
	}
	return !recalledValue(frame)
}

// discourseRank orders discourse labels by how far they sit from plain
// current narration; one tainting frame is enough to label a development.
var discourseRank = map[string]int{
	"narration": 0, "uncertain": 1, "belief": 2, "reported": 3,
	"recalled": 4, "hypothetical": 5, "negated": 6,
}

// DevelopmentDiscourse names the strongest reason a set of supporting frames
// is not plain current narration. Severity: negated > hypothetical >
// recalled > reported > belief > uncertain > narration. Exported so the
// evaluation harness labels developments with exactly the engine's rule.
func DevelopmentDiscourse(frames []types.NarrativeFrame) string {
	if len(frames) == 0 {
		return "uncertain"
	}
	result := "narration"
	consider := func(label string) {
		if discourseRank[label] > discourseRank[result] {
			result = label
		}
	}
	for _, frame := range frames {
		if frame.Polarity == "negated" {
			consider("negated")
		}
		if recalledValue(frame) {
			consider("recalled")
		}
		switch frame.Scope.Kind {
		case "current":
		case "flashback", "remembered":
			consider("recalled")
		case "hypothetical", "dream", "vision", "simulation", "alternate", "story_within_story":
			consider("hypothetical")
		default:
			consider("uncertain")
		}
		switch frame.Epistemic {
		case "narration":
		case "attributed_claim":
			consider("reported")
		case "belief", "speculation":
			consider("belief")
		default:
			consider("uncertain")
		}
	}
	return result
}
