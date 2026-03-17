import { useState } from 'react'
import { useBookStore } from '../../store/bookStore'
import type { Section } from '../../types/draftline'
import { FRONT_MATTER_TYPES, BODY_TYPES, BACK_MATTER_TYPES } from '../../types/draftline'

interface Props {
  section: Section
}

function getTypesForSection(section: Section): string[] {
  if (section === 'front_matter') return FRONT_MATTER_TYPES
  if (section === 'body') return BODY_TYPES
  if (section === 'back_matter') return BACK_MATTER_TYPES
  return []
}

function getSectionLabel(section: Section): string {
  if (section === 'front_matter') return 'Front Matter'
  if (section === 'body') return 'Body'
  if (section === 'back_matter') return 'Back Matter'
  return ''
}

export default function NewChapterDialog({ section }: Props) {
  const { addChapter, closeNewChapterDialog } = useBookStore()
  const types = getTypesForSection(section)
  const [type, setType] = useState(types[0] || 'Chapter')
  const [title, setTitle] = useState('')

  function handleAdd() {
    const finalTitle = title.trim() || type
    addChapter(section, { title: finalTitle, type, content: '<p></p>' })
    closeNewChapterDialog()
  }

  function handleKey(e: React.KeyboardEvent) {
    if (e.key === 'Enter') handleAdd()
    if (e.key === 'Escape') closeNewChapterDialog()
  }

  return (
    <div className="dialog-overlay">
      <div className="dialog">
        <div className="dialog-title">New {getSectionLabel(section)} Item</div>

        <div className="dialog-field">
          <label className="dialog-label">Type</label>
          <select
            className="dialog-select"
            value={type}
            onChange={(e) => setType(e.target.value)}
          >
            {types.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
        </div>

        <div className="dialog-field">
          <label className="dialog-label">Title <span style={{ color: 'var(--text-muted)' }}>(optional — defaults to type)</span></label>
          <input
            className="dialog-input"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onKeyDown={handleKey}
            placeholder={type}
            autoFocus
          />
        </div>

        <div className="dialog-actions">
          <button className="dialog-btn" onClick={closeNewChapterDialog}>Cancel</button>
          <button className="dialog-btn primary" onClick={handleAdd}>Add</button>
        </div>
      </div>
    </div>
  )
}
