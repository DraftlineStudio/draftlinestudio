// Planner (planner.json) types, re-exported from draftline.ts so callers
// keep importing them from there.

// ── Planner (planner.json) ──────────────────────────────────────────────────

export type PlannerLaneKind = 'main' | 'subplot' | 'character'

export interface PlannerLane {
  id: string
  name: string
  kind: PlannerLaneKind | string
  color: string
  character_id?: string
}

// A card's scene link: the chapter's stable ID and a 1-based scene number.
export interface PlannerLink {
  chapter_id: string
  scene: number
}

export type PlannerCardStatus = 'planned' | 'drafted'

// A plot card. It sits on lines[0] at chapter_id ('' = Later, not yet pinned);
// the other lines are drawn as crossings. `changes` and `stakes` are the
// promise the card makes.
export interface PlannerCard {
  id: string
  // The scratch note an imported outline card came from.
  source_id?: string
  // Stable identity of the source movement within that note. This lets
  // repeated proposals show only movements not accepted before.
  source_key?: string
  title: string
  synopsis: string
  lines: string[]
  who: string[]
  // The names of `who` when it was set, so a card still names its people
  // after re-indexing reassigns codex IDs.
  who_names?: string[]
  changes?: string
  stakes?: string
  chapter_id: string
  link?: PlannerLink
  status: PlannerCardStatus | string
  // How the card was made: by hand, or from an imported outline.
  origin?: 'manual' | 'outline' | string
  updated?: string
}

export interface PlannerNote {
  id: string
  title: string
  body: string
  updated?: string
  excluded?: boolean
  system?: 'dead' | string
}

// A card taken off the timeline, kept whole so Reinstate can put it back
// where it was rather than as a fresh card in Later. `deleted` is when it
// was taken off.
export interface DeadPlannerCard {
  card: PlannerCard
  deleted: string
}

export type BeatTemplateId = 'none' | 'three-act' | 'save-the-cat'

export interface PlannerData {
  version: number
  lanes: PlannerLane[]
  cards: PlannerCard[]
  notes: PlannerNote[]
  synopsis?: Record<string, string>
  beat_template?: BeatTemplateId | string
  hidden_lanes?: string[]
  compact?: boolean
  // Deleted cards awaiting reinstatement or nothing at all, newest first.
  dead_cards?: DeadPlannerCard[]
}
