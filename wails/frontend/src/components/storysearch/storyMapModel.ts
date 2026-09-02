// Story Map derivation — pure, deterministic, no AI at render time.
//
// Everything here is computed from the story fingerprint the Go side already
// built (internal/storyfingerprint). Eras come from contexts plus anchored day
// runs, node placement comes from story days and narrative order, and every
// warning drawn (dashed strokes, loop-backs, dangling threads) is backed by a
// fingerprint record — nothing is invented for effect.

import { types } from '../../../wailsjs/go/models'
import type { EvidenceRecord } from '../../types/draftline'

// ── Layout constants ────────────────────────────────────────────────────────

/** Hard cap on rendered nodes so huge manuscripts never tank the SVG. */
export const MAP_MAX_NODES = 120

export const ERA_GAP = 3
export const PAD_LEFT = 8
const ERA_MIN_W = 64
const ERA_MAX_W = 380
const ERA_PER_EVENT = 26
const MIN_WIDTH = 980
const RIGHT_PAD = 190

/** Lane centres for 2-lane and 3-lane layouts (primary lane first). */
const LANE_YS_2 = [66, 122] as const
const LANE_YS_3 = [58, 114, 170] as const
const HEIGHT_2 = 236
const HEIGHT_3 = 292

export const NODE_LABEL_MAX = 18
const QUOTE_MAX = 200
const NAV_QUERY_MAX = 80

// ── Output shapes ───────────────────────────────────────────────────────────

export type EraKind = 'primary' | 'past' | 'memory' | 'dream' | 'simulation' | 'unknown'

export interface MapEra {
  id: string
  contextId: string
  /** Chip heading, e.g. "DAY 3–5" or the context label uppercased. */
  label: string
  /** Chip subline, e.g. "12 ev · recalled". */
  sub: string
  kind: EraKind
  /** True when a live review diagnostic touches an event in this era. */
  anomaly: boolean
  x: number
  width: number
  eventIds: string[]
  /** First/last anchored story day, when this era is a day segment. */
  dayFrom?: number
  dayTo?: number
}

export interface MapTick {
  x: number
  /** "DAY 4 · FRI" when a weekday is known from the story-time label. */
  label: string
}

export interface MapNode {
  id: string
  x: number
  y: number
  r: number
  lane: number
  /** 1-based position in narrative order among the rendered nodes. */
  ord: number
  label: string
  kind: EraKind
  eraId: string
  /** Dashed warning stroke: placement is inferred and unconfirmed. */
  dashed: boolean
  major: boolean
  narrativeOrder: number
}

export interface MapCheckpointMark {
  id: string
  x: number
  title: string
  status: string
  /** "4/6" — satisfied requirements over total, empty when untracked. */
  ratio: string
}

export interface MapDangle {
  d: string
  tx: number
  ty: number
  label: string
  threadId: string
}

export interface MapLoop {
  d: string
  labelX: number
  labelY: number
  label: string
  diagnosticId: string
}

export interface StoryMapLayout {
  eras: MapEra[]
  ticks: MapTick[]
  nodes: MapNode[]
  pathD: string
  checkpoints: MapCheckpointMark[]
  dangles: MapDangle[]
  loop: MapLoop | null
  width: number
  height: number
  laneCount: number
  /** Events dropped by the importance cap ("+N more events"). */
  hiddenCount: number
}

export interface StoryMapOptions {
  maxNodes?: number
}

// ── Small pure helpers ──────────────────────────────────────────────────────

export function truncateLabel(value: string, max = NODE_LABEL_MAX): string {
  const clean = (value ?? '').trim()
  if (clean.length <= max) return clean
  return `${clean.slice(0, max - 1).trimEnd()}…`
}

/** Anchored story day for an event, or null when the engine never placed it. */
export function eventDay(event: types.FingerprintEvent): number | null {
  const time = event.story_time
  if (!time) return null
  if (typeof time.day_offset === 'number') return time.day_offset
  if (typeof time.earliest_day === 'number') return time.earliest_day
  return null
}

const WEEKDAYS: [RegExp, string][] = [
  [/\bmondays?\b|\bmon\b/i, 'MON'],
  [/\btuesdays?\b|\btue\b/i, 'TUE'],
  [/\bwednesdays?\b|\bwed\b/i, 'WED'],
  [/\bthursdays?\b|\bthu\b/i, 'THU'],
  [/\bfridays?\b|\bfri\b/i, 'FRI'],
  [/\bsaturdays?\b|\bsat\b/i, 'SAT'],
  [/\bsundays?\b|\bsun\b/i, 'SUN'],
]

/** Extract a weekday abbreviation from a story-time label, if one is named. */
export function weekdayFromLabel(label: string | undefined): string | null {
  if (!label) return null
  for (const [pattern, short] of WEEKDAYS) {
    if (pattern.test(label)) return short
  }
  return null
}

/** True when an applied author story_day correction targets this event. */
export function hasStoryDayCorrection(
  fingerprint: types.StoryFingerprint,
  eventId: string,
): boolean {
  return (fingerprint.author_model?.corrections ?? []).some(
    correction => correction.kind === 'story_day' && correction.target_id === eventId,
  )
}

/** Dashed warning stroke: inferred placement the author has not confirmed. */
export function isUncertain(
  event: types.FingerprintEvent,
  fingerprint: types.StoryFingerprint,
): boolean {
  if (hasStoryDayCorrection(fingerprint, event.id)) return false
  const time = event.story_time
  if (!time) return true
  if (time.precision === 'relative' || time.precision === 'unknown') return true
  return time.confidence < 0.7
}

function contextKind(kind: string | undefined): EraKind {
  if (kind === 'primary' || kind === 'past' || kind === 'memory' || kind === 'dream' || kind === 'simulation') return kind
  return 'unknown'
}

// ── Era derivation ──────────────────────────────────────────────────────────

interface EraDraft {
  id: string
  contextId: string
  label: string
  kind: EraKind
  events: types.FingerprintEvent[]
  dayFrom?: number
  dayTo?: number
  /** 0 = unanchored flashback frames, 1 = day-anchored, 2 = unanchored rest. */
  band: number
}

function primaryContextId(
  contexts: types.StoryContext[],
  eventsByContext: Map<string, types.FingerprintEvent[]>,
): string {
  const declared = contexts.find(context => context.kind === 'primary')
  if (declared) return declared.id
  let best = ''
  let bestCount = -1
  for (const [id, list] of eventsByContext) {
    if (list.length > bestCount) { best = id; bestCount = list.length }
  }
  return best
}

/**
 * Contexts + anchored day runs become the era bands.
 *
 * The primary context splits into contiguous day segments where events are
 * anchored (a gap of more than one story day starts a new segment); its
 * unanchored remainder stays a single floating era. Every other context is one
 * era. Ordering is story time where the data supports it: unanchored
 * past/memory/dream frames first, then day segments by day, then everything
 * still unplaced — each group ordered by narrative appearance.
 */
export function deriveEras(fingerprint: types.StoryFingerprint): EraDraft[] {
  const events = fingerprint.events ?? []
  const contexts = fingerprint.contexts ?? []
  const byContext = new Map<string, types.FingerprintEvent[]>()
  for (const event of events) {
    const key = event.context_id || 'ctx-unknown'
    const list = byContext.get(key) ?? []
    list.push(event)
    byContext.set(key, list)
  }
  const contextById = new Map(contexts.map(context => [context.id, context]))
  const primaryId = primaryContextId(contexts, byContext)

  const drafts: EraDraft[] = []
  for (const [contextId, list] of byContext) {
    const context = contextById.get(contextId)
    const kind = contextId === primaryId ? 'primary' : contextKind(context?.kind)
    const label = (context?.label || context?.kind || 'unplaced').toUpperCase()

    if (contextId !== primaryId) {
      drafts.push({
        id: `ctx:${contextId}`,
        contextId,
        label,
        kind,
        events: list,
        band: kind === 'past' || kind === 'memory' || kind === 'dream' ? 0 : 2,
      })
      continue
    }

    // Primary context: split anchored events into contiguous day runs.
    const anchored = list
      .filter(event => eventDay(event) !== null)
      .sort((a, b) => (eventDay(a) ?? 0) - (eventDay(b) ?? 0) || a.narrative_order - b.narrative_order)
    const floating = list.filter(event => eventDay(event) === null)

    let segment: types.FingerprintEvent[] = []
    const flush = () => {
      if (segment.length === 0) return
      const from = eventDay(segment[0]) ?? 0
      const to = eventDay(segment[segment.length - 1]) ?? from
      drafts.push({
        id: `day:${from}:${to}`,
        contextId,
        label: from === to ? `DAY ${from}` : `DAY ${from}–${to}`,
        kind: 'primary',
        events: segment,
        dayFrom: from,
        dayTo: to,
        band: 1,
      })
      segment = []
    }
    for (const event of anchored) {
      const day = eventDay(event) ?? 0
      const prev = segment.length > 0 ? (eventDay(segment[segment.length - 1]) ?? day) : null
      if (prev !== null && day - prev > 1) flush()
      segment.push(event)
    }
    flush()

    if (floating.length > 0) {
      drafts.push({
        id: `ctx:${contextId}:floating`,
        contextId,
        label,
        kind: 'primary',
        events: floating,
        band: 2,
      })
    }
  }

  const minOrder = (draft: EraDraft) => Math.min(...draft.events.map(event => event.narrative_order))
  drafts.sort((a, b) => {
    if (a.band !== b.band) return a.band - b.band
    if (a.band === 1) return (a.dayFrom ?? 0) - (b.dayFrom ?? 0)
    return minOrder(a) - minOrder(b)
  })
  return drafts
}

// ── Lane assignment ─────────────────────────────────────────────────────────

/**
 * Present/primary lane on top, recalled/nested contexts below. More than two
 * contexts adds a third lane; beyond that, extra contexts overflow into the
 * nearest (bottom) lane so the map never grows unbounded rows.
 */
export function assignLanes(
  fingerprint: types.StoryFingerprint,
  eras: EraDraft[],
): { laneOfContext: Map<string, number>; laneCount: number } {
  const primary = eras.find(era => era.kind === 'primary')
  const primaryContext = primary?.contextId ?? ''
  const counts = new Map<string, number>()
  for (const era of eras) {
    if (era.contextId === primaryContext) continue
    counts.set(era.contextId, (counts.get(era.contextId) ?? 0) + era.events.length)
  }
  const others = [...counts.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
  const laneOfContext = new Map<string, number>()
  laneOfContext.set(primaryContext, 0)
  others.forEach(([contextId], index) => {
    laneOfContext.set(contextId, Math.min(index + 1, 2))
  })
  const laneCount = Math.min(1 + others.length, 3)
  return { laneOfContext, laneCount: Math.max(laneCount, 1) }
}

// ── Manuscript path (curve logic mirrors the design mock) ───────────────────

/** Cubic segments through the nodes in narrative order; same-lane hops lift. */
export function buildManuscriptPath(nodes: MapNode[], topLaneY: number): string {
  let pathD = ''
  nodes.forEach((node, index) => {
    if (index === 0) {
      pathD = `M ${node.x} ${node.y}`
      return
    }
    const prev = nodes[index - 1]
    const mx = (prev.x + node.x) / 2
    const lift = prev.y === node.y ? (node.y === topLaneY ? -14 : 14) : 0
    pathD += ` C ${mx} ${prev.y + lift}, ${mx} ${node.y + lift}, ${node.x} ${node.y}`
  })
  return pathD
}

// ── Full layout ─────────────────────────────────────────────────────────────

export function buildStoryMapLayout(
  fingerprint: types.StoryFingerprint,
  opts: StoryMapOptions = {},
): StoryMapLayout {
  const maxNodes = opts.maxNodes ?? MAP_MAX_NODES
  const all = [...(fingerprint.events ?? [])].sort((a, b) => a.narrative_order - b.narrative_order)

  // Importance cap: keep the highest-importance events, preserve order.
  let visible = all
  let hiddenCount = 0
  if (all.length > maxNodes) {
    const keep = new Set(
      [...all]
        .sort((a, b) => b.importance - a.importance || a.narrative_order - b.narrative_order)
        .slice(0, maxNodes)
        .map(event => event.id),
    )
    visible = all.filter(event => keep.has(event.id))
    hiddenCount = all.length - visible.length
  }

  const capped = types.StoryFingerprint.createFrom({})
  // Era derivation reads only events/contexts; feed it the capped view so era
  // widths and counts describe what is actually drawn.
  capped.events = visible
  capped.contexts = fingerprint.contexts ?? []
  const eraDrafts = deriveEras(capped)
  const { laneOfContext, laneCount } = assignLanes(capped, eraDrafts)
  const laneYs = laneCount >= 3 ? LANE_YS_3 : LANE_YS_2
  const height = laneCount >= 3 ? HEIGHT_3 : HEIGHT_2
  const topLaneY = laneYs[0]

  const flagged = new Set<string>()
  for (const diagnostic of fingerprint.diagnostics ?? []) {
    if (diagnostic.status) continue
    for (const id of diagnostic.event_ids ?? []) flagged.add(id)
  }

  // Place eras left to right, then nodes inside their era band.
  const eras: MapEra[] = []
  const nodesById = new Map<string, MapNode>()
  const ticks: MapTick[] = []
  let x = PAD_LEFT
  for (const draft of eraDrafts) {
    const width = Math.max(ERA_MIN_W, Math.min(ERA_MAX_W, ERA_MIN_W + draft.events.length * ERA_PER_EVENT))
    const kindNote = draft.band === 1 ? '' : ` · ${draft.kind === 'primary' ? 'floating' : draft.kind}`
    eras.push({
      id: draft.id,
      contextId: draft.contextId,
      label: draft.label,
      sub: `${draft.events.length} ev${kindNote}`,
      kind: draft.kind,
      anomaly: draft.events.some(event => flagged.has(event.id)),
      x,
      width,
      eventIds: draft.events.map(event => event.id),
      dayFrom: draft.dayFrom,
      dayTo: draft.dayTo,
    })

    const lane = laneOfContext.get(draft.contextId) ?? 0
    const y = laneYs[Math.min(lane, laneYs.length - 1)]
    const ordered = [...draft.events].sort((a, b) => {
      const dayA = eventDay(a)
      const dayB = eventDay(b)
      if (dayA !== null && dayB !== null && dayA !== dayB) return dayA - dayB
      return a.narrative_order - b.narrative_order
    })
    const seenDays = new Set<number>()
    ordered.forEach((event, index) => {
      const nodeX = Math.round(x + 10 + ((index + 0.5) / ordered.length) * (width - 20))
      const major = event.importance >= 0.7
      nodesById.set(event.id, {
        id: event.id,
        x: nodeX,
        y,
        r: major ? 8 : 6.5,
        lane,
        ord: 0, // assigned below in narrative order
        label: truncateLabel(event.author_summary || event.summary),
        kind: draft.kind,
        eraId: draft.id,
        dashed: isUncertain(event, fingerprint),
        major,
        narrativeOrder: event.narrative_order,
      })
      // One tick per distinct anchored day, at its first node.
      const day = eventDay(event)
      if (draft.band === 1 && day !== null && !seenDays.has(day)) {
        seenDays.add(day)
        const weekday = weekdayFromLabel(event.story_time?.label)
        ticks.push({ x: nodeX - 14, label: weekday ? `DAY ${day} · ${weekday}` : `DAY ${day}` })
      }
    })
    x += width + ERA_GAP
  }

  const width = Math.max(MIN_WIDTH, x + RIGHT_PAD)

  const nodes = visible
    .map(event => nodesById.get(event.id))
    .filter((node): node is MapNode => node !== undefined)
  nodes.forEach((node, index) => { node.ord = index + 1 })

  const pathD = buildManuscriptPath(nodes, topLaneY)

  // Checkpoints: a dashed success line + flag at the first matched event.
  // They live on the author model; the engine evaluates status and matches
  // in place there (internal/fingerprint/checkpoints.go).
  const checkpoints: MapCheckpointMark[] = []
  for (const checkpoint of fingerprint.author_model?.checkpoints ?? []) {
    if (checkpoint.status !== 'fulfilled' && checkpoint.status !== 'partial') continue
    const firstMatched = (checkpoint.matched_event_ids ?? [])
      .map(id => nodesById.get(id))
      .filter((node): node is MapNode => node !== undefined)
      .sort((a, b) => a.narrativeOrder - b.narrativeOrder)[0]
    if (!firstMatched) continue
    const requirements = checkpoint.requirements ?? []
    const satisfied = requirements.filter(requirement => requirement.satisfied).length
    checkpoints.push({
      id: checkpoint.id,
      x: firstMatched.x,
      title: checkpoint.title,
      status: checkpoint.status,
      ratio: requirements.length > 0 ? `${satisfied}/${requirements.length}` : '',
    })
  }

  return {
    eras,
    ticks,
    nodes,
    pathD,
    checkpoints,
    dangles: buildDangles(fingerprint, nodesById, width, height),
    loop: buildLoop(fingerprint, nodesById, height),
    width,
    height,
    laneCount,
    hiddenCount,
  }
}

// ── Dangling open threads ───────────────────────────────────────────────────

const OPEN_THREAD_STATES = new Set(['seeded', 'active', 'escalating', 'dormant', 'converging'])

/**
 * Up to two open threads get a dashed warning curve from their seed node
 * toward the right edge — the visual "this is still unpaid" cue.
 */
export function buildDangles(
  fingerprint: types.StoryFingerprint,
  nodesById: Map<string, MapNode>,
  width: number,
  height: number,
): MapDangle[] {
  const seedOf = (thread: types.StoryThread): MapNode | undefined => {
    if (thread.opened_by_event_id) {
      const node = nodesById.get(thread.opened_by_event_id)
      if (node) return node
    }
    for (const id of thread.event_ids ?? []) {
      const node = nodesById.get(id)
      if (node) return node
    }
    return undefined
  }

  const open = (fingerprint.threads ?? [])
    .filter(thread => OPEN_THREAD_STATES.has(thread.state))
    .map(thread => ({ thread, seed: seedOf(thread) }))
    .filter((entry): entry is { thread: types.StoryThread; seed: MapNode } => entry.seed !== undefined)
    .sort((a, b) =>
      (b.thread.event_ids?.length ?? 0) - (a.thread.event_ids?.length ?? 0) ||
      b.thread.confidence - a.thread.confidence ||
      a.thread.id.localeCompare(b.thread.id))
    .slice(0, 2)

  return open.map(({ thread, seed }, index) => {
    const baseY = height - 52 - index * 16
    const endX = width - 44
    const endY = baseY - 4
    const startY = seed.y + seed.r + 2
    const d = `M ${seed.x} ${startY} C ${seed.x + 150} ${baseY}, ${Math.max(seed.x + 220, endX - 250)} ${baseY}, ${endX} ${endY}`
    return {
      d,
      tx: endX - 210,
      ty: endY - 6,
      label: `OPEN · ${truncateLabel(thread.label, 26)} →`,
      threadId: thread.id,
    }
  })
}

// ── Loop-back curve (near-duplicate first/last chapters) ────────────────────

/**
 * When a near_duplicate_chapter diagnostic spans the manuscript's first and
 * last chapters, draw a dashed warning curve looping the last node back to
 * the first — the review desk owns the actual decision.
 */
export function buildLoop(
  fingerprint: types.StoryFingerprint,
  nodesById: Map<string, MapNode>,
  height: number,
): MapLoop | null {
  const events = fingerprint.events ?? []
  if (events.length === 0) return null
  let minChapter = Infinity
  let maxChapter = -Infinity
  for (const event of events) {
    if (event.chapter_index < minChapter) minChapter = event.chapter_index
    if (event.chapter_index > maxChapter) maxChapter = event.chapter_index
  }
  if (minChapter >= maxChapter) return null

  const eventById = new Map(events.map(event => [event.id, event]))
  for (const diagnostic of fingerprint.diagnostics ?? []) {
    if (diagnostic.kind !== 'near_duplicate_chapter' || diagnostic.status) continue
    const chapters = new Set(diagnostic.chapter_indices ?? [])
    for (const id of diagnostic.event_ids ?? []) {
      const chapter = eventById.get(id)?.chapter_index
      if (chapter !== undefined) chapters.add(chapter)
    }
    if (!chapters.has(minChapter) || !chapters.has(maxChapter)) continue

    const involved = [...(diagnostic.event_ids ?? [])]
      .map(id => nodesById.get(id))
      .filter((node): node is MapNode => node !== undefined)
      .sort((a, b) => a.narrativeOrder - b.narrativeOrder)
    const first = involved[0] ?? nodesById.get(events[0].id)
    const last = involved[involved.length - 1] ?? nodesById.get(events[events.length - 1].id)
    if (!first || !last || first.id === last.id) continue

    const dipY = height - 26
    const d = `M ${last.x} ${last.y + 12} C ${Math.max(first.x + 120, last.x - 120)} ${dipY}, ${first.x + 160} ${dipY}, ${first.x} ${first.y + 12}`
    return {
      d,
      labelX: Math.round((first.x + last.x) / 2) - 80,
      labelY: dipY - 6,
      label: 'possible loop — see Review',
      diagnosticId: diagnostic.id,
    }
  }
  return null
}

// ── Event detail pane model ─────────────────────────────────────────────────

export interface EventObligation {
  symbol: '○' | '✓' | '!'
  tone: 'open' | 'resolved' | 'flagged'
  text: string
}

export interface EventDetail {
  id: string
  /** Chip text, e.g. "Present · major" / "Recalled". */
  kindChip: string
  kind: EraKind
  when: string
  title: string
  location: string
  quote: string
  needsAnchor: boolean
  /** Day used by the Confirm-placement write. */
  anchorDay: number
  anchorPercent: number
  anchorLabel: string
  stateChanges: string[]
  obligations: EventObligation[]
  chips: string[]
  chapterLabel: string
  navigation: { section: string; sectionIndex: number; query: string } | null
}

const KIND_CHIP: Record<EraKind, string> = {
  primary: 'Present',
  past: 'Past',
  memory: 'Recalled',
  dream: 'Dream',
  simulation: 'Simulation',
  unknown: 'Unplaced',
}

export function describeEvent(
  fingerprint: types.StoryFingerprint,
  evidenceById: Map<string, EvidenceRecord>,
  eventId: string,
): EventDetail | null {
  const event = (fingerprint.events ?? []).find(candidate => candidate.id === eventId)
  if (!event) return null

  const contexts = fingerprint.contexts ?? []
  const context = contexts.find(candidate => candidate.id === event.context_id)
  const kind: EraKind = context?.kind === 'primary' || !context
    ? 'primary'
    : contextKind(context.kind)
  const kindChip = `${KIND_CHIP[kind]}${event.importance >= 0.85 ? ' · major' : ''}`

  const time = event.story_time
  const day = eventDay(event)
  const when = time?.label
    ? time.label
    : day !== null
      ? `Day ${day}`
      : `unplaced · ${Math.round((time?.confidence ?? 0) * 100)}%`

  const chapterLabel = event.chapter_title || `Chapter ${event.chapter_index + 1}`
  const places = (event.locations ?? []).slice(0, 2).map(term => term.text).filter(Boolean)
  const location = places.length > 0 ? `${chapterLabel} · ${places.join(', ')}` : chapterLabel

  const record = event.evidence_ids
    .map(id => evidenceById.get(id))
    .find((candidate): candidate is EvidenceRecord => candidate !== undefined)
  let quote = (record?.text ?? '').trim()
  if (quote.length > QUOTE_MAX) quote = `${quote.slice(0, QUOTE_MAX - 1).trimEnd()}…`

  const corrected = hasStoryDayCorrection(fingerprint, eventId)
  const needsAnchor = !corrected &&
    (time?.precision === 'relative' || time?.precision === 'unknown') &&
    (time?.confidence ?? 0) < 0.9

  const stateChanges = (fingerprint.states ?? [])
    .filter(interval => interval.start_event_id === eventId)
    .map(interval => {
      const subject = interval.entity_name || interval.entity_id
      const qualifier = interval.qualifier ? ` · ${interval.qualifier}` : ''
      return `${interval.kind}: ${subject} — ${interval.value}${qualifier}`
    })

  const obligations: EventObligation[] = []
  for (const thread of fingerprint.threads ?? []) {
    if (thread.opened_by_event_id === eventId) {
      obligations.push({ symbol: '○', tone: 'open', text: `${thread.label} — opened` })
    } else if (thread.resolved_by_event_id === eventId) {
      obligations.push({ symbol: '✓', tone: 'resolved', text: `${thread.label} — resolved` })
    } else if ((thread.event_ids ?? []).includes(eventId)) {
      obligations.push({ symbol: '!', tone: 'flagged', text: `${thread.label} — ${thread.state}` })
    }
  }

  return {
    id: event.id,
    kindChip,
    kind,
    when,
    title: event.author_summary || event.summary,
    location,
    quote,
    needsAnchor,
    anchorDay: time?.day_offset ?? time?.earliest_day ?? 0,
    anchorPercent: Math.round((time?.confidence ?? 0) * 100),
    anchorLabel: time?.label ?? '',
    stateChanges,
    obligations,
    chips: event.character_names ?? [],
    chapterLabel,
    navigation: record
      ? {
          section: String(record.section),
          sectionIndex: record.section_index,
          query: (record.text ?? '').slice(0, NAV_QUERY_MAX),
        }
      : null,
  }
}

// ── Author-model write payload (Confirm placement) ──────────────────────────

export interface AuthorModelDraft {
  contexts: types.StoryContext[]
  checkpoints: types.StoryCheckpoint[]
  canon: types.CanonRule[]
  corrections: types.FingerprintCorrection[]
  profiles: string[]
  voice_notes: types.CharacterVoiceNotes[]
}

/**
 * Clone the author model and pin the event to its inferred story day
 * (day_offset ?? earliest_day ?? 0). Replaces an existing story_day
 * correction for the same event rather than stacking duplicates.
 */
export function appendStoryDayCorrection(
  fingerprint: types.StoryFingerprint,
  eventId: string,
): { model: AuthorModelDraft; day: number } | null {
  const event = (fingerprint.events ?? []).find(candidate => candidate.id === eventId)
  if (!event) return null
  const day = event.story_time?.day_offset ?? event.story_time?.earliest_day ?? 0

  const base = fingerprint.author_model
  // Status vocabulary comes from the engine (internal/fingerprint/identity.go
  // reconcileCorrections): "active" while the target exists, else "orphaned".
  const correction = types.FingerprintCorrection.createFrom({
    id: `story-day-${eventId}`,
    target_id: eventId,
    kind: 'story_day',
    value: String(day),
    status: 'active',
  })
  const corrections = (base?.corrections ?? []).filter(
    existing => !(existing.kind === 'story_day' && existing.target_id === eventId),
  )
  corrections.push(correction)

  return {
    model: {
      contexts: [...(base?.contexts ?? [])],
      checkpoints: [...(base?.checkpoints ?? [])],
      canon: [...(base?.canon ?? [])],
      corrections,
      profiles: [...(base?.profiles ?? [])],
      voice_notes: [...(base?.voice_notes ?? [])],
    },
    day,
  }
}
