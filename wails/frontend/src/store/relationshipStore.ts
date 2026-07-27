// Relationship Store - runs relationship analysis for the Story Web

import { create } from 'zustand'
import type { BookData } from '../types/draftline'
import { AnalyzeRelationships } from '../../wailsjs/go/main/App'

interface RelationshipStore {
  isAnalyzing: boolean
  analyzeRelationships: (book: BookData) => Promise<BookData | null>
}

export const useRelationshipStore = create<RelationshipStore>((set) => ({
  isAnalyzing: false,

  analyzeRelationships: async (book) => {
    set({ isAnalyzing: true })
    try {
      const result = await AnalyzeRelationships(book as any)
      if (result.success && result.book) {
        return result.book as unknown as BookData
      }
      console.error('Relationship analysis failed:', result.error)
      return null
    } catch (err) {
      console.error('Relationship analysis error:', err)
      return null
    } finally {
      set({ isAnalyzing: false })
    }
  },
}))
