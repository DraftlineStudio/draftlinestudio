package indexing

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"draftline/internal/types"
)

// Scene detection operates on stripped text (see StripHTML), the same
// coordinate space as mention offsets.

// Regexes are compiled once at package load. Recompiling them per chapter
// (which DetectScenes is called for) was a measurable indexing cost on large
// books.
var (
	paragraphSplitRe = regexp.MustCompile(`\n\s*\n`)

	// sceneBreakMarker is the one list of marker forms every scene-break
	// pattern is built from: * * *, ***, ⁂, # # #, ###, - - -, ---, ~ ~ ~,
	// . . . (an <hr> is already "* * *" after StripHTML).
	sceneBreakMarker = `(?:\*\s*\*\s*\*|\*{3,}|⁂|#\s*#\s*#|#{3,}|-\s*-\s*-|-{3,}|~\s*~\s*~|\.\s*\.\s*\.)`

	// sceneBreakLineRe matches a marker on a line of its own.
	sceneBreakLineRe = regexp.MustCompile(`\n\s*` + sceneBreakMarker + `\s*\n`)

	// sceneBreakParagraphRe matches a whole paragraph that is nothing but a
	// marker: the one scene-break rule shared by scene detection, the
	// manuscript memory's scene index, the isolated narrative engine's block
	// scenes, and the Planner's scene numbers.
	sceneBreakParagraphRe = regexp.MustCompile(`^\s*` + sceneBreakMarker + `\s*$`)

	// leadingSceneBreakRe matches a marker line at the start of a sentence.
	// The sentence segmenter does not split at a marker (it has no terminal
	// punctuation), so the first sentence after a break arrives with the
	// marker glued to its front unless it is cut off.
	leadingSceneBreakRe = regexp.MustCompile(`^\s*` + sceneBreakMarker + `\s*\n\s*`)
)

// textPiece is a contiguous slice of chapter text with its offsets.
type textPiece struct {
	text  string
	start int
	end   int
}

// splitAtSceneBreaks cuts a segmenter sentence at every scene-break marker
// it contains — a marker glued to its front, or one in its middle when the
// paragraph before the marker ended without terminal punctuation — and
// returns the non-empty prose pieces, trimmed, with chapter offsets that
// locate them exactly.
func splitAtSceneBreaks(text string, start int) []textPiece {
	pieces := []textPiece{}
	add := func(segment string, at int) {
		trimmed := strings.TrimSpace(segment)
		if trimmed == "" {
			return
		}
		at += strings.Index(segment, trimmed)
		pieces = append(pieces, textPiece{text: trimmed, start: at, end: at + len(trimmed)})
	}
	offset := 0
	if match := leadingSceneBreakRe.FindStringIndex(text); match != nil {
		offset = match[1]
	}
	base := offset
	for _, match := range sceneBreakLineRe.FindAllStringIndex(text[base:], -1) {
		add(text[offset:base+match[0]], start+offset)
		offset = base + match[1]
	}
	add(text[offset:], start+offset)
	return pieces
}

// ParagraphInfo holds information about a detected paragraph.
type ParagraphInfo struct {
	Text        string
	StartOffset int
	EndOffset   int
}

// IsSceneBreakParagraph reports whether a paragraph of stripped text is a
// scene-break marker and nothing else: * * *, ***, ⁂, # # #, ###, - - -,
// ---, ~ ~ ~ or . . . (an <hr> is "* * *" once stripped). Every scene
// number in the application derives from this one predicate.
func IsSceneBreakParagraph(text string) bool {
	return sceneBreakParagraphRe.MatchString(text)
}

// SceneBreakOffsets returns the start offset of every scene-break paragraph
// in stripped chapter text, in order.
func SceneBreakOffsets(text string) []int {
	offsets := []int{}
	for _, para := range splitIntoParagraphs(text) {
		if IsSceneBreakParagraph(para.Text) {
			offsets = append(offsets, para.StartOffset)
		}
	}
	return offsets
}

// SceneAt returns the 1-based scene that contains the given offset of
// stripped chapter text: one more than the number of scene-break
// paragraphs that start at or before it. The marker paragraph itself
// belongs to the scene it opens.
func SceneAt(text string, offset int) int {
	scene := 1
	for _, start := range SceneBreakOffsets(text) {
		if offset < start {
			break
		}
		scene++
	}
	return scene
}

// SceneCount returns how many scenes stripped chapter text has: zero for a
// chapter with no prose, otherwise one more than its scene breaks.
func SceneCount(text string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	return len(SceneBreakOffsets(text)) + 1
}

// DetectScenes analyzes stripped chapter text and returns scene/paragraph
// boundaries. A paragraph that is only a scene-break marker
// (IsSceneBreakParagraph) is recorded as the break; every other blank-line
// separated block is a paragraph.
func DetectScenes(text string, chapterIndex int) []types.SceneRecord {
	scenes := []types.SceneRecord{}

	paragraphs := splitIntoParagraphs(text)

	if len(paragraphs) == 0 {
		return scenes
	}

	for i, para := range paragraphs {
		sceneType := "paragraph"
		if IsSceneBreakParagraph(para.Text) {
			sceneType = "scene_break"
		}

		scenes = append(scenes, types.SceneRecord{
			ID:           fmt.Sprintf("scene-%d-%d", chapterIndex, i),
			ChapterIndex: chapterIndex,
			StartOffset:  para.StartOffset,
			EndOffset:    para.EndOffset,
			SceneType:    sceneType,
			CharacterIDs: []string{},
		})
	}

	return scenes
}

// splitIntoParagraphs extracts blank-line separated blocks with their offsets.
func splitIntoParagraphs(text string) []ParagraphInfo {
	paragraphs := []ParagraphInfo{}

	parts := paragraphSplitRe.Split(text, -1)
	offset := 0

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		idx := strings.Index(text[offset:], part)
		if idx >= 0 {
			startOffset := offset + idx
			endOffset := startOffset + len(part)

			paragraphs = append(paragraphs, ParagraphInfo{
				Text:        part,
				StartOffset: startOffset,
				EndOffset:   endOffset,
			})

			offset = endOffset
		}
	}

	return paragraphs
}

// GetCharactersInScene finds which characters appear in a scene based on mentions.
func GetCharactersInScene(scene types.SceneRecord, mentions []types.MentionRecord, entityMap map[string]string) []string {
	characterIDs := make(map[string]bool)

	for _, mention := range mentions {
		if mention.Chapter == scene.ChapterIndex &&
			mention.CharOffset >= scene.StartOffset &&
			mention.CharOffset < scene.EndOffset {

			if charID, ok := entityMap[mention.ID]; ok {
				characterIDs[charID] = true
			}
		}
	}

	result := make([]string, 0, len(characterIDs))
	for id := range characterIDs {
		result = append(result, id)
	}
	sort.Strings(result)

	return result
}

// BuildMentionToEntityMap creates a lookup from mention ID to entity/character ID.
func BuildMentionToEntityMap(entities []types.EntityRecord) map[string]string {
	result := make(map[string]string)
	for _, entity := range entities {
		// Legacy entities without a status remain analyzable for archive
		// compatibility, but current Needs Review and rejected candidates must
		// never become relationship endpoints.
		if entity.DetectionStatus == "review" || entity.DetectionStatus == "rejected" {
			continue
		}
		for _, mentionID := range entity.MentionIDs {
			result[mentionID] = entity.ID
		}
	}
	return result
}
