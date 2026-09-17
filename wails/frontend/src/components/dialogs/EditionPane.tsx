// One edition: the record shared by every format published under it, the
// formats themselves, the locked manuscript if there is one, and the cover.
//
// Everything here writes straight to the book through bookStore, which dirties
// it and autosaves five seconds later, exactly as typing in a chapter does.
// There is no separate Save for the publishing record; the dialog's Save
// button belongs to the shared book form beside it.

import CoverCard from './CoverCard'
import type { Edition, EditionIndex, EditionSnapshot, Metadata } from '../../types/draftline'
import { EDITION_STATUSES, editionBadge, kindDot } from './editionModel'
import { templateFor, templateSummary } from './formatTemplate'
import { editionSnapshotRows } from './snapshotModel'

interface Props {
  meta: Partial<Metadata>
  index: EditionIndex
  edition: Edition
  snapshot?: EditionSnapshot
  /** The working draft's length today, for the snapshot's drift row. */
  draftWords: number
  onEdition: (patch: Partial<Edition>) => void
  onSelectFormat: (formatID: string) => void
  onAddFormat: () => void
  onDuplicate: () => void
  onRemove: () => void
  onOpenSnapshot: () => void
  onRelease: () => void
  /** Removing or replacing the cover goes through the shell's confirmation. */
  onAskCover: (action: 'remove' | 'replace', act: () => void) => void
  onSaveCoverCopy: () => void
}

// A record written by another build may hold a status this list does not
// offer. It is kept rather than silently rewritten.
function statusOptions(value: string): string[] {
  const out = value && !EDITION_STATUSES.includes(value) ? [value, ...EDITION_STATUSES] : [...EDITION_STATUSES]
  if (!value) out.unshift('')
  return out
}

export default function EditionPane(props: Props) {
  const { meta, index, edition, snapshot } = props
  const badge = editionBadge(edition)
  const previous = index.editions.find(one => one.id === edition.previous_edition_id)
  const ready = edition.formats.filter(one => (one.isbn13 ?? '').trim()).length
  const readyText = snapshot
    ? 'text locked'
    : ready === edition.formats.length && ready > 0 ? 'all ready to export' : `${ready} with an ISBN`

  return <>
    <header className="bi-pane-head">
      <div className="bi-pane-head-text">
        <div className="bi-pane-title">
          <strong>{edition.label || 'Untitled edition'}</strong>
          <span className={`bi-badge ${badge.kind}`}>{badge.label}</span>
        </div>
        <span className="bi-pane-sub">
          © {edition.year || 'year not set'} · {edition.formats.length} {edition.formats.length === 1 ? 'format' : 'formats'} · {readyText}
        </span>
      </div>
      <div className="bi-pane-actions">
        <button type="button" className="dialog-btn sm" onClick={props.onDuplicate}>Duplicate as new edition</button>
        <button type="button" className="dialog-btn sm" onClick={props.onRemove}>Remove</button>
      </div>
    </header>

    <div className="bi-ed-columns">
      <div className="bi-ed-main">
        <section className="bi-card">
          <div className="bi-card-head">
            <span className="chapter-section-label">Edition</span>
            <small>Shared by every format under it.</small>
          </div>
          <div className="bi-grid two">
            <div className="bi-field">
              <label className="bi-field-label">Label</label>
              <input
                className="dialog-input" value={edition.label} placeholder="First edition"
                onChange={event => props.onEdition({ label: event.target.value })}
              />
              <div className="bi-field-hint">Printed on title and copyright pages.</div>
            </div>
            <div className="bi-field">
              <label className="bi-field-label">Copyright year</label>
              <input
                className="dialog-input mono" value={edition.year} placeholder="2026"
                onChange={event => props.onEdition({ year: event.target.value })}
              />
            </div>
            <div className="bi-field">
              <label className="bi-field-label">Status</label>
              <select
                className="dialog-select" value={edition.status}
                onChange={event => props.onEdition({ status: event.target.value })}
              >
                {statusOptions(edition.status).map(one => (
                  <option key={one || '—'} value={one}>{one || 'Not set'}</option>
                ))}
              </select>
            </div>
            <div className="bi-field">
              <label className="bi-field-label">Supersedes</label>
              <div className="bi-static">{previous ? `${previous.label} (${previous.year})` : 'None — original release'}</div>
            </div>
          </div>
          <div className="bi-field bi-field-wide">
            <label className="bi-field-label">Revision note</label>
            <textarea
              className="dialog-input bi-textarea" rows={3} value={edition.revision_note ?? ''}
              placeholder="What changed in this edition."
              onChange={event => props.onEdition({ revision_note: event.target.value })}
            />
            <div className="bi-field-hint">
              Optional. A later edition prints this on its copyright page; a first edition never does.
            </div>
          </div>
        </section>

        <section className="bi-card">
          <div className="bi-card-head">
            <span className="chapter-section-label">Formats</span>
            <small>Each has its own ISBN and export template.</small>
          </div>
          <div className="bi-format-list">
            {edition.formats.map(format => {
              const isbn = (format.isbn13 ?? '').trim()
              return (
                <button
                  type="button" key={format.id} className="bi-format-row"
                  onClick={() => props.onSelectFormat(format.id)}
                >
                  <span className="bi-ed-dot" style={{ background: kindDot(format.kind) }} />
                  <span className="bi-format-name">{format.format || 'Format'}</span>
                  <span className="bi-format-isbn">{isbn || 'No ISBN yet'}</span>
                  <span className="bi-format-summary">{templateSummary(format, templateFor(edition, format))}</span>
                  <span className={`bi-format-state${isbn ? ' ready' : ''}`}>{isbn ? 'Ready' : 'No ISBN'}</span>
                  <span className="bi-format-chevron" aria-hidden="true">›</span>
                </button>
              )
            })}
            <button type="button" className="bi-format-add" onClick={props.onAddFormat}>
              <span aria-hidden="true">+</span>Add format
            </button>
          </div>
        </section>

        {snapshot && (
          <section className="bi-card bi-snap">
            <div className="bi-card-head">
              <span className="chapter-section-label">Manuscript snapshot</span>
              <span className="bi-badge ok">Locked</span>
              <small>The text as it stood when this edition was first exported.</small>
            </div>
            <div className="bi-snap-body">
              <div className="bi-snap-page" aria-hidden="true">
                <span className="bi-snap-kicker">Chapter One</span>
                <i style={{ width: '88%' }} /><i /><i style={{ width: '94%' }} /><i style={{ width: '70%' }} />
                <span className="bi-snap-break">⁂</span>
                <i style={{ width: '84%' }} /><i /><i style={{ width: '60%' }} />
              </div>
              <div className="bi-snap-facts">
                {editionSnapshotRows(index, snapshot, props.draftWords).map(fact => (
                  <div className="bi-snap-fact" key={fact.label}>
                    <span>{fact.label}</span>
                    <em>{fact.value}</em>
                  </div>
                ))}
              </div>
            </div>
            <div className="bi-snap-actions">
              <button type="button" className="dialog-btn sm" onClick={props.onOpenSnapshot}>Open snapshot read-only</button>
              {/* The only destructive act on this screen, and the only one set
                  in the error colour. It opens a three-step confirmation. */}
              <button type="button" className="bi-danger-link" onClick={props.onRelease}>Release snapshot…</button>
            </div>
          </section>
        )}
      </div>

      {/* One cover per edition: every format under it publishes the same
          artwork, so it is attached here rather than repeated on each one. */}
      <CoverCard edition={edition} meta={meta} onAsk={props.onAskCover} onSaveCopy={props.onSaveCoverCopy} />
    </div>
  </>
}
