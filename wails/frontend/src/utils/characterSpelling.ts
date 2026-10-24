import type { Character } from '../types/draftline'
import { isConfirmedCharacter } from './characterStatus'

export function characterSpellingNames(characters: Character[]): string[] {
  return characters.filter(isConfirmedCharacter).flatMap(character => [
    character.name,
    ...(character.is_auto_detected ? character.spelling_aliases ?? [] : character.aliases ?? []),
  ])
}
