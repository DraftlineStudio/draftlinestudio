import { useBookStore } from '../../store/bookStore'

export default function UnsavedChangesDialog() {
  const { dialogs, closeUnsavedWarning, saveAndProceed, discardAndProceed } = useBookStore()
  const action = dialogs.pendingAction === 'new' ? 'creating a new book' : 'opening another book'

  return (
    <div className="dialog-overlay">
      <div className="dialog">
        <div className="dialog-title">Unsaved Changes</div>
        <p className="dialog-body">
          You have unsaved changes. Save before {action}?
        </p>
        <div className="dialog-actions">
          <button className="dialog-btn" onClick={closeUnsavedWarning}>Cancel</button>
          <button className="dialog-btn" onClick={discardAndProceed}>Discard</button>
          <button className="dialog-btn primary" onClick={saveAndProceed}>Save</button>
        </div>
      </div>
    </div>
  )
}
