// Book & editions: the record a book carries and, later, the editions
// published from it. The left rail selects what is being edited; the right
// pane is the form for it. Everything here is the book's own record, saved
// into the .draftline project.

import { useState } from 'react'
import { useAppStore } from '../../store/appStore'
import { useBookStore } from '../../store/bookStore'
import type { ISBNEntry, Metadata } from '../../types/draftline'
import {
  AUDIENCES, ISBN_FORMATS, LANGUAGES, blockingProblems, checkBook, isbnRows, metadataPatch, validISBN,
} from './bookInfoModel'

type Draft = Partial<Metadata>

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
  const { book, updateMetadata } = useBookStore()
  const closeMetadataDialog = useAppStore(s => s.closeMetadataDialog)
  const meta = book?.metadata

  const [draft, setDraft] = useState<Draft>(() => ({ ...meta }))
  const [isbns, setIsbns] = useState<ISBNEntry[]>(() => isbnRows(meta))

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
            <button type="button" className="bi-rail-row active">
              <span className="bi-rail-row-name">Shared book details</span>
              <span className="bi-rail-row-sub">carried by every edition</span>
            </button>
            <div className="bi-rail-head bi-rail-divider">
              <span className="chapter-section-label">Editions</span>
            </div>
            <div className="bi-rail-empty">
              No editions registered yet. An edition records the ISBN, the cover and the
              specification a format was published with.
            </div>
          </nav>

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
              <Field label="Copyright holder" hint="Not always the author.">
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
                  The eBook ISBN becomes the EPUB identifier. The last digit of an ISBN checks
                  the ones before it, so a mistyped digit is caught here.
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
        </div>

        <footer className="bi-footer">
          <span className="bi-footer-note">
            {blocking.length
              ? blocking[0].message
              : 'An ISBN is fixed once it is registered. Changing a cover, a trim size or a publisher means a new edition, never an edit to a published one.'}
          </span>
          <button className="dialog-btn" onClick={closeMetadataDialog}>Cancel</button>
          <button className="dialog-btn primary" onClick={handleSave} disabled={blocking.length > 0}>Save</button>
        </footer>
      </div>
    </div>
  )
}
