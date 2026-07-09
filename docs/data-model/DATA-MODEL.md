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
  title: string
  subtitle?: string
  type: 'Chapter' | 'Prologue' | 'Epilogue' | 'Part' |
        'Preface' | 'Dedication' | 'Acknowledgments' | 'Appendix' | string
  content: string  // HTML content
}
```

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
  autosave_interval: number  // minutes, 0 = disabled
}
```

## File Format Version

The manifest includes a version field:
```json
{
  "version": "2.0",
  "app_version": "0.12.02325",
  ...
}
```

This allows future format migrations.
