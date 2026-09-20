import { useBookStore } from '../../store/bookStore'

// Shown when a book about to be opened carries a claim from another device.
//
// The wording hedges on purpose and should stay that way. The claim reaches
// this machine through whatever syncs the author's folder, which can be
// minutes behind, so "may be open" is the strongest thing that is true. See
// internal/booklock.
//
// Opening a copy is offered first and styled as the recommendation, because it
// is the only choice here that cannot lose anybody's afternoon.
export default function BookLockDialog() {
  const { dialogs, openBookAnyway, openBookAsCopy, cancelBookLockWarning } = useBookStore()
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
