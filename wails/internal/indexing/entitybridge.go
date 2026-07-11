package indexing

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"draftline/internal/entityresolution"
	"draftline/internal/types"
)

// DetectMentions extracts name mentions with location information.
// This is an enhanced version of DetectCharacterNames that provides
// detailed position data for entity resolution.
func DetectMentions(text string, chapter int, baseOffset int) []entityresolution.Mention {
	mentions := []entityresolution.Mention{}

	// Track sentence boundaries for SentenceID
	sentences := splitIntoSentences(text)
	currentOffset := baseOffset

	for sentenceIdx, sentence := range sentences {
		sentenceID := fmt.Sprintf("ch%d-s%d", chapter, sentenceIdx)
		sentenceStart := currentOffset

		// Find all potential names in this sentence
		sentenceMentions := findNamesInText(sentence, sentenceID, chapter, sentenceStart)
		mentions = append(mentions, sentenceMentions...)

		currentOffset += len(sentence) + 1 // +1 for sentence separator
	}

	return mentions
}

// splitIntoSentences splits text into sentences.
func splitIntoSentences(text string) []string {
	// Simple sentence splitting on .!? followed by space or newline
	re := regexp.MustCompile(`[.!?]+\s+`)
	parts := re.Split(text, -1)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// findNamesInText finds potential character names in text with positions.
func findNamesInText(text string, sentenceID string, chapter int, baseOffset int) []entityresolution.Mention {
	mentions := []entityresolution.Mention{}

	// Multi-word name pattern: captures "Carlos" or "Carlos Ruiz" or "Carlos Ruiz Martinez"
	// Up to 3 words to avoid over-matching
	multiWordName := `([A-Z][a-z]{2,}(?:\s+[A-Z][a-z]{2,}){0,2})`

	// Pattern 1: Speaker AFTER closing quote
	// "Hello," said Carlos Ruiz.
	afterQuote := regexp.MustCompile(`[,.]"\s*(?i:` + DialogueVerbs + `)\s+` + multiWordName)
	// "Hello," Carlos Ruiz said.
	afterQuote2 := regexp.MustCompile(`[,.]"\s*` + multiWordName + `\s+(?i:` + DialogueVerbs + `)`)

	// Pattern 2: Speaker BEFORE opening quote
	// Carlos Ruiz said, "Hello"
	beforeQuote := regexp.MustCompile(`\b` + multiWordName + `\s+(?i:` + DialogueVerbs + `)\s*,?\s*"`)

	// Pattern 3: Title + Name (captures multi-word names after title)
	// Detective Carlos Ruiz, Dr. Jane Smith
	afterTitle := regexp.MustCompile(`\b(Mr|Mrs|Ms|Miss|Dr|Prof|Professor|Captain|Colonel|General|Lieutenant|Sergeant|Sgt|Officer|Detective|Agent|Lord|Lady|Sir|Dame|King|Queen|Prince|Princess|Senator|Governor|Mayor|Chief|Father|Mother|Sister|Brother|Uncle|Aunt)\.?\s+` + multiWordName)

	// Extract mentions with positions
	for _, match := range afterQuote.FindAllStringSubmatchIndex(text, -1) {
		if len(match) >= 4 {
			name := text[match[2]:match[3]]
			if !IsCommonWord(name) && !LooksLikeCommonWord(name) {
				mentions = append(mentions, entityresolution.Mention{
					ID:         fmt.Sprintf("m-%d", time.Now().UnixNano()),
					Text:       name,
					SentenceID: sentenceID,
					Chapter:    chapter,
					CharOffset: baseOffset + match[2],
				})
			}
		}
	}

	for _, match := range afterQuote2.FindAllStringSubmatchIndex(text, -1) {
		if len(match) >= 4 {
			name := text[match[2]:match[3]]
			if !IsCommonWord(name) && !LooksLikeCommonWord(name) {
				mentions = append(mentions, entityresolution.Mention{
					ID:         fmt.Sprintf("m-%d", time.Now().UnixNano()),
					Text:       name,
					SentenceID: sentenceID,
					Chapter:    chapter,
					CharOffset: baseOffset + match[2],
				})
			}
		}
	}

	for _, match := range beforeQuote.FindAllStringSubmatchIndex(text, -1) {
		if len(match) >= 4 {
			name := text[match[2]:match[3]]
			if !IsCommonWord(name) && !LooksLikeCommonWord(name) {
				mentions = append(mentions, entityresolution.Mention{
					ID:         fmt.Sprintf("m-%d", time.Now().UnixNano()),
					Text:       name,
					SentenceID: sentenceID,
					Chapter:    chapter,
					CharOffset: baseOffset + match[2],
				})
			}
		}
	}

	for _, match := range afterTitle.FindAllStringSubmatchIndex(text, -1) {
		// Full match is match[0]:match[1], title is match[2]:match[3], name is match[4]:match[5]
		if len(match) >= 6 {
			title := text[match[2]:match[3]]
			name := text[match[4]:match[5]]
			fullText := title + " " + name
			// Skip if context suggests this is not a character (e.g., "General Tso's Chicken")
			if !LooksLikeCommonWord(name) && !IsFalsePositiveContext(text, match[5]) {
				mentions = append(mentions, entityresolution.Mention{
					ID:         fmt.Sprintf("m-%d", time.Now().UnixNano()),
					Text:       fullText,
					SentenceID: sentenceID,
					Chapter:    chapter,
					CharOffset: baseOffset + match[2],
				})
			}
		}
	}

	return mentions
}

// ConvertEntitiesToCharacters converts resolved entities to Character records.
// This maintains backward compatibility with the existing StoryBible system.
func ConvertEntitiesToCharacters(entities []entityresolution.Entity, mentions []entityresolution.Mention) []types.Character {
	characters := make([]types.Character, 0, len(entities))

	// Build mention lookup for chapter information
	mentionMap := make(map[string]*entityresolution.Mention)
	for i := range mentions {
		mentionMap[mentions[i].ID] = &mentions[i]
	}

	for _, entity := range entities {
		// Calculate chapter mentions
		chapterMentions := make(map[int]int)
		firstChapter := -1

		for _, mid := range entity.MentionIDs {
			if m, ok := mentionMap[mid]; ok {
				chapterMentions[m.Chapter]++
				if firstChapter == -1 || m.Chapter < firstChapter {
					firstChapter = m.Chapter
				}
			}
		}

		if firstChapter == -1 {
			firstChapter = 0
		}

		char := types.Character{
			ID:              entity.ID,
			Name:            entity.Canonical,
			Role:            "minor", // Default role
			IsAutoDetected:  true,
			Aliases:         entity.Aliases,
			MentionCount:    len(entity.MentionIDs),
			FirstChapter:    firstChapter,
			ChapterMentions: chapterMentions,
		}
		characters = append(characters, char)
	}

	return characters
}

// ConvertMentionsToRecords converts entityresolution.Mention to types.MentionRecord.
func ConvertMentionsToRecords(mentions []entityresolution.Mention) []types.MentionRecord {
	records := make([]types.MentionRecord, len(mentions))
	for i, m := range mentions {
		records[i] = types.MentionRecord{
			ID:         m.ID,
			Text:       m.Text,
			SentenceID: m.SentenceID,
			Chapter:    m.Chapter,
			CharOffset: m.CharOffset,
		}
	}
	return records
}

// ConvertEntitiesToRecords converts entityresolution.Entity to types.EntityRecord.
func ConvertEntitiesToRecords(entities []entityresolution.Entity) []types.EntityRecord {
	records := make([]types.EntityRecord, len(entities))
	for i, e := range entities {
		records[i] = types.EntityRecord{
			ID:         e.ID,
			Canonical:  e.Canonical,
			Aliases:    e.Aliases,
			MentionIDs: e.MentionIDs,
			Confidence: e.Confidence,
			Titles:     e.Titles,
		}
	}
	return records
}

// ConvertSeparatedPairsToRecords converts entityresolution.SeparatedPair to types.SeparatedPairRecord.
func ConvertSeparatedPairsToRecords(pairs []entityresolution.SeparatedPair) []types.SeparatedPairRecord {
	records := make([]types.SeparatedPairRecord, len(pairs))
	for i, p := range pairs {
		records[i] = types.SeparatedPairRecord{
			MentionID1: p.MentionID1,
			MentionID2: p.MentionID2,
			Reason:     p.Reason,
		}
	}
	return records
}
