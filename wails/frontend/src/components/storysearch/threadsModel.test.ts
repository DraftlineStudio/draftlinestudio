import { describe, expect, it } from 'vitest'
import { types } from '../../../wailsjs/go/models'
import {
  buildThreadRows,
  MAX_ROWS,
  THREAD_PALETTE,
  threadColor,
} from './threadsModel'

interface EventSpec {
  id: string
  narrative_order?: number
  chapter_index?: number
  chapter_title?: string
  evidence_ids?: string[]
  story_time?: { day_offset?: number; earliest_day?: number; latest_day?: number; precision?: string }
}

function ev(spec: EventSpec): types.FingerprintEvent {
  return types.FingerprintEvent.createFrom({
    id: spec.id,
    summary: spec.id,
    evidence_ids: spec.evidence_ids ?? [`evi-${spec.id}`],
    kinds: ['action'],
    context_id: 'ctx-main',
    story_time: spec.story_time
      ? { context_id: 'ctx-main', precision: 'day', confidence: 0.9, ...spec.story_time }
      : undefined,
    chapter_id: `ch-${spec.chapter_index ?? 0}`,
    chapter_index: spec.chapter_index ?? 0,
    chapter_title: spec.chapter_title ?? `Chapter ${(spec.chapter_index ?? 0) + 1}`,
    paragraph_index: 0,
    start_offset: 0,
    narrative_order: spec.narrative_order ?? 0,
    importance: 0.5,
    confidence: 0.8,
    status: 'detected',
  })
}

function th(spec: Partial<types.StoryThread> & { id: string }): types.StoryThread {
  return types.StoryThread.createFrom({
    label: spec.id,
    kind: 'question',
    state: 'active',
    resolution: 0,
    confidence: 0.8,
    source: 'engine',
    ...spec,
  })
}

/** Four events spread across the manuscript so normalization has a span. */
function span(): types.FingerprintEvent[] {
  return [
    ev({ id: 'e0', narrative_order: 0, chapter_index: 0 }),
    ev({ id: 'e1', narrative_order: 10, chapter_index: 1 }),
    ev({ id: 'e2', narrative_order: 20, chapter_index: 2 }),
    ev({ id: 'e3', narrative_order: 30, chapter_index: 3 }),
  ]
}

describe('buildThreadRows sorting and states', () => {
  it('orders open/escalating, then dormant, converging, resolved', () => {
    const events = span()
    const result = buildThreadRows({
      events,
      threads: [
        th({ id: 'resolved', state: 'resolved', event_ids: ['e0', 'e1'], resolved_by_event_id: 'e1' }),
        th({ id: 'converging', state: 'converging', event_ids: ['e0', 'e2'] }),
        th({ id: 'dormant', state: 'dormant', dormant_chapters: 6, event_ids: ['e0'] }),
        th({ id: 'open', state: 'active', event_ids: ['e1', 'e2'] }),
      ],
    })
    expect(result.rows.map(row => row.id)).toEqual(['open', 'dormant', 'converging', 'resolved'])
  })

  it('labels active threads OPEN · seeded with the opening chapter', () => {
    const events = [
      ev({ id: 'e0', narrative_order: 0, chapter_index: 2, chapter_title: 'The Rooftop' }),
      ev({ id: 'e1', narrative_order: 10, chapter_index: 3 }),
    ]
    const result = buildThreadRows({
      events,
      threads: [th({ id: 't', state: 'active', opened_by_event_id: 'e0', event_ids: ['e0', 'e1'] })],
    })
    expect(result.rows[0].stateLabel).toBe('OPEN · seeded The Rooftop')
    expect(result.rows[0].stateColor).toBe('var(--status-warning)')
    expect(result.rows[0].openEnd).toBe(true)
    expect(result.rows[0].openingEventId).toBe('e0')
  })

  it('shows escalation progress as n/m from resolution over plotted beats', () => {
    const events = [
      ev({ id: 'e0', narrative_order: 0 }),
      ev({ id: 'e1', narrative_order: 1 }),
      ev({ id: 'e2', narrative_order: 2 }),
      ev({ id: 'e3', narrative_order: 3 }),
      ev({ id: 'e4', narrative_order: 4 }),
    ]
    const result = buildThreadRows({
      events,
      threads: [th({ id: 't', state: 'escalating', resolution: 0.4, event_ids: ['e0', 'e1', 'e2', 'e3', 'e4'] })],
    })
    expect(result.rows[0].stateLabel).toBe('ESCALATING · 2/5')
  })

  it('labels dormant threads with their dormant chapter count', () => {
    const result = buildThreadRows({
      events: span(),
      threads: [th({ id: 't', state: 'dormant', dormant_chapters: 6, event_ids: ['e0', 'e1'] })],
    })
    expect(result.rows[0].stateLabel).toBe('DORMANT 6 CH')
    expect(result.rows[0].category).toBe('dormant')
  })

  it('marks a resolved thread with later activity as REOPENED', () => {
    const events = span()
    const result = buildThreadRows({
      events,
      threads: [
        th({
          id: 't',
          state: 'resolved',
          event_ids: ['e0', 'e1', 'e3'],
          resolved_by_event_id: 'e1',
        }),
      ],
    })
    const row = result.rows[0]
    expect(row.stateLabel).toBe('REOPENED')
    expect(row.category).toBe('open')
    expect(row.openEnd).toBe(true)
    expect(row.ring).toBe(true)
    expect(row.note?.tone).toBe('error')
  })

  it('rings resolved threads at the resolution event and closes the line', () => {
    const events = span()
    const result = buildThreadRows({
      events,
      threads: [th({ id: 't', state: 'resolved', event_ids: ['e0', 'e2'], resolved_by_event_id: 'e2' })],
    })
    const row = result.rows[0]
    expect(row.stateLabel).toBe('RESOLVED')
    expect(row.stateColor).toBe('var(--text-muted)')
    expect(row.ring).toBe(true)
    expect(row.ringX).toBeCloseTo(2 / 3)
    expect(row.openEnd).toBe(false)
    expect(row.segments.every(segment => segment.kind !== 'open')).toBe(true)
  })
})

describe('buildThreadRows geometry', () => {
  it('normalizes x positions 0..1 over the whole manuscript', () => {
    const events = span()
    const result = buildThreadRows({
      events,
      threads: [th({ id: 't', state: 'resolved', event_ids: ['e0', 'e1', 'e2', 'e3'], resolved_by_event_id: 'e3' })],
    })
    const row = result.rows[0]
    expect(row.x0).toBe(0)
    expect(row.segments[0]).toMatchObject({ x1: 0, x2: 1, kind: 'solid' })
  })

  it('splits dormancy gaps (>= 3 chapters) into sparse-dash segments with a note', () => {
    const events = [
      ev({ id: 'e0', narrative_order: 0, chapter_index: 0 }),
      ev({ id: 'e1', narrative_order: 10, chapter_index: 1 }),
      ev({ id: 'e2', narrative_order: 20, chapter_index: 5 }),
      ev({ id: 'e3', narrative_order: 30, chapter_index: 6 }),
    ]
    const result = buildThreadRows({
      events,
      threads: [th({ id: 't', state: 'active', event_ids: ['e0', 'e1', 'e2', 'e3'] })],
    })
    const row = result.rows[0]
    // Last beat sits at the right edge, so the open continuation is zero-length
    // and omitted; the OPEN flag still shows via openEnd.
    expect(row.segments.map(segment => segment.kind)).toEqual(['solid', 'gap', 'solid'])
    expect(row.openEnd).toBe(true)
    expect(row.note?.text).toBe('dormant gap · 4 ch')
    expect(row.note?.x).toBeCloseTo(1 / 3)
  })

  it('extends unresolved threads to the right edge as an open dash', () => {
    const events = span()
    const result = buildThreadRows({
      events,
      threads: [th({ id: 't', state: 'active', event_ids: ['e0', 'e1'] })],
    })
    const last = result.rows[0].segments[result.rows[0].segments.length - 1]
    expect(last.kind).toBe('open')
    expect(last.x2).toBe(1)
  })

  it('uses story days as the x basis when at least 70% of events are anchored', () => {
    const events = [
      ev({ id: 'e0', narrative_order: 0, story_time: { day_offset: 0 } }),
      ev({ id: 'e1', narrative_order: 10, story_time: { day_offset: 10 } }),
      // Flashback written late but set on day 2: story-day basis pulls it left.
      ev({ id: 'e2', narrative_order: 20, story_time: { day_offset: 2 } }),
    ]
    const result = buildThreadRows({
      events,
      threads: [th({ id: 't', state: 'active', event_ids: ['e0', 'e2'] })],
    })
    expect(result.basis).toBe('story_day')
    // e2 sits at day 2 of 10, not at its narrative position (end).
    const solid = result.rows[0].segments.find(segment => segment.kind === 'solid')
    expect(solid?.x2).toBeCloseTo(0.2)
  })

  it('falls back to narrative order when anchors are sparse', () => {
    const events = [
      ev({ id: 'e0', narrative_order: 0, story_time: { day_offset: 0 } }),
      ev({ id: 'e1', narrative_order: 10 }),
      ev({ id: 'e2', narrative_order: 20 }),
    ]
    const result = buildThreadRows({ events, threads: [th({ id: 't', event_ids: ['e0', 'e1'] })] })
    expect(result.basis).toBe('narrative')
  })
})

describe('buildThreadRows cap, counts, colours, convergence', () => {
  it('caps at 40 rows keeping open threads and reports the remainder', () => {
    const events = Array.from({ length: 45 }, (_, index) =>
      ev({ id: `e${index}`, narrative_order: index, chapter_index: index % 20 }),
    )
    const threads = Array.from({ length: 45 }, (_, index) =>
      th({
        id: `t${index}`,
        state: index < 3 ? 'active' : 'resolved',
        event_ids: [`e${index}`],
        confidence: 0.5 + (index % 10) / 100,
      }),
    )
    const result = buildThreadRows({ events, threads })
    expect(result.rows.length).toBe(MAX_ROWS)
    expect(result.moreCount).toBe(5)
    expect(result.counts.all).toBe(45)
    const kept = new Set(result.rows.map(row => row.id))
    expect(kept.has('t0') && kept.has('t1') && kept.has('t2')).toBe(true)
  })

  it('counts categories across all threads, not just visible rows', () => {
    const result = buildThreadRows({
      events: span(),
      threads: [
        th({ id: 'a', state: 'active', event_ids: ['e0'] }),
        th({ id: 'b', state: 'dormant', dormant_chapters: 3, event_ids: ['e1'] }),
        th({ id: 'c', state: 'resolved', event_ids: ['e2'] }),
        th({ id: 'd', state: 'intentionally_deferred', event_ids: ['e3'] }),
      ],
    })
    expect(result.counts).toEqual({ all: 4, open: 1, dormant: 1, resolved: 2 })
  })

  it('assigns stable palette colours keyed by thread id', () => {
    const color = threadColor('thread-abc')
    expect(color).toBe(threadColor('thread-abc'))
    expect(THREAD_PALETTE).toContain(color)
    const result = buildThreadRows({
      events: span(),
      threads: [th({ id: 'thread-abc', event_ids: ['e0'] })],
    })
    expect(result.rows[0].color).toBe(color)
  })

  it('reports the first converging parent pair as row indices', () => {
    const events = span()
    const result = buildThreadRows({
      events,
      threads: [
        th({ id: 'a', state: 'active', event_ids: ['e0', 'e1'] }),
        th({ id: 'b', state: 'active', event_ids: ['e0', 'e2'] }),
        th({ id: 'child', state: 'converging', parent_ids: ['a', 'b'], event_ids: ['e2', 'e3'] }),
      ],
    })
    expect(result.converge).toBeDefined()
    const rowIds = result.rows.map(row => row.id)
    expect(result.converge?.fromRow).toBe(Math.min(rowIds.indexOf('a'), rowIds.indexOf('b')))
    expect(result.converge?.toRow).toBe(Math.max(rowIds.indexOf('a'), rowIds.indexOf('b')))
    expect(result.converge?.label).toContain('CONVERGE')
    expect(result.converge?.label).toContain('child')
  })

  it('returns an empty result for an empty fingerprint', () => {
    const result = buildThreadRows({ events: [], threads: [] })
    expect(result.rows).toEqual([])
    expect(result.moreCount).toBe(0)
    expect(result.converge).toBeUndefined()
  })
})
