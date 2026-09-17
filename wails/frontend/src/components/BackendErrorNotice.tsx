// When the Go side fails, say so.
//
// Wails already recovers a panic in a bound method: its dispatcher catches it
// and rejects the promise instead of taking the window down. What it cannot do
// is tell the writer. An unhandled rejection reaches nobody, the component
// awaiting the call never updates, and the screen sits there looking frozen —
// which is the "blank UI with no explanation" this replaces.
//
// A React error boundary is no help here. Those catch errors thrown during
// render; a rejected promise from a binding is neither, and goes straight past
// them. So this listens for the rejection itself, at the window.
//
// One notice, not a queue. A failing call is usually a failing call repeated —
// a retried save, a re-render that asks again — and stacking twelve copies of
// the same message helps nobody. Repeats of the same text are counted instead.

import { useEffect, useState } from 'react'
import { failureMessage } from './backendError'

interface Failure {
  message: string
  /** How many times this same message has arrived. */
  count: number
}

export default function BackendErrorNotice() {
  const [failure, setFailure] = useState<Failure | null>(null)

  useEffect(() => {
    function onRejection(event: PromiseRejectionEvent) {
      const message = failureMessage(event.reason)
      // Leave the console entry alone. It is the only record with a stack,
      // and it is what a bug report needs.
      console.error('[backend]', message)
      setFailure(previous =>
        previous && previous.message === message
          ? { message, count: previous.count + 1 }
          : { message, count: 1 },
      )
    }
    window.addEventListener('unhandledrejection', onRejection)
    return () => window.removeEventListener('unhandledrejection', onRejection)
  }, [])

  if (!failure) return null

  return (
    <div className="be-notice" role="alert">
      <div className="be-notice-body">
        <strong>Draftline could not finish that.</strong>
        <p>{failure.message}</p>
        <p className="be-notice-reassure">
          Your manuscript is untouched. If it keeps happening, close the book and reopen it.
        </p>
      </div>
      {failure.count > 1 ? <span className="be-notice-count">×{failure.count}</span> : null}
      <button
        type="button"
        className="be-notice-close"
        aria-label="Dismiss"
        onClick={() => setFailure(null)}
      >
        ✕
      </button>
    </div>
  )
}
