import type { Character, CharacterEvent, RelationshipRecord } from '../types/draftline'

// Manually created characters are confirmed by definition. Auto-detected
// candidates must be explicitly accepted by the detector or the writer;
// legacy candidates without a status stay out of precision-first sidebars
// until the manuscript is re-detected.
export function isConfirmedCharacter(character: Character): boolean {
  return !character.is_auto_detected || character.detection_status === 'accepted'
}

export function confirmedCharacterIds(characters: Character[]): Set<string> {
  return new Set(characters.filter(isConfirmedCharacter).map(character => character.id))
}

export function confirmedRelationships(
  relationships: RelationshipRecord[],
  confirmedIds: Set<string>,
): RelationshipRecord[] {
  return relationships.filter(relationship =>
    confirmedIds.has(relationship.character1_id) && confirmedIds.has(relationship.character2_id))
}

export function confirmedEvents(events: CharacterEvent[], confirmedIds: Set<string>): CharacterEvent[] {
  return events.filter(event =>
    event.character_ids.length > 0 && event.character_ids.every(id => confirmedIds.has(id)))
}
