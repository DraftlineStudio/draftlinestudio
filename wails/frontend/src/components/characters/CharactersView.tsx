// CharactersView — the character codex: a swimlane chapter-presence grid
// (one row per character, one column per chapter) with a detail pane on the
// right. Replaces the old force-graph Cast workspace; scales to hundreds of
// characters where the graph became an unreadable hairball.
// All detection is local pattern-matching — no AI, nothing leaves the machine.

import { useMemo, useState, useEffect } from 'react'
import { useBookStore } from '../../store/bookStore'
import { useRelationshipStore } from '../../store/relationshipStore'
import { characterColor, characterInitials } from '../../utils/characterVisuals'
import { hexToRgba } from '../../utils/accentColor'
import type { BookData, Character, CharacterRole, CharacterEvent, MentionRecord, Section } from '../../types/draftline'
import './characters.css'

function allChapters(book: BookData) {
  return [...(book.front_matter || []), ...(book.body || []), ...(book.back_matter || [])]
}

function chapterName(book: BookData, index: number): string {
  return allChapters(book)[index]?.title || `Chapter ${index + 1}`
}

// Discrete alpha ramp for grid cells: reads as "none / few / some / many / lots".
function cellAlpha(v: number): number {
  return v <= 0 ? 0 : v <= 2 ? 0.3 : v <= 4 ? 0.55 : v <= 6 ? 0.78 : 0.95
}

// Maps a combined-chapter index (front_matter + body + back_matter) back to a
// navigable (section, index) pair.
function chapterLocation(book: BookData, index: number): { section: Section; index: number } {
  const front = book.front_matter?.length || 0
  const body = book.body?.length || 0
  if (index < front) return { section: 'front_matter', index }
  if (index < front + body) return { section: 'body', index: index - front }
  return { section: 'back_matter', index: index - front - body }
}

function stripHtml(html: string): string {
  const div = document.createElement('div')
  div.innerHTML = html
  return div.textContent || ''
}

// Best-effort context around a mention: locate the mention text near its
// recorded offset in the stripped chapter text and slice to word boundaries.
// Offsets come from the Go-side text stripper, so treat them as a hint, not
// an exact position; fall back to the bare mention text when unlocatable.
function mentionExcerpt(book: BookData, m: MentionRecord): string {
  const html = allChapters(book)[m.chapter]?.content
  if (!html || !m.text) return m.text || ''
  const text = stripHtml(html)
  let idx = text.indexOf(m.text, Math.max(0, (m.char_offset ?? 0) - 300))
  if (idx === -1) idx = text.indexOf(m.text)
  if (idx === -1) return m.text
  let start = Math.max(0, idx - 90)
  let end = Math.min(text.length, idx + m.text.length + 90)
  if (start > 0) {
    const sp = text.indexOf(' ', start)
    if (sp !== -1 && sp < idx) start = sp + 1
  }
  if (end < text.length) {
    const sp = text.lastIndexOf(' ', end)
    if (sp > idx + m.text.length) end = sp
  }
  return `${start > 0 ? '…' : ''}${text.slice(start, end).trim()}${end < text.length ? '…' : ''}`
}

const EMPTY_CELL = 'rgba(255,255,255,0.03)'

type SortMode = 'first' | 'mentions-desc' | 'mentions-asc'
type ViewMode = 'grid' | 'heat'

export default function CharactersView() {
  const {
    book, setViewMode, setCurrentChapter, indexBook, isIndexing, updateBook,
    addCharacter, updateCharacter, deleteCharacter, clearAllCharacters,
    mergeEntities, splitEntity,
  } = useBookStore()
  const { isAnalyzing, analyzeRelationships } = useRelationshipStore()

  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [query, setQuery] = useState('')
  const [sortMode, setSortMode] = useState<SortMode>('first')
  const [laneView, setLaneView] = useState<ViewMode>('grid')
  const [mergeFrom, setMergeFrom] = useState<string | null>(null)
  const [mergeWith, setMergeWith] = useState<string | null>(null)
  const [editing, setEditing] = useState(false)
  const [adding, setAdding] = useState(false)
  const [splitting, setSplitting] = useState(false)
  const [confirmClear, setConfirmClear] = useState(false)

  const busy = isIndexing || isAnalyzing

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setViewMode('editor')
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [setViewMode])

  const characters = book?.story_bible?.characters ?? []
  const relationships = book?.analysis?.relationships?.relationships ?? []
  const events = book?.analysis?.relationships?.events ?? []
  const entityIds = useMemo(
    () => new Set((book?.analysis?.entity_resolution?.entities ?? []).map(e => e.id)),
    [book],
  )

  const charMap = useMemo(() => {
    const m = new Map<string, Character>()
    for (const c of characters) m.set(c.id, c)
    return m
  }, [characters])

  const sorted = useMemo(() => {
    const q = query.trim().toLowerCase()
    const list = q
      ? characters.filter(c =>
          c.name.toLowerCase().includes(q) ||
          (c.aliases || []).some(a => a.toLowerCase().includes(q)))
      : [...characters]
    const byMentions = (a: Character, b: Character) => (b.mention_count || 0) - (a.mention_count || 0)
    if (sortMode === 'first') {
      list.sort((a, b) =>
        (a.first_chapter ?? Number.MAX_SAFE_INTEGER) - (b.first_chapter ?? Number.MAX_SAFE_INTEGER) ||
        byMentions(a, b))
    } else if (sortMode === 'mentions-asc') {
      list.sort((a, b) => -byMentions(a, b))
    } else {
      list.sort(byMentions)
    }
    return list
  }, [characters, query, sortMode])

  // The pane always shows someone when characters exist (matches the design).
  useEffect(() => {
    if (!selectedId && sorted.length > 0) setSelectedId(sorted[0].id)
    else if (selectedId && !charMap.has(selectedId)) setSelectedId(sorted[0]?.id ?? null)
  }, [selectedId, sorted, charMap])

  const selected = selectedId ? charMap.get(selectedId) ?? null : null
  const chapters = book ? allChapters(book) : []
  const chapterCount = chapters.length

  // Detect characters and weave relationships in one action.
  const detect = async () => {
    if (!book) return
    await indexBook()
    const fresh = useBookStore.getState().book
    if (!fresh?.analysis?.entity_resolution?.entities?.length) return
    const updated = await analyzeRelationships(fresh)
    if (updated) updateBook(updated)
  }

  const pickRow = (id: string) => {
    if (mergeFrom) {
      if (id !== mergeFrom && entityIds.has(id)) setMergeWith(id)
      return
    }
    setSelectedId(id)
    setEditing(false)
    setSplitting(false)
  }

  const jumpToChapter = (chapterIndex: number) => {
    if (!book) return
    const loc = chapterLocation(book, chapterIndex)
    setCurrentChapter(loc.section, loc.index)
    setViewMode('editor')
  }

  const doMerge = async (keepId: string) => {
    if (!mergeFrom || !mergeWith) return
    const otherId = keepId === mergeFrom ? mergeWith : mergeFrom
    const ok = await mergeEntities([keepId, otherId], '')
    if (ok) {
      setSelectedId(keepId)
      setMergeFrom(null)
      setMergeWith(null)
    }
  }

  if (!book) return null

  const gridCols = `repeat(${chapterCount}, 18px)`
  const heatCols = `repeat(${chapterCount}, 14px)`
  const heatWidth = chapterCount * 14

  return (
    <div className="chars-view">
      <header className="chars-header">
        <span className="chars-heading">Characters</span>
        <span className="chars-count">
          {characters.length} characters{relationships.length ? ` · ${relationships.length} relationships` : ''}
        </span>
        <div className="chars-header-actions">
          <button className="tool-card-btn" onClick={detect} disabled={busy}>
            {busy ? 'Working…' : characters.length ? 'Re-Detect' : 'Detect Characters'}
          </button>
          <button className="tool-card-btn secondary" onClick={() => setViewMode('editor')} title="Back to writing (Esc)">
            Close
          </button>
        </div>
      </header>

      <div className="chars-body">
        <div className="chars-main">
          <div className="chars-toolbar">
            <input
              className="dialog-input"
              placeholder="Filter characters…"
              value={query}
              onChange={e => setQuery(e.target.value)}
            />
            <select className="dialog-select" value={laneView} onChange={e => setLaneView(e.target.value as ViewMode)}>
              <option value="grid">Grid view</option>
              <option value="heat">Heatmap view</option>
            </select>
            {mergeFrom ? (
              <div className="chars-merge-banner">
                Click who <strong>{charMap.get(mergeFrom)?.name}</strong> really is…
                <button className="ai-link-btn" onClick={() => { setMergeFrom(null); setMergeWith(null) }}>cancel</button>
              </div>
            ) : (
              <>
                <button className="ai-link-btn" onClick={() => { setAdding(true); setEditing(false); setSplitting(false) }}>+ Add character</button>
                {characters.length > 0 && (confirmClear ? (
                  <>
                    <button className="ai-link-btn" style={{ color: '#E06C75' }} onClick={() => { clearAllCharacters(); setConfirmClear(false); setSelectedId(null) }}>
                      Clear {characters.length}?
                    </button>
                    <button className="ai-link-btn" onClick={() => setConfirmClear(false)}>keep</button>
                  </>
                ) : (
                  <button className="ai-link-btn" onClick={() => setConfirmClear(true)} title="Remove all characters for a fresh detection">
                    Clear all
                  </button>
                ))}
              </>
            )}
            <span style={{ flex: 1 }} />
            <span className="chars-sort-label">SORTED BY</span>
            <select className="dialog-select" value={sortMode} onChange={e => setSortMode(e.target.value as SortMode)}>
              <option value="first">First appearance</option>
              <option value="mentions-desc">Mentions (descending)</option>
              <option value="mentions-asc">Mentions (ascending)</option>
            </select>
          </div>

          {sorted.length === 0 ? (
            <div className="chars-empty">
              <p>
                {characters.length === 0
                  ? 'Detect your characters to see who appears where across the book.'
                  : 'No matches.'}
              </p>
              {characters.length === 0 && (
                <>
                  <button className="tool-card-btn" onClick={detect} disabled={busy}>
                    {busy ? 'Working…' : 'Detect Characters'}
                  </button>
                  <p className="chars-empty-note">Detection runs entirely on your machine — no AI, no network.</p>
                </>
              )}
            </div>
          ) : (
            <div className="chars-lanes">
              <div className="chars-lane-head">
                <div className="chars-lane-sticky">
                  <div className="chars-col-label" style={{ width: 200 }}>CHARACTER</div>
                  {laneView === 'heat' && <div className="chars-col-label" style={{ width: 92 }}>ROLE</div>}
                </div>
                {laneView === 'grid' ? (
                  <div className="chars-cells" style={{ gridTemplateColumns: gridCols }}>
                    {chapters.map((_, i) => (
                      <div key={i} className="chars-ch-label">{i + 1}</div>
                    ))}
                  </div>
                ) : (
                  <div className="chars-heat-scale" style={{ width: heatWidth }}>
                    <span>CH 1</span><span>CH {chapterCount}</span>
                  </div>
                )}
              </div>
              {sorted.map(c => {
                const color = characterColor(c.name)
                const on = c.id === selectedId
                return (
                  <div
                    key={c.id}
                    className={`chars-row${on ? ' selected' : ''}${mergeFrom === c.id ? ' merge-source' : ''}`}
                    onClick={() => pickRow(c.id)}
                  >
                    <div className="chars-lane-sticky">
                      <div className="chars-name-cell">
                        <span className="chars-dot" style={{ background: color }} />
                        <span className="chars-name">{c.name}</span>
                        <span className="chars-mentions">{c.mention_count || ''}</span>
                      </div>
                      {laneView === 'heat' && (
                        <select
                          className="chars-role-select"
                          value={c.role || 'minor'}
                          onClick={e => e.stopPropagation()}
                          onChange={e => updateCharacter({ ...c, role: e.target.value as CharacterRole })}
                        >
                          <option value="protagonist">Protagonist</option>
                          <option value="antagonist">Antagonist</option>
                          <option value="supporting">Supporting</option>
                          <option value="minor">Minor</option>
                          <option value="other">Other</option>
                        </select>
                      )}
                    </div>
                    {laneView === 'grid' ? (
                      <div className="chars-cells" style={{ gridTemplateColumns: gridCols }}>
                        {chapters.map((_, i) => {
                          const v = c.chapter_mentions?.[i] || 0
                          return (
                            <div
                              key={i}
                              className="chars-cell"
                              title={`${chapterName(book, i)} · ${v} ${v === 1 ? 'mention' : 'mentions'}`}
                              style={{ background: v > 0 ? hexToRgba(color, cellAlpha(v)) : (on ? 'rgba(255,255,255,0.07)' : EMPTY_CELL) }}
                            />
                          )
                        })}
                      </div>
                    ) : (
                      <div className="chars-heat" style={{ gridTemplateColumns: heatCols, width: heatWidth }}>
                        {chapters.map((_, i) => {
                          const v = c.chapter_mentions?.[i] || 0
                          return (
                            <div
                              key={i}
                              title={`${chapterName(book, i)} · ${v} ${v === 1 ? 'mention' : 'mentions'}`}
                              style={{ background: v > 0 ? hexToRgba(color, Math.min(0.92, 0.15 + v * 0.09)) : EMPTY_CELL }}
                            />
                          )
                        })}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {(selected || adding) && (
          <aside className="chars-pane">
            {adding ? (
              <CharacterForm
                char={null}
                onSave={c => { addCharacter(c); setAdding(false); setSelectedId(c.id) }}
                onCancel={() => setAdding(false)}
              />
            ) : editing && selected ? (
              <CharacterForm
                char={selected}
                onSave={c => { updateCharacter(c); setEditing(false) }}
                onCancel={() => setEditing(false)}
              />
            ) : splitting && selected ? (
              <SplitPanel
                book={book}
                char={selected}
                onDone={() => setSplitting(false)}
                onSplit={splitEntity}
              />
            ) : mergeWith && mergeFrom ? (
              <MergeConfirm
                a={charMap.get(mergeFrom)!}
                b={charMap.get(mergeWith)!}
                onKeep={doMerge}
                onCancel={() => setMergeWith(null)}
              />
            ) : selected ? (
              <DetailPane
                book={book}
                char={selected}
                charMap={charMap}
                relationships={relationships}
                events={events}
                entityBacked={entityIds.has(selected.id)}
                onSelect={setSelectedId}
                onEdit={() => setEditing(true)}
                onSplit={() => setSplitting(true)}
                onMerge={() => { setMergeFrom(selected.id); setMergeWith(null) }}
                onDelete={() => { deleteCharacter(selected.id); setSelectedId(null) }}
                onJump={jumpToChapter}
              />
            ) : null}
          </aside>
        )}
      </div>
    </div>
  )
}

// ── Detail pane ──────────────────────────────────────────────────────────────

function DetailPane({ book, char, charMap, relationships, events, entityBacked, onSelect, onEdit, onSplit, onMerge, onDelete, onJump }: {
  book: BookData
  char: Character
  charMap: Map<string, Character>
  relationships: { id: string; character1_id: string; character2_id: string; strength: number; interaction_count: number }[]
  events: CharacterEvent[]
  entityBacked: boolean
  onSelect: (id: string) => void
  onEdit: () => void
  onSplit: () => void
  onMerge: () => void
  onDelete: () => void
  onJump: (chapterIndex: number) => void
}) {
  const color = characterColor(char.name)

  const bonds = relationships
    .filter(r => r.character1_id === char.id || r.character2_id === char.id)
    .map(r => ({ otherId: r.character1_id === char.id ? r.character2_id : r.character1_id, rel: r }))
    .filter(b => charMap.has(b.otherId))
    .sort((a, b) => b.rel.strength - a.rel.strength)

  // Fixed 60° spokes: no layout pass, no simulation — positions never shift.
  const spokes = bonds.slice(0, 6).map(({ otherId, rel }, k) => {
    const other = charMap.get(otherId)!
    const angle = -90 + k * 60
    const rad = (angle * Math.PI) / 180
    const cx = 136, cy = 100, r = 74
    return {
      otherId,
      angle,
      left: Math.max(2, Math.min(198, cx + Math.cos(rad) * r - 36)),
      top: cy + Math.sin(rad) * r - 13,
      lineWidth: Math.max(1, Math.min(3, 1 + rel.strength * 2)),
      color: characterColor(other.name),
      initials: characterInitials(other.name),
      name: other.name,
    }
  })

  const moments = events
    .filter(e => e.character_ids?.includes(char.id))
    .sort((a, b) => a.chapter_index - b.chapter_index)

  const chapters = allChapters(book)
  const chapterCount = char.chapter_mentions ? Object.keys(char.chapter_mentions).length : 0

  // Up to three mention excerpts spread across the character's arc.
  const entity = entityBacked ? book.analysis?.entity_resolution?.entities?.find(e => e.id === char.id) : undefined
  const mentionRecords = entity
    ? (book.analysis?.entity_resolution?.mentions ?? [])
        .filter(m => entity.mention_ids.includes(m.id))
        .sort((a, b) => a.chapter - b.chapter || a.char_offset - b.char_offset)
    : []
  const shownMentions = mentionRecords.length <= 3
    ? mentionRecords
    : [mentionRecords[0], mentionRecords[Math.floor(mentionRecords.length / 2)], mentionRecords[mentionRecords.length - 1]]

  return (
    <>
      <div className="chars-pane-top">
        <span className="chars-portrait" style={{ color, background: hexToRgba(color, 0.12) }}>
          {characterInitials(char.name)}
        </span>
        <div style={{ minWidth: 0 }}>
          <div className="chars-pane-name">{char.name}</div>
          <div className="chars-pane-role">{char.role || 'minor'}</div>
        </div>
      </div>

      {char.aliases && char.aliases.length > 0 && (
        <div className="chars-pane-meta">aka {char.aliases.slice(0, 5).join(', ')}</div>
      )}

      <div className="chars-pane-meta">
        {char.first_chapter !== undefined && <>First appears: <strong>{chapterName(book, char.first_chapter)}</strong></>}
        {chapterCount > 0 && <> · {chapterCount} chapter{chapterCount === 1 ? '' : 's'}</>}
        {char.mention_count ? <> · {char.mention_count} mentions</> : null}
      </div>

      {char.description && <div className="chars-pane-meta">{char.description}</div>}

      {spokes.length > 0 && (
        <>
          <div className="chars-section-label">Strongest ties</div>
          <div className="chars-ego">
            {spokes.map(s => (
              <div
                key={`line-${s.otherId}`}
                className="chars-ego-spoke"
                style={{ width: 74, height: s.lineWidth, background: hexToRgba(s.color, 0.55), transform: `rotate(${s.angle}deg)` }}
              />
            ))}
            {spokes.map(s => (
              <button
                key={s.otherId}
                className="chars-ego-node"
                style={{ left: s.left, top: s.top }}
                title={s.name}
                onClick={() => onSelect(s.otherId)}
              >
                <span className="chars-ego-orb" style={{ color: s.color }}>{s.initials}</span>
                <span className="chars-ego-name">{s.name}</span>
              </button>
            ))}
            <div className="chars-ego-center" style={{ color }}>{characterInitials(char.name)}</div>
          </div>
        </>
      )}

      {chapters.length > 0 && (
        <>
          <div className="chars-section-label">Appears in</div>
          <div>
            <div className="chars-strip" style={{ gridTemplateColumns: `repeat(${chapters.length}, 1fr)` }}>
              {chapters.map((_, i) => {
                const v = char.chapter_mentions?.[i] || 0
                return <div key={i} style={{ background: v > 0 ? hexToRgba(color, cellAlpha(v)) : 'rgba(255,255,255,0.05)' }} />
              })}
            </div>
            <div className="chars-strip-labels"><span>Ch 1</span><span>Ch {chapters.length}</span></div>
          </div>
        </>
      )}

      {moments.length > 0 && (
        <>
          <div className="chars-section-label">Key events</div>
          <div>
            {moments.map(m => (
              <div key={m.id} className="chars-event">
                <span className="chars-event-dot" style={{ background: color }} />
                <span className="chars-event-text">
                  {m.description}<span className="muted"> · {chapterName(book, m.chapter_index)}</span>
                </span>
              </div>
            ))}
          </div>
        </>
      )}

      {shownMentions.length > 0 && (
        <>
          <div className="chars-section-label">Mentions</div>
          <div>
            {shownMentions.map(m => (
              <div key={m.id} className="chars-mention">
                <div className="chars-mention-text">{mentionExcerpt(book, m)}</div>
                <div className="chars-mention-meta">
                  <span className="muted">{chapterName(book, m.chapter)}</span>
                  <button className="chars-jump" onClick={() => onJump(m.chapter)}>Jump to chapter →</button>
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      <div className="chars-pane-actions">
        <button className="tool-card-btn secondary" onClick={onEdit}>Edit</button>
        {entityBacked && (char.mention_count ?? 0) >= 2 && (
          <button className="tool-card-btn secondary" onClick={onSplit} title="Some mentions belong to a different person">Split…</button>
        )}
        {entityBacked && (
          <button className="tool-card-btn secondary" onClick={onMerge} title="This character is the same person as another">Same as…</button>
        )}
        <button className="tool-card-btn secondary" style={{ color: '#E06C75' }} onClick={onDelete}>Delete</button>
      </div>
    </>
  )
}

// ── Merge confirmation ───────────────────────────────────────────────────────

function MergeConfirm({ a, b, onKeep, onCancel }: {
  a: Character
  b: Character
  onKeep: (keepId: string) => void
  onCancel: () => void
}) {
  return (
    <div>
      <div className="chars-section-label" style={{ marginBottom: 8 }}>Same person — keep which name?</div>
      <p className="chars-pane-meta" style={{ marginBottom: 10 }}>The other name becomes an alias. Remembered on every re-detect.</p>
      <button className="tool-card-btn" style={{ display: 'block', width: '100%', marginBottom: 6 }} onClick={() => onKeep(a.id)}>{a.name}</button>
      <button className="tool-card-btn" style={{ display: 'block', width: '100%', marginBottom: 6 }} onClick={() => onKeep(b.id)}>{b.name}</button>
      <button className="ai-link-btn" onClick={onCancel}>Cancel</button>
    </div>
  )
}

// ── Split panel ──────────────────────────────────────────────────────────────

function SplitPanel({ book, char, onDone, onSplit }: {
  book: BookData
  char: Character
  onDone: () => void
  onSplit: (entityId: string, mentionIds: string[], newCanonical: string) => Promise<boolean>
}) {
  const [checked, setChecked] = useState<Set<string>>(new Set())
  const [newName, setNewName] = useState('')
  const [busy, setBusy] = useState(false)

  const entity = book.analysis?.entity_resolution?.entities?.find(e => e.id === char.id)
  const mentions = entity
    ? (book.analysis?.entity_resolution?.mentions ?? []).filter(m => entity.mention_ids.includes(m.id))
    : []

  const toggle = (id: string) => {
    setChecked(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const canSplit = checked.size > 0 && checked.size < mentions.length

  return (
    <div>
      <div className="chars-section-label" style={{ marginBottom: 8 }}>Split "{char.name}"</div>
      <p className="chars-pane-meta" style={{ marginBottom: 10 }}>Check the mentions that belong to a different person.</p>
      <div className="split-mentions">
        {mentions.map(m => (
          <label key={m.id} className="split-mention">
            <input type="checkbox" checked={checked.has(m.id)} onChange={() => toggle(m.id)} />
            <span>{m.text}</span>
            <span className="muted">{chapterName(book, m.chapter)}</span>
          </label>
        ))}
      </div>
      <input
        className="dialog-input"
        style={{ marginBottom: 8 }}
        value={newName}
        onChange={e => setNewName(e.target.value)}
        placeholder="Name for the new character (optional)"
      />
      <div style={{ display: 'flex', gap: 8 }}>
        <button
          className="tool-card-btn"
          disabled={!canSplit || busy}
          onClick={async () => {
            setBusy(true)
            const ok = await onSplit(char.id, Array.from(checked), newName.trim())
            setBusy(false)
            if (ok) onDone()
          }}
        >
          {busy ? 'Splitting…' : `Split off ${checked.size}`}
        </button>
        <button className="ai-link-btn" onClick={onDone}>Cancel</button>
      </div>
    </div>
  )
}

// ── Character form ───────────────────────────────────────────────────────────

function CharacterForm({ char, onSave, onCancel }: {
  char: Character | null
  onSave: (c: Character) => void
  onCancel: () => void
}) {
  const [name, setName] = useState(char?.name ?? '')
  const [role, setRole] = useState<CharacterRole | string>(char?.role ?? 'supporting')
  const [desc, setDesc] = useState(char?.description ?? '')
  const [notes, setNotes] = useState(char?.notes ?? '')

  return (
    <div>
      <div className="chars-section-label" style={{ marginBottom: 8 }}>{char ? 'Edit character' : 'New character'}</div>
      <div className="dialog-field">
        <label className="dialog-label">Name</label>
        <input className="dialog-input" value={name} onChange={e => setName(e.target.value)} autoFocus />
      </div>
      <div className="dialog-field">
        <label className="dialog-label">Role</label>
        <select className="dialog-select" value={role} onChange={e => setRole(e.target.value as CharacterRole)}>
          <option value="protagonist">Protagonist</option>
          <option value="antagonist">Antagonist</option>
          <option value="supporting">Supporting</option>
          <option value="minor">Minor</option>
          <option value="other">Other</option>
        </select>
      </div>
      <div className="dialog-field">
        <label className="dialog-label">Description</label>
        <textarea className="dialog-input" rows={3} value={desc} onChange={e => setDesc(e.target.value)} style={{ resize: 'vertical' }} />
      </div>
      <div className="dialog-field">
        <label className="dialog-label">Notes</label>
        <textarea className="dialog-input" rows={3} value={notes} onChange={e => setNotes(e.target.value)} style={{ resize: 'vertical' }} />
      </div>
      <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
        <button
          className="tool-card-btn"
          disabled={!name.trim()}
          onClick={() => onSave({ ...(char ?? { id: `char-manual-${Date.now()}` }), name: name.trim(), role, description: desc, notes } as Character)}
        >
          {char ? 'Save' : 'Add'}
        </button>
        <button className="ai-link-btn" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}
