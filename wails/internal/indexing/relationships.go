package indexing

import (
	"fmt"
	"time"

	"draftline/internal/types"
)

// RelationshipAnalyzer orchestrates the full relationship analysis pipeline.
type RelationshipAnalyzer struct {
	config CoOccurrenceConfig
}

// NewRelationshipAnalyzer creates a new analyzer with default config.
func NewRelationshipAnalyzer() *RelationshipAnalyzer {
	return &RelationshipAnalyzer{
		config: DefaultCoOccurrenceConfig(),
	}
}

// AnalyzeBook performs full relationship analysis on a book.
func (ra *RelationshipAnalyzer) AnalyzeBook(book types.BookData) (*types.RelationshipData, error) {
	// Validate input
	if book.Analysis.EntityResolution == nil {
		return nil, fmt.Errorf("book must have entity resolution data before relationship analysis")
	}

	entityData := book.Analysis.EntityResolution
	if len(entityData.Entities) == 0 {
		return &types.RelationshipData{
			LastAnalyzed: time.Now().Format(time.RFC3339),
			Version:      1,
		}, nil
	}

	// Build mention-to-entity lookup
	entityMap := BuildMentionToEntityMap(entityData.Entities)

	// Collect all scenes and interactions across chapters. Chapters MUST be
	// enumerated exactly as IndexBook does (AllChapters), and text MUST be
	// stripped the same way, so chapter indexes and offsets line up with
	// the stored mentions.
	allScenes := []types.SceneRecord{}
	allInteractions := []types.InteractionRecord{}

	for chIdx, chapter := range AllChapters(&book) {
		chapterText := ""
		if ShouldAnalyzeChapter(chapter) {
			chapterText = StripHTMLForAnalysis(chapter.Content)
		}

		// Detect scenes/paragraphs in this chapter
		scenes := DetectScenes(chapterText, chIdx)

		// Get mentions for this chapter
		chapterMentions := filterMentionsByChapter(entityData.Mentions, chIdx)

		// Populate character IDs for each scene
		for i := range scenes {
			scenes[i].CharacterIDs = GetCharactersInScene(scenes[i], chapterMentions, entityMap)
		}

		allScenes = append(allScenes, scenes...)

		// Detect interactions in this chapter
		interactions := DetectInteractions(
			chIdx,
			chapterText,
			scenes,
			chapterMentions,
			entityMap,
			ra.config,
		)

		allInteractions = append(allInteractions, interactions...)
	}

	// Build aggregated relationships from interactions
	relationships := BuildRelationshipsFromInteractions(allInteractions)

	// Auto-detect significant events
	events := ra.detectCharacterEvents(book, relationships, entityMap)

	return &types.RelationshipData{
		Scenes:        allScenes,
		Interactions:  allInteractions,
		Relationships: relationships,
		Events:        events,
		LastAnalyzed:  time.Now().Format(time.RFC3339),
		Version:       1,
	}, nil
}

// filterMentionsByChapter returns mentions that belong to a specific chapter.
func filterMentionsByChapter(mentions []types.MentionRecord, chapterIndex int) []types.MentionRecord {
	result := []types.MentionRecord{}
	for _, m := range mentions {
		if m.Chapter == chapterIndex {
			result = append(result, m)
		}
	}
	return result
}

// detectCharacterEvents identifies significant plot points.
func (ra *RelationshipAnalyzer) detectCharacterEvents(
	book types.BookData,
	relationships []types.RelationshipRecord,
	entityMap map[string]string,
) []types.CharacterEvent {
	events := []types.CharacterEvent{}
	entityData := book.Analysis.EntityResolution

	// Track first appearance of each character
	firstAppearance := make(map[string]int)
	for _, mention := range entityData.Mentions {
		if entityID, ok := entityMap[mention.ID]; ok {
			if existing, exists := firstAppearance[entityID]; !exists || mention.Chapter < existing {
				firstAppearance[entityID] = mention.Chapter
			}
		}
	}

	// Create introduction events (entity order for determinism)
	for _, entity := range entityData.Entities {
		chapter, ok := firstAppearance[entity.ID]
		if !ok {
			continue
		}
		events = append(events, types.CharacterEvent{
			ID:             fmt.Sprintf("evt-%d", len(events)+1),
			CharacterIDs:   []string{entity.ID},
			ChapterIndex:   chapter,
			EventType:      "introduction",
			Description:    fmt.Sprintf("%s first appears", entity.Canonical),
			IsAutoDetected: true,
		})
	}

	// Create "first meeting" events for relationships
	for _, rel := range relationships {
		// Find character names
		char1Name := ""
		char2Name := ""
		for _, entity := range entityData.Entities {
			if entity.ID == rel.Character1ID {
				char1Name = entity.Canonical
			}
			if entity.ID == rel.Character2ID {
				char2Name = entity.Canonical
			}
		}

		// Only create meeting event if relationship is significant
		if rel.InteractionCount >= 2 {
			event := types.CharacterEvent{
				ID:             fmt.Sprintf("evt-%d", len(events)+1),
				CharacterIDs:   []string{rel.Character1ID, rel.Character2ID},
				ChapterIndex:   rel.FirstChapter,
				EventType:      "meeting",
				Description:    fmt.Sprintf("%s and %s meet", char1Name, char2Name),
				IsAutoDetected: true,
			}
			events = append(events, event)
		}
	}

	return events
}

// GetCharacterRelationships returns relationships for a specific character.
func GetCharacterRelationships(characterID string, relationships []types.RelationshipRecord) []types.RelationshipRecord {
	result := []types.RelationshipRecord{}
	for _, rel := range relationships {
		if rel.Character1ID == characterID || rel.Character2ID == characterID {
			result = append(result, rel)
		}
	}
	return result
}

// GetCharacterTimeline builds a timeline of events for a character.
func GetCharacterTimeline(
	characterID string,
	book types.BookData,
	relationshipData *types.RelationshipData,
) []types.CharacterTimelineEvent {
	if book.Analysis.EntityResolution == nil {
		return []types.CharacterTimelineEvent{}
	}

	timeline := []types.CharacterTimelineEvent{}
	entityData := book.Analysis.EntityResolution
	entityMap := BuildMentionToEntityMap(entityData.Entities)

	// Group mentions by chapter
	mentionsByChapter := make(map[int]int)
	for _, mention := range entityData.Mentions {
		if entID, ok := entityMap[mention.ID]; ok && entID == characterID {
			mentionsByChapter[mention.Chapter]++
		}
	}

	// Add mention events
	for chapter, count := range mentionsByChapter {
		event := types.CharacterTimelineEvent{
			Chapter:     chapter,
			EventType:   "mention",
			Description: fmt.Sprintf("Mentioned %d times", count),
		}
		timeline = append(timeline, event)
	}

	// Add interaction events
	if relationshipData != nil {
		for _, interaction := range relationshipData.Interactions {
			isParticipant := false
			for _, p := range interaction.Participants {
				if p == characterID {
					isParticipant = true
					break
				}
			}

			if isParticipant {
				// Find other participant's name
				otherID := ""
				for _, p := range interaction.Participants {
					if p != characterID {
						otherID = p
						break
					}
				}

				otherName := ""
				for _, entity := range entityData.Entities {
					if entity.ID == otherID {
						otherName = entity.Canonical
						break
					}
				}

				eventType := "interaction"
				description := fmt.Sprintf("Interacts with %s", otherName)
				if interaction.InteractionType == "dialogue" {
					eventType = "dialogue"
					if interaction.DirectedFrom == characterID {
						description = fmt.Sprintf("Speaks to %s", otherName)
					} else {
						description = fmt.Sprintf("Spoken to by %s", otherName)
					}
				}

				event := types.CharacterTimelineEvent{
					Chapter:      interaction.ChapterIndex,
					EventType:    eventType,
					Description:  description,
					RelatedChars: []string{otherID},
				}
				timeline = append(timeline, event)
			}
		}

		// Add character events
		for _, evt := range relationshipData.Events {
			for _, charID := range evt.CharacterIDs {
				if charID == characterID {
					otherChars := []string{}
					for _, c := range evt.CharacterIDs {
						if c != characterID {
							otherChars = append(otherChars, c)
						}
					}

					event := types.CharacterTimelineEvent{
						Chapter:      evt.ChapterIndex,
						EventType:    evt.EventType,
						Description:  evt.Description,
						RelatedChars: otherChars,
					}
					timeline = append(timeline, event)
					break
				}
			}
		}
	}

	return timeline
}
