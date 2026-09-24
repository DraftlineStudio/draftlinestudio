import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../store/bookStore'

// Asking for the book is the recommendation now: the other device saves, proves
// what it saved, and lets go, which is the only route that ends with one
// current copy. Opening a copy is still the answer when it will not answer.
export default function BookLockDialog() {
  const { dialogs, askDeviceForBook, openBookAnyway, openBookAsCopy, cancelBookLockWarning } = useBookStore(useShallow(s => ({ dialogs: s.dialogs, askDeviceForBook: s.askDeviceForBook, openBookAnyway: s.openBookAnyway, openBookAsCopy: s.openBookAsCopy, cancelBookLockWarning: s.cancelBookLockWarning })))
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
          Asking for it is the clean way: {where} saves the book, lets go of it, and
          tells Draftline here what it saved, so this machine can be sure it has that
          version and not an older copy the sync client has yet to replace.
        </p>
        <p className="dialog-body">
          If it does not answer, open a copy. Whichever device saves last wins
          otherwise, and the other one&rsquo;s work is replaced.
        </p>
        <div className="dialog-actions">
          <button className="dialog-btn" onClick={cancelBookLockWarning}>Cancel</button>
          <button className="dialog-btn" onClick={openBookAnyway}>Open Anyway</button>
          <button className="dialog-btn" onClick={openBookAsCopy}>Open a Copy</button>
          <button className="dialog-btn primary" onClick={askDeviceForBook}>
            Ask {where} for it
          </button>
        </div>
      </div>
    </div>
  )
}
