// CastQuickRef — compact character reference for the tools sidebar.
// Read-only at a glance: who's who, how present they are, highlight in text.
// Full management and the relationship web live in the Cast view.

import { useState } from 'react'
import { useBookStore } from '../../store/bookStore'
import { characterColor, characterInitials } from '../../utils/characterVisuals'
import { hexToRgba } from '../../utils/accentColor'
import { isConfirmedCharacter } from '../../utils/characterStatus'
import '../cast/cast.css'

export default function CastQuickRef() {
  const {
    book, setViewMode, indexBook, isIndexing,
    highlightedCharacterId, setHighlightedCharacter,
  } = useBookStore()
  const [query, setQuery] = useState('')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  if (!book) return <div className="tool-empty-state">Open a project to see its cast.</div>

  const allCharacters = book.story_bible?.characters ?? []
  const characters = allCharacters.filter(isConfirmedCharacter)

  const q = query.trim().toLowerCase()
  const matching = q
    ? characters.filter(c =>
        c.name.toLowerCase().includes(q) ||
        (c.aliases || []).some(a => a.toLowerCase().includes(q)))
    : characters
  const filtered = [...matching].sort((a, b) => (b.mention_count || 0) - (a.mention_count || 0))

  const chapterName = (index: number) => {
    const all = [...(book.front_matter || []), ...(book.body || []), ...(book.back_matter || [])]
    return all[index]?.title || `Chapter ${index + 1}`
  }

  return (
    <div>
      <div style={{ display: 'flex', gap: 6, marginBottom: 8 }}>
        <button className="tool-card-btn" style={{ flex: 1 }} onClick={() => setViewMode('cast')}>
          Open Cast View
        </button>
      </div>

      {characters.length > 4 && (
        <input
          className="dialog-input"
          style={{ marginBottom: 6 }}
          placeholder="Filter cast…"
          value={query}
          onChange={e => setQuery(e.target.value)}
        />
      )}

      {filtered.map(c => {
        const color = characterColor(c.name)
        const expanded = expandedId === c.id
        const highlighted = highlightedCharacterId === c.id
        return (
          <div key={c.id}>
            <button
              className={`cast-row${expanded ? ' selected' : ''}`}
              onClick={() => setExpandedId(expanded ? null : c.id)}
            >
              <span className="cast-dot" style={{ background: color }} />
              <span className="cast-row-name">{c.name}</span>
              {c.mention_count ? <span className="cast-row-count">{c.mention_count}</span> : null}
            </button>
            {expanded && (
              <div style={{ padding: '4px 10px 8px 19px', fontSize: 11, lineHeight: 1.6, color: 'var(--text-secondary)' }}>
                {c.aliases && c.aliases.length > 0 && <div>aka {c.aliases.slice(0, 4).join(', ')}</div>}
                {c.first_chapter !== undefined && <div>First appears: {chapterName(c.first_chapter)}</div>}
                {c.description && <div style={{ marginTop: 2 }}>{c.description}</div>}
                <div style={{ display: 'flex', gap: 6, marginTop: 6 }}>
                  <button
                    className="ai-link-btn"
                    style={highlighted ? { color, textShadow: `0 0 8px ${hexToRgba(color, 0.5)}` } : undefined}
                    onClick={() => setHighlightedCharacter(highlighted ? null : c.id)}
                  >
                    {highlighted ? 'Stop highlighting' : 'Highlight in text'}
                  </button>
                  <button className="ai-link-btn" onClick={() => setViewMode('cast')}>Details…</button>
                </div>
              </div>
            )}
          </div>
        )
      })}

      {filtered.length === 0 && (
        <div className="characters-empty">
          {characters.length === 0 ? (
            <>
              <p>No characters yet.</p>
              <button className="tool-card-btn" onClick={() => indexBook()} disabled={isIndexing} style={{ marginTop: 6 }}>
                {isIndexing ? 'Detecting…' : 'Detect Cast'}
              </button>
            </>
          ) : (
            <p>No matches.</p>
          )}
        </div>
      )}
    </div>
  )
}
