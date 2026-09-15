// Planner dialogs: New Chapter, New Story Line, and Import Outline (paste,
// then review the proposed cards before accepting).

import { useMemo } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../store/bookStore'
import { usePlannerStore } from '../../store/plannerStore'
import { bookChapters, codexPeople, ensurePlanner } from './plannerModel'

function NewChapterDialog() {
  const book = useBookStore(s => s.book)
  const { chapterDialog, setChapterDialog, createChapter } = usePlannerStore(useShallow(s => ({ chapterDialog: s.chapterDialog, setChapterDialog: s.setChapterDialog, createChapter: s.createChapter })))
  if (!chapterDialog) return null
  const next = (book?.body.length ?? 0) + 1
  return (
    <div className="dialog-overlay" onClick={() => setChapterDialog(null)}>
      <div className="dialog" onClick={e => e.stopPropagation()}>
        <div className="dialog-title">New Chapter</div>
        <div className="dialog-subtitle">Adds an empty chapter to the manuscript body.</div>
        <div className="dialog-field">
          <label className="dialog-label">Chapter Title</label>
          <input
            className="dialog-input" autoFocus value={chapterDialog.title} placeholder={`Chapter ${next}`}
            onChange={e => setChapterDialog({ title: e.target.value })}
            onKeyDown={e => { if (e.key === 'Enter') createChapter(chapterDialog.title); if (e.key === 'Escape') setChapterDialog(null) }}
          />
        </div>
        <div className="pl-dialog-actions">
          <button className="dialog-btn" onClick={() => setChapterDialog(null)}>Cancel</button>
          <button className="dialog-btn primary" onClick={() => createChapter(chapterDialog.title)}>Create Chapter</button>
        </div>
      </div>
    </div>
  )
}

function NewLineDialog() {
  const book = useBookStore(s => s.book)
  const { lineDialog, setLineDialog, addSubplot, addCharacterLane } = usePlannerStore(useShallow(s => ({ lineDialog: s.lineDialog, setLineDialog: s.setLineDialog, addSubplot: s.addSubplot, addCharacterLane: s.addCharacterLane })))
  const planner = ensurePlanner(book)
  const codex = useMemo(() => codexPeople(book), [book])
  if (!lineDialog) return null
  const available = codex.filter(c => !planner.lanes.some(l => l.character_id === c.id))
  const isSubplot = lineDialog.kind === 'Subplot'
  const disabled = isSubplot && !lineDialog.name.trim()
  const create = () => {
    if (disabled) return
    if (isSubplot) addSubplot(lineDialog.name)
    else addCharacterLane(lineDialog.kind)
    setLineDialog(null)
  }
  return (
    <div className="dialog-overlay" onClick={() => setLineDialog(null)}>
      <div className="dialog" onClick={e => e.stopPropagation()}>
        <div className="dialog-title">New Story Line</div>
        <div className="dialog-subtitle">A lane on the timeline. Subplots are typed; character lines come from the codex.</div>
        <div className="dialog-field">
          <label className="dialog-label">Kind</label>
          <select className="dialog-select" value={lineDialog.kind} onChange={e => setLineDialog({ ...lineDialog, kind: e.target.value })}>
            <option value="Subplot">Subplot</option>
            {available.map(c => <option key={c.id} value={c.id}>Character · {c.name}</option>)}
          </select>
        </div>
        {isSubplot && (
          <div className="dialog-field">
            <label className="dialog-label">Name</label>
            <input
              className="dialog-input" autoFocus value={lineDialog.name} placeholder="e.g. The lighthouse weekend"
              onChange={e => setLineDialog({ ...lineDialog, name: e.target.value })}
              onKeyDown={e => { if (e.key === 'Enter') create(); if (e.key === 'Escape') setLineDialog(null) }}
            />
          </div>
        )}
        <div className="pl-dialog-actions">
          <button className="dialog-btn" onClick={() => setLineDialog(null)}>Cancel</button>
          <button className="dialog-btn primary" disabled={disabled} onClick={create}>Create Line</button>
        </div>
      </div>
    </div>
  )
}

function ImportOutlineDialog() {
  const book = useBookStore(s => s.book)
  const { importOpen, importText, importFromNoteId, proposals, setImportText, closeImport, runImport, updateProposal, dropProposal, backToPaste, acceptProposals } = usePlannerStore(useShallow(s => ({
    importOpen: s.importOpen, importText: s.importText, importFromNoteId: s.importFromNoteId, proposals: s.proposals, setImportText: s.setImportText,
    closeImport: s.closeImport, runImport: s.runImport, updateProposal: s.updateProposal, dropProposal: s.dropProposal, backToPaste: s.backToPaste, acceptProposals: s.acceptProposals,
  })))
  const planner = ensurePlanner(book)
  const chapters = useMemo(() => bookChapters(book), [book])
  const codex = useMemo(() => codexPeople(book), [book])
  if (!importOpen) return null
  const note = importFromNoteId ? planner.notes.find(n => n.id === importFromNoteId) : null
  const subtitle = proposals
    ? (note ? `Proposed from “${note.title || 'Untitled note'}” — edit, re-lane, drop, then accept. The note is not changed.` : 'Proposed cards — edit, re-lane, drop, then accept. The manuscript is not touched.')
    : 'Paste or open an outline. Cards are proposed from deterministic rules; nothing is committed until you accept.'

  const items = proposals?.proposals ?? []
  const accepted = items.filter(p => p.accepted)
  const maxChapter = Math.max(chapters.length, ...accepted.map(p => p.chapterNum), 0)
  const chapterOptions = Array.from({ length: Math.max(chapters.length, maxChapter) }, (_, k) => {
    const ch = chapters[k]
    return { num: k + 1, label: ch ? `Ch ${ch.num}${ch.title ? ` · ${ch.title}` : ''}` : `Ch ${k + 1} · new` }
  })
  const missing = maxChapter - chapters.length
  const notice = missing > 0
    ? (missing === 1 ? `Chapter ${maxChapter} does not exist yet; accepting adds it as an empty chapter.` : `Chapters ${chapters.length + 1}–${maxChapter} do not exist yet; accepting adds them as empty chapters.`)
    : ''
  const groups = new Map<number, typeof items>()
  items.forEach(p => { groups.set(p.chapterNum, [...(groups.get(p.chapterNum) ?? []), p]) })

  return (
    <div className="dialog-overlay" onClick={closeImport}>
      <div className="dialog pl-import" onClick={e => e.stopPropagation()}>
        <div className="dialog-title">Import Outline</div>
        <div className="dialog-subtitle">{subtitle}</div>
        {!proposals ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <textarea
              className="dialog-input pl-import-text" autoFocus value={importText} spellCheck={false}
              placeholder="Paste markdown headings, a numbered list, a beat sheet, a chapter-by-chapter synopsis, or a short story."
              onChange={e => setImportText(e.target.value)}
            />
            <div className="pl-dialog-hint">Headings and “Chapter N” lines set positions. List items and paragraphs become cards. Names the codex knows pick the line. Beat names land by manuscript percentage. Nothing is committed until you accept.</div>
            <div className="pl-dialog-actions">
              <button className="dialog-btn" onClick={closeImport}>Cancel</button>
              <button className="dialog-btn primary" disabled={!importText.trim()} onClick={runImport}>Propose Cards</button>
            </div>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <div className="pl-proposals">
              {[...groups.entries()].sort((a, b) => a[0] - b[0]).map(([num, group]) => (
                <div key={num}>
                  <div className="pl-prop-group">Chapter {num} · {chapters[num - 1]?.title || proposals.titles[num] || 'Untitled'}</div>
                  {group.map(p => (
                    <div key={p.id} className={`pl-prop-row${p.accepted ? '' : ' dropped'}`}>
                      <input type="checkbox" checked={p.accepted} onChange={() => updateProposal(p.id, { accepted: !p.accepted })} />
                      <div style={{ minWidth: 0 }}>
                        <input className="pl-prop-title" value={p.title} onChange={e => updateProposal(p.id, { title: e.target.value })} />
                        <div className="pl-prop-sub">{p.synopsis || (p.who.length ? `Who: ${p.who.map(id => codex.find(c => c.id === id)?.name ?? id).join(', ')}` : 'No synopsis')}</div>
                      </div>
                      <select className="pl-prop-select" value={p.laneId} onChange={e => updateProposal(p.id, { laneId: e.target.value })}>
                        {planner.lanes.map(l => <option key={l.id} value={l.id}>{l.name}</option>)}
                      </select>
                      <select className="pl-prop-select" value={p.chapterNum} onChange={e => updateProposal(p.id, { chapterNum: +e.target.value })}>
                        {chapterOptions.map(o => <option key={o.num} value={o.num}>{o.label}</option>)}
                      </select>
                      <button className="pl-prop-drop" title="Drop proposal" onClick={() => dropProposal(p.id)}>×</button>
                    </div>
                  ))}
                </div>
              ))}
              {items.length === 0 && <div className="pl-prop-group">Nothing to propose from that text.</div>}
            </div>
            {notice && <div className="pl-import-notice">{notice}</div>}
            <div className="pl-import-foot">
              <span className="pl-dialog-hint">{items.length} proposed · {accepted.length} accepted</span>
              <button className="dialog-btn" onClick={backToPaste}>Back</button>
              <button className="dialog-btn primary" disabled={accepted.length === 0} onClick={acceptProposals}>Accept {accepted.length} {accepted.length === 1 ? 'Card' : 'Cards'}</button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function DeleteNoteDialog() {
  const book = useBookStore(s => s.book)
  const { noteDeleteId, cancelDeleteNote, deleteNote } = usePlannerStore(useShallow(s => ({ noteDeleteId: s.noteDeleteId, cancelDeleteNote: s.cancelDeleteNote, deleteNote: s.deleteNote })))
  if (!noteDeleteId) return null
  const note = ensurePlanner(book).notes.find(n => n.id === noteDeleteId)
  if (!note) return null
  const words = note.body.trim() ? `${note.body.trim().split(/\s+/).length.toLocaleString()} words` : 'empty'
  return (
    <div className="dialog-overlay" onClick={cancelDeleteNote}>
      <div className="dialog" onClick={e => e.stopPropagation()} onKeyDown={e => { if (e.key === 'Escape') cancelDeleteNote() }}>
        <div className="dialog-title">Delete Note</div>
        <div className="dialog-subtitle">Delete “{note.title || 'Untitled note'}” ({words})? Notes are not kept in Dead ideas; this cannot be undone.</div>
        <div className="pl-dialog-actions">
          <button className="dialog-btn" autoFocus onClick={cancelDeleteNote}>Cancel</button>
          <button className="dialog-btn primary" onClick={() => deleteNote(note.id)}>Delete Note</button>
        </div>
      </div>
    </div>
  )
}

export default function PlannerDialogs() {
  return (
    <>
      <NewChapterDialog />
      <NewLineDialog />
      <DeleteNoteDialog />
      <ImportOutlineDialog />
    </>
  )
}
