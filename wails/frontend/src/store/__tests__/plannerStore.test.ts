// The Planner store's seam that writes to planner.json: the people named on
// a card, and the re-binding that keeps those names right after the codex
// reassigns IDs. Only the neighbouring stores are mocked; the model is real.
//
// All prose here is invented for these tests.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from 'zustand'
import type { BookData, Character, PlannerCard } from '../../types/draftline'
import { MAIN_LANE_ID } from '../../components/planner/plannerModel'

const mocks = vi.hoisted(() => ({ setStatusMessage: vi.fn() }))

vi.mock('../appStore', () => ({
  useAppStore: { getState: () => ({ setStatusMessage: mocks.setStatusMessage, settings: { analysis_enabled: true } }) },
}))
vi.mock('../bookStore', () => {
  const useBookStore = create<{ book: BookData | null; analysisRevision: number; updateBook: (b: BookData) => void; addChapter: () => void }>(set => ({
    book: null,
    analysisRevision: 1,
    updateBook: (book) => set({ book }),
    addChapter: () => {},
  }))
  return { useBookStore }
})

import { useBookStore } from '../bookStore'
import { usePlannerStore } from '../plannerStore'

const person = (id: string, name: string, aliases: string[] = []) =>
  ({ id, name, aliases, is_auto_detected: true, detection_status: 'accepted' }) as unknown as Character

function bookWith(characters: Character[], cards: PlannerCard[] = []): BookData {
  return {
    metadata: { title: 'Harbour', author: '', created: '2026-09-15T00:00:00Z' },
    body: [{ id: 'ch-1', title: 'One', type: 'chapter', content: '<p>Rhea climbed the tower.</p>' }],
    front_matter: [], back_matter: [],
    story_bible: { characters, plot_notes: '', timeline: '' },
    planner: { version: 1, lanes: [{ id: MAIN_LANE_ID, name: 'Main plot', kind: 'main', color: '#5aafe0' }], cards, notes: [] },
  } as unknown as BookData
}

const card = (who: string[], who_names?: string[]): PlannerCard =>
  ({ id: 'card-1', title: 'Watch', synopsis: '', lines: [MAIN_LANE_ID], who, who_names, chapter_id: 'ch-1', status: 'planned' })

const cards = () => useBookStore.getState().book!.planner!.cards

describe('planner store writes who_names beside who', () => {
  beforeEach(() => {
    usePlannerStore.getState().reset()
    useBookStore.setState({ book: bookWith([person('c-rhea', 'Rhea Marsh', ['Rhea']), person('c-tomas', 'Tomas')], [card([])]), analysisRevision: 1 })
  })

  it('updateCard({ who }) writes aligned names', () => {
    usePlannerStore.getState().updateCard('card-1', { who: ['c-tomas', 'c-rhea'] })
    expect(cards()[0].who).toEqual(['c-tomas', 'c-rhea'])
    expect(cards()[0].who_names).toEqual(['Tomas', 'Rhea Marsh'])
    usePlannerStore.getState().updateCard('card-1', { title: 'Watch the beacon' })
    expect(cards()[0].who_names).toEqual(['Tomas', 'Rhea Marsh'])
  })

  it('re-binds an ID that re-indexing handed to someone else, by the stored name', () => {
    usePlannerStore.getState().updateCard('card-1', { who: ['c-tomas'] })
    // Re-index: c-tomas is Eloise now, Tomas moved to c-t2.
    const book = useBookStore.getState().book!
    useBookStore.setState({ book: { ...book, story_bible: { characters: [person('c-rhea', 'Rhea Marsh', ['Rhea']), person('c-tomas', 'Eloise'), person('c-t2', 'Tomas')], plot_notes: '', timeline: '' } } })
    usePlannerStore.getState().updateCard('card-1', { who: [...cards()[0].who, 'c-rhea'] })
    expect(cards()[0].who).toEqual(['c-t2', 'c-rhea'])
    expect(cards()[0].who_names).toEqual(['Tomas', 'Rhea Marsh'])
  })
})
