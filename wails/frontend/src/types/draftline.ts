import type { EditionCover } from './cover'
import type { EditionWrap } from './wrap'
import type { EditionSnapshot } from './snapshot'
import type { PlannerData } from './planner'

// Re-exported so that a component reading a book has one door to go through
// for the types on it, rather than having to know which file each lives in.
export type { CoverSourceReport, EditionCover } from './cover'
export type { EditionWrap } from './wrap'

export type { EditionSnapshot, SnapshotResult } from './snapshot'

export * from './analysis'
import type { AnalysisData } from './analysis'

export type Section = 'copyright' | 'front_matter' | 'body' | 'back_matter'

export type ProjectType = 'book' | 'universe'

export interface RecentProject {
  type: ProjectType
  path: string
  name: string
  lastOpened: string  // ISO date string
  /** Addresses this project's cover art at /recent-cover/<key>. Given by
   *  the backend; never built here, and not a path. */
  coverKey: string
  stats: {
    books?: number
    chapters: number
    words: number
  }
}

export type CharacterRole = 'protagonist' | 'antagonist' | 'supporting' | 'minor' | 'other'

export interface Character {
  id: string
  name: string
  role: CharacterRole | string
  description: string
  appearance: string
  personality: string
  motivation: string
  notes: string
  // Auto-detection fields
  is_auto_detected?: boolean
  aliases?: string[]
  spelling_aliases?: string[] | null
  mention_count?: number
  first_chapter?: number
  chapter_mentions?: Record<number, number>
  attributes?: Record<string, string>
  entity_kind?: 'person' | 'group' | 'organization' | 'place' | 'object' | 'unknown' | string
  detection_status?: 'accepted' | 'review' | 'rejected' | string
  detection_score?: number
}

export interface StoryBible {
  characters: Character[]
  plot_notes: string
  timeline: string
}

// Read Aloud cast: whether cast mode is on for this book and which TTS voice
// each character speaks in (keyed by lowercased canonical character name).
export interface ReadAloudCast {
  cast_mode: boolean
  voices?: Record<string, string>
}

// Beat Sheet - Save the Cat! style story beats
export type BeatType =
  | 'opening_image' | 'theme_stated' | 'setup' | 'catalyst'
  | 'debate' | 'break_into_two' | 'b_story' | 'fun_and_games'
  | 'midpoint' | 'bad_guys_close_in' | 'all_is_lost' | 'dark_night'
  | 'break_into_three' | 'finale' | 'final_image' | 'custom'

export interface Beat {
  id: string
  chapter_index: number
  beat_type: BeatType | string
  description: string
  notes?: string
}

export interface BeatSheet {
  beats: Beat[]
}

// Foreshadowing Ledger - track plant → reinforce → payoff
export type ForeshadowingStatus = 'planted' | 'active' | 'resolved'

export interface ForeshadowingItem {
  id: string
  name: string
  description: string
  plant_chapter: number
  reinforce_chapters?: number[]  // Optional to match Go omitempty
  payoff_chapter?: number
  status: ForeshadowingStatus | string  // string to match Wails bindings
  notes?: string
}

export interface ForeshadowingLedger {
  items: ForeshadowingItem[]
}

// Knowledge Matrix - track who knows what secrets at each chapter
export interface SecretInfo {
  id: string
  name: string
  description: string
}

export interface KnowledgeEntry {
  secret_id: string
  character_id: string
  learns_chapter?: number    // When they definitively learn it
  suspected_chapter?: number // When they start to suspect
}

export interface KnowledgeMatrix {
  secrets: SecretInfo[]
  entries: KnowledgeEntry[]
}


export interface WritingGoals {
  target_word_count: number
  daily_word_goal: number
  words_today: number
  last_writing_date: string  // ISO date YYYY-MM-DD
}

// Writing Style Mixer - controls AI behavior for Expand/Smooth modes
// Each value: 0 = off, 1 = subtle, 2 = moderate, 3 = heavy
export interface WritingStyleOptions {
  metaphors: number
  similes: number
  sensory_detail: number
  internal_thought: number
  dialogue: number
  action: number
  description: number
  pacing: number
}

export const DEFAULT_STYLE_OPTIONS: WritingStyleOptions = {
  metaphors: 2,
  similes: 1,
  sensory_detail: 2,
  internal_thought: 2,
  dialogue: 1,
  action: 2,
  description: 2,
  pacing: 2,
}

export interface ISBNEntry {
  format: string // hardcover | paperback | ebook | audiobook | large_print | other | ''
  value: string
}

export interface Metadata {
  title: string
  subtitle?: string
  author: string
  // A series number is a string: "2", "2.5" and "Book Two" are all printed.
  series_name?: string
  series_number?: string
  // Legacy single ISBN; mirrors isbns[0] and is kept for format compatibility.
  isbn: string
  isbns?: ISBNEntry[]
  publisher: string
  // The line a book appears under, which may differ from the publisher that
  // owns it. Exports print the imprint when there is one.
  imprint?: string
  // A BCP 47 tag. Empty exports as "en", which is what every book got before
  // this field existed.
  language?: string
  copyright_holder?: string
  // Catalogue: what a storefront needs and an exporter can declare.
  bisac_1?: string
  bisac_2?: string
  audience?: string
  keywords?: string
  short_description?: string
  contributors?: string
  created: string
  modified: string
  // Manuscript word count, computed by the Go backend on open and save.
  word_count?: number
}

export interface ChapterItem {
  id?: string
  title: string
  subtitle?: string  // Optional chapter subheading
  type: string
  content: string
}

export interface ChapterHistoryEntry {
  id: string
  chapter_id: string
  section: string
  chapter_title: string
  created_at: string
  reason: string
  word_count: number
  content_hash: string
  file: string
}

export interface ChapterHistorySnapshot {
  entry: ChapterHistoryEntry
  content: string
}

// ── Planner (planner.json) ──────────────────────────────────────────────────

export * from './planner'

// ── Editions (editions/index.json) ──────────────────────────────────────────
// The book's publishing record: which editions exist and which formats each
// one was published in. Mirrors internal/types/editions.go; the field names
// are the design brief's own and are not renamed on the way across.
//
// ISBN-10 and spine width are absent on purpose. Both are derived from fields
// that are here, and a derived value stored twice is a derived value that goes
// stale. editionModel.ts computes them for display.

export type EditionKind = 'ebook' | 'print' | 'audio'

export interface EditionFormat {
  id: string
  // One of EditionKind. Typed as a string because that is what the generated
  // Wails binding hands across the bridge; kindDot() and the panel branch on
  // it and treat anything else as a plain record with no specification.
  kind: string
  // The printed word: eBook, Paperback, Hardcover, Large print.
  format?: string
  // Stored exactly as the author typed it, hyphens and all.
  isbn13?: string
  registration?: string
  edition_statement?: string
  publication_date?: string
  list_price?: string
  status?: string
  channels?: string
  // Print specification.
  trim?: string
  page_count?: string
  paper_stock?: string
  binding?: string
  bleed?: string
  interior?: string
  gutter?: string
  // Ebook specification.
  epub_version?: string
  layout?: string
  asin?: string
  drm?: string
  // Advanced.
  imprint_of_record?: string
  territory_rights?: string
  rights_notice?: string
  lccn?: string
  // 'standard' or 'custom', stored rather than worked out: comparing settings
  // against the defaults could not decide it. Empty means standard.
  typesetting?: string
  // The export wizard's options, frozen onto the format so that exporting
  // this edition twice produces the same file.
  export_settings?: Record<string, unknown>
  // The frozen manuscript this format was published from. It names a record
  // in the index's own snapshot catalogue; the text itself never crosses the
  // bridge. See types/snapshot.ts.
  snapshot_id?: string
  // The print-ready wraparound artwork for this printed object: back, spine
  // and front in one piece. Absent on an ebook or an audiobook.
  wrap?: EditionWrap
}

export interface Edition {
  id: string
  label: string
  year: string
  status: string
  cover_id?: string
  cover?: EditionCover
  // The edition this one supersedes. It makes the copyright year list
  // cumulative and marks the edition as later, which is the only condition
  // under which a revision note is printed.
  previous_edition_id?: string
  revision_note?: string
  formats: EditionFormat[]
}

export interface EditionIndex {
  version: number
  editions: Edition[]
  // The catalogue of frozen manuscripts: which text each published ISBN went
  // out with. Small records only — the text lives in the project file.
  snapshots?: EditionSnapshot[]
}

export interface BookData {
  version: string
  metadata: Metadata
  copyright: string
  front_matter: ChapterItem[]
  body: ChapterItem[]
  back_matter: ChapterItem[]
  file_path?: string
  story_bible?: StoryBible
  writing_goals?: WritingGoals
  style_options?: WritingStyleOptions
  is_indexed?: boolean
  last_indexed?: string
  beat_sheet?: BeatSheet
  // Per-book Read Aloud voice casting; keys of voices are lowercased
  // canonical character names. Optional archive member read_aloud_cast.json.
  read_aloud_cast?: ReadAloudCast
  // Story-line timeline. Optional archive member planner.json.
  planner?: PlannerData
  // The publishing record: editions, formats, ISBNs. Optional archive member
  // editions/index.json. Cover images and frozen manuscripts are bytes and
  // stay out of this struct, which is serialised on every save.
  editions?: EditionIndex
  foreshadowing?: ForeshadowingLedger
  knowledge_matrix?: KnowledgeMatrix
  // Entity resolution and other analysis results
  analysis?: AnalysisData
}

// Result of character indexing
export interface IndexResult {
  success: boolean
  error?: string
  characters_found: number
  new_characters: number
  characters?: Character[]
  // Updated book with characters and entity resolution data populated
  book: BookData
}

// Result of splitting an entity
export interface SplitEntityResult {
  success: boolean
  error?: string
  book?: BookData
  characters?: Character[]
}

export interface SaveResult {
  success: boolean
  file_path: string
  error?: string
  /** What the save could not do without failing, such as data it could not read
   *  out of the project it saved from. A successful save with warnings wrote the
   *  book but left something behind. */
  warnings?: string[]
}

export const FRONT_MATTER_TYPES = [
  'Author\'s Note',
  'Foreword',
  'Preface',
  'Introduction',
  'Prologue',
  'Dedication',
]

export const BODY_TYPES = [
  'Chapter',
  'Interlude',
  'Part',
  'Scene',
]

export const BACK_MATTER_TYPES = [
  'Epilogue',
  'Afterword',
  'Appendix',
  'About the Author',
  'Acknowledgments',
  'Glossary',
]

export const EDITOR_FONTS = [
  'Merriweather',
  'Georgia',
  'Times New Roman',
  'Garamond',
  'IBM Plex Sans',
  'Arial',
]

export const EDITOR_FONT_SIZES = ['12', '13', '14', '15', '16', '18', '20', '24']
