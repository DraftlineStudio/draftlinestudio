// Plot Store - Beats, Foreshadowing, Knowledge Matrix CRUD

import { create } from 'zustand'
import type { BookData, Beat, BeatSheet, ForeshadowingItem, ForeshadowingLedger, SecretInfo, KnowledgeEntry, KnowledgeMatrix } from '../types/draftline'

interface PlotStore {
  // Beat Sheet CRUD (pure functions that return updated book)
  addBeat: (book: BookData, beat: Beat) => BookData
  updateBeat: (book: BookData, beat: Beat) => BookData
  deleteBeat: (book: BookData, id: string) => BookData

  // Foreshadowing CRUD
  addForeshadowingItem: (book: BookData, item: ForeshadowingItem) => BookData
  updateForeshadowingItem: (book: BookData, item: ForeshadowingItem) => BookData
  deleteForeshadowingItem: (book: BookData, id: string) => BookData

  // Knowledge Matrix CRUD
  addSecret: (book: BookData, secret: SecretInfo) => BookData
  updateSecret: (book: BookData, secret: SecretInfo) => BookData
  deleteSecret: (book: BookData, id: string) => BookData
  setKnowledgeEntry: (book: BookData, entry: KnowledgeEntry) => BookData
  removeKnowledgeEntry: (book: BookData, secretId: string, characterId: string) => BookData
}

export const usePlotStore = create<PlotStore>(() => ({
  // Beat Sheet
  addBeat: (book, beat) => {
    const beatSheet: BeatSheet = book.beat_sheet ?? { beats: [] }
    return { ...book, beat_sheet: { ...beatSheet, beats: [...beatSheet.beats, beat] } }
  },

  updateBeat: (book, beat) => {
    const beatSheet: BeatSheet = book.beat_sheet ?? { beats: [] }
    return { ...book, beat_sheet: { ...beatSheet, beats: beatSheet.beats.map(b => b.id === beat.id ? beat : b) } }
  },

  deleteBeat: (book, id) => {
    const beatSheet: BeatSheet = book.beat_sheet ?? { beats: [] }
    return { ...book, beat_sheet: { ...beatSheet, beats: beatSheet.beats.filter(b => b.id !== id) } }
  },

  // Foreshadowing
  addForeshadowingItem: (book, item) => {
    const ledger: ForeshadowingLedger = book.foreshadowing ?? { items: [] }
    return { ...book, foreshadowing: { ...ledger, items: [...ledger.items, item] } }
  },

  updateForeshadowingItem: (book, item) => {
    const ledger: ForeshadowingLedger = book.foreshadowing ?? { items: [] }
    return { ...book, foreshadowing: { ...ledger, items: ledger.items.map(i => i.id === item.id ? item : i) } }
  },

  deleteForeshadowingItem: (book, id) => {
    const ledger: ForeshadowingLedger = book.foreshadowing ?? { items: [] }
    return { ...book, foreshadowing: { ...ledger, items: ledger.items.filter(i => i.id !== id) } }
  },

  // Knowledge Matrix
  addSecret: (book, secret) => {
    const matrix: KnowledgeMatrix = book.knowledge_matrix ?? { secrets: [], entries: [] }
    return { ...book, knowledge_matrix: { ...matrix, secrets: [...matrix.secrets, secret] } }
  },

  updateSecret: (book, secret) => {
    const matrix: KnowledgeMatrix = book.knowledge_matrix ?? { secrets: [], entries: [] }
    return { ...book, knowledge_matrix: { ...matrix, secrets: matrix.secrets.map(s => s.id === secret.id ? secret : s) } }
  },

  deleteSecret: (book, id) => {
    const matrix: KnowledgeMatrix = book.knowledge_matrix ?? { secrets: [], entries: [] }
    return {
      ...book,
      knowledge_matrix: {
        secrets: matrix.secrets.filter(s => s.id !== id),
        entries: matrix.entries.filter(e => e.secret_id !== id)
      }
    }
  },

  setKnowledgeEntry: (book, entry) => {
    const matrix: KnowledgeMatrix = book.knowledge_matrix ?? { secrets: [], entries: [] }
    const existing = matrix.entries.find(e => e.secret_id === entry.secret_id && e.character_id === entry.character_id)
    const newEntries = existing
      ? matrix.entries.map(e => (e.secret_id === entry.secret_id && e.character_id === entry.character_id) ? entry : e)
      : [...matrix.entries, entry]
    return { ...book, knowledge_matrix: { ...matrix, entries: newEntries } }
  },

  removeKnowledgeEntry: (book, secretId, characterId) => {
    const matrix: KnowledgeMatrix = book.knowledge_matrix ?? { secrets: [], entries: [] }
    return {
      ...book,
      knowledge_matrix: {
        ...matrix,
        entries: matrix.entries.filter(e => !(e.secret_id === secretId && e.character_id === characterId))
      }
    }
  },
}))
