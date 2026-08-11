// CastView — full-workspace view of the book's cast: character list on the
// left, relationship web in the center, dossier on the right, chapter spine
// below. Styled like the rest of the app (IDE look, app font).
// All detection is local pattern-matching — no AI, nothing leaves the machine.

import { useMemo, useState, useEffect } from 'react'
import { useBookStore } from '../../store/bookStore'
import { useRelationshipStore } from '../../store/relationshipStore'
import { characterColor, characterInitials } from '../../utils/characterVisuals'
import { hexToRgba } from '../../utils/accentColor'
import type { BookData, Character, CharacterRole, CharacterEvent } from '../../types/draftline'
import { WebCanvas } from './WebCanvas'
import './cast.css'

function allChapters(book: BookData) {
  return [...(book.front_matter || []), ...(book.body || []), ...(book.back_matter || [])]
}

function chapterName(book: BookData, index: number): string {
  return allChapters(book)[index]?.title || `Chapter ${index + 1}`
}

export default function CastView() {
  const {
    book, setViewMode, indexBook, isIndexing, updateBook,
    addCharacter, updateCharacter, deleteCharacter, clearAllCharacters,
    mergeEntities, splitEntity,
  } = useBookStore()
  const { isAnalyzing, analyzeRelationships } = useRelationshipStore()

  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [query, setQuery] = useState('')
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
  const hasWeb = relationships.length > 0
  const entityIds = useMemo(
    () => new Set((book?.analysis?.entity_resolution?.entities ?? []).map(e => e.id)),
    [book],
  )

  const charMap = useMemo(() => {
    const m = new Map<string, Character>()
    for (const c of characters) m.set(c.id, c)
    return m
  }, [characters])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    const list = q
      ? characters.filter(c =>
          c.name.toLowerCase().includes(q) ||
          (c.aliases || []).some(a => a.toLowerCase().includes(q)))
      : characters
    return [...list].sort((a, b) => (b.mention_count || 0) - (a.mention_count || 0))
  }, [characters, query])

  const selected = selectedId ? charMap.get(selectedId) ?? null : null

  // Detect characters and weave relationships in one action.
  const detect = async () => {
    if (!book) return
    await indexBook()
    const fresh = useBookStore.getState().book
    if (!fresh?.analysis?.entity_resolution?.entities?.length) return
    const updated = await analyzeRelationships(fresh)
    if (updated) updateBook(updated)
  }

  const pickForMerge = (id: string) => {
    if (!mergeFrom || id === mergeFrom) return
    if (!entityIds.has(id)) return
    setMergeWith(id)
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

  return (
    <div className="cast-view">
      <header className="cast-header">
        <div className="cast-header-title">
          <span className="cast-heading">Cast</span>
          <span className="cast-count">{characters.length} characters{hasWeb ? ` · ${relationships.length} relationships` : ''}</span>
        </div>
        <div className="cast-header-actions">
          <button className="tool-card-btn" onClick={detect} disabled={busy}>
            {busy ? 'Working…' : characters.length ? 'Re-Detect Cast' : 'Detect Cast'}
          </button>
          <button className="tool-card-btn secondary" onClick={() => setViewMode('editor')} title="Back to writing (Esc)">
            Close
          </button>
        </div>
      </header>

      <div className="cast-body">
        {/* ── Cast rail ── */}
        <aside className="cast-rail">
          <input
            className="dialog-input cast-search"
            placeholder="Filter cast…"
            value={query}
            onChange={e => setQuery(e.target.value)}
          />
          {mergeFrom && (
            <div className="cast-merge-banner">
              Click who <strong>{charMap.get(mergeFrom)?.name}</strong> really is…
              <button className="ai-link-btn" onClick={() => { setMergeFrom(null); setMergeWith(null) }}>cancel</button>
            </div>
          )}
          <div className="cast-list">
            {filtered.map(c => (
              <button
                key={c.id}
                className={`cast-row${selectedId === c.id ? ' selected' : ''}${mergeFrom === c.id ? ' merge-source' : ''}`}
                onClick={() => (mergeFrom ? pickForMerge(c.id) : (setSelectedId(c.id), setEditing(false), setSplitting(false)))}
              >
                <span className="cast-dot" style={{ background: characterColor(c.name) }} />
                <span className="cast-row-name">{c.name}</span>
                {c.mention_count ? <span className="cast-row-count">{c.mention_count}</span> : null}
              </button>
            ))}
            {filtered.length === 0 && (
              <div className="cast-rail-empty">
                {characters.length === 0 ? 'No characters yet — click Detect Cast.' : 'No matches.'}
              </div>
            )}
          </div>
          <div className="cast-rail-footer">
            <button className="ai-link-btn" onClick={() => { setAdding(true); setSelectedId(null) }}>+ Add character</button>
            <span style={{ flex: 1 }} />
            {confirmClear ? (
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
            )}
          </div>
        </aside>

        {/* ── Stage ── */}
        <main className="cast-stage">
          {hasWeb ? (
            <WebCanvas book={book} selectedId={selectedId} onSelect={id => { setSelectedId(id); setEditing(false); setSplitting(false) }} />
          ) : (
            <div className="cast-stage-empty">
              <p>
                {characters.length
                  ? 'Detect the cast again to map who interacts with whom.'
                  : 'Detect your cast to see the people of this book and the relationships between them.'}
              </p>
              <button className="tool-card-btn" onClick={detect} disabled={busy}>
                {busy ? 'Working…' : 'Detect Cast'}
              </button>
              <p className="cast-stage-note">Detection runs entirely on your machine — no AI, no network.</p>
            </div>
          )}
          {hasWeb && <StorySpine book={book} selected={selected} events={events} />}
        </main>

        {/* ── Dossier ── */}
        {(selected || adding) && (
          <aside className="cast-dossier">
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
            ) : mergeWith && mergeFrom && selected ? (
              <MergeConfirm
                a={charMap.get(mergeFrom)!}
                b={charMap.get(mergeWith)!}
                onKeep={doMerge}
                onCancel={() => setMergeWith(null)}
              />
            ) : selected ? (
              <Dossier
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
                onClose={() => setSelectedId(null)}
              />
            ) : null}
          </aside>
        )}
      </div>
    </div>
  )
}

// ── Dossier ──────────────────────────────────────────────────────────────────

function Dossier({ book, char, charMap, relationships, events, entityBacked, onSelect, onEdit, onSplit, onMerge, onDelete, onClose }: {
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
  onClose: () => void
}) {
  const color = characterColor(char.name)

  const bonds = relationships
    .filter(r => r.character1_id === char.id || r.character2_id === char.id)
    .map(r => ({ otherId: r.character1_id === char.id ? r.character2_id : r.character1_id, rel: r }))
    .filter(b => charMap.has(b.otherId))
    .sort((a, b) => b.rel.strength - a.rel.strength)

  const moments = events
    .filter(e => e.character_ids?.includes(char.id))
    .sort((a, b) => a.chapter_index - b.chapter_index)

  const chapterCount = char.chapter_mentions ? Object.keys(char.chapter_mentions).length : 0

  return (
    <>
      <div className="dossier-top">
        <span className="cast-portrait" style={{ color, borderColor: color, background: hexToRgba(color, 0.12) }}>
          {characterInitials(char.name)}
        </span>
        <div className="dossier-title">
          <div className="dossier-name">{char.name}</div>
          <div className="dossier-role">{char.role || 'minor'}</div>
        </div>
        <button className="dossier-close" onClick={onClose} title="Close">×</button>
      </div>

      {char.aliases && char.aliases.length > 0 && (
        <div className="dossier-line muted">aka {char.aliases.slice(0, 5).join(', ')}</div>
      )}

      <div className="dossier-line">
        {char.first_chapter !== undefined && <>First appears: <strong>{chapterName(book, char.first_chapter)}</strong></>}
        {chapterCount > 0 && <> · {chapterCount} chapter{chapterCount === 1 ? '' : 's'}</>}
        {char.mention_count ? <> · {char.mention_count} mentions</> : null}
      </div>

      {char.description && <div className="dossier-line">{char.description}</div>}

      {bonds.length > 0 && (
        <div className="dossier-block">
          <div className="dossier-block-title">Relationships</div>
          {bonds.map(({ otherId, rel }) => {
            const other = charMap.get(otherId)!
            return (
              <button key={rel.id} className="cast-row" onClick={() => onSelect(otherId)}>
                <span className="cast-dot" style={{ background: characterColor(other.name) }} />
                <span className="cast-row-name">{other.name}</span>
                <span className="bond-meter"><span style={{ width: `${Math.round(rel.strength * 100)}%`, background: color }} /></span>
                <span className="cast-row-count">{rel.interaction_count}</span>
              </button>
            )
          })}
        </div>
      )}

      {moments.length > 0 && (
        <div className="dossier-block">
          <div className="dossier-block-title">Key events</div>
          {moments.map(m => (
            <div key={m.id} className="dossier-event">
              <span className="dossier-event-dot" style={{ background: color }} />
              <div>
                <div>{m.description}</div>
                <div className="muted">{chapterName(book, m.chapter_index)}</div>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="dossier-actions">
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
    <div className="dossier-block">
      <div className="dossier-block-title">Same person — keep which name?</div>
      <p className="dossier-line muted">The other name becomes an alias. Remembered on every re-detect.</p>
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
    <div className="dossier-block">
      <div className="dossier-block-title">Split "{char.name}"</div>
      <p className="dossier-line muted">Check the mentions that belong to a different person.</p>
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
    <div className="dossier-block">
      <div className="dossier-block-title">{char ? 'Edit character' : 'New character'}</div>
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

// ── Story spine ──────────────────────────────────────────────────────────────

function StorySpine({ book, selected, events }: {
  book: BookData
  selected: Character | null
  events: CharacterEvent[]
}) {
  const chapters = allChapters(book)
  if (!chapters.length) return null

  const color = selected ? characterColor(selected.name) : 'var(--app-accent, #5B8BFF)'
  const mentionsAt = (idx: number): number => selected?.chapter_mentions?.[idx] || 0
  const maxMentions = selected?.chapter_mentions
    ? Math.max(1, ...Object.values(selected.chapter_mentions))
    : 1
  const meetingsAt = (idx: number) =>
    events.some(e => e.event_type === 'meeting' && e.chapter_index === idx &&
      (!selected || e.character_ids?.includes(selected.id)))

  return (
    <footer className="story-spine">
      <span className="spine-label">{selected ? selected.name : 'All chapters'}</span>
      <div className="spine-track">
        {chapters.map((ch, idx) => {
          const m = mentionsAt(idx)
          const intensity = selected ? m / maxMentions : 0
          return (
            <div key={idx} className="spine-chapter" title={`${ch.title || `Chapter ${idx + 1}`}${m ? ` — ${m} mentions` : ''}`}>
              <div
                className="spine-glow"
                style={selected && m > 0 ? {
                  background: hexToRgba(color.startsWith('#') ? color : '#5B8BFF', 0.35 + intensity * 0.6),
                  height: `${4 + intensity * 14}px`,
                } : undefined}
              />
              {meetingsAt(idx) && <span className="spine-meeting" style={{ color }}>◆</span>}
            </div>
          )
        })}
      </div>
    </footer>
  )
}
