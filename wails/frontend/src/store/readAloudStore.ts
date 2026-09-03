// Read Aloud Store - voice model install state and (in later builds) playback.
// Nothing here touches the model or spawns workers while the plugin is
// disabled; the store only talks to the Go download/status methods on demand.

import { create } from 'zustand'
import { ReadAloudStatus, DownloadReadAloudModel, CancelReadAloudDownload, RemoveReadAloudModel } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

export type ReadAloudModelState = 'unknown' | 'missing' | 'downloading' | 'ready' | 'error'

export interface ReadAloudDownloadProgress {
  file: string
  received: number
  total: number
  overall_received: number
  overall_total: number
}

interface ReadAloudStore {
  modelState: ReadAloudModelState
  bytesTotal: number
  download: ReadAloudDownloadProgress | null
  modelError: string | null

  refreshModelStatus: () => Promise<void>
  downloadModel: () => void
  cancelDownload: () => void
  removeModel: () => Promise<void>
}

// The backend emits download events globally; bind exactly once, lazily, so
// merely importing this module never touches the Wails runtime (vitest runs
// with the bindings mocked).
let eventsBound = false

export const useReadAloudStore = create<ReadAloudStore>((set, get) => {
  const bindEvents = () => {
    if (eventsBound) return
    eventsBound = true
    EventsOn('readaloud:progress', (p: ReadAloudDownloadProgress) => {
      set({ modelState: 'downloading', download: p })
    })
    EventsOn('readaloud:done', (result: { ok: boolean; error?: string }) => {
      set({ download: null })
      if (result?.ok) {
        set({ modelState: 'ready', modelError: null })
      } else {
        // Cancellation also lands here; refresh decides between missing/ready.
        set({ modelError: result?.error ?? null })
        void get().refreshModelStatus()
      }
    })
  }

  return {
    modelState: 'unknown',
    bytesTotal: 0,
    download: null,
    modelError: null,

    refreshModelStatus: async () => {
      bindEvents()
      try {
        const status = await ReadAloudStatus()
        set(s => ({
          bytesTotal: status.bytes_total,
          // An in-flight download keeps its state; events own that transition.
          modelState: s.modelState === 'downloading' ? s.modelState : (status.installed ? 'ready' : 'missing'),
        }))
      } catch (e) {
        set({ modelState: 'error', modelError: String(e) })
      }
    },

    downloadModel: () => {
      bindEvents()
      set({ modelState: 'downloading', modelError: null })
      try {
        void DownloadReadAloudModel()
      } catch (e) {
        set({ modelState: 'error', modelError: String(e) })
      }
    },

    cancelDownload: () => {
      try {
        void CancelReadAloudDownload()
      } catch { /* ignore */ }
    },

    removeModel: async () => {
      try {
        await RemoveReadAloudModel()
        set({ modelState: 'missing', download: null, modelError: null })
      } catch (e) {
        set({ modelError: String(e) })
      }
    },
  }
})
