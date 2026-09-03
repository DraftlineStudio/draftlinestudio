// Web Audio implementation of the controller's AudioPort. Chunks are
// scheduled back-to-back on one AudioContext clock (nextStartTime), which is
// what makes sentence handoff gapless. Pause/resume suspend the context, so
// the schedule — and the onended callbacks — freeze with the audio.
//
// Every source routes through one master GainNode so volume and mute apply
// to scheduled playback and one-shot previews alike.

import type { AudioChunk, AudioPort, PlaybackHandle } from './controller'

// Short exponential ramp for volume/mute changes: fast enough to feel
// immediate, long enough to avoid a click.
const GAIN_RAMP_SECONDS = 0.015

export class WebAudioPort implements AudioPort {
  private ctx: AudioContext | null = null
  private gain: GainNode | null = null
  private nextStartTime = 0
  private active = new Set<AudioBufferSourceNode>()
  // One-shot previews live outside the gapless schedule and outside
  // stopAll(): stopping playback must not cut a preview short.
  private oneShots = new Set<AudioBufferSourceNode>()
  private allocatedBytes = 0
  private volume = 1
  private muted = false

  constructor(private onDiagnostic?: (line: string) => void) {}

  private ensureContext(): AudioContext {
    if (!this.ctx) {
      this.ctx = new AudioContext()
      this.gain = this.ctx.createGain()
      this.gain.gain.value = this.effectiveGain()
      this.gain.connect(this.ctx.destination)
      this.nextStartTime = 0
    }
    return this.ctx
  }

  private effectiveGain(): number {
    return this.muted ? 0 : this.volume
  }

  private applyGain(): void {
    if (!this.ctx || !this.gain) return
    this.gain.gain.setTargetAtTime(this.effectiveGain(), this.ctx.currentTime, GAIN_RAMP_SECONDS)
  }

  setVolume(volume: number): void {
    this.volume = Math.max(0, Math.min(1, volume))
    this.applyGain()
  }

  setMuted(muted: boolean): void {
    this.muted = muted
    this.applyGain()
  }

  // The context clock: advances while playing, freezes across suspend(), so
  // elapsed-time accounting pauses with the audio for free.
  now(): number {
    return this.ctx?.currentTime ?? 0
  }

  private makeSource(chunk: AudioChunk): AudioBufferSourceNode {
    const ctx = this.ensureContext()
    const buffer = ctx.createBuffer(1, chunk.samples.length, chunk.sampleRate)
    // The DOM lib insists on a non-shared ArrayBuffer; worker transfers
    // always produce one.
    buffer.copyToChannel(chunk.samples as Float32Array<ArrayBuffer>, 0)
    const source = ctx.createBufferSource()
    source.buffer = buffer
    source.connect(this.gain!)
    return source
  }

  enqueue(chunk: AudioChunk, onEnded: () => void): PlaybackHandle {
    const ctx = this.ensureContext()
    const source = this.makeSource(chunk)

    let stopped = false
    source.onended = () => {
      this.active.delete(source)
      if (!stopped) onEnded()
    }

    const previousEnd = this.nextStartTime
    const startAt = Math.max(ctx.currentTime, previousEnd)
    const handoffGapMs = previousEnd > 0 ? Math.max(0, startAt - previousEnd) * 1000 : 0
    this.nextStartTime = startAt + source.buffer!.duration
    source.start(startAt)
    this.active.add(source)
    this.allocatedBytes += chunk.samples.byteLength
    this.onDiagnostic?.(`audio alloc: ${Math.round(chunk.samples.byteLength / 1024)} KB buffer (${source.buffer!.duration.toFixed(1)}s), handoff gap ${handoffGapMs.toFixed(1)} ms, ${Math.round(this.allocatedBytes / 1048576)} MB total enqueued this session, ${this.active.size} scheduled`)

    return {
      stop: () => {
        stopped = true
        this.active.delete(source)
        try { source.stop() } catch { /* not started yet is fine */ }
      },
    }
  }

  // Plays a chunk immediately, outside the gapless schedule (voice preview).
  // Never touches nextStartTime, so scheduled playback is unaffected.
  playOneShot(chunk: AudioChunk, onEnded: () => void): PlaybackHandle {
    const ctx = this.ensureContext()
    void ctx.resume()
    const source = this.makeSource(chunk)
    let stopped = false
    source.onended = () => {
      this.oneShots.delete(source)
      if (!stopped) onEnded()
    }
    source.start(ctx.currentTime)
    this.oneShots.add(source)
    return {
      stop: () => {
        stopped = true
        this.oneShots.delete(source)
        try { source.stop() } catch { /* not started yet is fine */ }
      },
    }
  }

  pause(): void {
    void this.ctx?.suspend()
  }

  resume(): void {
    void this.ctx?.resume()
  }

  stopAll(): void {
    for (const source of [...this.active]) {
      source.onended = null
      try { source.stop() } catch { /* ignore */ }
    }
    this.active.clear()
    this.nextStartTime = 0
  }

  // Releases the audio device entirely (plugin disable).
  close(): void {
    this.stopAll()
    for (const source of [...this.oneShots]) {
      source.onended = null
      try { source.stop() } catch { /* ignore */ }
    }
    this.oneShots.clear()
    void this.ctx?.close()
    this.ctx = null
    this.gain = null
  }
}
