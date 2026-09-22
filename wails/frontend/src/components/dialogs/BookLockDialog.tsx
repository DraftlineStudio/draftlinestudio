import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../store/bookStore'

// Opening a copy is the recommendation: it is the only choice that cannot lose
// anybody's work.
export default function BookLockDialog() {
  const { dialogs, openBookAnyway, openBookAsCopy, cancelBookLockWarning } = useBookStore(useShallow(s => ({ dialogs: s.dialogs, openBookAnyway: s.openBookAnyway, openBookAsCopy: s.openBookAsCopy, cancelBookLockWarning: s.cancelBookLockWarning })))
  const warning = dialogs.bookLockWarning
  if (!warning) return null

  const where = warning.info.device || 'another device'

  return (
    <div className="dialog-overlay">
      <div className="dialog">
        <div className="dialog-title">This book may be open elsewhere</div>
        <p className="dialog-body">
          {warning.info.message || `This book may be open on ${where}.`}
        </p>
        <p className="dialog-body">
          If it is, and the folder is synced by Dropbox, OneDrive, Proton Drive or
          similar, then whichever device saves last wins and the other one&rsquo;s
          work is replaced. Opening a copy avoids that entirely.
        </p>
        <div className="dialog-actions">
          <button className="dialog-btn" onClick={cancelBookLockWarning}>Cancel</button>
          <button className="dialog-btn" onClick={openBookAnyway}>Open Anyway</button>
          <button className="dialog-btn primary" onClick={openBookAsCopy}>Open a Copy</button>
        </div>
      </div>
    </div>
  )
}
