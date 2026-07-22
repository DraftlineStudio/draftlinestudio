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

// ParagraphInfo holds information about a detected paragraph.
type ParagraphInfo struct {
	Text        string
	StartOffset int
	EndOffset   int
}

// DetectScenes analyzes stripped chapter text and returns scene/paragraph
// boundaries. Scene breaks are marker lines (*** / ### / --- / . . .);
// paragraphs are blank-line separated blocks.
func DetectScenes(text string, chapterIndex int) []types.SceneRecord {
	scenes := []types.SceneRecord{}

	sceneBreakOffsets := detectSceneBreaks(text)
	paragraphs := splitIntoParagraphs(text)

	if len(paragraphs) == 0 {
		return scenes
	}

	breakIdx := 0
	for i, para := range paragraphs {
		sceneType := "paragraph"

		// A paragraph past the next scene break starts a new scene.
		if breakIdx < len(sceneBreakOffsets) && para.StartOffset >= sceneBreakOffsets[breakIdx] {
			breakIdx++
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

	parts := regexp.MustCompile(`\n\s*\n`).Split(text, -1)
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

// detectSceneBreaks finds scene break markers in stripped text.
func detectSceneBreaks(text string) []int {
	offsets := []int{}

	patterns := []string{
		`\n\s*\*\s*\*\s*\*\s*\n`, // * * *
		`\n\s*#\s*#\s*#\s*\n`,    // # # #
		`\n\s*-\s*-\s*-\s*\n`,    // - - -
		`\n\s*\*{3,}\s*\n`,       // ***
		`\n\s*#{3,}\s*\n`,        // ###
		`\n\s*-{3,}\s*\n`,        // ---
		`\n\s*~\s*~\s*~\s*\n`,    // ~ ~ ~
		`\n\s*\.\s*\.\s*\.\s*\n`, // . . .
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		for _, match := range re.FindAllStringIndex(text, -1) {
			offsets = append(offsets, match[0])
		}
	}

	sort.Ints(offsets)
	return offsets
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
		for _, mentionID := range entity.MentionIDs {
			result[mentionID] = entity.ID
		}
	}
	return result
}
