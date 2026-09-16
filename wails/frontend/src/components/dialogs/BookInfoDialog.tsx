// Book & editions: the record a book carries and the editions published from
// it. The left rail selects what is being edited; the right pane is the form
// for it. Everything here is the book's own record, saved into the .draftline
// project.
//
// The two halves save differently, on purpose. The shared book details are a
// form with a Save button, because changing a title is one decision. An
// edition is a record you build up over months, so every keystroke on it goes
// straight onto the book and rides the ordinary five-second autosave.

import { useState } from 'react'
import { useAppStore } from '../../store/appStore'
import { useBookStore } from '../../store/bookStore'
import type { EditionIndex, EditionKind, ISBNEntry, Metadata } from '../../types/draftline'
import {
  AUDIENCES, ISBN_FORMATS, LANGUAGES, blockingProblems, checkBook, isbnRows, metadataPatch, validISBN,
} from './bookInfoModel'
import { editionBadge, emptyEditionIndex, isbnLocked, kindDot } from './editionModel'
import EditionPane from './EditionPane'

type Draft = Partial<Metadata>

// What the rail has selected. null is the shared book details; an edition with
// no format is the edition itself.
type Selection = { editionID: string; formatID?: string } | null

const FORMAT_KINDS: { value: EditionKind; label: string }[] = [
  { value: 'ebook', label: 'eBook' },
  { value: 'print', label: 'Print' },
  { value: 'audio', label: 'Audiobook' },
]

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="bi-field">
      <label className="bi-field-label">{label}</label>
      {children}
      {hint && <div className="bi-field-hint">{hint}</div>}
    </div>
  )
}

function Section({ title, note, children }: { title: string; note?: string; children: React.ReactNode }) {
  return (
    <section className="bi-section">
      <div className="bi-section-head">
        <span className="chapter-section-label">{title}</span>
        {note && <span className="bi-section-note">{note}</span>}
      </div>
      <div className="bi-grid">{children}</div>
    </section>
  )
}

export default function BookInfoDialog() {
  const book = useBookStore(s => s.book)
  const updateMetadata = useBookStore(s => s.updateMetadata)
  const addEdition = useBookStore(s => s.addEdition)
  const addFormat = useBookStore(s => s.addFormat)
  const closeMetadataDialog = useAppStore(s => s.closeMetadataDialog)
  const meta = book?.metadata

  const [draft, setDraft] = useState<Draft>(() => ({ ...meta }))
  const [isbns, setIsbns] = useState<ISBNEntry[]>(() => isbnRows(meta))
  const [selection, setSelection] = useState<Selection>(null)
  const [newKind, setNewKind] = useState<EditionKind>('ebook')

  const index: EditionIndex = book?.editions ?? emptyEditionIndex()
  const formatCount = index.editions.reduce((n, e) => n + e.formats.length, 0)
  const selected = selection ? index.editions.find(e => e.id === selection.editionID) : undefined
  const selectedFormat = selected && selection?.formatID
    ? selected.formats.find(f => f.id === selection.formatID)
    : undefined

  const set = (key: keyof Metadata) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) =>
    setDraft(d => ({ ...d, [key]: e.target.value }))
  const setEntry = (index: number, patch: Partial<ISBNEntry>) =>
    setIsbns(list => list.map((entry, i) => (i === index ? { ...entry, ...patch } : entry)))

  const problems = checkBook(draft, isbns)
  const blocking = blockingProblems(problems)
  const problemFor = (field: string) => problems.find(p => p.field === field)

  function handleSave() {
    if (blocking.length) return
    updateMetadata(metadataPatch(draft, isbns))
    closeMetadataDialog()
  }

  function handleNewEdition() {
    const created = addEdition(String(new Date().getFullYear()))
    if (created) setSelection({ editionID: created })
  }

  function handleAddFormat() {
    if (!selection) return
    const created = addFormat(selection.editionID, newKind)
    if (created) setSelection({ editionID: selection.editionID, formatID: created })
  }

  // The rail's selection can be removed from under it — a deleted edition, a
  // deleted format — so a selection that no longer resolves falls back to the
  // shared book details rather than to an empty pane.
  const resolved: Selection = selected ? { editionID: selected.id, formatID: selectedFormat?.id } : null

  const footerNote = blocking.length
    ? blocking[0].message
    : selectedFormat && isbnLocked(selectedFormat)
      ? 'This format is published, so its ISBN cannot be edited. An ISBN is fixed once it is registered. Changing trim size, cover, or publisher means a new edition record — never an edit to a published one.'
      : 'An ISBN is fixed once it is registered. Changing trim size, cover, or publisher means a new edition record — never an edit to a published one.'

  return (
    <div className="dialog-overlay">
      <div className="dialog bi-dialog">
        <header className="bi-header">
          <span className="bi-mark">D</span>
          <span className="bi-header-title">Book &amp; editions</span>
          <span className="bi-header-sub">
            {(draft.title || 'Untitled')}{draft.author ? ` · ${draft.author}` : ''}
          </span>
          <span className="bi-header-stamp">
            {meta?.created ? `Created ${new Date(meta.created).toLocaleDateString()}` : ''}
            {meta?.modified ? ` · Modified ${new Date(meta.modified).toLocaleDateString()}` : ''}
          </span>
          <button className="bi-close" onClick={closeMetadataDialog} title="Close">✕</button>
        </header>

        <div className="bi-body">
          <nav className="bi-rail">
            <div className="bi-rail-head"><span className="chapter-section-label">Book</span></div>
            <button
              type="button" className={`bi-rail-row${resolved ? '' : ' active'}`}
              onClick={() => setSelection(null)}
            >
              <span className="bi-rail-row-name">Shared book details</span>
              <span className="bi-rail-row-sub">carried by every edition</span>
            </button>

            <div className="bi-rail-head bi-rail-divider bi-ed-rail-head">
              <span className="chapter-section-label">Editions</span>
              <span className="bi-ed-rail-count">
                {formatCount} format{formatCount === 1 ? '' : 's'}
              </span>
            </div>

            {index.editions.length === 0 && (
              <div className="bi-rail-empty">
                No editions registered yet. An edition records the ISBN, the cover and the
                specification a format was published with.
              </div>
            )}

            <div className="bi-ed-groups">
              {index.editions.map(edition => {
                const badge = editionBadge(edition)
                const groupActive = resolved?.editionID === edition.id && !resolved.formatID
                return (
                  <div className="bi-ed-group" key={edition.id}>
                    <button
                      type="button" className={`bi-ed-group-head${groupActive ? ' active' : ''}`}
                      onClick={() => setSelection({ editionID: edition.id })}
                    >
                      <span className="bi-ed-group-label">{edition.label || 'Untitled edition'}</span>
                      <span className="bi-ed-group-year">{edition.year}</span>
                      <span className={`bi-ed-badge ${badge.kind}`}>{badge.label}</span>
                    </button>
                    {edition.formats.length === 0 && (
                      <div className="bi-ed-group-empty">No formats yet</div>
                    )}
                    {edition.formats.map(format => {
                      const active = resolved?.formatID === format.id
                      return (
                        <button
                          type="button" key={format.id}
                          className={`bi-ed-format-row${active ? ' active' : ''}`}
                          onClick={() => setSelection({ editionID: edition.id, formatID: format.id })}
                        >
                          <span className="bi-ed-dot" style={{ background: kindDot(format.kind) }} />
                          <span className="bi-ed-format-text">
                            <span className="bi-ed-format-name">{format.format || format.kind}</span>
                            <span className="bi-ed-format-isbn">{format.isbn13 || 'No ISBN yet'}</span>
                          </span>
                        </button>
                      )
                    })}
                  </div>
                )
              })}
            </div>

            <div className="bi-ed-rail-foot">
              <button type="button" className="dialog-btn sm" onClick={handleNewEdition}>+ New edition</button>
              <div className="bi-ed-add-format">
                <select
                  className="dialog-select" value={newKind}
                  onChange={e => setNewKind(e.target.value as EditionKind)}
                  aria-label="Kind of format to add"
                >
                  {FORMAT_KINDS.map(k => <option key={k.value} value={k.value}>{k.label}</option>)}
                </select>
                <button
                  type="button" className="dialog-btn sm" onClick={handleAddFormat}
                  disabled={!resolved}
                  title={resolved ? 'Add this kind of format to the selected edition' : 'Select an edition first'}
                >
                  + Add format to edition
                </button>
              </div>
            </div>
          </nav>

          {resolved ? (
            <div className="bi-pane">
              <EditionPane
                meta={draft} index={index}
                editionID={resolved.editionID} formatID={resolved.formatID}
                handEdited={!!book?.copyright?.trim()}
                onSelect={(editionID, formatID) => setSelection(editionID ? { editionID, formatID } : null)}
              />
            </div>
          ) : (
          <div className="bi-pane">
            <p className="bi-pane-intro">
              These carry across every edition. Anything an edition changes, such as its
              ISBN, its cover or its trim size, belongs to the edition itself.
            </p>

            <Section title="Work">
              <Field label="Title">
                <input
                  className={`dialog-input${problemFor('title') ? ' invalid' : ''}`}
                  value={draft.title ?? ''} onChange={set('title')} autoFocus
                />
              </Field>
              <Field label="Subtitle" hint="Optional. Appears on covers and in storefront metadata.">
                <input className="dialog-input" value={draft.subtitle ?? ''} onChange={set('subtitle')} />
              </Field>
              <Field label="Author">
                <input className="dialog-input" value={draft.author ?? ''} onChange={set('author')} />
              </Field>
              <Field label="Series">
                <input className="dialog-input" value={draft.series_name ?? ''} onChange={set('series_name')} placeholder="Standalone" />
              </Field>
              <Field label="Number in series" hint='Printed as written: "2", "2.5" or "Book Two".'>
                <input className="dialog-input" value={draft.series_number ?? ''} onChange={set('series_number')} />
              </Field>
              <Field label="Publisher">
                <input className="dialog-input" value={draft.publisher ?? ''} onChange={set('publisher')} />
              </Field>
              <Field label="Imprint" hint="The line the book appears under. Exports print this when it is set.">
                <input className="dialog-input" value={draft.imprint ?? ''} onChange={set('imprint')} placeholder={draft.publisher || ''} />
              </Field>
              <Field label="Language" hint="Exports declare this. A book without one exports as English.">
                <input
                  className="dialog-input" list="bi-languages" value={draft.language ?? ''}
                  onChange={set('language')} placeholder="en-US"
                />
                <datalist id="bi-languages">
                  {LANGUAGES.map(l => <option key={l.value} value={l.value}>{l.label}</option>)}
                </datalist>
              </Field>
              <Field label="Copyright holder" hint="Not always the author. The generated copyright page is made out to this name.">
                <input className="dialog-input" value={draft.copyright_holder ?? ''} onChange={set('copyright_holder')} placeholder={draft.author || ''} />
              </Field>
            </Section>

            <Section title="Identifiers" note="An ISBN identifies one format of one edition.">
              <div className="bi-field bi-field-wide">
                {isbns.map((entry, index) => {
                  const problem = problemFor(`isbn-${index}`)
                  const known = entry.value.trim() !== '' && validISBN(entry.value)
                  return (
                    <div key={index} className="bi-isbn-row">
                      <select
                        className="dialog-select bi-isbn-format" value={entry.format}
                        onChange={e => setEntry(index, { format: e.target.value })}
                      >
                        {ISBN_FORMATS.map(f => <option key={f.value} value={f.value}>{f.label}</option>)}
                      </select>
                      <input
                        className={`dialog-input${problem ? ' invalid' : ''}`}
                        value={entry.value} placeholder="978-…"
                        onChange={e => setEntry(index, { value: e.target.value })}
                      />
                      <span className={`bi-isbn-mark${known ? ' ok' : problem ? ' bad' : ''}`} title={problem?.message ?? (known ? 'Checks out' : '')}>
                        {known ? '✓' : problem ? '!' : ''}
                      </span>
                      <button className="bi-isbn-remove" onClick={() => setIsbns(l => l.filter((_, i) => i !== index))} title="Remove">✕</button>
                    </div>
                  )
                })}
                <button className="bi-isbn-add" onClick={() => setIsbns(l => [...l, { format: '', value: '' }])}>+ Add ISBN</button>
                <div className="bi-field-hint">
                  The eBook ISBN here becomes the EPUB identifier: this list is the one exports read,
                  while an ISBN recorded against an edition belongs to that edition&apos;s record. The
                  last digit of an ISBN checks the ones before it, so a mistyped digit is caught here.
                </div>
              </div>
            </Section>

            <Section title="Catalogue" note="What a storefront reads.">
              <Field label="Subject code 1" hint="A BISAC code, such as FIC031000.">
                <input className="dialog-input" value={draft.bisac_1 ?? ''} onChange={set('bisac_1')} />
              </Field>
              <Field label="Subject code 2">
                <input className="dialog-input" value={draft.bisac_2 ?? ''} onChange={set('bisac_2')} />
              </Field>
              <Field label="Audience">
                <select className="dialog-select" value={draft.audience ?? ''} onChange={set('audience')}>
                  {AUDIENCES.map(a => <option key={a.value} value={a.value}>{a.label}</option>)}
                </select>
              </Field>
              <Field label="Keywords" hint="Comma separated.">
                <input className="dialog-input" value={draft.keywords ?? ''} onChange={set('keywords')} />
              </Field>
              <Field label="Contributors" hint="Cover, editing, translation. Free text.">
                <input className="dialog-input" value={draft.contributors ?? ''} onChange={set('contributors')} />
              </Field>
              <div className="bi-field bi-field-wide">
                <label className="bi-field-label">Short description</label>
                <textarea
                  className="dialog-input bi-textarea" rows={4}
                  value={draft.short_description ?? ''} onChange={set('short_description')}
                  placeholder="The blurb a storefront shows."
                />
              </div>
            </Section>
          </div>
          )}
        </div>

        <footer className="bi-footer">
          <span className="bi-footer-note">{footerNote}</span>
          <button className="dialog-btn" onClick={closeMetadataDialog}>Cancel</button>
          <button className="dialog-btn primary" onClick={handleSave} disabled={blocking.length > 0}>Save</button>
        </footer>
      </div>
    </div>
  )
}
