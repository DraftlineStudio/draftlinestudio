// Web Audio implementation of the controller's AudioPort. Chunks are
// scheduled back-to-back on one AudioContext clock (nextStartTime), which is
// what makes sentence handoff gapless. Pause/resume suspend the context, so
// the schedule — and the onended callbacks — freeze with the audio.

import type { AudioChunk, AudioPort, PlaybackHandle } from './controller'

export class WebAudioPort implements AudioPort {
  private ctx: AudioContext | null = null
  private nextStartTime = 0
  private active = new Set<AudioBufferSourceNode>()

  private ensureContext(): AudioContext {
    if (!this.ctx) {
      this.ctx = new AudioContext()
      this.nextStartTime = 0
    }
    return this.ctx
  }

  enqueue(chunk: AudioChunk, onEnded: () => void): PlaybackHandle {
    const ctx = this.ensureContext()
    const buffer = ctx.createBuffer(1, chunk.samples.length, chunk.sampleRate)
    // The DOM lib insists on a non-shared ArrayBuffer; worker transfers
    // always produce one.
    buffer.copyToChannel(chunk.samples as Float32Array<ArrayBuffer>, 0)
    const source = ctx.createBufferSource()
    source.buffer = buffer
    source.connect(ctx.destination)

    let stopped = false
    source.onended = () => {
      this.active.delete(source)
      if (!stopped) onEnded()
    }

    const startAt = Math.max(ctx.currentTime, this.nextStartTime)
    this.nextStartTime = startAt + buffer.duration
    source.start(startAt)
    this.active.add(source)

    return {
      stop: () => {
        stopped = true
        this.active.delete(source)
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
    void this.ctx?.close()
    this.ctx = null
  }
}
