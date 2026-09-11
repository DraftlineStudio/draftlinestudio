# Plugin System

Draftline treats built-in tools and future community extensions as plugins, but it does not yet execute third-party code. The first extensibility boundary is a **data-only analysis pack**: a signed manifest plus optional model weights consumed by a Draftline-owned analyzer runtime.

This distinction is deliberate. A downloaded model does not need arbitrary filesystem, shell, network, editor, or process access. Native or JavaScript code plugins require a separate permission sandbox and will not be accepted by the model-pack installer.

## Current foundation

- Bundled plugins declare stable IDs, capabilities, categories, and resource profiles in `frontend/src/features/registry.ts`.
- `draftline.story-analysis` provides local Prose v3 structure, pacing, readability, dialogue, keyword, and extractive-summary capabilities.
- Analysis results are rebuildable data in `analysis.json`; author-owned story-bible and plotting data are never overwritten.
- The shared analysis coordinator tracks `idle`, `stale`, `running`, `current`, and `error` states.
- Manuscript edits mark every dependent analyzer stale. After 15 seconds without another edit, analysis runs outside the typing path.
- Wails progress events identify the active analysis phase and chapter for the bottom status bar.
- `draftline.read-aloud` is the first shipped `optional-model` plugin and exercises the download pipeline this document plans: a pinned-revision manifest with per-file SHA-256 and byte counts (`wails/internal/readaloud/manifest.go`), streaming verification with cancel/resume, install under the application cache directory, and a loopback synthesis service (capability-token URL path, streamed PCM) rather than serving model files to the webview. See `docs/frontend/READ-ALOUD.md`.

## Analysis-pack manifest

The marketplace catalog and installer should exchange a manifest with these required fields:

```json
{
  "schema": 1,
  "id": "publisher.plugin-name",
  "name": "Plugin Name",
  "version": "publisher-defined-version",
  "publisher": "Publisher",
  "capabilities": ["analysis.coreference"],
  "runtime": "draftline-onnx-v1",
  "artifacts": [
    {
      "url": "https://huggingface.co/publisher/model/resolve/revision/model.onnx",
      "sha256": "...",
      "bytes": 123456789,
      "license": "Apache-2.0"
    }
  ],
  "resources": {
    "estimated_ram_bytes": 536870912,
    "hardware": "cpu",
    "languages": ["en"]
  },
  "permissions": ["manuscript.read"]
}
```

Model URLs must resolve to an immutable repository revision. The catalog must never point at a mutable `main` branch.

## Marketplace requirements

Before the Marketplace tab can install packages, the public catalog and client need:

1. Signed catalog metadata with key rotation and an application-pinned root key.
2. HTTPS download plus exact byte-count and SHA-256 verification.
3. Archive traversal protection and strict per-file/total-size limits.
4. Atomic install, health check, activation, rollback, and uninstall.
5. License, download size, expected peak memory, hardware, language, and permissions shown before consent.
6. Models stored under the Draftline application-data directory, not inside projects.
7. A single low-priority analysis worker with cancellation and memory limits.
8. No network access during inference; manuscript text never enters logs or crash reports.
9. Results tagged with plugin ID/version and invalidated when the analyzer changes.

## Planned lightweight packs

- Compact open-label entity recognition for objects, organizations, locations, clues, and story-specific entity types.
- Temporal-expression parsing for explicit dates, durations, and relative-time candidates.
- Optional coreference/event extraction only if its measured download, peak memory, and accuracy justify it.

Beat classification, foreshadowing/payoff, character knowledge, and story-bible facts must remain evidence-linked candidates that the author can confirm or reject. Plugins may propose derived facts; they may never silently rewrite author-owned data.
