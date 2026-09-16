// The right-hand pane of the Book & Editions screen when an edition, or one
// format of one edition, is selected.
//
// Everything here writes straight to the book through bookStore, which dirties
// it and autosaves five seconds later, exactly as typing in a chapter does.
// There is no separate Save for the publishing record; the dialog's Save
// button belongs to the shared book form beside it.
//
// The rows themselves are built in editionModel.ts, which is where the design
// brief's sections live. This file only draws them.

import { useState } from 'react'
import CoverCard from './CoverCard'
import SnapshotCard from './SnapshotCard'
import { RemoveCover } from '../../../wailsjs/go/main/App'
import { useBookStore } from '../../store/bookStore'
import type { Edition, EditionFormat, EditionIndex, Metadata } from '../../types/draftline'
import {
  EDITION_STATUSES, advancedFor, copyrightLines, editionBadge, formatBadge, formatSubtitle,
  formatTitle, kindDot, priorYears, sectionsFor,
  type EditionRow,
} from './editionModel'

interface Props {
  // The metadata as the author is currently typing it, so the generated
  // copyright page answers the form beside it rather than the last save.
  meta: Partial<Metadata>
  index: EditionIndex
  editionID: string
  formatID?: string
  // True when the book carries a copyright page somebody wrote by hand.
  handEdited: boolean
  onSelect: (editionID: string, formatID?: string) => void
}

function Badge({ label, kind }: { label: string; kind: string }) {
  return <span className={`bi-ed-badge ${kind}`}>{label}</span>
}

// A select keeps whatever the record already holds even when this build does
// not offer that wording: a value written by another version of Draftline is
// not wrong, and a dropdown that silently rewrites it would be.
function optionsWith(options: string[], value: string): string[] {
  const out = value && !options.includes(value) ? [value, ...options] : [...options]
  if (!value) out.unshift('')
  return out
}

function RowField({ row, onChange }: { row: EditionRow; onChange: (value: string) => void }) {
  const mono = row.mono ? ' mono' : ''
  if (row.kind === 'static') {
    return <div className={`bi-ed-static${mono}`}>{row.value}</div>
  }
  if (row.kind === 'select') {
    return (
      <select
        className="dialog-select" value={row.value}
        onChange={e => onChange(e.target.value)}
      >
        {optionsWith(row.options ?? [], row.value).map(option => (
          <option key={option || '—'} value={option}>{option || 'Not set'}</option>
        ))}
      </select>
    )
  }
  if (row.kind === 'textarea') {
    return (
      <textarea
        className="dialog-input bi-textarea" rows={3} value={row.value}
        placeholder={row.placeholder} onChange={e => onChange(e.target.value)}
      />
    )
  }
  return (
    <input
      className={`dialog-input${mono}`} value={row.value} placeholder={row.placeholder}
      readOnly={row.locked} disabled={row.locked}
      onChange={e => onChange(e.target.value)}
    />
  )
}

function Rows({ rows, onChange }: { rows: EditionRow[]; onChange: (row: EditionRow, value: string) => void }) {
  return (
    <div className="bi-grid">
      {rows.map(row => (
        <div className="bi-field" key={row.label}>
          <label className="bi-field-label">{row.label}</label>
          <RowField row={row} onChange={value => onChange(row, value)} />
          {row.hint && <div className="bi-field-hint">{row.hint}</div>}
        </div>
      ))}
    </div>
  )
}

export default function EditionPane({ meta, index, editionID, formatID, handEdited, onSelect }: Props) {
  const updateEdition = useBookStore(s => s.updateEdition)
  const updateFormat = useBookStore(s => s.updateFormat)
  const removeEdition = useBookStore(s => s.removeEdition)
  const removeFormat = useBookStore(s => s.removeFormat)
  const duplicateEdition = useBookStore(s => s.duplicateEdition)
  const [advOpen, setAdvOpen] = useState(false)

  const edition = index.editions.find(e => e.id === editionID)
  if (!edition) return null
  const format = formatID ? edition.formats.find(f => f.id === formatID) : undefined

  const duplicate = () => {
    const created = duplicateEdition(edition.id, String(new Date().getFullYear()))
    if (created) onSelect(created)
  }

  // Deleting an edition lets go of its artwork too. The save itself no longer
  // carries the cover of an edition the book does not have, so the project
  // file would come out right either way; this is so the megabyte stops being
  // held in memory the moment the author says the edition is gone.
  const remove = () => {
    void RemoveCover(edition.id)
    removeEdition(edition.id)
    onSelect('')
  }

  return (
    <div className="bi-ed-pane">
      {format
        ? <FormatPanel
            meta={meta} index={index} edition={edition} format={format} handEdited={handEdited}
            advOpen={advOpen} setAdvOpen={setAdvOpen}
            onEdition={patch => updateEdition(edition.id, patch)}
            onFormat={patch => updateFormat(format.id, patch)}
            onDuplicate={duplicate}
            onRemove={() => { removeFormat(format.id); onSelect(edition.id) }}
          />
        : <EditionPanel
            meta={meta} index={index} edition={edition} handEdited={handEdited}
            onEdition={patch => updateEdition(edition.id, patch)}
            onDuplicate={duplicate}
            onRemove={remove}
          />}
    </div>
  )
}

// ── The edition itself ─────────────────────────────────────────────────────

function EditionPanel({ meta, index, edition, handEdited, onEdition, onDuplicate, onRemove }: {
  meta: Partial<Metadata>
  index: EditionIndex
  edition: Edition
  handEdited: boolean
  onEdition: (patch: Partial<Edition>) => void
  onDuplicate: () => void
  onRemove: () => void
}) {
  const badge = editionBadge(edition)
  const previous = index.editions.find(e => e.id === edition.previous_edition_id)
  return (
    <>
      <header className="bi-ed-head">
        <div className="bi-ed-head-text">
          <div className="bi-ed-title-line">
            <span className="bi-ed-title">{edition.label || 'Untitled edition'}</span>
            <Badge label={badge.label} kind={badge.kind} />
          </div>
          <div className="bi-ed-subtitle">
            {edition.formats.length
              ? `${edition.formats.length} format${edition.formats.length === 1 ? '' : 's'} · copyright ${edition.year || 'year not set'}`
              : 'No formats yet. Add one from the rail to give this edition an ISBN.'}
          </div>
        </div>
        <div className="bi-ed-head-actions">
          <button className="dialog-btn sm" onClick={onDuplicate}>Duplicate as new edition</button>
          <button className="dialog-btn sm" onClick={onRemove}>Remove edition</button>
        </div>
      </header>

      <section className="bi-section">
        <div className="bi-section-head">
          <span className="chapter-section-label">Edition</span>
          <span className="bi-section-note">Shared by every format under it.</span>
        </div>
        <div className="bi-grid">
          <div className="bi-field">
            <label className="bi-field-label">Label</label>
            <input
              className="dialog-input" value={edition.label}
              onChange={e => onEdition({ label: e.target.value })}
              placeholder="First edition"
            />
            <div className="bi-field-hint">Printed on the copyright page with the publication month.</div>
          </div>
          <div className="bi-field">
            <label className="bi-field-label">Copyright year</label>
            <input
              className="dialog-input mono" value={edition.year}
              onChange={e => onEdition({ year: e.target.value })}
              placeholder="2026"
            />
            <div className="bi-field-hint">Added to the years the editions before it established.</div>
          </div>
          <div className="bi-field">
            <label className="bi-field-label">Status</label>
            <select className="dialog-select" value={edition.status} onChange={e => onEdition({ status: e.target.value })}>
              {optionsWith(EDITION_STATUSES, edition.status).map(option => (
                <option key={option || '—'} value={option}>{option || 'Not set'}</option>
              ))}
            </select>
          </div>
          <div className="bi-field">
            <label className="bi-field-label">Supersedes</label>
            <div className="bi-ed-static">{previous ? `${previous.label} (${previous.year})` : 'None — original release'}</div>
          </div>
          <div className="bi-field bi-field-wide">
            <label className="bi-field-label">Revision note</label>
            <textarea
              className="dialog-input bi-textarea" rows={3} value={edition.revision_note ?? ''}
              onChange={e => onEdition({ revision_note: e.target.value })}
              placeholder="What changed in this edition."
            />
            <div className="bi-field-hint">
              Optional. Printed on the copyright page of a later edition; a first edition never prints one.
            </div>
          </div>
        </div>
      </section>

      <CopyrightCard
        meta={meta} index={index} edition={edition}
        format={edition.formats[0]} handEdited={handEdited}
      />
    </>
  )
}

// ── One format ─────────────────────────────────────────────────────────────

function FormatPanel({ meta, index, edition, format, handEdited, advOpen, setAdvOpen, onEdition, onFormat, onDuplicate, onRemove }: {
  meta: Partial<Metadata>
  index: EditionIndex
  edition: Edition
  format: EditionFormat
  handEdited: boolean
  advOpen: boolean
  setAdvOpen: (open: boolean) => void
  onEdition: (patch: Partial<Edition>) => void
  onFormat: (patch: Partial<EditionFormat>) => void
  onDuplicate: () => void
  onRemove: () => void
}) {
  const badge = formatBadge(format)
  const sections = sectionsFor(edition, format)
  const advanced = advancedFor(index, edition, format)
  const apply = (row: EditionRow, value: string) => {
    if (!row.field || row.locked) return
    onFormat({ [row.field]: value } as Partial<EditionFormat>)
  }

  return (
    <>
      <header className="bi-ed-head">
        <div className="bi-ed-head-text">
          <div className="bi-ed-title-line">
            <span className="bi-ed-dot" style={{ background: kindDot(format.kind) }} />
            <span className="bi-ed-title">{formatTitle(edition, format)}</span>
            <Badge label={badge.label} kind={badge.kind} />
          </div>
          <div className="bi-ed-subtitle">{formatSubtitle(edition, format) || 'Not released yet.'}</div>
        </div>
        <div className="bi-ed-head-actions">
          <button className="dialog-btn sm" onClick={onDuplicate}>Duplicate as new edition</button>
          <button className="dialog-btn sm" onClick={onRemove}>Remove format</button>
        </div>
      </header>

      <StandingNote />

      <CoverCard edition={edition} meta={meta} />

      <SnapshotCard index={index} edition={edition} format={format} />

      {sections.map(section => (
        <section className="bi-section" key={section.label}>
          <div className="bi-section-head">
            <span className="chapter-section-label">{section.label}</span>
            {section.note && <span className="bi-section-note">{section.note}</span>}
          </div>
          <Rows rows={section.rows} onChange={apply} />
        </section>
      ))}

      <CopyrightCard meta={meta} index={index} edition={edition} format={format} handEdited={handEdited} />

      <section className="bi-section bi-ed-advanced">
        <button
          type="button" className="bi-ed-disclosure"
          onClick={() => setAdvOpen(!advOpen)} aria-expanded={advOpen}
        >
          <span className={`bi-ed-caret${advOpen ? ' open' : ''}`}>›</span>
          <span className="chapter-section-label">Advanced</span>
          {!advOpen && (
            <span className="bi-section-note">ISBN-10, imprint, territory, LCCN, retailer and print details</span>
          )}
        </button>
        {advOpen && (
          <>
            <Rows rows={advanced} onChange={apply} />
            <div className="bi-field bi-field-wide bi-ed-revision">
              <label className="bi-field-label">Revision note</label>
              <textarea
                className="dialog-input bi-textarea" rows={3} value={edition.revision_note ?? ''}
                onChange={e => onEdition({ revision_note: e.target.value })}
                placeholder="What changed in this edition."
              />
              <div className="bi-field-hint">
                Belongs to the whole edition. Printed on the copyright page of a later edition; a first
                edition never prints one.
              </div>
            </div>
          </>
        )}
      </section>
    </>
  )
}

// ── What the record does today ─────────────────────────────────────────────

// What an export does with this panel, said once at the top rather than as a
// disclaimer hung on every field.
function StandingNote() {
  return (
    <p className="bi-ed-standing">
      <strong>Read by an export.</strong> Choose this edition on the first step of the export wizard
      and the file carries it: its ISBN as the book's identifier, its cover art, its trim and gutter,
      the copyright page below, and the text frozen for it. Exporting this edition freezes the
      manuscript the first time and reads that frozen text every time after, so an export made
      after you have started the next edition still gives you this one's book.
    </p>
  )
}

// ── The generated copyright page ───────────────────────────────────────────

function CopyrightCard({ meta, index, edition, format, handEdited }: {
  meta: Partial<Metadata>
  index: EditionIndex
  edition: Edition
  format?: EditionFormat
  handEdited: boolean
}) {
  const subject: EditionFormat = format ?? { id: '', kind: edition.formats[0]?.kind ?? 'print' }
  const lines = copyrightLines(meta, edition, subject, priorYears(index, edition.id))
  return (
    <section className="bi-section">
      <div className="bi-section-head">
        <span className="chapter-section-label">Copyright page</span>
        <span className="bi-section-note">Written from the fields above. Nothing to type.</span>
      </div>
      <div className="bi-ed-copyright">
        {lines.map((line, i) => (
          <span key={i} className="bi-ed-copyright-line">{line || ' '}</span>
        ))}
      </div>
      <div className="bi-field-hint bi-ed-handwritten">
{handEdited ? (
          <>
            This is the page an export of this edition prints. The Copyright Page you wrote yourself,
            under Front Pages, follows underneath it — a copyright page carries more than the notice,
            and nothing you wrote there is dropped.
          </>
        ) : (
          <>
            This is the page an export of this edition prints. You have written nothing under Front
            Pages, so this is the whole of it.
          </>
        )}
      </div>
    </section>
  )
}
