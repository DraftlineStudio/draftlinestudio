import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../store/bookStore'

// What happened when asking for a book did not end in the book.
//
// It exists because the status bar does not. A handover that fails leaves the
// writer on the launch screen, which has no status bar, so a refusal announced
// there is announced to nobody: the overlay disappears and the screen goes back
// to how it was, which reads as nothing having happened at all.
export default function HandoverResultDialog() {
  const { report, openCopy, dismiss } = useBookStore(useShallow(s => ({
    report: s.dialogs.handoverReport,
    openCopy: s.openCopyAfterHandover,
    dismiss: s.dismissHandoverReport,
  })))
  if (!report) return null

  return (
    <div className="dialog-overlay">
      <div className="dialog">
        <div className="dialog-title">{report.title}</div>
        <p className="dialog-body">{report.message}</p>
        {report.offerCopy && (
          <p className="dialog-body">
            A copy opens straight away and cannot overwrite anybody: it is a
            separate book, and you can reconcile the two later.
          </p>
        )}
        <div className="dialog-actions">
          <button className="dialog-btn" onClick={dismiss}>Close</button>
          {report.offerCopy && (
            <button className="dialog-btn primary" onClick={openCopy}>Open a Copy</button>
          )}
        </div>
      </div>
    </div>
  )
}
