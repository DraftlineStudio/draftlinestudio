// Read Aloud Store - voice model install state and sentence playback.
// Nothing here touches the model or spawns workers while the plugin is
// disabled: the synthesis worker (and kokoro-js inside it) is created lazily
// on the first play, and shutdown() tears it down again. The editor-facing
// glue (collecting sentences at the cursor, highlighting) lives with the
// ReadAloud extension; this store exposes editor-agnostic playback over
// prepared sentence lists.

import { create } from 'zustand'
import { ReadAloudStatus, DownloadReadAloudModel, DownloadReadAloudGPUModel, DownloadReadAloudNative, CancelReadAloudDownload, RemoveReadAloudModel, VerifyReadAloudModel, ReadAloudServerURL, StartReadAloudMemLog, StopReadAloudMemLog, ShutdownReadAloudNative } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { ReadAloudController, type ControllerStatus, type SynthPort } from '../services/readaloud/controller'
import { NativeSynth, WorkerSynth, type ModelLoadProgress } from '../services/readaloud/tts'
import { WebAudioPort } from '../services/readaloud/audio'
import { collectSentences, buildGenerationUnits, firstUnitOfSentence, type DocSentence, type GenerationUnit } from '../services/readaloud/docSentences'
import { updateReadAloud, clearReadAloud, setReadAloudHandlers } from '../extensions/ReadAloud'
import { useAppStore } from './appStore'
import { useEditorStore } from './editorStore'
import type { Editor } from '@tiptap/react'

export type ReadAloudModelState = 'unknown' | 'missing' | 'downloading' | 'ready' | 'corrupt' | 'error'

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
  nativeSupported: boolean
  nativeInstalled: boolean
  nativeBytesTotal: number
  benchmarking: boolean
  // Full-hash verification state for this session. Playback refuses to load
  // the model until one verify pass has succeeded.
  verified: boolean
  verifying: boolean
  installInfo: { version: string; installedAt: string; bytes: number } | null
  corruptFiles: string[]

  status: ControllerStatus
  sentences: DocSentence[]
  currentIndex: number
  playerVisible: boolean
  playbackError: string | null
  modelLoading: ModelLoadProgress | null
  device: 'native' | 'wasm' | 'webgpu' | null
  // Worker load diagnostics (also mirrored to the WebView console).
  diagnostics: string[]

  refreshModelStatus: () => Promise<void>
  // Full sha256 audit against the pinned manifest; run on plugin enable and
  // before first playback. Returns whether the core bundle verified.
  verifyModel: () => Promise<boolean>
  downloadModel: () => void
  downloadGPUModel: () => void
  downloadNative: () => void
  cancelDownload: () => void
  removeModel: () => Promise<void>
  // Times a sentence on each installed backend and persists the faster
  // device. Only runs while playback is idle.
  runBenchmark: () => Promise<void>

  // Starts reading the given sentences (already mapped to editor positions).
  play: (sentences: DocSentence[], startIndex: number) => void
  // Pre-fills the synthesis buffer from the cursor without playing, so the
  // first press of play is instant. Safe to call repeatedly.
  prepareFromCursor: () => void
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
let editorWatcherBound = false

// Playback machinery, created on first play and torn down by shutdown().
// Module-scoped (not store state) because these are stateful class instances,
// matching the saveChain idiom in appStore.
let synth: (SynthPort & { shutdown(): void }) | null = null
let audio: WebAudioPort | null = null
let controller: ReadAloudController | null = null

// Playback runs on generation UNITS (clause-sized pieces of long sentences)
// while the UI and highlight run on SENTENCES. currentUnits maps between the
// two for the active/prepared queue.
let currentUnits: GenerationUnit[] = []
let currentSentences: DocSentence[] = []

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
    // Go-side memory sampling (process RSS every 2s while playing) joins the
    // same diagnostics stream as the worker's heap/wasm snapshots.
    EventsOn('readaloud:diag', (line: string) => {
      set(s => ({ diagnostics: [...s.diagnostics.slice(-59), line] }))
    })
    EventsOn('readaloud:done', (result: { ok: boolean; error?: string; group?: string }) => {
      set({ download: null, modelError: result?.ok ? null : (result?.error ?? null), verified: false })
      if (result?.ok) {
        // The installer stream-verified every byte it wrote; the audit pass
        // confirms the whole on-disk set and unlocks playback.
        set(s => ({ modelState: s.modelState === 'downloading' ? 'unknown' : s.modelState }))
        void get().verifyModel()
      } else {
        // Cancellation/failure: the status check decides what's on disk.
        set(s => ({ modelState: s.modelState === 'downloading' ? 'unknown' : s.modelState }))
        void get().refreshModelStatus()
      }
    })
  }

  const clearHighlight = () => withEditorView(clearReadAloud)

  const ensureController = (): ReadAloudController => {
    if (controller) return controller
    const config = () => {
      const settings = useAppStore.getState().settings
      return { device: settings.read_aloud_device, threads: settings.read_aloud_threads }
    }
    const diagnostic = (line: string) => set(s => ({ diagnostics: [...s.diagnostics.slice(-59), line] }))
    if (get().nativeInstalled) {
      synth = new NativeSynth(config, () => ReadAloudServerURL(), diagnostic)
      set({ device: 'native', modelLoading: null })
      diagnostic('backend: native sherpa-onnx (Windows/macOS/Linux)')
    } else {
      synth = new WorkerSynth(
        config,
        progress => set({ modelLoading: progress }),
        device => set({ device, modelLoading: null }),
        diagnostic,
        () => ReadAloudServerURL().then(url => (url ? `${url}/readaloud-models` : '')),
      )
    }
    audio = new WebAudioPort(
      line => set(s => ({ diagnostics: [...s.diagnostics.slice(-59), line] })),
    )
    controller = new ReadAloudController(synth, audio, {
      onStatus: status => {
        set({ status })
        if (status === 'playing') set({ modelLoading: null })
        // RSS sampling runs exactly while something is audible/pending.
        if (status === 'playing' || status === 'starting') void StartReadAloudMemLog()
        else void StopReadAloudMemLog()
      },
      onSentenceStart: unitIndex => {
        const sentenceIndex = currentUnits[unitIndex]?.sentenceIndex ?? unitIndex
        // Highlight covers the FULL sentence even while a mid-sentence
        // clause unit is the one actually playing; only move it when the
        // sentence actually changes.
        if (get().currentIndex !== sentenceIndex) {
          set({ currentIndex: sentenceIndex })
          withEditorView(view => updateReadAloud(view, { activeIndex: sentenceIndex }))
          scrollHighlightIntoView()
        }
        set(s => ({ diagnostics: [...s.diagnostics.slice(-59), `[main t+${Math.round(performance.now())}ms] play-start unit ${unitIndex} (sentence ${sentenceIndex + 1})`] }))
      },
      onError: message => {
        set({ playbackError: message, currentIndex: -1 })
        clearHighlight()
      },
      onFinished: () => {
        set({ currentIndex: -1 })
        clearHighlight()
      },
    }, get().nativeInstalled ? { startBufferSeconds: 0, startBufferUnits: 1 } : undefined)
    // A chapter switch remounts the editor; playback positions belong to the
    // old document, so stop cleanly (highlights die with the old view).
    // Subscribed once per app lifetime — re-subscribing on every controller
    // rebuild (enable/disable cycles) would accumulate listeners.
    if (!editorWatcherBound) {
      editorWatcherBound = true
      let lastEditor = useEditorStore.getState().editorRef
      useEditorStore.subscribe(state => {
        if (state.editorRef === lastEditor) return
        lastEditor = state.editorRef
        const store = get()
        if (store.status !== 'idle') store.stop()
      })
    }
    return controller
  }

  return {
    modelState: 'unknown',
    bytesTotal: 0,
    download: null,
    modelError: null,
    gpuInstalled: false,
    gpuBytesTotal: 0,
    nativeSupported: false,
    nativeInstalled: false,
    nativeBytesTotal: 0,
    benchmarking: false,
    verified: false,
    verifying: false,
    installInfo: null,
    corruptFiles: [],

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
          nativeSupported: status.native_supported,
          nativeInstalled: status.native_installed,
          nativeBytesTotal: status.native_bytes_total,
          // An in-flight download keeps its state; events own that
          // transition. A corrupt verdict outranks the cheap size check —
          // only a successful re-verify (or removal) clears it.
          modelState: s.modelState === 'downloading' || (s.modelState === 'corrupt' && (status.installed || status.native_installed))
            ? s.modelState
            : (status.installed || status.native_installed ? 'ready' : 'missing'),
        }))
      } catch (e) {
        set({ modelState: 'error', modelError: String(e) })
      }
    },

    verifyModel: async () => {
      if (get().verifying) return get().verified
      set({ verifying: true })
      const pushDiag = (line: string) => set(s => ({ diagnostics: [...s.diagnostics.slice(-59), line] }))
      try {
        const result = await VerifyReadAloudModel()
        if (result.verified || result.native_verified) {
          set({
            modelState: 'ready', verified: true, corruptFiles: [], modelError: null,
            gpuInstalled: result.gpu_verified,
            nativeInstalled: result.native_verified,
            installInfo: { version: result.version, installedAt: result.installed_at, bytes: result.bytes },
          })
          pushDiag(`verify: ${result.installed_at ? 'v' + result.version + ', ' : ''}${Math.round(result.bytes / (1024 * 1024))} MB, all hashes match`)
          return true
        }
        if (!result.installed && !result.native_verified) {
          set({ modelState: 'missing', verified: false, installInfo: null, corruptFiles: [] })
          pushDiag('verify: model not installed')
          return false
        }
        set({ modelState: 'corrupt', verified: false, corruptFiles: [...(result.corrupt ?? []), ...(result.missing ?? [])] })
        pushDiag(`verify: FAILED — ${(result.corrupt ?? []).length} corrupt, ${(result.missing ?? []).length} missing file(s); repair will re-download only those`)
        return false
      } catch (e) {
        set({ modelState: 'error', modelError: String(e), verified: false })
        return false
      } finally {
        set({ verifying: false })
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

    downloadNative: () => {
      bindEvents()
      set({ modelState: 'downloading', modelError: null })
      try {
        void DownloadReadAloudNative()
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
      // Benchmarks only run against a fully verified install.
      if (!get().verified && !(await get().verifyModel())) return
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
      const pushDiag = (line: string) => set(s => ({ diagnostics: [...s.diagnostics.slice(-59), line] }))
      get().shutdown() // stops playback, unloads the pipeline/worker
      try {
        // Drop any Cache API entries kokoro-js may have written for the
        // voice URLs (defensive: useBrowserCache is off, but the library
        // seeds a 'kokoro-voices' cache on its fetch path).
        try { await caches.delete('kokoro-voices') } catch { /* unsupported/empty */ }
        await RemoveReadAloudModel() // errors if anything is left on disk
        set({ modelState: 'missing', download: null, modelError: null, verified: false, installInfo: null, corruptFiles: [], gpuInstalled: false, nativeInstalled: false })
        pushDiag('remove: model directory deleted and confirmed empty; voice cache cleared; pipeline unloaded')
      } catch (e) {
        set({ modelError: String(e) })
        pushDiag(`remove FAILED: ${e instanceof Error ? e.message : String(e)}`)
      }
    },

    play: (sentences, startIndex) => {
      if (!sentences.length) return
      const start = Math.max(0, Math.min(startIndex, sentences.length - 1))
      const units = buildGenerationUnits(sentences, start)
      const startUnit = Math.max(0, firstUnitOfSentence(units, start))
      const proceed = () => {
        const settings = useAppStore.getState().settings
        set({ sentences, playbackError: null, playerVisible: true, currentIndex: -1 })
        withEditorView(view => updateReadAloud(view, {
          active: true,
          sentences: sentences.map(s => ({ from: s.from, to: s.to })),
          activeIndex: -1,
        }))
        const ctrl = ensureController()
        // If the buffer was pre-filled for exactly this queue while the
        // player sat open, arm it — first audio is already synthesized.
        if (
          ctrl.isPreparedFor(units.length)
          && currentUnits.length === units.length
          && currentUnits.every((u, i) => u.text === units[i].text)
          && ctrl.beginPlayback()
        ) {
          return
        }
        currentUnits = units
        currentSentences = sentences
        ctrl.start(units.map(u => u.text), startUnit, settings.read_aloud_voice, settings.read_aloud_speed)
      }
      if (get().modelState === 'ready' && get().verified) {
        proceed()
        return
      }
      // First playback of the session (or unknown/missing/corrupt state):
      // surface the player and run the full-hash verify; the model only
      // loads after it passes. A corrupt result leaves the repair state
      // showing instead of silently falling back.
      set({ playerVisible: true })
      void get().verifyModel().then(ok => {
        if (ok) proceed()
      })
    },

    prepareFromCursor: () => {
      // Warm the pipeline while the player sits open: build the queue from
      // the cursor and pre-fill the lookahead buffer without playing.
      // Producer-only; play() arms it instantly if the cursor hasn't moved.
      if (get().status !== 'idle' || get().modelState !== 'ready' || !get().verified) return
      const editor = liveEditor()
      if (!editor) return
      const sentences = collectSentences(editor.state.doc, editor.state.selection.from)
      if (!sentences.length) return
      const units = buildGenerationUnits(sentences, 0)
      if (
        currentUnits.length === units.length
        && currentUnits.every((u, i) => u.text === units[i].text)
      ) return // already prepared for this exact queue
      const settings = useAppStore.getState().settings
      currentUnits = units
      currentSentences = sentences
      set(s => ({ diagnostics: [...s.diagnostics.slice(-59), `[main t+${Math.round(performance.now())}ms] prefill: ${units.length} units queued from cursor`] }))
      ensureController().start(units.map(u => u.text), 0, settings.read_aloud_voice, settings.read_aloud_speed, false)
      set({ sentences })
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
      currentUnits = []
      currentSentences = []
      set({ currentIndex: -1, sentences: [] })
      clearHighlight()
    },

    // Skip/jump operate on SENTENCES for the user; playback runs on units.
    skip: (delta) => {
      if (!controller) return
      const target = get().currentIndex + delta
      if (target < 0) {
        controller.jumpTo(0)
        return
      }
      const unit = firstUnitOfSentence(currentUnits, target)
      if (unit === -1) {
        get().stop() // skipped past the end
        return
      }
      controller.jumpTo(unit)
    },

    jumpTo: (sentenceIndex) => {
      if (!controller) return
      const unit = firstUnitOfSentence(currentUnits, sentenceIndex)
      if (unit !== -1) controller.jumpTo(unit)
    },

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
      try { void ShutdownReadAloudNative() } catch { /* browser/test environment */ }
      audio?.close()
      try { void StopReadAloudMemLog() } catch { /* ignore */ }
      controller = null
      synth = null
      audio = null
      currentUnits = []
      currentSentences = []
      clearHighlight()
      set({ status: 'idle', sentences: [], currentIndex: -1, playerVisible: false, modelLoading: null, playbackError: null })
    },
  }
})
