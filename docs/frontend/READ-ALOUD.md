# Read Aloud

Read Aloud is Draftline's opt-in, fully local text-to-speech plugin. It reads the manuscript sentence by sentence with the **Kokoro-82M** neural voice model (Apache 2.0, `onnx-community/Kokoro-82M-v1.0-ONNX`), running entirely inside the app via `kokoro-js` (transformers.js/ONNX). **No manuscript text, audio, or telemetry ever leaves the machine.** The only network activity in the plugin's life is the one-time model download; playback is verifiably offline — remote model loading is disabled at the library level and every runtime file is served from local disk.

## Enabling and the model download

The plugin ships disabled. Enable it under **Settings → Plugins → Read Aloud**; voice, speed, and model management live under **Settings → Read Aloud**. While disabled, nothing loads: the synthesis worker, `kokoro-js`, and the ONNX runtime are all behind a dynamic import that only runs on the first play.

The base one-time download is **~130 MB** from pinned, immutable revisions. An optional native continuous-playback bundle is about **168 MB** (platform dependent):

| Part | Source | Size |
|---|---|---|
| Kokoro q8 model + tokenizer/config | Hugging Face, pinned commit `1939ad2a…` | ~92 MB |
| 8 English voices (`.bin` embeddings) | same pinned commit | ~4 MB |
| onnxruntime-web runtime (`.mjs`/`.wasm`, plain + jsep) | jsdelivr, exact npm version | ~33 MB |
| *(optional, GPU path only)* fp32 model `onnx/model.onnx` | same pinned commit | ~311 MB |
| *(optional, native CPU)* sherpa-compatible Kokoro int8 model + voices/language data | Hugging Face + sherpa release, pinned commits/checksums | ~147 MB |
| *(optional, native CPU)* current platform's ONNX Runtime + sherpa C API | official sherpa Go platform package v1.13.7 | ~21-36 MB |

Every file is verified against a SHA-256 and byte count hard-coded in `wails/internal/readaloud/manifest.go` before it is installed; a longer-than-pinned body or checksum mismatch is refused. Downloads stream with byte-level progress (`readaloud:progress` events), can be cancelled, and resume at file granularity. Files land in `<UserCacheDir>/draftline/models/kokoro/` (Windows: `%LOCALAPPDATA%\draftline\models\kokoro`) and survive restarts. **Nothing model-related is embedded in the executable** — the bundle contains only app code (including the kokoro-js *library* as a lazy chunk); no `.onnx`, `.bin`, or `.wasm` bytes.

**Install lifecycle:** every successful install writes a `manifest.json` beside the files (model id + pinned revision, bundle version, installed-at, per-file name/bytes/sha256/group). `VerifyReadAloudModel()` re-hashes everything against the compiled pins — it runs on plugin enable, when the settings section opens, and before the first playback of a session; playback refuses to load the model until it passes. Settings show **Not installed / Installed vX (N MB, verified) / Corrupt (N files)** with a **Repair** button that re-downloads only failing files (the installer skips hash-verified ones). **Remove downloaded model** stops playback, unloads the worker, clears the `kokoro-voices` Cache API bucket, deletes every file plus the manifest, and confirms the directory is actually gone (logged to diagnostics). The whole cycle — fresh → install → verify → corrupt-detect → repair → remove-empty — is exercised against real downloads by the gated `TestFreshInstallLifecycle` (`READALOUD_E2E=1`).

## Architecture

```
Go downloader (internal/readaloud: pinned manifest, sha256, cancel/resume)
        ↓ installs to UserCacheDir/draftline/models/kokoro
Wails AssetServer fallback Handler  →  GET /readaloud-models/*  (read-only, traversal-proof)
        ↓ fetched by
readAloud.worker.ts (module worker: kokoro-js + transformers.js, explicit wasm+q8)
        ↓ Float32Array chunks (transferred)
ReadAloudController (five-unit lookahead, duration reserve, gapless, generation-guarded)
        ↓ AudioChunks                     ↓ sentence index
WebAudioPort (one AudioContext clock)   ReadAloud extension (ProseMirror decoration)
```

- **Why bytes route through Go:** transformers.js normally caches models in the browser Cache API, whose persistence inside the Wails WebView is not guaranteed. Draftline instead disables the browser cache entirely (`env.useBrowserCache = false`, `env.allowRemoteModels = false`, `env.localModelPath` → the local handler) and always loads from disk through the asset handler — the first use of the Wails `assetserver.Options.Handler` slot. kokoro-js's hardcoded Hugging Face voice URLs are rewritten to local paths by a scoped fetch wrapper inside our own worker.
- **One transformers instance:** `@huggingface/transformers` is pinned in `package.json` to the exact version `kokoro-js` resolves, so the worker configures the same `env` kokoro uses (kokoro's own `env` re-export only forwards `wasmPaths`). The pinned onnxruntime-web files in the Go manifest must match this version — update both together.
- **Sentence pipeline:** `services/readaloud/segmentation.ts` (pure text → spans) + `docSentences.ts` (spans → ProseMirror positions, coalescing across mark boundaries). The browser producer requests five generation units ahead and playback begins after at least two contiguous units and eight seconds of finished audio are ready (or the remaining queue is fully synthesized). Ordinary sentences stay whole for natural prosody; only sentences over 80 words may split at clause boundaries. Ready chunks are scheduled back-to-back on the AudioContext clock; skip/jump cancel in-flight synthesis; voice/speed changes keep the audible sentence and rebuild only the lookahead; a generation counter makes stale audio unplayable (`controller.ts`). This queue is bounded intentionally: buffering cannot repair a backend whose sustained generation is slower than playback. kokoro-js's `stream()` still waits for a complete sentence before yielding audio and exposes no sentence timestamps for paragraph-sized calls. True incremental playback requires a native generated-audio callback rather than a larger browser queue.
- **Native sentence pipeline:** `NativeSynth` sends the same sentence units to the loopback-only Go service. The service uses sherpa's generated-audio callback, propagates cancellation into native inference, and returns raw 24 kHz Float32 PCM. Auto mode creates at most three independent hot sessions with two inference threads each (fewer on smaller CPUs), allowing the existing concurrent lookahead requests to make progress in parallel. Native playback admits the first completed sentence immediately; one ONNX session is never re-entered.
- **Portable native loading:** `internal/readaloud/native.go` mirrors the stable sherpa v1.13.7 C ABI and loads official platform libraries through pinned PureGo v0.11.0. Draftline's normal build keeps `CGO_ENABLED=0`; Windows x64, macOS x64/arm64, and Linux x64/arm64 use the same Go implementation and platform-specific loader. Unsupported targets report native unavailable and continue using the browser path.
- **Highlight:** `extensions/ReadAloud.ts` decorates the active sentence (`.read-aloud-current`), scrolls it into view, maps positions across edits, and reports clicks (jump) and document changes (stop). Decorations never touch content or undo history.

## Controls

The player is a **bar docked at the bottom of the editor column** (the bottom counterpart of the chapter find bar), opened from the **rail icon pinned at the very bottom of the tools sidebar**. While the plugin is disabled neither exists; while the voice model is missing the rail icon carries a setup badge and both it and the bar's "Set up" button deep-link to Settings → Read Aloud (the AI Studio setup pattern).

| Action | Where |
|---|---|
| Open / close the player bar | Rail icon (bottom of the tools sidebar), or the bar's × |
| Read from cursor | Bar play button, or **Ctrl+Shift+L** |
| Read chapter | "Read chapter" button in the bar when idle |
| Pause / resume | Bar, or **Ctrl+Shift+L** while active |
| Skip ±1 sentence | Bar, or **Ctrl+Shift+.** / **Ctrl+Shift+,** |
| Jump to a sentence | Click it in the editor while playing |
| Stop | Bar stop button; editing, switching chapters, or closing the bar also stops |
| Voice / speed (0.8×–1.6×, default 1.2×) | Bar dropdowns or Settings → Read Aloud; persisted |

## Segmentation rules

Sentence-final punctuation inside closing quotes (`"Go away." Then he left.`); `?!` clusters; abbreviation and single-capital-initial suppression (`Mr.`, `J. R. R.`); decimals; `No.` only before a number; ellipses continue when prose resumes lowercase and end before a capital; em dashes never terminate; block boundaries always do. Ambiguity leans toward *not* splitting — a missed split just reads two sentences in one breath. Tests: `segmentation.test.ts` (26 cases), `controller.test.ts` (15 state-machine cases with fake ports), plus Go download/handler/security tests in `internal/readaloud`.

## Device selection, threading & diagnostics

When the optional native bundle is installed, Draftline prefers **native CPU + sherpa int8** for bounded parallel lookahead. Otherwise device⇒dtype follows the browser policy: **CPU runs WASM + q8** and **GPU runs WebGPU + fp32**. The GPU path needs the optional full-precision model (~311 MB), downloaded on demand from Settings → Read Aloud → Performance; a load-time smoke synthesis guards it and playback falls back to CPU if WebGPU misbehaves.

**Threading:** the Wails asset server sets `Cross-Origin-Opener-Policy: same-origin` and `Cross-Origin-Embedder-Policy: require-corp` on every response, making the webview cross-origin isolated so the ONNX runtime gets SharedArrayBuffer and real WASM threads. Threads default to Auto = `hardwareConcurrency − 1` (capped at 8) when isolation is active, else 1; Single is the fallback option. Device/thread changes tear the worker down and apply on the next playback session.

**Serving path:** WebView2's custom-scheme asset handler does not reliably intercept requests initiated inside *nested* workers (the runtime's pthread workers), so model and runtime files are served from a real loopback HTTP origin — a Go `net/http` server on `127.0.0.1` (ephemeral port, started on demand, URL via the `ReadAloudServerURL()` binding) with permissive CORS, `Cross-Origin-Resource-Policy: cross-origin`, COEP, and explicit MIME types. Every launch places all routes behind a new unpersisted 128-bit capability path; requests that know only the local port receive 404, preventing unrelated local web content from invoking synthesis. If the server can't start, the frontend falls back to the same-origin `/readaloud-models/` asset-handler path (threading may be degraded there). The CPU path also pins the exact **non-jsep** runtime pair (`ort-wasm-simd-threaded.mjs/.wasm`) via a `wasmPaths` object — JSEP exists for WebGPU and isn't needed for pure-CPU inference.

**Run performance check** (Settings → Performance) times a steady-state sentence on each installed backend and persists the faster device — the cross-platform answer to "which config should this machine use". On the development machine (Ryzen 5 7535U, 6C/12T), measured CPU/WASM q8 generation took roughly 1.6–1.7× the resulting audio duration and therefore could not sustain uninterrupted playback indefinitely. Warm WebGPU fp32 generation measured roughly 0.56–0.92× audio duration after its one-time model load, so it can fill the reserve while playback continues. These figures describe that machine rather than a universal hardware guarantee.

The same machine's native sherpa session was slower than realtime in isolation (roughly 1.54–1.57 RTF with two threads). The bounded three-session/two-thread lookahead generated 16.81 seconds of mixed-length speech in 13.46 seconds, an aggregate RTF of 0.801. That is why native uses several small independent sessions: an oversized eleven-thread session regressed to roughly 1.8–1.9 RTF. Machines below eight logical CPUs use fewer sessions, so uninterrupted Kokoro playback still cannot be guaranteed on every low-power device.

Every model load logs a fixed diagnostic sequence — to the WebView console (`[readaloud]` prefix) and to the selectable, copyable Diagnostics readout in settings: `navigator.gpu` presence; resolved device and dtype; `crossOriginIsolated`; ONNX-runtime wasm `numThreads`/`simd`; each runtime `.wasm` file actually fetched; model load time; and per-sentence synthesis wall time for the first three sentences (a single "model loaded" line before them proves the model is held for the whole session, not reloaded per sentence).

## Limitations

- English voices only (eight bundled; the manifest pins each voice file).
- Native installation supports Windows x64, macOS x64/arm64, and Linux x64/arm64. Other targets retain browser CPU/WebGPU playback.
- Low-power CPUs may remain slower than realtime even with the bounded native pool; WebGPU remains preferable where it initializes cleanly.
- Native playback trades memory for continuous throughput because each pool member owns an ONNX session. The pool is capped at three sessions/six inference threads and is released before uninstall.
- `TestInstallRealBundle` (`READALOUD_E2E=1`) downloads the real bundle and verifies every pin against the live sources — run it after changing the manifest.

## File map

Go: `internal/readaloud/{manifest,download,status,handler,server,native,native_loader_*}.go`, bound methods in `wails/readaloud.go`, handler mounted in `main.go`. Frontend: `store/readAloudStore.ts`, `services/readaloud/{segmentation,docSentences,controller,tts,audio,voices}.ts`, `workers/readAloud.worker.ts`, `extensions/ReadAloud.ts`, `components/editor/ReadAloudBar.tsx` (docked player bar), the rail icon in `components/ToolsPanel.tsx`, settings in `components/dialogs/settings/ReadAloudSection.tsx`, styles under the `read-aloud-` prefix in `global.css`.
