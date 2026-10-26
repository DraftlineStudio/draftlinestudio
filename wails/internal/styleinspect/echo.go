package styleinspect

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"draftline/internal/types"
)

// The echo rule: a notable word repeated within a short run of narration.
// Repetition inside dialogue is a character's voice and is never flagged;
// a confirmed character's name is never an echo.

// echoWindowWords is how far back, in words, a repeat still reads as an
// echo.
const echoWindowWords = 40

// A word that recurs throughout a chapter is its standing vocabulary — a
// viewpoint word like a camera in found-footage narration — not an echo.
// The threshold scales with chapter length so a short scene keeps its
// sensitivity.
func topicalThreshold(totalWords int) int {
	threshold := totalWords / 400
	if threshold < 5 {
		threshold = 5
	}
	return threshold
}

// Function words and auxiliaries that repeat in any prose without being
// echoes. Words shorter than four letters never flag, so this list only
// needs the longer ones.
var echoStopwords = map[string]bool{
	"that": true, "this": true, "with": true, "from": true, "they": true,
	"them": true, "their": true, "theirs": true, "there": true, "then": true,
	"than": true, "have": true, "been": true, "being": true, "were": true,
	"will": true, "would": true, "could": true, "should": true, "must": true,
	"what": true, "when": true, "where": true, "which": true, "while": true,
	"whose": true, "your": true, "yours": true, "ours": true, "mine": true,
	"about": true, "into": true, "onto": true, "over": true, "under": true,
	"after": true, "before": true, "again": true, "against": true,
	"because": true, "through": true, "between": true, "among": true,
	"during": true, "within": true, "without": true, "around": true,
	"toward": true, "towards": true, "upon": true, "though": true,
	"although": true, "every": true, "each": true, "some": true, "most": true,
	"more": true, "much": true, "many": true, "very": true, "just": true,
	"only": true, "even": true, "ever": true, "never": true, "still": true,
	"also": true, "once": true, "back": true, "down": true, "away": true,
	"here": true, "nothing": true, "something": true, "anything": true,
	"everything": true, "someone": true, "anyone": true, "everyone": true,
	"himself": true, "herself": true, "itself": true, "themselves": true,
	"myself": true, "yourself": true, "said": true, "says": true,
	"other": true, "another": true, "these": true, "those": true,
	"both": true, "such": true, "same": true,
	"like": true, "went": true, "came": true, "told": true, "asked": true,
}

type wordToken struct {
	key   string // lowercase, possessive stripped
	start int    // byte offset in chapter text
	end   int
	index int // word ordinal in the chapter
}

func tokenizeWords(text string) []wordToken {
	var tokens []wordToken
	start := -1
	ordinal := 0
	flush := func(end int) {
		if start < 0 {
			return
		}
		word := text[start:end]
		key := strings.ToLower(word)
		key = strings.TrimSuffix(strings.TrimSuffix(key, "'s"), "’s")
		tokens = append(tokens, wordToken{key: key, start: start, end: end, index: ordinal})
		ordinal++
		start = -1
	}
	for i, r := range text {
		if unicode.IsLetter(r) || r == '\'' || r == '’' {
			if start < 0 {
				start = i
			}
			continue
		}
		flush(i)
	}
	flush(len(text))
	return tokens
}

func insideSpans(spans []types.DialogueSpan, start, end int) bool {
	for _, span := range spans {
		if start >= span.StartOffset && end <= span.EndOffset {
			return true
		}
	}
	return false
}

func echoWordFindings(input chapterInput, chapterHash string, skipNames, ignoredWords map[string]bool) []types.StyleFinding {
	var findings []types.StyleFinding
	lastSeen := map[string]wordToken{}

	tokens := tokenizeWords(input.text)
	occurrences := map[string]int{}
	for _, token := range tokens {
		occurrences[token.key]++
	}
	threshold := topicalThreshold(len(tokens))

	for _, token := range tokens {
		if utf8.RuneCountInString(token.key) < 4 || echoStopwords[token.key] || skipNames[token.key] {
			continue
		}
		if ignoredWords[token.key] || occurrences[token.key] >= threshold {
			continue
		}
		if insideSpans(input.dialogueSpans, token.start, token.end) {
			continue
		}
		if previous, ok := lastSeen[token.key]; ok && token.index-previous.index <= echoWindowWords {
			prefix, suffix := contextAround(input.text, token.start, token.end)
			findings = append(findings, types.StyleFinding{
				ID:           findingID(chapterHash, "echo-word", token.start, token.end),
				Rule:         "echo-word",
				Severity:     "suggestion",
				ChapterIndex: input.index,
				StartOffset:  token.start,
				EndOffset:    token.end,
				Message: fmt.Sprintf("“%s” also appears %d words back.",
					input.text[token.start:token.end], token.index-previous.index),
				Snippet: input.text[token.start:token.end],
				Prefix:  prefix,
				Suffix:  suffix,
			})
		}
		lastSeen[token.key] = token
	}
	return findings
}
