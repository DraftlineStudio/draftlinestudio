import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../store/bookStore'

// Asking for the book is the clean route: the other device saves, proves what it
// saved, and lets go, which is the only ending with one current copy. A copy is
// the answer when it will not answer.
//
// There is deliberately no way to force past a live claim from here. It looked
// like the decisive choice and was the only one that could quietly lose a
// chapter: it opens whatever file is on this machine, which during a sync is the
// copy from before the other machine's last save — complete, openable, and
// wrong. Both remaining answers are safe, so the dangerous one is not offered.
export default function BookLockDialog() {
  const { dialogs, askDeviceForBook, openBookAsCopy, cancelBookLockWarning } = useBookStore(useShallow(s => ({ dialogs: s.dialogs, askDeviceForBook: s.askDeviceForBook, openBookAsCopy: s.openBookAsCopy, cancelBookLockWarning: s.cancelBookLockWarning })))
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
          Asking for it is the clean way: that computer saves the book, lets go of
          it, and says what it saved, so this one can be sure it has that version
          and not an older copy the sync client has yet to replace.
        </p>
        <p className="dialog-body">
          If it does not answer, open a copy. A copy is a book of its own and
          cannot overwrite anybody, and Compare with Current in AI Studio will
          show you what the two versions differ on when you reconcile them.
        </p>
        {/* The device name is in the sentence above, not in a button: a
            hostname can be any length, and a button is not the place to find
            that out. */}
        <div className="dialog-actions">
          <button className="dialog-btn" onClick={cancelBookLockWarning}>Cancel</button>
          <button className="dialog-btn" onClick={openBookAsCopy}>Open a Copy</button>
          <button className="dialog-btn primary" onClick={askDeviceForBook}>Ask For It</button>
        </div>
      </div>
    </div>
  )
}
