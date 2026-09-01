# Backend Guide (Go)

The Draftline backend is written in Go and uses [Wails v2](https://wails.io/) to bridge with the React frontend.

## Package Structure

```
wails/
├── main.go                    # Application entry point
├── app.go                     # App struct facade (~1,500 lines; guarded against growth)
├── story_search.go            # Thin Story Search Wails binding
├── import.go                  # EPUB/DOCX import pipeline + routing (see import/IMPORT.md)
├── import_sanitize.go         # Import decoder, XHTML sanitizer, chaptering
├── debt_guardrail_test.go     # 800-line file-size ratchet (see docs/TECHNICAL-DEBT.md)
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
    ├── indexing/              # Character detection
    │   ├── patterns.go        # Common words, regex patterns
    │   ├── characters.go      # Name detection, attribute extraction
    │   └── indexer.go         # Book/chapter indexing
    │
    ├── storysearch/           # Local scene/phrase search + confirmed alias expansion
    │   └── search.go           # Wails-independent search engine
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
Character name detection and attribute extraction for Story Bible.

### storysearch/
Deterministic whole-manuscript evidence retrieval. Searches explicit query
submissions by scene, expands only confirmed character aliases, and returns
source excerpts and chapter coordinates without storing another manuscript
copy or calling AI. `story_search.go` contains only the Wails-facing delegate.

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
