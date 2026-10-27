// Package styleinspect produces the style layer of a book's analysis:
// deterministic, source-located style findings that the editor underlines
// and the author can dismiss. Rules here need whole-chapter or
// dialogue-aware context; single-paragraph mechanics stay in the editor's
// instant checks.
package styleinspect

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"draftline/internal/plaintext"
	"draftline/internal/types"
)

// Engine identifies this producer; bump the suffix when rules change so
// every cached chapter recomputes.
const Engine = "styleinspect-v3"

// maxFindings caps the persisted findings list, the evidence-cap precedent.
const maxFindings = 20000

type chapterInput struct {
	index         int
	text          string
	dialogueSpans []types.DialogueSpan
}

// Analyze produces the book's style findings. Chapters whose text, cast,
// and dialogue layer are unchanged reuse their previous findings, and the
// author's dismissals survive for findings that still exist.
func Analyze(book *types.BookData, chapters []types.ChapterItem, shouldAnalyze func(types.ChapterItem) bool) *types.StyleData {
	skipNames := rosterNames(book)
	ignoredWords := map[string]bool{}
	if book.Analysis.Style != nil {
		for _, word := range book.Analysis.Style.IgnoredWords {
			ignoredWords[strings.ToLower(strings.TrimSpace(word))] = true
		}
	}
	fingerprint := skipFingerprint(skipNames) + "\x00" + skipFingerprint(ignoredWords)

	dialogueByChapter := map[int][]types.DialogueSpan{}
	dialogueHashes := map[int]string{}
	if book.Analysis.Dialogue != nil {
		for _, span := range book.Analysis.Dialogue.Spans {
			dialogueByChapter[span.ChapterIndex] = append(dialogueByChapter[span.ChapterIndex], span)
		}
		dialogueHashes = book.Analysis.Dialogue.ChapterHashes
	}

	var prior *types.StyleData
	if book.Analysis.Style != nil && book.Analysis.Style.Engine == Engine {
		prior = book.Analysis.Style
	}
	priorByChapter := map[int][]types.StyleFinding{}
	if prior != nil {
		for _, finding := range prior.Findings {
			priorByChapter[finding.ChapterIndex] = append(priorByChapter[finding.ChapterIndex], finding)
		}
	}

	result := &types.StyleData{
		ChapterHashes: map[int]string{},
		Engine:        Engine,
		Version:       1,
	}
	for index, chapter := range chapters {
		if shouldAnalyze != nil && !shouldAnalyze(chapter) {
			continue
		}
		text := plaintext.StripHTMLForAnalysis(chapter.Content)
		if strings.TrimSpace(text) == "" {
			continue
		}
		key := chapterCacheKey(text, fingerprint, dialogueHashes[index])
		result.ChapterHashes[index] = key
		if prior != nil && prior.ChapterHashes[index] == key {
			result.Findings = append(result.Findings, priorByChapter[index]...)
			continue
		}
		input := chapterInput{index: index, text: text, dialogueSpans: dialogueByChapter[index]}
		result.Findings = append(result.Findings, echoWordFindings(input, key, skipNames, ignoredWords)...)
		result.Findings = append(result.Findings, regionalFindings(input, key, ignoredWords)...)
	}

	if len(result.Findings) > maxFindings {
		result.Findings = result.Findings[:maxFindings]
		result.Truncated = true
	}

	// The author's word-level silences are kept verbatim: they are vocabulary
	// choices, not positions in prose.
	if book.Analysis.Style != nil {
		result.IgnoredWords = append(result.IgnoredWords, book.Analysis.Style.IgnoredWords...)
	}

	// Keep only decisions whose findings still exist: IDs hash the prose, so
	// anything else is a decision about text that is gone.
	if book.Analysis.Style != nil && len(book.Analysis.Style.Decisions) > 0 {
		alive := make(map[string]bool, len(result.Findings))
		for _, finding := range result.Findings {
			alive[finding.ID] = true
		}
		for _, decision := range book.Analysis.Style.Decisions {
			if alive[decision.FindingID] {
				result.Decisions = append(result.Decisions, decision)
			}
		}
	}
	return result
}

// rosterNames collects every confirmed character name and alias token, so a
// repeated name is never called an echo.
func rosterNames(book *types.BookData) map[string]bool {
	names := map[string]bool{}
	for _, character := range book.StoryBible.Characters {
		if character.IsAutoDetected && character.DetectionStatus != "accepted" {
			continue
		}
		for _, name := range append(append([]string{character.Name}, character.SpellingAliases...), character.Aliases...) {
			for _, token := range strings.Fields(name) {
				names[strings.ToLower(strings.Trim(token, ".,'’"))] = true
			}
		}
	}
	return names
}

func skipFingerprint(names map[string]bool) string {
	keys := make([]string, 0, len(names))
	for name := range names {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	hasher := sha256.New()
	for _, key := range keys {
		fmt.Fprintf(hasher, "%s\x00", key)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func chapterCacheKey(text, fingerprint, dialogueHash string) string {
	hasher := sha256.New()
	fmt.Fprintf(hasher, "%s\x00%s\x00%s\x00%s", Engine, fingerprint, dialogueHash, text)
	return hex.EncodeToString(hasher.Sum(nil))
}

func findingID(chapterHash, rule string, start, end int) string {
	hasher := sha256.New()
	fmt.Fprintf(hasher, "%s\x00%s\x00%d\x00%d", chapterHash, rule, start, end)
	return hex.EncodeToString(hasher.Sum(nil))[:16]
}

// contextAround fills a finding's prefix and suffix: enough surrounding text
// for the editor to place the underline uniquely, cut at rune boundaries and
// never crossing a paragraph break — the editor matches within one block.
func contextAround(text string, start, end int) (prefix, suffix string) {
	const contextBytes = 24
	from := start - contextBytes
	if from < 0 {
		from = 0
	}
	for from > 0 && !isRuneStart(text[from]) {
		from--
	}
	to := end + contextBytes
	if to > len(text) {
		to = len(text)
	}
	for to < len(text) && !isRuneStart(text[to]) {
		to++
	}
	prefix, suffix = text[from:start], text[end:to]
	if cut := strings.LastIndexByte(prefix, '\n'); cut >= 0 {
		prefix = prefix[cut+1:]
	}
	if cut := strings.IndexByte(suffix, '\n'); cut >= 0 {
		suffix = suffix[:cut]
	}
	return prefix, suffix
}

func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }
