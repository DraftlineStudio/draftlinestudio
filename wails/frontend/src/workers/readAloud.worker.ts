// Read Aloud synthesis worker. Loads kokoro-js (Kokoro-82M via
// transformers.js/ONNX) strictly from the locally installed bundle served by
// the Go asset handler — remote models are disabled, the onnxruntime .wasm
// files come from the same local path, and the library's hardcoded
// Hugging Face voice URLs are rewritten to local ones. After the one-time
// download nothing here can touch the network.
//
// Protocol (main → worker): init, synthesize, cancel, dispose.
// Protocol (worker → main): init-progress, ready, audio (transferred
// Float32Array), error.

const MODEL_ID = 'onnx-community/Kokoro-82M-v1.0-ONNX'
const HF_VOICES_PREFIX = `https://huggingface.co/${MODEL_ID}/resolve/main/voices/`

interface InitMessage { type: 'init'; base: string; preferredDevice: 'auto' | 'wasm' | 'webgpu' }
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
const cancelled = new Set<number>()
// Kokoro inference is not reentrant; requests run strictly in sequence.
let queue: Promise<void> = Promise.resolve()

function post(message: unknown, transfer?: Transferable[]) {
  ;(self as unknown as Worker).postMessage(message, transfer ?? [])
}

function redirectVoiceFetches(base: string) {
  const nativeFetch = self.fetch.bind(self)
  self.fetch = ((input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
    if (url.startsWith(HF_VOICES_PREFIX)) {
      return nativeFetch(`${base}/hf/${MODEL_ID}/voices/${url.slice(HF_VOICES_PREFIX.length)}`, init)
    }
    return nativeFetch(input as RequestInfo, init)
  }) as typeof fetch
}

async function loadModel(base: string, preferredDevice: InitMessage['preferredDevice']): Promise<void> {
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
    if (!self.crossOriginIsolated) {
      // Without cross-origin isolation there is no SharedArrayBuffer, and
      // the threaded wasm build must run single-threaded.
      wasm.numThreads = 1
    }
  }

  const progress = (p: { status?: string; file?: string; loaded?: number; total?: number }) => {
    if (p.status === 'progress' || p.status === 'ready') {
      post({ type: 'init-progress', file: p.file ?? '', loaded: p.loaded ?? 0, total: p.total ?? 0 })
    }
  }

  const wantGpu = preferredDevice !== 'wasm'
    && typeof (navigator as { gpu?: unknown }).gpu !== 'undefined'
  if (wantGpu) {
    try {
      const candidate = (await KokoroTTS.from_pretrained(MODEL_ID, {
        dtype: 'q8', device: 'webgpu', progress_callback: progress,
      })) as unknown as KokoroModel
      // Smoke-test: some WebGPU stacks load q8 but produce NaN or silence.
      const smoke = await candidate.generate('Hi.', { voice: 'af_heart', speed: 1 })
      const ok = smoke.audio.length > 0
        && Number.isFinite(smoke.audio[0])
        && smoke.audio.some(v => v !== 0)
      if (ok) {
        tts = candidate
        post({ type: 'ready', device: 'webgpu' })
        return
      }
    } catch {
      // Fall through to wasm.
    }
  }

  tts = (await KokoroTTS.from_pretrained(MODEL_ID, {
    dtype: 'q8', device: 'wasm', progress_callback: progress,
  })) as unknown as KokoroModel
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
      const audio = await tts.generate(msg.text, { voice: msg.voice, speed: msg.speed })
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
        initPromise = loadModel(msg.base, msg.preferredDevice).catch(e => {
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
