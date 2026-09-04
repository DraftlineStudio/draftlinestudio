// CharacterQuickRef — chapter-first character sidebar for the tools panel.
// Characters in the chapter you're writing come first with their local mention
// counts; everyone else follows with a mini presence strip. Every row expands
// an inline card (meta, presence, top ties, jump links) — nothing else moves.
// Full management lives in the Characters codex.

import { useMemo, useState } from 'react'
import { useBookStore } from '../../store/bookStore'
import { characterColor } from '../../utils/characterVisuals'
import { hexToRgba } from '../../utils/accentColor'
import { isConfirmedCharacter } from '../../utils/characterStatus'
import { allChapters, chapterName, cellAlpha, chapterLocation, requestCharacterFocus } from '../characters/shared'
import type { Character } from '../../types/draftline'

export default function CharacterQuickRef() {
  const {
    book, setViewMode, setCurrentChapter, indexBook, isIndexing,
    currentSection, currentIndex,
    highlightedCharacterId, setHighlightedCharacter,
  } = useBookStore()
  const [query, setQuery] = useState('')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const characters = useMemo(
    () => (book?.story_bible?.characters ?? []).filter(isConfirmedCharacter),
    [book],
  )
  const relationships = book?.analysis?.relationships?.relationships ?? []

  if (!book) return <div className="tool-empty-state">Open a project to see its characters.</div>

  const chapters = allChapters(book)
  // Combined index of the chapter being written; -1 for the copyright page.
  const front = book.front_matter?.length || 0
  const bodyLen = book.body?.length || 0
  const curIdx = currentSection === 'front_matter' ? currentIndex
    : currentSection === 'body' ? front + currentIndex
    : currentSection === 'back_matter' ? front + bodyLen + currentIndex
    : -1

  const q = query.trim().toLowerCase()
  const matching = q
    ? characters.filter(c =>
        c.name.toLowerCase().includes(q) ||
        (c.aliases || []).some(a => a.toLowerCase().includes(q)))
    : characters
  const byMentions = [...matching].sort((a, b) => (b.mention_count || 0) - (a.mention_count || 0))
  const here = curIdx >= 0 ? byMentions.filter(c => (c.chapter_mentions?.[curIdx] || 0) > 0) : []
  const hereIds = new Set(here.map(c => c.id))
  const rest = byMentions.filter(c => !hereIds.has(c.id))

  const openCodex = (focusId?: string) => {
    if (focusId) requestCharacterFocus(focusId)
    setViewMode('cast')
  }

  const jumpToFirstMention = (c: Character) => {
    if (c.first_chapter === undefined) return
    const loc = chapterLocation(book, c.first_chapter)
    setCurrentChapter(loc.section, loc.index)
  }

  const stripCells = (c: Character, color: string) =>
    chapters.map((_, i) => {
      const v = c.chapter_mentions?.[i] || 0
      return (
        <div
          key={i}
          title={`${chapterName(book, i)} · ${v} ${v === 1 ? 'mention' : 'mentions'}`}
          style={{ background: v > 0 ? hexToRgba(color, cellAlpha(v)) : 'rgba(255,255,255,0.04)' }}
        />
      )
    })

  const detailCard = (c: Character, color: string) => {
    const ties = relationships
      .filter(r => r.character1_id === c.id || r.character2_id === c.id)
      .map(r => ({ otherId: r.character1_id === c.id ? r.character2_id : r.character1_id, strength: r.strength }))
      .map(t => ({ ...t, other: characters.find(o => o.id === t.otherId) }))
      .filter(t => t.other)
      .sort((a, b) => b.strength - a.strength)
      .slice(0, 3)
    const maxW = ties.length ? ties[0].strength : 1
    const highlighted = highlightedCharacterId === c.id
    const chapterCount = c.chapter_mentions ? Object.keys(c.chapter_mentions).length : 0

    return (
      <div className="chars-side-card">
        <div className="chars-side-card-meta">
          {c.first_chapter !== undefined && <>First: {chapterName(book, c.first_chapter)}</>}
          {chapterCount > 0 && <> · {chapterCount} chapter{chapterCount === 1 ? '' : 's'}</>}
          {c.mention_count ? <> · {c.mention_count} mentions</> : null}
          {c.aliases && c.aliases.length > 0 && <div>aka {c.aliases.slice(0, 4).join(', ')}</div>}
        </div>
        <div className="chars-side-strip" style={{ gridTemplateColumns: `repeat(${chapters.length}, 1fr)` }}>
          {stripCells(c, color)}
        </div>
        {ties.length > 0 && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            {ties.map(t => {
              const tColor = characterColor(t.other!.name)
              return (
                <div key={t.otherId} className="chars-side-tie">
                  <span className="chars-dot" style={{ width: 6, height: 6, background: tColor }} />
                  <span className="chars-name">{t.other!.name}</span>
                  <span className="chars-side-tiebar">
                    <span style={{ width: `${Math.round((100 * t.strength) / maxW)}%`, background: tColor }} />
                  </span>
                </div>
              )
            })}
          </div>
        )}
        <div className="chars-side-links">
          <button className="ai-link-btn" onClick={() => jumpToFirstMention(c)}>Jump to first mention</button>
          <button className="ai-link-btn" onClick={() => openCodex(c.id)}>Open in Characters</button>
          <button
            className="ai-link-btn"
            style={highlighted ? { color, textShadow: `0 0 8px ${hexToRgba(color, 0.5)}` } : undefined}
            onClick={() => setHighlightedCharacter(highlighted ? null : c.id)}
          >
            {highlighted ? 'Stop highlighting' : 'Highlight'}
          </button>
        </div>
      </div>
    )
  }

  const hereRow = (c: Character) => {
    const color = characterColor(c.name)
    const open = expandedId === c.id
    return (
      <div key={c.id}>
        <button className={`chars-side-row${open ? ' open' : ''}`} onClick={() => setExpandedId(open ? null : c.id)}>
          <span className="chars-dot" style={{ background: color }} />
          <span className="chars-name">{c.name}</span>
          <span className="chars-side-count">{c.chapter_mentions?.[curIdx] || 0} here</span>
        </button>
        {open && detailCard(c, color)}
      </div>
    )
  }

  const restRow = (c: Character) => {
    const color = characterColor(c.name)
    const open = expandedId === c.id
    return (
      <div key={c.id}>
        <button className={`chars-side-row2${open ? ' open' : ''}`} onClick={() => setExpandedId(open ? null : c.id)}>
          <div className="chars-side-namerow">
            <span className="chars-dot" style={{ background: color }} />
            <span className="chars-name">{c.name}</span>
            <span className="chars-side-count">{c.mention_count || ''}</span>
          </div>
          <div className="chars-side-ministrip" style={{ gridTemplateColumns: `repeat(${chapters.length}, 1fr)` }}>
            {stripCells(c, color)}
          </div>
        </button>
        {open && detailCard(c, color)}
      </div>
    )
  }

  return (
    <div className="chars-side">
      <div className="chars-side-toolbar">
        <input
          className="dialog-input"
          placeholder="Filter characters…"
          value={query}
          onChange={e => setQuery(e.target.value)}
        />
        <button className="chars-side-open" title="Open Characters view" onClick={() => openCodex()}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <path d="M15 3h6v6" /><path d="M10 14 21 3" /><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
          </svg>
        </button>
      </div>

      <div className="chars-side-list">
        {here.length > 0 && (
          <>
            <div className="chars-side-sect">
              <span className="chars-side-label">In {curIdx >= 0 ? chapterName(book, curIdx) : 'this chapter'}</span>
              <span className="muted">· {here.length}</span>
            </div>
            {here.map(hereRow)}
          </>
        )}

        {rest.length > 0 && (
          <>
            <div className="chars-side-sect">
              <span className="chars-side-label">{here.length > 0 ? 'Everyone else' : 'Characters'}</span>
            </div>
            {rest.map(restRow)}
          </>
        )}

        {byMentions.length === 0 && (
          <div className="chars-side-empty">
            {characters.length === 0 ? (
              <>
                <p>No characters yet.</p>
                <button className="tool-card-btn" onClick={() => indexBook()} disabled={isIndexing} style={{ width: 'auto', padding: '0 12px', marginTop: 6 }}>
                  {isIndexing ? 'Detecting…' : 'Detect Characters'}
                </button>
              </>
            ) : (
              <p>No matches.</p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
