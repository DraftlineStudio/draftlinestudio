import { describe, expect, it } from 'vitest'
import { buildStructureMapLayout, overviewCoverage } from './storyStructureMapModel'

const beat = (id: string, order: number, salience: number, extra = {}) => ({
  id, summary: id, evidence_ids: [`e-${id}`], fingerprint_event_ids: [`fp-${id}`],
  narrative_order: order, salience, confidence: .9, context_id: 'primary',
  story_time: { context_id: 'primary', day_offset: order, precision: 'day', confidence: .9 },
  chapter_title: `Chapter ${order + 1}`, ...extra,
})

describe('Story Structure map', () => {
  it('preserves structural thread coverage before filling by salience', () => {
    const events = Array.from({ length: 50 }, (_, index) => beat(`beat-${index}`, index, 1 - index / 100))
    const quietRequired = beat('quiet-thread', 51, .05)
    events.push(quietRequired)
    const visible = overviewCoverage({
      significant_events: events,
      narrative_threads: [{ id: 'thread', event_ids: ['quiet-thread'], evidence_ids: [], sequence_ids: [], scene_ids: [] }],
      arcs: [], scenes: [], sequences: [],
    })
    expect(visible).toHaveLength(36)
    expect(visible.some(event => event.id === 'quiet-thread')).toBe(true)
  })

  it('changes ordering between manuscript and fictional time', () => {
    const fingerprint = {
      contexts: [{ id: 'primary', kind: 'primary', label: 'Present', confidence: 1, source: 'auto' }],
      events: [], states: [], threads: [], diagnostics: [],
      structure: {
        significant_events: [
          beat('reader-first', 0, .8, { story_time: { context_id: 'primary', day_offset: 5, precision: 'day', confidence: .9 } }),
          beat('flashback', 1, .8, { story_time: { context_id: 'primary', day_offset: 1, precision: 'day', confidence: .9 } }),
        ], scenes: [], sequences: [], narrative_threads: [], arcs: [],
      },
    }
    const manuscript = buildStructureMapLayout(fingerprint as never, 'manuscript', 'event')
    const story = buildStructureMapLayout(fingerprint as never, 'story', 'event')
    expect(manuscript?.nodes.map(node => node.id)).toEqual(['reader-first', 'flashback'])
    expect(story?.nodes.map(node => node.id)).toEqual(['flashback', 'reader-first'])
  })
})
