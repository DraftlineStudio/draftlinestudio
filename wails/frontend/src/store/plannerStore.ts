// Planner store: the Planner's transient UI state (which view is open, the
// selected card, drags, dialogs) plus every mutation of the persisted Planner
// data. Persisted data lives on the book (book.planner → planner.json) and is
// written through bookStore's existing updateBook action so it dirties the
// book and rides the normal save pipeline; this store holds no second copy.

import { create } from 'zustand'
import { PlannerDetectCards } from '../../wailsjs/go/main/App'
import type { BeatTemplateId, PlannerCard, PlannerData, PlannerDetectedCard, PlannerLane, PlannerLink, PlannerNote } from '../types/draftline'
import { useBookStore } from './bookStore'
import { useAppStore } from './appStore'
import {
  adoptedId, bookChapters, codexPeople, deadIdeaBlock, ensurePlanner, LANE_PALETTE, MAIN_LANE_ID, newId, nowStamp,
  parseOutline, type ParsedOutline, type Proposal,
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
  chapterDialog: { title: string } | null
  lineDialog: { kind: string; name: string } | null
  importOpen: boolean
  importText: string
  importFromNoteId: string | null
  proposals: ParsedOutline | null
  // Unplanned proposals from the narrative engine (Plot Walker on).
  detected: PlannerDetectedCard[]
  detecting: boolean
  detectError: string
  detectedRevision: number

  // View state
  pick: (view: PlannerView) => void
  setPanelOpen: (open: boolean) => void
  select: (id: string | null) => void
  setDrag: (id: string | null) => void
  setBoardBy: (by: 'chapter' | 'line') => void
  setNoteId: (id: string | null) => void
  toggleNoteMono: () => void
  setLinkOpen: (open: boolean) => void
  openChapterDialog: () => void
  setChapterDialog: (d: { title: string } | null) => void
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
  adoptDetected: (id: string) => void
  dismissDetected: (id: string) => void
  addSubplot: (name: string) => void
  addCharacterLane: (characterId: string) => void
  toggleLaneHidden: (laneId: string) => void
  setBeatTemplate: (t: BeatTemplateId) => void
  setPlotWalker: (on: boolean) => void
  setCompact: (compact: boolean) => void
  createChapter: (title: string) => void

  // Notes
  newNote: (title?: string, body?: string) => string
  updateNote: (id: string, patch: Partial<PlannerNote>) => void
  deleteNote: (id: string) => void

  // Synopsis
  setSynopsis: (chapterId: string, text: string) => void
  resetSynopsis: () => void

  // Import Outline
  openImport: (text?: string, fromNoteId?: string | null) => void
  setImportText: (text: string) => void
  closeImport: () => void
  runImport: () => void
  updateProposal: (id: string, patch: Partial<Proposal>) => void
  dropProposal: (id: string) => void
  backToPaste: () => void
  acceptProposals: () => void

  // Detection
  refreshDetected: () => Promise<void>
}

const initialUI = {
  view: 'timeline' as PlannerView, panelOpen: true, selected: null, drag: null, boardBy: 'chapter' as const,
  noteId: null, noteMono: false, linkOpen: false, chapterDialog: null, lineDialog: null,
  importOpen: false, importText: '', importFromNoteId: null, proposals: null,
  detected: [] as PlannerDetectedCard[], detecting: false, detectError: '', detectedRevision: -1,
}

function newChapterId(): string {
  return typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? `ch-${crypto.randomUUID().replace(/-/g, '')}`
    : `ch-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
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
  openChapterDialog: () => set({ chapterDialog: { title: '' } }),
  setChapterDialog: (chapterDialog) => set({ chapterDialog }),
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

  updateCard: (id, patch) => get().mutate(p => ({
    ...p, cards: p.cards.map(c => (c.id === id ? { ...c, ...patch, updated: nowStamp() } : c)),
  })),

  moveCard: (id, chapterId, laneId, beforeId) => {
    get().mutate(p => {
      const src = p.cards.find(c => c.id === id)
      if (!src) return p
      const moved: PlannerCard = { ...src, chapter_id: chapterId, updated: nowStamp() }
      if (laneId) moved.lines = [laneId, ...moved.lines.filter(l => l !== laneId)]
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

  adoptDetected: (detectedId) => {
    const d = get().detected.find(x => x.id === detectedId)
    if (!d) return
    const id = adoptedId(d.id)
    get().mutate(p => ({
      ...p, cards: p.cards.some(c => c.id === id) ? p.cards : [...p.cards, {
        id, source_id: d.evidence[0]?.source_id, title: d.title, synopsis: d.synopsis, lines: [MAIN_LANE_ID], who: d.who, changes: '', stakes: '',
        chapter_id: d.chapter_id, link: { chapter_id: d.chapter_id, scene: d.scene }, status: 'drafted', origin: 'adopted',
        dev_kind: d.kind, evidence: d.evidence, updated: nowStamp(),
      }],
    }))
    set({ selected: id })
  },

  dismissDetected: (detectedId) => {
    const d = get().detected.find(x => x.id === detectedId)
    const book = useBookStore.getState().book
    get().mutate(p => {
      const notes = d
        ? p.notes.map(n => (n.system === 'dead'
          ? { ...n, body: n.body + deadIdeaBlock({ id: d.id, title: d.title, synopsis: d.synopsis, lines: [MAIN_LANE_ID], who: d.who, chapter_id: d.chapter_id, status: 'drafted' }, 'Dismissed', bookChapters(book), p.lanes, codexPeople(book)), updated: nowStamp() }
          : n))
        : p.notes
      return { ...p, notes, dismissed: [...new Set([...(p.dismissed ?? []), detectedId])] }
    })
    set(s => (s.selected === detectedId ? { selected: null } : {}))
  },

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

  setPlotWalker: (on) => {
    get().mutate(p => ({ ...p, plot_walker: on }))
    if (on) void get().refreshDetected()
  },

  setCompact: (compact) => get().mutate(p => ({ ...p, compact })),

  createChapter: (title) => {
    useBookStore.getState().addChapter('body', { id: newChapterId(), title: title.trim() || `Chapter ${(useBookStore.getState().book?.body.length ?? 0) + 1}`, type: 'chapter', content: '' })
    set({ chapterDialog: null })
  },

  newNote: (title = '', body = '') => {
    const id = newId('note')
    get().mutate(p => ({ ...p, notes: [{ id, title, body, updated: nowStamp() }, ...p.notes] }))
    set({ noteId: id, view: 'scratch' })
    return id
  },

  updateNote: (id, patch) => get().mutate(p => ({
    ...p, notes: p.notes.map(n => (n.id === id ? { ...n, ...patch, updated: nowStamp() } : n)),
  })),

  deleteNote: (id) => {
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
    set({ proposals: parseOutline(get().importText, bookChapters(book), codexPeople(book), planner.lanes) })
  },

  updateProposal: (id, patch) => set(s => (s.proposals
    ? { proposals: { ...s.proposals, proposals: s.proposals.proposals.map(q => (q.id === id ? { ...q, ...patch } : q)) } }
    : {})),

  dropProposal: (id) => set(s => (s.proposals
    ? { proposals: { ...s.proposals, proposals: s.proposals.proposals.filter(q => q.id !== id) } }
    : {})),

  backToPaste: () => set({ proposals: null }),

  // Accepting adds planned cards, creates any chapters the outline reaches
  // past the end of the manuscript (empty, titled from the outline), and
  // keeps the pasted text as a note unless it came from one.
  acceptProposals: () => {
    const s = get()
    const bookStore = useBookStore.getState()
    const book = bookStore.book
    if (!s.proposals || !book) return
    const accepted = s.proposals.proposals.filter(q => q.accepted)
    const maxChapter = Math.max(book.body.length, ...accepted.map(q => q.chapterNum))
    for (let n = book.body.length + 1; n <= maxChapter; n++) {
      bookStore.addChapter('body', { id: newChapterId(), title: s.proposals.titles[n] || '', type: 'chapter', content: '' })
    }
    const chapters = bookChapters(useBookStore.getState().book)
    const fromNote = s.importFromNoteId
    const noteId = fromNote ?? newId('note')
    const cards: PlannerCard[] = accepted.map(q => ({
      id: newId('card'), source_id: noteId, title: q.title, synopsis: q.synopsis, lines: [q.laneId], who: q.who, changes: '', stakes: '',
      chapter_id: chapters[q.chapterNum - 1]?.id ?? '', status: 'planned', origin: 'outline', updated: nowStamp(),
    }))
    get().mutate(p => ({
      ...p,
      cards: [...p.cards, ...cards],
      notes: fromNote ? p.notes : [{ id: noteId, title: 'Imported outline', body: s.importText, updated: nowStamp() }, ...p.notes],
    }))
    set({ importOpen: false, proposals: null, importFromNoteId: null, noteId, view: 'timeline', panelOpen: true })
    useAppStore.getState().setStatusMessage(`${cards.length} card${cards.length === 1 ? '' : 's'} added to the Planner`)
  },

  refreshDetected: async () => {
    const { book, analysisRevision } = useBookStore.getState()
    if (!book || get().detecting) return
    set({ detecting: true, detectError: '' })
    try {
      const result = await PlannerDetectCards(book as any)
      const current = useBookStore.getState()
      if (current.book !== book || current.analysisRevision !== analysisRevision) return
      if (!result.error && result.source_id && get().planner().source_id !== result.source_id) {
        get().mutate(p => ({ ...p, source_id: result.source_id }))
      }
      set({ detected: result.error ? [] : (result.cards ?? []), detectError: result.error ?? '', detectedRevision: analysisRevision })
    } catch (e) {
      const current = useBookStore.getState()
      if (current.book !== book || current.analysisRevision !== analysisRevision) return
      set({ detected: [], detectError: String(e) })
    } finally {
      set({ detecting: false })
    }
  },
}))
