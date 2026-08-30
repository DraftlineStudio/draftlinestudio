package indexing

import (
	"fmt"
	"strings"

	"draftline/internal/types"
)

// Manual entity correction: merging characters the resolver kept apart.
// Merges are stored as name-based MergeRules so they survive re-indexing
// even after the text (and therefore mention IDs) change.

// MergeCharacterEntities merges two or more entities into one. The first ID
// is the primary; canonical overrides the display name if non-empty.
// A MergeRule is recorded for each absorbed entity.
func MergeCharacterEntities(book *types.BookData, entityIDs []string, canonical string) error {
	if book.Analysis.EntityResolution == nil {
		return fmt.Errorf("no entity resolution data found")
	}
	if len(entityIDs) < 2 {
		return fmt.Errorf("need at least 2 entities to merge")
	}

	ed := book.Analysis.EntityResolution
	byID := make(map[string]int)
	for i, e := range ed.Entities {
		byID[e.ID] = i
	}

	primaryIdx, ok := byID[entityIDs[0]]
	if !ok {
		return fmt.Errorf("entity %s not found", entityIDs[0])
	}

	// Record name rules before mutating, then absorb.
	absorbed := map[string]bool{}
	for _, id := range entityIDs[1:] {
		idx, ok := byID[id]
		if !ok {
			return fmt.Errorf("entity %s not found", id)
		}
		if id == entityIDs[0] {
			continue
		}
		ed.MergeRules = append(ed.MergeRules, types.MergeRule{
			Name1: ed.Entities[primaryIdx].Canonical,
			Name2: ed.Entities[idx].Canonical,
		})
		mergeEntityRecords(&ed.Entities[primaryIdx], &ed.Entities[idx])
		absorbed[id] = true
	}

	remaining := make([]types.EntityRecord, 0, len(ed.Entities))
	for _, e := range ed.Entities {
		if !absorbed[e.ID] {
			remaining = append(remaining, e)
		}
	}
	ed.Entities = remaining

	if canonical != "" {
		for i := range ed.Entities {
			if ed.Entities[i].ID == entityIDs[0] {
				setCanonical(&ed.Entities[i], canonical)
			}
		}
	}

	rebuildCharactersFromEntities(book)

	// Relationship data references the absorbed entity IDs — stale now.
	// Clearing it makes the Relationships panel prompt a re-analyze.
	book.Analysis.Relationships = nil

	return nil
}

// applyMergeRules re-applies stored "same person" rules after a fresh
// resolution run: any two entities matching a rule's names are merged.
func applyMergeRules(entities []types.EntityRecord, rules []types.MergeRule) []types.EntityRecord {
	for _, rule := range rules {
		idx1 := findEntityByName(entities, rule.Name1)
		idx2 := findEntityByName(entities, rule.Name2)
		if idx1 == -1 || idx2 == -1 || idx1 == idx2 {
			continue
		}
		// Keep the entity with the rule's primary name.
		mergeEntityRecords(&entities[idx1], &entities[idx2])
		entities = append(entities[:idx2], entities[idx2+1:]...)
	}
	return entities
}

// findEntityByName locates an entity whose canonical name or alias matches
// (case-insensitive).
func findEntityByName(entities []types.EntityRecord, name string) int {
	lower := strings.ToLower(name)
	for i, e := range entities {
		if strings.ToLower(e.Canonical) == lower {
			return i
		}
	}
	for i, e := range entities {
		for _, a := range e.Aliases {
			if strings.ToLower(a) == lower {
				return i
			}
		}
	}
	return -1
}

// mergeEntityRecords absorbs entity b into entity a.
func mergeEntityRecords(a, b *types.EntityRecord) {
	seen := map[string]bool{strings.ToLower(a.Canonical): true}
	for _, al := range a.Aliases {
		seen[strings.ToLower(al)] = true
	}
	add := func(s string) {
		if s != "" && !seen[strings.ToLower(s)] {
			seen[strings.ToLower(s)] = true
			a.Aliases = append(a.Aliases, s)
		}
	}
	add(b.Canonical)
	for _, al := range b.Aliases {
		add(al)
	}

	a.MentionIDs = append(a.MentionIDs, b.MentionIDs...)

	for _, t := range b.Titles {
		found := false
		for _, existing := range a.Titles {
			if strings.EqualFold(strings.TrimSuffix(existing, "."), strings.TrimSuffix(t, ".")) {
				found = true
				break
			}
		}
		if !found {
			a.Titles = append(a.Titles, t)
		}
	}
}

// setCanonical changes an entity's display name, demoting the old one to an alias.
func setCanonical(e *types.EntityRecord, canonical string) {
	if strings.EqualFold(e.Canonical, canonical) {
		e.Canonical = canonical
		return
	}
	old := e.Canonical
	e.Canonical = canonical
	// Demote old canonical to alias; remove new canonical from aliases.
	aliases := make([]string, 0, len(e.Aliases)+1)
	for _, a := range e.Aliases {
		if !strings.EqualFold(a, canonical) {
			aliases = append(aliases, a)
		}
	}
	found := false
	for _, a := range aliases {
		if strings.EqualFold(a, old) {
			found = true
		}
	}
	if !found {
		aliases = append(aliases, old)
	}
	e.Aliases = aliases
}

// rebuildCharactersFromEntities regenerates auto-detected StoryBible
// characters from the current entity records, preserving manual ones.
func rebuildCharactersFromEntities(book *types.BookData) {
	ed := book.Analysis.EntityResolution

	mentions := recordsToMentions(ed.Mentions)
	entities := recordsToEntities(ed.Entities)

	// Keep author curation from auto-detected records. Detection owns mention
	// metrics and aliases; the author owns role, description, appearance,
	// personality, motivation, and notes. Match by any prior name/alias so a
	// better canonical spelling does not erase that work.
	curatedByName := make(map[string][]types.Character)
	for _, char := range book.StoryBible.Characters {
		if !char.IsAutoDetected {
			continue
		}
		for _, name := range append([]string{char.Name}, char.Aliases...) {
			key := strings.ToLower(strings.TrimSpace(name))
			if key != "" {
				curatedByName[key] = append(curatedByName[key], char)
			}
		}
	}

	preservedChars := make([]types.Character, 0)
	charNames := make(map[string]bool)
	for _, char := range book.StoryBible.Characters {
		if !char.IsAutoDetected {
			preservedChars = append(preservedChars, char)
			charNames[strings.ToLower(char.Name)] = true
		}
	}
	for _, char := range ConvertEntitiesToCharacters(entities, mentions) {
		var matches []types.Character
		seen := make(map[string]bool)
		for _, name := range append([]string{char.Name}, char.Aliases...) {
			for _, old := range curatedByName[strings.ToLower(strings.TrimSpace(name))] {
				if !seen[old.ID] {
					matches = append(matches, old)
					seen[old.ID] = true
				}
			}
		}
		if len(matches) == 1 {
			old := matches[0]
			char.Role = old.Role
			char.Description = old.Description
			char.Appearance = old.Appearance
			char.Personality = old.Personality
			char.Motivation = old.Motivation
			char.Notes = old.Notes
			if old.DetectionStatus == "accepted" {
				char.DetectionStatus = "accepted"
			}
		}
		if !charNames[strings.ToLower(char.Name)] {
			preservedChars = append(preservedChars, char)
			charNames[strings.ToLower(char.Name)] = true
		}
	}
	book.StoryBible.Characters = preservedChars
}
