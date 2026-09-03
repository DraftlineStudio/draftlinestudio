import { describe, it, expect, beforeEach } from 'vitest'
import {
  ReadAloudController,
  type AudioChunk,
  type AudioPort,
  type ControllerEvents,
  type ControllerStatus,
  type PlaybackHandle,
  type SynthPort,
} from './controller'

// ── fakes ────────────────────────────────────────────────────────────────────

interface Deferred {
  id: number
  text: string
  voice: string
  speed: number
  resolve: (chunk?: AudioChunk) => void
  reject: (err: Error) => void
}

class FakeSynth implements SynthPort {
  requests: Deferred[] = []
  cancelledIds: number[] = []

  synthesize(id: number, text: string, voice: string, speed: number): Promise<AudioChunk> {
    return new Promise((resolve, reject) => {
      this.requests.push({
        id, text, voice, speed,
        resolve: (chunk?: AudioChunk) => resolve(chunk ?? { samples: new Float32Array(8), sampleRate: 24000 }),
        reject,
      })
    })
  }

  cancel(id: number): void {
    this.cancelledIds.push(id)
    // Like the real worker port: a cancelled request never resolves.
    this.requests = this.requests.filter(r => r.id !== id)
  }

  // Resolve the oldest unresolved request.
  async resolveNext(): Promise<void> {
    const req = this.requests.shift()
    if (!req) throw new Error('no pending synthesis request')
    req.resolve()
    await flush()
  }
}

interface Enqueued {
  onEnded: () => void
  stopped: boolean
}

class FakeAudio implements AudioPort {
  queue: Enqueued[] = []
  pauses = 0
  resumes = 0
  stopAllCalls = 0

  enqueue(_chunk: AudioChunk, onEnded: () => void): PlaybackHandle {
    const entry: Enqueued = { onEnded, stopped: false }
    this.queue.push(entry)
    return { stop: () => { entry.stopped = true } }
  }

  pause(): void { this.pauses++ }
  resume(): void { this.resumes++ }
  stopAll(): void {
    this.stopAllCalls++
    for (const entry of this.queue) entry.stopped = true
    this.queue = this.queue.filter(e => !e.stopped)
  }

  // Ends the currently playing (oldest live) chunk naturally.
  endCurrent(): void {
    const entry = this.queue.shift()
    if (!entry) throw new Error('nothing playing')
    entry.onEnded()
  }
}

function flush(): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, 0))
}

// ── harness ──────────────────────────────────────────────────────────────────

let synth: FakeSynth
let audio: FakeAudio
let statuses: ControllerStatus[]
let started: number[]
let errors: string[]
let finishes: number
let controller: ReadAloudController

const SENTENCES = ['One.', 'Two.', 'Three.']

beforeEach(() => {
  synth = new FakeSynth()
  audio = new FakeAudio()
  statuses = []
  started = []
  errors = []
  finishes = 0
  const events: ControllerEvents = {
    onStatus: s => statuses.push(s),
    onSentenceStart: i => started.push(i),
    onError: m => errors.push(m),
    onFinished: () => finishes++,
  }
  controller = new ReadAloudController(synth, audio, events)
})

async function startPlaying(startIndex = 0, voice = 'af_heart', speed = 1.2) {
  controller.start(SENTENCES, startIndex, voice, speed)
  await synth.resolveNext() // first sentence becomes audible
}

describe('ReadAloudController', () => {
  it('starts by firing the whole lookahead window without awaiting playback', async () => {
    controller.start(SENTENCES, 0, 'af_heart', 1.2)
    expect(statuses).toEqual(['starting'])
    // Producer: every unit in the window is requested up front; the worker
    // serializes actual synthesis.
    expect(synth.requests.map(r => r.text)).toEqual(['One.', 'Two.', 'Three.'])

    await synth.resolveNext()
    expect(statuses).toEqual(['starting', 'playing'])
    expect(started).toEqual([0])
    expect(audio.queue).toHaveLength(1)
    // Remaining requests still in flight — production never waited on play.
    expect(synth.requests.map(r => r.text)).toEqual(['Two.', 'Three.'])
  })

  it('prepares silently and arms instantly on beginPlayback', async () => {
    controller.start(SENTENCES, 0, 'af_heart', 1.2, false)
    // Prepared: producer fills, nothing audible, status untouched.
    expect(statuses).toEqual([])
    expect(controller.isPreparedFor(SENTENCES.length)).toBe(true)
    expect(synth.requests.map(r => r.text)).toEqual(['One.', 'Two.', 'Three.'])
    await synth.resolveNext()
    expect(audio.queue).toHaveLength(0)
    expect(started).toEqual([])

    // Arming plays the pre-filled first chunk with zero synthesis wait.
    expect(controller.beginPlayback()).toBe(true)
    expect(started).toEqual([0])
    expect(statuses[statuses.length - 1]).toBe('playing')
  })

  it('schedules the next chunk before the current ends for gapless handoff', async () => {
    await startPlaying()
    await synth.resolveNext() // sentence two ready while one still plays
    expect(audio.queue).toHaveLength(2) // scheduled back-to-back

    audio.endCurrent()
    // Two is audible immediately; three now synthesizes.
    expect(started).toEqual([0, 1])
    expect(synth.requests.map(r => r.text)).toEqual(['Three.'])
  })

  it('waits for lagging synthesis and plays it the moment it lands', async () => {
    await startPlaying()
    audio.endCurrent() // sentence one ends before two is ready
    expect(started).toEqual([0])

    await synth.resolveNext()
    expect(started).toEqual([0, 1])
    expect(audio.queue).toHaveLength(1)
  })

  it('finishes after the last sentence and reports idle', async () => {
    await startPlaying()
    await synth.resolveNext() // two ready
    audio.endCurrent() // one done, two playing
    await synth.resolveNext() // three ready
    audio.endCurrent() // two done, three playing
    expect(started).toEqual([0, 1, 2])

    audio.endCurrent() // three done
    expect(finishes).toBe(1)
    expect(statuses[statuses.length - 1]).toBe('idle')
    expect(errors).toEqual([])
  })

  it('pauses and resumes without touching the queue', async () => {
    await startPlaying()
    const stopAllsAfterStart = audio.stopAllCalls
    controller.pause()
    expect(audio.pauses).toBe(1)
    expect(statuses[statuses.length - 1]).toBe('paused')
    controller.resume()
    expect(statuses[statuses.length - 1]).toBe('playing')
    expect(audio.stopAllCalls).toBe(stopAllsAfterStart)
    expect(audio.queue).toHaveLength(1)
  })

  it('skip forward cancels in-flight synthesis and restarts at the target', async () => {
    await startPlaying() // playing 0, synthesizing 1
    const inflightId = synth.requests[0].id
    controller.skip(1)
    expect(synth.cancelledIds).toContain(inflightId)
    expect(audio.stopAllCalls).toBeGreaterThan(0)
    expect(statuses[statuses.length - 1]).toBe('starting')

    await synth.resolveNext() // target sentence 1
    expect(started).toEqual([0, 1])
    expect(statuses[statuses.length - 1]).toBe('playing')
  })

  it('skip past the end stops playback cleanly', async () => {
    await startPlaying(2) // last sentence
    controller.skip(1)
    expect(finishes).toBe(1)
    expect(statuses[statuses.length - 1]).toBe('idle')
  })

  it('skip back before the start clamps to the first sentence', async () => {
    await startPlaying(0)
    controller.skip(-1)
    await synth.resolveNext()
    expect(started).toEqual([0, 0])
  })

  it('jumpTo replays a cached sentence without re-synthesizing', async () => {
    await startPlaying() // 0 cached after this
    await synth.resolveNext() // 1 cached
    audio.endCurrent() // playing 1
    controller.jumpTo(0)
    // Both 0 and its lookahead 1 are cached: audible and scheduled
    // immediately; the depth-2 lookahead synthesizes 2 in the background.
    expect(started).toEqual([0, 1, 0])
    expect(synth.requests.map(r => r.text)).toEqual(['Three.'])
    expect(audio.queue).toHaveLength(2)

    audio.endCurrent()
    // Cached 1 takes over gaplessly; 2 is still the only synthesis.
    expect(started).toEqual([0, 1, 0, 1])
    expect(synth.requests.map(r => r.text)).toEqual(['Three.'])
  })

  it('a speed change keeps the current sentence and re-synthesizes the lookahead', async () => {
    await startPlaying() // playing 0 at 1.2
    await synth.resolveNext() // 1 scheduled at 1.2
    const scheduled = audio.queue[1]

    controller.setSpeed(1.5)
    expect(scheduled.stopped).toBe(true) // old-speed chunk unscheduled
    expect(synth.requests.map(r => ({ text: r.text, speed: r.speed }))).toEqual([
      { text: 'Two.', speed: 1.5 },
    ])
    expect(started).toEqual([0]) // current sentence uninterrupted

    await synth.resolveNext()
    audio.endCurrent()
    expect(started).toEqual([0, 1])
  })

  it('a voice change mid-synthesis cancels the stale request and re-requests', async () => {
    controller.start(SENTENCES, 0, 'af_heart', 1.2)
    const oldId = synth.requests[0].id
    controller.setVoice('bm_george')
    // The old-voice request is cancelled; only the new-voice one remains.
    expect(synth.cancelledIds).toContain(oldId)
    expect(synth.requests.map(r => r.voice)).toEqual(['bm_george'])
    await synth.resolveNext()
    expect(started).toEqual([0])
  })

  it('a synthesis error stops playback and surfaces the message', async () => {
    controller.start(SENTENCES, 0, 'af_heart', 1.2)
    synth.requests.shift()!.reject(new Error('model exploded'))
    await flush()
    expect(errors).toEqual(['model exploded'])
    expect(statuses[statuses.length - 1]).toBe('idle')
    expect(audio.queue).toHaveLength(0)
  })

  it('drops results from a superseded generation', async () => {
    controller.start(SENTENCES, 0, 'af_heart', 1.2)
    const stale = synth.requests.shift()!
    controller.stop()
    stale.resolve()
    await flush()
    expect(started).toEqual([])
    expect(audio.queue).toHaveLength(0)
    expect(statuses[statuses.length - 1]).toBe('idle')
  })

  it('stop during playback silences everything and reports idle', async () => {
    await startPlaying()
    controller.stop()
    expect(audio.queue).toHaveLength(0)
    expect(statuses[statuses.length - 1]).toBe('idle')
    // The lookahead request for sentence two was cancelled.
    expect(synth.cancelledIds.length).toBeGreaterThan(0)
  })

  it('starting with an empty sentence list finishes immediately', () => {
    controller.start([], 0, 'af_heart', 1.2)
    expect(finishes).toBe(1)
    expect(statuses).toEqual([])
  })
})
