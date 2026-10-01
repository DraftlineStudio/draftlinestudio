// Planner dialogs: New Story Line, Import Outline (paste, then review the
// proposed cards before accepting), and the deletion asks for notes, one card,
// and every card at once.

import { useMemo, useState } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../store/bookStore'
import { usePlannerStore } from '../../store/plannerStore'
import { codexPeople, ensurePlanner } from './plannerModel'

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
  const { importOpen, importText, importFromNoteId, proposals, setImportText, closeImport, runImport, updateProposal, updateProposalCharacter, setProposalGroupLane, addProposalLane, dropProposal, backToPaste, acceptProposals } = usePlannerStore(useShallow(s => ({
    importOpen: s.importOpen, importText: s.importText, importFromNoteId: s.importFromNoteId, proposals: s.proposals, setImportText: s.setImportText,
    closeImport: s.closeImport, runImport: s.runImport, updateProposal: s.updateProposal, updateProposalCharacter: s.updateProposalCharacter, setProposalGroupLane: s.setProposalGroupLane,
    addProposalLane: s.addProposalLane, dropProposal: s.dropProposal, backToPaste: s.backToPaste, acceptProposals: s.acceptProposals,
  })))
  const [trackGroupId, setTrackGroupId] = useState<string | null>(null)
  const [trackName, setTrackName] = useState('')
  const planner = ensurePlanner(book)
  const codex = useMemo(() => codexPeople(book), [book])
  if (!importOpen) return null
  const note = importFromNoteId ? planner.notes.find(n => n.id === importFromNoteId) : null
  const subtitle = proposals
    ? (note ? `Proposed from “${note.title || 'Untitled note'}” — review cards, plot tracks, and characters, then accept. The note is not changed.` : 'Review cards, plot tracks, and characters, then accept. Every card starts in Later; the manuscript is not touched.')
    : 'Paste or open any outline. Draftline groups its larger movements into cards; nothing is committed until you accept.'

  const items = proposals?.proposals ?? []
  const proposedLanes = proposals?.lanes ?? []
  const characters = proposals?.characters ?? []
  const accepted = items.filter(p => p.accepted)
  const acceptedCharacters = characters.filter(character => character.accepted)
  const groups = new Map<string, typeof items>()
  items.forEach(p => { groups.set(p.groupId, [...(groups.get(p.groupId) ?? []), p]) })

  return (
    <div className="dialog-overlay" onClick={closeImport}>
      <div className="dialog pl-import" onClick={e => e.stopPropagation()}>
        <div className="dialog-title">Import Outline</div>
        <div className="dialog-subtitle">{subtitle}</div>
        {!proposals ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <textarea
              className="dialog-input pl-import-text" autoFocus value={importText} spellCheck={false}
              placeholder="Paste any outline, notes, beat sheet, synopsis, or short story. Formatting is optional."
              onChange={e => setImportText(e.target.value)}
            />
            <div className="pl-dialog-hint">Headings identify larger movements and optional story-line groups. Supporting bullets and paragraphs stay with their movement instead of becoming tiny cards. Accepted cards go to Later so you can attach them to chapters when the manuscript is ready.</div>
            <div className="pl-dialog-actions">
              <button className="dialog-btn" onClick={closeImport}>Cancel</button>
              <button className="dialog-btn primary" disabled={!importText.trim()} onClick={runImport}>Propose Cards</button>
            </div>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {characters.length > 0 && (
              <div className="pl-proposal-characters">
                <div className="pl-prop-group"><span>Characters from outline</span><span className="pl-prop-group-spacer" /><span>Review before adding to the codex</span></div>
                {characters.map(character => (
                  <div key={character.id} className={`pl-prop-character${character.accepted ? '' : ' dropped'}`}>
                    <input type="checkbox" checked={character.accepted} onChange={() => updateProposalCharacter(character.id, { accepted: !character.accepted })} />
                    <div className="pl-prop-main">
                      <input className="pl-prop-title" value={character.name} onChange={e => updateProposalCharacter(character.id, { name: e.target.value })} />
                      <div className="pl-prop-syn">{character.description || 'No profile details supplied.'}</div>
                    </div>
                    <label className="pl-prop-character-track">
                      <input type="checkbox" checked={character.createLane} disabled={!character.accepted} onChange={e => updateProposalCharacter(character.id, { createLane: e.target.checked })} />
                      Character track
                    </label>
                  </div>
                ))}
              </div>
            )}
            <div className="pl-proposals">
              {[...groups.entries()].map(([groupId, group]) => {
                const groupName = group[0]?.groupName || 'Outline'
                const laneValue = group.every(p => p.laneId === group[0]?.laneId) ? group[0]?.laneId : ''
                const createTrack = () => {
                  if (!trackName.trim()) return
                  addProposalLane(groupId, trackName)
                  setTrackGroupId(null)
                  setTrackName('')
                }
                return (
                <div key={groupId}>
                  <div className="pl-prop-group">
                    <span>{groupName}</span>
                    <span className="pl-prop-group-spacer" />
                    {trackGroupId === groupId ? (<>
                      <input
                        className="dialog-input pl-prop-track-input" autoFocus value={trackName} placeholder="Plot track name"
                        onChange={e => setTrackName(e.target.value)}
                        onKeyDown={e => { if (e.key === 'Enter') createTrack(); if (e.key === 'Escape') { setTrackGroupId(null); setTrackName('') } }}
                      />
                      <button className="dialog-btn" disabled={!trackName.trim()} onClick={createTrack}>Add</button>
                      <button className="pl-prop-drop" title="Cancel" onClick={() => { setTrackGroupId(null); setTrackName('') }}>×</button>
                    </>) : (<>
                      <span>Plot track</span>
                      <select className="pl-prop-select" value={laneValue} onChange={e => setProposalGroupLane(groupId, e.target.value)}>
                        {!laneValue && <option value="" disabled>Mixed tracks</option>}
                        {planner.lanes.map(l => <option key={l.id} value={l.id}>{l.name}</option>)}
                        {proposedLanes.map(l => <option key={l.id} value={l.id}>New · {l.name}</option>)}
                      </select>
                      <button className="pl-prop-new-track" onClick={() => { setTrackGroupId(groupId); setTrackName('') }}>+ New track</button>
                    </>)}
                  </div>
                  {group.map(p => (
                    <div key={p.id} className={`pl-prop-row${p.accepted ? '' : ' dropped'}`}>
                      <input type="checkbox" checked={p.accepted} onChange={() => updateProposal(p.id, { accepted: !p.accepted })} />
                      <div style={{ minWidth: 0 }}>
                        <input className="pl-prop-title" value={p.title} onChange={e => updateProposal(p.id, { title: e.target.value })} />
                        <div className="pl-prop-sub">{p.synopsis || (p.who.length ? `Who: ${p.who.map(id => codex.find(c => c.id === id)?.name ?? id).join(', ')}` : 'No synopsis')}</div>
                      </div>
                      <select className="pl-prop-select" value={p.laneId} onChange={e => updateProposal(p.id, { laneId: e.target.value })}>
                        {planner.lanes.map(l => <option key={l.id} value={l.id}>{l.name}</option>)}
                        {proposedLanes.map(l => <option key={l.id} value={l.id}>New · {l.name}</option>)}
                      </select>
                      <button className="pl-prop-drop" title="Drop proposal" onClick={() => dropProposal(p.id)}>×</button>
                    </div>
                  ))}
                </div>
              )})}
              {items.length === 0 && (
                <div className="pl-prop-group">{note ? 'No new cards — accepted movements from this note are already in the Planner.' : 'Nothing to propose from that text.'}</div>
              )}
            </div>
            <div className="pl-import-foot">
              <span className="pl-dialog-hint">{accepted.length}/{items.length} cards · {acceptedCharacters.length}/{characters.length} characters · cards start in Later</span>
              <button className="dialog-btn" onClick={backToPaste}>Back</button>
              <button className="dialog-btn primary" disabled={accepted.length === 0 && acceptedCharacters.length === 0} onClick={acceptProposals}>Accept Import</button>
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

// The two card-delete dialogs are mounted only while open, so the Dead ideas
// checkbox and the typed confirmation start fresh on each ask.
function DeleteCardDialog({ id }: { id: string }) {
  const book = useBookStore(s => s.book)
  const { cancelDeleteCard, deleteCard } = usePlannerStore(useShallow(s => ({ cancelDeleteCard: s.cancelDeleteCard, deleteCard: s.deleteCard })))
  const [recordDead, setRecordDead] = useState(true)
  const card = ensurePlanner(book).cards.find(c => c.id === id)
  if (!card) return null
  return (
    <div className="dialog-overlay" onClick={cancelDeleteCard}>
      <div className="dialog" onClick={e => e.stopPropagation()} onKeyDown={e => { if (e.key === 'Escape') cancelDeleteCard() }}>
        <div className="dialog-title">Delete Card</div>
        <div className="dialog-subtitle">Delete “{card.title || 'Untitled card'}”? The card leaves the timeline; the manuscript is not touched.</div>
        <label className="pl-dialog-check">
          <input type="checkbox" checked={recordDead} onChange={e => setRecordDead(e.target.checked)} />
          Record it in Dead ideas
        </label>
        <div className="pl-dialog-hint">{recordDead ? 'Its title, synopsis and position are kept in the Dead ideas note.' : 'The card is discarded without a record.'}</div>
        <div className="pl-dialog-actions">
          <button className="dialog-btn" autoFocus onClick={cancelDeleteCard}>Cancel</button>
          <button className="dialog-btn primary" onClick={() => deleteCard(card.id, recordDead)}>Delete Card</button>
        </div>
      </div>
    </div>
  )
}

function DeleteAllCardsDialog() {
  const book = useBookStore(s => s.book)
  const { closeDeleteAll, deleteAllCards } = usePlannerStore(useShallow(s => ({ closeDeleteAll: s.closeDeleteAll, deleteAllCards: s.deleteAllCards })))
  const [recordDead, setRecordDead] = useState(false)
  const [typed, setTyped] = useState('')
  const count = ensurePlanner(book).cards.length
  const phrase = `DELETE ${count}`
  // The count is part of the phrase, so the confirmation cannot be muscle
  // memory from a smaller delete. Spacing is forgiven; wording and count are
  // not.
  const armed = typed.trim().replace(/\s+/g, ' ') === phrase
  const confirm = () => { if (armed) deleteAllCards(recordDead) }
  return (
    <div className="dialog-overlay" onClick={closeDeleteAll}>
      <div className="dialog" onClick={e => e.stopPropagation()} onKeyDown={e => { if (e.key === 'Escape') closeDeleteAll() }}>
        <div className="dialog-title">Delete All Cards</div>
        <div className="dialog-subtitle">
          All {count} {count === 1 ? 'card' : 'cards'} in the Planner will be deleted. Story lines, notes and the manuscript are left alone. This cannot be undone.
        </div>
        <label className="pl-dialog-check">
          <input type="checkbox" checked={recordDead} onChange={e => setRecordDead(e.target.checked)} />
          Record them in Dead ideas
        </label>
        <div className="pl-dialog-hint">{recordDead ? `Adds ${count} ${count === 1 ? 'entry' : 'entries'} to the Dead ideas note.` : 'The cards are discarded without a record.'}</div>
        <div className="dialog-field pl-confirm-field">
          <label className="dialog-label">Type <span className="pl-confirm-phrase">{phrase}</span> to confirm</label>
          <input
            className="dialog-input pl-confirm-input" autoFocus value={typed} spellCheck={false} placeholder={phrase}
            onChange={e => setTyped(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter') confirm() }}
          />
        </div>
        <div className="pl-dialog-actions">
          <button className="dialog-btn" onClick={closeDeleteAll}>Cancel</button>
          <button className="dialog-btn primary" disabled={!armed} onClick={confirm}>Delete {count} {count === 1 ? 'Card' : 'Cards'}</button>
        </div>
      </div>
    </div>
  )
}

export default function PlannerDialogs() {
  const { cardDeleteId, deleteAllOpen } = usePlannerStore(useShallow(s => ({ cardDeleteId: s.cardDeleteId, deleteAllOpen: s.deleteAllOpen })))
  return (
    <>
      <NewLineDialog />
      <DeleteNoteDialog />
      {cardDeleteId && <DeleteCardDialog id={cardDeleteId} />}
      {deleteAllOpen && <DeleteAllCardsDialog />}
      <ImportOutlineDialog />
    </>
  )
}
