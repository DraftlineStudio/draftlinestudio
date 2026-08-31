# Frontend Guide (React/TypeScript)

The Draftline frontend is a React application using TypeScript, bundled with Vite.

## File Structure

```
frontend/src/
├── main.tsx                    # Entry point
├── App.tsx                     # Main layout
├── components/
│   ├── EditorPanel.tsx         # Rich editor container + diff view
│   ├── ChapterPanel.tsx        # Chapter list sidebar
│   ├── ToolsPanel.tsx          # Slim router (~121 lines)
│   ├── TitleBar.tsx            # Custom window title bar
│   ├── StatusBar.tsx           # Bottom status bar
│   ├── WelcomeScreen.tsx       # Start screen
│   ├── CodexPanel.tsx          # Full character codex view
│   ├── ContextMenu.tsx         # Right-click menu
│   ├── editor/
│   │   ├── RichEditor.tsx      # TipTap editor wrapper
│   │   ├── Toolbar.tsx         # Formatting toolbar
│   │   └── InlinePrompt.tsx    # Ctrl+L prompt UI
│   ├── tools/                  # ToolsPanel feature modules
│   │   ├── constants.ts        # SECTION_CONFIG, AI_MODES
│   │   ├── types.ts            # GlyphSection, AIMode, AIState
│   │   ├── GlyphIcon.tsx       # Glyph bar icons
│   │   ├── Dashboard/
│   │   │   └── index.tsx       # Word counts, goals, session stats
│   │   ├── AIStudio/
│   │   │   └── index.tsx       # AI modes, style mixer, streaming
│   │   └── Analysis/           # Analysis sidebar suite (see ANALYSIS-SIDEBARS.md)
│   │       ├── shared.ts       # Cross-panel nav, deltas, dismissals
│   │       ├── analysis.css    # Shared .an-* visual vocabulary
│   │       └── *Panel.tsx      # One component + css per panel
│   └── dialogs/
│       ├── AppSettingsDialog.tsx
│       ├── ExportWizard.tsx
│       ├── MetadataDialog.tsx
│       ├── NewBookWizard.tsx
│       └── NewChapterDialog.tsx
├── store/
│   ├── bookStore.ts            # Main book state (Zustand)
│   └── appStore.ts             # App settings state
├── services/
│   ├── spellCheck.ts           # Spell checking service
│   ├── grammarCheck.ts         # Local grammar and style rules
│   └── aiDetection.ts          # AI content detection
├── extensions/
│   ├── FontSize.ts             # TipTap font size extension
│   ├── CharacterHighlight.ts   # Character name highlighting
│   ├── SpellCheck.ts           # Persistent spelling decorations
│   └── GrammarCheck.ts         # Grammar/style decorations
├── features/
│   └── registry.ts             # Bundled feature/plugin metadata
├── utils/
│   ├── diff.ts                 # Word-level diff algorithm
│   ├── textUtils.ts            # HTML/text utilities
│   ├── accentColor.ts          # Windows accent color
│   └── sunTimes.ts             # Sunrise/sunset for auto-theme
├── hooks/
│   └── useAutoTheme.ts         # Auto light/dark theme
├── types/
│   └── draftline.ts            # TypeScript interfaces
└── styles/
    └── global.css              # Global styles
```

## Key Components

### App.tsx
Main application layout with three-panel design:
```
+------------+------------------+------------+
|  Chapter   |                  |   Tools    |
|   Panel    |   Editor Panel   |   Panel    |
|            |                  |            |
+------------+------------------+------------+
|              Status Bar                    |
+--------------------------------------------+
```

### EditorPanel.tsx
Contains:
- **RichEditor** - The TipTap editor instance
- **DiffPanel** - AI rewrite review with accept/reject
- **Toolbar** - Formatting controls

### ChapterPanel.tsx
- Chapter list (front matter, body, back matter)
- Drag-and-drop reordering
- Add/delete chapter controls
- Chapter type indicators

### ToolsPanel.tsx
Slim router that displays feature modules based on active glyph selection and the persisted plugin feature flags.

### tools/ Feature Modules
Each feature is isolated in its own folder:
- **Dashboard/** - Word counts, writing goals, session stats, AI detection
- **AIStudio/** - Rewrite modes, style mixer, streaming output, setup guidance
- **Analysis/** - The analysis sidebar suite (Signals, Prose, Pacing, Chapters, Worth Reviewing, AI Analysis) — see `docs/frontend/ANALYSIS-SIDEBARS.md`. The old single Story Analysis pane (`PlotWalker/`) was replaced by this suite in 0.16.02454.

The Story Bible (plot notes, timeline) and Plot Walker planning tools (beat sheet,
foreshadowing ledger, knowledge matrix) were retired from the sidebar in 0.16.02449.
Their data model, `plotStore.ts`/`storyBibleStore.ts` reducers, and `.draftline`
file-format fields remain intact so existing project files round-trip unchanged;
the Go settings keys `story_bible_enabled`/`plot_walker_enabled` are also retained
for settings-file compatibility but no longer gate any UI.

### Writing diagnostics and plugins

Spelling and grammar are independent bundled features managed from **Settings > Plugins**. Spelling checks remain local; broad typo suggestions use a prewarmed, length-indexed bounded search in a web worker to avoid both main-thread blocking and first-use delays. Dictionary lookup canonicalizes typographic apostrophes while preserving manuscript typography in replacement suggestions. Confirmed character names and aliases form a transient project lexicon, separate from the writer's persistent custom dictionary. Grammar checks use deterministic local rules and distinguish grammar diagnostics from optional style guidance.

The bundled feature catalog lives in `features/registry.ts`. Stable IDs, capabilities, categories, resource profiles, and persisted setting keys provide the client contract for future community analysis packs. The Marketplace tab currently previews this boundary; package discovery and installation require the signed catalog and installer described in `docs/architecture/PLUGIN-SYSTEM.md`.

`AnalysisCoordinator.tsx` keeps manuscript-scale work off the typing path. Content mutations increment a lightweight revision, mark Characters/Story/Pacing stale, and reset a 15-second idle timer. `analysisStore.ts` runs the consolidated Wails analysis call, rejects results if a newer revision exists, and consumes `analysis:progress` events for the bottom status bar. The analysis sidebar suite (`tools/Analysis/`, see `ANALYSIS-SIDEBARS.md`) reads the persisted results from `book.analysis.story`.

## State Management

### bookStore.ts (Zustand)

```typescript
interface BookState {
  // Book Data
  book: BookData | null
  currentChapterIndex: number
  currentSection: 'front_matter' | 'body' | 'back_matter'
  isDirty: boolean

  // Editor State
  editorSelection: { from: number; to: number } | null
  pendingDiff: DiffResult | null

  // Actions
  loadBook: (path: string) => Promise<void>
  saveBook: () => Promise<SaveResult>
  updateChapterContent: (content: string) => void
  addChapter: (title: string, type: string) => void
  deleteChapter: (index: number) => void
  reorderChapter: (from: number, to: number) => void

  // AI Actions
  rewriteSelection: (mode: string, styleOpts: StyleOptions) => Promise<void>
  generateInline: (prompt: string) => Promise<void>

  // Story Bible / planning-data actions (UI retired 0.16.02449; kept for
  // file-format compatibility and delegated to plotStore/storyBibleStore)
  updateCharacter: (id: string, data: Partial<Character>) => void
  addBeat: (beat: Beat) => void
  // ... etc
}
```

### appStore.ts
```typescript
interface AppState {
  settings: AppSettings
  claudeStatus: ClaudeCodeStatus

  loadSettings: () => Promise<void>
  saveSettings: (settings: AppSettings) => Promise<void>
  checkClaudeCode: () => Promise<void>
}
```

## The Editor

Uses [TipTap](https://tiptap.dev/) (built on ProseMirror):

```typescript
// RichEditor.tsx
const editor = useEditor({
  extensions: [
    StarterKit,
    Underline,
    TextStyle,
    FontFamily,
    FontSize,        // Custom extension
    TextAlign,
    CharacterHighlight, // Custom extension
  ],
  content: chapter.content,
  onUpdate: ({ editor }) => {
    updateChapterContent(editor.getHTML())
  },
})
```

### Custom Extensions

**FontSize.ts**
```typescript
// Allows font-size attribute on text
declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    fontSize: {
      setFontSize: (size: string) => ReturnType
    }
  }
}
```

**CharacterHighlight.ts**
```typescript
// Highlights character names in editor
// Triggered from Story Bible panel
```

## Wails Integration

Generated bindings in `wailsjs/`:

```typescript
// Calling Go functions
import { SaveBook, RewriteText } from '../wailsjs/go/main/App'

const result = await SaveBook(bookData)
if (!result.success) {
  showError(result.error)
}
```

```typescript
// Listening to events
import { EventsOn } from '../wailsjs/runtime'

EventsOn("ai:token", (token: string) => {
  appendToResult(token)
})
```

## Styling

- Global CSS in `global.css`
- CSS custom properties for theming
- Light/dark mode via `data-theme` attribute

```css
:root {
  --bg-primary: #ffffff;
  --text-primary: #1a1a1a;
}

[data-theme="dark"] {
  --bg-primary: #1a1a1a;
  --text-primary: #ffffff;
}
```

## Key Patterns

### Optimistic Updates
```typescript
// Update UI immediately, then persist
const updateChapter = (content: string) => {
  set({ book: { ...book, chapters: updatedChapters } })
  markDirty()
  // Save happens on explicit save or autosave
}
```

### Error Boundaries
```tsx
<ErrorBoundary fallback={<ErrorScreen />}>
  <App />
</ErrorBoundary>
```

### Dialog Pattern
```tsx
// Controlled by appStore
{showSettingsDialog && (
  <AppSettingsDialog onClose={() => setShowSettings(false)} />
)}
```

## Development

```bash
cd wails/frontend
npm install
npm run dev      # Dev server (usually run via wails dev)
npm run build    # Production build
```
