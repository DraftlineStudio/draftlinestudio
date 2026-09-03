// SynthPort implementation over the readAloud worker. The worker (and with
// it kokoro-js and the ONNX runtime) is created lazily on the first
// synthesis request and stays alive across plays so the model loads once per
// session; shutdown() terminates it and frees the model memory. Device and
// threading come from user settings — WASM single-threaded q8 is the
// explicit cross-platform default, never autodetected.

import type { AudioChunk, SynthPort } from './controller'

const MODEL_BASE = '/readaloud-models'

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
  ) {}

  private ensureReady(): Promise<void> {
    // Single construction guard: concurrent callers (and every later
    // sentence) await this same promise — the pipeline is built exactly once
    // per WorkerSynth lifetime, torn down only by shutdown().
    if (this.ready) return this.ready
    const worker = new Worker(new URL('../../workers/readAloud.worker.ts', import.meta.url), { type: 'module' })
    this.worker = worker
    this.becameReady = false
    this.onDiagnostic?.(`worker start #${++workerStarts} (a growing count here means the pipeline is being rebuilt — it should stay at 1 per session)`)
    this.ready = new Promise<void>((resolve, reject) => {
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
    worker.postMessage({ type: 'init', base: MODEL_BASE, device: config.device, threads: config.threads })
    return this.ready
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
      this.worker.postMessage({ type: 'dispose' })
      this.worker.terminate()
      this.worker = null
    }
    this.ready = null
    for (const [, request] of this.pending) {
      request.reject(new Error('Read Aloud stopped'))
    }
    this.pending.clear()
  }
}
