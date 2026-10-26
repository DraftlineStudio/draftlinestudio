package dialogue

import (
	"strings"
	"unicode"
)

// Sentence segmentation for attribution, a port of the Read Aloud
// segmenter (frontend/src/services/readaloud/segmentation.ts). The rules
// favor prose fiction: terminators may hide inside closing quotes,
// abbreviations and initials do not end sentences, ellipses continue a
// sentence when the text resumes in lowercase, and em dashes never
// terminate. Keep the two ports in step.

// Compared lowercase with internal dots removed, so "i.e." matches "ie".
// Only consulted when the terminator is a single period.
var segmentAbbreviations = map[string]bool{
	"mr": true, "mrs": true, "ms": true, "dr": true, "st": true, "prof": true,
	"sr": true, "jr": true, "vs": true, "etc": true, "al": true,
	"ie": true, "eg": true, "cf": true, "mt": true, "ave": true, "blvd": true,
	"capt": true, "gen": true, "lt": true, "col": true, "sgt": true,
	"fig": true, "vol": true, "ch": true, "pp": true, "approx": true,
	"dept": true, "est": true,
}

const terminators = ".!?…"
const closers = "\"”’')]»"

type sentenceSpan struct {
	start int // byte offset, inclusive
	end   int // byte offset, exclusive
}

func isTerminator(r rune) bool { return strings.ContainsRune(terminators, r) }
func isCloser(r rune) bool     { return strings.ContainsRune(closers, r) }

// The word immediately before the rune index, allowing internal periods
// ("e.g").
func tokenBefore(runes []rune, position int) string {
	start := position
	for start > 0 && (unicode.IsLetter(runes[start-1]) || runes[start-1] == '.') {
		start--
	}
	return string(runes[start:position])
}

// segmentText splits text into sentence spans with byte offsets. Span
// boundaries include the terminator run and any closing quotes/brackets;
// leading whitespace is excluded. Text without a final terminator still
// yields a final span.
func segmentText(text string) []sentenceSpan {
	runes := []rune(text)
	// byteOf[i] is the byte offset of rune i; byteOf[len] is len(text).
	byteOf := make([]int, len(runes)+1)
	{
		offset := 0
		for i, r := range runes {
			byteOf[i] = offset
			offset += len(string(r))
		}
		byteOf[len(runes)] = offset
	}

	var spans []sentenceSpan
	sentenceStart := 0
	i := 0

	pushSpan := func(endExclusive int) {
		for sentenceStart < endExclusive && unicode.IsSpace(runes[sentenceStart]) {
			sentenceStart++
		}
		if sentenceStart >= endExclusive {
			sentenceStart = endExclusive
			return
		}
		body := strings.TrimSpace(string(runes[sentenceStart:endExclusive]))
		letters := 0
		for _, r := range body {
			if unicode.IsLetter(r) || unicode.IsNumber(r) {
				letters++
			}
		}
		// Merge fragments too small to stand ("A.", stray closers) into the
		// previous sentence.
		if letters < 2 && len(spans) > 0 {
			spans[len(spans)-1].end = byteOf[endExclusive]
		} else {
			spans = append(spans, sentenceSpan{start: byteOf[sentenceStart], end: byteOf[endExclusive]})
		}
		sentenceStart = endExclusive
	}

	for i < len(runes) {
		if !isTerminator(runes[i]) {
			i++
			continue
		}

		// Consume the whole terminator run (?!, !!!, ...).
		runStart := i
		for i < len(runes) && isTerminator(runes[i]) {
			i++
		}
		run := string(runes[runStart:i])

		// Then any closing quotes/brackets ("Go away." → boundary after the
		// quote).
		end := i
		for end < len(runes) && isCloser(runes[end]) {
			end++
		}

		if run == "." {
			before := tokenBefore(runes, runStart)
			compact := strings.ToLower(strings.ReplaceAll(before, ".", ""))
			beforeRunes := []rune(before)
			// Single-capital initials: initials in a name.
			if len(beforeRunes) == 1 && unicode.IsUpper(beforeRunes[0]) {
				i = end
				continue
			}
			if segmentAbbreviations[compact] {
				i = end
				continue
			}
			// "No." is an abbreviation only in front of a number (No. 5).
			if compact == "no" {
				rest := strings.TrimLeft(string(runes[end:]), " \t")
				if rest != "" && rest[0] >= '0' && rest[0] <= '9' {
					i = end
					continue
				}
			}
			// Decimals and versions: 3.50, 0.17.
			if runStart > 0 && runStart+1 < len(runes) &&
				unicode.IsDigit(runes[runStart-1]) && unicode.IsDigit(runes[runStart+1]) {
				i = end
				continue
			}
		}

		// What follows decides. EOF always ends the sentence.
		if end >= len(runes) {
			pushSpan(end)
			i = end
			continue
		}
		// A boundary needs whitespace after the closers; "end.Next" is a typo
		// left unsplit rather than split falsely.
		if !unicode.IsSpace(runes[end]) {
			i = end
			continue
		}
		next := end
		for next < len(runes) && unicode.IsSpace(runes[next]) {
			next++
		}
		// A trailing-off ellipsis continues when the text resumes in
		// lowercase; the same rule keeps "he said." unsplit inside tags.
		if next < len(runes) && unicode.IsLower(runes[next]) {
			i = end
			continue
		}
		pushSpan(end)
		i = end
	}

	pushSpan(len(runes))
	return spans
}
