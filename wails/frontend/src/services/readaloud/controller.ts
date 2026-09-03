// Read Aloud playback controller: a pure sentence-queue state machine.
// Synthesis and audio output are injected ports, so the whole pipeline —
// lookahead, gapless handoff, skip/jump, speed changes, stale-result
// discipline — is unit-testable without a worker, a model, or an
// AudioContext (vitest runs in a plain node environment).
//
// Pipeline invariants:
// - Lookahead is LOOKAHEAD generation units: the worker serializes them and
//   the next ready unit is scheduled back-to-back for gapless handoff.
//   Playback starts with a duration reserve instead of trusting a raw unit
//   count, because sentence lengths and synthesis times vary sharply.
// - Hard resets (start/stop/skip/jump) bump a generation counter; every async
//   completion checks it, so a stale chunk can never play.
// - Voice/speed changes keep the sentence that is already audible, discard
//   the cached/scheduled lookahead, and re-synthesize it with new parameters.

export interface AudioChunk {
  samples: Float32Array
  sampleRate: number
}

export interface SynthPort {
  synthesize(id: number, text: string, voice: string, speed: number): Promise<AudioChunk>
  cancel(id: number): void
}

export interface PlaybackHandle {
  stop(): void
}

export interface AudioPort {
  // Schedules a chunk to play after everything already enqueued (gapless).
  // onEnded fires when the chunk finishes naturally — not when stopped.
  enqueue(chunk: AudioChunk, onEnded: () => void): PlaybackHandle
  pause(): void
  resume(): void
  stopAll(): void
}

export type ControllerStatus = 'idle' | 'starting' | 'playing' | 'paused'

// Producer/consumer split: the producer keeps the buffer filled to LOOKAHEAD
// units ahead of the playhead and never awaits playback (requests are fired
// eagerly; the worker serializes actual synthesis). The consumer plays from
// the cache and only ever waits on generation when the buffer is empty (the
// waitingToPlay path).
const LOOKAHEAD = 5

// Do not begin (or resume after an underrun) on the first available sliver of
// audio. A short unit can finish before the following long unit is ready even
// on a faster-than-real-time backend. Holding a contiguous runway makes
// that uneven sentence pair sound like prose instead of speak-wait-speak.
const START_BUFFER_SECONDS = 6
const START_BUFFER_UNITS = 1

// How many already-played chunks stay cached (instant short skip-backs).
// Without a bound, a chapter-length play accumulates every sentence's
// Float32Array for the whole session.
const CACHE_BEHIND = 4

export interface ControllerEvents {
  onStatus(status: ControllerStatus): void
  // durationSec is the audible length of the unit that just started (0 when
  // its chunk is unexpectedly absent).
  onSentenceStart(index: number, durationSec: number): void
  onError(message: string): void
  onFinished(): void
  // Fired whenever a unit's audio lands in the cache — the raw material for
  // elapsed/remaining time estimation. Optional; playback ignores it.
  onUnitAudio?(index: number, durationSec: number): void
}

export interface ControllerOptions {
  startBufferSeconds?: number
  startBufferUnits?: number
}

export class ReadAloudController {
  private status: ControllerStatus = 'idle'
  private sentences: string[] = []
  private voice = ''
  // Per-unit voice overrides (cast mode): index-aligned with sentences; null
  // entries fall back to the base voice. null altogether = base voice only,
  // and every code path behaves exactly as before overrides existed.
  private voiceOverrides: (string | null)[] | null = null
  private speed = 1
  private generation = 0
  private nextRequestId = 1

  private playingIndex: number | null = null
  private preparedStart: number | null = null
  private scheduledNext: { index: number; handle: PlaybackHandle } | null = null
  // Outstanding synthesis requests, index → request id (≤ LOOKAHEAD + the
  // immediately-needed sentence).
  private pendingSynth = new Map<number, number>()
  // The index to make audible as soon as its chunk exists (initial start,
  // post-skip target, or a gap where synthesis lagged behind playback).
  private waitingToPlay: number | null = null
  private cache = new Map<number, AudioChunk>()
  private readonly startBufferSeconds: number
  private readonly startBufferUnits: number

  constructor(
    private synth: SynthPort,
    private audio: AudioPort,
    private events: ControllerEvents,
    options: ControllerOptions = {},
  ) {
    this.startBufferSeconds = Math.max(0, options.startBufferSeconds ?? START_BUFFER_SECONDS)
    this.startBufferUnits = Math.max(1, options.startBufferUnits ?? START_BUFFER_UNITS)
  }

  // autoplay=false prepares: the producer pre-fills the buffer from
  // startIndex but nothing becomes audible until beginPlayback(). Used to
  // warm the pipeline the moment the player opens, before play is pressed.
  start(sentences: string[], startIndex: number, voice: string, speed: number, autoplay = true, voiceOverrides: (string | null)[] | null = null): void {
    this.reset()
    if (!sentences.length) {
      if (autoplay) this.events.onFinished()
      return
    }
    this.sentences = sentences
    this.voice = voice
    this.voiceOverrides = voiceOverrides
    this.speed = speed
    const clamped = Math.max(0, Math.min(startIndex, sentences.length - 1))
    this.preparedStart = clamped
    if (autoplay) {
      this.setStatus('starting')
      this.audio.resume()
      this.playFrom(clamped)
    } else {
      // Producer only: fill the buffer silently.
      this.topUp()
    }
  }

  // Arms a prepared (autoplay=false) queue. No-op unless prepared and idle.
  beginPlayback(): boolean {
    if (this.status !== 'idle' || this.preparedStart === null || !this.sentences.length) return false
    this.setStatus('starting')
    this.audio.resume()
    this.playFrom(this.preparedStart)
    return true
  }

  // True when a prepared buffer for this many sentences is standing by.
  isPreparedFor(count: number): boolean {
    return this.status === 'idle' && this.preparedStart !== null && this.sentences.length === count
  }

  pause(): void {
    if (this.status !== 'playing' && this.status !== 'starting') return
    this.audio.pause()
    this.setStatus('paused')
  }

  resume(): void {
    if (this.status !== 'paused') return
    this.audio.resume()
    this.setStatus(this.playingIndex !== null ? 'playing' : 'starting')
  }

  stop(): void {
    // An autoplay=false queue prepares while its public status remains idle.
    // It still owns synthesis requests and cached audio that must be released
    // when the player closes or its document changes.
    if (this.status === 'idle' && this.preparedStart === null) return
    this.reset()
    this.setStatus('idle')
  }

  skip(delta: 1 | -1): void {
    const from = this.playingIndex ?? this.waitingToPlay
    if (from === null) return
    const target = from + delta
    if (target >= this.sentences.length) {
      this.finish()
      return
    }
    this.jumpTo(target)
  }

  jumpTo(index: number): void {
    if (this.status === 'idle' || !this.sentences.length) return
    const target = Math.max(0, Math.min(index, this.sentences.length - 1))
    this.softReset()
    this.setStatus('starting')
    this.audio.resume()
    this.playFrom(target)
  }

  setVoice(voice: string): void {
    if (voice === this.voice) return
    this.voice = voice
    this.reprimeLookahead()
  }

  // Replaces the per-unit voice map (cast changes mid-play or while
  // prepared). Cheap no-op when the effective per-unit voices are unchanged.
  setVoiceOverrides(next: (string | null)[] | null): void {
    const prev = this.voiceOverrides
    const allNull = (o: (string | null)[] | null) => !o || o.every(v => v === null)
    const unchanged = prev === next
      || (allNull(prev) && allNull(next))
      || (!!prev && !!next && prev.length === next.length && prev.every((v, i) => v === next[i]))
    this.voiceOverrides = next
    if (!unchanged) this.reprimeLookahead()
  }

  setSpeed(speed: number): void {
    if (speed === this.speed) return
    this.speed = speed
    this.reprimeLookahead()
  }

  // ── internals ──────────────────────────────────────────────────────────────

  private setStatus(status: ControllerStatus): void {
    if (this.status === status) return
    this.status = status
    this.events.onStatus(status)
  }

  // Full teardown: also clears the synthesized-chunk cache.
  private reset(): void {
    this.softReset()
    this.cache.clear()
    this.sentences = []
    this.voiceOverrides = null
  }

  private effectiveVoice(index: number): string {
    return this.voiceOverrides?.[index] ?? this.voice
  }

  // Teardown that keeps the cache (skip back can replay instantly).
  private softReset(): void {
    this.generation++
    this.cancelPending()
    this.scheduledNext = null
    this.audio.stopAll()
    this.playingIndex = null
    this.preparedStart = null
    this.waitingToPlay = null
  }

  private cancelPending(): void {
    for (const [, id] of this.pendingSynth) {
      this.synth.cancel(id)
    }
    this.pendingSynth.clear()
  }

  private finish(): void {
    this.reset()
    this.setStatus('idle')
    this.events.onFinished()
  }

  private fail(message: string): void {
    this.reset()
    this.setStatus('idle')
    this.events.onError(message)
  }

  private playFrom(index: number): void {
    this.preparedStart = null
    this.waitingToPlay = index
    // Fire the whole lookahead window immediately; the worker serializes it.
    // Playback begins only after tryStartWaiting sees a useful contiguous
    // reserve, preventing short-then-long sentence pairs from draining it.
    this.topUp()
    this.tryStartWaiting()
  }

  private request(index: number): void {
    if (index >= this.sentences.length || this.cache.has(index)) return
    if (this.pendingSynth.has(index)) return
    const generation = this.generation
    const voice = this.effectiveVoice(index)
    const speed = this.speed
    const id = this.nextRequestId++
    this.pendingSynth.set(index, id)
    this.synth.synthesize(id, this.sentences[index], voice, speed).then(
      chunk => {
        if (generation !== this.generation || this.pendingSynth.get(index) !== id) return
        this.pendingSynth.delete(index)
        // A voice/speed change while synthesizing makes this chunk stale.
        if (voice !== this.effectiveVoice(index) || speed !== this.speed) {
          this.request(index)
          return
        }
        this.cache.set(index, chunk)
        this.events.onUnitAudio?.(index, chunk.sampleRate > 0 ? chunk.samples.length / chunk.sampleRate : 0)
        this.deliver(index, chunk)
        this.tryStartWaiting()
        // Producer pump: every completed unit immediately requests the next
        // uncovered one, independent of playback progress.
        this.topUp()
      },
      err => {
        if (generation !== this.generation || this.pendingSynth.get(index) !== id) return
        this.pendingSynth.delete(index)
        this.fail(err instanceof Error ? err.message : String(err))
      },
    )
  }

  // Keeps the lookahead buffer filled to LOOKAHEAD units beyond the
  // playhead (or the prepared start while nothing is audible).
  private topUp(): void {
    const base = this.playingIndex ?? this.waitingToPlay ?? this.preparedStart
    if (base === null) return
    for (let index = base; index <= base + LOOKAHEAD && index < this.sentences.length; index++) {
      if (!this.cache.has(index) && !this.pendingSynth.has(index)) {
        this.request(index)
      }
    }
  }

  private deliver(index: number, chunk: AudioChunk): void {
    if (this.playingIndex !== null && index === this.playingIndex + 1 && !this.scheduledNext) {
      this.enqueueChunk(index, chunk)
    }
    // Otherwise it stays cached for its turn.
  }

  /**
   * Starts a waiting queue only when enough consecutive synthesized audio is
   * available to absorb normal variation in inference time. Reaching the end
   * of the queue is also sufficient, so a one-sentence selection never waits
   * for audio that cannot exist.
   */
  private tryStartWaiting(): void {
    const start = this.waitingToPlay
    if (start === null || this.playingIndex !== null) return
    const first = this.cache.get(start)
    if (!first) return

    let seconds = 0
    let units = 0
    let index = start
    for (; index < this.sentences.length; index++) {
      const chunk = this.cache.get(index)
      if (!chunk) break
      seconds += chunk.sampleRate > 0 ? chunk.samples.length / chunk.sampleRate : 0
      units++
      if (units >= this.startBufferUnits && seconds >= this.startBufferSeconds) break
    }
    const reachedEnd = start + units >= this.sentences.length
    if (!reachedEnd && (units < this.startBufferUnits || seconds < this.startBufferSeconds)) return

    this.waitingToPlay = null
    this.enqueueChunk(start, first)
    this.onChunkStarted(start)
  }

  private enqueueChunk(index: number, chunk: AudioChunk): void {
    const generation = this.generation
    const handle = this.audio.enqueue(chunk, () => {
      if (generation !== this.generation) return
      this.onChunkEnded(index)
    })
    this.scheduledNext = { index, handle }
  }

  private onChunkStarted(index: number): void {
    if (this.scheduledNext?.index === index) this.scheduledNext = null
    this.playingIndex = index
    if (this.status !== 'paused') this.setStatus('playing')
    const chunk = this.cache.get(index)
    this.events.onSentenceStart(index, chunk && chunk.sampleRate > 0 ? chunk.samples.length / chunk.sampleRate : 0)
    // Evict chunks that have fallen behind the replay window so long plays
    // hold a bounded number of audio buffers.
    for (const cachedIndex of this.cache.keys()) {
      if (cachedIndex < index - CACHE_BEHIND) this.cache.delete(cachedIndex)
    }
    this.primeNext(index)
  }

  // Gapless handoff: make sure the immediate next chunk is scheduled if it
  // is already cached, then let the producer keep the buffer filled.
  private primeNext(index: number): void {
    const next = index + 1
    if (next < this.sentences.length) {
      const cached = this.cache.get(next)
      if (cached) this.deliver(next, cached)
    }
    this.topUp()
  }

  private onChunkEnded(index: number): void {
    if (this.playingIndex === index) this.playingIndex = null
    if (this.scheduledNext) {
      // The next chunk is already scheduled back-to-back; it is now audible.
      this.onChunkStarted(this.scheduledNext.index)
      return
    }
    const next = index + 1
    if (next >= this.sentences.length) {
      this.finish()
      return
    }
    // Synthesis lagged behind playback: play the moment the chunk lands.
    this.waitingToPlay = next
    const cached = this.cache.get(next)
    if (cached) {
      this.tryStartWaiting()
    } else {
      this.request(next)
    }
    this.topUp()
  }

  // After a voice/speed change: keep what is audible, rebuild the lookahead.
  private reprimeLookahead(): void {
    if (this.status === 'idle') {
      // A prepared (autoplay=false) queue sits at status idle but owns cached
      // audio and in-flight requests in the OLD voice/speed. Silently re-fill
      // the buffer with the new parameters so arming it never plays stale
      // chunks.
      if (this.preparedStart === null) return
      this.cache.clear()
      this.cancelPending()
      this.topUp()
      return
    }
    this.cache.clear()
    this.cancelPending()
    if (this.scheduledNext) {
      const { index, handle } = this.scheduledNext
      handle.stop()
      this.scheduledNext = null
      this.request(index)
    } else if (this.waitingToPlay !== null) {
      this.request(this.waitingToPlay)
    } else if (this.playingIndex !== null) {
      this.primeNext(this.playingIndex)
    }
  }
}
