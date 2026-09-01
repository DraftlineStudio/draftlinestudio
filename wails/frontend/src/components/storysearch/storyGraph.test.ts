import { describe, expect, it } from 'vitest'
import type { StoryTimelineEvent, StoryTimelineResult } from '../../types/draftline'
import {
  buildStoryGraph,
  deriveCharacterLanes,
  deriveLinks,
  deriveThreads,
  eventTier,
  positionInChapter,
} from './storyGraph'

function event(partial: Partial<StoryTimelineEvent> & { id: string }): StoryTimelineEvent {
  return {
    evidence_ids: [],
    primary_type: 'state',
    event_types: ['state'],
    text: partial.id,
    source_text: partial.id,
    chapter_id: `ch-${partial.chapter_index ?? 0}`,
    chapter_index: 0,
    chapter_title: 'Chapter 1',
    section: 'body',
    section_index: 0,
    paragraph_index: 0,
    sentence_index: 0,
    start_offset: 0,
    time_kind: 'manuscript',
    time_label: 'Manuscript order',
    confidence: 0.8,
    status: 'detected',
    ...partial,
  }
}

function result(events: StoryTimelineEvent[]): StoryTimelineResult {
  const chapterIndexes = [...new Set(events.map(item => item.chapter_index))].sort((a, b) => a - b)
  return {
    success: true,
    engine: 'test',
    events,
    chapters: chapterIndexes.map(index => ({
      chapter_index: index,
      chapter_title: `Chapter ${index + 1}`,
      event_count: events.filter(item => item.chapter_index === index).length,
      explicit_time_count: 0,
    })),
    characters: [],
    locations: [],
    event_types: [],
    explicit_time_count: 0,
    relative_time_count: 0,
  }
}

describe('eventTier', () => {
  it('treats decisive evidence types as key beats', () => {
    expect(eventTier(event({ id: 'a', primary_type: 'turning_point' }))).toBe(0)
    expect(eventTier(event({ id: 'b', primary_type: 'discovery' }))).toBe(0)
  })

  it('promotes anything the author confirmed or pinned', () => {
    expect(eventTier(event({ id: 'a', primary_type: 'state', status: 'confirmed' }))).toBe(0)
    expect(eventTier(event({ id: 'b', primary_type: 'state', pinned: true }))).toBe(0)
  })

  it('promotes a multi-character scene with an explicit time anchor', () => {
    const anchored = event({ id: 'a', character_ids: ['x', 'y'], time_kind: 'anchored' })
    expect(eventTier(anchored)).toBe(0)
  })

  it('ranks introductions and shared scenes as notable, everything else minor', () => {
    expect(eventTier(event({ id: 'a', primary_type: 'introduction' }))).toBe(1)
    expect(eventTier(event({ id: 'b', character_ids: ['x', 'y'] }))).toBe(1)
    expect(eventTier(event({ id: 'c' }))).toBe(2)
  })
})

describe('deriveThreads', () => {
  it('makes a story strand from a recurring non-person anchor', () => {
    const events = [
      event({ id: '1', thread_terms: [{ text: 'IBM', label: 'ORG' }], character_ids: ['evan'], character_names: ['Evan'], chapter_index: 0 }),
      event({ id: '2', thread_terms: [{ text: 'One IBM Plaza', label: 'FAC' }], character_ids: ['eve'], character_names: ['Eveline'], chapter_index: 2 }),
    ]
    const { lanes, laneOf } = deriveThreads(events)
    expect(lanes).toHaveLength(1)
    expect(lanes[0].name).toBe('IBM')
    expect(lanes[0].from).toBe(0)
    expect(lanes[0].to).toBe(2)
    expect(laneOf.get('1')).toBe(lanes[0].id)
  })

  it('does not promote a one-off named term into a story strand', () => {
    const events = [
      event({ id: '1', thread_terms: [{ text: 'Blue folder', label: 'PRODUCT' }], character_ids: ['evan'], character_names: ['Evan'] }),
      event({ id: '2', character_ids: ['evan'], character_names: ['Evan'] }),
    ]
    const { lanes } = deriveThreads(events)
    expect(lanes.map(lane => lane.name)).toEqual(['Evan-led beats'])
    expect(lanes[0].eventCount).toBe(2)
  })

  it('routes uncast events to an explicitly unlinked rail', () => {
    const { lanes, laneOf } = deriveThreads([event({ id: '1' })])
    expect(lanes[0].name).toBe('Unlinked beats')
    expect(laneOf.get('1')).toBeDefined()
  })

  it('keeps every recurring strand membership on a crossover beat', () => {
    const events = [
      event({ id: '1', thread_terms: [{ text: 'IBM', label: 'ORG' }, { text: 'Tunnel', label: 'LOC' }] }),
      event({ id: '2', thread_terms: [{ text: 'IBM', label: 'ORG' }] }),
      event({ id: '3', thread_terms: [{ text: 'Tunnel', label: 'LOC' }] }),
    ]
    const { memberships } = deriveThreads(events)
    expect(memberships.get('1')).toHaveLength(2)
  })

  it('assigns every event a lane', () => {
    const events = [
      event({ id: '1', character_ids: ['a'], character_names: ['A'] }),
      event({ id: '2', character_ids: ['b'], character_names: ['B'] }),
      event({ id: '3' }),
    ]
    const { laneOf } = deriveThreads(events)
    expect([...laneOf.keys()].sort()).toEqual(['1', '2', '3'])
  })
})

describe('deriveCharacterLanes', () => {
  it('orders lanes by appearance count', () => {
    const events = [
      event({ id: '1', character_ids: ['evan'], character_names: ['Evan'] }),
      event({ id: '2', character_ids: ['evan'], character_names: ['Evan'] }),
      event({ id: '3', character_ids: ['marcus'], character_names: ['Marcus'] }),
    ]
    const { lanes } = deriveCharacterLanes(events, [])
    expect(lanes.map(lane => lane.name)).toEqual(['Evan', 'Marcus'])
    expect(lanes[0].eventCount).toBe(2)
  })

  it('places a beat on its lead character rail', () => {
    const events = [event({ id: '1', character_ids: ['evan', 'eve'], character_names: ['Evan', 'Eveline'] })]
    const { laneOf } = deriveCharacterLanes(events, [])
    expect(laneOf.get('1')).toBe('evan')
  })

  it('leaves uncast events off the graph rather than guessing a lane', () => {
    const { lanes, laneOf } = deriveCharacterLanes([event({ id: '1' })], [])
    expect(lanes).toHaveLength(0)
    expect(laneOf.get('1')).toBeUndefined()
  })
})

describe('deriveLinks', () => {
  it('links the nearest forward beat sharing a character', () => {
    const graph = buildStoryGraph(result([
      event({ id: '1', character_ids: ['evan'], character_names: ['Evan'], chapter_index: 0, primary_type: 'discovery' }),
      event({ id: '2', character_ids: ['evan'], character_names: ['Evan'], chapter_index: 1, primary_type: 'discovery' }),
    ]), 'characters', 0)
    expect(graph.links).toHaveLength(1)
    expect(graph.links[0]).toMatchObject({ from: 0, to: 1 })
    expect(graph.links[0].reason).toContain('Evan')
  })

  it('only ever links forward in manuscript order', () => {
    const graph = buildStoryGraph(result([
      event({ id: '1', character_ids: ['a'], character_names: ['A'], chapter_index: 0, primary_type: 'discovery' }),
      event({ id: '2', character_ids: ['a'], character_names: ['A'], chapter_index: 1, primary_type: 'discovery' }),
      event({ id: '3', character_ids: ['a'], character_names: ['A'], chapter_index: 2, primary_type: 'discovery' }),
    ]), 'characters', 0)
    for (const link of graph.links) expect(link.to).toBeGreaterThan(link.from)
  })

  it('produces no links for a single beat', () => {
    expect(deriveLinks([])).toEqual([])
  })
})

describe('buildStoryGraph', () => {
  it('filters by zoom tier and reports the untruncated total', () => {
    const events = [
      event({ id: 'key', primary_type: 'turning_point', character_ids: ['a'], character_names: ['A'] }),
      event({ id: 'minor', primary_type: 'state', character_ids: ['a'], character_names: ['A'] }),
    ]
    const keyOnly = buildStoryGraph(result(events), 'characters', 0)
    expect(keyOnly.nodes).toHaveLength(1)
    expect(keyOnly.totalEvents).toBe(2)

    const everything = buildStoryGraph(result(events), 'characters', 2)
    expect(everything.nodes).toHaveLength(2)
  })

  it('counts only plotted beats per chapter', () => {
    const graph = buildStoryGraph(result([
      event({ id: '1', chapter_index: 0, primary_type: 'discovery', character_ids: ['a'], character_names: ['A'] }),
      event({ id: '2', chapter_index: 0, primary_type: 'state', character_ids: ['a'], character_names: ['A'] }),
    ]), 'characters', 0)
    expect(graph.chapters[0].count).toBe(1)
  })

  it('survives an empty timeline', () => {
    const graph = buildStoryGraph(result([]), 'threads', 2)
    expect(graph.nodes).toEqual([])
    expect(graph.lanes).toEqual([])
    expect(graph.links).toEqual([])
  })
})

describe('positionInChapter', () => {
  it('keeps every beat inside the column, away from the grid lines', () => {
    const samples = [0, 1, 5, 50, 400, 5000]
    for (const paragraph_index of samples) {
      const node = { index: 0, event: event({ id: 'x', paragraph_index }), laneId: 'a', tier: 0, major: true }
      const position = positionInChapter(node)
      expect(position).toBeGreaterThanOrEqual(0.12)
      expect(position).toBeLessThanOrEqual(0.86)
    }
  })

  it('preserves paragraph order within a chapter', () => {
    const early = positionInChapter({ index: 0, event: event({ id: 'a', paragraph_index: 2 }), laneId: 'l', tier: 0, major: false })
    const late = positionInChapter({ index: 1, event: event({ id: 'b', paragraph_index: 60 }), laneId: 'l', tier: 0, major: false })
    expect(late).toBeGreaterThan(early)
  })
})
