// PlotWalker - Beats, Foreshadowing, Knowledge Matrix, Story Analysis

import { useState, useMemo } from 'react'
import { useBookStore, getGlobalChapterIndex } from '../../../store/bookStore'
import type { Beat, BeatType, ForeshadowingItem, ForeshadowingStatus, SecretInfo, KnowledgeEntry } from '../../../types/draftline'
import { genId } from '../types'
import { BEAT_TYPES } from '../constants'

// ── Beat Sheet ───────────────────────────────────────────────────────────────

export function BeatsSection() {
  const { book, addBeat, updateBeat, deleteBeat, currentSection, currentIndex } = useBookStore()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [addingNew, setAddingNew] = useState(false)

  if (!book) return <div className="tool-empty-state">Open a project first.</div>

  const beats = book.beat_sheet?.beats ?? []
  const globalChapterIndex = getGlobalChapterIndex(book, currentSection, currentIndex)
  const allChapters = [...book.front_matter, ...book.body, ...book.back_matter]

  // Group beats by chapter
  const beatsByChapter: Record<number, Beat[]> = {}
  beats.forEach(beat => {
    if (!beatsByChapter[beat.chapter_index]) beatsByChapter[beat.chapter_index] = []
    beatsByChapter[beat.chapter_index].push(beat)
  })

  const handleAddBeat = (chapterIdx: number, beatType: BeatType | string, description: string, notes: string) => {
    addBeat({
      id: genId(),
      chapter_index: chapterIdx,
      beat_type: beatType,
      description,
      notes: notes || undefined,
    })
    setAddingNew(false)
  }

  const handleUpdateBeat = (beat: Beat, updates: Partial<Beat>) => {
    updateBeat({ ...beat, ...updates })
    setEditingId(null)
  }

  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Beat Sheet</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>
        Track story beats per chapter. Based on Save the Cat! methodology.
      </p>

      {addingNew ? (
        <BeatForm
          chapters={allChapters}
          currentChapter={globalChapterIndex}
          onSubmit={handleAddBeat}
          onCancel={() => setAddingNew(false)}
        />
      ) : (
        <button className="bible-add-btn" onClick={() => setAddingNew(true)}>+ Add Beat</button>
      )}

      <div className="beats-list" style={{ marginTop: 12 }}>
        {Object.keys(beatsByChapter).sort((a, b) => Number(a) - Number(b)).map(chIdxStr => {
          const chIdx = Number(chIdxStr)
          const chapterBeats = beatsByChapter[chIdx]
          const chapter = allChapters[chIdx]
          return (
            <div key={chIdx} className="beat-chapter-group">
              <div className="beat-chapter-header">
                Ch. {chIdx + 1}: {chapter?.title || 'Untitled'}
              </div>
              {chapterBeats.map(beat => (
                editingId === beat.id ? (
                  <BeatForm
                    key={beat.id}
                    chapters={allChapters}
                    currentChapter={beat.chapter_index}
                    initialBeat={beat}
                    onSubmit={(chIdx, beatType, desc, notes) => handleUpdateBeat(beat, { chapter_index: chIdx, beat_type: beatType, description: desc, notes: notes || undefined })}
                    onCancel={() => setEditingId(null)}
                  />
                ) : (
                  <div key={beat.id} className="beat-card">
                    <div className="beat-card-header">
                      <span className="beat-type-badge">{BEAT_TYPES.find(t => t.value === beat.beat_type)?.label || beat.beat_type}</span>
                      <div className="beat-card-actions">
                        <button className="beat-edit-btn" onClick={() => setEditingId(beat.id)}>Edit</button>
                        <button className="beat-delete-btn" onClick={() => deleteBeat(beat.id)}>Delete</button>
                      </div>
                    </div>
                    <p className="beat-description">{beat.description}</p>
                    {beat.notes && <p className="beat-notes">{beat.notes}</p>}
                  </div>
                )
              ))}
            </div>
          )
        })}
        {beats.length === 0 && !addingNew && (
          <div className="tool-empty-state" style={{ marginTop: 12 }}>No beats defined yet.</div>
        )}
      </div>
    </div>
  )
}

function BeatForm({ chapters, currentChapter, initialBeat, onSubmit, onCancel }: {
  chapters: { title: string }[]
  currentChapter: number
  initialBeat?: Beat
  onSubmit: (chapterIdx: number, beatType: BeatType | string, description: string, notes: string) => void
  onCancel: () => void
}) {
  const [chapterIdx, setChapterIdx] = useState(initialBeat?.chapter_index ?? currentChapter)
  const [beatType, setBeatType] = useState<BeatType | string>(initialBeat?.beat_type ?? 'catalyst')
  const [description, setDescription] = useState(initialBeat?.description ?? '')
  const [notes, setNotes] = useState(initialBeat?.notes ?? '')

  return (
    <div className="beat-form">
      <div className="beat-form-row">
        <label>Chapter:</label>
        <select value={chapterIdx} onChange={e => setChapterIdx(Number(e.target.value))}>
          {chapters.map((ch, i) => (
            <option key={i} value={i}>Ch. {i + 1}: {ch.title || 'Untitled'}</option>
          ))}
        </select>
      </div>
      <div className="beat-form-row">
        <label>Beat Type:</label>
        <select value={beatType} onChange={e => setBeatType(e.target.value)}>
          {BEAT_TYPES.map(bt => (
            <option key={bt.value} value={bt.value}>{bt.label}</option>
          ))}
        </select>
      </div>
      <div className="beat-form-row">
        <label>Description:</label>
        <textarea
          value={description}
          onChange={e => setDescription(e.target.value)}
          placeholder="What happens in this beat..."
          rows={2}
        />
      </div>
      <div className="beat-form-row">
        <label>Notes:</label>
        <textarea
          value={notes}
          onChange={e => setNotes(e.target.value)}
          placeholder="Optional notes..."
          rows={1}
        />
      </div>
      <div className="beat-form-actions">
        <button className="bible-save-btn" onClick={() => onSubmit(chapterIdx, beatType, description, notes)}>
          {initialBeat ? 'Update' : 'Add'}
        </button>
        <button className="bible-cancel-btn" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}

// ── Foreshadowing Ledger ─────────────────────────────────────────────────────

export function ForeshadowingSection() {
  const { book, addForeshadowingItem, updateForeshadowingItem, deleteForeshadowingItem } = useBookStore()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [addingNew, setAddingNew] = useState(false)

  if (!book) return <div className="tool-empty-state">Open a project first.</div>

  const items = book.foreshadowing?.items ?? []
  const allChapters = [...book.front_matter, ...book.body, ...book.back_matter]

  const handleAdd = (item: Omit<ForeshadowingItem, 'id'>) => {
    addForeshadowingItem({ ...item, id: genId() })
    setAddingNew(false)
  }

  const handleUpdate = (item: ForeshadowingItem) => {
    updateForeshadowingItem(item)
    setEditingId(null)
  }

  const getStatusColor = (status: ForeshadowingStatus | string) => {
    switch (status) {
      case 'planted': return '#fb923c'
      case 'active': return '#3b82f6'
      case 'resolved': return '#4ade80'
      default: return '#888888'
    }
  }

  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Foreshadowing Ledger</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>
        Track plant → reinforce → payoff chains across chapters.
      </p>

      {addingNew ? (
        <ForeshadowingForm
          chapters={allChapters}
          onSubmit={handleAdd}
          onCancel={() => setAddingNew(false)}
        />
      ) : (
        <button className="bible-add-btn" onClick={() => setAddingNew(true)}>+ Add Foreshadowing</button>
      )}

      <div className="foreshadow-list" style={{ marginTop: 12 }}>
        {items.map(item => (
          editingId === item.id ? (
            <ForeshadowingForm
              key={item.id}
              chapters={allChapters}
              initialItem={item}
              onSubmit={(i) => handleUpdate(i as ForeshadowingItem)}
              onCancel={() => setEditingId(null)}
            />
          ) : (
            <div key={item.id} className="foreshadow-card">
              <div className="foreshadow-card-header">
                <span className="foreshadow-name">{item.name}</span>
                <span className="foreshadow-status" style={{ background: `${getStatusColor(item.status)}22`, color: getStatusColor(item.status) }}>
                  {item.status}
                </span>
              </div>
              <p className="foreshadow-description">{item.description}</p>
              <div className="foreshadow-chapters">
                <span><strong>Plant:</strong> Ch. {item.plant_chapter + 1}</span>
                {(item.reinforce_chapters?.length ?? 0) > 0 && (
                  <span><strong>Reinforce:</strong> {item.reinforce_chapters?.map(c => `Ch. ${c + 1}`).join(', ')}</span>
                )}
                {item.payoff_chapter !== undefined && (
                  <span><strong>Payoff:</strong> Ch. {item.payoff_chapter + 1}</span>
                )}
              </div>
              {item.notes && <p className="foreshadow-notes">{item.notes}</p>}
              <div className="foreshadow-card-actions">
                <button className="beat-edit-btn" onClick={() => setEditingId(item.id)}>Edit</button>
                <button className="beat-delete-btn" onClick={() => deleteForeshadowingItem(item.id)}>Delete</button>
              </div>
            </div>
          )
        ))}
        {items.length === 0 && !addingNew && (
          <div className="tool-empty-state" style={{ marginTop: 12 }}>No foreshadowing items yet.</div>
        )}
      </div>
    </div>
  )
}

function ForeshadowingForm({ chapters, initialItem, onSubmit, onCancel }: {
  chapters: { title: string }[]
  initialItem?: ForeshadowingItem
  onSubmit: (item: ForeshadowingItem | Omit<ForeshadowingItem, 'id'>) => void
  onCancel: () => void
}) {
  const [name, setName] = useState(initialItem?.name ?? '')
  const [description, setDescription] = useState(initialItem?.description ?? '')
  const [plantChapter, setPlantChapter] = useState(initialItem?.plant_chapter ?? 0)
  const [reinforceChapters, setReinforceChapters] = useState(initialItem?.reinforce_chapters?.join(', ') ?? '')
  const [payoffChapter, setPayoffChapter] = useState<string>(initialItem?.payoff_chapter?.toString() ?? '')
  const [status, setStatus] = useState<ForeshadowingStatus | string>(initialItem?.status ?? 'planted')
  const [notes, setNotes] = useState(initialItem?.notes ?? '')

  const handleSubmit = () => {
    const reinforce = reinforceChapters
      .split(',')
      .map(s => parseInt(s.trim()) - 1)
      .filter(n => !isNaN(n) && n >= 0)
    const payoff = payoffChapter ? parseInt(payoffChapter) - 1 : undefined

    const item = {
      ...(initialItem ? { id: initialItem.id } : {}),
      name,
      description,
      plant_chapter: plantChapter,
      reinforce_chapters: reinforce,
      payoff_chapter: payoff !== undefined && payoff >= 0 ? payoff : undefined,
      status,
      notes: notes || undefined,
    }
    onSubmit(item as ForeshadowingItem)
  }

  return (
    <div className="foreshadow-form">
      <div className="beat-form-row">
        <label>Name:</label>
        <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="Short label..." />
      </div>
      <div className="beat-form-row">
        <label>Description:</label>
        <textarea value={description} onChange={e => setDescription(e.target.value)} placeholder="What is being foreshadowed..." rows={2} />
      </div>
      <div className="beat-form-row">
        <label>Plant Chapter:</label>
        <select value={plantChapter} onChange={e => setPlantChapter(Number(e.target.value))}>
          {chapters.map((ch, i) => (
            <option key={i} value={i}>Ch. {i + 1}: {ch.title || 'Untitled'}</option>
          ))}
        </select>
      </div>
      <div className="beat-form-row">
        <label>Reinforce (comma-sep):</label>
        <input type="text" value={reinforceChapters} onChange={e => setReinforceChapters(e.target.value)} placeholder="e.g. 5, 8, 12" />
      </div>
      <div className="beat-form-row">
        <label>Payoff Chapter:</label>
        <input type="text" value={payoffChapter} onChange={e => setPayoffChapter(e.target.value)} placeholder="e.g. 15 (leave empty if unresolved)" />
      </div>
      <div className="beat-form-row">
        <label>Status:</label>
        <select value={status} onChange={e => setStatus(e.target.value as ForeshadowingStatus)}>
          <option value="planted">Planted</option>
          <option value="active">Active</option>
          <option value="resolved">Resolved</option>
        </select>
      </div>
      <div className="beat-form-row">
        <label>Notes:</label>
        <textarea value={notes} onChange={e => setNotes(e.target.value)} placeholder="Optional notes..." rows={1} />
      </div>
      <div className="beat-form-actions">
        <button className="bible-save-btn" onClick={handleSubmit}>{initialItem ? 'Update' : 'Add'}</button>
        <button className="bible-cancel-btn" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}

// ── Knowledge Matrix ─────────────────────────────────────────────────────────

export function KnowledgeSection() {
  const { book, addSecret, updateSecret, deleteSecret, setKnowledgeEntry } = useBookStore()
  const [addingSecret, setAddingSecret] = useState(false)
  const [editingSecretId, setEditingSecretId] = useState<string | null>(null)

  if (!book) return <div className="tool-empty-state">Open a project first.</div>

  const secrets = book.knowledge_matrix?.secrets ?? []
  const entries = book.knowledge_matrix?.entries ?? []
  const characters = book.story_bible?.characters ?? []

  // Build lookup: secretId -> characterId -> entry
  const entryLookup = useMemo(() => {
    const lookup: Record<string, Record<string, KnowledgeEntry>> = {}
    entries.forEach(e => {
      if (!lookup[e.secret_id]) lookup[e.secret_id] = {}
      lookup[e.secret_id][e.character_id] = e
    })
    return lookup
  }, [entries])

  const handleAddSecret = (name: string, description: string) => {
    addSecret({ id: genId(), name, description })
    setAddingSecret(false)
  }

  const handleUpdateSecret = (secret: SecretInfo) => {
    updateSecret(secret)
    setEditingSecretId(null)
  }

  const handleCellChange = (secretId: string, charId: string, field: 'learns' | 'suspected', value: string) => {
    const existing = entryLookup[secretId]?.[charId]
    const numVal = value ? parseInt(value) - 1 : undefined

    setKnowledgeEntry({
      secret_id: secretId,
      character_id: charId,
      learns_chapter: field === 'learns' ? (numVal !== undefined && numVal >= 0 ? numVal : undefined) : existing?.learns_chapter,
      suspected_chapter: field === 'suspected' ? (numVal !== undefined && numVal >= 0 ? numVal : undefined) : existing?.suspected_chapter,
    })
  }

  if (characters.length === 0) {
    return (
      <div>
        <div className="tool-label" style={{ marginBottom: 4 }}>Knowledge Matrix</div>
        <p className="settings-hint" style={{ marginBottom: 8 }}>
          Track which characters know which secrets, and when they learn them.
        </p>
        <div className="tool-empty-state" style={{ marginTop: 12 }}>
          Add characters first in the Characters tab.
        </div>
      </div>
    )
  }

  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Knowledge Matrix</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>
        Track which characters know which secrets. Enter chapter numbers.
      </p>

      {addingSecret ? (
        <SecretForm onSubmit={handleAddSecret} onCancel={() => setAddingSecret(false)} />
      ) : (
        <button className="bible-add-btn" onClick={() => setAddingSecret(true)}>+ Add Secret</button>
      )}

      {secrets.length > 0 && (
        <div className="knowledge-matrix" style={{ marginTop: 12 }}>
          <table>
            <thead>
              <tr>
                <th>Secret / Info</th>
                {characters.slice(0, 5).map(char => (
                  <th key={char.id} title={char.name}>{char.name.slice(0, 8)}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {secrets.map(secret => (
                <tr key={secret.id}>
                  <td className="knowledge-secret-cell">
                    {editingSecretId === secret.id ? (
                      <SecretForm
                        initialSecret={secret}
                        onSubmit={(name, desc) => handleUpdateSecret({ ...secret, name, description: desc })}
                        onCancel={() => setEditingSecretId(null)}
                        inline
                      />
                    ) : (
                      <div className="knowledge-secret-name">
                        <span title={secret.description}>{secret.name}</span>
                        <div className="knowledge-secret-actions">
                          <button onClick={() => setEditingSecretId(secret.id)}>Edit</button>
                          <button onClick={() => deleteSecret(secret.id)}>Del</button>
                        </div>
                      </div>
                    )}
                  </td>
                  {characters.slice(0, 5).map(char => {
                    const entry = entryLookup[secret.id]?.[char.id]
                    return (
                      <td key={char.id} className="knowledge-cell">
                        <input
                          type="text"
                          className="knowledge-cell-input"
                          placeholder="—"
                          value={entry?.learns_chapter !== undefined ? entry.learns_chapter + 1 : ''}
                          onChange={e => handleCellChange(secret.id, char.id, 'learns', e.target.value)}
                          title={`When ${char.name} learns this`}
                        />
                      </td>
                    )
                  })}
                </tr>
              ))}
            </tbody>
          </table>
          {characters.length > 5 && (
            <p className="settings-hint" style={{ marginTop: 8 }}>
              Showing first 5 characters.
            </p>
          )}
        </div>
      )}

      {secrets.length === 0 && !addingSecret && (
        <div className="tool-empty-state" style={{ marginTop: 12 }}>No secrets/info items yet.</div>
      )}
    </div>
  )
}

function SecretForm({ initialSecret, onSubmit, onCancel, inline }: {
  initialSecret?: SecretInfo
  onSubmit: (name: string, description: string) => void
  onCancel: () => void
  inline?: boolean
}) {
  const [name, setName] = useState(initialSecret?.name ?? '')
  const [description, setDescription] = useState(initialSecret?.description ?? '')

  if (inline) {
    return (
      <div className="secret-form-inline">
        <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="Name" />
        <button onClick={() => onSubmit(name, description)}>OK</button>
        <button onClick={onCancel}>X</button>
      </div>
    )
  }

  return (
    <div className="secret-form">
      <div className="beat-form-row">
        <label>Name:</label>
        <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="Secret/info name..." />
      </div>
      <div className="beat-form-row">
        <label>Description:</label>
        <textarea value={description} onChange={e => setDescription(e.target.value)} placeholder="What is this secret..." rows={2} />
      </div>
      <div className="beat-form-actions">
        <button className="bible-save-btn" onClick={() => onSubmit(name, description)}>{initialSecret ? 'Update' : 'Add'}</button>
        <button className="bible-cancel-btn" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}

// ── Issues / Story Analysis ─────────────────────────────────────────────────

export function IssuesSection() {
  const { book } = useBookStore()

  if (!book) {
    return <div className="tool-empty-state">Open a project to see analysis.</div>
  }

  return (
    <div className="issues-section">
      <div className="tool-empty-state">
        <p>Story analysis coming soon.</p>
        <p style={{ fontSize: '11px', opacity: 0.7, marginTop: '8px' }}>
          This panel will provide continuity checking and story structure analysis.
        </p>
      </div>
    </div>
  )
}
