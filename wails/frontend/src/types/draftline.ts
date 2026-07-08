export type Section = 'copyright' | 'front_matter' | 'body' | 'back_matter'

export type ProjectType = 'book' | 'universe'

export interface RecentProject {
  type: ProjectType
  path: string
  name: string
  lastOpened: string  // ISO date string
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
  mention_count?: number
  first_chapter?: number
  chapter_mentions?: Record<number, number>
  attributes?: Record<string, string>
}

export interface StoryBible {
  characters: Character[]
  plot_notes: string
  timeline: string
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

export interface Metadata {
  title: string
  author: string
  isbn: string
  publisher: string
  created: string
  modified: string
}

export interface ChapterItem {
  title: string
  subtitle?: string  // Optional chapter subheading
  type: string
  content: string
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
  foreshadowing?: ForeshadowingLedger
  knowledge_matrix?: KnowledgeMatrix
}

// Result of character indexing
export interface IndexResult {
  success: boolean
  error?: string
  characters_found: number
  new_characters: number
  updated_characters: number
  characters?: Character[]
}

export interface SaveResult {
  success: boolean
  file_path: string
  error?: string
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
