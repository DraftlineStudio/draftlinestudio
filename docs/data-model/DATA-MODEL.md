# Data Model

## The .draftline File Format

A `.draftline` file is a ZIP archive with a specific structure:

```
book.draftline (ZIP)
│
├── manifest.json           # Book metadata and chapter index
│
├── body/                   # Main chapters
│   ├── 000.html
│   ├── 001.html
│   └── ...
│
├── front_matter/           # Preface, dedication, etc.
│   └── ...
│
├── back_matter/            # Appendix, acknowledgments, etc.
│   └── ...
│
├── analysis.json           # Rebuildable characters, relationships, story metrics, evidence
├── planner.json            # Planner: story lines, cards, notes, synopsis (optional)
├── history/                # Chapter snapshot history (format 2.2+)
│   ├── index.json          # Snapshot metadata
│   └── snapshots/          # Deduplicated chapter versions
├── story_bible.json        # Characters, plot notes, timeline
├── read_aloud_cast.json    # Per-book Read Aloud voice casting (optional)
├── beat_sheet.json         # Story structure beats
├── foreshadowing.json      # Plant/payoff tracking
└── knowledge_matrix.json   # Secrets and who knows them
```

## Core Types

### BookData
The main container for all book content:

```typescript
interface BookData {
  version: string
  metadata: Metadata
  copyright: string
  front_matter: ChapterItem[]
  body: ChapterItem[]
  back_matter: ChapterItem[]
  file_path?: string
  story_bible?: StoryBible
  beat_sheet?: BeatSheet
  foreshadowing?: ForeshadowingLedger
  knowledge_matrix?: KnowledgeMatrix
  writing_goals?: WritingGoals
  style_options?: WritingStyleOptions
  is_indexed?: boolean
  last_indexed?: string  // ISO timestamp
  read_aloud_cast?: ReadAloudCast  // per-book voice casting
  planner?: PlannerData            // Planner story lines, cards, notes
  analysis?: AnalysisData          // entity resolution + other analysis results
}
```

### Metadata
```typescript
interface Metadata {
  title: string
  author: string
  isbn: string          // legacy single ISBN; mirrors isbns[0]
  isbns?: ISBNEntry[]   // per-format ISBNs (hardcover, paperback, ebook, ...)
  publisher: string
  created: string   // ISO timestamp
  modified: string  // ISO timestamp
  word_count?: number  // manuscript word count, computed by the Go backend on open and save
}
```

### ChapterItem
```typescript
interface ChapterItem {
  id: string       // Stable across chapter renames and reordering
  title: string
  subtitle?: string
  type: 'Chapter' | 'Prologue' | 'Epilogue' | 'Part' |
        'Preface' | 'Dedication' | 'Acknowledgments' | 'Appendix' | string
  content: string  // HTML content
}
```

### Chapter History

Archive format 2.2 stores snapshot metadata in `history/index.json` and chapter HTML in `history/snapshots/`. A snapshot records the stable chapter ID, section and title at capture time, timestamp, reason, word count, and SHA-256 content hash. Identical content is not stored twice. Draftline retains at most 50 snapshots per chapter and 1,000 per project.

Automatic snapshots are activity-driven: editing marks the affected chapter, and Draftline records its current content after ten minutes. An unchanged or background-idle project produces no snapshot. Chapter history remains active when activity-based autosave is disabled. Accepted AI and comparison passes atomically record the chapter immediately before and after the change rather than waiting for the periodic timer. A writer can also take a manual snapshot at any time (Chapter History dialog, or *Snapshot Chapter Now* in the project menu) with an optional label such as "Before rewrite"; it is stored under that reason and saves the manuscript in the same archive write. Ordinary saves copy unchanged history in its compressed ZIP representation.

### Planner

`planner.json` (version 1) is written once the Planner has been used for a
book and holds everything the writer typed there:

- `lanes`: the story lines (the main plot, typed subplots, and character arcs
  keyed by `character_id`).
- `cards`: `title`, `synopsis`, `lines`, `who` with the matching `who_names`,
  `changes`, `stakes`, `chapter_id`, an optional scene `link`, `status`
  (`planned` or `drafted`), `origin` (`manual` or `outline`), and `source_id`
  for a card proposed from an imported outline note.
- `notes`: scratch notes; the one with `system: "dead"` is the automatic
  Dead ideas note, and a note may be `excluded` from Propose Cards.
- `synopsis`: per-chapter synopsis edits keyed by chapter ID. A chapter with
  no entry is generated from its cards instead.
- `beat_template`, `hidden_lanes`, and `compact`: view settings.

Cards reference chapters by their stable IDs, never by index, so reordering or
renaming chapters does not move cards; a card with an empty `chapter_id` sits
in *Later*. A book that never opened the Planner has no `planner.json`. An
unknown version is refused rather than overwritten, so a file written by a
newer Draftline is safe. See `docs/frontend/PLANNER.md`.

The Planner stores no analysis results. Nothing in `planner.json` is derived
from the manuscript.

### Derived Story Analysis

`analysis.json` stores rebuildable analysis separately from author-owned manuscript and planning data. Its `story` object contains an engine/version identifier, manuscript content hash, timestamp, aggregate metrics, per-chapter metrics, and evidence-based observations. Current built-in metrics include sentence and paragraph structure, dialogue density, readability, part-of-speech ratios, keywords, an extractive summary, and a descriptive tempo signal.

Its `evidence` object is the persistent local fact/event index. Each admitted
record contains a stable content-derived ID, kind and cue type, exact source
sentence, stable chapter ID plus chapter/section/paragraph/sentence coordinates,
confirmed character IDs and names, locally detected named terms, primary verb,
explicit time expressions, cue confidence and rationale, and a
`detected | confirmed | rejected` review status. Unrelated earlier edits do not
renumber unchanged evidence records. Reanalysis preserves confirmed/rejected
decisions while their exact source remains present and always preserves
author-created records.

The built-in index is conservative: it admits explainable introductions,
discoveries/revelations, arrivals/departures/transitions, direct interactions,
stated conditions, and explicit time references anchored to named story
subjects. It does not treat every sentence as a plot event. Records are capped
at 50,000 with an explicit `truncated` flag. This adds no new archive entry or
manifest migration—the future-proof `analysis.json` container remains part of
archive format 2.2, while its internal analysis schema advances independently.

Editing prose marks Characters, Story, and Pacing analysis stale. After 15 seconds without another edit, Draftline runs the local pipeline and persists the new derived results on the next save. A result produced from an older frontend revision is discarded, so background analysis cannot replace newer author text.

## Story Bible

### StoryBible
```typescript
interface StoryBible {
  characters: Character[]
  plot_notes: string    // Freeform markdown
  timeline: string      // Freeform markdown
}
```

### Character
```typescript
interface Character {
  id: string
  name: string
  role: 'protagonist' | 'antagonist' | 'supporting' | 'minor' | 'other'
  description: string
  appearance: string
  personality: string
  motivation: string
  notes: string

  // Auto-detection fields
  is_auto_detected: boolean
  aliases: string[]
  mention_count: number
  first_chapter: number
  chapter_mentions: Record<number, number>  // chapter index -> count
  attributes: {
    eye_color?: string
    hair_color?: string
    age?: string
  }
}
```

## Beat Sheet (Story Structure)

> The planning schemas below (beat sheet, foreshadowing ledger, knowledge matrix) are persisted for file-format compatibility; no current UI surface writes them.

### BeatSheet
```typescript
interface BeatSheet {
  beats: Beat[]
}
```

### Beat
```typescript
interface Beat {
  id: string
  chapter_index: number
  beat_type: BeatType
  description: string
  notes?: string
}

type BeatType =
  | 'Opening Image'
  | 'Theme Stated'
  | 'Setup'
  | 'Catalyst'
  | 'Debate'
  | 'Break into Two'
  | 'B Story'
  | 'Fun & Games'
  | 'Midpoint'
  | 'Bad Guys Close In'
  | 'All Is Lost'
  | 'Dark Night'
  | 'Break into Three'
  | 'Finale'
  | 'Final Image'
  | 'Custom'
```

## Foreshadowing Ledger

### ForeshadowingLedger
```typescript
interface ForeshadowingLedger {
  items: ForeshadowingItem[]
}
```

### ForeshadowingItem
```typescript
interface ForeshadowingItem {
  id: string
  name: string                    // Short label
  description: string             // Full description
  plant_chapter: number           // Where planted
  reinforce_chapters: number[]    // Where reinforced
  payoff_chapter?: number         // Where paid off (nullable)
  status: 'planted' | 'active' | 'resolved'
  notes?: string
}
```

## Knowledge Matrix

### KnowledgeMatrix
```typescript
interface KnowledgeMatrix {
  secrets: SecretInfo[]
  entries: KnowledgeEntry[]
}
```

### SecretInfo
```typescript
interface SecretInfo {
  id: string
  name: string        // Short label
  description: string // Full description
}
```

### KnowledgeEntry
```typescript
interface KnowledgeEntry {
  secret_id: string
  character_id: string
  learns_chapter?: number  // When they learn the secret
}
```

## Writing Goals

```typescript
interface WritingGoals {
  target_word_count: number
  daily_word_goal: number
  words_written_today: number
  last_session_date: string  // ISO date
}
```

## Style Options

```typescript
interface WritingStyleOptions {
  prose_guide: string        // Sample prose for AI to match
  style_mixer: StyleMixer
}

interface StyleMixer {
  metaphors: number      // 0-3
  similes: number        // 0-3
  sensory: number        // 0-3
  internal_thought: number // 0-3
  dialogue: number       // 0-3
  action: number         // 0-3
  description: number    // 0-3
  pacing: number         // 0-3
}
```

## App Settings

Stored separately in OS config directory:

```typescript
interface AppSettings {
  // Application
  default_author: string
  default_publisher: string
  default_copyright: string
  default_save_dir: string
  activity_autosave_enabled: boolean // paused-edit project saves; chapter history is independent
  custom_dictionary?: string[]
  update_check_enabled: boolean

  // Theme
  theme_mode: 'light' | 'dark' | 'auto'
  auto_theme_use_manual: boolean
  auto_theme_dawn: string  // "HH:MM"
  auto_theme_dusk: string  // "HH:MM"

  // Plugins / analysis
  spell_check_enabled: boolean
  grammar_check_enabled: boolean
  cast_enabled: boolean
  analysis_enabled: boolean
  analysis_cpu_profile: 'adaptive' | 'gentle' | 'balanced' | 'fast'

  // Read Aloud
  read_aloud_enabled: boolean
  read_aloud_voice: string
  read_aloud_speed: number
  read_aloud_volume: number  // 0-1
  read_aloud_glow: boolean

  // AI Configuration
  ai_enabled: boolean
  ai_mode: 'claudecode' | 'codex' | 'api' | 'local'
  ai_task_routes?: Record<string, string>  // per-task ai_mode overrides
  ai_provider: 'claude' | 'openai' | ''
  has_api_key: boolean       // keys live in the OS keyring, never in settings
  ai_debug_logging: boolean  // opt-in local prompt/manuscript logging
  ai_model: string
  ai_local_endpoint: string  // e.g. http://localhost:11434/v1
  ai_local_model: string
  prose_guide: string

  // Book defaults
  book_font: string
  editor_font_size: 'small' | 'normal' | 'large'
  book_font_size: number      // export font size in points
  book_line_spacing: string   // "1.0" | "1.25" | "1.5" | "2.0"
  book_drop_caps: boolean
  book_trim_size: string      // "6x9" | "5.5x8.5" | "5x8" | "7x10" | "A5"

  // UI persistence
  sidebar_panel_width: number
  sidebar_active_section: string
  characters_lane_view: 'grid' | 'heat' | 'weave'
}
```

(The full struct is `AppSettings` in `wails/internal/types/settings.go`; a few legacy keys — `dark_mode`, `story_bible_enabled`, `read_aloud_device`, `read_aloud_threads` — are retained for settings-file compatibility.)

## File Format Version

The manifest includes a version field:
```json
{
  "version": "2.2",
  "app_version": "0.19.02587",
  ...
}
```

This allows future format migrations.
