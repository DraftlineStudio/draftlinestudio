// Book & editions.
//
// The left rail is the whole record at a glance: the shared book details, then
// every edition with its formats under it and whether each one is ready. The
// right pane is whichever of those is selected.
//
// The two halves save differently, on purpose. The shared book details are a
// form with a Save button, because changing a title is one decision. An
// edition is a record built up over months, so every keystroke on it goes
// straight onto the book and rides the ordinary five-second autosave.
//
// Two things on this screen are wired to the export flow rather than only
// described by it: a format's typeset template is the export wizard's own
// settings (formatTemplate.ts), and the locked manuscript shown on an edition
// is what an export of that edition froze. Releasing it is the one
// irreversible act here, and it is guarded by a three-step confirmation.

import { useMemo, useState } from 'react'
import {
  AttachWrapDialog, CheckCoverSource, ExportStoredArtwork, RemoveCover, SetWrapStored,
} from '../../../wailsjs/go/main/App'
import { useAppStore } from '../../store/appStore'
import { useBookStore } from '../../store/bookStore'
import type { Edition, EditionFormat, EditionIndex, ISBNEntry, Metadata } from '../../types/draftline'
import {
  AUDIENCES, LANGUAGES, blockingProblems, checkBook, isbnRows, metadataPatch,
} from './bookInfoModel'
import { coverThumbURL } from './coverModel'
import { FORMAT_WORDS, editionBadge, emptyEditionIndex, kindDot, kindForFormat } from './editionModel'
import {
  customPatch, preferencePatch, standardPatch, templateEditPatch, templateFor,
} from './formatTemplate'
import { editionDeletion, formatDeletion, type Deletion } from './deleteModel'
import {
  artworkPrompt, sourceStateOf, type ArtworkAction, type ArtworkPrompt, type ArtworkSubject,
} from './artworkModel'
import { editionSnapshot } from './snapshotModel'
import ArtworkActionDialog from './ArtworkActionDialog'
import ConfirmDeleteDialog from './ConfirmDeleteDialog'
import EditionPane from './EditionPane'
import FormatPane from './FormatPane'
import KeepWrapDialog from './KeepWrapDialog'
import ReleaseSnapshotDialog from './ReleaseSnapshotDialog'

type Draft = Partial<Metadata>

// What the rail has selected. null is the shared book details; an edition with
// no format is the edition itself.
type Selection = { editionID: string; formatID?: string } | null

// One artwork action, waiting on its confirmation.
interface PendingArt {
  editionID: string
  /** Empty for the edition's cover; a format id for that format's wrap. */
  formatID: string
  /** The archive member holding the stored bytes, when there are any. */
  member: string
  checksum: string
  action: ArtworkAction
  subject: ArtworkSubject
  prompt?: ArtworkPrompt
  /** How the edition cover re-opens its own picker. */
  onReplace?: () => void
}

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
    <section className="bi-card">
      <div className="bi-card-head">
        <span className="chapter-section-label">{title}</span>
        {note && <small>{note}</small>}
      </div>
      <div className="bi-grid">{children}</div>
    </section>
  )
}

// A section that starts shut. Setting a book up is about its title and its
// author; what a storefront reads can wait until there is a storefront.
function FoldedSection({ title, note, children }: {
  title: string
  note?: string
  children: React.ReactNode
}) {
  const [open, setOpen] = useState(false)
  return (
    <section className="bi-card">
      <button
        type="button" className="bi-disclosure" onClick={() => setOpen(!open)} aria-expanded={open}
      >
        <span className={`bi-caret${open ? ' open' : ''}`} aria-hidden="true">›</span>
        <span className="chapter-section-label">{title}</span>
        {note && <small>{note}</small>}
      </button>
      {open && <div className="bi-grid">{children}</div>}
    </section>
  )
}

export default function BookInfoDialog() {
  const book = useBookStore(s => s.book)
  const updateMetadata = useBookStore(s => s.updateMetadata)
  const addEdition = useBookStore(s => s.addEdition)
  const duplicateEdition = useBookStore(s => s.duplicateEdition)
  const removeEdition = useBookStore(s => s.removeEdition)
  const updateEdition = useBookStore(s => s.updateEdition)
  const addFormat = useBookStore(s => s.addFormat)
  const updateFormat = useBookStore(s => s.updateFormat)
  const removeFormat = useBookStore(s => s.removeFormat)
  const releaseEditionSnapshot = useBookStore(s => s.releaseEditionSnapshot)
  const closeMetadataDialog = useAppStore(s => s.closeMetadataDialog)
  const openExportWizard = useAppStore(s => s.openExportWizard)
  const setStatusMessage = useAppStore(s => s.setStatusMessage)
  const meta = book?.metadata

  const [draft, setDraft] = useState<Draft>(() => ({ ...meta }))
  // The ISBNs already on the record. There is no longer a screen for them —
  // an ISBN belongs to a format on an edition — but a save must not drop what
  // an earlier version stored, so they are read and handed straight back.
  const isbns = useMemo<ISBNEntry[]>(() => isbnRows(meta), [meta])
  const [selection, setSelection] = useState<Selection>(null)
  const [adding, setAdding] = useState<string | null>(null)
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [wrapError, setWrapError] = useState('')
  // Which format's artwork is being worked on. Attaching a wrap opens a native
  // picker and then reads a file that can be hundreds of megabytes; without
  // this the screen looks frozen and the picker can be opened again on top of
  // itself, which uploads the same artwork twice.
  const [artBusy, setArtBusy] = useState('')
  // An artwork action waiting on its confirmation, with everything the answer
  // needs: which file, where it lives, and what to do once it is safe to.
  const [pendingArt, setPendingArt] = useState<PendingArt | null>(null)
  const [releasing, setReleasing] = useState('')
  // What is about to be deleted, and how hard it will be to confirm. Nothing
  // on this screen is removed on a single click.
  const [deleting, setDeleting] = useState<Deletion | null>(null)
  const [deletingID, setDeletingID] = useState('')
  const [keeping, setKeeping] = useState('')

  const index: EditionIndex = book?.editions ?? emptyEditionIndex()
  const total = index.editions.reduce((n, e) => n + e.formats.length, 0)
  const ready = index.editions.reduce(
    (n, e) => n + e.formats.filter(f => (f.isbn13 ?? '').trim()).length, 0)

  const selected = selection ? index.editions.find(e => e.id === selection.editionID) : undefined
  const selectedFormat = selected && selection?.formatID
    ? selected.formats.find(f => f.id === selection.formatID)
    : undefined
  // A selection can be removed from under the rail — a deleted edition, a
  // deleted format — so one that no longer resolves falls back to the book.
  const resolved: Selection = selected ? { editionID: selected.id, formatID: selectedFormat?.id } : null

  const draftWords = useDraftWords(book)

  const set = (key: keyof Metadata) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) =>
    setDraft(d => ({ ...d, [key]: e.target.value }))

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

  function handleAddFormat(editionID: string, word: string) {
    const created = addFormat(editionID, kindForFormat(word), word)
    setAdding(null)
    if (created) setSelection({ editionID, formatID: created })
  }

  // The picker runs in Go and hands back a path, so a file of tens of
  // megabytes never crosses the bridge. Draftline reads what the file is and
  // records where it lives; keeping a copy is the separate act below.
  async function attachWrap(editionID: string, formatID: string) {
    if (artBusy) return
    setArtBusy(formatID)
    setWrapError('')
    try {
      const result = await AttachWrapDialog(editionID, formatID)
      if (result.cancelled) return
      if (!result.success || !result.wrap) { setWrapError(result.error || 'That artwork could not be read.'); return }
      updateFormat(formatID, { wrap: result.wrap })
    } catch (e) {
      setWrapError(String(e))
    } finally {
      setArtBusy('')
    }
  }

  async function keepWrap(formatID: string, keep: boolean) {
    if (artBusy) return
    const edition = index.editions.find(e => e.formats.some(f => f.id === formatID))
    const format = edition?.formats.find(f => f.id === formatID)
    if (!edition || !format?.wrap) return
    setArtBusy(formatID)
    setWrapError('')
    try {
      const result = await SetWrapStored(edition.id, formatID, format.wrap as never, keep)
      if (!result.success || !result.wrap) { setWrapError(result.error || 'That copy could not be made.'); return }
      updateFormat(formatID, { wrap: result.wrap })
    } catch (e) {
      setWrapError(String(e))
    } finally {
      setArtBusy('')
    }
  }

  // Ticking the box asks what it will cost. Unticking it asks something more
  // serious — whether this project is the only place the artwork exists — so
  // it goes through the artwork confirmation rather than happening at once.
  function onKeepWrap(edition: Edition, format: EditionFormat, keep: boolean) {
    if (artBusy) return
    if (!keep) return void askArtwork(wrapAction(edition, format, 'unstore'))
    setKeeping(format.id)
  }

  // Everything an artwork decision about one format's wrap needs.
  function wrapAction(edition: Edition, format: EditionFormat, action: ArtworkAction): PendingArt {
    const wrap = format.wrap
    return {
      editionID: edition.id,
      formatID: format.id,
      member: wrap?.member ?? '',
      checksum: wrap?.source_checksum ?? '',
      action,
      subject: {
        noun: 'the wrap',
        name: (format.format ?? '').trim() || 'this format',
        published: (format.status ?? '').trim().toLowerCase() === 'published',
        fileName: wrap?.file_name ?? '',
        sourcePath: wrap?.source_path ?? '',
        stored: !!wrap?.stored,
      },
    }
  }

  // Everything an artwork decision about an edition's cover needs.
  function coverAction(edition: Edition, action: ArtworkAction, act: () => void): PendingArt {
    const cover = edition.cover
    return {
      editionID: edition.id,
      formatID: '',
      member: cover?.file ?? '',
      checksum: cover?.source_checksum ?? '',
      action,
      subject: {
        noun: 'the cover',
        name: (edition.label ?? '').trim() || 'this edition',
        published: (edition.status ?? '').trim().toLowerCase() === 'published',
        fileName: cover?.file ?? '',
        sourcePath: cover?.source_path ?? '',
        // A cover is always kept in the project: that is what attaching one
        // does. The original it was made from may or may not still exist.
        stored: !!cover,
      },
      onReplace: act,
    }
  }

  async function saveCoverCopy(edition: Edition) {
    const cover = edition.cover
    if (!cover) return
    const saved = await ExportStoredArtwork(edition.id, cover.file, cover.file)
    if (!saved.success && saved.error !== 'cancelled') setWrapError(saved.error || 'That copy could not be saved.')
    else if (saved.success) setStatusMessage(`Saved ${cover.file} to ${saved.file_path}`)
  }

  async function saveWrapCopy(editionID: string, format: EditionFormat) {
    const wrap = format.wrap
    if (!wrap?.member) return
    const saved = await ExportStoredArtwork(editionID, wrap.member, wrap.file_name)
    if (!saved.success && saved.error !== 'cancelled') {
      setWrapError(saved.error || 'That copy could not be saved.')
      return
    }
    if (saved.success) setStatusMessage(`Saved ${wrap.file_name} to ${saved.file_path}`)
  }

  // ── Artwork that can be lost ─────────────────────────────────────────────

  // Asking before anything is dropped. The original's recorded location is
  // checked first, because whether this project holds the only copy is the
  // whole of the decision and it cannot be guessed from the record alone.
  async function askArtwork(art: PendingArt) {
    let state: ReturnType<typeof sourceStateOf> = 'missing'
    if (art.subject.sourcePath) {
      try {
        const report = await CheckCoverSource(art.subject.sourcePath, art.checksum)
        state = sourceStateOf(report.status, art.subject.sourcePath)
      } catch {
        state = 'unknown'
      }
    }
    setPendingArt({ ...art, prompt: artworkPrompt(art.subject, art.action, state) })
  }

  // Acting, once it has been confirmed. A forced backup is not optional and
  // not "best effort": if the file is not written, nothing is removed.
  async function runArtwork(art: PendingArt, backup: boolean) {
    setPendingArt(null)
    if (backup) {
      const saved = await ExportStoredArtwork(art.editionID, art.member, art.subject.fileName)
      if (!saved.success) {
        if (saved.error !== 'cancelled') setWrapError(saved.error || 'That copy could not be saved.')
        return
      }
    }
    if (art.action === 'unstore') return void keepWrapOrCover(art, false)
    if (art.action === 'replace') return void replaceArtwork(art)
    removeArtwork(art)
  }

  function removeArtwork(art: PendingArt) {
    if (art.formatID) {
      updateFormat(art.formatID, { wrap: undefined })
      return
    }
    if (art.onReplace) return art.onReplace()
    void RemoveCover(art.editionID)
    updateEdition(art.editionID, { cover: undefined, cover_id: '' })
  }

  async function replaceArtwork(art: PendingArt) {
    if (art.formatID) return void attachWrap(art.editionID, art.formatID)
    // The edition cover's own picker lives on the cover card, which owns the
    // large-copy choice that goes with it.
    art.onReplace?.()
  }

  function keepWrapOrCover(art: PendingArt, keep: boolean) {
    if (art.formatID) return void keepWrap(art.formatID, keep)
  }

  const releasingEdition = index.editions.find(e => e.id === releasing)
  const releasingSnapshot = editionSnapshot(index, releasingEdition)
  const keepingFormat = index.editions.flatMap(e => e.formats).find(f => f.id === keeping)

  const footerNote = blocking.length
    ? blocking[0].message
    : 'An ISBN is fixed once registered. A new trim, cover, or publisher means a new format or edition, never an edit to a published one.'

  if (!book) return null

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
              <span className="bi-rail-row-sub">Title, author, publisher, subjects</span>
            </button>

            <div className="bi-rail-head bi-rail-editions">
              <span className="chapter-section-label">Editions</span>
              <span className="bi-rail-count">
                {total ? `${ready} of ${total} formats ready` : 'None yet'}
              </span>
            </div>

            <div className="bi-rail-groups">
              {index.editions.length === 0 && (
                <p className="bi-rail-empty">
                  An edition records the cover, the ISBNs and the specification a book was
                  published with.
                </p>
              )}
              {index.editions.map(edition => {
                const badge = editionBadge(edition)
                const active = resolved?.editionID === edition.id && !resolved.formatID
                const thumb = coverThumbURL(edition)
                return (
                  <div className="bi-rail-group" key={edition.id}>
                    <button
                      type="button" className={`bi-rail-edition${active ? ' active' : ''}`}
                      onClick={() => setSelection({ editionID: edition.id })}
                    >
                      {thumb
                        ? <img className="bi-rail-thumb" src={thumb} alt="" aria-hidden="true" />
                        : <span className="bi-rail-thumb empty" aria-hidden="true" />}
                      <span className="bi-rail-edition-text">
                        <strong>{edition.label || 'Untitled edition'}</strong>
                        <small>© {edition.year || '—'} · {edition.formats.length} {edition.formats.length === 1 ? 'format' : 'formats'}</small>
                      </span>
                      <span className={`bi-badge ${badge.kind}`}>{badge.label}</span>
                    </button>

                    {edition.formats.map(format => {
                      const isbn = (format.isbn13 ?? '').trim()
                      const on = resolved?.formatID === format.id
                      return (
                        <button
                          type="button" key={format.id}
                          className={`bi-rail-format${on ? ' active' : ''}`}
                          onClick={() => setSelection({ editionID: edition.id, formatID: format.id })}
                        >
                          <span className="bi-ed-dot" style={{ background: kindDot(format.kind) }} />
                          <span className="bi-rail-format-text">
                            <span>{format.format || 'Format'}</span>
                            <small>{isbn || 'No ISBN yet'}</small>
                          </span>
                          <span className={`bi-rail-state${isbn ? ' ready' : ''}`}>
                            {isbn ? '✓ Ready' : 'No ISBN'}
                          </span>
                        </button>
                      )
                    })}

                    {/* Inside the edition it adds to. A single button at the
                        foot of the rail dropped a format into whatever was
                        selected, and nothing can be dragged between editions
                        to undo that. */}
                    <div className="bi-rail-add">
                      <button
                        type="button" className="bi-rail-add-row"
                        onClick={() => setAdding(adding === edition.id ? null : edition.id)}
                      >
                        <span aria-hidden="true">+</span>Add format
                      </button>
                      {adding === edition.id && (
                        <div className="bi-rail-add-choices">
                          {FORMAT_WORDS.map(word => (
                            <button
                              type="button" key={word} className="bi-rail-add-choice"
                              onClick={() => handleAddFormat(edition.id, word)}
                            >{word}</button>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>

            <div className="bi-rail-foot">
              <button type="button" className="dialog-btn sm" onClick={handleNewEdition}>+ New edition</button>
            </div>
          </nav>

          <div className="bi-pane">
            {selected && selectedFormat && (
              <FormatPane
                meta={draft} edition={selected} format={selectedFormat}
                options={templateFor(selected, selectedFormat)}
                advancedOpen={advancedOpen} onAdvanced={setAdvancedOpen}
                onFormat={patch => updateFormat(selectedFormat.id, patch)}
                onPreference={options => updateFormat(selectedFormat.id, preferencePatch(options))}
                onTemplate={options => updateFormat(selectedFormat.id, templateEditPatch(options))}
                onStandard={() => updateFormat(selectedFormat.id, standardPatch(templateFor(selected, selectedFormat)))}
                onCustom={() => {
                  updateFormat(selectedFormat.id, customPatch())
                  setAdvancedOpen(true)
                }}
                onRemove={() => {
                  setDeleting(formatDeletion(index, selected, selectedFormat))
                  setDeletingID(selectedFormat.id)
                }}
                onExport={() => { closeMetadataDialog(); openExportWizard(selectedFormat.id) }}
                onAddWrap={() => void attachWrap(selected.id, selectedFormat.id)}
                onReplaceWrap={() => void askArtwork(wrapAction(selected, selectedFormat, 'replace'))}
                onRemoveWrap={() => void askArtwork(wrapAction(selected, selectedFormat, 'remove'))}
                onSaveWrapCopy={() => void saveWrapCopy(selected.id, selectedFormat)}
                onKeepWrap={keep => onKeepWrap(selected, selectedFormat, keep)}
                wrapBusy={artBusy === selectedFormat.id}
                wrapError={wrapError}
              />
            )}

            {selected && !selectedFormat && (
              <EditionPane
                meta={draft} index={index} edition={selected}
                snapshot={editionSnapshot(index, selected)} draftWords={draftWords}
                onEdition={patch => updateEdition(selected.id, patch)}
                onSelectFormat={id => setSelection({ editionID: selected.id, formatID: id })}
                onAddFormat={() => setAdding(selected.id)}
                onDuplicate={() => {
                  const created = duplicateEdition(selected.id, String(new Date().getFullYear()))
                  if (created) setSelection({ editionID: created })
                }}
                onRemove={() => {
                  setDeleting(editionDeletion(index, selected))
                  setDeletingID(selected.id)
                }}
                onOpenSnapshot={() => setStatusMessage('Reading a locked manuscript back is not built yet. The text is safe in the project file.')}
                onCompare={() => setStatusMessage('Comparing a locked manuscript with the working draft is not built yet.')}
                onRelease={() => setReleasing(selected.id)}
                onAskCover={(action, act) => void askArtwork(coverAction(selected, action, act))}
                onSaveCoverCopy={() => void saveCoverCopy(selected)}
              />
            )}

            {!selected && (
              <>
                <header className="bi-pane-head">
                  <div className="bi-pane-head-text">
                    <div className="bi-pane-title"><strong>Shared book details</strong></div>
                    <span className="bi-pane-sub">
                      Carried by every edition. Anything that changes between editions lives on the
                      edition or its formats.
                    </span>
                  </div>
                </header>

                <Section title="Work">
                  <Field label="Title">
                    <input
                      className={`dialog-input${problemFor('title') ? ' invalid' : ''}`}
                      value={draft.title ?? ''} onChange={set('title')} autoFocus
                    />
                  </Field>
                  <Field label="Subtitle" hint="Optional.">
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
                  <Field label="Imprint" hint="The line the book appears under.">
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

                <FoldedSection title="Storefront details" note="What a shop reads. Nothing here is needed to write.">
                  <Field label="BISAC subject" hint="A code such as FIC031000.">
                    <input className="dialog-input" value={draft.bisac_1 ?? ''} onChange={set('bisac_1')} />
                  </Field>
                  <Field label="Second subject">
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
                </FoldedSection>

              </>
            )}
          </div>
        </div>

        <footer className="bi-footer">
          <span className="bi-footer-note">{footerNote}</span>
          <button className="dialog-btn" onClick={closeMetadataDialog}>Cancel</button>
          <button className="dialog-btn primary" onClick={handleSave} disabled={blocking.length > 0}>Save</button>
        </footer>

        {pendingArt?.prompt && (
          <ArtworkActionDialog
            prompt={pendingArt.prompt} name={pendingArt.subject.name}
            onCancel={() => setPendingArt(null)}
            onConfirm={backup => void runArtwork(pendingArt, backup)}
          />
        )}

        {deleting && (
          <ConfirmDeleteDialog
            deletion={deleting}
            onCancel={() => { setDeleting(null); setDeletingID('') }}
            onDelete={() => {
              if (deleting.kind === 'edition') {
                // Letting go of the artwork too. The save no longer carries
                // the cover of an edition the book does not have, so the file
                // would come out right either way; this is so the megabyte
                // stops being held in memory the moment it is deleted.
                void RemoveCover(deletingID)
                removeEdition(deletingID)
                setSelection(null)
              } else {
                const owner = index.editions.find(e => e.formats.some(f => f.id === deletingID))
                removeFormat(deletingID)
                setSelection(owner ? { editionID: owner.id } : null)
              }
              setStatusMessage(`Deleted ${deleting.name}.`)
              setDeleting(null)
              setDeletingID('')
            }}
          />
        )}

        {releasingEdition && releasingSnapshot && (
          <ReleaseSnapshotDialog
            edition={releasingEdition} snapshot={releasingSnapshot} draftWords={draftWords}
            onKeep={() => setReleasing('')}
            onRelease={() => {
              releaseEditionSnapshot(releasingEdition.id)
              setReleasing('')
              setStatusMessage(`Released the locked text of ${releasingEdition.label || 'that edition'}.`)
            }}
          />
        )}

        {keepingFormat?.wrap && (
          <KeepWrapDialog
            wrap={keepingFormat.wrap}
            projectName={(draft.title || 'this project') + '.draftline'}
            projectBytes={0}
            onLink={() => setKeeping('')}
            onKeep={() => { void keepWrap(keepingFormat.id, true); setKeeping('') }}
          />
        )}
      </div>
    </div>
  )
}

// The working draft's length, for the snapshot's drift row and the release
// dialog's first sentence. Counted the same way the export wizard counts it.
function useDraftWords(book: ReturnType<typeof useBookStore.getState>['book']): number {
  if (!book) return 0
  const count = (html: string) => {
    const text = html.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
    return text ? text.split(' ').length : 0
  }
  const group = (items: Array<{ content: string }> = []) => items.reduce((n, item) => n + count(item.content), 0)
  return group(book.front_matter) + group(book.body) + group(book.back_matter)
}
