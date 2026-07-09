package indexing

import (
	"fmt"
	"strings"
	"time"

	"draftline/internal/types"
)

// Book scans all chapters and extracts/updates character information.
func Book(book types.BookData) types.IndexResult {
	if book.StoryBible.Characters == nil {
		book.StoryBible.Characters = []types.Character{}
	}

	// Build a map of existing characters by name (case-insensitive)
	existingChars := make(map[string]*types.Character)
	for i := range book.StoryBible.Characters {
		char := &book.StoryBible.Characters[i]
		existingChars[strings.ToLower(char.Name)] = char
		// Also index aliases
		for _, alias := range char.Aliases {
			existingChars[strings.ToLower(alias)] = char
		}
	}

	// Aggregate all detected names across all chapters
	allNames := make(map[string]int)                // name -> total mentions
	chapterMentions := make(map[string]map[int]int) // name -> (chapter -> count)
	firstChapter := make(map[string]int)            // name -> first chapter seen
	fullText := ""                                  // For attribute extraction

	// Process all sections
	sections := []struct {
		name  string
		items []types.ChapterItem
	}{
		{"front_matter", book.FrontMatter},
		{"body", book.Body},
		{"back_matter", book.BackMatter},
	}

	chapterIndex := 0
	for _, section := range sections {
		for _, chapter := range section.items {
			text := StripHTML(chapter.Content)
			fullText += " " + text
			names := DetectCharacterNames(text)

			for name, count := range names {
				allNames[name] += count

				// Track chapter mentions
				if chapterMentions[name] == nil {
					chapterMentions[name] = make(map[int]int)
				}
				chapterMentions[name][chapterIndex] += count

				// Track first chapter
				if _, exists := firstChapter[name]; !exists {
					firstChapter[name] = chapterIndex
				}
			}
			chapterIndex++
		}
	}

	// Filter: only keep names with 3+ mentions (likely real characters)
	newChars := 0
	updatedChars := 0

	for name, count := range allNames {
		if count < 3 {
			continue // Skip names with too few mentions
		}

		nameLower := strings.ToLower(name)

		if existing, found := existingChars[nameLower]; found {
			// Update existing character
			existing.MentionCount = count
			existing.ChapterMentions = chapterMentions[name]
			if existing.FirstChapter == 0 {
				existing.FirstChapter = firstChapter[name]
			}
			// Try to extract more attributes
			newAttrs := ExtractAttributes(fullText, name)
			if existing.Attributes == nil {
				existing.Attributes = make(map[string]string)
			}
			for k, v := range newAttrs {
				if existing.Attributes[k] == "" {
					existing.Attributes[k] = v
				}
			}
			updatedChars++
		} else {
			// Create new character
			newChar := types.Character{
				ID:              fmt.Sprintf("char-%d", time.Now().UnixNano()),
				Name:            name,
				Role:            "minor", // Default role
				IsAutoDetected:  true,
				MentionCount:    count,
				FirstChapter:    firstChapter[name],
				ChapterMentions: chapterMentions[name],
				Attributes:      ExtractAttributes(fullText, name),
			}
			book.StoryBible.Characters = append(book.StoryBible.Characters, newChar)
			existingChars[nameLower] = &newChar
			newChars++
		}
	}

	return types.IndexResult{
		Success:           true,
		CharactersFound:   len(allNames),
		NewCharacters:     newChars,
		UpdatedCharacters: updatedChars,
		Characters:        book.StoryBible.Characters,
	}
}

// Chapter scans a single chapter for character mentions (incremental update).
func Chapter(book types.BookData, section string, chapterIndex int) types.IndexResult {
	var content string
	switch section {
	case "front_matter":
		if chapterIndex < len(book.FrontMatter) {
			content = book.FrontMatter[chapterIndex].Content
		}
	case "body":
		if chapterIndex < len(book.Body) {
			content = book.Body[chapterIndex].Content
		}
	case "back_matter":
		if chapterIndex < len(book.BackMatter) {
			content = book.BackMatter[chapterIndex].Content
		}
	}

	if content == "" {
		return types.IndexResult{Success: true, CharactersFound: 0}
	}

	text := StripHTML(content)
	names := DetectCharacterNames(text)

	// Build existing character map
	existingChars := make(map[string]*types.Character)
	for i := range book.StoryBible.Characters {
		char := &book.StoryBible.Characters[i]
		existingChars[strings.ToLower(char.Name)] = char
		for _, alias := range char.Aliases {
			existingChars[strings.ToLower(alias)] = char
		}
	}

	newChars := 0
	for name, count := range names {
		if count < 2 {
			continue
		}

		nameLower := strings.ToLower(name)
		if existing, found := existingChars[nameLower]; found {
			// Update mention count for this chapter
			if existing.ChapterMentions == nil {
				existing.ChapterMentions = make(map[int]int)
			}
			existing.ChapterMentions[chapterIndex] = count
			existing.MentionCount = 0
			for _, c := range existing.ChapterMentions {
				existing.MentionCount += c
			}
		} else {
			// New character found
			newChar := types.Character{
				ID:              fmt.Sprintf("char-%d", time.Now().UnixNano()),
				Name:            name,
				Role:            "minor",
				IsAutoDetected:  true,
				MentionCount:    count,
				FirstChapter:    chapterIndex,
				ChapterMentions: map[int]int{chapterIndex: count},
				Attributes:      ExtractAttributes(text, name),
			}
			book.StoryBible.Characters = append(book.StoryBible.Characters, newChar)
			newChars++
		}
	}

	return types.IndexResult{
		Success:         true,
		CharactersFound: len(names),
		NewCharacters:   newChars,
		Characters:      book.StoryBible.Characters,
	}
}
