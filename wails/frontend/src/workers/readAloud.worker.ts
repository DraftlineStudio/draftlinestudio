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
  workerIndex: number
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
let sentenceCount = 0
let phonemizeTimed = false
let workerIndex = 0
const cancelled = new Set<number>()
// Kokoro inference is not reentrant; requests run strictly in sequence.
let queue: Promise<void> = Promise.resolve()

function post(message: unknown, transfer?: Transferable[]) {
  ;(self as unknown as Worker).postMessage(message, transfer ?? [])
}

// Diagnostics go to the WebView console AND to the main thread so the
// settings panel can show them without devtools. Each line carries a
// timestamp and the worker-start counter so memory numbers are attributable.
function diag(line: string) {
  const stamped = `[w${workerIndex} t+${Math.round(performance.now())}ms] ${line}`
  console.log('[readaloud]', stamped)
  post({ type: 'diag', line: stamped })
}

// ── Memory instrumentation + the 4 GB clamp ─────────────────────────────────
// The threaded ONNX runtime creates its WebAssembly.Memory as
// {initial:256, maximum:65536(=4 GB), shared:true}. Growable SHARED memories
// cannot relocate their buffer, so the engine reserves — and on Windows
// commits — the full maximum eagerly, and every extra instantiation stacks
// another 4 GB that never shrinks. Kokoro q8 needs a few hundred MB, so the
// constructor is wrapped to cap every large wasm memory at 1 GB. Growth past
// the cap raises a RangeError inside generate(), which surfaces through the
// normal error path instead of eating the machine.
const MEMORY_CAP_PAGES = 16384 // × 64 KiB = 1 GiB
// WeakRefs: instrumentation must never be the thing keeping an abandoned
// instance's memory alive. (Local declaration — the project TS lib predates
// ES2021, but every supported WebView ships WeakRef.)
declare const WeakRef: new <T extends object>(target: T) => { deref(): T | undefined }
const trackedMemories: { deref(): WebAssembly.Memory | undefined }[] = []
{
  const NativeMemory = WebAssembly.Memory
  const Patched = function (this: unknown, descriptor: WebAssembly.MemoryDescriptor) {
    const requested = descriptor?.maximum
    if (descriptor && typeof requested === 'number' && requested > MEMORY_CAP_PAGES) {
      descriptor = { ...descriptor, maximum: MEMORY_CAP_PAGES }
      diag(`wasm memory clamped: requested max ${Math.round(requested / 16384)} GiB → 1 GiB (initial ${descriptor.initial} pages, shared ${descriptor.shared === true})`)
    }
    const memory = new NativeMemory(descriptor)
    if (descriptor && ((descriptor.maximum ?? 0) >= 4096 || descriptor.shared)) {
      trackedMemories.push(new WeakRef(memory))
    }
    return memory
  }
  Patched.prototype = NativeMemory.prototype
  ;(self as unknown as { WebAssembly: { Memory: unknown } }).WebAssembly.Memory =
    Patched as unknown as typeof WebAssembly.Memory
}

function memorySnapshot(label: string) {
  const heap = (performance as unknown as { memory?: { usedJSHeapSize: number } }).memory
  let wasmBytes = 0
  let live = 0
  for (const ref of trackedMemories) {
    const memory = ref.deref()
    if (!memory) continue
    live++
    try { wasmBytes += memory.buffer.byteLength } catch { /* detached */ }
  }
  diag(`mem[${label}]: jsHeap ${heap ? Math.round(heap.usedJSHeapSize / 1048576) + ' MB' : 'n/a'}, wasm ${Math.round(wasmBytes / 1048576)} MB across ${live} live memories`)
}

// Surface uncaught failures — including errors bubbling up from the nested
// pthread workers the ONNX runtime spawns — with real detail instead of the
// useless "[object Event]" a bare toString produces.
self.addEventListener('error', (event: ErrorEvent) => {
  diag(`worker error: ${event.message || String(event.error)} (${event.filename || '?'}:${event.lineno ?? '?'}:${event.colno ?? '?'})`)
})
self.addEventListener('unhandledrejection', (event: PromiseRejectionEvent) => {
  const r = event.reason
  diag(`worker unhandled rejection: ${r instanceof Error ? `${r.message}\n${r.stack ?? ''}` : String(r)}`)
})

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
    if (device === 'wasm') {
      // Pure-CPU inference needs no JSEP (that build exists for WebGPU);
      // pinning the exact non-jsep pair avoids the jsep loader's extra
      // machinery in its nested pthread workers.
      wasm.wasmPaths = {
        mjs: `${base}/ort/ort-wasm-simd-threaded.mjs`,
        wasm: `${base}/ort/ort-wasm-simd-threaded.wasm`,
      } as unknown as string
      diag('ort runtime: non-jsep pair (ort-wasm-simd-threaded.mjs/.wasm)')
    } else {
      wasm.wasmPaths = `${base}/ort/`
      diag('ort runtime: jsep directory (webgpu)')
    }
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
      if (!ok) {
        // Release the bad candidate's session before falling back — an
        // undisposed instance pins its whole wasm memory forever.
        await disposeModel(candidate)
        throw new Error('webgpu produced invalid audio in the smoke test')
      }
      tts = candidate
      diag(`model loaded in ${Math.round(performance.now() - started)} ms (webgpu fp32)`)
      memorySnapshot('after webgpu construct')
      post({ type: 'ready', device: 'webgpu' })
      return
    } catch (e) {
      diag(`webgpu fp32 failed (${e instanceof Error ? e.message : String(e)}) — falling back to wasm q8`)
      resolvedDevice = 'wasm'
      dtype = 'q8'
    }
  }

  const loadWasm = async () => {
    const started = performance.now()
    tts = (await KokoroTTS.from_pretrained(MODEL_ID, {
      dtype, device: 'wasm', progress_callback: progress,
    })) as unknown as KokoroModel
    diag(`pipeline constructed in ${Math.round(performance.now() - started)} ms (wasm q8, threads ${wasm?.numThreads ?? '?'}) — this line must appear once per session`)
    memorySnapshot('after wasm construct')
  }
  try {
    await loadWasm()
  } catch (e) {
    // The threaded runtime can fail where single-threaded works (nested
    // pthread workers are pickier about headers/environment). Retry once
    // before giving up so a threading problem degrades, not breaks.
    if (wasm && wasm.numThreads !== 1) {
      diag(`threaded wasm init failed (${e instanceof Error ? e.message : String(e)}) — retrying with numThreads 1`)
      wasm.numThreads = 1
      await loadWasm()
    } else {
      throw e
    }
  }
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
      // One-off phonemization timing (kokoro re-phonemizes inside
      // generate(); a single extra call isolates that cost honestly).
      if (!phonemizeTimed) {
        phonemizeTimed = true
        try {
          const { phonemize } = await import('phonemizer')
          const t0 = performance.now()
          await phonemize(msg.text, 'en-us')
          diag(`phonemization: ${Math.round(performance.now() - t0)} ms for ${msg.text.length} chars`)
        } catch (e) {
          diag(`phonemization timing unavailable: ${e instanceof Error ? e.message : String(e)}`)
        }
      }
      const started = performance.now()
      const audio = await tts.generate(msg.text, { voice: msg.voice, speed: msg.speed })
      // Per-sentence timing. Exactly one "pipeline constructed" line ever
      // appearing before these proves the model is held across sentences,
      // not rebuilt per play. RTF = generate seconds / audio seconds;
      // < 0.5 means synthesis runs at least twice as fast as playback.
      {
        sentenceCount++
        const ms = performance.now() - started
        const audioSec = audio.audio.length / audio.sampling_rate
        const rtf = audioSec > 0 ? ms / 1000 / audioSec : 0
        diag(`sentence ${sentenceCount}: generate ${Math.round(ms)} ms, ${msg.text.length} chars → ${audioSec.toFixed(1)}s audio, RTF ${rtf.toFixed(2)}`)
        if (sentenceCount <= 3 || sentenceCount % 10 === 0) memorySnapshot(`after sentence ${sentenceCount}`)
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

// Releases the transformers model's ORT session explicitly — worker
// termination frees the heap eventually, but an explicit release makes the
// RSS drop observable immediately.
async function disposeModel(model: KokoroModel | null) {
  if (!model) return
  try {
    await (model as unknown as { model?: { dispose?: () => Promise<void> } }).model?.dispose?.()
    diag('pipeline disposed (session released)')
  } catch (e) {
    diag(`dispose warning: ${e instanceof Error ? e.message : String(e)}`)
  }
}

self.onmessage = (event: MessageEvent<InMessage>) => {
  const msg = event.data
  switch (msg.type) {
    case 'init':
      workerIndex = msg.workerIndex
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
    case 'dispose': {
      const model = tts
      tts = null
      void disposeModel(model).then(() => {
        memorySnapshot('after dispose')
        self.close()
      })
      break
    }
  }
}
