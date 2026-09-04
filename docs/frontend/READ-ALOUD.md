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

## Player

The docked player bar (bottom of the editor column) carries a speaker
equalizer and current-speaker readout, prev/play–pause/next/stop transport, a
click-to-seek progress bar with a diamond playhead (plus a 2px progress line
along the bar's top edge), the chapter/sentence/time-remaining readout, the
current voice chip, a cast-mode toggle, one-click speed cycling, and mute
with a draggable volume slider. Closing the bar leaves a floating reopen
button at the bottom-right of the editor. Narrow windows switch to a compact
readout automatically.

The expand chevron (or the voice chip) opens the expanded panel: a "Now
reading" strip with the current sentence in the speaker's color, the Voice
Cast section, and a Chapter Progress column with the larger seek bar, a
speaker legend, and time-left / elapsed / dialogue-line stats.

## Controls

| Action | Control |
|---|---|
| Open or close player | Bottom tools-rail Read Aloud icon, or the floating reopen button |
| Read from cursor | Play or Ctrl+Shift+L |
| Read chapter | Read chapter |
| Pause or resume | Player or Ctrl+Shift+L |
| Previous/next sentence | Transport buttons or Ctrl+Shift+, / Ctrl+Shift+. |
| Seek | Click the progress bar (bar or panel) |
| Jump to sentence | Click prose while playback is active |
| Speed | Speed button cycles 0.8×–2×; full list in Settings |
| Volume / mute | Slider and mute button (volume persists; mute is per-session) |
| Stop | Stop button, edit, chapter switch, or close player |

Voice, speed, volume, and the glow-accent toggle persist in settings; new
installations default to 1.1× speed. Time-left/elapsed figures are exact for
already-synthesized audio and self-calibrating estimates for the rest.
Diagnostics report native synthesis wall time, audio duration, real-time
factor, playback handoff gaps, and process RSS.

## Voice cast

Cast mode reads dialogue in per-character voices. Attribution is fully local
and deterministic — a quote-span tracker plus dialogue-tag parsing ("…,"
Marcus said / said Marcus / she said with pronoun resolution), action beats,
nearby-mention lookup, two-speaker alternation, and single-speaker monologue
continuation — over the roster of confirmed characters from the book's
Characters data. The Voice Cast section shows each detected speaker with
line counts; assign voices manually, per row via Cast voice, or all at once
with Auto-cast from Characters (deterministic, gender-aware from pronoun
evidence, never overwrites manual choices). Voice previews speak a short
sample while playback is paused or stopped.

Assignments and the cast-mode flag persist inside the .draftline file as an
optional `read_aloud_cast.json` member keyed by lowercased character name;
books that never cast are byte-identical to before. Changing a voice mid-play
lets the audible sentence finish and re-synthesizes everything after it.

In cast mode, sentences split at quote boundaries for synthesis: quoted
speech plays in the character's voice while tags and asides ("…," she said,
checking the hallway) stay with the narrator, audiobook-style. Tiny fragments
merge into their neighbor rather than being spoken alone.

Accuracy expectations: name-tagged dialogue attributes reliably. A gender
pre-pass learns each character's pronoun from narration co-reference, so
pronoun tags resolve against positive evidence (a nearby name of the wrong
gender can't take the line). Dialogue follows the one-speaker-per-paragraph
convention, and untagged paragraph hand-offs alternate between the two most
recent speakers. What still lands in the explicit "Unknown speaker" bucket
(read in the narrator's voice): dialogue with no tags, no nearby mentions,
and no alternation pattern to follow. Out of scope for now (read as
narration): em-dash dialogue and single-quote-delimited dialogue.

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
- Audio (master gain, one-shot previews): wails/frontend/src/services/readaloud/audio.ts
- Speaker attribution: wails/frontend/src/services/readaloud/attribution.ts
- Cast model / auto-cast: wails/frontend/src/services/readaloud/cast.ts
- Time estimates: wails/frontend/src/services/readaloud/estimates.ts
- Speed constants: wails/frontend/src/services/readaloud/speeds.ts
- State: wails/frontend/src/store/readAloudStore.ts
- Player UI: wails/frontend/src/components/editor/readaloud/
- Cast persistence: wails/internal/types/book.go (ReadAloudCast), wails/internal/book/{save,open}.go
- Highlighting: wails/frontend/src/extensions/ReadAloud.ts
