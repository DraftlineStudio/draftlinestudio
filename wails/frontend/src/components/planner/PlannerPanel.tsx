// The Planner's right side: a 292px tools panel (overview, note tools,
// synopsis export, or the card inspector) and the 45px glyph rail that
// switches between Timeline, Board, Scratch, and Synopsis.

import { useMemo } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../store/bookStore'
import { usePlannerStore, type PlannerView } from '../../store/plannerStore'
import type { BeatTemplateId } from '../../types/draftline'
import {
  BEAT_NAMES, bookChapters, chapterLabel, codexPeople, countWords, displayCards, ensurePlanner, evidenceText, parseOutline,
  STATUS_COLORS, synopsisMarkdown, type CardStatus,
} from './plannerModel'

const STATUS_NOTES: Record<CardStatus, string> = {
  planned: 'not yet linked to a scene',
  drafted: 'linked to a scene',
  kept: 'the linked scene fulfils the promise',
  drifted: 'scene exists, promised development not found',
  unplanned: 'found in the text, no card',
  found: 'found in the text',
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="dashboard-section">
      <div className="dashboard-section-header"><span className="dashboard-section-title">{title}</span></div>
      {children}
    </div>
  )
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="settings-toggle">
      <input type="checkbox" checked={checked} onChange={e => onChange(e.target.checked)} />
      <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
    </label>
  )
}

function ScratchTools() {
  const book = useBookStore(s => s.book)
  const { noteId, openImport, updateNote } = usePlannerStore(useShallow(s => ({ noteId: s.noteId, openImport: s.openImport, updateNote: s.updateNote })))
  const planner = ensurePlanner(book)
  const note = planner.notes.find(n => n.id === noteId) ?? null
  const proposals = useMemo(() => (note ? parseOutline(note.body, bookChapters(book), codexPeople(book), planner.lanes).proposals : []), [note, book, planner.lanes])
  const codex = codexPeople(book)
  const names = [...new Set(proposals.flatMap(p => p.who))].map(id => codex.find(c => c.id === id)?.name.split(' ')[0] ?? id).join(', ') || 'none'
  return (
    <div className="pl-panel-body">
      <Section title="Import Outline">
        <div className="pl-copy" style={{ marginBottom: 10 }}>Bring in an outline from elsewhere. It lands here as a note first; cards are proposed from it, never written directly.</div>
        <div className="pl-btn-col">
          <button className="dialog-btn" onClick={() => openImport('')}>Paste Outline…</button>
          <label className="dialog-btn" style={{ textAlign: 'center', cursor: 'pointer' }}>
            Open File…
            <input
              type="file" accept=".md,.markdown,.txt,text/plain,text/markdown" style={{ display: 'none' }}
              onChange={e => {
                const file = e.target.files?.[0]
                if (!file) return
                void file.text().then(text => openImport(text))
                e.target.value = ''
              }}
            />
          </label>
        </div>
        <div className="pl-hint" style={{ marginTop: 8 }}>Markdown, numbered lists, beat sheets, chapter synopses, or a short story.</div>
      </Section>
      {note && (
        <Section title="This Note">
          <div className="pl-stat-row"><span>Words</span><span>{countWords(note.body).toLocaleString()}</span></div>
          <div className="pl-stat-row"><span>Would propose</span><span>{proposals.length} {proposals.length === 1 ? 'card' : 'cards'}</span></div>
          <div className="pl-stat-row"><span>Names found</span><span>{names}</span></div>
          <div className="pl-toggle-row divided">
            <span>Include in analysis</span>
            <Toggle checked={!note.excluded} onChange={v => updateNote(note.id, { excluded: !v })} />
          </div>
          <div className="pl-hint" style={{ marginTop: 4 }}>
            {note.excluded ? 'Ignored by Propose Cards. Kept as a note only.' : 'Available to Propose Cards.'}
          </div>
          <div style={{ marginTop: 10 }}>
            <button className="dialog-btn primary" style={{ width: '100%', height: 25, fontSize: 11.5 }} disabled={!!note.excluded} onClick={() => openImport(note.body, note.id)}>Propose Cards…</button>
          </div>
          <div className="pl-hint" style={{ marginTop: 8 }}>Runs the outline rules on this note and opens the proposal view. Accepting adds planned cards; the note stays as it is.</div>
        </Section>
      )}
      {note?.system === 'dead' && (
        <Section title="Dead Ideas">
          <div className="pl-copy">Every card deleted from the timeline or board, and every dismissed unplanned development, is written here automatically. Turn analysis on and propose cards to bring one back.</div>
        </Section>
      )}
    </div>
  )
}

function SynopsisTools() {
  const book = useBookStore(s => s.book)
  const newNote = usePlannerStore(s => s.newNote)
  const setStatus = useBookStore(s => s.book) // placeholder to keep hook order stable
  void setStatus
  const planner = ensurePlanner(book)
  const chapters = bookChapters(book)
  const markdown = () => synopsisMarkdown(planner, chapters)
  const words = chapters.reduce((a, ch) => a + countWords(planner.synopsis?.[ch.id] ?? ''), 0)
  return (
    <div className="pl-panel-body">
      <Section title="Export">
        <div className="pl-btn-col">
          <button className="dialog-btn" onClick={() => { void navigator.clipboard?.writeText(markdown()) }}>Copy as Markdown</button>
          <button className="dialog-btn" onClick={() => newNote(`Synopsis — ${new Date().toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}`, markdown())}>Save as Scratch</button>
        </div>
        <div className="pl-hint" style={{ marginTop: 8 }}>{words ? `${words.toLocaleString()} edited words · ` : ''}one paragraph per chapter, chapters without cards left out.</div>
      </Section>
      <Section title="Source">
        <div className="pl-copy">Each paragraph joins the synopses of that chapter's cards in timeline order. Chapters without cards are left blank rather than invented.</div>
      </Section>
    </div>
  )
}

function Overview() {
  const book = useBookStore(s => s.book)
  const { detected, detectError, setPlotWalker, setBeatTemplate } = usePlannerStore(useShallow(s => ({
    detected: s.detected, detectError: s.detectError, setPlotWalker: s.setPlotWalker, setBeatTemplate: s.setBeatTemplate,
  })))
  const planner = ensurePlanner(book)
  const cards = displayCards(planner, detected)
  const counts: Record<string, number> = {}
  cards.forEach(c => { counts[c.st] = (counts[c.st] ?? 0) + 1 })
  const order: CardStatus[] = planner.plot_walker ? ['planned', 'drafted', 'unplanned'] : ['planned', 'drafted']
  return (
    <div className="pl-panel-body">
      <Section title="Plot Walker">
        <div className="pl-toggle-row">
          <span>Reconcile cards with the text</span>
          <Toggle checked={!!planner.plot_walker} onChange={setPlotWalker} />
        </div>
        {planner.plot_walker
          ? <div className="pl-copy" style={{ marginTop: 8 }}>Developments the narrative engine finds in the text appear as <span style={{ color: STATUS_COLORS.unplanned }}>unplanned</span> cards to adopt or dismiss. Matching them to your cards as <span style={{ color: STATUS_COLORS.kept }}>kept</span> or <span style={{ color: STATUS_COLORS.drifted }}>drifted</span> arrives with the next Plot Walker milestone.</div>
          : <div className="pl-hint" style={{ marginTop: 8 }}>Off. Linking a card to a scene marks it drafted; nothing is checked against the text.</div>}
        {planner.plot_walker && detectError && <div className="pl-hint" style={{ color: 'var(--status-error, #E06C75)' }}>{detectError}</div>}
      </Section>
      <Section title="Beat Template">
        <select className="dialog-select" value={planner.beat_template ?? 'none'} onChange={e => setBeatTemplate(e.target.value as BeatTemplateId)}>
          {(Object.keys(BEAT_NAMES) as BeatTemplateId[]).map(k => <option key={k} value={k}>{BEAT_NAMES[k]}</option>)}
        </select>
        <div className="pl-hint">Beat marks sit at manuscript percentages across the chapter axis.</div>
      </Section>
      <Section title="Cards by Status">
        {order.map(k => (
          <div key={k} className="pl-status-row">
            <span className="pl-card-dot" style={{ background: STATUS_COLORS[k] }} />
            <span>{k}</span>
            <span>{counts[k] ?? 0}</span>
          </div>
        ))}
        <div className="pl-hint" style={{ marginTop: 8 }}>Select a card to edit it. Drag to another chapter or line; double-click a cell to add one there.</div>
      </Section>
    </div>
  )
}

function Inspector({ id }: { id: string }) {
  const book = useBookStore(s => s.book)
  const { detected, linkOpen, setLinkOpen, updateCard, linkCard, deleteCard, adoptDetected, dismissDetected } = usePlannerStore(useShallow(s => ({
    detected: s.detected, linkOpen: s.linkOpen, setLinkOpen: s.setLinkOpen, updateCard: s.updateCard, linkCard: s.linkCard,
    deleteCard: s.deleteCard, adoptDetected: s.adoptDetected, dismissDetected: s.dismissDetected,
  })))
  const planner = ensurePlanner(book)
  const chapters = useMemo(() => bookChapters(book), [book])
  const codex = useMemo(() => codexPeople(book), [book])
  const card = displayCards(planner, detected).find(c => c.id === id)
  if (!card) return <div className="pl-panel-body"><div className="pl-hint">This card is no longer on the timeline.</div></div>
  const color = STATUS_COLORS[card.st]
  const linkedChapter = card.link ? chapters.find(ch => ch.id === card.link!.chapter_id) : undefined
  const linkLabel = card.link ? `${chapterLabel(linkedChapter, 'Chapter')}${(linkedChapter?.scenes ?? 0) > 1 ? ` · Scene ${card.link.scene}` : ''}` : ''
  const scenes = chapters.filter(ch => ch.drafted).flatMap(ch => Array.from({ length: ch.scenes }, (_, i) => ({
    chapter_id: ch.id, scene: i + 1, label: `${chapterLabel(ch)}${ch.scenes > 1 ? ` · Scene ${i + 1}` : ''}`,
  })))
  const evidence = evidenceText(card)
  const locked = !!card.unplanned
  const setField = (key: 'title' | 'synopsis' | 'changes' | 'stakes') => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => updateCard(card.id, { [key]: e.target.value })

  return (
    <div className="pl-inspector">
      <div className="pl-insp-status">
        <span className="pl-card-dot" style={{ background: color }} />
        <span className="pl-insp-status-label" style={{ color }}>{card.st}</span>
        <span className="pl-insp-note">{STATUS_NOTES[card.st]}</span>
      </div>

      {card.unplanned && (
        <div className="pl-callout">
          <div className="pl-copy">Plot Walker found this <span style={{ color: 'var(--text-primary)' }}>{card.dev_kind || 'development'}</span> in the text with no card promising it.</div>
          <div className="pl-callout-actions">
            <button className="dialog-btn primary" onClick={() => adoptDetected(card.id)}>Adopt as card</button>
            <button className="dialog-btn" onClick={() => dismissDetected(card.id)}>Dismiss</button>
          </div>
        </div>
      )}

      <div className="pl-field">
        <span className="pl-field-label">Title</span>
        <input className="dialog-input" value={card.title} readOnly={locked} onChange={setField('title')} />
      </div>
      <div className="pl-field">
        <span className="pl-field-label">Synopsis</span>
        <textarea className="dialog-input" value={card.synopsis} readOnly={locked} onChange={setField('synopsis')} />
      </div>

      <div className="pl-field">
        <span className="pl-field-label">Lines</span>
        <div className="pl-chips">
          {planner.lanes.map(l => {
            const on = card.lines.includes(l.id)
            return (
              <button
                key={l.id} className={`pl-chip-btn${on ? ' on' : ''}`} style={on ? { borderColor: l.color } : undefined}
                onClick={() => { if (locked) return; const lines = on ? card.lines.filter(x => x !== l.id) : [...card.lines, l.id]; if (lines.length) updateCard(card.id, { lines }) }}
              >
                <span className="pl-chip-swatch" style={{ background: l.color }} />{l.name}
              </button>
            )
          })}
        </div>
        <div className="pl-hint" style={{ marginTop: 5 }}>First line is where the card sits; others are drawn as a crossing.</div>
      </div>

      <div className="pl-field">
        <span className="pl-field-label">Who</span>
        <div className="pl-chips">
          {codex.length === 0 && <span className="pl-hint" style={{ marginTop: 0 }}>Confirm characters in the codex to name them here.</span>}
          {codex.map(c => {
            const on = card.who.includes(c.id)
            return (
              <button key={c.id} className={`pl-chip-btn${on ? ' on' : ''}`} onClick={() => { if (!locked) updateCard(card.id, { who: on ? card.who.filter(x => x !== c.id) : [...card.who, c.id] }) }}>
                {c.name}
              </button>
            )
          })}
        </div>
      </div>

      <div className="pl-field">
        <span className="pl-field-label">What changes</span>
        <textarea className="dialog-input short" value={card.changes ?? ''} readOnly={locked} placeholder="The development this card promises" onChange={setField('changes')} />
      </div>
      <div className="pl-field">
        <span className="pl-field-label">Stakes</span>
        <input className="dialog-input" value={card.stakes ?? ''} readOnly={locked} placeholder="What is at risk" onChange={setField('stakes')} />
      </div>
      <div className="pl-field">
        <span className="pl-field-label">Position</span>
        <select className="dialog-select" value={card.chapter_id} disabled={locked} onChange={e => updateCard(card.id, { chapter_id: e.target.value })}>
          {chapters.map(ch => <option key={ch.id} value={ch.id}>{chapterLabel(ch)}</option>)}
          <option value="">Later</option>
        </select>
      </div>

      <div className="pl-field">
        <span className="pl-field-label">Linked scene</span>
        {card.link ? (
          <div className="pl-link-box">
            <span>{linkLabel}</span>
            {!locked && <button className="pl-link-unlink" onClick={() => linkCard(card.id, null)}>Unlink</button>}
          </div>
        ) : linkOpen ? (
          <div className="pl-link-picker">
            {scenes.length === 0 && <div className="pl-link-opt" style={{ color: 'var(--text-muted)', cursor: 'default' }}>No written scenes yet.</div>}
            {scenes.map(sc => (
              <div key={`${sc.chapter_id}:${sc.scene}`} className="pl-link-opt" onClick={() => { linkCard(card.id, { chapter_id: sc.chapter_id, scene: sc.scene }); setLinkOpen(false) }}>
                <span className="pl-link-opt-dot" />{sc.label}
              </div>
            ))}
            <div className="pl-link-cancel" onClick={() => setLinkOpen(false)}>Cancel</div>
          </div>
        ) : (
          <>
            <button className="dialog-btn sm" onClick={() => setLinkOpen(true)}>Link to Scene…</button>
            <div className="pl-hint" style={{ marginTop: 5 }}>Linking marks the card drafted.</div>
          </>
        )}
      </div>

      {evidence && (
        <div className="pl-field">
          <span className="pl-field-label">Plot Walker · {card.dev_kind || 'development'}</span>
          <div className="pl-evidence" style={{ borderLeftColor: color }}>{evidence}</div>
          <div className="pl-hint" style={{ marginTop: 5 }}>Evidence span in {linkLabel || chapterLabel(chapters.find(ch => ch.id === card.chapter_id))}.</div>
        </div>
      )}

      {!locked && (
        <div className="pl-insp-foot">
          <button className="pl-delete-btn" onClick={() => deleteCard(card.id)}>Delete Card</button>
        </div>
      )}
    </div>
  )
}

const GLYPHS: { view: PlannerView; title: string; icon: React.ReactNode }[] = [
  { view: 'timeline', title: 'Timeline', icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M3 6h18" /><path d="M3 12h18" /><path d="M3 18h18" /><rect x="6" y="4" width="6" height="4" rx="1" fill="var(--bg-panel)" /><rect x="13" y="10" width="7" height="4" rx="1" fill="var(--bg-panel)" /><rect x="8" y="16" width="5" height="4" rx="1" fill="var(--bg-panel)" /></svg> },
  { view: 'board', title: 'Board', icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="3" width="5" height="18" rx="1" /><rect x="10" y="3" width="5" height="12" rx="1" /><rect x="17" y="3" width="4" height="8" rx="1" /></svg> },
  { view: 'scratch', title: 'Scratch', icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M12 20h9" /><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z" /></svg> },
  { view: 'synopsis', title: 'Synopsis', icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" /><polyline points="14 2 14 8 20 8" /><line x1="16" y1="13" x2="8" y2="13" /><line x1="16" y1="17" x2="8" y2="17" /></svg> },
]

export default function PlannerPanel() {
  const { view, panelOpen, selected, pick, setPanelOpen, select } = usePlannerStore(useShallow(s => ({
    view: s.view, panelOpen: s.panelOpen, selected: s.selected, pick: s.pick, setPanelOpen: s.setPanelOpen, select: s.select,
  })))
  const isCards = view === 'timeline' || view === 'board'
  const title = selected && isCards ? 'Card' : view === 'scratch' ? 'Scratch' : view === 'synopsis' ? 'Synopsis' : 'Overview'
  return (
    <div className="pl-right">
      {panelOpen && (
        <div className="pl-panel">
          <div className="pl-panel-head">
            <div className="pl-panel-head-left">
              {selected && isCards && (
                <button className="pl-panel-back" title="Back to overview" onClick={() => select(null)}>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
                </button>
              )}
              <span className="pl-panel-title">{title}</span>
            </div>
            <button className="pl-panel-close" title="Hide panel" onClick={() => setPanelOpen(false)}>
              <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"><path d="M2 2l8 8M10 2l-8 8" /></svg>
            </button>
          </div>
          {view === 'scratch' && <ScratchTools />}
          {view === 'synopsis' && <SynopsisTools />}
          {isCards && (selected ? <Inspector id={selected} /> : <Overview />)}
        </div>
      )}
      <div className="glyph-bar">
        {GLYPHS.map(g => (
          <button key={g.view} className={`glyph-btn${panelOpen && view === g.view ? ' active' : ''}`} title={g.title} onClick={() => pick(g.view)}>{g.icon}</button>
        ))}
      </div>
    </div>
  )
}
