// Releasing an edition's locked text.
//
// This is the most destructive thing the Book & Editions screen can do, and it
// is deliberately the slowest. Three steps, in this order:
//
//   1  what releasing means, with the working draft's actual drift in it, so
//      the sentence is about this book rather than about the feature;
//   2  what is lost, itemised, and the alternative said plainly — a typo in a
//      reprint is a second edition, not a released snapshot;
//   3  type the edition's name. Not "yes", not a checkbox: the name, which
//      cannot be produced by a reflex.
//
// The three steps are not friction for its own sake. A released snapshot is
// the only record of the words readers bought under those ISBNs, and there is
// no undo — the text is dropped from the project file on the next save once
// nothing points at it any more. Do not collapse this into one confirm.

import { useState } from 'react'
import type { Edition, EditionSnapshot } from '../../types/draftline'
import {
  RELEASE_STEPS, draftDrift, formatsOnSnapshot, frozenLabel, releaseConfirmed, releaseStepLabel,
} from './snapshotModel'

interface Props {
  edition: Edition
  snapshot: EditionSnapshot
  /** The working draft's length today, for the drift sentence. */
  draftWords: number
  onRelease: () => void
  onKeep: () => void
}

export default function ReleaseSnapshotDialog({ edition, snapshot, draftWords, onRelease, onKeep }: Props) {
  const [step, setStep] = useState(1)
  const [typed, setTyped] = useState('')

  const name = (edition.label ?? '').trim() || 'this edition'
  const matches = releaseConfirmed(edition, typed)
  const mismatch = typed.length > 0 && !matches
  const formats = formatsOnSnapshot(edition, snapshot.id)
  const drift = draftDrift(snapshot, draftWords)
  const status = (edition.status ?? '').trim().toLowerCase() || 'registered'

  const nextLabel = releaseStepLabel(step)
  const blocked = step === RELEASE_STEPS && !matches

  function next() {
    if (blocked) return
    if (step < RELEASE_STEPS) return setStep(step + 1)
    onRelease()
  }

  return (
    <div className="bi-modal-scrim">
      <div className="bi-modal bi-release">
        <header className="bi-modal-head">
          <span>Release snapshot — {name}</span>
          <span className="bi-release-dots" aria-label={`Step ${step} of ${RELEASE_STEPS}`}>
            {[1, 2, 3].map(n => <i key={n} className={n <= step ? 'on' : ''} />)}
          </span>
        </header>

        <div className="bi-modal-body">
          {step === 1 && <>
            <strong className="bi-modal-title">Are you sure?</strong>
            <p>
              This edition is <em>{status}</em>.
              {formats.length ? ` Its ${formats.join(', ')} ${formats.length === 1 ? 'was' : 'were'} exported from this snapshot.` : ''}
              {' '}Releasing it means the next export of any of its ISBNs will use the working draft
              {drift ? `, which is already ${drift.replace(' since', ' ahead')}` : ''}.
            </p>
          </>}

          {step === 2 && <>
            <strong className="bi-modal-title">What you lose</strong>
            <ul className="bi-release-losses">
              <li><span>—</span>The only record of the text readers bought under these ISBNs.</li>
              <li><span>—</span>The ability to reprint this edition exactly as sold.</li>
              <li><span>—</span>Comparing the working draft against what went out.</li>
            </ul>
            <p>
              If you only want to fix a typo in a reprint, keep the snapshot and register a new
              edition instead.
            </p>
          </>}

          {step === 3 && <>
            <strong className="bi-modal-title">Type the edition name to confirm</strong>
            <p>
              Type <em className="bi-release-phrase">{name}</em> below. This cannot be undone
              {snapshot.frozen ? `; the text was locked on ${frozenLabel(snapshot.frozen)}` : ''}.
            </p>
            <input
              className="dialog-input mono" value={typed} placeholder={name} autoFocus
              onChange={event => setTyped(event.target.value)}
              onKeyDown={event => { if (event.key === 'Enter' && matches) next() }}
            />
            {mismatch && <span className="bi-modal-note">Doesn’t match yet.</span>}
          </>}
        </div>

        <footer className="bi-modal-foot">
          <span className="bi-modal-step">Step {step} of {RELEASE_STEPS}</span>
          <button type="button" className="dialog-btn" onClick={onKeep}>Keep snapshot</button>
          <button type="button" className="dialog-btn danger" onClick={next} disabled={blocked}>
            {nextLabel}
          </button>
        </footer>
      </div>
    </div>
  )
}
