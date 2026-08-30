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
│   │   ├── constants.ts        # SECTION_CONFIG, AI_MODES, BEAT_TYPES
│   │   ├── types.ts            # GlyphSection, AIMode, AIState
│   │   ├── GlyphIcon.tsx       # Glyph bar icons
│   │   ├── Dashboard/
│   │   │   └── index.tsx       # Word counts, goals, session stats
│   │   ├── AIStudio/
│   │   │   └── index.tsx       # AI modes, style mixer, streaming
│   │   ├── StoryBible/
│   │   │   └── index.tsx       # Characters, plot notes, timeline
│   │   └── PlotWalker/
│   │       └── index.tsx       # Beats, foreshadowing, knowledge matrix
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
- **StoryBible/** - Characters (with merge/highlight), plot notes, timeline
- **PlotWalker/** - Beat sheet, foreshadowing ledger, knowledge matrix, issues

### Writing diagnostics and plugins

Spelling and grammar are independent bundled features managed from **Settings > Plugins**. Spelling checks remain local; broad typo suggestions run in a web worker to avoid blocking editor input. Grammar checks use deterministic local rules and distinguish grammar diagnostics from optional style guidance.

The bundled feature catalog lives in `features/registry.ts`. The registry separates feature identity and metadata from its persisted setting key, providing the foundation for a future community-plugin catalog.

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

  // Story Bible Actions
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
