// Safety net for the live character path: CharactersView, RichEditor, and
// CharacterQuickRef all route through these transforms via bookStore's
// delegating wrappers. applyCharacterDecision (entity-resolution rewrite on
// accept/reject) was previously uncovered.

import { describe, it, expect, beforeEach } from 'vitest'
import { useStoryBibleStore } from '../storyBibleStore'
import type { BookData, Character } from '../../types/draftline'

function makeBook(overrides: Partial<BookData> = {}): BookData {
  return {
    version: '2.0',
    metadata: { title: 'T', author: '', publisher: '', created: '', modified: '' },
    copyright: '',
    front_matter: [],
    body: [],
    back_matter: [],
    story_bible: { characters: [], plot_notes: '', timeline: '' },
    ...overrides,
  } as BookData
}

function makeCharacter(overrides: Partial<Character> = {}): Character {
  return {
    id: 'c1',
    name: 'Mara',
    role: 'supporting',
    description: '',
    appearance: '',
    personality: '',
    motivation: '',
    notes: '',
    ...overrides,
  } as Character
}

describe('storyBibleStore character CRUD', () => {
  beforeEach(() => {
    useStoryBibleStore.setState({ highlightedCharacterId: null })
  })

  it('addCharacter appends without mutating the original book', () => {
    const book = makeBook()
    const updated = useStoryBibleStore.getState().addCharacter(book, makeCharacter())
    expect(updated.story_bible!.characters).toHaveLength(1)
    expect(book.story_bible!.characters).toHaveLength(0)
  })

  it('addCharacter tolerates a book with no story_bible', () => {
    const book = makeBook({ story_bible: undefined })
    const updated = useStoryBibleStore.getState().addCharacter(book, makeCharacter())
    expect(updated.story_bible!.characters.map(c => c.id)).toEqual(['c1'])
  })

  it('updateCharacter replaces by id', () => {
    const book = makeBook({
      story_bible: { characters: [makeCharacter(), makeCharacter({ id: 'c2', name: 'Hanlon' })], plot_notes: '', timeline: '' },
    })
    const updated = useStoryBibleStore.getState().updateCharacter(book, makeCharacter({ name: 'Mara Ruiz' }))
    expect(updated.story_bible!.characters.find(c => c.id === 'c1')!.name).toBe('Mara Ruiz')
    expect(updated.story_bible!.characters.find(c => c.id === 'c2')!.name).toBe('Hanlon')
  })

  it('deleteCharacter removes by id', () => {
    const book = makeBook({
      story_bible: { characters: [makeCharacter()], plot_notes: '', timeline: '' },
    })
    const updated = useStoryBibleStore.getState().deleteCharacter(book, 'c1')
    expect(updated.story_bible!.characters).toHaveLength(0)
  })
})

describe('applyCharacterDecision (entity resolution)', () => {
  const resolutionBook = (char: Character) =>
    makeBook({
      story_bible: { characters: [char], plot_notes: '', timeline: '' },
      analysis: {
        entity_resolution: {
          entities: [{ id: char.id, canonical: char.name, detection_status: 'pending' }],
          decisions: [
            { names: ['Mara', 'M.'], status: 'rejected' }, // stale decision for the same names
            { names: ['Hanlon'], status: 'accepted' },     // unrelated, must survive
          ],
        },
      },
    } as unknown as Partial<BookData>)

  it('accepting an auto-detected character marks its entity and replaces stale decisions', () => {
    const char = makeCharacter({ is_auto_detected: true, aliases: ['M.'], detection_status: 'accepted' })
    const updated = useStoryBibleStore.getState().updateCharacter(resolutionBook(char), char)
    const resolution = updated.analysis!.entity_resolution! as any
    expect(resolution.entities[0].detection_status).toBe('accepted')
    const decisionsFor = (name: string) =>
      resolution.decisions.filter((d: any) => d.names.some((n: string) => n.toLowerCase() === name.toLowerCase()))
    expect(decisionsFor('Mara')).toHaveLength(1)
    expect(decisionsFor('Mara')[0].status).toBe('accepted')
    expect(decisionsFor('Hanlon')).toHaveLength(1) // unrelated decision preserved
  })

  it('deleting an auto-detected character records a rejection', () => {
    const char = makeCharacter({ is_auto_detected: true })
    const updated = useStoryBibleStore.getState().deleteCharacter(resolutionBook(char), char.id)
    const resolution = updated.analysis!.entity_resolution! as any
    expect(resolution.entities[0].detection_status).toBe('rejected')
    const maraDecisions = resolution.decisions.filter((d: any) => d.names.includes('Mara'))
    expect(maraDecisions).toHaveLength(1)
    expect(maraDecisions[0].status).toBe('rejected')
  })

  it('manual (non-auto-detected) characters never touch entity resolution', () => {
    const char = makeCharacter({ is_auto_detected: false, detection_status: 'accepted' })
    const book = resolutionBook(char)
    const updated = useStoryBibleStore.getState().updateCharacter(book, char)
    expect(updated.analysis).toBe(book.analysis)
  })
})

describe('highlighting', () => {
  beforeEach(() => {
    useStoryBibleStore.setState({ highlightedCharacterId: null })
  })

  it('returns the highlighted character name plus aliases', () => {
    const book = makeBook({
      story_bible: {
        characters: [makeCharacter({ aliases: ['M.', 'the archivist'] })],
        plot_notes: '',
        timeline: '',
      },
    })
    const store = useStoryBibleStore.getState()
    expect(store.getHighlightedCharacterNames(book)).toEqual([])
    store.setHighlightedCharacter('c1')
    expect(useStoryBibleStore.getState().getHighlightedCharacterNames(book)).toEqual(['Mara', 'M.', 'the archivist'])
    expect(useStoryBibleStore.getState().getHighlightedCharacterNames(null)).toEqual([])
  })
})
