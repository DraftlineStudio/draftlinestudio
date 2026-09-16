// The synopsis a chapter's own cards make: the order the sentences come in,
// the trace back to the card behind each one, and the writer's own text when
// they have edited a chapter by hand.
//
// All prose here is invented for these tests.

import { describe, expect, it } from 'vitest'
import type { PlannerCard, PlannerData, PlannerLane } from '../../../types/draftline'
import { emptyPlanner, MAIN_LANE_ID, type PlannerChapter } from '../plannerModel'
import { generatedSynopsis, synopsisEntries, synopsisMarkdown, synopsisText, synopsisTrace } from '../plannerSynopsis'

const lanes: PlannerLane[] = [
  { id: MAIN_LANE_ID, name: 'Main plot', kind: 'main', color: '#5aafe0' },
  { id: 'lane-rhea', name: 'Rhea Marsh', kind: 'character', color: '#F472B6', character_id: 'c-rhea' },
]

const chapters: PlannerChapter[] = [
  { id: 'ch-1', num: 1, title: 'The Beacon', words: 900, scenes: 2, drafted: true },
  { id: 'ch-2', num: 2, title: 'Landfall', words: 600, scenes: 1, drafted: true },
]

const card = (over: Partial<PlannerCard> = {}): PlannerCard => ({
  id: 'card-1', title: 'Rhea finds the lamp', synopsis: '', lines: [MAIN_LANE_ID], who: [],
  chapter_id: 'ch-1', status: 'planned', ...over,
})

const plannerWith = (cards: PlannerCard[]): PlannerData => ({ ...emptyPlanner(), lanes, cards })

describe('the synopsis is made of the chapter’s own cards', () => {
  const planner = plannerWith([
    card({ id: 'card-1', synopsis: 'Rhea finds the lamp out.', lines: ['lane-rhea'] }),
    card({
      id: 'card-2', title: 'Tomas climbs', synopsis: 'Tomas climbs after her.',
      lines: [MAIN_LANE_ID], link: { chapter_id: 'ch-1', scene: 2 },
    }),
  ])

  it('orders entries by story line, then by card', () => {
    expect(synopsisEntries(planner, 'ch-1').map(e => [e.text, e.cardId])).toEqual([
      ['Tomas climbs after her.', 'card-2'],
      ['Rhea finds the lamp out.', 'card-1'],
    ])
  })

  it('leaves a chapter with no card synopses blank rather than inventing one', () => {
    expect(generatedSynopsis(planner, 'ch-2')).toBe('')
  })

  it('round-trips the entries into the Markdown export with a trace each', () => {
    const entries = synopsisEntries(planner, 'ch-1')
    const md = synopsisMarkdown(planner, chapters)
    expect(md.split('\n\n')[1]).toBe(entries.map(e => e.text).join(' '))
    for (const entry of entries) {
      expect(md).toContain(`- ${entry.text} ${synopsisTrace(entry, chapters)}`)
    }
    expect(synopsisTrace(entries[0], chapters)).toBe('[Chapter 1 · Scene 2]')
    // A card with no position at all still traces somewhere truthful.
    expect(synopsisTrace({ ...entries[0], chapterId: 'nowhere', scene: undefined }, chapters)).toBe('[Later]')
  })

  it('keeps an edited chapter as the writer’s own text, with no traces', () => {
    const edited = { ...planner, synopsis: { 'ch-1': 'Her own words about the tower.' } }
    expect(synopsisText(edited, 'ch-1')).toBe('Her own words about the tower.')
    const md = synopsisMarkdown(edited, chapters)
    expect(md).toContain('Her own words about the tower.')
    expect(md).not.toContain('[Chapter 1 ·')
  })
})
