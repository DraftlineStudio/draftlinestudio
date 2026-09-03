// Read Aloud synthesis worker. Loads kokoro-js (Kokoro-82M via
// transformers.js/ONNX) strictly from the locally installed bundle served by
// the Go asset handler — remote models are disabled, the onnxruntime .wasm
// files come from the same local path, and the library's hardcoded
// Hugging Face voice URLs are rewritten to local ones. After the one-time
// download nothing here can touch the network.
//
// Device policy: WASM + q8 is the explicit default on every platform — no
// autodetection. WebGPU is opt-in via settings (it produced corrupted audio
// on real hardware when chosen automatically). Every load logs a fixed
// diagnostic sequence so cross-platform issues are debuggable from the
// WebView console and the settings panel.
//
// Protocol (main → worker): init, synthesize, cancel, dispose.
// Protocol (worker → main): diag, init-progress, ready, audio (transferred
// Float32Array), error.

const MODEL_ID = 'onnx-community/Kokoro-82M-v1.0-ONNX'
const HF_VOICES_PREFIX = `https://huggingface.co/${MODEL_ID}/resolve/main/voices/`

interface InitMessage {
  type: 'init'
  base: string
  device: 'wasm' | 'webgpu'
  threads: 'single' | 'auto'
}
interface SynthesizeMessage { type: 'synthesize'; id: number; text: string; voice: string; speed: number }
interface CancelMessage { type: 'cancel'; id: number }
interface DisposeMessage { type: 'dispose' }
type InMessage = InitMessage | SynthesizeMessage | CancelMessage | DisposeMessage

// kokoro-js has no bundled types; the surface we use is tiny.
interface KokoroModel {
  generate(text: string, options: { voice: string; speed: number }): Promise<{ audio: Float32Array; sampling_rate: number }>
}

let tts: KokoroModel | null = null
let initPromise: Promise<void> | null = null
let timedSentences = 0
const cancelled = new Set<number>()
// Kokoro inference is not reentrant; requests run strictly in sequence.
let queue: Promise<void> = Promise.resolve()

function post(message: unknown, transfer?: Transferable[]) {
  ;(self as unknown as Worker).postMessage(message, transfer ?? [])
}

// Diagnostics go to the WebView console AND to the main thread so the
// settings panel can show them without devtools.
function diag(line: string) {
  console.log('[readaloud]', line)
  post({ type: 'diag', line })
}

function redirectVoiceFetches(base: string) {
  const nativeFetch = self.fetch.bind(self)
  self.fetch = ((input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
    if (url.startsWith(HF_VOICES_PREFIX)) {
      return nativeFetch(`${base}/hf/${MODEL_ID}/voices/${url.slice(HF_VOICES_PREFIX.length)}`, init)
    }
    if (url.includes('.wasm') || url.includes('/ort/')) {
      diag(`runtime fetch: ${url}`)
    }
    return nativeFetch(input as RequestInfo, init)
  }) as typeof fetch
}

async function loadModel(base: string, device: InitMessage['device'], threads: InitMessage['threads']): Promise<void> {
  redirectVoiceFetches(base)
  // kokoro-js's own `env` re-export is a narrow facade (wasmPaths only), but
  // it consumes @huggingface/transformers as an external import, so
  // configuring the shared transformers env instance directly governs every
  // model/tokenizer fetch kokoro makes. package.json pins the same
  // transformers version kokoro-js resolves — one instance, one env.
  const [{ KokoroTTS }, { env }] = await Promise.all([
    import('kokoro-js'),
    import('@huggingface/transformers'),
  ])

  env.allowRemoteModels = false
  env.allowLocalModels = true
  env.localModelPath = `${base}/hf/`
  // No reliance on Cache API persistence in the WebView: bytes always come
  // from disk through the Go asset handler.
  env.useBrowserCache = false
  const wasm = env.backends.onnx.wasm
  if (wasm) {
    wasm.wasmPaths = `${base}/ort/`
    // Real threads need SharedArrayBuffer, which needs cross-origin
    // isolation (the Wails asset server sets COOP/COEP for exactly this).
    // Leave one core for the UI and audio pipeline.
    const cores = Math.max(1, (navigator.hardwareConcurrency || 2) - 1)
    wasm.numThreads = threads === 'auto' && self.crossOriginIsolated ? Math.min(cores, 8) : 1
  }

  // Fixed diagnostic sequence — keep the order stable; users report these.
  const gpuPresent = typeof (navigator as { gpu?: unknown }).gpu !== 'undefined'
  diag(`navigator.gpu present: ${gpuPresent}`)
  let resolvedDevice: 'wasm' | 'webgpu' = device
  if (device === 'webgpu' && !gpuPresent) {
    diag('webgpu requested but navigator.gpu is absent — falling back to wasm')
    resolvedDevice = 'wasm'
  }
  // Device⇒dtype is fixed policy: WASM runs the q8 model (fp32 in WASM is
  // roughly double the work for no audible gain); WebGPU runs fp32 (the
  // quantized variants are what produced corrupted audio there).
  let dtype: 'q8' | 'fp32' = resolvedDevice === 'webgpu' ? 'fp32' : 'q8'
  diag(`device: ${resolvedDevice} (requested ${device}), dtype: ${dtype}`)
  diag(`crossOriginIsolated: ${self.crossOriginIsolated === true}`)
  const wasmAny = wasm as unknown as { numThreads?: number; simd?: boolean } | undefined
  diag(`ort wasm numThreads: ${wasmAny?.numThreads ?? 'default'}, simd: ${wasmAny?.simd ?? 'default'}`)

  const progress = (p: { status?: string; file?: string; loaded?: number; total?: number }) => {
    if (p.status === 'progress' || p.status === 'ready') {
      post({ type: 'init-progress', file: p.file ?? '', loaded: p.loaded ?? 0, total: p.total ?? 0 })
    }
  }

  if (resolvedDevice === 'webgpu') {
    try {
      const started = performance.now()
      const candidate = (await KokoroTTS.from_pretrained(MODEL_ID, {
        dtype, device: 'webgpu', progress_callback: progress,
      })) as unknown as KokoroModel
      // Some WebGPU stacks initialize but render garbage; verify one
      // utterance before trusting it.
      const smoke = await candidate.generate('Hi.', { voice: 'af_heart', speed: 1 })
      const ok = smoke.audio.length > 0
        && smoke.audio.every(v => Number.isFinite(v))
        && smoke.audio.some(v => v !== 0)
      if (!ok) throw new Error('webgpu produced invalid audio in the smoke test')
      tts = candidate
      diag(`model loaded in ${Math.round(performance.now() - started)} ms (webgpu fp32)`)
      post({ type: 'ready', device: 'webgpu' })
      return
    } catch (e) {
      diag(`webgpu fp32 failed (${e instanceof Error ? e.message : String(e)}) — falling back to wasm q8`)
      resolvedDevice = 'wasm'
      dtype = 'q8'
    }
  }

  const started = performance.now()
  tts = (await KokoroTTS.from_pretrained(MODEL_ID, {
    dtype, device: 'wasm', progress_callback: progress,
  })) as unknown as KokoroModel
  diag(`model loaded in ${Math.round(performance.now() - started)} ms (wasm q8)`)
  post({ type: 'ready', device: 'wasm' })
}

function handleSynthesize(msg: SynthesizeMessage) {
  queue = queue.then(async () => {
    if (cancelled.delete(msg.id)) return
    if (!tts) {
      post({ type: 'error', id: msg.id, message: 'Voice model is not loaded' })
      return
    }
    try {
      const started = performance.now()
      const audio = await tts.generate(msg.text, { voice: msg.voice, speed: msg.speed })
      // Time the first few sentences. Exactly one "model loaded" line ever
      // appearing before these proves the model is held across sentences,
      // not reloaded per play. The audio-seconds ratio shows real-time
      // headroom (>1× means synthesis outruns playback).
      if (timedSentences < 3) {
        timedSentences++
        const ms = performance.now() - started
        const audioSec = audio.audio.length / audio.sampling_rate
        const ratio = ms > 0 ? (audioSec * 1000) / ms : 0
        diag(`sentence ${timedSentences}: ${Math.round(ms)} ms for ${msg.text.length} chars → ${audioSec.toFixed(1)}s audio (${ratio.toFixed(1)}× real-time)`)
      }
      if (cancelled.delete(msg.id)) return
      const samples = audio.audio
      post({ type: 'audio', id: msg.id, samples, sampleRate: audio.sampling_rate }, [samples.buffer])
    } catch (e) {
      if (cancelled.delete(msg.id)) return
      post({ type: 'error', id: msg.id, message: e instanceof Error ? e.message : String(e) })
    }
  })
}

self.onmessage = (event: MessageEvent<InMessage>) => {
  const msg = event.data
  switch (msg.type) {
    case 'init':
      if (!initPromise) {
        initPromise = loadModel(msg.base, msg.device, msg.threads).catch(e => {
          initPromise = null
          post({ type: 'error', message: e instanceof Error ? e.message : String(e) })
        })
      }
      break
    case 'synthesize':
      handleSynthesize(msg)
      break
    case 'cancel':
      cancelled.add(msg.id)
      break
    case 'dispose':
      tts = null
      self.close()
      break
  }
}
