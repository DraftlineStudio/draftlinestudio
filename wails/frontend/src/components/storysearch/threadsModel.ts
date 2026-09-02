// Threads tab derivation — pure, deterministic, no AI.
//
// Turns the story fingerprint's obligation threads (questions, threats,
// commitments, mysteries) into plot-ready rows. Every x coordinate is
// normalized 0..1 across the whole manuscript: on a story-day basis when
// enough events carry time anchors, otherwise on narrative (manuscript)
// order — the caller surfaces which basis was used via `basis`.
//
// Nothing here guesses. States come straight from the engine; the only
// derived state is REOPENED, claimed strictly when a resolved thread has
// events after its resolution event's narrative order.

import { types } from '../../../wailsjs/go/models'

export type ThreadBasis = 'story_day' | 'narrative'
export type ThreadCategory = 'open' | 'dormant' | 'resolved'
export type SegmentKind = 'solid' | 'gap' | 'open' | 'intentional'

/** Row line colours rotate through design tokens, keyed by a stable id hash. */
export const THREAD_PALETTE = [
  'var(--status-warning)',
  'var(--status-success)',
  'var(--section-back)',
  'var(--section-front)',
  'var(--status-error)',
  'var(--app-accent)',
] as const

export const MAX_ROWS = 40
/** Chapters of silence between consecutive thread events that read as dormancy. */
export const DORMANT_GAP_CHAPTERS = 3
/** Share of anchored events required before columns switch to story time. */
export const STORY_DAY_THRESHOLD = 0.7

export interface ThreadSegment {
  x1: number
  x2: number
  kind: SegmentKind
}

export interface ThreadNote {
  text: string
  x: number
  tone: 'muted' | 'error'
}

export interface ThreadRow {
  id: string
  name: string
  stateLabel: string
  /** CSS colour (token var) for the state label text. */
  stateColor: string
  /** CSS colour (token var) for the row's line, seed dot and ring. */
  color: string
  category: ThreadCategory
  /** False when none of the thread's event ids resolved to real events. */
  plotted: boolean
  /** Seed dot position (first event by narrative order), 0..1. */
  x0: number
  segments: ThreadSegment[]
  /** Ring marker at the resolution event when the thread was resolved. */
  ring: boolean
  ringX: number
  /** Show the "OPEN →" flag at the right edge. */
  openEnd: boolean
  note?: ThreadNote
  /** Event to jump to on row click (opening event of the thread). */
  openingEventId?: string
  confidence: number
}

export interface ThreadConverge {
  /** Row indices (into rows) of the first converging parent pair. */
  fromRow: number
  toRow: number
  x: number
  label: string
}

export interface ThreadRowsResult {
  rows: ThreadRow[]
  basis: ThreadBasis
  /** Threads hidden by the MAX_ROWS cap. */
  moreCount: number
  counts: { all: number; open: number; resolved: number; dormant: number }
  converge?: ThreadConverge
}

/** Stable palette pick for one thread id (FNV-1a so rows keep their colour). */
export function threadColor(id: string): string {
  let hash = 2166136261
  for (let i = 0; i < id.length; i++) {
    hash ^= id.charCodeAt(i)
    hash = Math.imul(hash, 16777619) >>> 0
  }
  return THREAD_PALETTE[hash % THREAD_PALETTE.length]
}

function clamp01(value: number): number {
  return Math.max(0, Math.min(1, value))
}

/** Best story-day estimate for an event, or undefined when unanchored. */
function dayOf(event: types.FingerprintEvent): number | undefined {
  const time: types.StoryTime | undefined = event.story_time
  if (!time) return undefined
  if (typeof time.day_offset === 'number') return time.day_offset
  if (typeof time.earliest_day === 'number' && typeof time.latest_day === 'number') {
    return (time.earliest_day + time.latest_day) / 2
  }
  return undefined
}

function chapterLabel(event: types.FingerprintEvent): string {
  return event.chapter_title || `Chapter ${event.chapter_index + 1}`
}

interface RowDraft {
  row: ThreadRow
  rank: number
  firstX: number
}

const WARN = 'var(--status-warning)'
const MUTED = 'var(--text-muted)'

function deriveRow(
  thread: types.StoryThread,
  eventById: Map<string, types.FingerprintEvent>,
  xOf: (event: types.FingerprintEvent) => number,
): RowDraft {
  // Collect every resolvable event the thread claims, opening/resolution included.
  const seen = new Set<string>()
  const threadEvents: types.FingerprintEvent[] = []
  const push = (id?: string) => {
    if (!id || seen.has(id)) return
    seen.add(id)
    const event = eventById.get(id)
    if (event) threadEvents.push(event)
  }
  push(thread.opened_by_event_id)
  for (const id of thread.event_ids ?? []) push(id)
  push(thread.resolved_by_event_id)
  threadEvents.sort((a, b) => a.narrative_order - b.narrative_order)

  const points = threadEvents.map(xOf)
  const plotted = threadEvents.length > 0
  const x0 = points[0] ?? 0
  const lastX = points[points.length - 1] ?? 0

  // Solid activity runs, split by dormancy gaps (chapter delta >= 3) which
  // render as a sparse dash between the last beat before the gap and the next.
  const segments: ThreadSegment[] = []
  let gapNote: ThreadNote | undefined
  let maxGap = 0
  let runStart = x0
  for (let i = 1; i < threadEvents.length; i++) {
    const gap = threadEvents[i].chapter_index - threadEvents[i - 1].chapter_index
    if (gap >= DORMANT_GAP_CHAPTERS) {
      if (points[i - 1] !== runStart) segments.push({ x1: runStart, x2: points[i - 1], kind: 'solid' })
      segments.push({ x1: points[i - 1], x2: points[i], kind: 'gap' })
      if (gap > maxGap) {
        maxGap = gap
        gapNote = { text: `dormant gap · ${gap} ch`, x: points[i - 1], tone: 'muted' }
      }
      runStart = points[i]
    }
  }
  if (plotted && lastX !== runStart) segments.push({ x1: runStart, x2: lastX, kind: 'solid' })

  const openedBy = thread.opened_by_event_id ? eventById.get(thread.opened_by_event_id) : undefined
  const openingEvent = openedBy ?? threadEvents[0]
  const resolvedEvent = thread.resolved_by_event_id ? eventById.get(thread.resolved_by_event_id) : undefined
  const reopened =
    thread.state === 'resolved' &&
    resolvedEvent !== undefined &&
    threadEvents.some(event => event.narrative_order > resolvedEvent.narrative_order)

  let stateLabel: string
  let stateColor: string
  let category: ThreadCategory
  let rank: number
  let openEnd = false
  let continuation: SegmentKind | null = null
  let ring = false
  let ringX = 0
  let note = gapNote

  if (reopened && resolvedEvent) {
    stateLabel = 'REOPENED'
    stateColor = WARN
    category = 'open'
    rank = 0
    openEnd = true
    continuation = 'open'
    ring = true
    ringX = xOf(resolvedEvent)
    note = { text: 'resolution contradicted · reopened', x: ringX, tone: 'error' }
  } else {
    switch (thread.state) {
      case 'escalating': {
        // n/m: resolution progress expressed over the thread's plotted beats.
        const total = threadEvents.length
        const done = Math.max(0, Math.min(total, Math.round((thread.resolution ?? 0) * total)))
        stateLabel = total > 0 ? `ESCALATING · ${done}/${total}` : 'ESCALATING'
        stateColor = WARN
        category = 'open'
        rank = 0
        openEnd = true
        continuation = 'open'
        break
      }
      case 'dormant': {
        const chapters = thread.dormant_chapters ?? maxGap
        stateLabel = chapters > 0 ? `DORMANT ${chapters} CH` : 'DORMANT'
        stateColor = WARN
        category = 'dormant'
        rank = 1
        openEnd = true
        continuation = 'open'
        break
      }
      case 'converging':
        stateLabel = 'CONVERGING'
        stateColor = 'var(--app-accent-text)'
        category = 'open'
        rank = 2
        openEnd = true
        continuation = 'open'
        break
      case 'resolved':
        stateLabel = 'RESOLVED'
        stateColor = MUTED
        category = 'resolved'
        rank = 4
        if (resolvedEvent) {
          ring = true
          ringX = xOf(resolvedEvent)
        } else if (plotted) {
          ring = true
          ringX = lastX
        }
        break
      case 'abandoned':
        stateLabel = 'ABANDONED'
        stateColor = MUTED
        category = 'resolved'
        rank = 3
        break
      case 'intentionally_deferred':
        stateLabel = 'INTENTIONALLY DEFERRED'
        stateColor = 'var(--section-back)'
        category = 'resolved'
        rank = 3
        continuation = 'intentional'
        break
      default:
        // seeded / active / any future open-ish state.
        stateLabel = openingEvent ? `OPEN · seeded ${chapterLabel(openingEvent)}` : 'OPEN'
        stateColor = WARN
        category = 'open'
        rank = 0
        openEnd = true
        continuation = 'open'
    }
  }

  if (!plotted) openEnd = false
  if (continuation && plotted && lastX < 1) segments.push({ x1: lastX, x2: 1, kind: continuation })

  return {
    rank,
    firstX: x0,
    row: {
      id: thread.id,
      name: thread.label || 'Untitled thread',
      stateLabel,
      stateColor,
      color: threadColor(thread.id),
      category,
      plotted,
      x0,
      segments,
      ring,
      ringX,
      openEnd,
      note,
      openingEventId: openingEvent?.id,
      confidence: thread.confidence ?? 0,
    },
  }
}

/**
 * Build the Threads tab rows from a story fingerprint.
 * Rows sort open/escalating first, then dormant, converging, resolved;
 * within a group by confidence (desc) so the MAX_ROWS cap keeps the most
 * relevant threads.
 */
export function buildThreadRows(
  fingerprint: Pick<types.StoryFingerprint, 'events' | 'threads'>,
): ThreadRowsResult {
  const events = fingerprint.events ?? []
  const threads = fingerprint.threads ?? []
  const eventById = new Map<string, types.FingerprintEvent>(events.map(event => [event.id, event]))

  let minOrder = Infinity
  let maxOrder = -Infinity
  let minDay = Infinity
  let maxDay = -Infinity
  let anchoredCount = 0
  for (const event of events) {
    if (event.narrative_order < minOrder) minOrder = event.narrative_order
    if (event.narrative_order > maxOrder) maxOrder = event.narrative_order
    const day = dayOf(event)
    if (day !== undefined) {
      anchoredCount += 1
      if (day < minDay) minDay = day
      if (day > maxDay) maxDay = day
    }
  }
  const orderSpan = maxOrder - minOrder
  const daySpan = maxDay - minDay
  const basis: ThreadBasis =
    events.length > 0 && anchoredCount / events.length >= STORY_DAY_THRESHOLD && daySpan > 0
      ? 'story_day'
      : 'narrative'

  const narrativeX = (event: types.FingerprintEvent): number =>
    orderSpan > 0 ? clamp01((event.narrative_order - minOrder) / orderSpan) : 0
  const xOf = (event: types.FingerprintEvent): number => {
    if (basis === 'story_day') {
      const day = dayOf(event)
      // Unanchored events fall back to their manuscript position as a proxy.
      if (day !== undefined) return clamp01((day - minDay) / daySpan)
    }
    return narrativeX(event)
  }

  const drafts = threads.map(thread => deriveRow(thread, eventById, xOf))
  drafts.sort(
    (a, b) =>
      a.rank - b.rank ||
      b.row.confidence - a.row.confidence ||
      a.row.name.localeCompare(b.row.name),
  )

  const counts = { all: drafts.length, open: 0, resolved: 0, dormant: 0 }
  for (const draft of drafts) counts[draft.row.category] += 1

  const capped = drafts.slice(0, MAX_ROWS)
  const rows = capped.map(draft => draft.row)
  const moreCount = drafts.length - rows.length

  // First converging pair: a thread whose parent_ids land on two visible rows.
  let converge: ThreadConverge | undefined
  const rowIndexById = new Map(rows.map((row, index) => [row.id, index] as const))
  for (const thread of threads) {
    const parentRows = (thread.parent_ids ?? [])
      .map(id => rowIndexById.get(id))
      .filter((index): index is number => index !== undefined)
    if (parentRows.length < 2) continue
    const sorted = [...new Set(parentRows)].sort((a, b) => a - b)
    if (sorted.length < 2) continue

    const childIndex = rowIndexById.get(thread.id)
    let x = 0.5
    if (childIndex !== undefined && capped[childIndex].row.plotted) {
      x = capped[childIndex].firstX
    } else {
      const childEvents = [thread.opened_by_event_id, ...(thread.event_ids ?? [])]
        .map(id => (id ? eventById.get(id) : undefined))
        .filter((event): event is types.FingerprintEvent => event !== undefined)
      if (childEvents.length > 0) x = Math.min(...childEvents.map(xOf))
    }
    converge = { fromRow: sorted[0], toRow: sorted[1], x, label: `CONVERGE · ${thread.label || 'merged thread'}` }
    break
  }

  return { rows, basis, moreCount, counts, converge }
}
