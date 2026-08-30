import { describe, expect, it } from 'vitest'
import type { Character, CharacterEvent, RelationshipRecord } from '../types/draftline'
import {
  confirmedCharacterIds,
  confirmedEvents,
  confirmedRelationships,
  isConfirmedCharacter,
} from './characterStatus'

function character(id: string, auto: boolean, status?: string): Character {
  return {
    id,
    name: id,
    role: 'minor',
    description: '',
    appearance: '',
    personality: '',
    motivation: '',
    notes: '',
    is_auto_detected: auto,
    detection_status: status,
  }
}

describe('confirmed character sidebar scope', () => {
  it('includes manual and accepted characters but excludes possible candidates', () => {
    expect(isConfirmedCharacter(character('manual', false))).toBe(true)
    expect(isConfirmedCharacter(character('accepted', true, 'accepted'))).toBe(true)
    expect(isConfirmedCharacter(character('review', true, 'review'))).toBe(false)
    expect(isConfirmedCharacter(character('legacy-unknown', true))).toBe(false)
    expect(isConfirmedCharacter(character('rejected', true, 'rejected'))).toBe(false)
  })

  it('removes relationships and events containing any possible character', () => {
    const ids = confirmedCharacterIds([
      character('hanlon', true, 'accepted'),
      character('mara', true, 'accepted'),
      character('wacker', true, 'review'),
    ])
    const relationships = [
      { id: 'real', character1_id: 'hanlon', character2_id: 'mara' },
      { id: 'false', character1_id: 'hanlon', character2_id: 'wacker' },
    ] as RelationshipRecord[]
    const events = [
      { id: 'real', character_ids: ['hanlon', 'mara'] },
      { id: 'false', character_ids: ['hanlon', 'wacker'] },
    ] as CharacterEvent[]

    expect(confirmedRelationships(relationships, ids).map(item => item.id)).toEqual(['real'])
    expect(confirmedEvents(events, ids).map(item => item.id)).toEqual(['real'])
  })
})
