package styleinspect

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"draftline/internal/types"
)

// The regional-vocabulary rule: expressions one region's readers use without
// thinking and another's stumble over (a bubbler, a davenport). Entries come
// from the bundled dictionary — hand-written seeds plus reviewed Wiktionary
// extractions (see tools/regional-dictionary). The rule only ever suggests:
// dialogue is left alone (a character's vocabulary is a voice), and entries
// with everyday non-regional senses (buggy, jimmies) stay out of matching
// until the engine can weigh context, so a false hint stays rare.

//go:embed data/regional.json
var regionalData []byte

type regionalEntry struct {
	ID           string   `json:"id"`
	Forms        []string `json:"forms"`
	Regions      []string `json:"regions"`
	Meaning      string   `json:"meaning"`
	Alternatives []string `json:"alternatives"`
	Ambiguity    string   `json:"ambiguity"`
}

type regionalDictionary struct {
	Entries []regionalEntry `json:"entries"`
}

// regionalIndex maps a form's first word to the entries whose forms start
// with it; multi-word forms match token runs.
type regionalIndex struct {
	byFirstToken map[string][]regionalMatch
}

type regionalMatch struct {
	entry  *regionalEntry
	tokens []string
}

func buildRegionalIndex() *regionalIndex {
	var dictionary regionalDictionary
	if err := json.Unmarshal(regionalData, &dictionary); err != nil {
		// A broken bundled file is a build defect; fail soft at runtime.
		return &regionalIndex{byFirstToken: map[string][]regionalMatch{}}
	}
	index := &regionalIndex{byFirstToken: map[string][]regionalMatch{}}
	for i := range dictionary.Entries {
		entry := &dictionary.Entries[i]
		if entry.Ambiguity == "high" {
			continue // context weighing is not built yet; stay quiet
		}
		for _, form := range entry.Forms {
			tokens := strings.Fields(strings.ToLower(form))
			if len(tokens) == 0 {
				continue
			}
			index.byFirstToken[tokens[0]] = append(index.byFirstToken[tokens[0]],
				regionalMatch{entry: entry, tokens: tokens})
		}
	}
	return index
}

var regionalWords = buildRegionalIndex()

func regionalFindings(input chapterInput, chapterHash string, ignoredWords map[string]bool) []types.StyleFinding {
	if len(regionalWords.byFirstToken) == 0 {
		return nil
	}
	var findings []types.StyleFinding
	tokens := tokenizeWords(input.text)

	for i, token := range tokens {
		matches, ok := regionalWords.byFirstToken[token.key]
		if !ok {
			continue
		}
		for _, match := range matches {
			if i+len(match.tokens) > len(tokens) {
				continue
			}
			matched := true
			for offset, formToken := range match.tokens {
				if tokens[i+offset].key != formToken {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}
			start := token.start
			end := tokens[i+len(match.tokens)-1].end
			if insideSpans(input.dialogueSpans, start, end) {
				continue // a character's vocabulary is a voice, not a slip
			}
			if ignoredWords[strings.ToLower(input.text[start:end])] {
				continue
			}
			prefix, suffix := contextAround(input.text, start, end)
			findings = append(findings, types.StyleFinding{
				ID:           findingID(chapterHash, "regional-word", start, end),
				Rule:         "regional-word",
				Severity:     "hint",
				ChapterIndex: input.index,
				StartOffset:  start,
				EndOffset:    end,
				Message: fmt.Sprintf("In %s this usually means %s; readers elsewhere may not know it.",
					strings.Join(match.entry.Regions, " and "), match.entry.Meaning),
				Snippet:      input.text[start:end],
				Prefix:       prefix,
				Suffix:       suffix,
				Replacements: match.entry.Alternatives,
			})
			break // one finding per position, longest-listed form first is fine
		}
	}
	return findings
}
