import { useShallow } from 'zustand/react/shallow'
import { useMemo, useState } from 'react'
import { useBookStore } from '../../store/bookStore'
import { useAppStore } from '../../store/appStore'
import type { Section } from '../../types/draftline'
import { FRONT_MATTER_TYPES, BODY_TYPES, BACK_MATTER_TYPES } from '../../types/draftline'
import { getSectionArray, nextChapterTitle } from '../../store/chapters'

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
  const { addChapter } = useBookStore(useShallow(s => ({ addChapter: s.addChapter })))
  const book = useBookStore(s => s.book)
  const closeNewChapterDialog = useAppStore(s => s.closeNewChapterDialog)
  const types = getTypesForSection(section)
  const [type, setType] = useState(types[0] || 'Chapter')
  const [title, setTitle] = useState('')
  // Typing takes the field over: the suggestion stops following the type
  // picker once the writer has said what they want, including to empty.
  const [typed, setTyped] = useState(false)

  const existing = useMemo(
    () => (book && section !== 'copyright' ? getSectionArray(book, section).map(item => item.title ?? '') : []),
    [book, section],
  )
  const value = typed ? title : nextChapterTitle(existing, type)

  function handleAdd() {
    const finalTitle = value.trim() || type
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
            value={value}
            onChange={(e) => { setTyped(true); setTitle(e.target.value) }}
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
