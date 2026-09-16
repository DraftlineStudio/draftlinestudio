// The text one ISBN stands for.
//
// It sits beside the cover on the format panel and answers one question: if
// you export this edition today, which words come out? A format with a frozen
// manuscript gives you the book as it was when that ISBN went out. A format
// without one gives you the draft on screen, and says so before you find out.
//
// The words never reach this component. Everything drawn here comes from the
// small catalogue record on the book; freezing is a call into Go, which builds
// the snapshot, keeps the bytes on its own side and hands back the record.

import { useState } from 'react'
import { FreezeSnapshot } from '../../../wailsjs/go/main/App'
import { useBookStore } from '../../store/bookStore'
import type { Edition, EditionFormat, EditionIndex } from '../../types/draftline'
import {
  SNAPSHOT_ABSENT_NOTE, SNAPSHOT_DETAILS_NOTE, SNAPSHOT_REFREEZE_NOTE,
  SNAPSHOT_RELEASE_NOTE, SNAPSHOT_SHARED_NOTE,
  frozenLabel, sharedWith, snapshotFacts, snapshotFootprint, snapshotFor,
} from './snapshotModel'

interface Props {
  index: EditionIndex
  edition: Edition
  format: EditionFormat
}

export default function SnapshotCard({ index, edition, format }: Props) {
  const book = useBookStore(s => s.book)
  const freezeFormat = useBookStore(s => s.freezeFormat)
  const releaseFormatSnapshot = useBookStore(s => s.releaseFormatSnapshot)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [note, setNote] = useState('')

  const snapshot = snapshotFor(index, format)
  const shared = snapshot ? sharedWith(index, snapshot.id, format.id) : []
  const footprint = snapshotFootprint(index)

  const freeze = async () => {
    if (!book) return
    setBusy(true)
    setError('')
    setNote('')
    try {
      const result = await FreezeSnapshot(book as never, format.id)
      if (!result.success || !result.snapshot) {
        setError(result.error || 'That text could not be frozen.')
        return
      }
      freezeFormat(format.id, result.snapshot)
      setNote(result.reused
        ? 'These are words this project had already frozen. They are stored once and this ISBN now points at them too.'
        : `Frozen. ${result.snapshot.word_count.toLocaleString()} words are now the text this ISBN stands for.`)
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }

  const release = () => {
    setError('')
    setNote(shared.length
      ? `Released. ${shared.join(' and ')} still exports these words, so they stay in the project file.`
      : 'Released. Nothing points at those words now, so the next save drops them.')
    releaseFormatSnapshot(format.id)
  }

  return (
    <section className="bi-ed-card bi-snap-card">
      <div className="bi-ed-card-body">
        <div className="bi-section-head">
          <span className="chapter-section-label">The text this ISBN stands for</span>
          <span className="bi-section-note">
            {snapshot ? 'Frozen at export. Editing the book does not change it.' : 'Nothing frozen yet.'}
          </span>
        </div>

        {snapshot ? (
          <div className="bi-ed-facts">
            {snapshotFacts(snapshot).map(fact => (
              <div className="bi-ed-fact" key={fact.label}>
                <span>{fact.label}</span><span>{fact.value}</span>
              </div>
            ))}
          </div>
        ) : (
          <p className="bi-ed-note">{SNAPSHOT_ABSENT_NOTE}</p>
        )}

        {snapshot && shared.length > 0 && (
          <p className="bi-snap-shared">
            <strong>Shared with {shared.join(', ')}.</strong> {SNAPSHOT_SHARED_NOTE}
          </p>
        )}

        <div className="bi-snap-actions">
          <button className="dialog-btn sm" disabled={busy} onClick={() => void freeze()}>
            {busy ? 'Freezing…' : snapshot ? 'Freeze the text as it is now' : 'Freeze now'}
          </button>
          {snapshot && (
            <button className="dialog-btn sm" disabled={busy} onClick={release}>
              Release the frozen text
            </button>
          )}
        </div>

        {error && <p className="bi-cover-error">{error}</p>}
        {note && <p className="bi-snap-note">{note}</p>}

        {snapshot && (
          <p className="bi-ed-note bi-snap-details">
            {SNAPSHOT_DETAILS_NOTE} {SNAPSHOT_REFREEZE_NOTE}
          </p>
        )}

        <div className="bi-field-hint bi-snap-hint">
          {snapshot
            ? <>Exporting this format reads those words, not the draft you have open. An export of it after you have written a second edition still gives you the first edition&rsquo;s book. Re-freezing replaces which text this ISBN stands for, which is almost never what a published number wants — {frozenLabel(snapshot.frozen) ? `this one was frozen on ${frozenLabel(snapshot.frozen)}` : 'this one is already frozen'}.</>
            : <>{SNAPSHOT_RELEASE_NOTE}</>}
        </div>

        <div className="bi-field-hint bi-snap-hint">
          {footprint.label} A project file holds a limited number of pieces; when it is close,
          Draftline says so and says which frozen text to release.
        </div>
      </div>
    </section>
  )
}
