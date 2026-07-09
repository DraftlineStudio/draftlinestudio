// Story Bible Store - Characters CRUD, highlighting, merging

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
  deleteAutoDetectedCharacters: (book: BookData) => BookData
  mergeCharacters: (book: BookData, primaryId: string, mergeIds: string[]) => BookData

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

  deleteAutoDetectedCharacters: (book) => {
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    return { ...book, story_bible: { ...bible, characters: bible.characters.filter(c => !c.is_auto_detected) } }
  },

  mergeCharacters: (book, primaryId, mergeIds) => {
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    const primary = bible.characters.find(c => c.id === primaryId)
    if (!primary) return book

    const toMerge = bible.characters.filter(c => mergeIds.includes(c.id))
    if (toMerge.length === 0) return book

    // Combine data from merged characters
    const mergedAliases = new Set<string>(primary.aliases || [])
    let totalMentions = primary.mention_count || 0
    let firstChapter = primary.first_chapter
    const mergedChapterMentions: Record<number, number> = { ...(primary.chapter_mentions || {}) }
    const mergedAttributes: Record<string, string> = { ...(primary.attributes || {}) }
    let notes = primary.notes || ''

    for (const char of toMerge) {
      mergedAliases.add(char.name)
      if (char.aliases) char.aliases.forEach(a => mergedAliases.add(a))
      totalMentions += char.mention_count || 0
      if (char.first_chapter !== undefined) {
        if (firstChapter === undefined || char.first_chapter < firstChapter) {
          firstChapter = char.first_chapter
        }
      }
      if (char.chapter_mentions) {
        for (const [ch, count] of Object.entries(char.chapter_mentions)) {
          const chNum = Number(ch)
          mergedChapterMentions[chNum] = (mergedChapterMentions[chNum] || 0) + count
        }
      }
      if (char.attributes) {
        for (const [key, val] of Object.entries(char.attributes)) {
          if (!mergedAttributes[key]) mergedAttributes[key] = val
        }
      }
      if (char.notes && char.notes.trim()) {
        notes = notes ? `${notes}\n\n[From ${char.name}]: ${char.notes}` : `[From ${char.name}]: ${char.notes}`
      }
    }

    mergedAliases.delete(primary.name)

    const updatedPrimary: Character = {
      ...primary,
      aliases: Array.from(mergedAliases),
      mention_count: totalMentions,
      first_chapter: firstChapter,
      chapter_mentions: mergedChapterMentions,
      attributes: mergedAttributes,
      notes,
    }

    const newCharacters = bible.characters
      .filter(c => !mergeIds.includes(c.id))
      .map(c => c.id === primaryId ? updatedPrimary : c)

    return { ...book, story_bible: { ...bible, characters: newCharacters } }
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
