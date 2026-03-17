import { useState } from 'react'
import { useBookStore } from '../../store/bookStore'

export default function MetadataDialog() {
  const { book, updateMetadata, closeMetadataDialog } = useBookStore()
  const meta = book?.metadata

  const [title, setTitle] = useState(meta?.title || '')
  const [author, setAuthor] = useState(meta?.author || '')
  const [isbn, setIsbn] = useState(meta?.isbn || '')
  const [publisher, setPublisher] = useState(meta?.publisher || '')

  function handleSave() {
    updateMetadata({ title, author, isbn, publisher })
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
          <label className="dialog-label">ISBN</label>
          <input
            className="dialog-input"
            value={isbn}
            onChange={(e) => setIsbn(e.target.value)}
            placeholder="ISBN"
          />
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
