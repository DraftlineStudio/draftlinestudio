import { useState } from 'react'
import { useBookStore } from '../store/bookStore'
import type { Character, CharacterRole } from '../types/draftline'

function genId(): string {
  return Math.random().toString(36).slice(2) + Date.now().toString(36)
}

export default function CodexPanel() {
  const {
    book, setViewMode,
    addCharacter, updateCharacter, deleteCharacter,
    deleteAutoDetectedCharacters,
    mergeCharacters, indexBook, isIndexing
  } = useBookStore()

  const [editingId, setEditingId] = useState<string | null>(null)
  const [addingNew, setAddingNew] = useState(false)
  const [sortBy, setSortBy] = useState<'name' | 'mentions' | 'first'>('mentions')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [mergeMode, setMergeMode] = useState(false)
  const [showMergeDialog, setShowMergeDialog] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  if (!book) {
    return (
      <div className="codex-panel">
        <div className="codex-empty">Open a project to view the Codex.</div>
      </div>
    )
  }

  const characters = book.story_bible?.characters ?? []
  const autoCount = characters.filter(c => c.is_auto_detected).length

  const sortedCharacters = [...characters].sort((a, b) => {
    if (sortBy === 'mentions') return (b.mention_count || 0) - (a.mention_count || 0)
    if (sortBy === 'first') return (a.first_chapter || 999) - (b.first_chapter || 999)
    return a.name.localeCompare(b.name)
  })

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

  const handleDeleteAllAutoDetected = () => {
    deleteAutoDetectedCharacters()
    setShowDeleteConfirm(false)
  }

  const selectedCharacters = characters.filter(c => selectedIds.has(c.id))

  // Get chapter name from index
  const getChapterName = (index: number) => {
    if (!book) return `Chapter ${index + 1}`
    const allChapters = [...(book.front_matter || []), ...(book.body || []), ...(book.back_matter || [])]
    return allChapters[index]?.title || `Chapter ${index + 1}`
  }

  return (
    <div className="codex-panel">
      {/* Header with back button */}
      <div className="codex-header">
        <button className="codex-back-btn" onClick={() => setViewMode('editor')}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M19 12H5M12 19l-7-7 7-7"/>
          </svg>
          Back to Editor
        </button>
        <h1 className="codex-title">Character Codex</h1>
      </div>

      {/* Action bar */}
      <div className="codex-actions">
        <div className="codex-stats">
          <span className="codex-count">{characters.length} characters</span>
          {autoCount > 0 && (
            <span className="codex-auto-badge">{autoCount} auto-detected</span>
          )}
        </div>

        <div className="codex-buttons">
          {mergeMode ? (
            <>
              <button
                className="codex-btn"
                onClick={() => setShowMergeDialog(true)}
                disabled={selectedIds.size < 2}
              >
                Merge ({selectedIds.size})
              </button>
              <button className="codex-btn secondary" onClick={cancelMergeMode}>
                Cancel
              </button>
            </>
          ) : (
            <>
              <button
                className="codex-btn secondary"
                onClick={() => indexBook()}
                disabled={isIndexing}
              >
                {isIndexing ? 'Indexing...' : 'Re-Index Book'}
              </button>

              {autoCount > 0 && (
                <button
                  className="codex-btn danger"
                  onClick={() => setShowDeleteConfirm(true)}
                >
                  Delete All Auto-Detected
                </button>
              )}

              <button
                className="codex-btn secondary"
                onClick={() => setMergeMode(true)}
                disabled={characters.length < 2}
              >
                Merge
              </button>

              <button
                className="codex-btn"
                onClick={() => setAddingNew(true)}
              >
                + Add Character
              </button>
            </>
          )}
        </div>
      </div>

      {/* Merge mode hint */}
      {mergeMode && (
        <div className="codex-merge-hint">
          Select 2+ characters that are the same person, then click Merge.
        </div>
      )}

      {/* Sort controls */}
      {characters.length > 0 && !mergeMode && (
        <div className="codex-sort">
          <span className="sort-label">Sort:</span>
          <button className={`sort-btn${sortBy === 'mentions' ? ' active' : ''}`} onClick={() => setSortBy('mentions')}>Mentions</button>
          <button className={`sort-btn${sortBy === 'first' ? ' active' : ''}`} onClick={() => setSortBy('first')}>First Appearance</button>
          <button className={`sort-btn${sortBy === 'name' ? ' active' : ''}`} onClick={() => setSortBy('name')}>Name</button>
        </div>
      )}

      {/* Character grid */}
      <div className="codex-grid">
        {addingNew && (
          <CodexCharacterForm
            char={null}
            onSave={c => { addCharacter(c); setAddingNew(false) }}
            onCancel={() => setAddingNew(false)}
          />
        )}
        {sortedCharacters.map(char =>
          editingId === char.id ? (
            <CodexCharacterForm
              key={char.id}
              char={char}
              onSave={c => { updateCharacter(c); setEditingId(null) }}
              onCancel={() => setEditingId(null)}
            />
          ) : (
            <CodexCharacterCard
              key={char.id}
              char={char}
              onEdit={() => setEditingId(char.id)}
              onDelete={() => deleteCharacter(char.id)}
              getChapterName={getChapterName}
              mergeMode={mergeMode}
              selected={selectedIds.has(char.id)}
              onToggleSelect={() => toggleSelect(char.id)}
            />
          )
        )}
      </div>

      {/* Empty state */}
      {characters.length === 0 && !addingNew && (
        <div className="codex-empty-state">
          <p>No characters yet.</p>
          <p className="hint">Click "Re-Index Book" to auto-detect characters from your manuscript, or add them manually.</p>
        </div>
      )}

      {/* Delete confirmation dialog */}
      {showDeleteConfirm && (
        <div className="dialog-overlay" onClick={() => setShowDeleteConfirm(false)}>
          <div className="dialog" onClick={e => e.stopPropagation()}>
            <div className="dialog-title">Delete All Auto-Detected Characters?</div>
            <p style={{ margin: '8px 0 16px', fontSize: 12, color: 'var(--text-secondary)' }}>
              This will remove {autoCount} auto-detected characters.
              Manually added characters will not be affected.
            </p>
            <div className="dialog-actions">
              <button className="ai-link-btn" onClick={() => setShowDeleteConfirm(false)}>Cancel</button>
              <button
                className="ai-run-btn"
                style={{ background: '#E06C75', borderColor: '#E06C75' }}
                onClick={handleDeleteAllAutoDetected}
              >
                Delete {autoCount} Characters
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Merge dialog */}
      {showMergeDialog && selectedCharacters.length >= 2 && (
        <div className="dialog-overlay" onClick={() => setShowMergeDialog(false)}>
          <div className="dialog" onClick={e => e.stopPropagation()}>
            <div className="dialog-title">Select Primary Name</div>
            <p style={{ margin: '4px 0 12px', fontSize: 11, color: 'var(--text-secondary)' }}>
              Other names will become aliases of the selected character.
            </p>
            <div className="merge-options">
              {selectedCharacters.map(char => (
                <button key={char.id} className="merge-option" onClick={() => handleMerge(char.id)}>
                  <span className="merge-option-name">{char.name}</span>
                  {char.mention_count !== undefined && <span className="merge-option-count">({char.mention_count} mentions)</span>}
                </button>
              ))}
            </div>
            <div style={{ marginTop: 12 }}>
              <button className="ai-link-btn" onClick={() => setShowMergeDialog(false)}>Cancel</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// Character card for the codex grid
function CodexCharacterCard({ char, onEdit, onDelete, getChapterName, mergeMode, selected, onToggleSelect }: {
  char: Character
  onEdit: () => void
  onDelete: () => void
  getChapterName: (index: number) => string
  mergeMode?: boolean
  selected?: boolean
  onToggleSelect?: () => void
}) {
  const [expanded, setExpanded] = useState(false)

  const handleClick = () => {
    if (mergeMode && onToggleSelect) {
      onToggleSelect()
    } else {
      setExpanded(v => !v)
    }
  }

  return (
    <div
      className={`codex-card${char.is_auto_detected ? ' auto-detected' : ''}${mergeMode ? ' merge-mode' : ''}${selected ? ' selected' : ''}`}
      onClick={handleClick}
    >
      {/* Merge checkbox */}
      {mergeMode && (
        <div className="codex-card-checkbox">
          <input type="checkbox" checked={selected} onChange={() => {}} />
        </div>
      )}

      <div className="codex-card-header">
        <span className="codex-card-name">{char.name || 'Unnamed'}</span>
        <span className={`character-role ${char.role}`}>{char.role}</span>
        {char.is_auto_detected && <span className="character-auto-badge" title="Auto-detected">auto</span>}
      </div>

      {/* Stats */}
      {(char.mention_count !== undefined || char.first_chapter !== undefined) && (
        <div className="codex-card-stats">
          {char.mention_count !== undefined && (
            <span className="codex-stat">{char.mention_count} mentions</span>
          )}
          {char.first_chapter !== undefined && (
            <span className="codex-stat">First: {getChapterName(char.first_chapter)}</span>
          )}
        </div>
      )}

      {char.description && <div className="codex-card-desc">{char.description}</div>}

      {/* Attributes */}
      {char.attributes && Object.keys(char.attributes).length > 0 && (
        <div className="codex-card-attributes">
          {Object.entries(char.attributes).map(([key, val]) => (
            <span key={key} className="codex-attribute">
              {key.replace('_', ' ')}: {val}
            </span>
          ))}
        </div>
      )}

      {/* Expanded details */}
      {expanded && !mergeMode && (
        <div className="codex-card-expanded">
          {/* Chapter appearances */}
          {char.chapter_mentions && Object.keys(char.chapter_mentions).length > 0 && (
            <div className="codex-chapters">
              <span className="codex-chapters-label">Appears in:</span>
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
            <div className="codex-field">
              <span className="codex-field-label">Also known as:</span>
              <span>{char.aliases.join(', ')}</span>
            </div>
          )}

          {/* Other fields */}
          {char.appearance && (
            <div className="codex-field">
              <span className="codex-field-label">Appearance</span>
              <span>{char.appearance}</span>
            </div>
          )}
          {char.personality && (
            <div className="codex-field">
              <span className="codex-field-label">Personality</span>
              <span>{char.personality}</span>
            </div>
          )}
          {char.motivation && (
            <div className="codex-field">
              <span className="codex-field-label">Motivation</span>
              <span>{char.motivation}</span>
            </div>
          )}
          {char.notes && (
            <div className="codex-field">
              <span className="codex-field-label">Notes</span>
              <span>{char.notes}</span>
            </div>
          )}

          <div className="codex-card-actions">
            <button className="tool-card-btn secondary" onClick={e => { e.stopPropagation(); onEdit() }}>Edit</button>
            <button className="tool-card-btn secondary danger" onClick={e => { e.stopPropagation(); onDelete() }}>Delete</button>
          </div>
        </div>
      )}
    </div>
  )
}

// Character form for the codex
function CodexCharacterForm({ char, onSave, onCancel }: {
  char: Character | null
  onSave: (c: Character) => void
  onCancel: () => void
}) {
  const [name, setName] = useState(char?.name ?? '')
  const [role, setRole] = useState<CharacterRole | string>(char?.role ?? 'supporting')
  const [desc, setDesc] = useState(char?.description ?? '')
  const [app, setApp] = useState(char?.appearance ?? '')
  const [persona, setPersona] = useState(char?.personality ?? '')
  const [motiv, setMotiv] = useState(char?.motivation ?? '')
  const [notes, setNotes] = useState(char?.notes ?? '')
  const [aliases, setAliases] = useState<string[]>(char?.aliases ?? [])
  const [newAlias, setNewAlias] = useState('')

  const addAlias = () => {
    const trimmed = newAlias.trim()
    if (trimmed && !aliases.includes(trimmed) && trimmed !== name.trim()) {
      setAliases([...aliases, trimmed])
      setNewAlias('')
    }
  }

  const removeAlias = (alias: string) => {
    setAliases(aliases.filter(a => a !== alias))
  }

  const handleSave = () => {
    if (name.trim()) {
      onSave({
        id: char?.id ?? genId(),
        name: name.trim(),
        role: role as CharacterRole,
        description: desc,
        appearance: app,
        personality: persona,
        motivation: motiv,
        notes,
        // Preserve auto-detection data if editing
        is_auto_detected: char?.is_auto_detected,
        aliases: aliases.length > 0 ? aliases : undefined,
        mention_count: char?.mention_count,
        first_chapter: char?.first_chapter,
        chapter_mentions: char?.chapter_mentions,
        attributes: char?.attributes,
      })
    }
  }

  return (
    <div className="codex-card codex-form" onClick={e => e.stopPropagation()}>
      <div className="codex-form-title">{char ? 'Edit Character' : 'New Character'}</div>

      <div className="codex-form-row">
        <div className="dialog-field" style={{ margin: 0, flex: 1 }}>
          <label className="dialog-label">Name</label>
          <input className="dialog-input" value={name} onChange={e => setName(e.target.value)} placeholder="Character name" autoFocus />
        </div>
        <div className="dialog-field" style={{ margin: 0, width: 120 }}>
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

      <div className="dialog-field" style={{ marginTop: 8 }}>
        <label className="dialog-label">Description</label>
        <textarea className="dialog-input" rows={2} value={desc} onChange={e => setDesc(e.target.value)} placeholder="Brief overview" style={{ resize: 'vertical' }} />
      </div>

      <div className="dialog-field" style={{ marginTop: 6 }}>
        <label className="dialog-label">Appearance</label>
        <textarea className="dialog-input" rows={2} value={app} onChange={e => setApp(e.target.value)} placeholder="Physical description" style={{ resize: 'vertical' }} />
      </div>

      <div className="codex-form-row" style={{ marginTop: 6 }}>
        <div className="dialog-field" style={{ margin: 0, flex: 1 }}>
          <label className="dialog-label">Personality</label>
          <textarea className="dialog-input" rows={2} value={persona} onChange={e => setPersona(e.target.value)} placeholder="Traits, quirks" style={{ resize: 'vertical' }} />
        </div>
        <div className="dialog-field" style={{ margin: 0, flex: 1 }}>
          <label className="dialog-label">Motivation</label>
          <textarea className="dialog-input" rows={2} value={motiv} onChange={e => setMotiv(e.target.value)} placeholder="Goals, fears" style={{ resize: 'vertical' }} />
        </div>
      </div>

      <div className="dialog-field" style={{ marginTop: 6 }}>
        <label className="dialog-label">Notes</label>
        <textarea className="dialog-input" rows={2} value={notes} onChange={e => setNotes(e.target.value)} placeholder="Arc beats, backstory" style={{ resize: 'vertical' }} />
      </div>

      {/* Aliases / Nicknames */}
      <div className="dialog-field" style={{ marginTop: 8 }}>
        <label className="dialog-label">Aliases / Nicknames</label>
        <p className="codex-alias-hint">Add alternate names so mentions are counted together (e.g., "Hanlon" for "Detective Hanlon")</p>
        {aliases.length > 0 && (
          <div className="codex-alias-list">
            {aliases.map(alias => (
              <span key={alias} className="codex-alias-tag">
                {alias}
                <button className="codex-alias-remove" onClick={() => removeAlias(alias)} title="Remove alias">&times;</button>
              </span>
            ))}
          </div>
        )}
        <div className="codex-alias-add">
          <input
            className="dialog-input"
            value={newAlias}
            onChange={e => setNewAlias(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); addAlias() } }}
            placeholder="Add nickname..."
            style={{ flex: 1 }}
          />
          <button className="tool-card-btn secondary" onClick={addAlias} disabled={!newAlias.trim()}>Add</button>
        </div>
      </div>

      <div className="codex-form-actions">
        <button className="ai-run-btn" onClick={handleSave} disabled={!name.trim()}>
          {char ? 'Save Changes' : 'Add Character'}
        </button>
        <button className="ai-link-btn" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}
