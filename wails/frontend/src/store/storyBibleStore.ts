// Story Bible Store - Characters CRUD and highlighting

import { create } from 'zustand'
import type { Character, StoryBible, BookData } from '../types/draftline'

function applyCharacterDecision(book: BookData, char: Character, status: 'accepted' | 'rejected'): BookData {
  if (!char.is_auto_detected) return book
  const resolution = book.analysis?.entity_resolution
  if (!resolution) return book

  const names = [char.name, ...(char.aliases ?? [])]
    .map(name => name.trim())
    .filter((name, index, all) => name && all.findIndex(other => other.toLowerCase() === name.toLowerCase()) === index)
  const normalized = new Set(names.map(name => name.toLowerCase()))
  const decisions = (resolution.decisions ?? []).filter(decision =>
    !decision.names.some(name => normalized.has(name.trim().toLowerCase())))

  return {
    ...book,
    analysis: {
      ...book.analysis,
      entity_resolution: {
        ...resolution,
        entities: (resolution.entities ?? []).map(entity =>
          entity.id === char.id ? { ...entity, detection_status: status } : entity),
        decisions: [...decisions, { names, status }],
      },
    },
  }
}

interface StoryBibleStore {
  // Character highlighting
  highlightedCharacterId: string | null
  setHighlightedCharacter: (id: string | null) => void

  // Character CRUD (requires book reference)
  addCharacter: (book: BookData, char: Character) => BookData
  updateCharacter: (book: BookData, char: Character) => BookData
  deleteCharacter: (book: BookData, id: string) => BookData

  // Helper to get highlighted character names
  getHighlightedCharacterNames: (book: BookData | null) => string[]
}

export const useStoryBibleStore = create<StoryBibleStore>((set, get) => ({
  highlightedCharacterId: null,

  setHighlightedCharacter: (id) => set({ highlightedCharacterId: id }),

  addCharacter: (book, char) => {
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    return { ...book, story_bible: { ...bible, characters: [...bible.characters, char] } }
  },

  updateCharacter: (book, char) => {
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    const updated = { ...book, story_bible: { ...bible, characters: bible.characters.map(c => c.id === char.id ? char : c) } }
    return char.detection_status === 'accepted' ? applyCharacterDecision(updated, char, 'accepted') : updated
  },

  deleteCharacter: (book, id) => {
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    const char = bible.characters.find(candidate => candidate.id === id)
    const updated = { ...book, story_bible: { ...bible, characters: bible.characters.filter(c => c.id !== id) } }
    return char?.is_auto_detected ? applyCharacterDecision(updated, char, 'rejected') : updated
  },

  getHighlightedCharacterNames: (book) => {
    const { highlightedCharacterId } = get()
    if (!highlightedCharacterId || !book?.story_bible?.characters) return []

    const char = book.story_bible.characters.find(c => c.id === highlightedCharacterId)
    if (!char) return []

    const names = [char.name]
    if (char.aliases) names.push(...char.aliases)
    return names
  },
}))
