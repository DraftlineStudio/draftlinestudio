# Read Aloud

Read Aloud is Draftline's opt-in, fully local text-to-speech plugin. It reads the manuscript sentence by sentence with the **Kokoro-82M** neural voice model (Apache 2.0, `onnx-community/Kokoro-82M-v1.0-ONNX`), running entirely inside the app via `kokoro-js` (transformers.js/ONNX). **No manuscript text, audio, or telemetry ever leaves the machine.** The only network activity in the plugin's life is the one-time model download; playback is verifiably offline — remote model loading is disabled at the library level and every runtime file is served from local disk.

## Enabling and the model download

The plugin ships disabled. Enable it under **Settings → Plugins → Read Aloud**; voice, speed, and model management live under **Settings → Read Aloud**. While disabled, nothing loads: the synthesis worker, `kokoro-js`, and the ONNX runtime are all behind a dynamic import that only runs on the first play.

The one-time download is **~130 MB** from pinned, immutable revisions:

| Part | Source | Size |
|---|---|---|
| Kokoro q8 model + tokenizer/config | Hugging Face, pinned commit `1939ad2a…` | ~92 MB |
| 8 English voices (`.bin` embeddings) | same pinned commit | ~4 MB |
| onnxruntime-web runtime (`.mjs`/`.wasm`, plain + jsep) | jsdelivr, exact npm version | ~33 MB |

Every file is verified against a SHA-256 and byte count hard-coded in `wails/internal/readaloud/manifest.go` before it is installed; a longer-than-pinned body or checksum mismatch is refused. Downloads stream with byte-level progress (`readaloud:progress` events), can be cancelled, and resume at file granularity. Files land in `<UserCacheDir>/draftline/models/kokoro/` (Windows: `%LOCALAPPDATA%\draftline\models\kokoro`) and survive restarts. **Remove downloaded model** in settings deletes the directory.

## Architecture

```
Go downloader (internal/readaloud: pinned manifest, sha256, cancel/resume)
        ↓ installs to UserCacheDir/draftline/models/kokoro
Wails AssetServer fallback Handler  →  GET /readaloud-models/*  (read-only, traversal-proof)
        ↓ fetched by
readAloud.worker.ts (module worker: kokoro-js + transformers.js, WebGPU→WASM probe)
        ↓ Float32Array chunks (transferred)
ReadAloudController (pure state machine: lookahead-1, gapless, generation-guarded)
        ↓ AudioChunks                     ↓ sentence index
WebAudioPort (one AudioContext clock)   ReadAloud extension (ProseMirror decoration)
```

- **Why bytes route through Go:** transformers.js normally caches models in the browser Cache API, whose persistence inside the Wails WebView is not guaranteed. Draftline instead disables the browser cache entirely (`env.useBrowserCache = false`, `env.allowRemoteModels = false`, `env.localModelPath` → the local handler) and always loads from disk through the asset handler — the first use of the Wails `assetserver.Options.Handler` slot. kokoro-js's hardcoded Hugging Face voice URLs are rewritten to local paths by a scoped fetch wrapper inside our own worker.
- **One transformers instance:** `@huggingface/transformers` is pinned in `package.json` to the exact version `kokoro-js` resolves, so the worker configures the same `env` kokoro uses (kokoro's own `env` re-export only forwards `wasmPaths`). The pinned onnxruntime-web files in the Go manifest must match this version — update both together.
- **Sentence pipeline:** `services/readaloud/segmentation.ts` (pure text → spans) + `docSentences.ts` (spans → ProseMirror positions, coalescing across mark boundaries). While sentence N plays, N+1 synthesizes and is scheduled back-to-back on the AudioContext clock; skip/jump cancel in-flight synthesis; voice/speed changes keep the audible sentence and rebuild only the lookahead; a generation counter makes stale audio unplayable (`controller.ts`).
- **Highlight:** `extensions/ReadAloud.ts` decorates the active sentence (`.read-aloud-current`), scrolls it into view, maps positions across edits, and reports clicks (jump) and document changes (stop). Decorations never touch content or undo history.

## Controls

| Action | Where |
|---|---|
| Read selection / from cursor | Toolbar speaker button, or **Ctrl+Shift+L** |
| Read chapter | Play button in the floating player when idle |
| Pause / resume | Player, or **Ctrl+Shift+L** while active |
| Skip ±1 sentence | Player, or **Ctrl+Shift+.** / **Ctrl+Shift+,** |
| Jump to a sentence | Click it in the editor while playing |
| Stop | Player stop button; editing, switching chapters, or closing the player also stops |
| Voice / speed (0.8×–1.6×, default 1.2×) | Player dropdowns or Settings → Read Aloud; persisted |

## Segmentation rules

Sentence-final punctuation inside closing quotes (`"Go away." Then he left.`); `?!` clusters; abbreviation and single-capital-initial suppression (`Mr.`, `J. R. R.`); decimals; `No.` only before a number; ellipses continue when prose resumes lowercase and end before a capital; em dashes never terminate; block boundaries always do. Ambiguity leans toward *not* splitting — a missed split just reads two sentences in one breath. Tests: `segmentation.test.ts` (26 cases), `controller.test.ts` (15 state-machine cases with fake ports), plus Go download/handler/security tests in `internal/readaloud`.

## Device selection

If WebGPU is available the worker loads the q8 model on it and smoke-tests one utterance (some WebGPU stacks produce NaN/silence); on any failure it falls back to WASM and remembers the choice (`localStorage['draftline.readaloud.device']`). Without cross-origin isolation the WASM path runs single-threaded. Expectations: Windows WebView2 — WebGPU likely; macOS WKWebView and Linux WebKitGTK — usually WASM. The WASM fallback is the baseline the feature is built against.

## Limitations

- English voices only (eight bundled; the manifest pins each voice file).
- Synthesis is roughly real-time on the WASM path; very long sentences take proportionally longer to start.
- Playback memory while active is a few hundred MB (ONNX inference); disabling the plugin terminates the worker and releases it.
- `TestInstallRealBundle` (`READALOUD_E2E=1`) downloads the real bundle and verifies every pin against the live sources — run it after changing the manifest.

## File map

Go: `internal/readaloud/{manifest,download,status,handler}.go`, bound methods in `wails/readaloud.go`, handler mounted in `main.go`. Frontend: `store/readAloudStore.ts`, `services/readaloud/{segmentation,docSentences,controller,tts,audio,voices}.ts`, `workers/readAloud.worker.ts`, `extensions/ReadAloud.ts`, `components/editor/ReadAloudPlayer.tsx`, settings in `components/dialogs/settings/ReadAloudSection.tsx`, styles under the `read-aloud-` prefix in `global.css`.
