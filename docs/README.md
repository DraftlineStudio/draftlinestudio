# Draftline Documentation

Draftline is a desktop writing application for novelists, built with Go (Wails) backend and React/TypeScript frontend.

## Quick Navigation

| Document | Description |
|----------|-------------|
| [Architecture Overview](architecture/OVERVIEW.md) | High-level system design and data flow |
| [Character Detection](architecture/CHARACTER-DETECTION.md) | Mention/entity pipeline architecture |
| [Plugin System](architecture/PLUGIN-SYSTEM.md) | Feature registry and plugin gating |
| [Plugin Development](plugins/README.md) | Building Draftline plugins: manifest, host API, sidecars |
| [Backend Guide](backend/BACKEND.md) | Go backend structure and APIs |
| [Book I/O](backend/book/BOOK.md) | Open/save/history for .draftline archives |
| [Import](backend/import/IMPORT.md) | EPUB/DOCX import and sanitization |
| [Export](backend/export/EXPORT.md) | EPUB/DOCX/PDF/Print PDF export |
| [Indexing](backend/indexing/INDEXING.md) | Character and story analysis indexing |
| [Backup](backend/backup/BACKUP.md) | Save-time backup rotation |
| [Logging](backend/logging/LOGGING.md) | Debug logging |
| [Platform](backend/platform/PLATFORM.md) | OS-specific helpers |
| [Types](backend/types/TYPES.md) | Shared Go type definitions |
| [File Associations](backend/FILE-ASSOCIATIONS.md) | OS file-type registration and open routing |
| [Frontend Guide](frontend/FRONTEND.md) | React components and state management |
| [Analysis Sidebars](frontend/ANALYSIS-SIDEBARS.md) | The analysis panel suite |
| [Bottom Bar](frontend/BOTTOM-BAR.md) | Bottom tool window and its panels |
| [Read Aloud](frontend/READ-ALOUD.md) | Text-to-speech plugin |
| [Data Model](data-model/DATA-MODEL.md) | File format and data structures |
| [AI Rewriting](ai/AI-REWRITING.md) | AI rewriting system and providers |
| [AI Prompt Engineering](ai/AI-PROMPT-ENGINEERING.md) | Prompt design for the rewrite modes |
| [Releasing](RELEASING.md) | Release workflow, installer packaging, and signing secrets |
| [Versioning](VERSION.md) | Version scheme and bump checklist |
| [Technical Debt](TECHNICAL-DEBT.md) | Live debt ledger and size guardrail |
| [Changelog](../CHANGELOG.md) | Version history and changes |

## Mental Model

```
+------------------+     +------------------+     +------------------+
|    FRONTEND      |     |     BACKEND      |     |    FILE SYSTEM   |
|   (React/TS)     |<--->|      (Go)        |<--->|  (.draftline)    |
+------------------+     +------------------+     +------------------+
        |                        |
        v                        v
+------------------+     +------------------+
|  Zustand Stores  |     |   AI Providers   |
| (bookStore + 7)  |     | Claude/GPT/etc   |
+------------------+     +------------------+
```

**The Core Loop:**
1. User writes in the **Rich Editor** (TipTap/ProseMirror)
2. Content stored in **Zustand stores** (bookStore.ts and friends)
3. On save, Go backend writes to **.draftline file** (ZIP archive)
4. AI features call external providers via Go backend
5. Results flow back through Wails bindings to React

## Key Concepts

### The .draftline File
A ZIP archive containing:
- `manifest.json` - Book metadata (including persisted word count) and section index
- `copyright.html`, `front_matter/`, `body/`, `back_matter/` - Section content as HTML
- `story_bible.json` - Characters, plot notes, timeline
- `beat_sheet.json`, `foreshadowing.json`, `knowledge_matrix.json` - Story-structure data (persisted for format compatibility; no current UI writes them)
- `analysis.json` - Persisted analysis results (evidence, fingerprint, structure)
- `history/index.json` + `history/snapshots/` - Chapter history
- `read_aloud_cast.json` - Read Aloud voice cast assignments

See [Data Model](data-model/DATA-MODEL.md) for the authoritative layout.

### Wails Bridge
Go functions are bound to the frontend via Wails. The `App` struct in `app.go` exposes methods that React calls directly:
```typescript
// Frontend calls Go
const result = await SaveBook(bookData)
const status = await CheckClaudeCode()
```

### State Management
State is split across eight Zustand stores with clear ownership:
- `bookStore` - book data, file I/O, autosave, save-entangled dialogs
- `appStore` - app-level UI state, settings, recent projects
- `editorStore` - editor bridge and pending AI diff
- `storyBibleStore` - character transforms and highlight
- `analysisStore`, `relationshipStore`, `chapterHistory`, `readAloudStore` - their named domains

## Architecture Layers

```
Layer 1: UI Components (React)
    - EditorPanel, ChapterPanel, ToolsPanel, analysis sidebars
    - Dialogs (Export, Settings, etc.)

Layer 2: State Management (Zustand)
    - Eight stores (see State Management above)

Layer 3: Backend Services (Go)
    - File I/O (open, save, export)
    - AI Integration (multiple providers)
    - Import/Export (EPUB, DOCX, PDF)
    - Analysis engine (internal/fingerprint, indexing, continuity)

Layer 4: External Systems
    - Claude Code / Codex CLIs
    - AI APIs (Anthropic, OpenAI, Google, xAI)
    - File System
```

## Getting Started

1. **Understand the data flow**: Start with [Architecture Overview](architecture/OVERVIEW.md)
2. **Frontend work**: Read [Frontend Guide](frontend/FRONTEND.md)
3. **Backend work**: Read [Backend Guide](backend/BACKEND.md)
4. **AI features**: Read [AI Prompt Engineering](ai/AI-PROMPT-ENGINEERING.md)
