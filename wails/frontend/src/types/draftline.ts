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
  entity_kind?: 'person' | 'group' | 'organization' | 'place' | 'object' | 'unknown' | string
  detection_status?: 'accepted' | 'review' | 'rejected' | string
  detection_score?: number
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

// Entity Resolution - clusters name mentions into unified entities
export interface MentionRecord {
  id: string
  text: string        // Raw text as it appears
  sentence_id: string // ID of containing sentence
  chapter: number     // Chapter index
  char_offset: number // Character offset in chapter
  person_evidence?: boolean
  non_person_evidence?: boolean
  strong_person_evidence?: boolean
}

export interface EntityRecord {
  id: string
  canonical: string   // Best display name
  aliases: string[]   // All name variations
  mention_ids: string[] // IDs of all mentions
  confidence: number  // Merge confidence (0-1)
  titles: string[]    // Honorifics/titles seen
  kind?: string
  detection_status?: string
  detection_score?: number
  character_id?: string // Link to Character record
}

export interface SeparatedPairRecord {
  mention_id_1: string
  mention_id_2: string
  reason?: string
}

// User-confirmed "same person" name pair, re-applied on every re-index
export interface MergeRule {
  name_1: string
  name_2: string
}

export interface EntityDecision {
  names: string[]
  status: 'accepted' | 'rejected' | string
}

export interface EntityData {
  mentions?: MentionRecord[]
  entities?: EntityRecord[]
  separated_pairs?: SeparatedPairRecord[]
  merge_rules?: MergeRule[]
  decisions?: EntityDecision[]
  last_resolved?: string
  version?: number
}

// ── Relationship Analysis Types ─────────────────────────────────────────────

// SceneRecord represents a detected scene or paragraph boundary
export interface SceneRecord {
  id: string
  chapter_index: number
  start_offset: number
  end_offset: number
  scene_type: 'paragraph' | 'scene_break' | 'chapter' | string
  character_ids: string[]
}

// InteractionRecord represents a detected interaction between characters
export interface InteractionRecord {
  id: string
  participants: string[]
  chapter_index: number
  scene_id?: string
  sentence_id?: string
  interaction_type: 'dialogue' | 'co_occurrence' | 'reference' | string
  directed_from?: string
  directed_to?: string
  confidence: number
  text_snippet?: string
}

// RelationshipRecord represents an aggregated relationship between two characters
export interface RelationshipRecord {
  id: string
  character1_id: string
  character2_id: string
  first_chapter: number
  last_chapter: number
  interaction_count: number
  strength: number  // 0-1
  chapter_history: number[]
  interaction_ids?: string[]
  type_breakdown: Record<string, number>
}

// CharacterEvent represents a significant plot point tied to characters
export interface CharacterEvent {
  id: string
  character_ids: string[]
  chapter_index: number
  event_type: 'introduction' | 'meeting' | 'conflict' | 'resolution' | 'death' | 'custom' | string
  description: string
  is_auto_detected: boolean
}

// RelationshipData stores all relationship and interaction analysis
export interface RelationshipData {
  scenes?: SceneRecord[]
  interactions?: InteractionRecord[]
  relationships?: RelationshipRecord[]
  events?: CharacterEvent[]
  last_analyzed?: string
  version?: number
}

// Result of relationship analysis
export interface RelationshipAnalysisResult {
  success: boolean
  error?: string
  book?: BookData
  scenes_detected: number
  interactions_found: number
  relationships_built: number
}

// Character timeline event for UI display
export interface CharacterTimelineEvent {
  chapter: number
  event_type: 'mention' | 'dialogue' | 'interaction' | 'event' | string
  description: string
  related_chars?: string[]
}

// Result of getting a character timeline
export interface CharacterTimelineResult {
  success: boolean
  error?: string
  character_id: string
  events: CharacterTimelineEvent[]
}

// ── Visualization Types for D3.js ─────────────────────────────────────────────

// Node for force-directed graph
export interface CharacterNode {
  id: string
  name: string
  role: CharacterRole | string
  mentionCount: number
  firstChapter: number
  x?: number
  y?: number
  fx?: number | null  // Fixed position for pinned nodes
  fy?: number | null
}

// Edge for force-directed graph
export interface RelationshipEdge {
  source: string | CharacterNode
  target: string | CharacterNode
  strength: number
  interactionCount: number
  firstChapter: number
  lastChapter: number
  typeBreakdown: Record<string, number>
}

// Graph data for D3 visualization
export interface RelationshipGraphData {
  nodes: CharacterNode[]
  edges: RelationshipEdge[]
}

// Future-proof container for analysis results
export interface AnalysisData {
  entity_resolution?: EntityData
  relationships?: RelationshipData
  story?: StoryAnalysisData
  evidence?: EvidenceData
  continuity?: ContinuityData
  // Story Fingerprint v2 — full contract lives in the generated bindings
  // (wailsjs/go/models.ts types.StoryFingerprint); typed there to avoid
  // hand-mirroring a large evolving surface.
  fingerprint?: import('../../wailsjs/go/models').types.StoryFingerprint
  // Future analysis types:
  // plot_analysis?: PlotAnalysisData
  // theme_analysis?: ThemeAnalysisData
  version?: number
}

export interface ContinuityDecision {
  signal_id: string
  status: 'reviewed' | 'dismissed' | string
  decided_at?: string
}

/** Only author decisions persist; the report itself is always rebuilt. */
export interface ContinuityData {
  decisions?: ContinuityDecision[]
  version?: number
}

export interface EvidenceTerm {
  text: string
  label: string
}

export interface EvidenceKnowledgeState {
  state: 'learned' | 'knows' | 'does_not_know' | 'attempts_to_recall' | 'believes' | 'does_not_believe' | 'suspects' | 'does_not_suspect' | 'shared' | 'withheld' | string
  character_ids?: string[]
  character_names?: string[]
  counterparty_ids?: string[]
  counterparty_names?: string[]
  cue: string
  confidence: number
}

export interface EvidenceRecord {
  id: string
  kind: 'event' | 'fact' | string
  evidence_type: 'introduction' | 'discovery' | 'transition' | 'interaction' | 'state' | 'time_reference' | string
  chapter_id: string
  chapter_index: number
  section: Section | string
  section_index: number
  paragraph_index: number
  sentence_index: number
  start_offset: number
  end_offset: number
  text: string
  character_ids?: string[]
  character_names?: string[]
  named_entities?: EvidenceTerm[]
  action?: string
  time_expressions?: string[]
  knowledge_states?: EvidenceKnowledgeState[]
  confidence: number
  rationale: string
  status: 'detected' | 'confirmed' | 'rejected' | string
  source: 'auto' | 'author' | string
  author_text?: string
  author_note?: string
  pinned?: boolean
  reviewed_at?: string
}

export interface EvidenceData {
  content_hash: string
  chapter_hashes?: Record<string, string>
  engine: string
  last_analyzed: string
  records: EvidenceRecord[]
  truncated?: boolean
  version: number
}

export interface StoryTimelineFacet {
  id: string
  label: string
  count: number
}

export interface StoryTimelineEvent {
  id: string
  evidence_ids: string[]
  primary_type: string
  event_types: string[]
  text: string
  source_text: string
  chapter_id: string
  chapter_index: number
  chapter_title: string
  section: Section | string
  section_index: number
  paragraph_index: number
  sentence_index: number
  start_offset: number
  character_ids?: string[]
  character_names?: string[]
  thread_terms?: EvidenceTerm[]
  locations?: EvidenceTerm[]
  time_expressions?: string[]
  time_kind: 'anchored' | 'relative' | 'manuscript' | string
  time_label: string
  confidence: number
  status: string
  pinned?: boolean
}

export interface StoryTimelineChapter {
  chapter_index: number
  chapter_title: string
  event_count: number
  explicit_time_count: number
}

export interface StoryTimelineResult {
  success: boolean
  error?: string
  engine: string
  events: StoryTimelineEvent[]
  chapters: StoryTimelineChapter[]
  characters: StoryTimelineFacet[]
  locations: StoryTimelineFacet[]
  event_types: StoryTimelineFacet[]
  explicit_time_count: number
  relative_time_count: number
}

export interface ContinuitySource {
  evidence_id?: string
  text?: string
  chapter_id?: string
  chapter_index: number
  chapter_title: string
  section: Section | string
  section_index: number
  paragraph_index?: number
  start_offset?: number
}

export interface ContinuitySignal {
  id: string
  kind: string
  category: string
  severity: 'review' | 'info' | string
  title: string
  detail: string
  character_ids?: string[]
  character_names?: string[]
  sources?: ContinuitySource[]
  confidence: number
}

export interface ContinuityFacet {
  id: string
  label: string
  count: number
}

export interface ContinuityReport {
  success: boolean
  error?: string
  engine: string
  signals: ContinuitySignal[]
  categories: ContinuityFacet[]
  characters: ContinuityFacet[]
  review_count: number
  info_count: number
  chapters_checked: number
}

export interface TermCount {
  term: string
  count: number
}

export interface ChapterAnalysis {
  chapter_id?: string
  chapter_index: number
  title: string
  word_count: number
  sentence_count: number
  paragraph_count: number
  scene_break_count: number
  dialogue_percent: number
  average_sentence_words: number
  average_paragraph_words: number
  reading_ease: number
  mean_grade_level: number
  mean_word_length: number
  short_sentence_percent: number
  long_sentence_percent: number
  verb_percent: number
  adverb_percent: number
  adjective_percent: number
  tempo_score: number
  tempo_label: 'measured' | 'balanced' | 'brisk' | string
  keywords?: TermCount[]
  extractive_summary?: string
}

export interface StoryAnalysisObservation {
  kind: 'structure' | 'pacing' | string
  level: 'notice' | string
  title: string
  detail: string
  chapter_index?: number
}

export interface StoryAnalysisData {
  content_hash: string
  engine: string
  last_analyzed: string
  overview: {
    chapter_count: number
    word_count: number
    sentence_count: number
    paragraph_count: number
    average_chapter_words: number
    average_sentence_words: number
    dialogue_percent: number
    reading_ease: number
    mean_grade_level: number
    tempo_score: number
  }
  chapters: ChapterAnalysis[]
  observations?: StoryAnalysisObservation[]
  version: number
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
