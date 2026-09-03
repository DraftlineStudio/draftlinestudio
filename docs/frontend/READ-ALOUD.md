# Read Aloud

Read Aloud is Draftline's opt-in, entirely local text-to-speech plugin. It uses
Kokoro through the native sherpa-onnx runtime. Browser WASM, WebGPU inference,
remote synthesis, and automatic fallback are deliberately unsupported.

## Installation

Enable Settings → Plugins → Read Aloud, then open Settings → Read Aloud and
install the voice model. Draftline downloads one checksum-pinned bundle for
the current platform:

- Windows x64
- Linux x64 or ARM64
- macOS Intel or Apple Silicon

Unsupported architectures report that native playback is unavailable. They do
not download another platform's binaries or fall back to browser or network
synthesis.

The platform-dependent download is approximately 370–385 MiB. It contains the
clean FP32 Kokoro model, voice table, English pronunciation data, eSpeak data, ONNX
Runtime, and sherpa C API. Every artifact has an immutable source revision,
expected byte count, and SHA-256 in wails/internal/readaloud/manifest.go.
Files are stored under UserCacheDir/draftline/models/kokoro/.

Install, repair, verification, and removal all operate on this one bundle.
Verification hashes every artifact and checks that eSpeak data extracted
successfully before declaring playback ready.

## Playback pipeline

Sentence and clause queue → authenticated loopback requests → one bounded hot
sherpa-onnx session → 24 kHz Float32 PCM → Web Audio scheduling and exact
sentence highlighting.

The local service binds only to 127.0.0.1 on an ephemeral port. Every launch
uses a fresh, unpersisted 128-bit capability path. The WebView sends manuscript
text to /readaloud-native/synthesize; the response exposes its sample rate
through X-Draftline-Sample-Rate and returns raw little-endian Float32 PCM.
No manuscript text or audio leaves the machine.

Auto threading uses one hot session with at most half of the machine's logical
CPUs, capped at six inference threads. Requests are serialized in manuscript
order because eSpeak phonemization has process-global state and concurrent
sessions can silently corrupt speech. The FP32 engine remains faster than real
time while ordered lookahead keeps later sentences ready. Single mode uses the
same session with one inference thread as a low-resource compatibility option.
The engine is released when the plugin is disabled or the model is removed.

Long sentences may split at natural clause boundaries for latency, but the UI
continues to highlight and navigate by full sentence. Playback waits for a
six-second contiguous runway (unless the selection ends sooner); generated
successors are scheduled on one Web Audio clock without inserting synthetic
transport gaps. Draftline appends a deliberate 220 ms rest after ordinary
sentences and 280 ms after questions or exclamations. Internal clause chunks
receive no sentence-level pause.

## Controls

| Action | Control |
|---|---|
| Open or close player | Bottom tools-rail Read Aloud icon |
| Read from cursor | Play or Ctrl+Shift+L |
| Read chapter | Read chapter |
| Pause or resume | Player or Ctrl+Shift+L |
| Previous/next sentence | Ctrl+Shift+, / Ctrl+Shift+. |
| Jump to sentence | Click prose while playback is active |
| Stop | Stop button, edit, chapter switch, or close player |

Voice and speed persist in settings. Diagnostics report native synthesis wall
time, audio duration, real-time factor, playback handoff gaps, and process RSS.

## Verification

- Go tests: go test ./...
- Frontend tests: npm test -- --run
- Production frontend: npm run build
- The gated TestNativeSynthesisRealBundle exercises the installed real model.

The Go backend is built with CGO_ENABLED=0. Cross-compilation covers every
listed Linux and macOS target; the real-model test exercises dynamic loading,
callback PCM, waveform integrity, and serialized sequential latency on Windows.

## File map

- Backend: wails/internal/readaloud/
- Wails bindings: wails/readaloud.go
- Frontend transport: wails/frontend/src/services/readaloud/tts.ts
- Queue: wails/frontend/src/services/readaloud/controller.ts
- Audio: wails/frontend/src/services/readaloud/audio.ts
- State: wails/frontend/src/store/readAloudStore.ts
- Highlighting: wails/frontend/src/extensions/ReadAloud.ts
