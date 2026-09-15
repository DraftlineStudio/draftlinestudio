package fingerprint

// Quoted-speech and reported-clause handling for frame extraction (v5): a
// sentence whose subject sits inside quotation marks is a character's
// claim, and a term inside a "learned that …" clause belongs to the clause,
// not to the sentence's subject.

import (
	"strings"

	"draftline/internal/types"
)

// startsInsideQuote reports whether the sentence opens with quoted speech,
// so a subject anchored at its start is spoken about, not narrated.
func startsInsideQuote(text string) bool {
	trimmed := strings.TrimLeft(text, " —-")
	return strings.HasPrefix(trimmed, "“") || strings.HasPrefix(trimmed, "\"")
}

// cueInsideQuote reports whether the first occurrence of cue in text lies
// within a quoted span.
func cueInsideQuote(text, cue string) bool {
	cue = strings.TrimSpace(cue)
	if cue == "" {
		return false
	}
	at := strings.Index(text, cue)
	if at < 0 {
		return false
	}
	for _, span := range quotedRe.FindAllStringIndex(text, -1) {
		if at > span[0] && at < span[1] {
			return true
		}
	}
	return false
}

// attributeToSpeech re-labels a frame as a character's claim. The speaker is
// the tagged speaker of the sentence when the roster knows one; otherwise
// the attribution is unknown and recorded as an abstention rather than
// guessed.
func attributeToSpeech(fc *frameContext, text string, frame *types.NarrativeFrame) {
	frame.Epistemic = "attributed_claim"
	frame.Attribution = types.NarrativeAttribution{Kind: "unknown", Confidence: .5}
	if tag := claimTagRe.FindStringSubmatchIndex(text); tag != nil {
		if speaker, ok := fc.roster.findIn(text[tag[2]:tag[3]]); ok {
			frame.Attribution = types.NarrativeAttribution{Kind: "character", EntityName: speaker, Cue: strings.TrimSpace(text[tag[2]:tag[5]]), Confidence: .85}
			return
		}
	}
	frame.Abstentions = append(frame.Abstentions, "attribution: the sentence is quoted speech with no roster speaker")
}

// insideReportedClause reports whether position at in rest follows a
// knowledge or belief cue, placing it inside the clause that cue governs.
func insideReportedClause(rest string, at int) bool {
	for _, re := range []interface{ FindStringIndex(string) []int }{knowledgeRe, beliefRe} {
		if match := re.FindStringIndex(rest); match != nil && match[0] < at {
			return true
		}
	}
	return false
}
