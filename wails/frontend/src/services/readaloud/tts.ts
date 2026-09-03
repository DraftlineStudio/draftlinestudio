// Native sherpa-onnx synthesis. Draftline has one Read Aloud protocol: text
// goes to its authenticated loopback service and raw Float32 PCM comes back.
// No browser inference worker, WASM runtime, WebGPU model, or remote fallback
// participates in playback.

import type { AudioChunk, SynthPort } from './controller'

export interface SynthConfig {
  threads: 'single' | 'auto'
}

export class NativeSynth implements SynthPort {
  private pending = new Map<number, AbortController>()

  constructor(
    private getConfig: () => SynthConfig,
    private getBase: () => Promise<string>,
    private onDiagnostic?: (line: string) => void,
  ) {}

  async synthesize(id: number, text: string, voice: string, speed: number): Promise<AudioChunk> {
    const abort = new AbortController()
    this.pending.set(id, abort)
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
      const samples = new Float32Array(bytes)
      const elapsed = performance.now() - started
      const audioMs = samples.length / sampleRate * 1000
      this.onDiagnostic?.(`native synthesis ${elapsed.toFixed(0)} ms for ${(audioMs / 1000).toFixed(1)}s audio (RTF ${(elapsed / audioMs).toFixed(2)})`)
      return { samples, sampleRate }
    } finally {
      this.pending.delete(id)
    }
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
