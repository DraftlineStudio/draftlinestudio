# Draftline Documentation

Draftline is a desktop writing application for novelists, built with Go (Wails) backend and React/TypeScript frontend.

## Quick Navigation

| Document | Description |
|----------|-------------|
| [Architecture Overview](architecture/OVERVIEW.md) | High-level system design and data flow |
| [Backend Guide](backend/BACKEND.md) | Go backend structure and APIs |
| [Frontend Guide](frontend/FRONTEND.md) | React components and state management |
| [Data Model](data-model/DATA-MODEL.md) | File format and data structures |
| [AI Rewriting](ai/AI-REWRITING.md) | AI rewriting system and providers |
| [AI Prompt Engineering](ai/AI-PROMPT-ENGINEERING.md) | Prompt design for the rewrite modes |
| [Releasing](RELEASING.md) | Release workflow, installer packaging, and signing secrets |
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
|   Zustand Store  |     |   AI Providers   |
|   (bookStore)    |     | Claude/GPT/etc   |
+------------------+     +------------------+
```

**The Core Loop:**
1. User writes in the **Rich Editor** (TipTap/ProseMirror)
2. Content stored in **Zustand store** (bookStore.ts)
3. On save, Go backend writes to **.draftline file** (ZIP archive)
4. AI features call external providers via Go backend
5. Results flow back through Wails bindings to React

## Key Concepts

### The .draftline File
A ZIP archive containing:
- `manifest.json` - Book metadata and chapter index
- `body/*.html` - Chapter content as HTML
- `story_bible.json` - Characters, plot notes, timeline
- `beat_sheet.json` - Story structure beats
- `foreshadowing.json` - Plant/payoff tracking
- `knowledge_matrix.json` - Who knows what secrets

### Wails Bridge
Go functions are bound to the frontend via Wails. The `App` struct in `app.go` exposes methods that React calls directly:
```typescript
// Frontend calls Go
const result = await SaveBook(bookData)
const status = await CheckClaudeCode()
```

### State Management
Zustand store (`bookStore.ts`) is the single source of truth for:
- Current book data
- Active chapter
- Editor state
- UI preferences

## Architecture Layers

```
Layer 1: UI Components (React)
    - EditorPanel, ChapterPanel, ToolsPanel
    - Dialogs (Export, Settings, etc.)

Layer 2: State Management (Zustand)
    - bookStore (book data + actions)
    - appStore (app-level settings)

Layer 3: Backend Services (Go)
    - File I/O (open, save, export)
    - AI Integration (multiple providers)
    - Import/Export (EPUB, DOCX, PDF)

Layer 4: External Systems
    - Claude Code CLI
    - AI APIs (Anthropic, OpenAI, Google, xAI)
    - File System
```

## Getting Started

1. **Understand the data flow**: Start with [Architecture Overview](architecture/OVERVIEW.md)
2. **Frontend work**: Read [Frontend Guide](frontend/FRONTEND.md)
3. **Backend work**: Read [Backend Guide](backend/BACKEND.md)
4. **AI features**: Read [AI Prompt Engineering](ai/AI-PROMPT-ENGINEERING.md)
