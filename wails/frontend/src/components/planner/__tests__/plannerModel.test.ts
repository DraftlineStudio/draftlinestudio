import { describe, expect, it } from 'vitest'
import {
  bookChapters, codexPeople, columnConnectors, deadIdeaBlock, displayCards, emptyPlanner, ensurePlanner,
  laneRows, layoutMetrics, MAIN_LANE_ID, parseOutline, rebindWho, statusOf,
  whoIncludes, whoNames, whoText, type CodexPerson, type PlannerChapter,
} from '../plannerModel'
import { generatedSynopsis, synopsisMarkdown } from '../plannerSynopsis'
import type { BookData, PlannerData, PlannerLane } from '../../../types/draftline'

// All prose here is invented for these tests.

const codex: CodexPerson[] = [
  { id: 'c-rhea', name: 'Rhea Marsh', aliases: ['Rhea Marsh', 'Rhea'] },
  { id: 'c-tomas', name: 'Tomas', aliases: ['Tomas'] },
]
const lanes: PlannerLane[] = [
  { id: MAIN_LANE_ID, name: 'Main plot', kind: 'main', color: '#5aafe0' },
  { id: 'lane-rhea', name: 'Rhea Marsh', kind: 'character', color: '#F472B6', character_id: 'c-rhea' },
]
const chapters: PlannerChapter[] = [
  { id: 'ch-1', num: 1, title: 'The Beacon', words: 900, scenes: 2, drafted: true },
  { id: 'ch-2', num: 2, title: '', words: 0, scenes: 0, drafted: false },
  { id: 'ch-3', num: 3, title: 'Landfall', words: 400, scenes: 1, drafted: true },
]

describe('Import Outline rules', () => {
  it('turns headings into chapter positions and list items into cards, naming the chapter from the heading', () => {
    const { proposals, titles } = parseOutline(
      '# Chapter 1 — The Beacon\n- Rhea climbs the tower. The lamp is dark.\n- Tomas wakes the keeper.\n\n# Chapter 3\n1. The boat lands at dawn.\n',
      chapters, codex, lanes,
    )
    expect(titles).toEqual({ 1: 'The Beacon' })
    expect(proposals.map(p => [p.chapterNum, p.title, p.synopsis, p.laneId, p.who])).toEqual([
      [1, 'Rhea climbs the tower.', 'The lamp is dark.', 'lane-rhea', ['c-rhea']],
      [1, 'Tomas wakes the keeper.', '', MAIN_LANE_ID, ['c-tomas']],
      [3, 'The boat lands at dawn.', '', MAIN_LANE_ID, []],
    ])
    expect(proposals.every(p => p.accepted)).toBe(true)
  })

  it('reaches past the last chapter and lets the reviewer create the missing ones', () => {
    const { proposals } = parseOutline('Chapter 5: Reckoning\n- The keeper confesses.', chapters, codex, lanes)
    expect(proposals[0].chapterNum).toBe(5)
  })

  it('lands recognised beat names at their manuscript percentage on the main line', () => {
    const many = Array.from({ length: 10 }, (_, i) => ({ ...chapters[0], id: `ch-${i + 1}`, num: i + 1 }))
    const { proposals } = parseOutline('Midpoint: Rhea learns who lit the beacon.\nClimax — the tower falls.', many, codex, lanes)
    expect(proposals.map(p => [p.title, p.chapterNum])).toEqual([
      ['Midpoint — Rhea learns who lit the beacon.', 5],
      ['Climax — the tower falls.', 9],
    ])
  })

  it('clusters an unstructured short story into scene-sized proposals across the drafted chapters', () => {
    // The design brief: a short story pasted as the bones of a novel should
    // land as a few scene-sized cards, not one card per paragraph.
    const paragraphs = Array.from({ length: 20 }, (_, i) => `Paragraph ${i + 1} tells what happened next. Rhea kept walking.`)
    const { proposals } = parseOutline(paragraphs.join('\n\n'), chapters, codex, lanes)
    expect(proposals.length).toBeGreaterThan(1)
    expect(proposals.length).toBeLessThan(20)
    expect(proposals[0].chapterNum).toBe(1)
    expect(proposals[0].who).toEqual(['c-rhea'])
    // Structured text is never clustered, however long.
    const listed = parseOutline(paragraphs.map(p => `- ${p}`).join('\n'), chapters, codex, lanes)
    expect(listed.proposals).toHaveLength(20)
  })

  it('keeps nested list details with their parent card', () => {
    const { proposals } = parseOutline(
      '- Rhea enters the tower.\n  - She hears the gears stop.\n  - The stairwell goes dark.\n- Tomas waits below.\n',
      chapters, codex, lanes,
    )
    expect(proposals.map(p => [p.title, p.synopsis])).toEqual([
      ['Rhea enters the tower.', 'She hears the gears stop. The stairwell goes dark.'],
      ['Tomas waits below.', ''],
    ])
  })

  it('shortens long first sentences into titles without cutting words in half', () => {
    const { proposals } = parseOutline(`- ${'longword '.repeat(20)}ends here.`, chapters, codex, lanes)
    expect(proposals[0].title.length).toBeLessThanOrEqual(72)
    expect(proposals[0].title.endsWith('…')).toBe(true)
  })

  it('matches non-ASCII aliases and shortens Unicode titles on character boundaries', () => {
    const people: CodexPerson[] = [{ id: 'c-eloise', name: 'Éloise', aliases: ['Éloise'] }]
    const text = `- Éloise ${'🌊 '.repeat(40)}crosses the channel.`
    const { proposals } = parseOutline(text, chapters, people, lanes)
    expect(proposals[0].who).toEqual(['c-eloise'])
    expect(proposals[0].title).not.toMatch(/[\uD800-\uDBFF]$/)
    expect(Array.from(proposals[0].title).length).toBeLessThanOrEqual(72)
  })
})

describe('Planner data invariants', () => {
  it('always has a main lane first and a Dead Ideas note', () => {
    const p = ensurePlanner({ planner: { version: 1, lanes: [lanes[1]], cards: [], notes: [] } } as unknown as BookData)
    expect(p.lanes[0].id).toBe(MAIN_LANE_ID)
    expect(p.notes.some(n => n.system === 'dead')).toBe(true)
    expect(emptyPlanner().cards).toEqual([])
  })

  it('repairs missing required arrays without mutating imported data', () => {
    const imported = { version: 1 } as PlannerData
    const book = { planner: imported } as BookData
    const planner = ensurePlanner(book)
    expect(planner.lanes[0].id).toBe(MAIN_LANE_ID)
    expect(planner.cards).toEqual([])
    expect(planner.notes.some(n => n.system === 'dead')).toBe(true)
    expect(imported).toEqual({ version: 1 })
  })

  it('reads chapters, scene counts, and confirmed characters from the book', () => {
    const book = {
      body: [
        { id: 'a', title: 'One', type: 'chapter', content: '<p>Some words here.</p><hr class="scene"/><p>More words.</p>' },
        { id: 'b', title: '', type: 'chapter', content: '' },
      ],
      story_bible: { characters: [
        { id: 'x', name: 'Rhea', is_auto_detected: true, detection_status: 'accepted', aliases: ['R.'] },
        { id: 'y', name: 'Ghost', is_auto_detected: true, detection_status: 'review' },
        { id: 'z', name: 'Harbour', entity_kind: 'place' },
      ] },
    } as unknown as BookData
    expect(bookChapters(book)).toEqual([
      { id: 'a', num: 1, title: 'One', words: 5, scenes: 2, drafted: true },
      { id: 'b', num: 2, title: '', words: 0, scenes: 0, drafted: false },
    ])
    expect(codexPeople(book)).toEqual([{ id: 'x', name: 'Rhea', aliases: ['Rhea', 'R.'] }])
  })

  it('counts a centred break paragraph as a scene break', () => {
    const book = {
      body: [{ id: 'a', title: 'One', type: 'chapter', content: '<p>Some words here.</p><p>* * *</p><p>More words.</p><hr/><p>And more.</p>' }],
      story_bible: { characters: [] },
    } as unknown as BookData
    expect(bookChapters(book)[0].scenes).toBe(3)
  })

  // Every marker form the backend's scene rule accepts
  // (internal/indexing/scenes.go sceneBreakMarker) counts here too, so the
  // Planner's count before the first analysis is the count after it.
  it('counts every scene-break marker form the backend counts', () => {
    const chapterWith = (marker: string) => ({
      body: [{ id: 'a', title: 'One', type: 'chapter', content: `<p>Rhea waited.</p><p>${marker}</p><p>Tomas answered.</p>` }],
      story_bible: { characters: [] },
    } as unknown as BookData)
    for (const marker of ['* * *', '***', '****', '⁂', '# # #', '###', '####', '- - -', '---', '----', '~ ~ ~', '. . .']) {
      expect([marker, bookChapters(chapterWith(marker))[0].scenes]).toEqual([marker, 2])
    }
    // A paragraph that is not only a marker is prose, not a break.
    expect(bookChapters(chapterWith('*** and then'))[0].scenes).toBe(1)
  })
})

describe('cards on screen', () => {
  const planner: PlannerData = {
    ...emptyPlanner(), lanes,
    cards: [
      { id: 'k1', title: 'Beacon fails', synopsis: 'The light dies.', lines: [MAIN_LANE_ID, 'lane-rhea'], who: ['c-rhea'], chapter_id: 'ch-1', link: { chapter_id: 'ch-1', scene: 1 }, status: 'drafted' },
      { id: 'k2', title: 'Landfall', synopsis: 'They land.', lines: [MAIN_LANE_ID], who: [], chapter_id: 'ch-3', status: 'planned' },
      { id: 'k3', title: 'Second on the main line', synopsis: '', lines: [MAIN_LANE_ID], who: [], chapter_id: 'ch-1', status: 'drafted', link: { chapter_id: 'ch-1', scene: 2 } },
    ],
  }

  it('derives a card’s status from its scene link', () => {
    expect(statusOf(planner.cards[0])).toBe('drafted')
    expect(statusOf(planner.cards[1])).toBe('planned')
    expect(displayCards(planner).map(c => [c.id, c.st])).toEqual([['k1', 'drafted'], ['k2', 'planned'], ['k3', 'drafted']])
  })

  it('sizes lane rows by the busiest cell and draws one crossing per multi-line card', () => {
    const m = layoutMetrics(true, false)
    const cards = displayCards(planner)
    const rows = laneRows(lanes, cards, ['ch-1', 'ch-3', ''], m)
    // Two cards sit on the main line in chapter 1.
    expect(rows[0].height).toBe(m.pad * 2 + 2 * m.cardH + m.gap)
    expect(rows[1].height).toBe(m.pad * 2 + m.cardH)
    expect(rows[1].top).toBe(m.headerH + rows[0].height)
    const { connectors, ties } = columnConnectors(cards.filter(c => c.chapter_id === 'ch-1'), rows, lanes, m)
    expect(connectors).toHaveLength(1)
    expect(ties).toEqual([expect.objectContaining({ laneId: 'lane-rhea', color: '#5aafe0' })])
  })

  it('builds the synopsis from card synopses in lane order and leaves empty chapters out', () => {
    expect(generatedSynopsis(planner, 'ch-1')).toBe('The light dies.')
    expect(generatedSynopsis(planner, 'ch-2')).toBe('')
    // A generated paragraph carries a trace per sentence; a chapter the
    // writer edited by hand is their own text and carries none.
    const md = synopsisMarkdown({ ...planner, synopsis: { 'ch-3': 'Edited landing.' } }, chapters)
    expect(md).toBe([
      '## Chapter 1 — The Beacon',
      '',
      'The light dies.',
      '',
      '- The light dies. [Chapter 1 · Scene 1]',
      '',
      '## Chapter 3 — Landfall',
      '',
      'Edited landing.',
    ].join('\n'))
  })

  it('writes a deleted card into Dead Ideas with where it came from', () => {
    const block = deadIdeaBlock(planner.cards[0], 'Deleted', chapters, lanes, codex)
    expect(block).toContain('## Beacon fails')
    expect(block).toContain('Deleted from Chapter 1 · The Beacon · Main plot · Rhea Marsh')
  })
})

describe('who names survive re-indexing', () => {
  const card = { id: 'w1', title: 'Watch', synopsis: '', lines: [MAIN_LANE_ID], who: ['c-rhea', 'c-old'], who_names: ['Rhea Marsh', 'Old Keeper'], chapter_id: 'ch-1', status: 'planned' }

  it('names a person from the codex first, then from the names stored on the card, then by raw ID', () => {
    expect(whoText(card, codex)).toBe('Rhea, Old')
    expect(whoText({ ...card, who_names: undefined }, codex)).toBe('Rhea, c-old')
    expect(whoText({ ...card, who: ['c-rhea', 'c-old', 'c-new'], who_names: ['Rhea Marsh', 'Old Keeper'] }, [])).toBe('Rhea, Old, c-new')
  })

  it('writes the names for a new who list, keeping a previously stored name for an ID the codex no longer knows', () => {
    expect(whoNames(['c-tomas', 'c-old'], codex, card)).toEqual(['Tomas', 'Old Keeper'])
    expect(whoNames(['c-gone'], codex)).toEqual(['c-gone'])
    expect(whoNames([], codex, card)).toEqual([])
  })

  // Re-indexing reassigned the positional IDs: c-old is Eloise now and the
  // Old Keeper moved to c-keeper; c-rhea is still Rhea.
  const reindexed: CodexPerson[] = [
    { id: 'c-rhea', name: 'Rhea Marsh', aliases: ['Rhea Marsh', 'Rhea'] },
    { id: 'c-old', name: 'Eloise Dane', aliases: ['Eloise Dane', 'Eloise'] },
    { id: 'c-keeper', name: 'Old Keeper', aliases: ['Old Keeper', 'the keeper'] },
  ]

  it('keeps naming the person the stored name means when the ID now belongs to someone else', () => {
    expect(whoText(card, reindexed)).toBe('Rhea, Old')
    expect(whoNames(['c-rhea', 'c-old'], reindexed, card)).toEqual(['Rhea Marsh', 'Old Keeper'])
    // With no stored name there is nothing to check the ID against.
    expect(whoText({ ...card, who_names: undefined }, reindexed)).toBe('Rhea, Eloise')
    // A stored name the codex no longer has anywhere still wins over the stranger under its ID.
    expect(whoText({ ...card, who_names: ['Rhea Marsh', 'Night Warden'] }, reindexed)).toBe('Rhea, Night')
    // A stored alias still counts as the person under the ID.
    expect(whoText({ ...card, who_names: ['Rhea', 'Old Keeper'] }, reindexed)).toBe('Rhea, Old')
  })

  it('re-binds who to the current IDs of the people the stored names mean, each once', () => {
    expect(rebindWho(['c-rhea', 'c-old'], reindexed, card)).toEqual({ who: ['c-rhea', 'c-keeper'], who_names: ['Rhea Marsh', 'Old Keeper'] })
    expect(rebindWho(['c-old', 'c-keeper'], reindexed, card)).toEqual({ who: ['c-keeper'], who_names: ['Old Keeper'] })
    expect(rebindWho(['c-gone'], reindexed)).toEqual({ who: ['c-gone'], who_names: ['c-gone'] })
    expect(whoIncludes(card, reindexed[2], reindexed)).toBe(true)
    expect(whoIncludes(card, reindexed[1], reindexed)).toBe(false)
  })
})
