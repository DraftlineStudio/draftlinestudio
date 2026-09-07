# Backend Guide (Go)

The Draftline backend is written in Go and uses [Wails v2](https://wails.io/) to bridge with the React frontend.

## Package Structure

```
wails/
├── main.go                    # Application entry point
├── app.go                     # App struct facade (~1,470 lines; guarded against growth)
├── analysis.go                # Full local-analysis pipeline orchestration + progress
├── story_search.go            # Thin Story Search Wails binding
├── fileopen.go                # OS file-open plumbing (see FILE-ASSOCIATIONS.md)
├── fileassoc_windows.go       # Per-user HKCU association self-registration
├── import.go                  # EPUB/DOCX import pipeline + routing (see import/IMPORT.md)
├── import_sanitize.go         # Import decoder, XHTML sanitizer, chaptering
├── debt_guardrail_test.go     # 800-line file-size ratchet (fails `go test` when a source file outgrows its allowance)
├── setup.go                   # Claude Code + Node.js setup
├── hidewindow_windows.go      # Windows-specific process hiding
├── hidewindow_other.go        # No-op for non-Windows
│
└── internal/
    ├── types/                 # Shared data structures
    │   ├── book.go            # BookData, ChapterItem, Metadata
    │   ├── characters.go      # Character, StoryBible
    │   ├── structure.go       # Beat, Foreshadowing, KnowledgeMatrix
    │   ├── settings.go        # AppSettings, ClaudeCodeStatus
    │   ├── export.go          # ExportOptions, PDFOptions, PrintPDFOptions
    │   ├── evidence.go        # Persistent source-located fact/event records
    │   └── results.go         # SaveResult, ExportResult, AIRewriteResult
    │
    ├── book/                  # Book lifecycle operations
    │   ├── open.go            # Open .draftline files
    │   └── save.go            # Write .draftline files
    │
    ├── backup/                # Automatic backup system
    │   └── backup.go          # Create, List, Restore backups
    │
    ├── export/                # Multi-format export
    │   ├── helpers.go         # HTML parsing, escaping utilities
    │   ├── epub.go            # EPUB 3.0 export
    │   ├── docx.go            # Microsoft Word export
    │   ├── pdf.go             # Standard PDF export
    │   └── print.go           # Print-ready PDF (bleed, crop marks)
    │
    ├── ai/                    # AI rewriting system
    │   ├── prompt.go          # System prompt building
    │   ├── diff.go            # Paragraph diff format parsing
    │   └── providers/         # AI HTTP transport (Anthropic/OpenAI-compat/Gemini
    │                          # + pure CLI helpers); app.go injects an Emit closure
    │
    ├── indexing/              # Local manuscript analysis
    │   ├── patterns.go        # Common words, regex patterns
    │   ├── characters.go      # Name detection, attribute extraction
    │   ├── evidence.go        # Persistent fact/event candidate extraction
    │   └── indexer.go         # Book/chapter indexing
    │
    ├── storysearch/           # Local detail trails + confirmed alias expansion
    │   ├── insight.go          # Query intent, fingerprint summaries, signals
    │   └── search.go           # Wails-independent source search engine
    │
    └── logging/               # Debug logging
        └── debug.go           # AI operation logging
```

## The App Struct

The App struct is a thin facade that delegates to internal packages:

```go
type App struct {
    ctx         context.Context
    currentFile string
    settings    types.AppSettings
    cancelMu    sync.Mutex
    cancelFunc  context.CancelFunc
}
```

### Delegation Pattern

App methods are thin wrappers that call into internal packages:

```go
func (a *App) SaveBook(b types.BookData) types.SaveResult {
    if a.currentFile == "" {
        return a.SaveBookAs(b)
    }
    return a.writeBook(b, a.currentFile)
}

func (a *App) writeBook(b types.BookData, path string) types.SaveResult {
    result := book.Write(path, b, AppVersion)
    if result.Success {
        a.currentFile = path
    }
    return result
}
```

## Internal Packages

### [types/](types/TYPES.md)
All shared data structures used across packages. Zero dependencies.

### [book/](book/BOOK.md)
Reading and writing `.draftline` project files (ZIP archives).

### [backup/](backup/BACKUP.md)
Automatic backup creation before saves. Keeps last 10 backups per file.

### [export/](export/EXPORT.md)
Multi-format export: EPUB, DOCX, PDF, and print-ready PDF.

### [ai/](../ai/AI-REWRITING.md)
AI prompt building and response parsing. Since 0.16.02468 the HTTP provider transport lives in `ai/providers` (`Request` carries the resolved model, API key, settings, and an `Emit` closure, so the package never imports the Wails runtime). The Claude Code / Codex CLI drivers stay in app.go alongside setup.go's exec helpers; their pure helpers (arg building, prompt-safe failure messages) are in `ai/providers/cli.go`.

### [indexing/](indexing/INDEXING.md)
Character name detection, attribute extraction, relationships, story metrics,
and the persistent fact/event evidence index. The evidence pass uses the
bundled prose/v3 model plus conservative deterministic cues; it stores exact
source sentences and coordinates in `analysis.json`, performs no network call,
and is rebuildable from the manuscript.

See [Analysis Concurrency and Performance](ANALYSIS-PERFORMANCE.md) for the
stage-local pool design, large-manuscript memory gate, and measured 12-thread
responsiveness results.

See [Narrative Fingerprint Pipeline](NARRATIVE-FINGERPRINT.md) for the
evidence/assertion/promotion boundary, epistemic and reality-scope model, and
the plain-text semantic diagnostic used before higher-level visualization.

Build 02476 adds author-owned review provenance without modifying source
evidence. Confirm/reject status, a separate author interpretation, notes, pins,
and review timestamps follow stable evidence IDs through reanalysis. The
frontend derives a bounded local review queue from high-value and suspicious
records while the complete archive remains available for inspection.

Build 02484 upgrades the evidence engine to `prose-v3-evidence-v2`. Evidence
records can now carry sentence-local knowledge claims for learning, knowing,
explicitly not knowing, attempting to recall, belief, suspicion, communication,
and withholding. Claims require a confirmed named character in a conservative
local grammatical position; Draftline does not resolve pronouns or distant
clause subjects by guesswork. Existing evidence classifications remain stable
so the upgrade does not orphan compatible author review decisions.

### [Story Structure](STORY-STRUCTURE.md)
The rebuildable semantic hierarchy over the lossless fingerprint, including
fixture-gated significant-event aggregation and durable author decisions.

### storytimeline/
Deterministic projection of the persistent evidence index into a manuscript-
order event trail. It merges classifications from the same source sentence,
excludes rejected and ordinary non-timed facts, retains confirmed and pinned
author evidence, derives conservative character/place/type facets, and labels
explicit, relative, or manuscript-only timing without guessing dates.
`story_timeline.go` is the thin Wails-facing delegate.

### continuity/
Read-only comparison engine over the current story fingerprint. The first
version emits explainable review prompts for confirmed-character identity and
appearance gaps, paired knowledge states, a narrow set of physical attributes,
nearby clock references, and chapter event density. Signals retain exact source
coordinates and never mutate manuscript or evidence data. `continuity.go` is
the thin Wails-facing delegate.

### storysearch/
Deterministic whole-manuscript evidence retrieval. Searches explicit query
submissions by scene, expands only confirmed character aliases, and returns
source excerpts and chapter coordinates without storing another manuscript
copy or calling AI. Search results can attach matching persisted evidence so
the Evidence Index and ad-hoc retrieval use the same source-located records.
`story_search.go` contains only the Wails-facing delegate.

Build 02475 layers deterministic Detail Search over those sources. Natural
question framing is reduced to searchable concepts without changing quoted
phrases; the engine then summarizes complete chapter coverage, relevant
event/fact evidence, related named details, missing confirmed names, singleton
occurrences, and discovery cues. Evidence association is limited to the actual
query-bearing paragraphs rather than every record in a matching chapter.

Knowledge questions are interpreted separately from general discovery queries.
The returned manuscript-order trail identifies named knowers or communicators,
explicit counterparties when present, epistemic state, cue, confidence, exact
source sentence, and chapter navigation. Rejected evidence is excluded through
the same source pipeline as every other Detail Search conclusion.

### [logging/](logging/LOGGING.md)
Debug logging for AI operations.

### [platform/](platform/PLATFORM.md)
Platform-specific utilities (Windows console hiding).

## AI Provider Integration

Provider-neutral HTTP transport lives in `internal/ai/providers`; the managed
Claude Code and Codex CLI drivers remain in `app.go` because they require:
- App context for cancellation
- Settings for API keys and endpoints
- Wails event emission for streaming

Supported providers:
- **Claude Code CLI** - Uses installed `claude` command
- **Anthropic API** - Direct Claude API calls
- **OpenAI** - GPT-4 and GPT-3.5
- **Gemini** - Google's Gemini Pro
- **Grok** - xAI's Grok
- **Local AI** - Any OpenAI-compatible endpoint

## Event System

Backend pushes events to frontend:

```go
// Streaming AI tokens
runtime.EventsEmit(a.ctx, "ai:token", token)

// Setup progress
runtime.EventsEmit(a.ctx, "setup:progress", "Installing Claude Code...")
```

## Error Handling

All exported functions return result structs with error fields:

```go
type SaveResult struct {
    Success  bool   `json:"success"`
    FilePath string `json:"file_path,omitempty"`
    Error    string `json:"error,omitempty"`
}
```

This allows clean error handling in TypeScript without exceptions.

## Building

```bash
cd wails
wails build
```

Output: `build/bin/draftline.exe` (Windows)

## Development

```bash
cd wails
wails dev
```

Hot-reloads both Go backend and React frontend.
