// Read Aloud Store - voice model install state and sentence playback.
// Nothing here loads the native model while the plugin is disabled. Native
// sessions are created lazily on first preparation/play and released by
// shutdown(). The editor-facing
// glue (collecting sentences at the cursor, highlighting) lives with the
// ReadAloud extension; this store exposes editor-agnostic playback over
// prepared sentence lists.

import { create } from 'zustand'
import { ReadAloudStatus, DownloadReadAloudNative, CancelReadAloudDownload, RemoveReadAloudModel, VerifyReadAloudModel, ReadAloudServerURL, StartReadAloudMemLog, StopReadAloudMemLog, ShutdownReadAloudNative } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { ReadAloudController, type AudioChunk, type ControllerStatus, type PlaybackHandle, type SynthPort } from '../services/readaloud/controller'
import { NativeSynth } from '../services/readaloud/tts'
import { WebAudioPort } from '../services/readaloud/audio'
import { collectSentences, buildGenerationUnits, buildCastGenerationUnits, firstUnitOfSentence, type DocSentence, type GenerationUnit } from '../services/readaloud/docSentences'
import { DurationEstimator } from '../services/readaloud/estimates'
import { clampReadAloudSpeed, nextReadAloudSpeed } from '../services/readaloud/speeds'
import { attributeSpeakers, type AttributedSentence, type AttributionResult, type RosterEntry, type SpeakerKey } from '../services/readaloud/attribution'
import { autoCast, buildChapterCast, buildRoster, castVoiceKey, type ChapterSpeaker } from '../services/readaloud/cast'
import { useBookStore } from './bookStore'
import { updateSpotlight, clearSpotlight, setSpotlightHandlers } from '../extensions/RangeSpotlight'
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
  nativeSupported: boolean
  nativeInstalled: boolean
  nativeBytesTotal: number
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
  modelLoading: null
  device: 'native' | null
  // Worker load diagnostics (also mirrored to the WebView console).
  diagnostics: string[]
  // Player UI session state: the expanded panel, session-only mute, and the
  // 4 Hz progress snapshot (null while idle).
  expanded: boolean
  muted: boolean
  progress: { fraction: number; elapsedSec: number; remainingSec: number } | null
  // Voice cast: mirrors the book's cast_mode, the current chapter's speaker
  // rows, per-speaker dialogue ticks for the seek bar (empty when cast mode
  // is off), and the speaker of the sentence being read (null = Narration).
  castMode: boolean
  speakers: ChapterSpeaker[]
  ticks: Array<{ fraction: number; color: string }>
  currentSpeaker: { name: string; color: string } | null
  dialogueLineCount: number
  // Voice being previewed in the cast panel (drives the row EQ animation).
  previewingVoice: string | null

  refreshModelStatus: () => Promise<void>
  // Full sha256 audit against the pinned manifest; run on plugin enable and
  // before first playback. Returns whether the core bundle verified.
  verifyModel: () => Promise<boolean>
  downloadModel: () => void
  cancelDownload: () => void
  removeModel: () => Promise<void>
  // Times a sentence on each installed backend and persists the faster
  // device. Only runs while playback is idle.

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
  // Seek anywhere in the collected queue by progress-bar fraction; restarts
  // idle playback at the target sentence.
  seekToFraction: (fraction: number) => void
  setVoice: (voice: string) => void
  setSpeed: (speed: number) => void
  cycleSpeed: () => void
  // applyVolume is the live drag path (audio only); setVolume persists.
  applyVolume: (volume: number) => void
  setVolume: (volume: number) => void
  toggleMute: () => void
  setExpanded: (expanded: boolean) => void
  // Voice cast actions. All persist through the book (read_aloud_cast.json)
  // and re-voice not-yet-synthesized playback immediately.
  setCastMode: (on: boolean) => void
  setSpeakerVoice: (key: SpeakerKey, voiceId: string) => void
  autoCastVoices: () => void
  // Speaks a short sample in the given voice, outside the playback schedule.
  // Available only while nothing is audibly playing (the synthesis queue is
  // serialized); clicking the active preview again cancels it.
  previewVoice: (voiceId: string) => void
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

// Progress accounting for the active queue. Measured durations arrive via
// onUnitAudio; unmeasured units are estimated from the calibrating
// seconds-per-character rate. All module-scope, like the pipeline singletons.
const estimator = new DurationEstimator()
let unitDurations: (number | null)[] = []
let playingUnit = -1
let playingUnitDuration = 0
let playingUnitStartClock = 0
let progressTimer: ReturnType<typeof setInterval> | null = null

function resetProgressTracking(unitCount: number): void {
  unitDurations = new Array(unitCount).fill(null)
  playingUnit = -1
  playingUnitDuration = 0
  playingUnitStartClock = 0
}

// Voice preview: a fixed sample phrase synthesized through the SAME
// serialized NativeSynth (eSpeak phonemization has process-global state —
// never a second session) and played as a one-shot outside the gapless
// schedule. Request ids live in their own range so they can never collide
// with the controller's counter.
const PREVIEW_TEXT = 'The evening settled softly over the harbor, and the city lights came on.'
const previewChunks = new Map<string, AudioChunk>() // `${voice}@${speed}` → chunk
let previewRequestId = 1 << 30
let previewHandle: PlaybackHandle | null = null

function stopPreview(): void {
  previewHandle?.stop()
  previewHandle = null
  if (useReadAloudStore.getState().previewingVoice !== null) {
    useReadAloudStore.setState({ previewingVoice: null })
  }
}

// ── Voice cast machinery ─────────────────────────────────────────────────────
// Attribution runs over the FULL chapter once per document version and is
// joined to whatever sub-queue plays (cursor/selection/chapter) by
// DocSentence.from — the segmenter keeps true positions under range slicing,
// so the two always agree on boundaries.

let attributionDoc: unknown = null
let attribution: AttributionResult | null = null
let attributionRoster: RosterEntry[] = []
let sentenceSpeakerByFrom = new Map<number, AttributedSentence>()

function combinedChapterIndex(): number {
  const bookState = useBookStore.getState()
  const book = bookState.book
  if (!book) return 0
  const front = book.front_matter?.length ?? 0
  const body = book.body?.length ?? 0
  if (bookState.currentSection === 'front_matter') return bookState.currentIndex
  if (bookState.currentSection === 'body') return front + bookState.currentIndex
  return front + body + bookState.currentIndex
}

// Recomputes speaker attribution for the current chapter when the document
// changed; a few ms of pure string work, memoized on the ProseMirror doc.
function ensureAttribution(): void {
  const editor = liveEditor()
  const book = useBookStore.getState().book
  if (!editor || !book) {
    attributionDoc = null
    attribution = null
    attributionRoster = []
    sentenceSpeakerByFrom = new Map()
    return
  }
  const doc = editor.state.doc
  if (attributionDoc === doc && attribution) return
  const chapterSentences = collectSentences(doc)
  const chapterText = chapterSentences.map(s => s.text).join('\n')
  attributionRoster = buildRoster(book, combinedChapterIndex(), chapterText)
  attribution = attributeSpeakers(chapterSentences, attributionRoster)
  attributionDoc = doc
  sentenceSpeakerByFrom = new Map(attribution.sentences.map(s => [s.from, s]))
}

// Rebuilds the derived cast state (speaker rows, seek-bar ticks, dialogue
// count) for the active queue. Ticks follow the mock: hidden entirely while
// cast mode is off.
function refreshCastState(): void {
  const book = useBookStore.getState().book
  const settings = useAppStore.getState().settings
  const state = useReadAloudStore.getState()
  if (!attribution) {
    useReadAloudStore.setState({ castMode: false, speakers: [], ticks: [], dialogueLineCount: 0, currentSpeaker: null })
    return
  }
  const castMode = book?.read_aloud_cast?.cast_mode ?? false
  const speakers = buildChapterCast(attribution, attributionRoster, book?.read_aloud_cast, settings.read_aloud_voice)
  const queue = state.sentences
  const ticks: Array<{ fraction: number; color: string }> = []
  let dialogueLineCount = 0
  if (queue.length) {
    const colorFor = new Map(speakers.map(s => [s.key, s.color]))
    for (let i = 0; i < queue.length; i++) {
      const attributed = sentenceSpeakerByFrom.get(queue[i].from)
      if (!attributed || attributed.kind === 'narration') continue
      dialogueLineCount++
      if (!castMode) continue
      ticks.push({
        fraction: queue.length > 1 ? i / (queue.length - 1) : 0,
        color: colorFor.get(attributed.speaker) || 'var(--text-muted)',
      })
    }
  }
  useReadAloudStore.setState({ castMode, speakers, ticks: castMode ? ticks : [], dialogueLineCount })
  refreshCurrentSpeaker()
}

// The speaker shown in the EQ/label/chip. Cast mode off forces the display
// to Narration even though attribution is retained (mock behavior).
function refreshCurrentSpeaker(): void {
  const state = useReadAloudStore.getState()
  let next: { name: string; color: string } | null = null
  if (state.castMode && state.currentIndex >= 0) {
    const sentence = state.sentences[state.currentIndex]
    const attributed = sentence ? sentenceSpeakerByFrom.get(sentence.from) : undefined
    if (attributed && attributed.kind !== 'narration') {
      const speaker = state.speakers.find(s => s.key === attributed.speaker)
      if (speaker && speaker.key !== 'narrator') next = { name: speaker.name, color: speaker.color }
    }
  }
  const prev = state.currentSpeaker
  if (prev?.name !== next?.name || prev?.color !== next?.color) {
    useReadAloudStore.setState({ currentSpeaker: next })
  }
}

// Per-unit synthesis voices for the queue: character voice when cast mode is
// on and the sentence's speaker has one; null falls back to the narrator.
// Units flagged as unquoted (tags/asides from cast splitting) always stay
// with the narrator, whoever the sentence's speaker is.
function buildVoiceOverrides(units: GenerationUnit[], sentences: DocSentence[]): (string | null)[] | null {
  const state = useReadAloudStore.getState()
  if (!state.castMode) return null
  const voiceFor = new Map(state.speakers.map(s => [s.key, s.voice]))
  let any = false
  const overrides = units.map(unit => {
    if (unit.quoted === false) return null
    const sentence = sentences[unit.sentenceIndex]
    const attributed = sentence ? sentenceSpeakerByFrom.get(sentence.from) : undefined
    if (!attributed || attributed.speaker === 'narrator' || attributed.speaker === 'unknown') return null
    const voice = voiceFor.get(attributed.speaker) ?? null
    if (voice) any = true
    return voice
  })
  return any ? overrides : null
}

// Quote ranges for the cast unit splitter, keyed by sentence position.
function quotedRangesByFrom(): Map<number, Array<[number, number]>> {
  const map = new Map<number, Array<[number, number]>>()
  for (const [from, attributed] of sentenceSpeakerByFrom) {
    if (attributed.quotedRanges.length) map.set(from, attributed.quotedRanges)
  }
  return map
}

// Builds the synthesis queue for a sentence list: quote-boundary split units
// in cast mode (speech in character voices, asides with the narrator),
// whole-sentence units otherwise.
function buildQueueUnits(sentences: DocSentence[], castMode: boolean): GenerationUnit[] {
  return castMode
    ? buildCastGenerationUnits(sentences, quotedRangesByFrom())
    : buildGenerationUnits(sentences, 0)
}

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
    document.querySelector('.range-spotlight-current')?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  })
}

// Elapsed/remaining/playhead snapshot for the active queue. Exact for played
// and measured units (the AudioContext clock freezes across pause), estimated
// for units not yet synthesized.
function computeProgress(sentenceCount: number, speed: number): { fraction: number; elapsedSec: number; remainingSec: number } | null {
  if (!audio || playingUnit < 0 || !currentUnits.length) return null
  let elapsed = 0
  let total = 0
  for (let i = 0; i < currentUnits.length; i++) {
    const duration = unitDurations[i] ?? estimator.secondsFor(currentUnits[i].text.length, speed)
    total += duration
    if (i < playingUnit) elapsed += duration
  }
  const intra = playingUnitDuration > 0
    ? Math.min(playingUnitDuration, Math.max(0, audio.now() - playingUnitStartClock))
    : 0
  elapsed += intra
  const sentenceIndex = currentUnits[playingUnit]?.sentenceIndex ?? 0
  const unitFraction = playingUnitDuration > 0 ? intra / playingUnitDuration : 0
  const fraction = sentenceCount > 0 ? Math.min(1, (sentenceIndex + unitFraction) / sentenceCount) : 0
  return { fraction, elapsedSec: elapsed, remainingSec: Math.max(0, total - elapsed) }
}

function pushProgress(): void {
  const state = useReadAloudStore.getState()
  if (state.status === 'idle') return
  const next = computeProgress(state.sentences.length, useAppStore.getState().settings.read_aloud_speed)
  if (!next) return
  const prev = state.progress
  // Change-gated: the display rounds to seconds, so identical rounded values
  // (and sub-pixel fractions) never cause a re-render.
  if (prev
    && Math.round(prev.elapsedSec) === Math.round(next.elapsedSec)
    && Math.round(prev.remainingSec) === Math.round(next.remainingSec)
    && Math.abs(prev.fraction - next.fraction) < 0.002) return
  useReadAloudStore.setState({ progress: next })
}

function startProgressTicker(): void {
  if (progressTimer) return
  progressTimer = setInterval(pushProgress, 250)
}

function stopProgressTicker(): void {
  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }
}

// Click-to-jump and edit-stops-playback callbacks for the ReadAloud
// extension. Module-level, like the extension's own handler slot.
setSpotlightHandlers({
  onRangeClick: index => useReadAloudStore.getState().jumpTo(index),
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

  const clearHighlight = () => withEditorView(clearSpotlight)

  const ensureController = (): ReadAloudController => {
    if (controller) return controller
    const config = () => ({ threads: useAppStore.getState().settings.read_aloud_threads })
    const diagnostic = (line: string) => set(s => ({ diagnostics: [...s.diagnostics.slice(-59), line] }))
    if (!get().nativeInstalled) throw new Error('Native Read Aloud is not installed')
    synth = new NativeSynth(config, () => ReadAloudServerURL(), diagnostic)
    set({ device: 'native', modelLoading: null })
    diagnostic('backend: native sherpa-onnx')
    audio = new WebAudioPort(
      line => set(s => ({ diagnostics: [...s.diagnostics.slice(-59), line] })),
    )
    audio.setVolume(useAppStore.getState().settings.read_aloud_volume)
    audio.setMuted(get().muted)
    controller = new ReadAloudController(synth, audio, {
      onStatus: status => {
        set({ status })
        if (status === 'playing') set({ modelLoading: null })
        // RSS sampling runs exactly while something is audible/pending.
        if (status === 'playing' || status === 'starting') {
          stopPreview() // a sample must never talk over real playback
          void StartReadAloudMemLog()
          startProgressTicker()
        } else {
          void StopReadAloudMemLog()
          if (status === 'idle') {
            stopProgressTicker()
            set({ progress: null })
          }
        }
      },
      onUnitAudio: (unitIndex, seconds) => {
        if (unitIndex < unitDurations.length) unitDurations[unitIndex] = seconds
        const text = currentUnits[unitIndex]?.text ?? ''
        estimator.record(text.length, seconds, useAppStore.getState().settings.read_aloud_speed)
      },
      onSentenceStart: (unitIndex, durationSec) => {
        playingUnit = unitIndex
        playingUnitDuration = durationSec
        playingUnitStartClock = audio?.now() ?? 0
        pushProgress()
        const sentenceIndex = currentUnits[unitIndex]?.sentenceIndex ?? unitIndex
        // Highlight covers the FULL sentence even while a mid-sentence
        // clause unit is the one actually playing; only move it when the
        // sentence actually changes.
        if (get().currentIndex !== sentenceIndex) {
          set({ currentIndex: sentenceIndex })
          refreshCurrentSpeaker()
          withEditorView(view => updateSpotlight(view, { activeIndex: sentenceIndex }))
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
    })
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
    nativeSupported: false,
    nativeInstalled: false,
    nativeBytesTotal: 0,
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
    expanded: false,
    muted: false,
    progress: null,
    castMode: false,
    speakers: [],
    ticks: [],
    currentSpeaker: null,
    dialogueLineCount: 0,
    previewingVoice: null,

    refreshModelStatus: async () => {
      bindEvents()
      try {
        const status = await ReadAloudStatus()
        set(s => ({
          bytesTotal: status.bytes_total,
          nativeSupported: status.native_supported,
          nativeInstalled: status.native_installed,
          nativeBytesTotal: status.native_bytes_total,
          // An in-flight download keeps its state; events own that
          // transition. A corrupt verdict outranks the cheap size check —
          // only a successful re-verify (or removal) clears it.
          modelState: s.modelState === 'downloading' || (s.modelState === 'corrupt' && status.native_installed)
            ? s.modelState
            : (status.native_installed ? 'ready' : 'missing'),
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
        if (result.native_verified) {
          set({
            modelState: 'ready', verified: true, corruptFiles: [], modelError: null,
            nativeInstalled: result.native_verified,
            installInfo: { version: result.version, installedAt: result.installed_at, bytes: result.bytes },
          })
          pushDiag(`verify: ${result.installed_at ? 'v' + result.version + ', ' : ''}${Math.round(result.bytes / (1024 * 1024))} MB, all hashes match`)
          return true
        }
        if (!result.installed) {
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

    removeModel: async () => {
      const pushDiag = (line: string) => set(s => ({ diagnostics: [...s.diagnostics.slice(-59), line] }))
      get().shutdown()
      try {
        await RemoveReadAloudModel() // errors if anything is left on disk
        set({ modelState: 'missing', download: null, modelError: null, verified: false, installInfo: null, corruptFiles: [], nativeInstalled: false })
        pushDiag('remove: native model directory deleted and confirmed empty; pipeline unloaded')
      } catch (e) {
        set({ modelError: String(e) })
        pushDiag(`remove FAILED: ${e instanceof Error ? e.message : String(e)}`)
      }
    },

    play: (sentences, startIndex) => {
      if (!sentences.length) return
      const start = Math.max(0, Math.min(startIndex, sentences.length - 1))
      const proceed = () => {
        const settings = useAppStore.getState().settings
        ensureAttribution()
        set({ sentences, playbackError: null, playerVisible: true, currentIndex: -1 })
        refreshCastState()
        // Units are cast-aware (quote-boundary splits), so they can only be
        // built after attribution and cast state are current.
        const units = buildQueueUnits(sentences, get().castMode)
        const startUnit = Math.max(0, firstUnitOfSentence(units, start))
        withEditorView(view => updateSpotlight(view, {
          active: true,
          ranges: sentences.map(s => ({ from: s.from, to: s.to })),
          activeIndex: -1,
        }))
        const ctrl = ensureController()
        const overrides = buildVoiceOverrides(units, sentences)
        // If the buffer was pre-filled for exactly this queue while the
        // player sat open, arm it — first audio is already synthesized.
        // Only valid from the top: beginPlayback always plays from the
        // prepared start (0), so any other startUnit must fall through to a
        // fresh start() at the requested position.
        if (
          startUnit === 0
          && ctrl.isPreparedFor(units.length)
          && currentUnits.length === units.length
          && currentUnits.every((u, i) => u.text === units[i].text)
        ) {
          // Sync the cast first: a changed voice map silently refills the
          // prepared buffer instead of arming stale audio.
          ctrl.setVoiceOverrides(overrides)
          if (ctrl.beginPlayback()) return
        }
        currentUnits = units
        currentSentences = sentences
        resetProgressTracking(units.length)
        ctrl.start(units.map(u => u.text), startUnit, settings.read_aloud_voice, settings.read_aloud_speed, true, overrides)
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
      ensureAttribution()
      const castMode = useBookStore.getState().book?.read_aloud_cast?.cast_mode ?? false
      const units = buildQueueUnits(sentences, castMode)
      if (
        currentUnits.length === units.length
        && currentUnits.every((u, i) => u.text === units[i].text)
      ) return // already prepared for this exact queue
      const settings = useAppStore.getState().settings
      currentUnits = units
      currentSentences = sentences
      resetProgressTracking(units.length)
      set(s => ({ diagnostics: [...s.diagnostics.slice(-59), `[main t+${Math.round(performance.now())}ms] prefill: ${units.length} units queued from cursor`] }))
      set({ sentences })
      refreshCastState()
      ensureController().start(units.map(u => u.text), 0, settings.read_aloud_voice, settings.read_aloud_speed, false, buildVoiceOverrides(units, sentences))
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
      resetProgressTracking(0)
      stopProgressTicker()
      set({ currentIndex: -1, sentences: [], progress: null, ticks: [], currentSpeaker: null })
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

    seekToFraction: (fraction) => {
      const sentences = get().sentences
      if (!sentences.length) return
      const clamped = Math.max(0, Math.min(1, fraction))
      const index = Math.round(clamped * (sentences.length - 1))
      if (get().status !== 'idle') {
        get().jumpTo(index)
        return
      }
      // Idle with a collected queue (stopped or prepared): start there.
      get().play(sentences, index)
    },

    setVoice: (voice) => {
      void useAppStore.getState().saveSettings({ read_aloud_voice: voice })
      controller?.setVoice(voice)
    },

    setSpeed: (speed) => {
      const clamped = clampReadAloudSpeed(speed)
      void useAppStore.getState().saveSettings({ read_aloud_speed: clamped })
      controller?.setSpeed(clamped)
    },

    cycleSpeed: () => {
      get().setSpeed(nextReadAloudSpeed(useAppStore.getState().settings.read_aloud_speed))
    },

    applyVolume: (volume) => {
      audio?.setVolume(Math.max(0, Math.min(1, volume)))
    },

    setVolume: (volume) => {
      const clamped = Math.max(0, Math.min(1, volume))
      audio?.setVolume(clamped)
      void useAppStore.getState().saveSettings({ read_aloud_volume: clamped })
    },

    toggleMute: () => {
      const muted = !get().muted
      audio?.setMuted(muted)
      set({ muted })
    },

    setExpanded: (expanded) => set({ expanded }),

    setCastMode: (on) => {
      const book = useBookStore.getState().book
      if (!book) return
      useBookStore.getState().updateReadAloudCast({ cast_mode: on, voices: book.read_aloud_cast?.voices ?? {} })
      refreshCastState()
      controller?.setVoiceOverrides(buildVoiceOverrides(currentUnits, currentSentences))
    },

    setSpeakerVoice: (key, voiceId) => {
      if (key === 'narrator') {
        // The narrator's voice is the app-level default voice.
        get().setVoice(voiceId)
        refreshCastState()
        return
      }
      const nameKey = castVoiceKey(key)
      const book = useBookStore.getState().book
      if (!nameKey || !book) return
      const cast = book.read_aloud_cast
      useBookStore.getState().updateReadAloudCast({
        cast_mode: cast?.cast_mode ?? get().castMode,
        voices: { ...(cast?.voices ?? {}), [nameKey]: voiceId },
      })
      refreshCastState()
      controller?.setVoiceOverrides(buildVoiceOverrides(currentUnits, currentSentences))
    },

    previewVoice: (voiceId) => {
      // Toggle off an in-flight preview of the same voice.
      if (get().previewingVoice === voiceId) {
        stopPreview()
        return
      }
      // Idle or paused only: the serialized synthesis queue must not fight
      // audible playback for the engine.
      const status = get().status
      if (status === 'playing' || status === 'starting') return
      if (get().modelState !== 'ready' || !get().verified) return
      stopPreview()
      ensureController() // creates synth + audio without starting playback
      if (!synth || !audio) return
      const speed = useAppStore.getState().settings.read_aloud_speed
      const cacheKey = `${voiceId}@${speed}`
      set({ previewingVoice: voiceId })
      const playChunk = (chunk: AudioChunk) => {
        if (get().previewingVoice !== voiceId || !audio) return
        previewHandle = audio.playOneShot(chunk, () => {
          previewHandle = null
          if (get().previewingVoice === voiceId) set({ previewingVoice: null })
        })
      }
      const cached = previewChunks.get(cacheKey)
      if (cached) {
        playChunk(cached)
        return
      }
      synth.synthesize(previewRequestId++, PREVIEW_TEXT, voiceId, speed).then(
        chunk => {
          previewChunks.set(cacheKey, chunk)
          // The cache never outgrows the voice catalog by much; drop the
          // oldest entry past a dozen.
          if (previewChunks.size > 12) {
            const oldest = previewChunks.keys().next().value
            if (oldest) previewChunks.delete(oldest)
          }
          playChunk(chunk)
        },
        err => {
          if (get().previewingVoice === voiceId) set({ previewingVoice: null })
          set(s => ({ diagnostics: [...s.diagnostics.slice(-59), `preview failed: ${err instanceof Error ? err.message : String(err)}`] }))
        },
      )
    },

    autoCastVoices: () => {
      const book = useBookStore.getState().book
      if (!book || !attribution) return
      const settings = useAppStore.getState().settings
      const assignments = autoCast(get().speakers, attribution.genderEvidence, settings.read_aloud_voice)
      if (!Object.keys(assignments).length) return
      const cast = book.read_aloud_cast
      useBookStore.getState().updateReadAloudCast({
        cast_mode: cast?.cast_mode ?? get().castMode,
        voices: { ...(cast?.voices ?? {}), ...assignments },
      })
      refreshCastState()
      controller?.setVoiceOverrides(buildVoiceOverrides(currentUnits, currentSentences))
    },

    setPlayerVisible: (visible) => {
      if (!visible) get().stop()
      set({ playerVisible: visible, expanded: visible && get().expanded })
    },

    shutdown: () => {
      stopPreview()
      previewChunks.clear()
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
      resetProgressTracking(0)
      stopProgressTicker()
      estimator.reset()
      clearHighlight()
      set({ status: 'idle', sentences: [], currentIndex: -1, playerVisible: false, modelLoading: null, playbackError: null, expanded: false, progress: null, ticks: [], currentSpeaker: null, speakers: [], dialogueLineCount: 0 })
    },
  }
})
