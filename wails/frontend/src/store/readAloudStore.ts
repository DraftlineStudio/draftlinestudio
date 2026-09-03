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
import { collectSentences, type DocSentence } from '../services/readaloud/docSentences'
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

  const clearHighlight = () => withEditorView(clearReadAloud)

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
      const proceed = () => {
        const settings = useAppStore.getState().settings
        set({ sentences, playbackError: null, playerVisible: true, currentIndex: -1 })
        withEditorView(view => updateReadAloud(view, {
          active: true,
          sentences: sentences.map(s => ({ from: s.from, to: s.to })),
          activeIndex: -1,
        }))
        ensureController().start(
          sentences.map(s => s.text),
          Math.max(0, Math.min(startIndex, sentences.length - 1)),
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
