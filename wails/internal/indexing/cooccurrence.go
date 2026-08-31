package indexing

import (
	"fmt"
	"sort"
	"strings"

	"draftline/internal/types"
)

// CoOccurrenceConfig controls detection sensitivity.
type CoOccurrenceConfig struct {
	SameSentenceWeight  float64
	SameParagraphWeight float64
	SameSceneWeight     float64
	DialogueWeight      float64
	MinConfidence       float64
}

// DefaultCoOccurrenceConfig returns standard configuration.
func DefaultCoOccurrenceConfig() CoOccurrenceConfig {
	return CoOccurrenceConfig{
		SameSentenceWeight:  0.9,
		SameParagraphWeight: 0.7,
		SameSceneWeight:     0.5,
		DialogueWeight:      1.0,
		MinConfidence:       0.3,
	}
}

// DetectInteractions finds all character interactions in a chapter.
func DetectInteractions(
	chapterIndex int,
	chapterText string,
	scenes []types.SceneRecord,
	mentions []types.MentionRecord,
	entityMap map[string]string, // mentionID -> entityID
	config CoOccurrenceConfig,
) []types.InteractionRecord {
	interactions := []types.InteractionRecord{}

	// Group mentions by scene
	mentionsByScene := groupMentionsByScene(scenes, mentions)

	for _, scene := range scenes {
		sceneMentions := mentionsByScene[scene.ID]
		if len(sceneMentions) < 2 {
			continue // Need at least 2 mentions for interaction
		}

		// Get unique entities in this scene
		entitiesInScene := getUniqueEntities(sceneMentions, entityMap)
		if len(entitiesInScene) < 2 {
			continue // Need at least 2 different characters
		}

		// Check for dialogue interactions
		dialogueInteractions := detectDialogueInteractions(
			chapterIndex, chapterText, scene, sceneMentions, entityMap, config,
		)
		interactions = append(interactions, dialogueInteractions...)

		// Check for co-occurrence interactions
		cooccurrenceInteractions := detectCoOccurrenceInteractions(
			chapterIndex, scene, sceneMentions, entityMap, config,
		)
		interactions = append(interactions, cooccurrenceInteractions...)
	}

	// Deduplicate interactions (same participants in same scene)
	interactions = deduplicateInteractions(interactions)

	// Assign deterministic IDs in document order.
	for i := range interactions {
		interactions[i].ID = fmt.Sprintf("int-%d-%d", chapterIndex, i)
	}

	return interactions
}

// groupMentionsByScene assigns mentions to their containing scenes.
func groupMentionsByScene(scenes []types.SceneRecord, mentions []types.MentionRecord) map[string][]types.MentionRecord {
	result := make(map[string][]types.MentionRecord)

	for _, scene := range scenes {
		result[scene.ID] = []types.MentionRecord{}
	}

	for _, mention := range mentions {
		for _, scene := range scenes {
			if mention.Chapter == scene.ChapterIndex &&
				mention.CharOffset >= scene.StartOffset &&
				mention.CharOffset < scene.EndOffset {
				result[scene.ID] = append(result[scene.ID], mention)
				break // Each mention belongs to one scene
			}
		}
	}

	return result
}

// getUniqueEntities returns unique entity IDs from mentions.
func getUniqueEntities(mentions []types.MentionRecord, entityMap map[string]string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, mention := range mentions {
		if entityID, ok := entityMap[mention.ID]; ok {
			if !seen[entityID] {
				seen[entityID] = true
				result = append(result, entityID)
			}
		}
	}

	return result
}

// detectDialogueInteractions finds characters speaking to or about each other.
func detectDialogueInteractions(
	chapterIndex int,
	chapterText string,
	scene types.SceneRecord,
	mentions []types.MentionRecord,
	entityMap map[string]string,
	config CoOccurrenceConfig,
) []types.InteractionRecord {
	interactions := []types.InteractionRecord{}

	// Extract scene text
	sceneText := ""
	if scene.EndOffset <= len(chapterText) {
		sceneText = chapterText[scene.StartOffset:scene.EndOffset]
	}

	// Find dialogue patterns (text in quotes)
	dialogueRanges := findDialogueRanges(sceneText)

	// For each dialogue, find nearby character mentions
	for _, dRange := range dialogueRanges {
		// Adjust offsets to chapter-level
		dialogueStart := scene.StartOffset + dRange.Start
		dialogueEnd := scene.StartOffset + dRange.End

		// Find speaker (character mention before dialogue)
		speaker := findSpeaker(dialogueStart, mentions, entityMap)

		// Find addressee (character mentioned in or just after dialogue)
		addressee := findAddressee(dialogueStart, dialogueEnd, mentions, entityMap, speaker)

		if speaker != "" && addressee != "" && speaker != addressee {
			interaction := types.InteractionRecord{
				Participants:    sortedPair(speaker, addressee),
				ChapterIndex:    chapterIndex,
				SceneID:         scene.ID,
				InteractionType: "dialogue",
				DirectedFrom:    speaker,
				DirectedTo:      addressee,
				Confidence:      config.DialogueWeight,
				TextSnippet:     truncateText(dRange.Text, 100),
			}
			interactions = append(interactions, interaction)
		}
	}

	return interactions
}

// DialogueRange represents a quoted section of text.
type DialogueRange struct {
	Start int
	End   int
	Text  string
}

// findDialogueRanges locates quoted text in the scene.
func findDialogueRanges(text string) []DialogueRange {
	ranges := []DialogueRange{}

	// Handle various quote styles. Must align with the dialogue delimiters the
	// mention scanner recognizes in isQuoteInitial: straight double, curly
	// double (U+201C/U+201D), and guillemets. Straight and curly SINGLE quotes
	// are deliberately excluded — their closing forms are indistinguishable
	// from apostrophes ("Kira's") and would fabricate dialogue from possessives.
	quoteChars := []struct {
		open  string
		close string
	}{
		{`"`, `"`}, // straight double (open/close identical)
		{"“", "”"}, // curly double “ ”
		{`«`, `»`}, // guillemets
	}

	for _, q := range quoteChars {
		pos := 0
		for {
			start := strings.Index(text[pos:], q.open)
			if start == -1 {
				break
			}
			start += pos

			end := strings.Index(text[start+len(q.open):], q.close)
			if end == -1 {
				break
			}
			end += start + len(q.open)

			dialogueText := text[start+len(q.open) : end]
			if len(dialogueText) > 0 {
				ranges = append(ranges, DialogueRange{
					Start: start,
					End:   end + len(q.close),
					Text:  dialogueText,
				})
			}

			pos = end + len(q.close)
		}
	}

	return ranges
}

// findSpeaker looks for a character mention before the dialogue.
func findSpeaker(dialogueStart int, mentions []types.MentionRecord, entityMap map[string]string) string {
	var closestMention *types.MentionRecord
	closestDist := 200 // Max characters to look back

	for i := range mentions {
		m := &mentions[i]
		dist := dialogueStart - m.CharOffset
		if dist > 0 && dist < closestDist {
			closestDist = dist
			closestMention = m
		}
	}

	if closestMention != nil {
		if entityID, ok := entityMap[closestMention.ID]; ok {
			return entityID
		}
	}

	return ""
}

// findAddressee looks for character mentioned in dialogue or being addressed.
func findAddressee(dialogueStart, dialogueEnd int, mentions []types.MentionRecord, entityMap map[string]string, speakerID string) string {
	// First, check for direct address within dialogue
	for _, m := range mentions {
		if m.CharOffset >= dialogueStart && m.CharOffset < dialogueEnd {
			if entityID, ok := entityMap[m.ID]; ok {
				if entityID != speakerID {
					return entityID
				}
			}
		}
	}

	// Then check for mention just after dialogue (dialogue tag)
	for _, m := range mentions {
		dist := m.CharOffset - dialogueEnd
		if dist >= 0 && dist < 100 { // Within 100 chars after
			if entityID, ok := entityMap[m.ID]; ok {
				if entityID != speakerID {
					return entityID
				}
			}
		}
	}

	return ""
}

// detectCoOccurrenceInteractions finds characters appearing together.
func detectCoOccurrenceInteractions(
	chapterIndex int,
	scene types.SceneRecord,
	mentions []types.MentionRecord,
	entityMap map[string]string,
	config CoOccurrenceConfig,
) []types.InteractionRecord {
	interactions := []types.InteractionRecord{}

	// Group by entity
	mentionsByEntity := make(map[string][]types.MentionRecord)
	for _, m := range mentions {
		if entityID, ok := entityMap[m.ID]; ok {
			mentionsByEntity[entityID] = append(mentionsByEntity[entityID], m)
		}
	}

	// Get entity list for pairwise comparison
	entities := make([]string, 0, len(mentionsByEntity))
	for entityID := range mentionsByEntity {
		entities = append(entities, entityID)
	}
	sort.Strings(entities)

	// Compare each pair of entities
	for i := 0; i < len(entities); i++ {
		for j := i + 1; j < len(entities); j++ {
			entityA := entities[i]
			entityB := entities[j]

			// Find closest mention pair
			closestDist := -1
			for _, mA := range mentionsByEntity[entityA] {
				for _, mB := range mentionsByEntity[entityB] {
					dist := abs(mA.CharOffset - mB.CharOffset)
					if closestDist == -1 || dist < closestDist {
						closestDist = dist
					}
				}
			}

			// Calculate confidence based on proximity
			var confidence float64
			var interactionType string

			if closestDist < 100 { // Same sentence (~100 chars)
				confidence = config.SameSentenceWeight
				interactionType = "co_occurrence"
			} else if closestDist < 500 { // Same paragraph
				confidence = config.SameParagraphWeight
				interactionType = "co_occurrence"
			} else {
				confidence = config.SameSceneWeight
				interactionType = "co_occurrence"
			}

			if confidence >= config.MinConfidence {
				interaction := types.InteractionRecord{
					Participants:    sortedPair(entityA, entityB),
					ChapterIndex:    chapterIndex,
					SceneID:         scene.ID,
					InteractionType: interactionType,
					Confidence:      confidence,
				}
				interactions = append(interactions, interaction)
			}
		}
	}

	return interactions
}

// sortedPair returns two strings in alphabetical order.
func sortedPair(a, b string) []string {
	if a < b {
		return []string{a, b}
	}
	return []string{b, a}
}

// deduplicateInteractions removes duplicate interactions, keeping the higher
// confidence record. Input order is preserved for deterministic output.
func deduplicateInteractions(interactions []types.InteractionRecord) []types.InteractionRecord {
	indexByKey := make(map[string]int)
	result := make([]types.InteractionRecord, 0, len(interactions))

	for _, inter := range interactions {
		// Create a key from participants, scene, and type
		key := fmt.Sprintf("%s:%s:%s:%s",
			inter.SceneID,
			strings.Join(inter.Participants, ","),
			inter.InteractionType,
			inter.DirectedFrom+">"+inter.DirectedTo,
		)

		if idx, ok := indexByKey[key]; ok {
			if inter.Confidence > result[idx].Confidence {
				result[idx] = inter
			}
		} else {
			indexByKey[key] = len(result)
			result = append(result, inter)
		}
	}

	return result
}

// truncateText shortens text to maxLen characters.
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// abs returns absolute value.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// BuildRelationshipsFromInteractions aggregates interactions into relationships.
func BuildRelationshipsFromInteractions(interactions []types.InteractionRecord) []types.RelationshipRecord {
	// Group interactions by participant pair
	pairInteractions := make(map[string][]types.InteractionRecord)

	for _, inter := range interactions {
		if len(inter.Participants) != 2 {
			continue
		}
		key := strings.Join(inter.Participants, ":")
		pairInteractions[key] = append(pairInteractions[key], inter)
	}

	// Build relationship records in deterministic (sorted key) order.
	keys := make([]string, 0, len(pairInteractions))
	for key := range pairInteractions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	relationships := []types.RelationshipRecord{}

	for _, key := range keys {
		inters := pairInteractions[key]
		parts := strings.Split(key, ":")
		if len(parts) != 2 {
			continue
		}

		// Aggregate data
		chapters := make(map[int]bool)
		typeCount := make(map[string]int)
		interactionIDs := make([]string, 0, len(inters))
		totalConfidence := 0.0

		for _, inter := range inters {
			chapters[inter.ChapterIndex] = true
			typeCount[inter.InteractionType]++
			interactionIDs = append(interactionIDs, inter.ID)
			totalConfidence += inter.Confidence
		}

		// Get chapter range
		chapterList := make([]int, 0, len(chapters))
		for ch := range chapters {
			chapterList = append(chapterList, ch)
		}
		sort.Ints(chapterList)

		// Calculate strength
		// Formula: log(count+1) * type_weight + span_bonus
		dialogueWeight := float64(typeCount["dialogue"]) * 1.0
		cooccurrWeight := float64(typeCount["co_occurrence"]) * 0.5
		referenceWeight := float64(typeCount["reference"]) * 0.3

		baseStrength := (dialogueWeight + cooccurrWeight + referenceWeight) / float64(len(inters))
		spanBonus := float64(len(chapterList)) * 0.05

		strength := baseStrength + spanBonus
		if strength > 1.0 {
			strength = 1.0
		}

		rel := types.RelationshipRecord{
			ID:               fmt.Sprintf("rel-%d", len(relationships)+1),
			Character1ID:     parts[0],
			Character2ID:     parts[1],
			FirstChapter:     chapterList[0],
			LastChapter:      chapterList[len(chapterList)-1],
			InteractionCount: len(inters),
			Strength:         strength,
			ChapterHistory:   chapterList,
			InteractionIDs:   interactionIDs,
			TypeBreakdown:    typeCount,
		}

		relationships = append(relationships, rel)
	}

	// Sort by strength descending (stable for deterministic output)
	sort.SliceStable(relationships, func(i, j int) bool {
		return relationships[i].Strength > relationships[j].Strength
	})

	return relationships
}
