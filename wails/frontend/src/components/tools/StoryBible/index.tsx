// StoryBible - Characters, Plot Notes, and Timeline

import { useState } from 'react'
import { useBookStore } from '../../../store/bookStore'
import type { Character, CharacterRole } from '../../../types/draftline'
import { genId } from '../types'

// ── Characters Section ───────────────────────────────────────────────────────

export function CharactersSection() {
  const {
    book, addCharacter, updateCharacter, deleteCharacter, mergeCharacters,
    currentSection, currentIndex, setViewMode
  } = useBookStore()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [addingNew, setAddingNew] = useState(false)
  const [sortBy, setSortBy] = useState<'name' | 'mentions' | 'first'>('mentions')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [mergeMode, setMergeMode] = useState(false)
  const [showMergeDialog, setShowMergeDialog] = useState(false)

  if (!book) return <div className="tool-empty-state">Open a project to manage characters.</div>

  const allCharacters = book.story_bible?.characters ?? []

  // Get current chapter content for searching
  const getCurrentChapterContent = (): string => {
    if (currentSection === 'copyright') return book.copyright || ''
    const arr = currentSection === 'front_matter' ? book.front_matter
      : currentSection === 'body' ? book.body
      : book.back_matter
    return arr[currentIndex]?.content || ''
  }

  // Filter to characters appearing in current chapter by searching content
  // This respects aliases - if "Ruiz" is an alias, it will find mentions of "Ruiz" too
  const chapterContent = getCurrentChapterContent().toLowerCase()
  const characters = allCharacters.filter(char => {
    // Check if character name or any alias appears in chapter
    const namesToCheck = [char.name, ...(char.aliases || [])]
    return namesToCheck.some(name => {
      // Use word boundary check (simple version)
      const lowerName = name.toLowerCase()
      const idx = chapterContent.indexOf(lowerName)
      if (idx === -1) return false
      // Check it's a word boundary (not part of a longer word)
      const before = idx > 0 ? chapterContent[idx - 1] : ' '
      const after = idx + lowerName.length < chapterContent.length ? chapterContent[idx + lowerName.length] : ' '
      const isWordBoundary = !/[a-z]/.test(before) && !/[a-z]/.test(after)
      return isWordBoundary
    })
  })

  // Sort characters
  const sortedCharacters = [...characters].sort((a, b) => {
    if (sortBy === 'mentions') return (b.mention_count || 0) - (a.mention_count || 0)
    if (sortBy === 'first') return (a.first_chapter || 999) - (b.first_chapter || 999)
    return a.name.localeCompare(b.name)
  })

  const autoCount = characters.filter(c => c.is_auto_detected).length

  const toggleSelect = (id: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleMerge = (primaryId: string) => {
    const mergeIds = Array.from(selectedIds).filter(id => id !== primaryId)
    mergeCharacters(primaryId, mergeIds)
    setSelectedIds(new Set())
    setMergeMode(false)
    setShowMergeDialog(false)
  }

  const cancelMergeMode = () => {
    setMergeMode(false)
    setSelectedIds(new Set())
    setShowMergeDialog(false)
  }

  const selectedCharacters = characters.filter(c => selectedIds.has(c.id))

  return (
    <>
      {/* Merge dialog */}
      {showMergeDialog && selectedCharacters.length >= 2 && (
        <div className="merge-dialog">
          <div className="merge-dialog-header">Select Primary Name</div>
          <div className="merge-dialog-hint">Other names will become aliases of the selected character.</div>
          <div className="merge-options">
            {selectedCharacters.map(char => (
              <button key={char.id} className="merge-option" onClick={() => handleMerge(char.id)}>
                <span className="merge-option-name">{char.name}</span>
                {char.mention_count !== undefined && <span className="merge-option-count">({char.mention_count} mentions)</span>}
              </button>
            ))}
          </div>
          <button className="ai-link-btn" onClick={() => setShowMergeDialog(false)}>Cancel</button>
        </div>
      )}

      {/* Header with stats and actions */}
      <div className="characters-header">
        <div className="characters-stats">
          <span className="characters-count">{characters.length} in chapter</span>
          {autoCount > 0 && <span className="characters-auto-badge">{autoCount} auto</span>}
        </div>
        <div className="characters-actions">
          {mergeMode ? (
            <>
              <button
                className="tool-card-btn"
                onClick={() => setShowMergeDialog(true)}
                disabled={selectedIds.size < 2}
                title="Merge selected characters"
              >
                Merge ({selectedIds.size})
              </button>
              <button className="tool-card-btn secondary" onClick={cancelMergeMode}>Cancel</button>
            </>
          ) : (
            <>
              <button
                className="tool-card-btn"
                onClick={() => setViewMode('codex')}
                title="Open full character codex"
              >
                Codex
              </button>
              <button
                className="tool-card-btn secondary"
                onClick={() => setMergeMode(true)}
                disabled={characters.length < 2}
                title="Select characters to merge"
              >
                Merge
              </button>
            </>
          )}
        </div>
      </div>

      {/* Merge mode hint */}
      {mergeMode && (
        <div className="merge-mode-hint">
          Select 2+ characters that are the same person, then click Merge.
        </div>
      )}

      {/* Sort controls */}
      {characters.length > 0 && !mergeMode && (
        <div className="characters-sort">
          <span className="sort-label">Sort:</span>
          <button className={`sort-btn${sortBy === 'mentions' ? ' active' : ''}`} onClick={() => setSortBy('mentions')}>Mentions</button>
          <button className={`sort-btn${sortBy === 'first' ? ' active' : ''}`} onClick={() => setSortBy('first')}>First Appearance</button>
          <button className={`sort-btn${sortBy === 'name' ? ' active' : ''}`} onClick={() => setSortBy('name')}>Name</button>
        </div>
      )}

      {/* Character list */}
      {addingNew && (
        <CharacterForm char={null} onSave={c => { addCharacter(c); setAddingNew(false) }} onCancel={() => setAddingNew(false)} />
      )}
      {sortedCharacters.map(char =>
        editingId === char.id ? (
          <CharacterForm key={char.id} char={char} onSave={c => { updateCharacter(c); setEditingId(null) }} onCancel={() => setEditingId(null)} />
        ) : (
          <CharacterCard
            key={char.id}
            char={char}
            onEdit={() => setEditingId(char.id)}
            onDelete={() => deleteCharacter(char.id)}
            mergeMode={mergeMode}
            selected={selectedIds.has(char.id)}
            onToggleSelect={() => toggleSelect(char.id)}
          />
        )
      )}
      {!addingNew && !mergeMode && (
        <button className="tool-card-btn" style={{ marginTop: 4 }} onClick={() => setAddingNew(true)}>+ Add Character</button>
      )}

      {/* Empty state with link to codex */}
      {characters.length === 0 && !addingNew && (
        <div className="characters-empty">
          <p>No characters in this chapter.</p>
          <p className="hint">
            <button className="ai-link-btn" onClick={() => setViewMode('codex')}>Open Codex</button>
            {' '}to see all characters or re-index the book.
          </p>
        </div>
      )}
    </>
  )
}

function CharacterCard({ char, onEdit, onDelete, mergeMode, selected, onToggleSelect }: {
  char: Character
  onEdit: () => void
  onDelete: () => void
  mergeMode?: boolean
  selected?: boolean
  onToggleSelect?: () => void
}) {
  const { book, highlightedCharacterId, setHighlightedCharacter } = useBookStore()
  const [expanded, setExpanded] = useState(false)

  const isHighlighted = highlightedCharacterId === char.id

  // Get chapter name from index
  const getChapterName = (index: number) => {
    if (!book) return `Chapter ${index + 1}`
    const allChapters = [...(book.front_matter || []), ...(book.body || []), ...(book.back_matter || [])]
    return allChapters[index]?.title || `Chapter ${index + 1}`
  }

  const handleClick = () => {
    if (mergeMode && onToggleSelect) {
      onToggleSelect()
    } else {
      setExpanded(v => !v)
    }
  }

  const handleHighlightToggle = (e: React.MouseEvent) => {
    e.stopPropagation()
    setHighlightedCharacter(isHighlighted ? null : char.id)
  }

  return (
    <div
      className={`character-card${char.is_auto_detected ? ' auto-detected' : ''}${mergeMode ? ' merge-mode' : ''}${selected ? ' selected' : ''}${isHighlighted ? ' highlighting' : ''}`}
      onClick={handleClick}
    >
      <div className="character-card-header">
        {mergeMode && (
          <span className={`merge-checkbox${selected ? ' checked' : ''}`}>
            {selected ? '☑' : '☐'}
          </span>
        )}
        <span className="character-name">{char.name || 'Unnamed'}</span>
        <div className="character-badges">
          {char.is_auto_detected && <span className="character-auto-badge" title="Auto-detected">✨</span>}
          <span className={`character-role ${char.role}`}>{char.role}</span>
          {!mergeMode && (
            <button
              className={`character-highlight-btn${isHighlighted ? ' active' : ''}`}
              onClick={handleHighlightToggle}
              title={isHighlighted ? 'Stop highlighting in text' : 'Highlight in text'}
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="11" cy="11" r="8" />
                <path d="M21 21l-4.35-4.35" />
              </svg>
            </button>
          )}
        </div>
      </div>

      {/* Quick stats row */}
      {(char.mention_count !== undefined || char.first_chapter !== undefined) && (
        <div className="character-stats-row">
          {char.mention_count !== undefined && (
            <span className="character-stat" title="Total mentions">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z" />
              </svg>
              {char.mention_count}
            </span>
          )}
          {char.first_chapter !== undefined && (
            <span className="character-stat" title={`First appears in ${getChapterName(char.first_chapter)}`}>
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M4 19.5A2.5 2.5 0 016.5 17H20" />
                <path d="M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z" />
              </svg>
              Ch {char.first_chapter + 1}
            </span>
          )}
        </div>
      )}

      {/* Auto-detected attributes */}
      {char.attributes && Object.keys(char.attributes).length > 0 && (
        <div className="character-attributes">
          {char.attributes.eye_color && <span className="char-attr">Eyes: {char.attributes.eye_color}</span>}
          {char.attributes.hair_color && <span className="char-attr">Hair: {char.attributes.hair_color}</span>}
          {char.attributes.age && <span className="char-attr">Age: {char.attributes.age}</span>}
        </div>
      )}

      {char.description && <div className="character-desc-preview">{char.description}</div>}

      {expanded && (
        <div className="character-expanded" onClick={e => e.stopPropagation()}>
          {char.appearance && <div className="char-field"><span className="char-field-label">Appearance</span><span>{char.appearance}</span></div>}
          {char.personality && <div className="char-field"><span className="char-field-label">Personality</span><span>{char.personality}</span></div>}
          {char.motivation && <div className="char-field"><span className="char-field-label">Motivation</span><span>{char.motivation}</span></div>}
          {char.notes && <div className="char-field"><span className="char-field-label">Notes</span><span>{char.notes}</span></div>}

          {/* Chapter mentions breakdown */}
          {char.chapter_mentions && Object.keys(char.chapter_mentions).length > 0 && (
            <div className="char-field">
              <span className="char-field-label">Appears in</span>
              <div className="chapter-mentions">
                {Object.entries(char.chapter_mentions)
                  .sort(([a], [b]) => Number(a) - Number(b))
                  .map(([chIdx, count]) => (
                    <span key={chIdx} className="chapter-mention-badge">
                      {getChapterName(Number(chIdx))} ({count})
                    </span>
                  ))}
              </div>
            </div>
          )}

          {/* Aliases */}
          {char.aliases && char.aliases.length > 0 && (
            <div className="char-field">
              <span className="char-field-label">Also known as</span>
              <span>{char.aliases.join(', ')}</span>
            </div>
          )}

          <div className="character-card-actions">
            <button className="tool-card-btn secondary" style={{ fontSize: 11, padding: '2px 10px' }} onClick={e => { e.stopPropagation(); onEdit() }}>Edit</button>
            <button className="tool-card-btn secondary" style={{ fontSize: 11, padding: '2px 10px', color: '#E06C75' }} onClick={e => { e.stopPropagation(); onDelete() }}>Delete</button>
          </div>
        </div>
      )}
    </div>
  )
}

function CharacterForm({ char, onSave, onCancel }: { char: Character | null; onSave: (c: Character) => void; onCancel: () => void }) {
  const [name, setName] = useState(char?.name ?? '')
  const [role, setRole] = useState<CharacterRole | string>(char?.role ?? 'supporting')
  const [desc, setDesc] = useState(char?.description ?? '')
  const [app, setApp] = useState(char?.appearance ?? '')
  const [persona, setPersona] = useState(char?.personality ?? '')
  const [motiv, setMotiv] = useState(char?.motivation ?? '')
  const [notes, setNotes] = useState(char?.notes ?? '')

  return (
    <div className="character-form">
      <div className="character-form-title">{char ? 'Edit Character' : 'New Character'}</div>
      <div className="character-form-row">
        <div className="dialog-field" style={{ margin: 0 }}>
          <label className="dialog-label">Name</label>
          <input className="dialog-input" value={name} onChange={e => setName(e.target.value)} placeholder="Character name" autoFocus />
        </div>
        <div className="dialog-field" style={{ margin: 0 }}>
          <label className="dialog-label">Role</label>
          <select className="dialog-select" value={role} onChange={e => setRole(e.target.value as CharacterRole)}>
            <option value="protagonist">Protagonist</option>
            <option value="antagonist">Antagonist</option>
            <option value="supporting">Supporting</option>
            <option value="minor">Minor</option>
            <option value="other">Other</option>
          </select>
        </div>
      </div>
      <div className="dialog-field" style={{ marginTop: 6 }}>
        <label className="dialog-label">Description</label>
        <textarea className="dialog-input" rows={2} value={desc} onChange={e => setDesc(e.target.value)} placeholder="Brief overview of this character's role" style={{ resize: 'vertical' }} />
      </div>
      <div className="dialog-field" style={{ marginTop: 4 }}>
        <label className="dialog-label">Appearance</label>
        <textarea className="dialog-input" rows={2} value={app} onChange={e => setApp(e.target.value)} placeholder="Physical description, distinguishing features" style={{ resize: 'vertical' }} />
      </div>
      <div className="character-form-row" style={{ marginTop: 4 }}>
        <div className="dialog-field" style={{ margin: 0 }}>
          <label className="dialog-label">Personality</label>
          <textarea className="dialog-input" rows={2} value={persona} onChange={e => setPersona(e.target.value)} placeholder="Traits, quirks, voice" style={{ resize: 'vertical' }} />
        </div>
        <div className="dialog-field" style={{ margin: 0 }}>
          <label className="dialog-label">Motivation</label>
          <textarea className="dialog-input" rows={2} value={motiv} onChange={e => setMotiv(e.target.value)} placeholder="Goals, fears, driving need" style={{ resize: 'vertical' }} />
        </div>
      </div>
      <div className="dialog-field" style={{ marginTop: 4 }}>
        <label className="dialog-label">Notes</label>
        <textarea className="dialog-input" rows={2} value={notes} onChange={e => setNotes(e.target.value)} placeholder="Arc beats, continuity flags, backstory" style={{ resize: 'vertical' }} />
      </div>
      <div className="ai-actions" style={{ marginTop: 8 }}>
        <button className="ai-run-btn" onClick={() => { if (name.trim()) onSave({ id: char?.id ?? genId(), name: name.trim(), role, description: desc, appearance: app, personality: persona, motivation: motiv, notes }) }} disabled={!name.trim()}>
          {char ? 'Save Changes' : 'Add Character'}
        </button>
        <button className="ai-link-btn" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}

// ── Plot Notes ───────────────────────────────────────────────────────────────

export function PlotSection() {
  const { book, updateStoryBibleText } = useBookStore()
  if (!book) return <div className="tool-empty-state">Open a project first.</div>
  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Plot Bible</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>Outline acts, subplots, themes, and unresolved threads.</p>
      <textarea
        className="bible-textarea"
        value={book.story_bible?.plot_notes ?? ''}
        onChange={e => updateStoryBibleText('plot_notes', e.target.value)}
        placeholder={"Act 1:\n  Opening image...\n  Inciting incident...\n\nAct 2:\n  ...\n\nThemes:\n  ..."}
      />
    </div>
  )
}

// ── Timeline ─────────────────────────────────────────────────────────────────

export function TimelineSection() {
  const { book, updateStoryBibleText } = useBookStore()
  if (!book) return <div className="tool-empty-state">Open a project first.</div>
  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Story Timeline</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>Chronological events in story time. One event per line.</p>
      <textarea
        className="bible-textarea"
        value={book.story_bible?.timeline ?? ''}
        onChange={e => updateStoryBibleText('timeline', e.target.value)}
        placeholder={"Day 1 — Marcus arrives in the city (Ch. 1)\nDay 1, evening — He meets Clara (Ch. 2)\nDay 3 — The letter arrives (Ch. 4)"}
      />
    </div>
  )
}
