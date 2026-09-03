// Native sherpa-onnx synthesis. Draftline has one Read Aloud protocol: text
// goes to its authenticated loopback service and raw Float32 PCM comes back.
// No browser inference worker, WASM runtime, WebGPU model, or remote fallback
// participates in playback.

import type { AudioChunk, SynthPort } from './controller'

export interface SynthConfig {
  threads: 'single' | 'auto'
}

const SENTENCE_PAUSE_MS = 220
const STRONG_SENTENCE_PAUSE_MS = 280

// Very long sentences can be split into clause-sized synthesis units. Those
// units already receive Kokoro's punctuation cadence and must not sound like
// separate sentences. Completed sentences receive a small explicit rest so
// individually generated buffers do not run together unnaturally.
function pauseAfter(text: string): number {
  const ending = text.trim()
  if (/[,;:—–-]["'”’\)\]]*$/.test(ending)) return 0
  if (/[!?]["'”’\)\]]*$/.test(ending)) return STRONG_SENTENCE_PAUSE_MS
  return SENTENCE_PAUSE_MS
}

function appendSentencePause(samples: Float32Array, sampleRate: number, text: string): Float32Array {
  const pauseMs = pauseAfter(text)
  const pauseSamples = Math.round(sampleRate * pauseMs / 1000)
  if (pauseSamples <= 0) return samples
  const result = new Float32Array(samples.length + pauseSamples)
  result.set(samples)
  return result
}

export class NativeSynth implements SynthPort {
  private pending = new Map<number, AbortController>()
  // sherpa's eSpeak phonemizer has process-global state. Keep requests in
  // document order and never invoke it concurrently; the native engine is
  // faster than real time and the controller maintains an audio runway.
  private queue: Promise<void> = Promise.resolve()

  constructor(
    private getConfig: () => SynthConfig,
    private getBase: () => Promise<string>,
    private onDiagnostic?: (line: string) => void,
  ) {}

  async synthesize(id: number, text: string, voice: string, speed: number): Promise<AudioChunk> {
    const abort = new AbortController()
    this.pending.set(id, abort)
    const result = this.queue.then(async () => {
      const started = performance.now()
      try {
        const base = await this.getBase()
        if (!base) throw new Error('Native Read Aloud service is unavailable')
        const response = await fetch(`${base}/readaloud-native/synthesize`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            text, voice, speed,
            threads: this.getConfig().threads === 'single' ? 1 : 0,
          }),
          signal: abort.signal,
        })
        if (!response.ok) throw new Error((await response.text()) || `Native synthesis failed (${response.status})`)
        const sampleRate = Number(response.headers.get('X-Draftline-Sample-Rate'))
        if (!Number.isFinite(sampleRate) || sampleRate <= 0) {
          throw new Error('Native synthesis response did not expose its sample rate')
        }
        const bytes = await response.arrayBuffer()
        if (bytes.byteLength === 0 || bytes.byteLength % 4 !== 0) throw new Error('Native synthesis returned invalid PCM')
        const generatedSamples = new Float32Array(bytes)
        const elapsed = performance.now() - started
        const audioMs = generatedSamples.length / sampleRate * 1000
        const pauseMs = pauseAfter(text)
        const samples = appendSentencePause(generatedSamples, sampleRate, text)
        this.onDiagnostic?.(`native synthesis ${elapsed.toFixed(0)} ms for ${(audioMs / 1000).toFixed(1)}s audio (RTF ${(elapsed / audioMs).toFixed(2)}), ${pauseMs} ms sentence pause`)
        return { samples, sampleRate }
      } finally {
        this.pending.delete(id)
      }
    })
    // A failed or cancelled request must not poison later queue entries.
    this.queue = result.then(() => undefined, () => undefined)
    return result
  }

  cancel(id: number): void {
    this.pending.get(id)?.abort()
    this.pending.delete(id)
  }

  shutdown(): void {
    for (const abort of this.pending.values()) abort.abort()
    this.pending.clear()
  }
}
