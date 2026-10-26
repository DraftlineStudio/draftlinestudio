// Analysis result types: entity resolution, relationships, evidence,
// story metrics, timeline, and continuity. Split from draftline.ts, whose
// BookData carries them under .analysis; type-only circular imports are
// erased at compile time.
import type { BookData, CharacterRole, Section } from './draftline'

// Entity Resolution - clusters name mentions into unified entities
export interface MentionRecord {
  id: string
  text: string        // Raw text as it appears
  sentence_id: string // ID of containing sentence
  chapter: number     // Chapter index
  char_offset: number // Character offset in chapter
  actor_evidence?: boolean
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
  unresolved_mention_ids?: string[]
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
  dialogue?: DialogueData
  style?: StyleData
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

// Dialogue layer: located, speaker-attributed quotation spans (byte offsets
// into the chapter's analysis plaintext).
export interface DialogueSpan {
  id: string
  chapter_index: number
  start_offset: number
  end_offset: number
  speaker_entity_id?: string
  speaker_name?: string
  kind: string            // 'dialogue' | 'mixed'
  attribution: string     // tag | adjacent-tag | pretag | continuation | nearby | alternation | none
  confidence: number
}

export interface DialogueData {
  spans?: DialogueSpan[]
  chapter_hashes?: Record<number, string>
  engine?: string
  version?: number
}

// Style layer: located style findings plus the author's dismissals. The
// editor places each underline by searching for prefix+snippet+suffix, so a
// finding the text has outgrown simply stops rendering.
export interface StyleFinding {
  id: string
  rule: string
  severity: string        // 'suggestion' | 'hint'
  chapter_index: number
  start_offset: number
  end_offset: number
  message: string
  snippet: string
  prefix?: string
  suffix?: string
  replacements?: string[]
}

export interface StyleDecision {
  finding_id: string
  status: string          // 'dismissed'
  decided_at?: string
}

export interface StyleData {
  findings?: StyleFinding[]
  decisions?: StyleDecision[]
  chapter_hashes?: Record<number, string>
  engine?: string
  version?: number
  truncated?: boolean
}
