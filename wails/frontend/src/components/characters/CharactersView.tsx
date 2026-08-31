// CharactersView — the character codex: a swimlane chapter-presence grid
// (one row per character, one column per chapter) with a detail pane on the
// right. Replaces the old force-graph Cast workspace; scales to hundreds of
// characters where the graph became an unreadable hairball.
// All detection is local pattern-matching — no AI, nothing leaves the machine.

import { useMemo, useState, useEffect } from 'react'
import { useBookStore } from '../../store/bookStore'
import { useAppStore } from '../../store/appStore'
import { useRelationshipStore } from '../../store/relationshipStore'
import { characterColor, characterInitials } from '../../utils/characterVisuals'
import { hexToRgba } from '../../utils/accentColor'
import { confirmedCharacterIds, confirmedEvents, confirmedRelationships, isConfirmedCharacter } from '../../utils/characterStatus'
import type { BookData, Character, CharacterRole, CharacterEvent, MentionRecord } from '../../types/draftline'
import './characters.css'

import { allChapters, chapterName, cellAlpha, chapterLocation, takeCharacterFocus } from './shared'

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
type StatusFilter = 'accepted' | 'review' | 'all'

export default function CharactersView() {
  const {
    book, setViewMode, setCurrentChapter, indexBook, isIndexing, updateBook,
    addCharacter, updateCharacter, deleteCharacter, clearAllCharacters,
    mergeEntities, splitEntity,
  } = useBookStore()
  const { isAnalyzing, analyzeRelationships } = useRelationshipStore()
  const { settings, saveSettings } = useAppStore()

  const savedLane: ViewMode = settings.characters_lane_view === 'heat' ? 'heat' : 'grid'
  // A pending focus (sidebar's "Open in Characters") wins over auto-select.
  const [selectedId, setSelectedId] = useState<string | null>(() => takeCharacterFocus())
  const [query, setQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('accepted')
  const [sortMode, setSortMode] = useState<SortMode>(savedLane === 'heat' ? 'mentions-desc' : 'first')
  const [laneView, setLaneView] = useState<ViewMode>(savedLane)
  const [mergeFrom, setMergeFrom] = useState<string | null>(null)
  const [editing, setEditing] = useState(false)
  const [adding, setAdding] = useState(false)
  const [splitting, setSplitting] = useState(false)
  const [confirmClear, setConfirmClear] = useState(false)

  const busy = isIndexing || isAnalyzing

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      if (mergeFrom) setMergeFrom(null)
      else setViewMode('editor')
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [mergeFrom, setViewMode])

  const characters = book?.story_bible?.characters ?? []
  const reviewCount = characters.filter(c => c.detection_status === 'review').length
  const characterIdsForSidebar = useMemo(() => confirmedCharacterIds(characters), [characters])
  const relationships = useMemo(
    () => confirmedRelationships(book?.analysis?.relationships?.relationships ?? [], characterIdsForSidebar),
    [book, characterIdsForSidebar],
  )
  const events = useMemo(
    () => confirmedEvents(book?.analysis?.relationships?.events ?? [], characterIdsForSidebar),
    [book, characterIdsForSidebar],
  )
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
    const byStatus = characters.filter(c =>
      statusFilter === 'all' ||
      (statusFilter === 'review' ? c.detection_status === 'review' : isConfirmedCharacter(c)))
    const list = q
      ? byStatus.filter(c =>
          c.name.toLowerCase().includes(q) ||
          (c.aliases || []).some(a => a.toLowerCase().includes(q)))
      : [...byStatus]
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
  }, [characters, query, sortMode, statusFilter])

  // The pane always shows someone when characters exist (matches the design).
  useEffect(() => {
    if (!selectedId && sorted.length > 0) setSelectedId(sorted[0].id)
    else if (selectedId && !sorted.some(c => c.id === selectedId)) setSelectedId(sorted[0]?.id ?? null)
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

  const doMerge = async (targetId: string): Promise<boolean> => {
    if (!mergeFrom || targetId === mergeFrom) return false
    const ok = await mergeEntities([targetId, mergeFrom], '')
    if (ok) {
      setSelectedId(targetId)
      setMergeFrom(null)
    }
    return ok
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
          {characters.length - reviewCount} characters{reviewCount ? ` · ${reviewCount} need review` : ''}{relationships.length ? ` · ${relationships.length} relationships` : ''}
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
            <select
              className="dialog-select"
              value={laneView}
              onChange={e => {
                const v = e.target.value as ViewMode
                setLaneView(v)
                void saveSettings({ characters_lane_view: v })
                // Heatmap reads as a ranked order — put the biggest presences on top.
                if (v === 'heat') setSortMode('mentions-desc')
              }}
            >
              <option value="grid">Grid view</option>
              <option value="heat">Heatmap view</option>
            </select>
            <select className="dialog-select" value={statusFilter} onChange={e => setStatusFilter(e.target.value as StatusFilter)}>
              <option value="accepted">Characters</option>
              <option value="review">Needs review ({reviewCount})</option>
              <option value="all">All candidates</option>
            </select>
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
                    className={`chars-row${on ? ' selected' : ''}`}
                    onClick={() => pickRow(c.id)}
                  >
                    <div className="chars-lane-sticky">
                      <div className="chars-name-cell">
                        <span className="chars-dot" style={{ background: color }} />
                        <span className="chars-name">{c.name}</span>
                        {c.detection_status === 'review' && <span className="chars-review-badge" title="Detection is uncertain; verify this story entity">REVIEW</span>}
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
                onConfirm={() => updateCharacter({ ...selected, detection_status: 'accepted' })}
                onSplit={() => setSplitting(true)}
                onMerge={() => setMergeFrom(selected.id)}
                onDelete={() => { deleteCharacter(selected.id); setSelectedId(null) }}
                onJump={jumpToChapter}
              />
            ) : null}
          </aside>
        )}
      </div>

      {mergeFrom && charMap.has(mergeFrom) && (
        <MergeCharacterModal
          source={charMap.get(mergeFrom)!}
          candidates={characters.filter(character => character.id !== mergeFrom && entityIds.has(character.id))}
          onMerge={doMerge}
          onClose={() => setMergeFrom(null)}
        />
      )}
    </div>
  )
}

// ── Detail pane ──────────────────────────────────────────────────────────────

function DetailPane({ book, char, charMap, relationships, events, entityBacked, onSelect, onEdit, onConfirm, onSplit, onMerge, onDelete, onJump }: {
  book: BookData
  char: Character
  charMap: Map<string, Character>
  relationships: { id: string; character1_id: string; character2_id: string; strength: number; interaction_count: number }[]
  events: CharacterEvent[]
  entityBacked: boolean
  onSelect: (id: string) => void
  onEdit: () => void
  onConfirm: () => void
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

      {char.detection_status === 'review' && (
        <div className="chars-review-callout">
          <span>Draftline is not certain this is a participating character.</span>
          <button className="ai-link-btn" onClick={onConfirm}>Confirm character</button>
        </div>
      )}

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
        <button
          className="tool-card-btn secondary"
          style={{ color: '#E06C75' }}
          onClick={onDelete}
          title={char.is_auto_detected ? 'Hide this false detection and remember the decision on future re-detects' : 'Delete this character'}
        >
          {char.is_auto_detected ? 'Not a character' : 'Delete'}
        </button>
      </div>
    </>
  )
}

// ── Character merge modal ────────────────────────────────────────────────────

function MergeCharacterModal({ source, candidates, onMerge, onClose }: {
  source: Character
  candidates: Character[]
  onMerge: (targetId: string) => Promise<boolean>
  onClose: () => void
}) {
  const [search, setSearch] = useState('')
  const [targetId, setTargetId] = useState<string | null>(null)
  const [merging, setMerging] = useState(false)

  const matches = useMemo(() => {
    const needle = search.trim().toLowerCase()
    return candidates
      .filter(character => !needle || character.name.toLowerCase().includes(needle) ||
        (character.aliases ?? []).some(alias => alias.toLowerCase().includes(needle)))
      .sort((a, b) => {
        const aStarts = needle && a.name.toLowerCase().startsWith(needle) ? 0 : 1
        const bStarts = needle && b.name.toLowerCase().startsWith(needle) ? 0 : 1
        return aStarts - bStarts || a.name.localeCompare(b.name)
      })
  }, [candidates, search])

  const target = targetId ? candidates.find(character => character.id === targetId) ?? null : null
  const submit = async () => {
    if (!target || merging) return
    setMerging(true)
    const ok = await onMerge(target.id)
    if (!ok) setMerging(false)
  }

  return (
    <div className="dialog-overlay" onMouseDown={onClose}>
      <div
        className="dialog chars-merge-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="character-merge-title"
        onMouseDown={event => event.stopPropagation()}
      >
        <div className="dialog-title" id="character-merge-title">Merge “{source.name}”</div>
        <p className="dialog-subtitle">Search for the correct character. The selected character keeps its display name; “{source.name}” becomes an alias.</p>
        <input
          className="dialog-input"
          value={search}
          onChange={event => { setSearch(event.target.value); setTargetId(null) }}
          placeholder="Search character names or aliases…"
          aria-label="Search characters to merge"
          autoFocus
        />

        <div className="chars-merge-results" role="listbox" aria-label="Merge targets">
          {matches.map(character => {
            const selected = character.id === targetId
            const aliases = (character.aliases ?? []).filter(alias => alias.toLowerCase() !== character.name.toLowerCase())
            return (
              <button
                key={character.id}
                type="button"
                role="option"
                aria-selected={selected}
                className={`chars-merge-result${selected ? ' selected' : ''}`}
                onClick={() => setTargetId(character.id)}
                disabled={merging}
              >
                <span className="chars-dot" style={{ background: characterColor(character.name) }} />
                <span className="chars-merge-result-text">
                  <strong>{character.name}</strong>
                  {aliases.length > 0 && <small>Aliases: {aliases.slice(0, 3).join(', ')}</small>}
                </span>
                <span className="chars-merge-result-count">{character.mention_count ?? 0} mentions</span>
              </button>
            )
          })}
          {matches.length === 0 && (
            <div className="chars-merge-empty">
              {search.trim()
                ? `No mergeable character matches “${search.trim()}”.`
                : 'No other detected characters are available to merge.'}
            </div>
          )}
        </div>

        {target && (
          <div className="chars-merge-preview">
            <span>{source.name}</span><strong>→</strong><span>{target.name}</span>
          </div>
        )}

        <div className="dialog-actions">
          <button className="dialog-btn" onClick={onClose} disabled={merging}>Cancel</button>
          <button className="dialog-btn primary" onClick={submit} disabled={!target || merging}>
            {merging ? 'Merging…' : target ? `Merge into ${target.name}` : 'Select a character'}
          </button>
        </div>
      </div>
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
