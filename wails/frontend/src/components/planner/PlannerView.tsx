// Planner center pane: the toolbar and one of four views — the story-line
// timeline, the card board, scratch notes, or the synopsis. Persisted data
// comes from the book (planner.json); transient view state from plannerStore.

import { useEffect, useRef } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../store/bookStore'
import { usePlannerStore } from '../../store/plannerStore'
import type { PlannerLane } from '../../types/draftline'
import {
  BEAT_NAMES, beatMarks, bookChapters, chapterLabel, codexPeople, columnConnectors, countWords, displayCards,
  ensurePlanner, laneChips, laneRows, LATER_COLUMN_ID, layoutMetrics, MAIN_LANE_ID, relativeStamp, STATUS_COLORS, whoText,
  type CodexPerson, type DisplayCard, type PlannerChapter,
} from './plannerModel'
import { generatedSynopsis, synopsisEntries, synopsisText, synopsisTrace } from './plannerSynopsis'
import PlannerDialogs from './PlannerDialogs'

const LATER_ID = LATER_COLUMN_ID

interface Column {
  id: string
  kicker: string
  title: string
  muted: boolean
  meta: string
  dim: boolean
  later: boolean
}

function columnsFor(chapters: PlannerChapter[], cards: DisplayCard[]): Column[] {
  const cols: Column[] = chapters.map(ch => ({
    id: ch.id,
    kicker: `Chapter ${ch.num}`,
    title: ch.title || 'Untitled',
    muted: !ch.drafted && !ch.title,
    meta: ch.drafted ? `${ch.words.toLocaleString()} words · ${ch.scenes} ${ch.scenes === 1 ? 'scene' : 'scenes'}` : 'empty',
    dim: !ch.drafted,
    later: false,
  }))
  const later = cards.filter(c => c.chapter_id === LATER_ID).length
  cols.push({ id: LATER_ID, kicker: 'Later', title: 'Unpinned', muted: true, meta: `${later} ${later === 1 ? 'card' : 'cards'} · no chapter yet`, dim: true, later: true })
  return cols
}

function CardTile({ card, lane, codex, selected, compact, onSelect, onDragStart, onDropBefore }: {
  card: DisplayCard; lane: PlannerLane | undefined; codex: CodexPerson[]; selected: boolean; compact: boolean
  onSelect: () => void; onDragStart: (e: React.DragEvent) => void; onDropBefore?: (e: React.DragEvent) => void
}) {
  const color = STATUS_COLORS[card.st]
  return (
    <div
      className={`pl-card${selected ? ' selected' : ''}`}
      style={{ height: compact ? 66 : 92, borderLeftColor: lane?.color }}
      draggable
      onDragStart={onDragStart}
      onDragOver={onDropBefore ? e => e.preventDefault() : undefined}
      onDrop={onDropBefore}
      onClick={e => { e.stopPropagation(); onSelect() }}
    >
      <div className="pl-card-top">
        <span className="pl-card-title">{card.title}</span>
        <span className="pl-card-dot" title={card.st} style={{ background: color }} />
      </div>
      <div className="pl-card-syn" style={{ WebkitLineClamp: compact ? 1 : 3 }}>{card.synopsis || '—'}</div>
      <div className="pl-card-foot">
        <span className="pl-card-status" style={{ color }}>{card.st}</span>
        <span className="pl-card-who">{whoText(card, codex)}</span>
      </div>
    </div>
  )
}

function TimelineView() {
  const { selected, drag, select, setDrag, moveCard, addCard, openLineDialog, openChapterDialog, openImport } = usePlannerStore(useShallow(s => ({
    selected: s.selected, drag: s.drag, select: s.select, setDrag: s.setDrag, moveCard: s.moveCard,
    addCard: s.addCard, openLineDialog: s.openLineDialog, openChapterDialog: s.openChapterDialog, openImport: s.openImport,
  })))
  const book = useBookStore(s => s.book)
  const planner = ensurePlanner(book)
  const chapters = bookChapters(book)
  const codex = codexPeople(book)
  const cards = displayCards(planner)
  const compact = planner.compact !== false
  const hasBeats = !!planner.beat_template && planner.beat_template !== 'none'
  const m = layoutMetrics(compact, hasBeats)
  const hidden = new Set(planner.hidden_lanes ?? [])
  const lanes = planner.lanes.filter(l => !hidden.has(l.id))
  const columns = columnsFor(chapters, cards)
  const rows = laneRows(lanes, cards, columns.map(c => c.id), m)
  const gridH = rows.reduce((a, r) => a + r.height, m.headerH)
  const gridW = chapters.length * m.colW
  const marks = beatMarks(planner.beat_template, gridW)

  const allowDrop = (e: React.DragEvent) => e.preventDefault()

  return (
    <div className="pl-timeline">
      <div className="pl-grid">
        <div className="pl-lanes">
          <div className="pl-lanes-head" style={{ height: m.headerH }}><span className="pl-kicker">Story lines</span></div>
          {rows.map(r => (
            <div key={r.lane.id} className="pl-lane-cell" style={{ height: r.height }}>
              <span className="pl-lane-bar" style={{ background: r.lane.color }} />
              <div style={{ minWidth: 0 }}>
                <div className="pl-lane-name">{r.lane.name}</div>
                <div className="pl-lane-sub">{r.lane.kind} · {cards.filter(c => c.lines.includes(r.lane.id)).length} cards</div>
              </div>
            </div>
          ))}
          <div className="pl-lane-add" onClick={openLineDialog}><span className="pl-lane-add-bar" /><span>+ Story line</span></div>
        </div>
        <div className="pl-cols">
          {hasBeats && (
            <div className="pl-beats" style={{ width: gridW }}>
              {marks.map(b => <div key={b.name} className="pl-beat" style={{ left: b.left }}><span>{b.name}</span></div>)}
            </div>
          )}
          {columns.map((col, index) => {
            const inColumn = cards.filter(c => c.chapter_id === col.id)
            const { connectors, ties } = columnConnectors(inColumn, rows, planner.lanes, m)
            const column = (
              <div key={col.id || 'later'} className={`pl-col${col.dim ? ' dim' : ''}`} style={{ width: m.colW, minWidth: m.colW }}>
                <div className="pl-col-head" style={{ height: m.headerH }}>
                  <div className="pl-kicker">{col.kicker}</div>
                  <div className={`pl-col-title${col.muted ? ' muted' : ''}`}>{col.title}</div>
                  <div className="pl-col-meta">{col.meta}</div>
                </div>
                {rows.map(r => (
                  <div
                    key={r.lane.id}
                    className="pl-cell"
                    style={{ height: r.height }}
                    onDragOver={allowDrop}
                    onDrop={e => { e.preventDefault(); if (drag) moveCard(drag, col.id, r.lane.id, null) }}
                    onDoubleClick={e => { if (e.target === e.currentTarget) addCard(col.id, r.lane.id) }}
                  >
                    {ties.filter(t => t.laneId === r.lane.id).map((t, i) => (
                      <span key={i} className="pl-tie" style={{ left: t.left, top: t.top, background: t.color }} />
                    ))}
                    {inColumn.filter(c => c.lines[0] === r.lane.id).map(c => (
                      <CardTile
                        key={c.id} card={c} lane={planner.lanes.find(l => l.id === c.lines[0])} codex={codex} selected={selected === c.id} compact={compact}
                        onSelect={() => select(c.id)}
                        onDragStart={e => { e.dataTransfer.effectAllowed = 'move'; setDrag(c.id) }}
                      />
                    ))}
                  </div>
                ))}
                {connectors.map((cn, i) => (
                  <div
                    key={i} className="pl-connector"
                    style={{ left: cn.left, top: cn.top, height: cn.height, width: cn.width, borderLeftColor: cn.color,
                      borderTop: cn.topCap ? `1.5px solid ${cn.color}` : 'none', borderBottom: cn.bottomCap ? `1.5px solid ${cn.color}` : 'none' }}
                  />
                ))}
              </div>
            )
            // The "Add Chapter" column sits between the last chapter and Later.
            if (index === columns.length - 1) {
              return [
                <div key="add" className="pl-add-col" style={{ width: m.colW, minWidth: m.colW, minHeight: gridH }} onClick={openChapterDialog} title="Add an empty chapter to the manuscript">
                  <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"><path d="M12 5v14M5 12h14" /></svg>
                  <span className="pl-add-col-label">Add Chapter</span>
                  {chapters.length === 0 && <span className="pl-add-col-hint">No chapters yet. Cards need a chapter to sit in.</span>}
                </div>,
                column,
              ]
            }
            return column
          })}
        </div>
      </div>
      {cards.length === 0 && chapters.length > 0 && (
        <div className="pl-empty" style={{ top: m.headerH }}>
          <div className="pl-empty-tag">Every line is empty. Add a card where a line meets a chapter, or import an outline.</div>
          <div className="pl-empty-actions">
            <button className="dialog-btn" onClick={() => addCard(chapters[0].id, MAIN_LANE_ID)}>New Card</button>
            <button className="dialog-btn primary" onClick={() => openImport()}>Import Outline…</button>
          </div>
          <div className="pl-empty-hint">Double-click any cell to add a card there</div>
        </div>
      )}
    </div>
  )
}

function BoardView() {
  const { selected, drag, boardBy, select, setDrag, moveCard, addCard, openLineDialog, openChapterDialog } = usePlannerStore(useShallow(s => ({
    selected: s.selected, drag: s.drag, boardBy: s.boardBy, select: s.select, setDrag: s.setDrag,
    moveCard: s.moveCard, addCard: s.addCard, openLineDialog: s.openLineDialog, openChapterDialog: s.openChapterDialog,
  })))
  const book = useBookStore(s => s.book)
  const planner = ensurePlanner(book)
  const chapters = bookChapters(book)
  const cards = displayCards(planner)
  const hidden = new Set(planner.hidden_lanes ?? [])
  const lanes = planner.lanes.filter(l => !hidden.has(l.id))
  const chapterOrder = new Map(chapters.map((c, i) => [c.id, i]))
  const orderKey = (c: DisplayCard) => (c.chapter_id === LATER_ID ? 1e9 : (chapterOrder.get(c.chapter_id) ?? 1e8))

  const boardColumns = boardBy === 'line'
    ? lanes.map(l => {
      const inLane = cards.filter(c => c.lines[0] === l.id).sort((a, b) => orderKey(a) - orderKey(b))
      const crossing = cards.filter(c => c.lines.includes(l.id) && c.lines[0] !== l.id).length
      return { key: l.id, kicker: l.kind, accent: l.color, title: l.name, muted: false, meta: `${inLane.length} ${inLane.length === 1 ? 'card' : 'cards'} · ${crossing} crossing`, cards: inLane,
        onDrop: () => { const d = cards.find(x => x.id === drag); if (drag) moveCard(drag, d?.chapter_id ?? (chapters[0]?.id ?? LATER_ID), l.id, null) },
        onNew: () => addCard(chapters[0]?.id ?? LATER_ID, l.id) }
    })
    : columnsFor(chapters, cards).map(col => ({
      key: col.id || 'later', kicker: col.kicker, accent: 'transparent', title: col.title, muted: col.muted, meta: col.meta,
      cards: cards.filter(c => c.chapter_id === col.id),
      onDrop: () => { if (drag) moveCard(drag, col.id, null, null) },
      onNew: () => addCard(col.id, MAIN_LANE_ID),
    }))

  return (
    <div className="pl-board">
      <div className="pl-board-row">
        {boardColumns.map(col => (
          <div key={col.key} className="pl-board-col" onDragOver={e => e.preventDefault()} onDrop={e => { e.preventDefault(); col.onDrop() }}>
            <div className="pl-board-head" style={{ borderTopColor: col.accent }}>
              <div className="pl-kicker">{col.kicker}</div>
              <div className={`pl-col-title${col.muted ? ' muted' : ''}`}>{col.title}</div>
              <div className="pl-col-meta">{col.meta}</div>
            </div>
            <div className="pl-board-cards">
              {col.cards.map(c => {
                const chips = boardBy === 'line'
                  ? [{ name: c.chapter_id === LATER_ID ? 'Later' : `Ch ${chapters.find(ch => ch.id === c.chapter_id)?.num ?? '?'}`, color: 'var(--text-secondary)' },
                    ...laneChips(c, planner.lanes).slice(1).map(l => ({ name: l.name, color: l.color }))]
                  : laneChips(c, planner.lanes).map(l => ({ name: l.name, color: l.color }))
                const color = STATUS_COLORS[c.st]
                return (
                  <div
                    key={c.id}
                    className={`pl-board-card${selected === c.id ? ' selected' : ''}`}
                    draggable
                    onDragStart={e => { e.dataTransfer.effectAllowed = 'move'; setDrag(c.id) }}
                    onDragOver={e => e.preventDefault()}
                    onDrop={e => {
                      e.preventDefault(); e.stopPropagation()
                      if (!drag || drag === c.id) return
                      if (boardBy === 'line') { const d = cards.find(x => x.id === drag); moveCard(drag, d?.chapter_id ?? c.chapter_id, c.lines[0], c.id) }
                      else moveCard(drag, c.chapter_id, null, c.id)
                    }}
                    onClick={e => { e.stopPropagation(); select(c.id) }}
                  >
                    <div className="pl-card-top">
                      <span className="pl-card-title">{c.title}</span>
                      <span className="pl-card-dot" title={c.st} style={{ background: color }} />
                    </div>
                    <div className="pl-card-syn">{c.synopsis || '—'}</div>
                    <div className="pl-board-card-foot">
                      {chips.map((chip, i) => <span key={i} className="pl-chip" title={chip.name} style={{ color: chip.color }}>{chip.name}</span>)}
                      <span style={{ flex: 1 }} />
                      <span className="pl-card-status" style={{ color }}>{c.st}</span>
                    </div>
                  </div>
                )
              })}
              <button className="pl-board-new" onClick={col.onNew}>+ Card</button>
            </div>
          </div>
        ))}
        {boardBy === 'chapter'
          ? <div className="pl-board-add" onClick={openChapterDialog}><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"><path d="M12 5v14M5 12h14" /></svg>Add Chapter</div>
          : <div className="pl-board-add" onClick={openLineDialog}>+ Story line</div>}
      </div>
    </div>
  )
}

function ScratchView() {
  const { noteId, noteMono, updateNote, newNote } = usePlannerStore(useShallow(s => ({ noteId: s.noteId, noteMono: s.noteMono, updateNote: s.updateNote, newNote: s.newNote })))
  const book = useBookStore(s => s.book)
  const planner = ensurePlanner(book)
  const note = planner.notes.find(n => n.id === noteId) ?? null
  if (!note) {
    return (
      <div className="pl-scratch">
        <div className="pl-scratch-empty">
          <div className="pl-empty-tag">A place for ideas you may or may not use.</div>
          <button className="dialog-btn" onClick={() => newNote()}>New Note</button>
        </div>
      </div>
    )
  }
  return (
    <div className="pl-scratch">
      <textarea
        className={`pl-scratch-text${noteMono ? ' mono' : ''}`}
        value={note.body}
        onChange={e => updateNote(note.id, { body: e.target.value })}
        placeholder="Write anything. Headings, lists, beat names, or loose prose. Nothing here touches the manuscript or the timeline until you propose cards."
        spellCheck={false}
      />
    </div>
  )
}

function SynopsisView() {
  const setSynopsis = usePlannerStore(s => s.setSynopsis)
  const book = useBookStore(s => s.book)
  const planner = ensurePlanner(book)
  const chapters = bookChapters(book)
  const cards = displayCards(planner)
  const refs = useRef<Record<string, HTMLTextAreaElement | null>>({})
  const grow = (el: HTMLTextAreaElement | null) => { if (!el) return; el.style.height = 'auto'; el.style.height = `${el.scrollHeight}px` }
  useEffect(() => { Object.values(refs.current).forEach(grow) })
  return (
    <div className="pl-synopsis" id="pl-synopsis-scroll">
      <div className="pl-synopsis-inner">
        {chapters.map(ch => {
          const text = synopsisText(planner, ch.id)
          const entries = synopsisEntries(planner, ch.id)
          const n = cards.filter(c => c.chapter_id === ch.id).length
          const edited = planner.synopsis?.[ch.id] !== undefined
          return (
            <div key={ch.id} className="pl-syn-row" id={`pl-syn-${ch.id}`}>
              <div className="pl-syn-head">
                <span className="pl-syn-num">Chapter {ch.num}</span>
                <span className="pl-syn-title">{ch.title || 'Untitled'}</span>
                <span className="pl-syn-meta">{n ? `${n} ${n === 1 ? 'card' : 'cards'}` : 'no cards'}{edited ? ' · edited' : ''}</span>
              </div>
              <textarea
                ref={el => { refs.current[ch.id] = el; grow(el) }}
                className="pl-syn-text"
                value={text}
                rows={2}
                onInput={e => grow(e.currentTarget)}
                onChange={e => setSynopsis(ch.id, e.target.value)}
                placeholder="No cards in this chapter."
                spellCheck={false}
              />
              {!edited && entries.length > 0 && (
                <div className="pl-synopsis-traces">
                  {entries.map((entry, i) => (
                    <span key={`${entry.cardId}-${i}`} className="pl-synopsis-trace" title={entry.text}>
                      {synopsisTrace(entry, chapters)}
                    </span>
                  ))}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

export default function PlannerView() {
  const { view, boardBy, noteId, noteMono, setBoardBy, addCard, openLineDialog, setCompact, updateNote, askDeleteNote, toggleNoteMono, resetSynopsis } = usePlannerStore(useShallow(s => ({
    view: s.view, boardBy: s.boardBy, noteId: s.noteId, noteMono: s.noteMono,
    setBoardBy: s.setBoardBy, addCard: s.addCard, openLineDialog: s.openLineDialog, setCompact: s.setCompact, updateNote: s.updateNote,
    askDeleteNote: s.askDeleteNote, toggleNoteMono: s.toggleNoteMono, resetSynopsis: s.resetSynopsis,
  })))
  const book = useBookStore(s => s.book)
  const planner = ensurePlanner(book)
  const chapters = bookChapters(book)
  const cards = displayCards(planner)
  const note = planner.notes.find(n => n.id === noteId) ?? null
  const isCards = view === 'timeline' || view === 'board'
  const titles = { timeline: 'Timeline', board: 'Board', scratch: 'Scratchpad', synopsis: 'Synopsis' }

  // One derivation for the toolbar counts, so the toolbar, the overview and
  // the sidebar cannot disagree.
  const counts = cards.reduce<Record<string, number>>((acc, c) => ({ ...acc, [c.st]: (acc[c.st] ?? 0) + 1 }), {})
  const statusSummary = (['planned', 'drafted'] as const)
    .filter(k => counts[k]).map(k => `${counts[k]} ${k}`).join(' · ')
  const synWords = chapters.reduce((a, ch) => a + countWords(synopsisText(planner, ch.id)), 0)
  const synChapters = chapters.filter(ch => generatedSynopsis(planner, ch.id)).length

  return (
    <div className="pl-center">
      <div className="pl-toolbar">
        <span className="pl-tool-title">{titles[view]}</span>
        {view === 'board' && (<>
          <span className="pl-tool-sep" />
          <span className="pl-tool-label">Group by</span>
          <button className={`toolbar-btn${boardBy === 'chapter' ? ' active' : ''}`} onClick={() => setBoardBy('chapter')}>Chapters</button>
          <button className={`toolbar-btn${boardBy === 'line' ? ' active' : ''}`} onClick={() => setBoardBy('line')}>Story Lines</button>
        </>)}
        {isCards && (<>
          <span className="pl-tool-sep" />
          <button className="toolbar-btn" title="New card on the main line" onClick={() => addCard(chapters[0]?.id ?? LATER_ID, MAIN_LANE_ID)}>+ New Card</button>
          <button className="toolbar-btn" title="New story line" onClick={openLineDialog}>+ New Line</button>
          <span className="pl-tool-spacer" />
          <span className="pl-tool-status">{statusSummary}</span>
          <label className="settings-toggle">
            <input type="checkbox" checked={planner.compact !== false} onChange={e => setCompact(e.target.checked)} />
            <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
            <span className="settings-toggle-label">Compact</span>
          </label>
        </>)}
        {view === 'scratch' && note && (<>
          <span className="pl-tool-sep" />
          <input
            className="pl-note-title-input" value={note.title} placeholder="Untitled note" readOnly={!!note.system}
            onChange={e => { if (!note.system) updateNote(note.id, { title: e.target.value }) }}
          />
          <span className="pl-tool-status">{countWords(note.body)} words</span>
          <span className="pl-tool-sep" />
          <button className={`toolbar-btn${noteMono ? ' active' : ''}`} onClick={toggleNoteMono} title="Monospace">Mono</button>
          {!note.system && <button className="toolbar-btn" onClick={() => askDeleteNote(note.id)} title="Delete note">Delete</button>}
        </>)}
        {view === 'synopsis' && (<>
          <span className="pl-tool-sep" />
          <button className="toolbar-btn" onClick={resetSynopsis} title="Rebuild every paragraph from the cards, discarding edits">Rebuild from Cards</button>
          <span className="pl-tool-spacer" />
          <span className="pl-tool-status">{synWords.toLocaleString()} words · {synChapters} of {chapters.length} chapters</span>
        </>)}
      </div>
      {view === 'timeline' && <TimelineView />}
      {view === 'board' && <BoardView />}
      {view === 'scratch' && <ScratchView />}
      {view === 'synopsis' && <SynopsisView />}
      <PlannerDialogs />
    </div>
  )
}

