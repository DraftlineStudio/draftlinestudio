import { describe, expect, it } from 'vitest'
import { useStoryBibleStore } from '../storyBibleStore'
import type { BookData, Character } from '../../types/draftline'

function fixture(status: string): { book: BookData; character: Character } {
  const character: Character = {
    id: 'person', name: 'Nelwin', role: 'minor', description: '', appearance: '',
    personality: '', motivation: '', notes: '', is_auto_detected: true, detection_status: status,
  }
  const book = {
    story_bible: { characters: [character], plot_notes: '', timeline: '' },
    analysis: { entity_resolution: { entities: [], decisions: [] } },
  } as unknown as BookData
  return { book, character }
}

describe('writer confirmation', () => {
  it('does not turn an edit to an automatically accepted character into writer confirmation', () => {
    const { book, character } = fixture('accepted')
    const updated = useStoryBibleStore.getState().updateCharacter(book, { ...character, notes: 'Watchful' })
    expect(updated.analysis?.entity_resolution?.decisions).toEqual([])
  })

  it('records an explicit acceptance so it survives fresh detection', () => {
    const { book, character } = fixture('review')
    const updated = useStoryBibleStore.getState().updateCharacter(book, { ...character, detection_status: 'accepted' })
    expect(updated.analysis?.entity_resolution?.decisions).toEqual([{ names: ['Nelwin'], status: 'accepted' }])
  })
})
