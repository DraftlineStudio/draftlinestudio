package fingerprint

// Narrative-unit partitioning and content-word overlap for development
// synthesis (v5): scene/chapter units in the evidence coordinate space, and
// the deterministic word-set relations that tie facts to open questions
// and goals.

import (
	"sort"
	"strings"

	"draftline/internal/indexing"
	"draftline/internal/types"
)

// ── narrative units ──────────────────────────────────────────────────────────

type unitRef struct {
	chapter int
	scene   int
}

type narrativeUnit struct {
	ref    unitRef
	frames []types.NarrativeFrame
}

// sceneLocator maps a (global chapter index, stripped-text offset) pair to a
// 0-based scene ordinal, using the one scene-break rule of the indexing
// package (indexing.SceneBreakOffsets over StripHTMLForAnalysis text), so
// the development's scene index is indexing.SceneAt minus one.
type sceneLocator struct {
	breaks map[int][]int
}

func newSceneLocator(book *types.BookData) *sceneLocator {
	locator := &sceneLocator{breaks: map[int][]int{}}
	global := 0
	for _, chapters := range [][]types.ChapterItem{book.FrontMatter, book.Body, book.BackMatter} {
		for _, chapter := range chapters {
			text := indexing.StripHTMLForAnalysis(chapter.Content)
			if strings.TrimSpace(text) != "" {
				locator.breaks[global] = indexing.SceneBreakOffsets(text)
			}
			global++
		}
	}
	return locator
}

func (l *sceneLocator) sceneOf(chapter, offset int) int {
	scene := 0
	for _, breakOffset := range l.breaks[chapter] {
		if offset < breakOffset {
			break
		}
		scene++
	}
	return scene
}

// groupIntoUnits partitions frames (already in narrative order) into ordered
// scene/chapter units.
func groupIntoUnits(book *types.BookData, frames []types.NarrativeFrame) []narrativeUnit {
	locator := newSceneLocator(book)
	units := []narrativeUnit{}
	index := map[unitRef]int{}
	for _, frame := range frames {
		offset := 0
		if len(frame.EvidenceSpans) > 0 {
			offset = frame.EvidenceSpans[0].StartOffset
		}
		ref := unitRef{chapter: frame.ChapterIndex, scene: locator.sceneOf(frame.ChapterIndex, offset)}
		position, exists := index[ref]
		if !exists {
			position = len(units)
			index[ref] = position
			units = append(units, narrativeUnit{ref: ref})
		}
		units[position].frames = append(units[position].frames, frame)
	}
	sort.SliceStable(units, func(i, j int) bool {
		if units[i].ref.chapter != units[j].ref.chapter {
			return units[i].ref.chapter < units[j].ref.chapter
		}
		return units[i].ref.scene < units[j].ref.scene
	})
	return units
}

// ── key overlap ──────────────────────────────────────────────────────────────

// developmentStopWords extends the identity stop list with interrogatives,
// auxiliaries, copulas, discourse connectives, pronouns, and the epistemic
// verbs the extractors key on: two phrases sharing only "but ... knew"
// identify nothing, and "who" in a question can never appear in its answer.
var developmentStopWords = map[string]bool{
	"who": true, "what": true, "where": true, "when": true, "why": true,
	"how": true, "whether": true, "someone": true, "something": true,
	"had": true, "has": true, "have": true, "was": true, "were": true,
	"is": true, "are": true, "be": true, "been": true, "being": true,
	"did": true, "does": true, "do": true, "would": true, "could": true,
	"that": true, "this": true, "it": true, "not": true, "never": true,
	"before": true, "after": true, "until": true, "there": true, "still": true,
	"but": true, "so": true, "also": true, "yet": true, "only": true,
	"even": true, "just": true, "well": true, "better": true,
	"know": true, "knew": true, "known": true, "learned": true,
	"realized": true, "remember": true, "remembered": true, "tried": true,
	"understood": true, "wondered": true, "noticed": true, "believe": true,
	"believed": true, "he": true, "she": true, "they": true, "him": true,
	"her": true, "himself": true, "herself": true, "them": true,
	// Contraction remnants after QualifierKey strips apostrophes.
	"didnt": true, "dont": true, "doesnt": true, "wasnt": true,
	"werent": true, "couldnt": true, "wouldnt": true, "shouldnt": true,
	"isnt": true, "arent": true, "wont": true, "hadnt": true,
	"hasnt": true, "havent": true, "hed": true, "shed": true,
	"theyd": true, "hes": true, "shes": true, "youre": true,
}

// ContentWords returns the normalized content-word set of a verbatim
// phrase's six-word head (its QualifierKey): the proposition of the clause,
// without the trailing adjuncts ("before night", "somewhere in the tower")
// that an answer or a restatement legitimately drops. It is the
// deterministic basis for relating a fact to an open question or goal.
// Exported for the plotwalker evaluation harness, which must relate
// developments to expectations with exactly the engine's own word rule.
// The subject's own name tokens never count as content — a character's name
// appearing in two phrases relates nothing about the story.
func ContentWords(text, subject string) map[string]bool {
	return contentWordsOf(QualifierKey(text), subject)
}

// phraseWords is ContentWords over the whole phrase, not its head. It is
// used only on the fact side of an acquisition or stake tie, where the
// stake's object may be named late ("returned to the village with the key
// and the ledger"); the stake side always keeps its head, so a tie can
// never grow past the stake's own proposition.
func phraseWords(text, subject string) map[string]bool {
	return contentWordsOf(normalizePhrase(text), subject)
}

func contentWordsOf(normalized, subject string) map[string]bool {
	nameTokens := map[string]bool{}
	for _, token := range strings.Fields(strings.ToLower(subject)) {
		nameTokens[nonKeyRe.ReplaceAllString(token, "")] = true
	}
	words := map[string]bool{}
	for _, word := range strings.Fields(normalized) {
		if !anchorStopWords[word] && !developmentStopWords[word] && !nameTokens[word] {
			words[word] = true
		}
	}
	return words
}

// OverlapWords returns the sorted words two content-word sets share.
func OverlapWords(a, b map[string]bool) []string {
	shared := []string{}
	for word := range a {
		if b[word] {
			shared = append(shared, word)
		}
	}
	sort.Strings(shared)
	return shared
}

// covers reports whether the fact's content words include every content word
// of the question's head — the question is fully addressed, not merely
// touched. A question head with fewer than two content words is too thin
// to ever declare covered; it can only be matched by an identical key.
func covers(fact, question map[string]bool) bool {
	if len(question) < 2 {
		return false
	}
	for word := range question {
		if !fact[word] {
			return false
		}
	}
	return true
}
