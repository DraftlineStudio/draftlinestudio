// Read Aloud Store - voice model install state and sentence playback.
// Nothing here touches the model or spawns workers while the plugin is
// disabled: the synthesis worker (and kokoro-js inside it) is created lazily
// on the first play, and shutdown() tears it down again. The editor-facing
// glue (collecting sentences at the cursor, highlighting) lives with the
// ReadAloud extension; this store exposes editor-agnostic playback over
// prepared sentence lists.

import { create } from 'zustand'
import { ReadAloudStatus, DownloadReadAloudModel, CancelReadAloudDownload, RemoveReadAloudModel } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { ReadAloudController, type ControllerStatus } from '../services/readaloud/controller'
import { WorkerSynth, type ModelLoadProgress } from '../services/readaloud/tts'
import { WebAudioPort } from '../services/readaloud/audio'
import type { DocSentence } from '../services/readaloud/docSentences'
import { useAppStore } from './appStore'

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

  status: ControllerStatus
  sentences: DocSentence[]
  currentIndex: number
  playerVisible: boolean
  playbackError: string | null
  modelLoading: ModelLoadProgress | null
  device: 'wasm' | 'webgpu' | null

  refreshModelStatus: () => Promise<void>
  downloadModel: () => void
  cancelDownload: () => void
  removeModel: () => Promise<void>

  // Starts reading the given sentences (already mapped to editor positions).
  play: (sentences: DocSentence[], startIndex: number) => void
  togglePause: () => void
  stop: () => void
  skip: (delta: 1 | -1) => void
  jumpTo: (index: number) => void
  setVoice: (voice: string) => void
  setSpeed: (speed: number) => void
  setPlayerVisible: (visible: boolean) => void
  // Full teardown: stops playback and unloads the model/worker (plugin
  // disable). The next play starts a fresh worker.
  shutdown: () => void
}

// The backend emits download events globally; bind exactly once, lazily, so
// merely importing this module never touches the Wails runtime (vitest runs
// with the bindings mocked).
let eventsBound = false

// Playback machinery, created on first play and torn down by shutdown().
// Module-scoped (not store state) because these are stateful class instances,
// matching the saveChain idiom in appStore.
let synth: WorkerSynth | null = null
let audio: WebAudioPort | null = null
let controller: ReadAloudController | null = null

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

  const ensureController = (): ReadAloudController => {
    if (controller) return controller
    synth = new WorkerSynth(
      progress => set({ modelLoading: progress }),
      device => set({ device, modelLoading: null }),
    )
    audio = new WebAudioPort()
    controller = new ReadAloudController(synth, audio, {
      onStatus: status => {
        set({ status })
        if (status === 'playing') set({ modelLoading: null })
      },
      onSentenceStart: index => set({ currentIndex: index }),
      onError: message => set({ playbackError: message, currentIndex: -1 }),
      onFinished: () => set({ currentIndex: -1 }),
    })
    return controller
  }

  return {
    modelState: 'unknown',
    bytesTotal: 0,
    download: null,
    modelError: null,

    status: 'idle',
    sentences: [],
    currentIndex: -1,
    playerVisible: false,
    playbackError: null,
    modelLoading: null,
    device: null,

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
      get().shutdown()
      try {
        await RemoveReadAloudModel()
        set({ modelState: 'missing', download: null, modelError: null })
      } catch (e) {
        set({ modelError: String(e) })
      }
    },

    play: (sentences, startIndex) => {
      if (!sentences.length) return
      const settings = useAppStore.getState().settings
      set({ sentences, playbackError: null, playerVisible: true, currentIndex: -1 })
      ensureController().start(
        sentences.map(s => s.text),
        Math.max(0, Math.min(startIndex, sentences.length - 1)),
        settings.read_aloud_voice,
        settings.read_aloud_speed,
      )
    },

    togglePause: () => {
      const status = get().status
      if (status === 'playing' || status === 'starting') controller?.pause()
      else if (status === 'paused') controller?.resume()
    },

    stop: () => {
      controller?.stop()
      set({ currentIndex: -1, sentences: [] })
    },

    skip: (delta) => controller?.skip(delta),

    jumpTo: (index) => controller?.jumpTo(index),

    setVoice: (voice) => {
      void useAppStore.getState().saveSettings({ read_aloud_voice: voice })
      controller?.setVoice(voice)
    },

    setSpeed: (speed) => {
      const clamped = Math.min(1.6, Math.max(0.8, speed))
      void useAppStore.getState().saveSettings({ read_aloud_speed: clamped })
      controller?.setSpeed(clamped)
    },

    setPlayerVisible: (visible) => {
      if (!visible) get().stop()
      set({ playerVisible: visible })
    },

    shutdown: () => {
      controller?.stop()
      synth?.shutdown()
      audio?.close()
      controller = null
      synth = null
      audio = null
      set({ status: 'idle', sentences: [], currentIndex: -1, playerVisible: false, modelLoading: null, playbackError: null })
    },
  }
})
