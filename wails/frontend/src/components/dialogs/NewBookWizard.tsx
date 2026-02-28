import { useState } from 'react'
import { useBookStore } from '../../store/bookStore'
import { useAppStore } from '../../store/appStore'

export default function NewBookWizard() {
  const { confirmNewBook, cancelNewBookWizard } = useBookStore()
  const { settings } = useAppStore()

  const [title, setTitle] = useState('')
  const [author, setAuthor] = useState(settings.default_author)
  const [publisher, setPublisher] = useState(settings.default_publisher)

  function handleCreate() {
    confirmNewBook(title.trim() || 'Untitled', author.trim(), publisher.trim())
  }

  function handleKey(e: React.KeyboardEvent) {
    if (e.key === 'Enter') handleCreate()
    if (e.key === 'Escape') cancelNewBookWizard()
  }

  return (
    <div className="dialog-overlay">
      <div className="dialog">
        <div className="dialog-title">New Book</div>

        <div className="dialog-field">
          <label className="dialog-label">Title</label>
          <input
            className="dialog-input"
            value={title}
            onChange={e => setTitle(e.target.value)}
            placeholder="Untitled"
            autoFocus
            onKeyDown={handleKey}
          />
        </div>

        <div className="dialog-field">
          <label className="dialog-label">Author</label>
          <input
            className="dialog-input"
            value={author}
            onChange={e => setAuthor(e.target.value)}
            placeholder="Author name"
            onKeyDown={handleKey}
          />
        </div>

        <div className="dialog-field">
          <label className="dialog-label">Publisher</label>
          <input
            className="dialog-input"
            value={publisher}
            onChange={e => setPublisher(e.target.value)}
            placeholder="Publisher (optional)"
            onKeyDown={handleKey}
          />
        </div>

        <div className="dialog-actions">
          <button className="dialog-btn" onClick={cancelNewBookWizard}>Cancel</button>
          <button className="dialog-btn primary" onClick={handleCreate}>Create</button>
        </div>
      </div>
    </div>
  )
}
