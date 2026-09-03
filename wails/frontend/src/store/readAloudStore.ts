// Read Aloud Store - voice model install state and sentence playback.
// Nothing here touches the model or spawns workers while the plugin is
// disabled: the synthesis worker (and kokoro-js inside it) is created lazily
// on the first play, and shutdown() tears it down again. The editor-facing
// glue (collecting sentences at the cursor, highlighting) lives with the
// ReadAloud extension; this store exposes editor-agnostic playback over
// prepared sentence lists.

import { create } from 'zustand'
import { ReadAloudStatus, DownloadReadAloudModel, DownloadReadAloudGPUModel, CancelReadAloudDownload, RemoveReadAloudModel } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { ReadAloudController, type ControllerStatus } from '../services/readaloud/controller'
import { WorkerSynth, type ModelLoadProgress } from '../services/readaloud/tts'
import { WebAudioPort } from '../services/readaloud/audio'
import { collectSentences, splitLeadClause, type DocSentence } from '../services/readaloud/docSentences'
import { updateReadAloud, clearReadAloud, setReadAloudHandlers } from '../extensions/ReadAloud'
import { useAppStore } from './appStore'
import { useEditorStore } from './editorStore'
import type { Editor } from '@tiptap/react'

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
  gpuInstalled: boolean
  gpuBytesTotal: number
  benchmarking: boolean

  status: ControllerStatus
  sentences: DocSentence[]
  currentIndex: number
  playerVisible: boolean
  playbackError: string | null
  modelLoading: ModelLoadProgress | null
  device: 'wasm' | 'webgpu' | null
  // Worker load diagnostics (also mirrored to the WebView console).
  diagnostics: string[]

  refreshModelStatus: () => Promise<void>
  downloadModel: () => void
  downloadGPUModel: () => void
  cancelDownload: () => void
  removeModel: () => Promise<void>
  // Times a sentence on each installed backend and persists the faster
  // device. Only runs while playback is idle.
  runBenchmark: () => Promise<void>

  // Starts reading the given sentences (already mapped to editor positions).
  play: (sentences: DocSentence[], startIndex: number) => void
  // Editor-scoped entry points: cursor → chapter end, current selection
  // (falls back to cursor when empty), or the whole chapter.
  playFromCursor: () => void
  playSelection: () => void
  playChapter: () => void
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

// The editorStore holds the full TipTap Editor behind a narrow interface;
// the cast recovers view access for decorations (see editorStore comment).
function liveEditor(): Editor | null {
  return (useEditorStore.getState().editorRef as unknown as Editor | null) ?? null
}

function withEditorView(fn: (view: Editor['view']) => void): void {
  const editor = liveEditor()
  if (editor?.view) fn(editor.view)
}

function scrollHighlightIntoView(): void {
  window.requestAnimationFrame(() => {
    document.querySelector('.read-aloud-current')?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  })
}

// Click-to-jump and edit-stops-playback callbacks for the ReadAloud
// extension. Module-level, like the extension's own handler slot.
setReadAloudHandlers({
  onSentenceClick: index => useReadAloudStore.getState().jumpTo(index),
  onDocEdited: () => {
    const store = useReadAloudStore.getState()
    if (store.status !== 'idle') store.stop()
  },
})

export const useReadAloudStore = create<ReadAloudStore>((set, get) => {
  const bindEvents = () => {
    if (eventsBound) return
    eventsBound = true
    EventsOn('readaloud:progress', (p: ReadAloudDownloadProgress) => {
      set({ modelState: 'downloading', download: p })
    })
    EventsOn('readaloud:done', (result: { ok: boolean; error?: string; group?: string }) => {
      set({ download: null, modelError: result?.ok ? null : (result?.error ?? null) })
      // Covers both groups (core and GPU) and cancellation alike: the
      // status check decides what is actually on disk now.
      void get().refreshModelStatus()
    })
  }

  const clearHighlight = () => withEditorView(clearReadAloud)

  const ensureController = (): ReadAloudController => {
    if (controller) return controller
    synth = new WorkerSynth(
      () => {
        const settings = useAppStore.getState().settings
        return { device: settings.read_aloud_device, threads: settings.read_aloud_threads }
      },
      progress => set({ modelLoading: progress }),
      device => set({ device, modelLoading: null }),
      line => set(s => ({ diagnostics: [...s.diagnostics.slice(-59), line] })),
    )
    audio = new WebAudioPort()
    controller = new ReadAloudController(synth, audio, {
      onStatus: status => {
        set({ status })
        if (status === 'playing') set({ modelLoading: null })
      },
      onSentenceStart: index => {
        set({ currentIndex: index })
        withEditorView(view => updateReadAloud(view, { activeIndex: index }))
        scrollHighlightIntoView()
      },
      onError: message => {
        set({ playbackError: message, currentIndex: -1 })
        clearHighlight()
      },
      onFinished: () => {
        set({ currentIndex: -1 })
        clearHighlight()
      },
    })
    // A chapter switch remounts the editor; playback positions belong to the
    // old document, so stop cleanly (highlights die with the old view).
    let lastEditor = useEditorStore.getState().editorRef
    useEditorStore.subscribe(state => {
      if (state.editorRef === lastEditor) return
      lastEditor = state.editorRef
      const store = get()
      if (store.status !== 'idle') store.stop()
    })
    return controller
  }

  return {
    modelState: 'unknown',
    bytesTotal: 0,
    download: null,
    modelError: null,
    gpuInstalled: false,
    gpuBytesTotal: 0,
    benchmarking: false,

    status: 'idle',
    sentences: [],
    currentIndex: -1,
    playerVisible: false,
    playbackError: null,
    modelLoading: null,
    device: null,
    diagnostics: [],

    refreshModelStatus: async () => {
      bindEvents()
      try {
        const status = await ReadAloudStatus()
        set(s => ({
          bytesTotal: status.bytes_total,
          gpuInstalled: status.gpu_installed,
          gpuBytesTotal: status.gpu_bytes_total,
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

    downloadGPUModel: () => {
      bindEvents()
      set({ modelState: 'downloading', modelError: null })
      try {
        void DownloadReadAloudGPUModel()
      } catch (e) {
        set({ modelState: 'error', modelError: String(e) })
      }
    },

    cancelDownload: () => {
      try {
        void CancelReadAloudDownload()
      } catch { /* ignore */ }
    },

    runBenchmark: async () => {
      if (get().benchmarking || get().status !== 'idle') return
      await get().refreshModelStatus()
      if (get().modelState !== 'ready') return
      set({ benchmarking: true })
      const pushDiag = (line: string) => set(s => ({ diagnostics: [...s.diagnostics.slice(-59), line] }))
      const settings = useAppStore.getState().settings
      const benchText = 'The quick brown fox jumps over the lazy dog while the church bells ring out across the quiet harbor town.'

      // Loads a fresh worker for the device, warms it with one sentence
      // (absorbing model load), then times a steady-state sentence.
      const timeDevice = async (
        device: 'wasm' | 'webgpu',
        threads: 'single' | 'auto',
      ): Promise<{ ms: number; resolved: string } | null> => {
        let resolved = ''
        const bench = new WorkerSynth(
          () => ({ device, threads }),
          undefined,
          d => { resolved = d },
          pushDiag,
        )
        try {
          await bench.synthesize(1, benchText, settings.read_aloud_voice, 1.0)
          const started = performance.now()
          await bench.synthesize(2, benchText, settings.read_aloud_voice, 1.0)
          return { ms: performance.now() - started, resolved }
        } catch (e) {
          pushDiag(`benchmark ${device} (threads ${threads}) failed: ${e instanceof Error ? e.message : String(e)}`)
          return null
        } finally {
          bench.shutdown()
        }
      }

      try {
        pushDiag(`benchmark: timing CPU (wasm q8, threads ${settings.read_aloud_threads})…`)
        let wasmResult = await timeDevice('wasm', settings.read_aloud_threads)
        // A threading failure must not condemn the whole wasm backend:
        // retry single-threaded before declaring it broken.
        if (!wasmResult && settings.read_aloud_threads !== 'single') {
          pushDiag('benchmark: retrying CPU single-threaded…')
          wasmResult = await timeDevice('wasm', 'single')
        }
        let gpuResult: { ms: number; resolved: string } | null = null
        if (get().gpuInstalled) {
          pushDiag('benchmark: timing GPU (webgpu fp32)…')
          gpuResult = await timeDevice('webgpu', settings.read_aloud_threads)
          if (gpuResult && gpuResult.resolved !== 'webgpu') {
            pushDiag('benchmark: webgpu fell back to wasm — GPU not usable on this machine')
            gpuResult = null
          }
        } else {
          pushDiag('benchmark: GPU model not installed — skipping webgpu')
        }
        const winner: 'wasm' | 'webgpu' = wasmResult && gpuResult && gpuResult.ms < wasmResult.ms ? 'webgpu' : 'wasm'
        pushDiag(`benchmark: wasm ${wasmResult ? Math.round(wasmResult.ms) + ' ms' : 'failed'}, webgpu ${gpuResult ? Math.round(gpuResult.ms) + ' ms' : 'n/a'} — keeping ${winner === 'webgpu' ? 'GPU (fp32)' : 'CPU (wasm q8)'}`)
        void useAppStore.getState().saveSettings({ read_aloud_device: winner })
      } finally {
        set({ benchmarking: false })
      }
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
      const start = Math.max(0, Math.min(startIndex, sentences.length - 1))
      // Fast start: the first spoken sentence begins at its first clause.
      const queue = [
        ...sentences.slice(0, start),
        ...splitLeadClause(sentences[start]),
        ...sentences.slice(start + 1),
      ]
      const proceed = () => {
        const settings = useAppStore.getState().settings
        set({ sentences: queue, playbackError: null, playerVisible: true, currentIndex: -1 })
        withEditorView(view => updateReadAloud(view, {
          active: true,
          sentences: queue.map(s => ({ from: s.from, to: s.to })),
          activeIndex: -1,
        }))
        ensureController().start(
          queue.map(s => s.text),
          start,
          settings.read_aloud_voice,
          settings.read_aloud_speed,
        )
      }
      if (get().modelState === 'ready') {
        proceed()
        return
      }
      // Model not (known to be) installed: surface the player, which offers
      // the download; start only if the check comes back installed.
      set({ playerVisible: true })
      void get().refreshModelStatus().then(() => {
        if (get().modelState === 'ready') proceed()
      })
    },

    playFromCursor: () => {
      const editor = liveEditor()
      if (!editor) return
      get().play(collectSentences(editor.state.doc, editor.state.selection.from), 0)
    },

    playSelection: () => {
      const editor = liveEditor()
      if (!editor) return
      const { from, to, empty } = editor.state.selection
      if (empty) {
        get().playFromCursor()
        return
      }
      get().play(collectSentences(editor.state.doc, from, to), 0)
    },

    playChapter: () => {
      const editor = liveEditor()
      if (!editor) return
      get().play(collectSentences(editor.state.doc), 0)
    },

    togglePause: () => {
      const status = get().status
      if (status === 'playing' || status === 'starting') controller?.pause()
      else if (status === 'paused') controller?.resume()
    },

    stop: () => {
      controller?.stop()
      set({ currentIndex: -1, sentences: [] })
      clearHighlight()
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
      clearHighlight()
      set({ status: 'idle', sentences: [], currentIndex: -1, playerVisible: false, modelLoading: null, playbackError: null })
    },
  }
})
