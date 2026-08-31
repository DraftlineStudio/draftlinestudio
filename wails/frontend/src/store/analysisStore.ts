import { create } from 'zustand'
import { AnalyzeBook } from '../../wailsjs/go/main/App'
import { useBookStore } from './bookStore'
import { useAppStore } from './appStore'
import type { BookData } from '../types/draftline'

export type AnalysisState = 'idle' | 'stale' | 'running' | 'current' | 'error'
export type AnalysisModule = 'characters' | 'story' | 'pacing'

export interface AnalysisProgressEvent {
  phase: 'characters' | 'relationships' | 'story' | 'complete' | string
  message: string
  chapter_index?: number
  chapter_title?: string
  current: number
  total: number
  percent: number
}

interface AnalysisStore {
  state: AnalysisState
  progress: number
  message: string
  modules: Record<AnalysisModule, AnalysisState>
  error: string
  markStale: () => void
  clear: () => void
  receiveProgress: (event: AnalysisProgressEvent) => void
  run: () => Promise<void>
}

const idleModules = (): Record<AnalysisModule, AnalysisState> => ({
  characters: 'idle', story: 'idle', pacing: 'idle',
})

export const useAnalysisStore = create<AnalysisStore>((set, get) => ({
  state: 'idle',
  progress: 0,
  message: '',
  modules: idleModules(),
  error: '',

  markStale: () => set(current => {
    if (current.state === 'running') return current
    return {
      state: 'stale',
      progress: 0,
      message: 'Story analysis is out of date',
      error: '',
      modules: { characters: 'stale', story: 'stale', pacing: 'stale' },
    }
  }),

  clear: () => set({ state: 'idle', progress: 0, message: '', modules: idleModules(), error: '' }),

  receiveProgress: event => set(current => {
    const modules = { ...current.modules }
    if (event.phase === 'characters') modules.characters = 'running'
    if (event.phase === 'relationships') modules.characters = 'current'
    if (event.phase === 'story') {
      modules.characters = 'current'
      modules.story = 'running'
      modules.pacing = 'running'
    }
    if (event.phase === 'complete') {
      modules.characters = 'current'
      modules.story = 'current'
      modules.pacing = 'current'
    }
    return {
      state: event.phase === 'complete' ? 'current' : 'running',
      progress: Math.max(0, Math.min(100, event.percent || 0)),
      message: event.message,
      modules,
    }
  }),

  run: async () => {
    if (get().state === 'running') return
    if (!useAppStore.getState().settings.analysis_enabled) return
    const before = useBookStore.getState()
    if (!before.book) return
    const revision = before.analysisRevision
    const bookIdentity = `${before.book.file_path || ''}\u0000${before.book.metadata.created || ''}`
    set({
      state: 'running', progress: 1, message: 'Preparing manuscript analysis', error: '',
      modules: { characters: 'running', story: 'stale', pacing: 'stale' },
    })
    try {
      const result = await AnalyzeBook(before.book as any)
      const after = useBookStore.getState()
      const currentIdentity = after.book ? `${after.book.file_path || ''}\u0000${after.book.metadata.created || ''}` : ''
      if (after.analysisRevision !== revision || currentIdentity !== bookIdentity) {
        set({
          state: 'stale', progress: 0, message: 'Story changed during analysis; waiting to run again',
          modules: { characters: 'stale', story: 'stale', pacing: 'stale' },
        })
        const retryRevision = after.analysisRevision
        setTimeout(() => {
          const latest = useBookStore.getState()
          if (get().state === 'stale' && latest.analysisRevision === retryRevision) void get().run()
        }, 15_000)
        return
      }
      if (!result.success || !result.book) {
        throw new Error(result.error || 'Analysis failed')
      }
      after.updateBook(result.book as unknown as BookData)
      set({
        state: 'current', progress: 100, message: 'Story analysis current', error: '',
        modules: { characters: 'current', story: 'current', pacing: 'current' },
      })
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      set({
        state: 'error', progress: 0, message: 'Story analysis failed', error: message,
        modules: { characters: 'error', story: 'error', pacing: 'error' },
      })
    }
  },
}))
