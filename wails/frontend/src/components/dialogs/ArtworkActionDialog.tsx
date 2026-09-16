// Confirming something that can lose artwork.
//
// One step normally, two when the record is published — the second asks for
// its own name typed out, the same guard the snapshot release and the delete
// confirmations use.
//
// The backup line is the point of the dialog. When the project holds the only
// copy, saving it out is not offered as a checkbox: it is stated as what
// happens first, and the action follows a file existing on disk. When the
// original is still there it is an offer, off by default, because nothing is
// at risk.

import { useState } from 'react'
import { type ArtworkPrompt, artworkConfirmed, artworkSteps } from './artworkModel'

interface Props {
  prompt: ArtworkPrompt
  /** The name that has to be typed on the second step. */
  name: string
  /** backup is true when a copy should be written out before acting. */
  onConfirm: (backup: boolean) => void
  onCancel: () => void
}

export default function ArtworkActionDialog({ prompt, name, onConfirm, onCancel }: Props) {
  const [step, setStep] = useState(1)
  const [typed, setTyped] = useState('')
  const [backup, setBackup] = useState(false)

  const steps = artworkSteps(prompt)
  const typing = step === 2
  const matches = artworkConfirmed(name, typed)
  const blocked = typing && !matches

  function next() {
    if (blocked) return
    if (step < steps) return setStep(step + 1)
    onConfirm(prompt.forcedBackup || backup)
  }

  return (
    <div className="bi-modal-scrim">
      <div className="bi-modal">
        <header className="bi-modal-head">
          <span>{prompt.title}</span>
          {steps > 1 && (
            <span className="bi-release-dots" aria-label={`Step ${step} of ${steps}`}>
              {[1, 2].map(n => <i key={n} className={n <= step ? 'on' : ''} />)}
            </span>
          )}
        </header>

        <div className="bi-modal-body">
          {!typing && <>
            {prompt.body.map(line => <p key={line}>{line}</p>)}
            {prompt.offerBackup && (
              <label className="bi-keep">
                <input type="checkbox" checked={backup} onChange={e => setBackup(e.target.checked)} />
                <span>Save a copy of the stored file first</span>
              </label>
            )}
            {prompt.typeToConfirm && (
              <p>This record is marked <em>published</em>, so the next step asks you to type its name.</p>
            )}
          </>}

          {typing && <>
            <strong className="bi-modal-title">Type the name to confirm</strong>
            <p>Type <em className="bi-release-phrase">{name}</em> below.</p>
            <input
              className="dialog-input mono" value={typed} placeholder={name} autoFocus
              onChange={e => setTyped(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter' && matches) next() }}
            />
            {typed.length > 0 && !matches && <span className="bi-modal-note">Doesn’t match yet.</span>}
          </>}
        </div>

        <footer className="bi-modal-foot">
          {steps > 1 && <span className="bi-modal-step">Step {step} of {steps}</span>}
          <button type="button" className="dialog-btn" onClick={onCancel}>Cancel</button>
          <button type="button" className="dialog-btn danger" onClick={next} disabled={blocked}>
            {steps > step ? 'Yes, continue' : prompt.confirmLabel}
          </button>
        </footer>
      </div>
    </div>
  )
}
