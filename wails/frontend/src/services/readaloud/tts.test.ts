import { afterEach, describe, expect, it, vi } from 'vitest'
import { NativeSynth } from './tts'

function pcmResponse(samples = 24_000): Response {
  const pcm = new Float32Array(samples)
  return {
    ok: true,
    headers: { get: (name: string) => name === 'X-Draftline-Sample-Rate' ? '24000' : null },
    arrayBuffer: async () => pcm.buffer,
  } as unknown as Response
}

describe('NativeSynth', () => {
  const originalFetch = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = originalFetch
    vi.restoreAllMocks()
  })

  it('serializes lookahead requests so native phonemization cannot overlap', async () => {
    const releases: Array<(response: Response) => void> = []
    globalThis.fetch = vi.fn(() => new Promise<Response>(resolve => releases.push(resolve))) as typeof fetch
    const synth = new NativeSynth(() => ({ threads: 'auto' }), async () => 'http://127.0.0.1/token')

    const first = synth.synthesize(1, 'First.', 'af_heart', 1.2)
    const second = synth.synthesize(2, 'Second.', 'af_heart', 1.2)
    await Promise.resolve()
    await Promise.resolve()

    expect(globalThis.fetch).toHaveBeenCalledTimes(1)
    releases.shift()!(pcmResponse())
    await first
    await Promise.resolve()
    expect(globalThis.fetch).toHaveBeenCalledTimes(2)
    releases.shift()!(pcmResponse())
    await second
  })

  it('continues the queue after a cancelled request', async () => {
    globalThis.fetch = vi.fn((_url, init) => new Promise<Response>((resolve, reject) => {
	  if (init?.signal?.aborted) {
	    reject(new DOMException('cancelled', 'AbortError'))
	    return
	  }
      init?.signal?.addEventListener('abort', () => reject(new DOMException('cancelled', 'AbortError')))
	  setTimeout(() => resolve(pcmResponse()), 0)
    })) as typeof fetch
    const synth = new NativeSynth(() => ({ threads: 'auto' }), async () => 'http://127.0.0.1/token')

    const cancelled = synth.synthesize(1, 'Cancel me.', 'af_heart', 1.2)
    const next = synth.synthesize(2, 'Keep me.', 'af_heart', 1.2)
    synth.cancel(1)

    await expect(cancelled).rejects.toThrow()
    await expect(next).resolves.toMatchObject({ sampleRate: 24_000 })
  })
})
