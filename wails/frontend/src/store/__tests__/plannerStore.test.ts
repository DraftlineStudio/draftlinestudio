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
  const useBookStore = create<{ book: BookData | null; analysisRevision: number; updateBook: (b: BookData) => void }>(set => ({
    book: null,
    analysisRevision: 1,
    updateBook: (book) => set({ book }),
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

describe('outline import stays inside the Planner', () => {
  beforeEach(() => {
    mocks.setStatusMessage.mockClear()
    usePlannerStore.getState().reset()
    useBookStore.setState({ book: bookWith([person('c-rhea', 'Rhea Marsh', ['Rhea'])]), analysisRevision: 1 })
  })

  it('creates unpinned cards and a detected story line without creating manuscript chapters', () => {
    const store = usePlannerStore.getState()
    store.openImport('# The signal\n## The lamp fails\n- Rhea finds the mechanism still.\n## The warning\n- The harbour closes before dark.')
    store.runImport()
    expect(usePlannerStore.getState().proposals!.proposals.every(item => item.laneId.startsWith('new:'))).toBe(true)
    usePlannerStore.getState().acceptProposals()

    const book = useBookStore.getState().book!
    expect(book.body).toHaveLength(1)
    expect(book.planner!.cards).toHaveLength(2)
    expect(book.planner!.cards.every(item => item.chapter_id === '' && item.status === 'planned')).toBe(true)
    const line = book.planner!.lanes.find(item => item.name === 'The signal')
    expect(line?.kind).toBe('subplot')
    expect(book.planner!.cards.every(item => item.lines[0] === line?.id)).toBe(true)
    expect(mocks.setStatusMessage).toHaveBeenCalledWith('2 cards added to Later')
  })

  it('creates and immediately assigns a custom plot track inside the proposal', () => {
    const store = usePlannerStore.getState()
    store.openImport('# The signal\n## The lamp fails\n- The mechanism stops.')
    store.runImport()
    const group = usePlannerStore.getState().proposals!.proposals[0].groupId
    usePlannerStore.getState().addProposalLane(group, 'Mechanical mystery')
    usePlannerStore.getState().acceptProposals()

    const planner = useBookStore.getState().book!.planner!
    const line = planner.lanes.find(item => item.name === 'Mechanical mystery')
    expect(line?.kind).toBe('subplot')
    expect(planner.cards[0].lines[0]).toBe(line?.id)
  })

  it('adds reviewed outline characters and their populated character tracks', () => {
    const store = usePlannerStore.getState()
    store.openImport('# Harbour story\n## Characters\n### Nessa Vale\nA cartographer tracking false signals.\n## The warning\n- Nessa finds a coded flag above the pier.')
    store.runImport()
    const proposal = usePlannerStore.getState().proposals!
    expect(proposal.proposals[0].who).toEqual([proposal.characters[0].id])
    usePlannerStore.getState().acceptProposals()

    const book = useBookStore.getState().book!
    const character = book.story_bible!.characters.find(item => item.name === 'Nessa Vale')
    const line = book.planner!.lanes.find(item => item.character_id === character?.id)
    expect(character?.description).toContain('cartographer')
    expect(line?.kind).toBe('character')
    expect(book.planner!.cards[0].who).toContain(character?.id)
    expect(book.planner!.cards[0].lines).toContain(line?.id)
    expect(mocks.setStatusMessage).toHaveBeenCalledWith('1 card added to Later; 1 character added')
  })

  it('proposes only additions when the same Scratchpad outline is expanded', () => {
    const first = '# Harbour\n## The bell\n- It rings before dawn.\n## The boat\n- It leaves in fog.'
    const expanded = '# Harbour\n## The bell\n- It now rings twice before dawn.\n## The boat\n- It leaves later in heavy fog.\n## The return\n- The boat comes back empty.'
    const store = usePlannerStore.getState()
    store.openImport(first, 'note-outline')
    store.runImport()
    usePlannerStore.getState().acceptProposals()
    expect(cards()).toHaveLength(2)

    usePlannerStore.getState().openImport(expanded, 'note-outline')
    usePlannerStore.getState().runImport()
    const proposals = usePlannerStore.getState().proposals!.proposals
    expect(proposals.map(proposal => proposal.title)).toEqual(['The return'])
    usePlannerStore.getState().acceptProposals()
    expect(cards()).toHaveLength(3)
  })
})

describe('lane moves', () => {
  beforeEach(() => {
    usePlannerStore.getState().reset()
    const book = bookWith([], [{ ...card([]), lines: [MAIN_LANE_ID, 'lane-crossing'] }])
    book.planner!.lanes.push(
      { id: 'lane-crossing', name: 'Crossing', kind: 'subplot', color: '#f00' },
      { id: 'lane-new', name: 'New primary', kind: 'subplot', color: '#0f0' },
    )
    useBookStore.setState({ book, analysisRevision: 1 })
  })

  it('replaces the former primary lane while preserving deliberate crossings', () => {
    usePlannerStore.getState().moveCard('card-1', 'ch-1', 'lane-new', null)
    expect(cards()[0].lines).toEqual(['lane-new', 'lane-crossing'])
  })
})
