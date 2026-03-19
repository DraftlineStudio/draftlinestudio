import { useState } from 'react'
import { useAppStore } from '../../store/appStore'

export default function NewUniverseWizard() {
  const { setShowNewUniverse } = useAppStore()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')

  const handleClose = () => {
    setShowNewUniverse(false)
  }

  const handleCreate = () => {
    // TODO: Implement universe creation in Phase 2
    alert('Universe creation will be available in a future update. For now, please use "New Book" to create individual books.')
    handleClose()
  }

  return (
    <div className="dialog-backdrop" onClick={handleClose}>
      <div className="dialog new-universe-wizard" onClick={e => e.stopPropagation()}>
        <div className="dialog-header">
          <h2>New Universe</h2>
          <button className="dialog-close" onClick={handleClose}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div className="dialog-body">
          <div className="storiverse-badge">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <circle cx="12" cy="12" r="10" />
              <circle cx="12" cy="12" r="4" />
              <line x1="12" y1="2" x2="12" y2="8" />
              <line x1="12" y1="16" x2="12" y2="22" />
            </svg>
            <span>Storiverse Feature</span>
          </div>

          <p className="wizard-description">
            Universes let you manage multiple books in a shared world with cross-book continuity tracking,
            a unified story bible, and timeline visualization.
          </p>

          <div className="form-group">
            <label htmlFor="universe-name">Universe Name</label>
            <input
              id="universe-name"
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="e.g., The Emberfall Chronicles"
              autoFocus
            />
          </div>

          <div className="form-group">
            <label htmlFor="universe-description">Description (optional)</label>
            <textarea
              id="universe-description"
              value={description}
              onChange={e => setDescription(e.target.value)}
              placeholder="A brief description of your universe..."
              rows={3}
            />
          </div>

          <div className="coming-soon-notice">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="8" x2="12" y2="12" />
              <line x1="12" y1="16" x2="12.01" y2="16" />
            </svg>
            <span>Universe features are coming in Phase 2. For now, you can create individual books.</span>
          </div>
        </div>

        <div className="dialog-footer">
          <button className="btn btn-secondary" onClick={handleClose}>Cancel</button>
          <button className="btn btn-primary" onClick={handleCreate} disabled>
            Create Universe (Coming Soon)
          </button>
        </div>
      </div>
    </div>
  )
}
