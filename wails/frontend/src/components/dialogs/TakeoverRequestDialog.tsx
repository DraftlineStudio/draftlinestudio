import { useEffect, useRef, useState } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../store/bookStore'

// How long whoever is at this machine has to keep the book. It is short on
// purpose: the usual case is that nobody is here at all, the machine was left
// running, and the writer is at the other one waiting to work.
const SECONDS = 5

// The question this machine is asked when the writer's other device wants the
// book. Saying nothing hands it over, which is the right default for a desktop
// left on overnight — and the reason the other side re-checks that somebody is
// still waiting before it acts on the silence.
export default function TakeoverRequestDialog() {
  const { request, grant, refuse } = useBookStore(useShallow(s => ({
    request: s.dialogs.takeoverRequest,
    grant: s.grantBookToDevice,
    refuse: s.refuseToHandBookOver,
  })))
  const [left, setLeft] = useState(SECONDS)
  // The countdown must fire once. Without this a re-render on the last tick
  // could hand the book over twice.
  const handed = useRef(false)

  useEffect(() => {
    if (!request) {
      handed.current = false
      setLeft(SECONDS)
      return
    }
    const timer = setInterval(() => {
      setLeft(remaining => {
        if (remaining > 1) return remaining - 1
        clearInterval(timer)
        if (!handed.current) {
          handed.current = true
          void grant()
        }
        return 0
      })
    }, 1000)
    return () => clearInterval(timer)
  }, [request, grant])

  if (!request) return null
  const where = request.device || 'Another device'

  return (
    <div className="dialog-overlay">
      <div className="dialog">
        <div className="dialog-title">{where} is asking for this book</div>
        <p className="dialog-body">
          {request.message || `${where} wants to work on this book.`} Draftline will
          save it and hand it over in <span className="takeover-count">{left}</span>{' '}
          second{left === 1 ? '' : 's'}.
        </p>
        <p className="dialog-body">
          Keep it here if you are working in it. {where} will be told, and can open
          a copy instead.
        </p>
        <div className="dialog-actions">
          <button className="dialog-btn primary" onClick={() => { handed.current = true; void refuse() }}>
            Keep it here
          </button>
          <button className="dialog-btn" onClick={() => { handed.current = true; void grant() }}>
            Hand it over now
          </button>
        </div>
      </div>
    </div>
  )
}
