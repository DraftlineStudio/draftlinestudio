// Planner store: the Planner's transient UI state (which view is open, the
// selected card, drags, dialogs) plus every mutation of the persisted Planner
// data. Persisted data lives on the book (book.planner → planner.json) and is
// written through bookStore's existing updateBook action so it dirties the
// book and rides the normal save pipeline; this store holds no second copy.

import { create } from 'zustand'
import type {
  BeatTemplateId, Character, PlannerCard, PlannerData, PlannerLane,
  PlannerLink, PlannerNote,
} from '../types/draftline'
import { useBookStore } from './bookStore'
import { useAppStore } from './appStore'
import {
  bookChapters, codexPeople, deadIdeaBlock, ensurePlanner, LANE_PALETTE, MAIN_LANE_ID, newId, nowStamp,
  onlyNewOutlineProposals, parseOutline, rebindWho, whoNames, type ParsedOutline, type Proposal, type ProposalCharacter,
} from '../components/planner/plannerModel'

export type PlannerView = 'timeline' | 'board' | 'scratch' | 'synopsis'

interface PlannerStore {
  view: PlannerView
  panelOpen: boolean
  selected: string | null
  drag: string | null
  boardBy: 'chapter' | 'line'
  noteId: string | null
  noteMono: boolean
  linkOpen: boolean
  lineDialog: { kind: string; name: string } | null
  importOpen: boolean
  importText: string
  importFromNoteId: string | null
  proposals: ParsedOutline | null

  // View state
  pick: (view: PlannerView) => void
  setPanelOpen: (open: boolean) => void
  select: (id: string | null) => void
  setDrag: (id: string | null) => void
  setBoardBy: (by: 'chapter' | 'line') => void
  setNoteId: (id: string | null) => void
  toggleNoteMono: () => void
  setLinkOpen: (open: boolean) => void
  openLineDialog: () => void
  setLineDialog: (d: { kind: string; name: string } | null) => void
  reset: () => void

  // Persisted data
  planner: () => PlannerData
  mutate: (fn: (p: PlannerData) => PlannerData) => void
  addCard: (chapterId: string, laneId: string) => void
  updateCard: (id: string, patch: Partial<PlannerCard>) => void
  moveCard: (id: string, chapterId: string, laneId: string | null, beforeId: string | null) => void
  deleteCard: (id: string) => void
  linkCard: (id: string, link: PlannerLink | null) => void
  addSubplot: (name: string) => void
  addCharacterLane: (characterId: string) => void
  toggleLaneHidden: (laneId: string) => void
  setBeatTemplate: (t: BeatTemplateId) => void
  setCompact: (compact: boolean) => void

  // Notes
  newNote: (title?: string, body?: string) => string
  updateNote: (id: string, patch: Partial<PlannerNote>) => void
  deleteNote: (id: string) => void
  // Deleting a note asks first; noteDeleteId is the note awaiting confirmation.
  noteDeleteId: string | null
  askDeleteNote: (id: string) => void
  cancelDeleteNote: () => void

  // Synopsis
  setSynopsis: (chapterId: string, text: string) => void
  resetSynopsis: () => void

  // Import Outline
  openImport: (text?: string, fromNoteId?: string | null) => void
  setImportText: (text: string) => void
  closeImport: () => void
  runImport: () => void
  updateProposal: (id: string, patch: Partial<Proposal>) => void
  updateProposalCharacter: (id: string, patch: Partial<ProposalCharacter>) => void
  setProposalGroupLane: (groupId: string, laneId: string) => void
  addProposalLane: (groupId: string, name: string) => void
  dropProposal: (id: string) => void
  backToPaste: () => void
  acceptProposals: () => void
}

const initialUI = {
  view: 'timeline' as PlannerView, panelOpen: true, selected: null, drag: null, boardBy: 'chapter' as const,
  noteId: null, noteMono: false, linkOpen: false, lineDialog: null, noteDeleteId: null as string | null,
  importOpen: false, importText: '', importFromNoteId: null, proposals: null,
}

export const usePlannerStore = create<PlannerStore>((set, get) => ({
  ...initialUI,

  pick: (view) => set(s => (s.view === view ? { panelOpen: !s.panelOpen } : { view, panelOpen: true, linkOpen: false })),
  setPanelOpen: (panelOpen) => set({ panelOpen }),
  select: (selected) => set({ selected, linkOpen: false }),
  setDrag: (drag) => set({ drag }),
  setBoardBy: (boardBy) => set({ boardBy }),
  setNoteId: (noteId) => set({ noteId }),
  toggleNoteMono: () => set(s => ({ noteMono: !s.noteMono })),
  setLinkOpen: (linkOpen) => set({ linkOpen }),
  openLineDialog: () => set({ lineDialog: { kind: 'Subplot', name: '' } }),
  setLineDialog: (lineDialog) => set({ lineDialog }),
  reset: () => set({ ...initialUI }),

  planner: () => ensurePlanner(useBookStore.getState().book),

  mutate: (fn) => {
    const bookStore = useBookStore.getState()
    const book = bookStore.book
    if (!book) return
    bookStore.updateBook({ ...book, planner: fn(ensurePlanner(book)) })
  },

  addCard: (chapterId, laneId) => {
    const id = newId('card')
    get().mutate(p => ({
      ...p,
      cards: [...p.cards, { id, title: 'Untitled card', synopsis: '', lines: [laneId || MAIN_LANE_ID], who: [], changes: '', stakes: '', chapter_id: chapterId, status: 'planned', origin: 'manual', updated: nowStamp() }],
    }))
    set({ selected: id, linkOpen: false })
  },

  // Whenever `who` changes, each ID is re-bound to the person its stored
  // name means and the people's names are written beside it, so the card
  // survives re-indexing reassigning codex IDs.
  updateCard: (id, patch) => {
    const codex = patch.who ? codexPeople(useBookStore.getState().book) : []
    get().mutate(p => ({
      ...p, cards: p.cards.map(c => (c.id === id
        ? { ...c, ...patch, ...(patch.who ? rebindWho(patch.who, codex, c) : {}), updated: nowStamp() }
        : c)),
    }))
  },

  moveCard: (id, chapterId, laneId, beforeId) => {
    get().mutate(p => {
      const src = p.cards.find(c => c.id === id)
      if (!src) return p
      const moved: PlannerCard = { ...src, chapter_id: chapterId, updated: nowStamp() }
      // A drag between lanes moves the card's primary line. Secondary lines
      // are deliberate crossings chosen in the inspector, so preserve those
      // without retaining the old primary lane as an accidental crossing.
      if (laneId) moved.lines = [laneId, ...moved.lines.slice(1).filter(l => l !== laneId)]
      const rest = p.cards.filter(c => c.id !== id)
      const at = beforeId ? rest.findIndex(c => c.id === beforeId) : -1
      if (at >= 0) rest.splice(at, 0, moved)
      else rest.push(moved)
      return { ...p, cards: rest }
    })
    set({ drag: null })
  },

  deleteCard: (id) => {
    const book = useBookStore.getState().book
    get().mutate(p => {
      const card = p.cards.find(c => c.id === id)
      if (!card) return p
      const block = deadIdeaBlock(card, 'Deleted', bookChapters(book), p.lanes, codexPeople(book))
      return {
        ...p,
        cards: p.cards.filter(c => c.id !== id),
        notes: p.notes.map(n => (n.system === 'dead' ? { ...n, body: n.body + block, updated: nowStamp() } : n)),
      }
    })
    set(s => (s.selected === id ? { selected: null } : {}))
  },

  linkCard: (id, link) => get().updateCard(id, { link: link ?? undefined, status: link ? 'drafted' : 'planned' }),

  addSubplot: (name) => {
    const trimmed = name.trim()
    if (!trimmed) return
    get().mutate(p => ({ ...p, lanes: [...p.lanes, { id: newId('lane'), name: trimmed, kind: 'subplot', color: LANE_PALETTE[p.lanes.length % LANE_PALETTE.length] }] }))
  },

  addCharacterLane: (characterId) => {
    const person = codexPeople(useBookStore.getState().book).find(c => c.id === characterId)
    if (!person) return
    get().mutate(p => (p.lanes.some(l => l.character_id === characterId)
      ? p
      : { ...p, lanes: [...p.lanes, { id: newId('lane'), name: person.name, kind: 'character', color: LANE_PALETTE[p.lanes.length % LANE_PALETTE.length], character_id: characterId } as PlannerLane] }))
  },

  toggleLaneHidden: (laneId) => get().mutate(p => {
    const hidden = new Set(p.hidden_lanes ?? [])
    if (hidden.has(laneId)) hidden.delete(laneId)
    else hidden.add(laneId)
    return { ...p, hidden_lanes: [...hidden] }
  }),

  setBeatTemplate: (beat_template) => get().mutate(p => ({ ...p, beat_template })),
  setCompact: (compact) => get().mutate(p => ({ ...p, compact })),

  newNote: (title = '', body = '') => {
    const id = newId('note')
    get().mutate(p => ({ ...p, notes: [{ id, title, body, updated: nowStamp() }, ...p.notes] }))
    set({ noteId: id, view: 'scratch' })
    return id
  },

  updateNote: (id, patch) => get().mutate(p => ({
    ...p, notes: p.notes.map(n => (n.id === id ? { ...n, ...patch, updated: nowStamp() } : n)),
  })),

  askDeleteNote: (id) => {
    if (get().planner().notes.find(n => n.id === id)?.system) return
    set({ noteDeleteId: id })
  },
  cancelDeleteNote: () => set({ noteDeleteId: null }),

  deleteNote: (id) => {
    set({ noteDeleteId: null })
    const planner = get().planner()
    if (planner.notes.find(n => n.id === id)?.system) return
    get().mutate(p => ({ ...p, notes: p.notes.filter(n => n.id !== id) }))
    set(s => (s.noteId === id ? { noteId: null } : {}))
  },

  setSynopsis: (chapterId, text) => get().mutate(p => ({ ...p, synopsis: { ...(p.synopsis ?? {}), [chapterId]: text } })),
  resetSynopsis: () => get().mutate(p => ({ ...p, synopsis: {} })),

  openImport: (text, fromNoteId = null) => set(s => ({
    importOpen: true, proposals: null, importText: text ?? s.importText, importFromNoteId: fromNoteId,
    view: 'scratch', panelOpen: true,
  })),
  setImportText: (importText) => set({ importText }),
  closeImport: () => set({ importOpen: false, proposals: null, importFromNoteId: null }),

  runImport: () => {
    const book = useBookStore.getState().book
    const planner = get().planner()
    const parsed = parseOutline(get().importText, codexPeople(book), planner.lanes)
    const sourceId = get().importFromNoteId
    set({ proposals: sourceId ? onlyNewOutlineProposals(parsed, planner.cards, sourceId) : parsed })
  },

  updateProposal: (id, patch) => set(s => (s.proposals
    ? { proposals: { ...s.proposals, proposals: s.proposals.proposals.map(q => (q.id === id ? { ...q, ...patch } : q)) } }
    : {})),

  updateProposalCharacter: (id, patch) => set(s => (s.proposals
    ? { proposals: { ...s.proposals, characters: s.proposals.characters.map(character => character.id === id ? { ...character, ...patch } : character) } }
    : {})),

  setProposalGroupLane: (groupId, laneId) => set(s => (s.proposals
    ? { proposals: { ...s.proposals, proposals: s.proposals.proposals.map(q => (q.groupId === groupId ? { ...q, laneId } : q)) } }
    : {})),

  addProposalLane: (groupId, name) => {
    const trimmed = name.trim()
    if (!trimmed) return
    const key = trimmed.toLocaleLowerCase()
    const existingLane = get().planner().lanes.find(l => l.name.trim().toLocaleLowerCase() === key)
    set(s => {
      if (!s.proposals) return {}
      const proposed = s.proposals.lanes.find(l => l.name.trim().toLocaleLowerCase() === key)
      const laneId = existingLane?.id ?? proposed?.id ?? `new:${newId('proposal-lane')}`
      return { proposals: {
        ...s.proposals,
        lanes: existingLane || proposed ? s.proposals.lanes : [...s.proposals.lanes, { id: laneId, name: trimmed }],
        proposals: s.proposals.proposals.map(q => (q.groupId === groupId ? { ...q, laneId } : q)),
      } }
    })
  },

  dropProposal: (id) => set(s => (s.proposals
    ? { proposals: { ...s.proposals, proposals: s.proposals.proposals.filter(q => q.id !== id) } }
    : {})),

  backToPaste: () => set({ proposals: null }),

  // Accepting adds unpinned planned cards and any story lines explicitly
  // chosen in the proposal view. The manuscript's chapter list is never
  // changed; cards can be attached to chapters later.
  acceptProposals: () => {
    const s = get()
    const bookStore = useBookStore.getState()
    const book = bookStore.book
    if (!s.proposals || !book) return
    const accepted = s.proposals.proposals.filter(q => q.accepted)
    const acceptedCharacters = s.proposals.characters.filter(character => character.accepted && character.name.trim())
    const fromNote = s.importFromNoteId
    const noteId = fromNote ?? newId('note')
    const existingCharacters = book.story_bible?.characters ?? []
    const characterIDs = new Map<string, string>()
    const addedCharacters: Character[] = []
    for (const proposed of acceptedCharacters) {
      const proposedNames = new Set([proposed.name, ...proposed.aliases].map(name => name.trim().toLocaleLowerCase()))
      const existing = existingCharacters.find(character => [character.name, ...(character.aliases ?? [])].some(name => proposedNames.has(name.trim().toLocaleLowerCase())))
      const id = existing?.id ?? newId('character')
      characterIDs.set(proposed.id, id)
      if (!existing) addedCharacters.push({
        id, name: proposed.name.trim(), aliases: proposed.aliases, role: 'supporting', description: proposed.description,
        appearance: '', personality: '', motivation: '', notes: '',
      })
    }
    const storyBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    const bookWithCharacters = { ...book, story_bible: { ...storyBible, characters: [...existingCharacters, ...addedCharacters] } }
    const codex = codexPeople(bookWithCharacters)
    const p = ensurePlanner(book)
    const laneIDs = new Map<string, string>()
    const addedLanes: PlannerLane[] = []
    for (const proposal of accepted) {
      if (!proposal.laneId.startsWith('new:') || laneIDs.has(proposal.laneId)) continue
      const proposedName = s.proposals.lanes.find(l => l.id === proposal.laneId)?.name ?? proposal.groupName
      const existing = p.lanes.find(l => l.name.trim().toLocaleLowerCase() === proposedName.trim().toLocaleLowerCase())
      const id = existing?.id ?? newId('lane')
      laneIDs.set(proposal.laneId, id)
      if (!existing) addedLanes.push({ id, name: proposedName, kind: 'subplot', color: LANE_PALETTE[(p.lanes.length + addedLanes.length) % LANE_PALETTE.length] })
    }
    const characterLaneIDs = new Map<string, string>()
    for (const proposed of acceptedCharacters.filter(character => character.createLane)) {
      const characterId = characterIDs.get(proposed.id)
      if (!characterId) continue
      const existing = [...p.lanes, ...addedLanes].find(l => l.character_id === characterId)
      const id = existing?.id ?? newId('lane')
      characterLaneIDs.set(characterId, id)
      if (!existing) addedLanes.push({ id, name: proposed.name.trim(), kind: 'character', color: LANE_PALETTE[(p.lanes.length + addedLanes.length) % LANE_PALETTE.length], character_id: characterId })
    }
    const cards: PlannerCard[] = accepted.map(q => {
      const who = [...new Set(q.who.map(id => characterIDs.get(id) ?? id).filter(id => codex.some(person => person.id === id)))]
      const characterLines = who.map(id => characterLaneIDs.get(id)).filter((id): id is string => !!id)
      let primary = laneIDs.get(q.laneId) ?? q.laneId
      if (primary === MAIN_LANE_ID && q.groupName === 'Outline' && characterLines.length) primary = characterLines[0]
      return {
        id: newId('card'), source_id: noteId, source_key: q.sourceKey, title: q.title, synopsis: q.synopsis,
        lines: [primary, ...characterLines.filter(id => id !== primary)], who, who_names: whoNames(who, codex), changes: '', stakes: '',
        chapter_id: '', status: 'planned', origin: 'outline', updated: nowStamp(),
      }
    })
    bookStore.updateBook({
      ...bookWithCharacters,
      planner: {
        ...p,
        lanes: [...p.lanes, ...addedLanes],
        cards: [...p.cards, ...cards],
        notes: fromNote ? p.notes : [{ id: noteId, title: 'Imported outline', body: s.importText, updated: nowStamp() }, ...p.notes],
      },
    })
    set({ importOpen: false, proposals: null, importFromNoteId: null, noteId, view: 'timeline', panelOpen: true })
    const messages = []
    if (accepted.length) messages.push(`${accepted.length} card${accepted.length === 1 ? '' : 's'} added to Later`)
    if (acceptedCharacters.length) messages.push(`${acceptedCharacters.length} character${acceptedCharacters.length === 1 ? '' : 's'} added`)
    useAppStore.getState().setStatusMessage(messages.join('; '))
  },
}))
