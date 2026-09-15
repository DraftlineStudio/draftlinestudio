// The left panel's body while the Planner is open: story lines and codex
// characters for the timeline and board, notes for scratch, chapters for the
// synopsis. The Manuscript | Planner tabs above it live in ChapterPanel.

import { useMemo } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../store/bookStore'
import { usePlannerStore } from '../../store/plannerStore'
import { bookChapters, codexPeople, countWords, displayCards, ensurePlanner, relativeStamp } from './plannerModel'

export default function PlannerSidebar() {
  const book = useBookStore(s => s.book)
  const { view, noteId, detected, setNoteId, newNote, toggleLaneHidden, addCharacterLane, openLineDialog } = usePlannerStore(useShallow(s => ({
    view: s.view, noteId: s.noteId, detected: s.detected, setNoteId: s.setNoteId, newNote: s.newNote,
    toggleLaneHidden: s.toggleLaneHidden, addCharacterLane: s.addCharacterLane, openLineDialog: s.openLineDialog,
  })))
  const planner = ensurePlanner(book)
  const chapters = useMemo(() => bookChapters(book), [book])
  const codex = useMemo(() => codexPeople(book), [book])
  const cards = displayCards(planner, detected)
  const hidden = new Set(planner.hidden_lanes ?? [])
  const linked = planner.cards.filter(c => c.link).length

  let body: React.ReactNode
  if (view === 'scratch') {
    const notes = [...planner.notes].sort((a, b) => (a.system ? 1 : 0) - (b.system ? 1 : 0))
    body = (
      <div className="pl-sidebar">
        <div className="pl-side-head">
          <span className="chapter-section-label">Notes</span>
          <button className="pl-side-add" title="New note" onClick={() => newNote()}>+</button>
        </div>
        {notes.map(n => {
          const snippet = (n.body.split('\n').find(l => l.trim() && !/^#/.test(l)) ?? 'Empty').replace(/^[-*]\s+/, '')
          return (
            <div key={n.id} className={`pl-note-row${n.id === noteId ? ' active' : ''}${n.excluded ? ' excluded' : ''}`} onClick={() => setNoteId(n.id)}>
              <div className="pl-note-row-top">
                <span className="pl-note-title">{n.title || 'Untitled note'}</span>
                {n.excluded && <span className="pl-note-off" title="Not included in analysis">off</span>}
              </div>
              <div className="pl-note-snippet">{snippet}</div>
              <div className="pl-note-meta">{countWords(n.body)} words · {relativeStamp(n.updated)}</div>
            </div>
          )
        })}
      </div>
    )
  } else if (view === 'synopsis') {
    body = (
      <div className="pl-sidebar">
        <div className="pl-side-head"><span className="chapter-section-label">Chapters</span></div>
        {chapters.map(ch => {
          const n = planner.cards.filter(c => c.chapter_id === ch.id).length
          return (
            <div
              key={ch.id} className="chapter-item"
              onClick={() => { const el = document.getElementById(`pl-syn-${ch.id}`); el?.scrollIntoView({ block: 'start', behavior: 'smooth' }) }}
            >
              <span className="chapter-item-title">Chapter {ch.num}{ch.title ? ` — ${ch.title}` : ''}</span>
              <span className="chapter-item-subtitle">{n ? `${n} ${n === 1 ? 'card' : 'cards'}` : 'no cards'}</span>
            </div>
          )
        })}
      </div>
    )
  } else {
    body = (
      <div className="pl-sidebar">
        <div className="pl-side-head">
          <span className="chapter-section-label">Story Lines</span>
          <button className="pl-side-add" title="New subplot line" onClick={openLineDialog}>+</button>
        </div>
        {planner.lanes.map(l => (
          <div key={l.id} className={`pl-lane-row${hidden.has(l.id) ? ' hidden' : ''}`} onClick={() => toggleLaneHidden(l.id)} title={hidden.has(l.id) ? 'Show on timeline' : 'Hide from timeline'}>
            <span className="pl-lane-swatch" style={{ background: hidden.has(l.id) ? 'transparent' : l.color, borderColor: l.color }} />
            <span className="pl-lane-row-name">{l.name}</span>
            <span className="pl-lane-kind">{l.kind}</span>
            <span className="pl-lane-count">{cards.filter(c => c.lines.includes(l.id)).length}</span>
          </div>
        ))}
        <div className="pl-side-divider">
          <div className="pl-side-head"><span className="chapter-section-label">Codex · Characters</span></div>
          {codex.length === 0 && <div className="pl-codex-row"><span className="pl-codex-name" style={{ color: 'var(--text-muted)' }}>No confirmed characters yet</span></div>}
          {codex.map(c => {
            const onTimeline = planner.lanes.some(l => l.character_id === c.id)
            return (
              <div key={c.id} className="pl-codex-row">
                <span className="pl-codex-name">{c.name}</span>
                {onTimeline
                  ? <span className="pl-codex-on">on timeline</span>
                  : <button className="pl-codex-add" onClick={() => addCharacterLane(c.id)}>+ Lane</button>}
              </div>
            )
          })}
        </div>
      </div>
    )
  }

  return (
    <>
      {body}
      <div className="pl-side-footer">{planner.cards.length} cards · {linked} linked to scenes</div>
    </>
  )
}
