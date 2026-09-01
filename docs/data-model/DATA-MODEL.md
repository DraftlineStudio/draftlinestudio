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
├── story_bible.json        # Characters, plot notes, timeline
├── beat_sheet.json         # Story structure beats
├── foreshadowing.json      # Plant/payoff tracking
└── knowledge_matrix.json   # Secrets and who knows them
```

## Core Types

### BookData
The main container for all book content:

```typescript
interface BookData {
  metadata: Metadata
  copyright: string
  front_matter: ChapterItem[]
  body: ChapterItem[]
  back_matter: ChapterItem[]
  story_bible: StoryBible
  beat_sheet: BeatSheet
  foreshadowing_ledger: ForeshadowingLedger
  knowledge_matrix: KnowledgeMatrix
  writing_goals: WritingGoals
  style_options: WritingStyleOptions
  is_indexed: boolean
  last_indexed: string  // ISO timestamp
}
```

### Metadata
```typescript
interface Metadata {
  title: string
  subtitle: string
  author: string
  series: string
  series_number: number
  genre: string
  keywords: string
  description: string
  isbn: string
  publisher: string
  language: string
  created: string   // ISO timestamp
  modified: string  // ISO timestamp
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

Automatic snapshots are activity-driven: editing marks the affected chapter, and Draftline records its current content after ten minutes. An unchanged or background-idle project produces no snapshot. Ordinary saves copy unchanged history in its compressed ZIP representation.

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
  activity_autosave_enabled: boolean // paused-edit saves + automatic chapter history
  // AI Configuration
  ai_enabled: boolean
  ai_mode: 'claudecode' | 'api' | 'local'
  ai_provider: 'claude' | 'openai' | 'gemini' | 'grok' | ''
  ai_api_key: string
  ai_local_endpoint: string

  // Theme
  theme: 'light' | 'dark' | 'system'
  auto_theme: boolean
  auto_theme_light_time: string  // "07:00"
  auto_theme_dark_time: string   // "19:00"

  // Editor
  font_family: string
  font_size: number
  line_height: number

  // Defaults for new books
  default_author: string
  default_language: string

  // Other
  show_word_count: boolean
}
```

## File Format Version

The manifest includes a version field:
```json
{
  "version": "2.2",
  "app_version": "0.16.02447",
  ...
}
```

This allows future format migrations.
