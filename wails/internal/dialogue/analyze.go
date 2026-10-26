// Package dialogue locates quoted speech in each chapter and attributes it
// to the book's confirmed characters, producing the persisted dialogue layer
// (types.DialogueData) that style inspections, character voice features, and
// Read Aloud casting consume.
package dialogue

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"draftline/internal/plaintext"
	"draftline/internal/types"
)

// Engine identifies this producer; bump the suffix when the algorithm
// changes so every cached chapter recomputes.
const Engine = "attribution-v1"

var confidenceByTier = map[string]float64{
	"tag":          0.95,
	"adjacent-tag": 0.90,
	"pretag":       0.75,
	"nearby":       0.60,
	"continuation": 0.55,
	"alternation":  0.40,
	"none":         0,
}

// Stronger tiers first, for picking a span's attribution from its sentences.
var tierRank = map[string]int{
	"tag": 6, "adjacent-tag": 5, "pretag": 4, "nearby": 3,
	"continuation": 2, "alternation": 1, "none": 0,
}

type rosterCharacter struct {
	entry       RosterEntry
	characterID string
	entityID    string
	name        string
}

// confirmedRoster builds the attributable cast: manual characters plus
// auto-detected ones the writer accepted, the same confirmation rule the
// continuity and search layers use.
func confirmedRoster(book *types.BookData) []rosterCharacter {
	entityByCharacter := map[string]string{}
	if book.Analysis.EntityResolution != nil {
		for _, entity := range book.Analysis.EntityResolution.Entities {
			if entity.CharacterID != "" {
				entityByCharacter[entity.CharacterID] = entity.ID
			}
		}
	}
	var roster []rosterCharacter
	for _, character := range book.StoryBible.Characters {
		if character.IsAutoDetected && character.DetectionStatus != "accepted" {
			continue
		}
		name := strings.TrimSpace(character.Name)
		if name == "" {
			continue
		}
		aliases := append([]string{}, character.SpellingAliases...)
		aliases = append(aliases, character.Aliases...)
		roster = append(roster, rosterCharacter{
			entry: RosterEntry{
				Key:     strings.ToLower(name),
				Name:    name,
				Aliases: aliases,
			},
			characterID: character.ID,
			entityID:    entityByCharacter[character.ID],
			name:        name,
		})
	}
	return roster
}

// rosterFingerprint folds everything attribution depends on besides the
// text, so a cast change recomputes cached chapters.
func rosterFingerprint(roster []rosterCharacter) string {
	hasher := sha256.New()
	for _, member := range roster {
		fmt.Fprintf(hasher, "%s\x00%s\x00%s\x00%s\x00%s\x1f",
			member.entry.Key, member.name, strings.Join(member.entry.Aliases, "\x01"),
			member.characterID, member.entityID)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func chapterCacheKey(text, fingerprint string) string {
	hasher := sha256.New()
	fmt.Fprintf(hasher, "%s\x00%s\x00%s", Engine, fingerprint, text)
	return hex.EncodeToString(hasher.Sum(nil))
}

// chapterSentences splits chapter plaintext into block-tagged sentences.
// Paragraphs are the blank-line blocks StripHTML emits; offsets are byte
// offsets into the chapter plaintext, the shared analysis coordinate space.
func chapterSentences(text string) []sentence {
	var sentences []sentence
	offset := 0
	block := 0
	for _, paragraph := range strings.Split(text, "\n\n") {
		for _, span := range segmentText(paragraph) {
			sentences = append(sentences, sentence{
				text:  paragraph[span.start:span.end],
				start: offset + span.start,
				block: block,
			})
		}
		offset += len(paragraph) + len("\n\n")
		block++
	}
	return sentences
}

// analyzeChapter runs attribution over one chapter and folds the sentence
// results into quote spans.
func analyzeChapter(chapterIndex int, text, chapterHash string, roster []rosterCharacter) []types.DialogueSpan {
	sentences := chapterSentences(text)
	if len(sentences) == 0 {
		return nil
	}
	entries := make([]RosterEntry, len(roster))
	byKey := make(map[string]rosterCharacter, len(roster))
	for i, member := range roster {
		entries[i] = member.entry
		byKey[member.entry.Key] = member
	}

	attributed, scans := attributeSpeakers(sentences, entries)

	type aggregate struct {
		start, end  int
		mixed       bool
		speaker     string
		attribution string
	}
	bySpan := map[int]*aggregate{}
	var order []int
	for i, scan := range scans {
		if scan.spanID < 0 || len(scan.quotedRanges) == 0 {
			continue
		}
		agg := bySpan[scan.spanID]
		if agg == nil {
			agg = &aggregate{start: scan.quotedRanges[0][0], end: scan.quotedRanges[0][1], attribution: "none"}
			bySpan[scan.spanID] = agg
			order = append(order, scan.spanID)
		}
		for _, r := range scan.quotedRanges {
			if r[0] < agg.start {
				agg.start = r[0]
			}
			if r[1] > agg.end {
				agg.end = r[1]
			}
		}
		if scan.kind == "mixed" {
			agg.mixed = true
		}
		result := attributed[i]
		if result.speaker != speakerUnknown && result.speaker != speakerNarrator &&
			(agg.speaker == "" || tierRank[result.confidence] > tierRank[agg.attribution]) {
			agg.speaker = result.speaker
			agg.attribution = result.confidence
		}
	}

	sort.Ints(order)
	spans := make([]types.DialogueSpan, 0, len(order))
	for _, id := range order {
		agg := bySpan[id]
		span := types.DialogueSpan{
			ChapterIndex: chapterIndex,
			StartOffset:  agg.start,
			EndOffset:    agg.end,
			Kind:         "dialogue",
			Attribution:  "none",
		}
		if agg.mixed {
			span.Kind = "mixed"
		}
		if agg.speaker != "" {
			member := byKey[agg.speaker]
			span.SpeakerName = member.name
			span.SpeakerEntityID = member.entityID
			span.Attribution = agg.attribution
			span.Confidence = confidenceByTier[agg.attribution]
		}
		idHasher := sha256.New()
		fmt.Fprintf(idHasher, "%s\x00%d\x00%d", chapterHash, agg.start, agg.end)
		span.ID = hex.EncodeToString(idHasher.Sum(nil))[:16]
		spans = append(spans, span)
	}
	return spans
}

// Analyze produces the book's dialogue layer. Chapters whose text and cast
// are unchanged reuse their previous spans instead of recomputing.
func Analyze(book *types.BookData, chapters []types.ChapterItem, shouldAnalyze func(types.ChapterItem) bool) *types.DialogueData {
	roster := confirmedRoster(book)
	fingerprint := rosterFingerprint(roster)

	var prior *types.DialogueData
	if book.Analysis.Dialogue != nil && book.Analysis.Dialogue.Engine == Engine {
		prior = book.Analysis.Dialogue
	}
	priorByChapter := map[int][]types.DialogueSpan{}
	if prior != nil {
		for _, span := range prior.Spans {
			priorByChapter[span.ChapterIndex] = append(priorByChapter[span.ChapterIndex], span)
		}
	}

	result := &types.DialogueData{
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
		key := chapterCacheKey(text, fingerprint)
		result.ChapterHashes[index] = key
		if prior != nil && prior.ChapterHashes[index] == key {
			result.Spans = append(result.Spans, priorByChapter[index]...)
			continue
		}
		result.Spans = append(result.Spans, analyzeChapter(index, text, key, roster)...)
	}
	return result
}
