// Deleting an edition or a format.
//
// One confirmation for a draft; two for anything marked Published, the second
// of which asks for the record's own name typed out. That is the same guard
// the snapshot release uses and for the same reason: a published record stands
// for an object in the world under a number that cannot be reused, and there
// is no undo here.

import { useState } from 'react'
import {
  type Deletion, deleteButtonLabel, deleteConfirmed, deleteSteps,
} from './deleteModel'

interface Props {
  deletion: Deletion
  onDelete: () => void
  onCancel: () => void
}

export default function ConfirmDeleteDialog({ deletion, onDelete, onCancel }: Props) {
  const [step, setStep] = useState(1)
  const [typed, setTyped] = useState('')

  const steps = deleteSteps(deletion)
  const typing = step === 2
  const matches = deleteConfirmed(deletion, typed)
  const mismatch = typed.length > 0 && !matches
  const blocked = typing && !matches

  function next() {
    if (blocked) return
    if (step < steps) return setStep(step + 1)
    onDelete()
  }

  return (
    <div className="bi-modal-scrim">
      <div className="bi-modal">
        <header className="bi-modal-head">
          <span>{deletion.kind === 'edition' ? 'Delete edition' : 'Delete format'}</span>
          {steps > 1 && (
            <span className="bi-release-dots" aria-label={`Step ${step} of ${steps}`}>
              {[1, 2].map(n => <i key={n} className={n <= step ? 'on' : ''} />)}
            </span>
          )}
        </header>

        <div className="bi-modal-body">
          {!typing && <>
            <strong className="bi-modal-title">{deletion.title}</strong>
            <ul className="bi-release-losses">
              {deletion.losses.map(loss => <li key={loss}><span>—</span>{loss}</li>)}
            </ul>
            {deletion.note && <p>{deletion.note}</p>}
            {deletion.published && (
              <p>
                This record is marked <em>published</em>, so the next step asks you to type its
                name.
              </p>
            )}
          </>}

          {typing && <>
            <strong className="bi-modal-title">Type the name to confirm</strong>
            <p>
              Type <em className="bi-release-phrase">{deletion.name}</em> below. This cannot be
              undone.
            </p>
            <input
              className="dialog-input mono" value={typed} placeholder={deletion.name} autoFocus
              onChange={event => setTyped(event.target.value)}
              onKeyDown={event => { if (event.key === 'Enter' && matches) next() }}
            />
            {mismatch && <span className="bi-modal-note">Doesn’t match yet.</span>}
          </>}
        </div>

        <footer className="bi-modal-foot">
          {steps > 1 && <span className="bi-modal-step">Step {step} of {steps}</span>}
          <button type="button" className="dialog-btn" onClick={onCancel}>Keep it</button>
          <button type="button" className="dialog-btn danger" onClick={next} disabled={blocked}>
            {deleteButtonLabel(deletion, step)}
          </button>
        </footer>
      </div>
    </div>
  )
}
