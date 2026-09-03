// Read Aloud playback controller: a pure sentence-queue state machine.
// Synthesis and audio output are injected ports, so the whole pipeline —
// lookahead, gapless handoff, skip/jump, speed changes, stale-result
// discipline — is unit-testable without a worker, a model, or an
// AudioContext (vitest runs in a plain node environment).
//
// Pipeline invariants:
// - Lookahead is exactly one sentence: while N plays, N+1 synthesizes and is
//   scheduled back-to-back for gapless handoff.
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

export interface ControllerEvents {
  onStatus(status: ControllerStatus): void
  onSentenceStart(index: number): void
  onError(message: string): void
  onFinished(): void
}

export class ReadAloudController {
  private status: ControllerStatus = 'idle'
  private sentences: string[] = []
  private voice = ''
  private speed = 1
  private generation = 0
  private nextRequestId = 1

  private playingIndex: number | null = null
  private scheduledNext: { index: number; handle: PlaybackHandle } | null = null
  private pendingSynth: { index: number; id: number } | null = null
  // The index to make audible as soon as its chunk exists (initial start,
  // post-skip target, or a gap where synthesis lagged behind playback).
  private waitingToPlay: number | null = null
  private cache = new Map<number, AudioChunk>()

  constructor(
    private synth: SynthPort,
    private audio: AudioPort,
    private events: ControllerEvents,
  ) {}

  start(sentences: string[], startIndex: number, voice: string, speed: number): void {
    this.reset()
    if (!sentences.length) {
      this.events.onFinished()
      return
    }
    this.sentences = sentences
    this.voice = voice
    this.speed = speed
    this.setStatus('starting')
    this.audio.resume()
    this.playFrom(Math.max(0, Math.min(startIndex, sentences.length - 1)))
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
    if (this.status === 'idle') return
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
  }

  // Teardown that keeps the cache (skip back can replay instantly).
  private softReset(): void {
    this.generation++
    this.cancelPending()
    this.scheduledNext = null
    this.audio.stopAll()
    this.playingIndex = null
    this.waitingToPlay = null
  }

  private cancelPending(): void {
    if (this.pendingSynth) {
      this.synth.cancel(this.pendingSynth.id)
      this.pendingSynth = null
    }
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
    this.waitingToPlay = index
    const cached = this.cache.get(index)
    if (cached) {
      this.deliver(index, cached)
    } else {
      this.request(index)
    }
  }

  private request(index: number): void {
    if (index >= this.sentences.length || this.cache.has(index)) return
    if (this.pendingSynth?.index === index) return
    this.cancelPending()
    const generation = this.generation
    const voice = this.voice
    const speed = this.speed
    const id = this.nextRequestId++
    this.pendingSynth = { index, id }
    this.synth.synthesize(id, this.sentences[index], voice, speed).then(
      chunk => {
        if (generation !== this.generation || this.pendingSynth?.id !== id) return
        this.pendingSynth = null
        // A voice/speed change while synthesizing makes this chunk stale.
        if (voice !== this.voice || speed !== this.speed) {
          this.request(index)
          return
        }
        this.cache.set(index, chunk)
        this.deliver(index, chunk)
      },
      err => {
        if (generation !== this.generation || this.pendingSynth?.id !== id) return
        this.pendingSynth = null
        this.fail(err instanceof Error ? err.message : String(err))
      },
    )
  }

  private deliver(index: number, chunk: AudioChunk): void {
    if (this.waitingToPlay === index) {
      this.waitingToPlay = null
      this.enqueueChunk(index, chunk)
      // Nothing was audible, so this chunk starts as soon as the clock runs.
      this.onChunkStarted(index)
    } else if (this.playingIndex !== null && index === this.playingIndex + 1 && !this.scheduledNext) {
      this.enqueueChunk(index, chunk)
    }
    // Otherwise it stays cached for its turn.
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
    this.events.onSentenceStart(index)
    this.primeNext(index)
  }

  // Gapless lookahead: schedule the next chunk if cached, else synthesize it.
  private primeNext(index: number): void {
    const next = index + 1
    if (next >= this.sentences.length) return
    const cached = this.cache.get(next)
    if (cached) {
      this.deliver(next, cached)
    } else {
      this.request(next)
    }
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
      this.deliver(next, cached)
    } else {
      this.request(next)
    }
  }

  // After a voice/speed change: keep what is audible, rebuild the lookahead.
  private reprimeLookahead(): void {
    if (this.status === 'idle') return
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
