import { describe, expect, it } from 'vitest'
import { types } from '../../../wailsjs/go/models'
import type { EvidenceRecord } from '../../types/draftline'
import {
  appendStoryDayCorrection,
  buildManuscriptPath,
  buildStoryMapLayout,
  describeEvent,
  deriveEras,
  eventDay,
  isUncertain,
  truncateLabel,
  weekdayFromLabel,
  type MapNode,
} from './storyMapModel'

// ── Fixtures ────────────────────────────────────────────────────────────────

let nextOrder = 0

function event(partial: Record<string, unknown> & { id: string }): types.FingerprintEvent {
  return types.FingerprintEvent.createFrom({
    summary: `summary for ${partial.id}`,
    evidence_ids: [],
    kinds: ['action'],
    context_id: 'ctx-primary',
    story_time: { context_id: 'ctx-primary', precision: 'day', confidence: 0.9 },
    chapter_id: 'ch-0',
    chapter_index: 0,
    chapter_title: 'Chapter 1',
    paragraph_index: 0,
    start_offset: 0,
    narrative_order: nextOrder++,
    importance: 0.5,
    confidence: 0.9,
    status: 'detected',
    ...partial,
  })
}

function fingerprint(partial: Record<string, unknown> = {}): types.StoryFingerprint {
  nextOrder = 0
  return types.StoryFingerprint.createFrom({
    content_hash: 'hash',
    engine: 'test',
    last_analyzed: '',
    version: 1,
    contexts: [
      { id: 'ctx-primary', kind: 'primary', label: 'Interrogation', confidence: 1, source: 'inferred' },
    ],
    temporal_constraints: [],
    assertions: [],
    events: [],
    states: [],
    threads: [],
    diagnostics: [],
    author_model: {},
    ...partial,
  })
}

function anchored(id: string, day: number, extra: Record<string, unknown> = {}): Record<string, unknown> & { id: string } {
  return {
    id,
    story_time: { context_id: 'ctx-primary', precision: 'day', confidence: 0.9, day_offset: day },
    ...extra,
  }
}

const memoryContext = { id: 'ctx-memory', kind: 'memory', label: 'The weekend', confidence: 0.8, source: 'inferred' }

function record(id: string, text: string): EvidenceRecord {
  return {
    id,
    kind: 'event',
    evidence_type: 'state',
    chapter_id: 'ch-0',
    chapter_index: 0,
    section: 'body',
    section_index: 4,
    paragraph_index: 2,
    sentence_index: 0,
    start_offset: 10,
    end_offset: 40,
    text,
    confidence: 0.9,
    rationale: '',
    status: 'detected',
    source: 'auto',
  }
}

// ── Helpers ─────────────────────────────────────────────────────────────────

describe('truncateLabel', () => {
  it('keeps short labels and truncates long ones with an ellipsis', () => {
    expect(truncateLabel('short')).toBe('short')
    const long = truncateLabel('a very long event summary indeed')
    expect(long.length).toBeLessThanOrEqual(18)
    expect(long.endsWith('…')).toBe(true)
  })
})

describe('eventDay', () => {
  it('prefers day_offset, falls back to earliest_day, else null', () => {
    expect(eventDay(event(anchored('a', 4)))).toBe(4)
    expect(eventDay(event({ id: 'b', story_time: { context_id: 'x', precision: 'range', confidence: 0.8, earliest_day: 2 } }))).toBe(2)
    expect(eventDay(event({ id: 'c', story_time: { context_id: 'x', precision: 'unknown', confidence: 0.3 } }))).toBeNull()
  })
})

describe('weekdayFromLabel', () => {
  it('finds a named weekday and returns its abbreviation', () => {
    expect(weekdayFromLabel('Friday, morning')).toBe('FRI')
    expect(weekdayFromLabel('two days after the rooftop')).toBeNull()
    expect(weekdayFromLabel(undefined)).toBeNull()
  })
})

describe('isUncertain', () => {
  it('flags relative/unknown precision and low confidence', () => {
    const fp = fingerprint()
    expect(isUncertain(event({ id: 'a', story_time: { context_id: 'x', precision: 'relative', confidence: 0.95 } }), fp)).toBe(true)
    expect(isUncertain(event({ id: 'b', story_time: { context_id: 'x', precision: 'day', confidence: 0.5 } }), fp)).toBe(true)
    expect(isUncertain(event({ id: 'c', story_time: { context_id: 'x', precision: 'day', confidence: 0.9 } }), fp)).toBe(false)
  })

  it('trusts an applied author story_day correction', () => {
    const fp = fingerprint({
      author_model: { corrections: [{ id: 'k', target_id: 'a', kind: 'story_day', value: '3', status: 'applied' }] },
    })
    const uncertain = event({ id: 'a', story_time: { context_id: 'x', precision: 'relative', confidence: 0.4 } })
    expect(isUncertain(uncertain, fp)).toBe(false)
  })
})

// ── Eras ────────────────────────────────────────────────────────────────────

describe('deriveEras', () => {
  it('splits the primary context into contiguous day segments', () => {
    const fp = fingerprint({
      events: [
        event(anchored('a', 3)), event(anchored('b', 4)), event(anchored('c', 5)),
        event(anchored('d', 9)),
        event({ id: 'e', story_time: { context_id: 'ctx-primary', precision: 'unknown', confidence: 0.3 } }),
      ],
    })
    const eras = deriveEras(fp)
    const labels = eras.map(era => era.label)
    expect(labels).toContain('DAY 3–5')
    expect(labels).toContain('DAY 9')
    // Floating remainder keeps the context label.
    expect(labels).toContain('INTERROGATION')
    // Day segments come before the floating remainder.
    expect(labels.indexOf('DAY 3–5')).toBeLessThan(labels.indexOf('INTERROGATION'))
  })

  it('places unanchored memory frames before the day timeline', () => {
    const fp = fingerprint({
      contexts: [
        { id: 'ctx-primary', kind: 'primary', label: 'Present', confidence: 1, source: 'inferred' },
        memoryContext,
      ],
      events: [
        event(anchored('a', 1)),
        event({ id: 'm', context_id: 'ctx-memory', story_time: { context_id: 'ctx-memory', precision: 'relative', confidence: 0.5 } }),
      ],
    })
    const eras = deriveEras(fp)
    expect(eras[0].kind).toBe('memory')
    expect(eras[0].label).toBe('THE WEEKEND')
    expect(eras[1].label).toBe('DAY 1')
  })
})

// ── Layout ──────────────────────────────────────────────────────────────────

describe('buildStoryMapLayout', () => {
  it('puts the primary context on the top lane and nested contexts below', () => {
    const fp = fingerprint({
      contexts: [
        { id: 'ctx-primary', kind: 'primary', label: 'Present', confidence: 1, source: 'inferred' },
        memoryContext,
      ],
      events: [
        event(anchored('a', 1)),
        event({ id: 'm', context_id: 'ctx-memory' }),
      ],
    })
    const layout = buildStoryMapLayout(fp)
    const a = layout.nodes.find(node => node.id === 'a')
    const m = layout.nodes.find(node => node.id === 'm')
    expect(a?.lane).toBe(0)
    expect(m?.lane).toBe(1)
    expect((a?.y ?? 0) < (m?.y ?? 0)).toBe(true)
    expect(layout.laneCount).toBe(2)
    expect(layout.height).toBe(236)
  })

  it('clamps a fourth context into the third lane', () => {
    const fp = fingerprint({
      contexts: [
        { id: 'ctx-primary', kind: 'primary', label: 'Present', confidence: 1, source: 'inferred' },
        memoryContext,
        { id: 'ctx-dream', kind: 'dream', label: 'Dreams', confidence: 0.7, source: 'inferred' },
        { id: 'ctx-sim', kind: 'simulation', label: 'The sim', confidence: 0.7, source: 'author' },
      ],
      events: [
        event(anchored('a', 1)),
        event({ id: 'm1', context_id: 'ctx-memory' }),
        event({ id: 'm2', context_id: 'ctx-memory' }),
        event({ id: 'd1', context_id: 'ctx-dream' }),
        event({ id: 's1', context_id: 'ctx-sim' }),
      ],
    })
    const layout = buildStoryMapLayout(fp)
    expect(layout.laneCount).toBe(3)
    const lanes = new Set(layout.nodes.map(node => node.lane))
    expect([...lanes].sort()).toEqual([0, 1, 2])
    // The two smallest contexts share the bottom lane.
    const d1 = layout.nodes.find(node => node.id === 'd1')
    const s1 = layout.nodes.find(node => node.id === 's1')
    expect(d1?.lane).toBe(2)
    expect(s1?.lane).toBe(2)
  })

  it('sizes nodes by importance and dashes uncertain placements', () => {
    const fp = fingerprint({
      events: [
        event(anchored('big', 1, { importance: 0.8 })),
        event(anchored('small', 1, { importance: 0.4 })),
        event({ id: 'loose', story_time: { context_id: 'ctx-primary', precision: 'relative', confidence: 0.5 } }),
      ],
    })
    const layout = buildStoryMapLayout(fp)
    const big = layout.nodes.find(node => node.id === 'big')
    const small = layout.nodes.find(node => node.id === 'small')
    const loose = layout.nodes.find(node => node.id === 'loose')
    expect(big?.r).toBe(8)
    expect(big?.major).toBe(true)
    expect(small?.r).toBe(6.5)
    expect(big?.dashed).toBe(false)
    expect(loose?.dashed).toBe(true)
  })

  it('orders nodes and ordinals by narrative order across eras', () => {
    const fp = fingerprint({
      contexts: [
        { id: 'ctx-primary', kind: 'primary', label: 'Present', confidence: 1, source: 'inferred' },
        memoryContext,
      ],
      events: [
        event(anchored('first', 2)),
        event({ id: 'recalled', context_id: 'ctx-memory' }),
        event(anchored('third', 3)),
      ],
    })
    const layout = buildStoryMapLayout(fp)
    expect(layout.nodes.map(node => node.id)).toEqual(['first', 'recalled', 'third'])
    expect(layout.nodes.map(node => node.ord)).toEqual([1, 2, 3])
  })

  it('emits one tick per anchored day, with a weekday when the label names one', () => {
    const fp = fingerprint({
      events: [
        event(anchored('a', 4, { story_time: { context_id: 'ctx-primary', precision: 'day', confidence: 0.9, day_offset: 4, label: 'Friday, morning' } })),
        event(anchored('b', 4)),
        event(anchored('c', 5)),
      ],
    })
    const layout = buildStoryMapLayout(fp)
    expect(layout.ticks.map(tick => tick.label)).toEqual(['DAY 4 · FRI', 'DAY 5'])
  })

  it('marks fulfilled checkpoints at their first matched event', () => {
    const fp = fingerprint({
      events: [event(anchored('a', 1)), event(anchored('b', 2))],
      author_model: {
        checkpoints: [
          {
            id: 'cp-1', title: 'released', kind: 'act', status: 'fulfilled', source: 'author',
            matched_event_ids: ['b', 'a'],
            requirements: [{ text: 'x', satisfied: true }, { text: 'y', satisfied: false }],
          },
          { id: 'cp-2', title: 'planned only', kind: 'act', status: 'planned', source: 'author', matched_event_ids: ['a'] },
        ],
      },
    })
    const layout = buildStoryMapLayout(fp)
    expect(layout.checkpoints).toHaveLength(1)
    const mark = layout.checkpoints[0]
    const a = layout.nodes.find(node => node.id === 'a')
    expect(mark.x).toBe(a?.x)
    expect(mark.ratio).toBe('1/2')
    expect(mark.title).toBe('released')
  })

  it('caps rendered nodes at the importance ceiling and reports the rest', () => {
    const events = Array.from({ length: 150 }, (_, index) =>
      event(anchored(`ev-${index}`, 1, { importance: index / 150 })))
    const layout = buildStoryMapLayout(fingerprint({ events }))
    expect(layout.nodes).toHaveLength(120)
    expect(layout.hiddenCount).toBe(30)
    // The lowest-importance events are the ones dropped.
    expect(layout.nodes.some(node => node.id === 'ev-0')).toBe(false)
    expect(layout.nodes.some(node => node.id === 'ev-149')).toBe(true)
  })

  it('draws the loop-back only for a first/last-chapter near-duplicate diagnostic', () => {
    const base = {
      events: [
        event(anchored('first', 1, { chapter_index: 0 })),
        event(anchored('mid', 2, { chapter_index: 5 })),
        event(anchored('last', 3, { chapter_index: 11 })),
      ],
    }
    const withLoop = buildStoryMapLayout(fingerprint({
      ...base,
      diagnostics: [{
        id: 'diag-loop', kind: 'near_duplicate_chapter', severity: 'review',
        title: 'Prologue and Epilogue are nearly identical', detail: '',
        event_ids: ['first', 'last'], chapter_indices: [0, 11], confidence: 0.98,
      }],
    }))
    expect(withLoop.loop).not.toBeNull()
    expect(withLoop.loop?.label).toBe('possible loop — see Review')
    expect(withLoop.loop?.diagnosticId).toBe('diag-loop')

    const noDiag = buildStoryMapLayout(fingerprint(base))
    expect(noDiag.loop).toBeNull()

    const midOnly = buildStoryMapLayout(fingerprint({
      ...base,
      diagnostics: [{
        id: 'diag-mid', kind: 'near_duplicate_chapter', severity: 'review',
        title: '', detail: '', chapter_indices: [4, 5], confidence: 0.9,
      }],
    }))
    expect(midOnly.loop).toBeNull()
  })

  it('draws up to two dashed dangles for open threads, from their seed nodes', () => {
    const fp = fingerprint({
      events: [event(anchored('a', 1)), event(anchored('b', 2)), event(anchored('c', 3))],
      threads: [
        { id: 't1', label: 'who is the jumper?', kind: 'question', state: 'active', opened_by_event_id: 'a', event_ids: ['a', 'b', 'c'], resolution: 0, confidence: 0.9, source: 'inferred' },
        { id: 't2', label: 'what the tunnel hides', kind: 'mystery', state: 'seeded', opened_by_event_id: 'b', event_ids: ['b'], resolution: 0, confidence: 0.8, source: 'inferred' },
        { id: 't3', label: 'a third open one', kind: 'threat', state: 'escalating', opened_by_event_id: 'c', event_ids: ['c'], resolution: 0, confidence: 0.7, source: 'inferred' },
        { id: 't4', label: 'already paid', kind: 'commitment', state: 'resolved', opened_by_event_id: 'a', event_ids: ['a'], resolution: 1, confidence: 0.9, source: 'inferred' },
      ],
    })
    const layout = buildStoryMapLayout(fp)
    expect(layout.dangles).toHaveLength(2)
    expect(layout.dangles[0].threadId).toBe('t1')
    expect(layout.dangles[0].label).toBe('OPEN · who is the jumper? →')
    const a = layout.nodes.find(node => node.id === 'a')
    expect(layout.dangles[0].d.startsWith(`M ${a?.x} `)).toBe(true)
  })

  it('flags eras touched by an undecided diagnostic as anomalies', () => {
    const fp = fingerprint({
      events: [event(anchored('a', 1))],
      diagnostics: [{
        id: 'diag-1', kind: 'checkpoint_too_early', severity: 'review',
        title: '', detail: '', event_ids: ['a'], confidence: 0.9,
      }],
    })
    expect(buildStoryMapLayout(fp).eras[0].anomaly).toBe(true)
  })
})

describe('buildManuscriptPath', () => {
  const node = (x: number, y: number): MapNode => ({
    id: `${x}-${y}`, x, y, r: 6.5, lane: 0, ord: 0, label: '', kind: 'primary',
    eraId: 'e', dashed: false, major: false, narrativeOrder: 0,
  })

  it('lifts same-lane hops away from the lane and dips between lanes straight', () => {
    const top = 66
    const path = buildManuscriptPath([node(10, 66), node(60, 66), node(110, 122)], top)
    expect(path.startsWith('M 10 66')).toBe(true)
    // Same top-lane hop: control points lifted 14px up.
    expect(path).toContain('C 35 52, 35 52, 60 66')
    // Lane change: no lift.
    expect(path).toContain('C 85 66, 85 122, 110 122')
  })
})

// ── Detail pane ─────────────────────────────────────────────────────────────

describe('describeEvent', () => {
  const evidence = new Map<string, EvidenceRecord>([
    ['rec-1', record('rec-1', 'The call said jumper. What waited on that rooftop was nothing so simple.')],
  ])

  it('assembles the full detail model from fingerprint joins', () => {
    const fp = fingerprint({
      contexts: [
        { id: 'ctx-primary', kind: 'primary', label: 'Present', confidence: 1, source: 'inferred' },
        memoryContext,
      ],
      events: [event({
        id: 'ev-1',
        context_id: 'ctx-memory',
        summary: 'The rooftop jumper call',
        evidence_ids: ['rec-1'],
        character_names: ['Hanlon', 'the jumper'],
        locations: [{ text: 'the rooftop', label: 'the rooftop' }],
        chapter_title: 'Chapter 2',
        importance: 0.9,
        story_time: { context_id: 'ctx-memory', precision: 'day', confidence: 0.9, day_offset: 2, label: 'Sunday night, late' },
      })],
      states: [
        { id: 's1', entity_id: 'hanlon', entity_name: 'Hanlon', kind: 'location', value: 'rooftop', context_id: 'ctx-memory', start_event_id: 'ev-1', evidence_ids: [], persistent: false, confidence: 0.9 },
        { id: 's2', entity_id: 'x', entity_name: 'Other', kind: 'location', value: 'elsewhere', context_id: 'ctx-memory', start_event_id: 'ev-other', evidence_ids: [], persistent: false, confidence: 0.9 },
      ],
      threads: [
        { id: 't1', label: 'who is the jumper?', kind: 'question', state: 'active', opened_by_event_id: 'ev-1', event_ids: ['ev-1'], resolution: 0, confidence: 0.9, source: 'inferred' },
        { id: 't2', label: 'the weekend frame', kind: 'temporal', state: 'resolved', resolved_by_event_id: 'ev-1', event_ids: ['ev-1'], resolution: 1, confidence: 0.9, source: 'inferred' },
        { id: 't3', label: 'gary is watching', kind: 'threat', state: 'dormant', event_ids: ['ev-1', 'ev-9'], resolution: 0.2, confidence: 0.7, source: 'inferred' },
      ],
    })
    const detail = describeEvent(fp, evidence, 'ev-1')
    expect(detail).not.toBeNull()
    expect(detail?.kindChip).toBe('Recalled · major')
    expect(detail?.kind).toBe('memory')
    expect(detail?.when).toBe('Sunday night, late')
    expect(detail?.title).toBe('The rooftop jumper call')
    expect(detail?.location).toBe('Chapter 2 · the rooftop')
    expect(detail?.quote).toContain('The call said jumper.')
    expect(detail?.needsAnchor).toBe(false)
    expect(detail?.stateChanges).toEqual(['location: Hanlon — rooftop'])
    expect(detail?.obligations).toEqual([
      { symbol: '○', tone: 'open', text: 'who is the jumper? — opened' },
      { symbol: '✓', tone: 'resolved', text: 'the weekend frame — resolved' },
      { symbol: '!', tone: 'flagged', text: 'gary is watching — dormant' },
    ])
    expect(detail?.chips).toEqual(['Hanlon', 'the jumper'])
    expect(detail?.chapterLabel).toBe('Chapter 2')
    expect(detail?.navigation).toEqual({
      section: 'body',
      sectionIndex: 4,
      query: 'The call said jumper. What waited on that rooftop was nothing so simple.'.slice(0, 80),
    })
  })

  it('reports unplaced timing and the anchor prompt for weak inferences', () => {
    const fp = fingerprint({
      events: [event({
        id: 'ev-loose',
        story_time: { context_id: 'ctx-primary', precision: 'relative', confidence: 0.62, label: 'two days after the rooftop', earliest_day: 4 },
      })],
    })
    const detail = describeEvent(fp, new Map(), 'ev-loose')
    expect(detail?.when).toBe('two days after the rooftop')
    expect(detail?.needsAnchor).toBe(true)
    expect(detail?.anchorDay).toBe(4)
    expect(detail?.anchorPercent).toBe(62)
    expect(detail?.navigation).toBeNull()
  })

  it('suppresses the anchor prompt once an author correction exists', () => {
    const fp = fingerprint({
      events: [event({
        id: 'ev-loose',
        story_time: { context_id: 'ctx-primary', precision: 'relative', confidence: 0.62 },
      })],
      author_model: { corrections: [{ id: 'c', target_id: 'ev-loose', kind: 'story_day', value: '4', status: 'applied' }] },
    })
    expect(describeEvent(fp, new Map(), 'ev-loose')?.needsAnchor).toBe(false)
  })

  it('falls back to Chapter N and unplaced confidence when data is missing', () => {
    const fp = fingerprint({
      events: [event({
        id: 'ev-bare',
        chapter_title: '',
        chapter_index: 3,
        story_time: { context_id: 'ctx-primary', precision: 'unknown', confidence: 0.31 },
      })],
    })
    const detail = describeEvent(fp, new Map(), 'ev-bare')
    expect(detail?.chapterLabel).toBe('Chapter 4')
    expect(detail?.when).toBe('unplaced · 31%')
  })

  it('returns null for an unknown event id', () => {
    expect(describeEvent(fingerprint(), new Map(), 'nope')).toBeNull()
  })
})

// ── Author-model write ──────────────────────────────────────────────────────

describe('appendStoryDayCorrection', () => {
  it('pins day_offset ?? earliest_day ?? 0 with an applied correction', () => {
    const fp = fingerprint({
      events: [
        event({ id: 'a', story_time: { context_id: 'x', precision: 'relative', confidence: 0.6, day_offset: 7, earliest_day: 5 } }),
        event({ id: 'b', story_time: { context_id: 'x', precision: 'relative', confidence: 0.6, earliest_day: 5 } }),
        event({ id: 'c', story_time: { context_id: 'x', precision: 'unknown', confidence: 0.3 } }),
      ],
    })
    expect(appendStoryDayCorrection(fp, 'a')?.day).toBe(7)
    expect(appendStoryDayCorrection(fp, 'b')?.day).toBe(5)
    const result = appendStoryDayCorrection(fp, 'c')
    expect(result?.day).toBe(0)
    const correction = result?.model.corrections.find(entry => entry.target_id === 'c')
    expect(correction?.kind).toBe('story_day')
    expect(correction?.value).toBe('0')
    // The engine's status vocabulary: "active" while the target exists.
    expect(correction?.status).toBe('active')
  })

  it('preserves the existing author model and replaces a prior day pin', () => {
    const fp = fingerprint({
      events: [event({ id: 'a', story_time: { context_id: 'x', precision: 'relative', confidence: 0.6, day_offset: 7 } })],
      author_model: {
        canon: [{ id: 'canon-1', subject: 's', predicate: 'p', object: 'o', polarity: 'positive' }],
        corrections: [
          { id: 'other', target_id: 'zz', kind: 'context', value: 'ctx-sim', status: 'applied' },
          { id: 'old-pin', target_id: 'a', kind: 'story_day', value: '2', status: 'applied' },
        ],
      },
    })
    const result = appendStoryDayCorrection(fp, 'a')
    expect(result?.model.canon).toHaveLength(1)
    const pins = result?.model.corrections.filter(entry => entry.kind === 'story_day' && entry.target_id === 'a')
    expect(pins).toHaveLength(1)
    expect(pins?.[0].value).toBe('7')
    expect(result?.model.corrections.some(entry => entry.id === 'other')).toBe(true)
  })

  it('returns null for an unknown event', () => {
    expect(appendStoryDayCorrection(fingerprint(), 'nope')).toBeNull()
  })
})
