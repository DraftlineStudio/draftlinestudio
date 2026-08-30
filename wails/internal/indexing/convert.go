package indexing

import (
	"fmt"
	"strings"

	"draftline/internal/entityresolution"
	"draftline/internal/types"
)

// Converters between entityresolution types and the stored record types.

// ConvertEntitiesToCharacters converts resolved entities to Character records
// for the StoryBible.
func ConvertEntitiesToCharacters(entities []entityresolution.Entity, mentions []entityresolution.Mention) []types.Character {
	characters := make([]types.Character, 0, len(entities))

	mentionMap := make(map[string]*entityresolution.Mention)
	for i := range mentions {
		mentionMap[mentions[i].ID] = &mentions[i]
	}

	// Distinct entities can share a canonical name (Dr. Chen vs Staff Sgt.
	// Chen are both "Chen" after title stripping) — disambiguate display
	// names with the entity's own titles.
	canonicalCount := map[string]int{}
	for _, entity := range entities {
		canonicalCount[strings.ToLower(entity.Canonical)]++
	}
	usedNames := map[string]bool{}

	for _, entity := range entities {
		if entity.DetectionStatus == "rejected" {
			continue
		}
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

		name := entity.Canonical
		if canonicalCount[strings.ToLower(name)] > 1 && len(entity.Titles) > 0 {
			name = strings.Join(entity.Titles, " ") + " " + entity.Canonical
		}
		for n := 2; usedNames[strings.ToLower(name)]; n++ {
			name = fmt.Sprintf("%s (%d)", entity.Canonical, n)
		}
		usedNames[strings.ToLower(name)] = true

		characters = append(characters, types.Character{
			ID:              entity.ID,
			Name:            name,
			Role:            "minor", // Default role
			IsAutoDetected:  true,
			Aliases:         entity.Aliases,
			MentionCount:    len(entity.MentionIDs),
			FirstChapter:    firstChapter,
			ChapterMentions: chapterMentions,
			EntityKind:      entity.Kind,
			DetectionStatus: entity.DetectionStatus,
			DetectionScore:  entity.DetectionScore,
		})
	}

	return characters
}

// ConvertMentionsToRecords converts entityresolution.Mention to types.MentionRecord.
func ConvertMentionsToRecords(mentions []entityresolution.Mention) []types.MentionRecord {
	records := make([]types.MentionRecord, len(mentions))
	for i, m := range mentions {
		records[i] = types.MentionRecord{
			ID:                   m.ID,
			Text:                 m.Text,
			SentenceID:           m.SentenceID,
			Chapter:              m.Chapter,
			CharOffset:           m.CharOffset,
			PersonEvidence:       m.PersonEvidence,
			NonPersonEvidence:    m.NonPersonEvidence,
			StrongPersonEvidence: m.StrongPersonEvidence,
		}
	}
	return records
}

// ConvertEntitiesToRecords converts entityresolution.Entity to types.EntityRecord.
func ConvertEntitiesToRecords(entities []entityresolution.Entity) []types.EntityRecord {
	records := make([]types.EntityRecord, len(entities))
	for i, e := range entities {
		records[i] = types.EntityRecord{
			ID:              e.ID,
			Canonical:       e.Canonical,
			Aliases:         e.Aliases,
			MentionIDs:      e.MentionIDs,
			Confidence:      e.Confidence,
			Titles:          e.Titles,
			Kind:            e.Kind,
			DetectionStatus: e.DetectionStatus,
			DetectionScore:  e.DetectionScore,
		}
	}
	return records
}

// recordsToMentions converts stored mention records back to resolver mentions.
func recordsToMentions(records []types.MentionRecord) []entityresolution.Mention {
	mentions := make([]entityresolution.Mention, len(records))
	for i, m := range records {
		mentions[i] = entityresolution.Mention{
			ID:                   m.ID,
			Text:                 m.Text,
			SentenceID:           m.SentenceID,
			Chapter:              m.Chapter,
			CharOffset:           m.CharOffset,
			PersonEvidence:       m.PersonEvidence,
			NonPersonEvidence:    m.NonPersonEvidence,
			StrongPersonEvidence: m.StrongPersonEvidence,
		}
	}
	return mentions
}

// recordsToEntities converts stored entity records back to resolver entities.
func recordsToEntities(records []types.EntityRecord) []entityresolution.Entity {
	entities := make([]entityresolution.Entity, len(records))
	for i, e := range records {
		entities[i] = entityresolution.Entity{
			ID:              e.ID,
			Canonical:       e.Canonical,
			Aliases:         e.Aliases,
			MentionIDs:      e.MentionIDs,
			Confidence:      e.Confidence,
			Titles:          e.Titles,
			Kind:            e.Kind,
			DetectionStatus: e.DetectionStatus,
			DetectionScore:  e.DetectionScore,
		}
	}
	return entities
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
