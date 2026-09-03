// SynthPort implementation over the readAloud worker. The worker (and with
// it kokoro-js and the ONNX runtime) is created lazily on the first
// synthesis request and stays alive across plays so the model loads once per
// session; shutdown() terminates it and frees the model memory. Device and
// threading come from user settings — WASM single-threaded q8 is the
// explicit cross-platform default, never autodetected.

import type { AudioChunk, SynthPort } from './controller'

// Fallback when the loopback server is unavailable: the Wails asset-handler
// path (same-origin). Threading may not survive there — WebView2 does not
// reliably route nested-worker requests through the asset scheme handler.
const FALLBACK_BASE = '/readaloud-models'

export interface ModelLoadProgress {
  file: string
  loaded: number
  total: number
}

export interface SynthConfig {
  device: 'wasm' | 'webgpu'
  threads: 'single' | 'auto'
}

interface PendingRequest {
  resolve: (chunk: AudioChunk) => void
  reject: (err: Error) => void
}

// Counts worker constructions across the session. The diagnostics surface it
// so "the pipeline is being rebuilt per play" is immediately visible.
let workerStarts = 0

export class WorkerSynth implements SynthPort {
  private worker: Worker | null = null
  private ready: Promise<void> | null = null
  private becameReady = false
  private pending = new Map<number, PendingRequest>()

  constructor(
    private getConfig: () => SynthConfig,
    private onLoadProgress?: (p: ModelLoadProgress) => void,
    private onDevice?: (device: 'wasm' | 'webgpu') => void,
    private onDiagnostic?: (line: string) => void,
    private getBase?: () => Promise<string>,
  ) {}

  private ensureReady(): Promise<void> {
    // Single construction guard: concurrent callers (and every later
    // sentence) await this same promise — the pipeline is built exactly once
    // per WorkerSynth lifetime, torn down only by shutdown().
    if (this.ready) return this.ready
    this.ready = this.boot()
    return this.ready
  }

  private async boot(): Promise<void> {
    let base = FALLBACK_BASE
    if (this.getBase) {
      try {
        base = (await this.getBase()) || FALLBACK_BASE
      } catch { /* fall back to the asset-handler path */ }
    }
    this.onDiagnostic?.(`model base: ${base}${base === FALLBACK_BASE ? ' (asset-handler fallback — loopback server unavailable)' : ' (loopback server)'}`)
    const worker = new Worker(new URL('../../workers/readAloud.worker.ts', import.meta.url), { type: 'module' })
    this.worker = worker
    this.becameReady = false
    const workerIndex = ++workerStarts
    this.onDiagnostic?.(`worker start #${workerIndex} (a growing count here means the pipeline is being rebuilt — it should stay at 1 per session)`)
    const readyPromise = new Promise<void>((resolve, reject) => {
      worker.onmessage = (event: MessageEvent) => {
        const msg = event.data
        switch (msg.type) {
          case 'diag':
            this.onDiagnostic?.(msg.line)
            break
          case 'init-progress':
            this.onLoadProgress?.({ file: msg.file, loaded: msg.loaded, total: msg.total })
            break
          case 'ready':
            this.becameReady = true
            this.onDevice?.(msg.device)
            resolve()
            break
          case 'audio': {
            const request = this.pending.get(msg.id)
            if (request) {
              this.pending.delete(msg.id)
              request.resolve({ samples: msg.samples, sampleRate: msg.sampleRate })
            }
            break
          }
          case 'error': {
            if (typeof msg.id === 'number') {
              const request = this.pending.get(msg.id)
              if (request) {
                this.pending.delete(msg.id)
                request.reject(new Error(msg.message))
              }
            } else if (!this.becameReady) {
              // Init-level failure: the model never loaded.
              reject(new Error(msg.message))
              this.shutdown()
            } else {
              // Post-ready failure: record it, keep the pipeline. Tearing
              // down here caused the rebuild-per-play loop.
              this.onDiagnostic?.(`worker reported: ${msg.message}`)
            }
            break
          }
        }
      }
      worker.onerror = (event: ErrorEvent) => {
        const detail = `${event.message || 'unknown error'} (${event.filename || '?'}:${event.lineno ?? '?'}:${event.colno ?? '?'})`
        this.onDiagnostic?.(`worker error event: ${detail}`)
        if (!this.becameReady) {
          reject(new Error(`Read Aloud worker failed: ${detail}`))
          this.shutdown()
        }
        // After ready: nested pthread-worker hiccups surface here; the
        // pipeline itself is intact, so never tear it down for these.
      }
    })

    const config = this.getConfig()
    worker.postMessage({ type: 'init', base, device: config.device, threads: config.threads, workerIndex })
    return readyPromise
  }

  synthesize(id: number, text: string, voice: string, speed: number): Promise<AudioChunk> {
    return this.ensureReady().then(() => new Promise<AudioChunk>((resolve, reject) => {
      const worker = this.worker
      if (!worker) {
        reject(new Error('Read Aloud worker is not running'))
        return
      }
      this.pending.set(id, { resolve, reject })
      worker.postMessage({ type: 'synthesize', id, text, voice, speed })
    }))
  }

  cancel(id: number): void {
    this.worker?.postMessage({ type: 'cancel', id })
    // The controller has already moved on; drop the local waiter quietly.
    this.pending.delete(id)
  }

  // Terminates the worker and unloads the model. The next synthesize() call
  // starts a fresh worker and reloads from the local bundle (picking up any
  // changed device/threads configuration).
  shutdown(): void {
    if (this.worker) {
      this.onDiagnostic?.('shutdown: disposing pipeline and terminating worker')
      this.worker.postMessage({ type: 'dispose' })
      // Give the dispose message a moment to release the ORT session before
      // hard-terminating; termination remains the backstop either way.
      const worker = this.worker
      setTimeout(() => worker.terminate(), 500)
      this.worker = null
    }
    this.ready = null
    for (const [, request] of this.pending) {
      request.reject(new Error('Read Aloud stopped'))
    }
    this.pending.clear()
  }
}

// Native sherpa-onnx synthesis uses the same controller contract as the
// browser worker, but inference runs in Draftline's loopback service through
// the platform library installed for Windows, macOS, or Linux. Responses are
// raw little-endian Float32 PCM; AbortController propagates skip/stop directly
// into sherpa's generation callback.
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
      if (!base) throw new Error('Read Aloud loopback service is unavailable')
      const config = this.getConfig()
      const response = await fetch(`${base}/readaloud-native/synthesize`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          text, voice, speed,
          threads: config.threads === 'single' ? 1 : 0,
        }),
        signal: abort.signal,
      })
      if (!response.ok) throw new Error((await response.text()) || `Native synthesis failed (${response.status})`)
      const sampleRate = Number(response.headers.get('X-Draftline-Sample-Rate'))
      if (!Number.isFinite(sampleRate) || sampleRate <= 0) throw new Error('Native synthesis returned no sample rate')
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
