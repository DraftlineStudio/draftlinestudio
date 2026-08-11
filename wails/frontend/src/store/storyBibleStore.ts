// Story Bible Store - Characters CRUD and highlighting

import { create } from 'zustand'
import type { Character, StoryBible, BookData } from '../types/draftline'

interface StoryBibleStore {
  // Character highlighting
  highlightedCharacterId: string | null
  setHighlightedCharacter: (id: string | null) => void

  // Character CRUD (requires book reference)
  addCharacter: (book: BookData, char: Character) => BookData
  updateCharacter: (book: BookData, char: Character) => BookData
  deleteCharacter: (book: BookData, id: string) => BookData
  // Story Bible text
  updateStoryBibleText: (book: BookData, field: 'plot_notes' | 'timeline', text: string) => BookData

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
    return { ...book, story_bible: { ...bible, characters: bible.characters.map(c => c.id === char.id ? char : c) } }
  },

  deleteCharacter: (book, id) => {
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    return { ...book, story_bible: { ...bible, characters: bible.characters.filter(c => c.id !== id) } }
  },

  updateStoryBibleText: (book, field, text) => {
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    return { ...book, story_bible: { ...bible, [field]: text } }
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
