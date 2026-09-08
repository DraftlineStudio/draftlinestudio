import { useState } from 'react'
import { useBookStore } from '../../store/bookStore'
import { useAppStore } from '../../store/appStore'
import type { ISBNEntry } from '../../types/draftline'

const ISBN_FORMATS: { value: string; label: string }[] = [
  { value: '', label: 'Format…' },
  { value: 'hardcover', label: 'Hardcover' },
  { value: 'paperback', label: 'Paperback' },
  { value: 'ebook', label: 'eBook' },
  { value: 'audiobook', label: 'Audiobook' },
  { value: 'large_print', label: 'Large Print' },
  { value: 'other', label: 'Other' },
]

export default function MetadataDialog() {
  const { book, updateMetadata } = useBookStore()
  const closeMetadataDialog = useAppStore(s => s.closeMetadataDialog)
  const meta = book?.metadata

  const [title, setTitle] = useState(meta?.title || '')
  const [author, setAuthor] = useState(meta?.author || '')
  const [publisher, setPublisher] = useState(meta?.publisher || '')
  // A legacy book with only the single field seeds the list for editing.
  const [isbns, setIsbns] = useState<ISBNEntry[]>(() => {
    if (meta?.isbns?.length) return meta.isbns.map(e => ({ ...e }))
    if (meta?.isbn) return [{ format: '', value: meta.isbn }]
    return []
  })

  function setEntry(index: number, patch: Partial<ISBNEntry>) {
    setIsbns(list => list.map((entry, i) => (i === index ? { ...entry, ...patch } : entry)))
  }

  function removeEntry(index: number) {
    setIsbns(list => list.filter((_, i) => i !== index))
  }

  function handleSave() {
    const cleaned = isbns
      .map(entry => ({ format: entry.format, value: entry.value.trim() }))
      .filter(entry => entry.value !== '')
    // The legacy field mirrors the first entry for format compatibility.
    updateMetadata({ title, author, publisher, isbns: cleaned, isbn: cleaned[0]?.value ?? '' })
    closeMetadataDialog()
  }

  return (
    <div className="dialog-overlay">
      <div className="dialog">
        <div className="dialog-title">Book Metadata</div>

        <div className="dialog-field">
          <label className="dialog-label">Title</label>
          <input
            className="dialog-input"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Book title"
            autoFocus
          />
        </div>

        <div className="dialog-field">
          <label className="dialog-label">Author</label>
          <input
            className="dialog-input"
            value={author}
            onChange={(e) => setAuthor(e.target.value)}
            placeholder="Author name"
          />
        </div>

        <div className="dialog-field">
          <label className="dialog-label">Publisher</label>
          <input
            className="dialog-input"
            value={publisher}
            onChange={(e) => setPublisher(e.target.value)}
            placeholder="Publisher"
          />
        </div>

        <div className="dialog-field">
          <label className="dialog-label">ISBNs</label>
          {isbns.map((entry, index) => (
            <div className="metadata-isbn-row" key={index}>
              <select
                className="dialog-select metadata-isbn-format"
                value={entry.format}
                onChange={(e) => setEntry(index, { format: e.target.value })}
              >
                {ISBN_FORMATS.map(option => (
                  <option key={option.value} value={option.value}>{option.label}</option>
                ))}
              </select>
              <input
                className="dialog-input"
                value={entry.value}
                onChange={(e) => setEntry(index, { value: e.target.value })}
                placeholder="978-…"
              />
              <button
                className="dialog-btn metadata-isbn-remove"
                onClick={() => removeEntry(index)}
                title="Remove this ISBN"
              >
                ✕
              </button>
            </div>
          ))}
          <button
            className="dialog-btn metadata-isbn-add"
            onClick={() => setIsbns(list => [...list, { format: '', value: '' }])}
          >
            + Add ISBN
          </button>
          <div className="settings-hint">
            Each format — hardcover, paperback, eBook, audiobook — carries its own ISBN. The eBook ISBN is used as the EPUB identifier on export.
          </div>
        </div>

        {meta?.created && (
          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 8 }}>
            Created: {new Date(meta.created).toLocaleDateString()}
            {meta.modified && ` · Modified: ${new Date(meta.modified).toLocaleDateString()}`}
          </div>
        )}

        <div className="dialog-actions">
          <button className="dialog-btn" onClick={closeMetadataDialog}>Cancel</button>
          <button className="dialog-btn primary" onClick={handleSave}>Save</button>
        </div>
      </div>
    </div>
  )
}
