// The Planner's scratchpad, reachable from the writing window: notes on the
// left, the note on the right, editable in place. It is the same notes the
// Planner shows, stored in planner.json, so a note written here is the note
// Propose Cards reads. Nothing here touches the manuscript.

import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../store/bookStore'
import { usePlannerStore } from '../../store/plannerStore'
import { countWords, ensurePlanner, relativeStamp } from '../planner/plannerModel'
import type { BookData } from '../../types/draftline'

interface Props {
  book: BookData
}

export default function ScratchpadPanel({ book }: Props) {
  const { noteId, setNoteId, newNote, updateNote, noteMono, toggleNoteMono } = usePlannerStore(useShallow(s => ({
    noteId: s.noteId, setNoteId: s.setNoteId, newNote: s.newNote,
    updateNote: s.updateNote, noteMono: s.noteMono, toggleNoteMono: s.toggleNoteMono,
  })))
  const setViewMode = useBookStore(s => s.setViewMode)
  const planner = ensurePlanner(book)
  // Dead Ideas sits last: it is written to automatically, not by hand.
  const notes = [...planner.notes].sort((a, b) => (a.system ? 1 : 0) - (b.system ? 1 : 0))
  const note = notes.find(n => n.id === noteId) ?? notes[0] ?? null

  if (!notes.length) {
    return (
      <div className="story-search-empty">
        <p>No notes yet. The scratchpad is for ideas you may or may not use.</p>
        <button type="button" className="dialog-btn" onClick={() => newNote()}>New note</button>
      </div>
    )
  }

  return (
    <div className="scratchpad-dock">
      <div className="scratchpad-dock-list">
        <div className="scratchpad-dock-list-head">
          <span className="chapter-section-label">Notes</span>
          <button type="button" className="pl-side-add" title="New note" onClick={() => newNote()}>+</button>
        </div>
        {notes.map(n => {
          const snippet = (n.body.split('\n').find(l => l.trim() && !/^#/.test(l)) ?? 'Empty').replace(/^[-*]\s+/, '')
          return (
            <button
              type="button"
              key={n.id}
              className={`scratchpad-dock-row${n.id === note?.id ? ' active' : ''}`}
              onClick={() => setNoteId(n.id)}
            >
              <span className="scratchpad-dock-row-title">{n.title || 'Untitled note'}</span>
              <span className="scratchpad-dock-row-snippet">{snippet}</span>
              <span className="scratchpad-dock-row-meta">{countWords(n.body)} words · {relativeStamp(n.updated)}</span>
            </button>
          )
        })}
      </div>

      <div className="scratchpad-dock-body">
        {note && (
          <>
            <div className="scratchpad-dock-bar">
              <input
                className="scratchpad-dock-title"
                value={note.title}
                placeholder="Untitled note"
                readOnly={!!note.system}
                onChange={e => { if (!note.system) updateNote(note.id, { title: e.target.value }) }}
              />
              <button
                type="button"
                className={`toolbar-btn${noteMono ? ' active' : ''}`}
                title="Monospace"
                onClick={toggleNoteMono}
              >
                Mono
              </button>
              <button
                type="button"
                className="toolbar-btn"
                title="Open this note in the Planner"
                onClick={() => { setNoteId(note.id); usePlannerStore.setState({ view: 'scratch', panelOpen: true }); setViewMode('planner') }}
              >
                Open in Planner
              </button>
            </div>
            <textarea
              className={`scratchpad-dock-text${noteMono ? ' mono' : ''}`}
              value={note.body}
              onChange={e => updateNote(note.id, { body: e.target.value })}
              placeholder="Write anything. Nothing here touches the manuscript."
              spellCheck={false}
            />
          </>
        )}
      </div>
    </div>
  )
}
