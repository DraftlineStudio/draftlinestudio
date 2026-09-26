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
        <div className="dialog-title">This book may already be open elsewhere</div>
        <p className="dialog-body">
          This book may already be open on another workstation or by another
          person. We can&rsquo;t open it here at the same time because that could
          cause conflicting saves or overwrite someone else&rsquo;s work.
        </p>
        <p className="dialog-body">
          You can open a separate copy on this machine and manually merge your
          changes later, or you can request to take over the project from {where}.
        </p>
        <p className="dialog-body">
          If no one responds within 30 seconds, this machine will take over
          automatically after confirming it has synced the most up-to-date copy
          of the project file.
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
