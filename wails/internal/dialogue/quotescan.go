package dialogue

import (
	"strings"
	"unicode"
)

// Phase A of attribution: reading each sentence's quotation structure.
// Ported from scanQuotes in frontend/src/services/readaloud/attribution.ts;
// keep the two ports in step.

// A sentence at most this many unquoted non-space characters is pure
// dialogue; more unquoted text makes it "mixed" (quote plus tag/narration).
const mixedUnquotedChars = 15

var quoteClosers = map[rune]rune{'“': '”', '«': '»', '"': '"'}

// A straight double quote opens speech when it follows whitespace, an
// opening bracket, or a dash; anywhere else it closes.
func isStraightQuoteOpener(prev rune) bool {
	return unicode.IsSpace(prev) || strings.ContainsRune("([—–-", prev)
}

// quoteScan is Phase A's reading of one sentence.
type quoteScan struct {
	kind              string // narration | dialogue | mixed
	startsInsideQuote bool
	quoteOpensHere    bool
	// Id of the quote span this sentence's speech belongs to (open → close,
	// paragraph continuations included); -1 for pure narration.
	spanID int
	// Byte ranges of quoted speech, chapter-absolute, quote marks included.
	quotedRanges [][2]int
	// Unquoted stretches (for tag matching) and the unquoted prefix before
	// the first quote (for pre-quote tag matching).
	unquoted            string
	unquotedBeforeQuote string
}

func scanQuotes(sentences []sentence) []quoteScan {
	scans := make([]quoteScan, 0, len(sentences))
	inQuote := false
	var closer rune
	nextSpanID := 0
	activeSpanID := -1
	prevBlock := -1

	for sIdx, sent := range sentences {
		runes := []rune(sent.text)
		byteOf := make([]int, len(runes)+1)
		{
			offset := 0
			for i, r := range runes {
				byteOf[i] = offset
				offset += len(string(r))
			}
			byteOf[len(runes)] = offset
		}

		index := 0
		quoteStartIdx := 0 // range start when the sentence begins inside a quote

		if sIdx > 0 && sent.block != prevBlock && inQuote {
			// Convention: a speech spanning paragraphs leaves its quote
			// unclosed and re-opens with a quote char at the next paragraph
			// start. If the new paragraph does NOT re-open, the quote is over
			// (or was a typo) — close it so an unclosed straight quote can't
			// swallow the chapter.
			var first rune
			for _, r := range runes {
				if !strings.ContainsRune(" \t\n\r", r) {
					first = r
					break
				}
			}
			if c, ok := quoteClosers[first]; ok && c == closer {
				// Continuation: consume the re-opening char, stay inside.
				for i, r := range runes {
					if r == first {
						quoteStartIdx = i
						break
					}
				}
				index = quoteStartIdx + 1
			} else {
				inQuote = false
				closer = 0
			}
		}
		prevBlock = sent.block

		startsInsideQuote := inQuote
		spanID := -1
		if startsInsideQuote {
			spanID = activeSpanID
		}
		quoteOpensHere := false
		quoted := 0
		var quotedRanges [][2]int
		var unquotedParts []string
		var current strings.Builder
		beforeQuote := ""
		haveBeforeQuote := false

		for ; index < len(runes); index++ {
			ch := runes[index]
			if !inQuote {
				var prev rune
				if index > 0 {
					prev = runes[index-1]
				}
				isOpener := ch == '“' || ch == '«' ||
					(ch == '"' && (index == 0 || isStraightQuoteOpener(prev)))
				if isOpener {
					inQuote = true
					closer = quoteClosers[ch]
					quoteOpensHere = true
					quoteStartIdx = index
					activeSpanID = nextSpanID
					nextSpanID++
					if spanID == -1 {
						spanID = activeSpanID
					}
					if !haveBeforeQuote {
						beforeQuote = current.String()
						haveBeforeQuote = true
					}
					if strings.TrimSpace(current.String()) != "" {
						unquotedParts = append(unquotedParts, current.String())
					}
					current.Reset()
					continue
				}
				current.WriteRune(ch)
			} else {
				if ch == closer {
					inQuote = false
					closer = 0
					quotedRanges = append(quotedRanges,
						[2]int{sent.start + byteOf[quoteStartIdx], sent.start + byteOf[index+1]})
					continue
				}
				quoted++
			}
		}
		if strings.TrimSpace(current.String()) != "" {
			unquotedParts = append(unquotedParts, current.String())
		}
		if inQuote {
			quotedRanges = append(quotedRanges,
				[2]int{sent.start + byteOf[quoteStartIdx], sent.start + byteOf[len(runes)]})
		}

		unquoted := strings.Join(unquotedParts, " ")
		unquotedDense := 0
		for _, r := range unquoted {
			if !strings.ContainsRune(" \t\n\r", r) {
				unquotedDense++
			}
		}
		kind := "narration"
		if quoted != 0 || startsInsideQuote || quoteOpensHere {
			if unquotedDense <= mixedUnquotedChars {
				kind = "dialogue"
			} else {
				kind = "mixed"
			}
		}
		if kind == "narration" {
			spanID = -1
		}

		scans = append(scans, quoteScan{
			kind:                kind,
			startsInsideQuote:   startsInsideQuote,
			quoteOpensHere:      quoteOpensHere,
			spanID:              spanID,
			quotedRanges:        quotedRanges,
			unquoted:            unquoted,
			unquotedBeforeQuote: beforeQuote,
		})
	}
	return scans
}
