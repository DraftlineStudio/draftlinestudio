package indexing

import (
	"fmt"
	"strings"
	"time"

	"draftline/internal/entityresolution"
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

	// Include all detected names, even single mentions.
	// The UI can filter/highlight based on mention count if needed.
	newChars := 0
	updatedChars := 0

	for name, count := range allNames {
		// Include all names - previously filtered to 3+ which missed single-mention characters

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
		// Include all mentions - previously filtered to 2+ which missed single-mention characters

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

// BookWithEntityResolution performs full book indexing with entity resolution.
// This is the enhanced version that clusters name variations together.
// It updates both the StoryBible (for backward compat) and Analysis.EntityResolution.
func BookWithEntityResolution(book *types.BookData) types.IndexResult {
	// Initialize structures
	if book.StoryBible.Characters == nil {
		book.StoryBible.Characters = []types.Character{}
	}
	if book.Analysis.EntityResolution == nil {
		book.Analysis.EntityResolution = &types.EntityData{Version: 1}
	}

	// Collect all mentions across chapters
	var allMentions []entityresolution.Mention

	sections := []struct {
		name  string
		items []types.ChapterItem
	}{
		{"front_matter", book.FrontMatter},
		{"body", book.Body},
		{"back_matter", book.BackMatter},
	}

	chapterIndex := 0
	fullText := ""
	for _, section := range sections {
		for _, chapter := range section.items {
			text := StripHTML(chapter.Content)
			fullText += " " + text

			// Detect mentions with position information
			mentions := DetectMentions(text, chapterIndex, 0)
			allMentions = append(allMentions, mentions...)

			chapterIndex++
		}
	}

	// Load existing separated pairs from storage
	resolver := entityresolution.NewResolver()
	if book.Analysis.EntityResolution != nil {
		for _, pair := range book.Analysis.EntityResolution.SeparatedPairs {
			resolver.SeparatedPairs = append(resolver.SeparatedPairs, entityresolution.SeparatedPair{
				MentionID1: pair.MentionID1,
				MentionID2: pair.MentionID2,
				Reason:     pair.Reason,
			})
		}
	}

	// Run entity resolution
	result := resolver.ResolveEntities(allMentions)

	// Include all detected entities, including single-mention ones.
	// The UI can filter/highlight based on mention count if needed.
	// Previously filtered to 3+ mentions which caused single-mention characters to be missed.
	filteredEntities := result.Entities

	// Preserve manually created (non-auto-detected) characters
	preservedChars := make([]types.Character, 0)
	for _, char := range book.StoryBible.Characters {
		if !char.IsAutoDetected {
			preservedChars = append(preservedChars, char)
		}
	}

	// Convert entities to characters for backward compat
	newCharacters := ConvertEntitiesToCharacters(filteredEntities, allMentions)

	// Merge: add preserved chars, then new auto-detected chars
	// Check for duplicates by name
	charNames := make(map[string]bool)
	for _, char := range preservedChars {
		charNames[strings.ToLower(char.Name)] = true
	}

	for _, char := range newCharacters {
		if !charNames[strings.ToLower(char.Name)] {
			// Try to extract attributes
			char.Attributes = ExtractAttributes(fullText, char.Name)
			preservedChars = append(preservedChars, char)
			charNames[strings.ToLower(char.Name)] = true
		}
	}

	book.StoryBible.Characters = preservedChars

	// Store entity resolution data
	book.Analysis.EntityResolution = &types.EntityData{
		Mentions:       ConvertMentionsToRecords(allMentions),
		Entities:       ConvertEntitiesToRecords(filteredEntities),
		SeparatedPairs: ConvertSeparatedPairsToRecords(result.SeparatedPairs),
		LastResolved:   time.Now().Format(time.RFC3339),
		Version:        1,
	}

	return types.IndexResult{
		Success:           true,
		CharactersFound:   len(allMentions),
		NewCharacters:     len(newCharacters),
		UpdatedCharacters: 0,
		Characters:        book.StoryBible.Characters,
	}
}

// SplitCharacterEntity splits an entity into two, moving specified mentions
// to a new entity. This is used when auto-merging incorrectly combined two
// different people (e.g., two characters both called "Ruiz").
func SplitCharacterEntity(book *types.BookData, entityID string, mentionIDs []string, newCanonical string) error {
	if book.Analysis.EntityResolution == nil {
		return fmt.Errorf("no entity resolution data found")
	}

	// Convert stored records back to entityresolution types
	entities := make([]entityresolution.Entity, len(book.Analysis.EntityResolution.Entities))
	for i, e := range book.Analysis.EntityResolution.Entities {
		entities[i] = entityresolution.Entity{
			ID:         e.ID,
			Canonical:  e.Canonical,
			Aliases:    e.Aliases,
			MentionIDs: e.MentionIDs,
			Confidence: e.Confidence,
			Titles:     e.Titles,
		}
	}

	// Create resolver with existing separated pairs
	resolver := entityresolution.NewResolver()
	for _, pair := range book.Analysis.EntityResolution.SeparatedPairs {
		resolver.SeparatedPairs = append(resolver.SeparatedPairs, entityresolution.SeparatedPair{
			MentionID1: pair.MentionID1,
			MentionID2: pair.MentionID2,
			Reason:     pair.Reason,
		})
	}

	// Perform the split
	newEntities, err := resolver.SplitEntity(entities, entityID, mentionIDs)
	if err != nil {
		return err
	}

	// Set the canonical name for the new entity
	if newCanonical != "" {
		for i := range newEntities {
			// The new entity is the one with our mentionIDs
			hasMention := false
			for _, mid := range newEntities[i].MentionIDs {
				for _, targetID := range mentionIDs {
					if mid == targetID {
						hasMention = true
						break
					}
				}
				if hasMention {
					break
				}
			}
			if hasMention && newEntities[i].ID != entityID {
				newEntities[i].Canonical = newCanonical
				break
			}
		}
	}

	// Update stored data
	book.Analysis.EntityResolution.Entities = ConvertEntitiesToRecords(newEntities)
	book.Analysis.EntityResolution.SeparatedPairs = ConvertSeparatedPairsToRecords(resolver.SeparatedPairs)
	book.Analysis.EntityResolution.LastResolved = time.Now().Format(time.RFC3339)

	// Rebuild characters from entities
	mentions := make([]entityresolution.Mention, len(book.Analysis.EntityResolution.Mentions))
	for i, m := range book.Analysis.EntityResolution.Mentions {
		mentions[i] = entityresolution.Mention{
			ID:         m.ID,
			Text:       m.Text,
			SentenceID: m.SentenceID,
			Chapter:    m.Chapter,
			CharOffset: m.CharOffset,
		}
	}

	// Preserve manually created characters
	preservedChars := make([]types.Character, 0)
	for _, char := range book.StoryBible.Characters {
		if !char.IsAutoDetected {
			preservedChars = append(preservedChars, char)
		}
	}

	// Convert entities to characters
	newCharacters := ConvertEntitiesToCharacters(newEntities, mentions)

	// Merge
	charNames := make(map[string]bool)
	for _, char := range preservedChars {
		charNames[strings.ToLower(char.Name)] = true
	}
	for _, char := range newCharacters {
		if !charNames[strings.ToLower(char.Name)] {
			preservedChars = append(preservedChars, char)
		}
	}

	book.StoryBible.Characters = preservedChars

	return nil
}
