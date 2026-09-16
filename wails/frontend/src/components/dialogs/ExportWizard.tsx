// Export.
//
// The flow starts from what is being exported rather than from what kind of
// file to write. An edition is one choice that carries every format registered
// under it; a reading copy and a from-scratch export are the two ways to make
// a file that belongs to no edition. Everything after that first screen — how
// many steps there are, which formats are on offer, whether the text can be
// locked — follows from it, and is computed in exportFlow.ts.
//
// The wizard owns the answers. ExportSteps draws them.

import { useEffect, useMemo, useState } from 'react'
import { useBookStore } from '../../store/bookStore'
import { useAppStore } from '../../store/appStore'
import {
  AttachCoverDialog, AttachWrapDialog, DiscardSnapshot, ExportEditionBundle,
  ExportDOCX, ExportEPUB, ExportPDF, ExportPrintPDF, FreezeSnapshot,
} from '../../../wailsjs/go/main/App'
import type { Edition, EditionSnapshot } from '../../types/draftline'
import { runExportWithFreeze, withFrozenSnapshot, type FreezeOutcome } from './snapshotModel'
import { coverThumbURL } from './coverModel'
import {
  artworkRows, bundleName, bundleRequest, editionItems, exportFormatFor, fileCount,
  findEdition, reviewRows, scratchItems, stepSummary, stepsFor, STEP_LABELS, writeField,
  type FlowItem, type FlowMode, type FlowStep, type OptionField, type SettingRow,
} from './exportFlow'
import {
  customTrimError, defaultWizardOptions, findFormat, wizardOptionsForFormat,
  type WizardOptions,
} from './exportSource'
import ExportSteps from './ExportSteps'
import { CloseGlyph, FormatGlyph, LockGlyph, SparkGlyph, StackGlyph, TickGlyph } from './exportGlyphs'

type Screen = 'start' | FlowStep | 'exporting'

interface Written { name: string; path: string }

export default function ExportWizard() {
  const { book, updateFormat, updateEdition, freezeFormat } = useBookStore()
  const { closeExportWizard, setStatusMessage } = useAppStore()
  const requestedFormatID = useAppStore(s => s.exportFormatID)
  const clearExportFormat = useAppStore(s => s.clearExportFormat)

  const [screen, setScreen] = useState<Screen>('start')
  const [mode, setMode] = useState<FlowMode>('custom')
  const [editionID, setEditionID] = useState('')
  const [selected, setSelected] = useState<string[]>([])
  const [tab, setTab] = useState('')
  const [includeArt, setIncludeArt] = useState(true)
  const [lock, setLock] = useState<boolean | null>(null)
  const [touched, setTouched] = useState<string[]>([])
  const [answers, setAnswers] = useState<Record<string, WizardOptions>>({})
  // The controls the design calls for that no exporter reads yet. They are
  // held here, keyed by format and row, so the screen remembers what was
  // chosen even though the file does not carry it.
  const [loose, setLooseState] = useState<Record<string, string | boolean>>({})
  const [artError, setArtError] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [written, setWritten] = useState<Written[]>([])
  const [note, setNote] = useState('')
  const [cancelNote, setCancelNote] = useState('')

  const index = book?.editions
  const editions = index?.editions ?? []
  const title = book?.metadata.title || 'Untitled'
  const edition = useMemo(() => findEdition(index, editionID), [index, editionID])

  const stats = useMemo(() => {
    const words = (html: string) => {
      const text = html.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
      return text ? text.split(' ').length : 0
    }
    const group = (items: Array<{ content: string }> = []) => items.reduce((sum, item) => sum + words(item.content), 0)
    if (!book) return { chapters: 0, words: 0 }
    return {
      chapters: book.body.length,
      words: words(book.copyright || '') + group(book.front_matter) + group(book.body) + group(book.back_matter),
    }
  }, [book])

  const offered = useMemo(
    () => (mode === 'edition' ? editionItems(edition) : scratchItems()),
    [mode, edition],
  )
  const chosen = useMemo(
    () => offered.filter(item => selected.includes(item.id)),
    [offered, selected],
  )

  // The answers for one format, seeded from its record the first time they are
  // asked for so that picking an edition really does prefill the wizard.
  function optionsFor(item: FlowItem): WizardOptions {
    const held = answers[item.id]
    if (held) return held
    const found = findFormat(index, item.id)
    return found ? wizardOptionsForFormat(found.edition, found.format) : defaultWizardOptions()
  }

  function setField(item: FlowItem, field: OptionField, value: string | number | boolean) {
    const next = writeField(optionsFor(item), field, value)
    setAnswers(current => ({ ...current, [item.id]: next }))
    setTouched(current => (current.includes(item.id) ? current : [...current, item.id]))
  }

  const looseKey = (item: FlowItem, row: SettingRow) => `${item.id}:${row.id}`
  function looseValue(item: FlowItem, row: SettingRow): string | boolean {
    const held = loose[looseKey(item, row)]
    if (held !== undefined) return held
    // A control nothing has answered yet shows the first choice, or on.
    return row.kind === 'select' ? String(row.choices[0]?.value ?? '') : true
  }
  function setLoose(item: FlowItem, row: SettingRow, value: string | boolean) {
    setLooseState(current => ({ ...current, [looseKey(item, row)]: value }))
    setTouched(current => (current.includes(item.id) ? current : [...current, item.id]))
  }

  const art = useMemo(() => artworkRows(chosen, edition, title), [chosen, edition, title])
  const files = fileCount(chosen, mode !== 'reading' && includeArt, art)
  const review = useMemo(() => reviewRows({
    mode, edition, items: chosen, includeArt: mode !== 'reading' && includeArt,
    art, lock, words: stats.words, fileCount: files, title,
  }), [mode, edition, chosen, includeArt, art, lock, stats.words, files, title])

  const steps = stepsFor(mode)
  const stepIndex = steps.indexOf(screen as FlowStep)
  const inWizard = stepIndex >= 0
  const activeItem = chosen.find(item => item.id === tab) ?? chosen[0]
  const trimError = activeItem && (activeItem.output === 'print-pdf' || activeItem.output === 'hc')
    ? customTrimError(optionsFor(activeItem).print)
    : ''

  // ── Starting ─────────────────────────────────────────────────────────────

  function startEdition(id: string) {
    const target = findEdition(index, id)
    if (!target) return
    setMode('edition')
    setEditionID(id)
    setSelected(target.formats.map(format => format.id))
    setTab(target.formats[0]?.id ?? '')
    setIncludeArt(true)
    setLock(null)
    setScreen('formats')
  }

  function startReading() {
    setMode('reading')
    setEditionID('')
    setSelected(['pdf'])
    setTab('pdf')
    setIncludeArt(false)
    setLock(null)
    setScreen('settings')
  }

  function startCustom() {
    setMode('custom')
    setEditionID('')
    setSelected(['docx', 'pdf'])
    setTab('docx')
    setIncludeArt(true)
    setLock(null)
    setScreen('formats')
  }

  // Opened from an edition's own Export button: land inside that edition with
  // the one format already chosen.
  useEffect(() => {
    if (!requestedFormatID) return
    const found = findFormat(index, requestedFormatID)
    if (found) {
      setMode('edition')
      setEditionID(found.edition.id)
      setSelected([found.format.id])
      setTab(found.format.id)
      setIncludeArt(true)
      setLock(null)
      setScreen('formats')
    }
    clearExportFormat()
  }, [requestedFormatID, index, clearExportFormat])

  // A book with no editions has nothing to choose between, so the flow opens
  // on the format picker instead of an empty gallery, with nothing chosen and
  // the footer saying so.

  function toggle(id: string) {
    setSelected(current => (current.includes(id) ? current.filter(x => x !== id) : [...current, id]))
  }

  // ── Moving ───────────────────────────────────────────────────────────────

  function goBack() {
    if (screen === 'start') return closeExportWizard()
    if (stepIndex <= 0) return setScreen('start')
    setScreen(steps[stepIndex - 1])
  }

  function goNext() {
    if (screen === 'start') {
      setMode('custom')
      setIncludeArt(true)
      setLock(null)
      return setScreen('settings')
    }
    if (stepIndex < 0) return
    if (stepIndex === steps.length - 1) return void runExport()
    setScreen(steps[stepIndex + 1])
  }

  const canContinue = screen === 'start'
    ? selected.length > 0
    : screen === 'formats' ? selected.length > 0
      : screen === 'finalize' ? lock !== null
        : !trimError

  const lastStep = stepIndex === steps.length - 1
  const footerHint = screen === 'start'
    ? (selected.length ? `${selected.length} format${selected.length > 1 ? 's' : ''} selected` : 'Select at least one format')
    : screen === 'finalize' && lock === null ? 'Choose whether to lock the text before exporting.'
      : screen === 'formats' && !selected.length ? 'Select at least one format.'
        : trimError ? trimError
          : lastStep ? 'A Save As window opens next.' : ''

  // ── Artwork ──────────────────────────────────────────────────────────────

  async function chooseArtwork(id: string) {
    const item = chosen.find(one => one.id === id)
    if (!item) return
    if (!edition) {
      setArtError('Artwork is kept on an edition. Register one in Book Info to attach a file here.')
      return
    }
    setArtError('')
    try {
      if (item.output === 'print-pdf' || item.output === 'hc') {
        const result = await AttachWrapDialog(edition.id, item.id)
        if (result.cancelled) return
        if (!result.success || !result.wrap) return setArtError(result.error || 'That artwork could not be read.')
        updateFormat(item.id, { wrap: result.wrap })
        return
      }
      const result = await AttachCoverDialog(edition.id, false)
      if (result.cancelled) return
      if (!result.success || !result.cover) return setArtError(result.error || 'That cover could not be read.')
      updateEdition(edition.id, { cover: result.cover, cover_id: result.cover.id })
    } catch (e) {
      setArtError(String(e))
    }
  }

  // ── Writing the files ────────────────────────────────────────────────────

  async function writeOne(item: FlowItem, outgoing: NonNullable<typeof book>) {
    const options = optionsFor(item)
    const format = exportFormatFor(item.output)
    if (format === 'epub') return await ExportEPUB(outgoing as any, options.epub as any)
    if (format === 'docx') return await ExportDOCX(outgoing as any, options.shared as any)
    if (format === 'pdf') return await ExportPDF(outgoing as any, options.pdf as any)
    return await ExportPrintPDF(outgoing as any, options.print as any)
  }

  // Each chosen format is written in turn, and the system's save dialog opens
  // once per file. Locking is the edition's decision rather than the format's,
  // so when it is taken every registered format in this export is frozen to
  // the words that went out.
  async function runExport() {
    if (!book || !chosen.length) return
    setScreen('exporting')
    setBusy(true)
    setError('')
    setNote('')
    setCancelNote('')
    setWritten([])
    try {
      if (mode === 'edition' && edition) await runBundle(edition)
      else await runLooseFiles()
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }

  // An edition is not a file. It is several objects that go out together, so
  // it is written as one publication-ready archive: one folder per format,
  // each holding its interior and the artwork that goes on it. Locking is the
  // edition's decision rather than any one format's, so the text is frozen
  // once and every chosen format is answered with that same snapshot.
  async function runBundle(one: Edition) {
    if (!book) return
    const ids = chosen.map(item => item.id)
    let outgoing = book
    let frozen: EditionSnapshot | null = null

    if (lock === true) {
      const result = await FreezeSnapshot(book as any, ids[0]) as FreezeOutcome
      if (!result.success || !result.snapshot) {
        setError(result.error || 'The text for this edition could not be frozen.')
        return
      }
      frozen = result.snapshot
      const index = ids.reduce(
        (current, id) => withFrozenSnapshot(current, id, frozen as EditionSnapshot),
        book.editions as NonNullable<typeof book.editions>,
      )
      outgoing = { ...book, editions: index }
    }

    const request = bundleRequest(one.id, includeArt, chosen, optionsFor)
    const result = await ExportEditionBundle(outgoing as any, request as any)

    if (!result.success) {
      // Nothing was written, so nothing is frozen either: a snapshot taken for
      // a bundle that never reached disk would claim an edition went out.
      if (frozen) void DiscardSnapshot(frozen.id)
      if (result.error === 'cancelled') setCancelNote('Nothing was written. The save window was closed.')
      else setError(result.error || 'That bundle could not be written.')
      return
    }

    if (frozen) {
      for (const id of ids) freezeFormat(id, frozen)
      setNote(`Froze ${frozen.word_count.toLocaleString()} words as the text this edition stands for. Exporting it again gives you these words, however far the book moves on.`)
    }
    setWritten([{ name: `${bundleName(title, one)}.zip`, path: result.file_path || '' }])
    setStatusMessage(`Exported ${bundleName(title, one)}.zip`)
  }

  // Everything that is not an edition stays what it has always been: one file
  // at a time, one save dialog each, nothing written back to the project.
  async function runLooseFiles() {
    if (!book) return
    const done: Written[] = []
    for (const item of chosen) {
      const run = await runExportWithFreeze({
        book,
        formatID: '',
        freeze: () => FreezeSnapshot(book as any, '') as Promise<FreezeOutcome>,
        write: outgoing => writeOne(item, outgoing),
        commit: () => undefined,
        discard: id => { void DiscardSnapshot(id) },
      })
      if (run.cancelled) {
        setCancelNote(run.note || 'Nothing was written. The save window was closed.')
        break
      }
      if (!run.ok) {
        setError(run.error)
        break
      }
      done.push({ name: item.label, path: run.filePath })
    }
    setWritten(done)
    if (done.length) setStatusMessage(`Exported ${done.length} ${done.length === 1 ? 'file' : 'files'}`)
  }

  if (!book) return null

  // ── The screens ──────────────────────────────────────────────────────────

  if (screen === 'exporting') {
    return <div className="dialog-overlay"><div className="dialog export-wizard export-wizard-result">
      {busy && <div className="export-result-state">
        <div className="export-progress-spinner" />
        <h2>Writing {chosen.length === 1 ? 'your file' : `${chosen.length} files`}…</h2>
        <p>Draftline is assembling the chosen formats.</p>
      </div>}
      {!busy && error && <div className="export-result-state error">
        <span className="export-result-icon">!</span>
        <h2>That file could not be created</h2>
        <p>{error}</p>
        <div className="dialog-actions">
          <button className="dialog-btn" onClick={() => setScreen(steps[steps.length - 1])}>Back</button>
          <button className="dialog-btn primary" onClick={() => void runExport()}>Try again</button>
        </div>
      </div>}
      {!busy && !error && <div className="export-result-state success">
        <span className="export-result-icon"><TickGlyph /></span>
        <h2>{written.length ? 'Your book is ready' : 'Nothing was written'}</h2>
        <div className="export-written">
          {written.map(file => <div className="export-written-row" key={file.path}><strong>{file.name}</strong><span>{file.path}</span></div>)}
        </div>
        {note && <p className="export-ed-frozen">{note}</p>}
        {cancelNote && <p className="export-ed-cancelled">{cancelNote}</p>}
        <div className="dialog-actions"><button className="dialog-btn primary" onClick={closeExportWizard}>Done</button></div>
      </div>}
    </div></div>
  }

  const header = (
    <header className="export-head">
      <span className="export-head-mark">D</span>
      <strong>Export</strong>
      <span className="export-head-sub">
        {[title, book.metadata.author, `${stats.chapters} chapters`, `${stats.words.toLocaleString()} words`]
          .filter(Boolean).join(' · ')}
      </span>
      <button type="button" className="export-head-close" onClick={closeExportWizard} aria-label="Close export">
        <CloseGlyph />
      </button>
    </header>
  )

  const footer = (hint: string) => (
    <footer className="export-foot">
      <span className="export-foot-hint">{hint}</span>
      <span className="export-foot-actions">
        <button type="button" className="dialog-btn" onClick={goBack}>{screen === 'start' ? 'Cancel' : 'Back'}</button>
        <button type="button" className="dialog-btn primary" onClick={goNext} disabled={!canContinue}>
          {lastStep && inWizard ? 'Export…' : 'Continue'}
        </button>
      </span>
    </footer>
  )

  if (screen === 'start') {
    return <div className="dialog-overlay" onKeyDown={e => e.key === 'Escape' && closeExportWizard()}>
      <div className="dialog export-wizard">
        {header}
        {editions.length > 0
          ? <div className="export-start">
            <div className="export-pane-head">
              <h2>What are you exporting?</h2>
              <p>Editions carry their cover art, format records and saved settings. Pick one to publish, or make a copy that isn’t tied to an edition.</p>
            </div>
            <div className="export-group">
              <div className="export-group-head">
                <span className="chapter-section-label">Your editions</span>
                <small>Manage in Book Info…</small>
              </div>
              <div className="export-ed-grid">
                {editions.map(one => {
                  const thumb = coverThumbURL(one)
                  const locked = one.formats.some(format => (format.snapshot_id ?? '').trim())
                  return (
                    <div className="export-ed-card" key={one.id}>
                      <div className="export-ed-cover">
                        {thumb ? <img src={thumb} alt="" /> : <span className="export-ed-plate"><em>{title}</em><i>{book.metadata.author}</i></span>}
                      </div>
                      <div className="export-ed-main">
                        <div className="export-ed-name">
                          <strong>{one.label || 'Edition'}</strong>
                          {one.status && <span className="export-badge ok">{one.status}</span>}
                        </div>
                        <span className="export-ed-sub">{[one.year, book.metadata.imprint || book.metadata.publisher].filter(Boolean).join(' · ')}</span>
                        <div className="export-ed-formats">
                          {one.formats.map(format => (
                            <div className="export-ed-format" key={format.id}>
                              <span className={`export-ed-dot ${format.kind}`} aria-hidden="true" />
                              <span className="export-ed-fmt">{format.format || 'Format'}</span>
                              <span className="export-ed-isbn">{(format.isbn13 ?? '').trim() || '—'}</span>
                            </div>
                          ))}
                          {!one.formats.length && <span className="export-ed-empty">No formats yet.</span>}
                        </div>
                        <div className="export-ed-actions">
                          <button type="button" className="dialog-btn primary sm" disabled={!one.formats.length}
                            onClick={() => startEdition(one.id)}>Export this edition</button>
                          {locked && <span className="export-ed-locked"><LockGlyph />Text locked</span>}
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
            <div className="export-group">
              <span className="chapter-section-label">Not tied to an edition</span>
              <div className="export-loose-grid">
                <button type="button" className="export-loose-card" onClick={startReading}>
                  <span className="export-loose-icon"><SparkGlyph /></span>
                  <span className="export-loose-text">
                    <strong>Share a reading copy</strong>
                    <small>A quick PDF of the current text for beta readers or reviewers. Uses the current draft, nothing is locked.</small>
                    <em>PDF · US Letter · Merriweather 12 pt</em>
                  </span>
                </button>
                <button type="button" className="export-loose-card" onClick={startCustom}>
                  <span className="export-loose-icon"><StackGlyph /></span>
                  <span className="export-loose-text">
                    <strong>Create an export from scratch</strong>
                    <small>Choose any mix of DOCX, PDF, print PDF and EPUB and set every option yourself. Nothing is written back to an edition.</small>
                    <em>Any format · current draft</em>
                  </span>
                </button>
              </div>
            </div>
          </div>
          : <div className="export-start">
            <div className="export-pane-head">
              <h2>Export the current draft</h2>
              <p>Pick everything you need in one pass. Each format gets its own settings on the next step.</p>
            </div>
            <div className="export-pick-grid">
              {scratchItems().map(item => {
                const on = selected.includes(item.id)
                return (
                  <button type="button" key={item.id} className={`export-pick-card${on ? ' selected' : ''}`}
                    onClick={() => toggle(item.id)} aria-pressed={on}>
                    <span className="export-pick-top">
                      <span className="export-pick-icon"><FormatGlyph output={item.output} /></span>
                      <span className={`export-check${on ? ' on' : ''}`} aria-hidden="true">{on && <TickGlyph />}</span>
                    </span>
                    <span className="export-pick-body">
                      <strong>{item.label}</strong>
                      <small>{item.desc}</small>
                      <em>{item.meta}</em>
                    </span>
                  </button>
                )
              })}
            </div>
            <div className="export-note-panel">
              <span className="export-note-text">
                <strong>Publishing this book?</strong>
                <small>Register an edition in Book Info to keep your cover art and format settings together. Exports can then lock the text so every reprint matches.</small>
              </span>
            </div>
          </div>}
        {/* The gallery has no footer: every card on it is its own way in. The
            format picker does, because it is a selection that has to be
            confirmed. */}
        {editions.length === 0 && footer(footerHint)}
      </div>
    </div>
  }

  const summaryInput = {
    items: chosen, touched: touched.length > 0, fromEdition: mode === 'edition',
    includeArt: mode !== 'reading' && includeArt, lock, fileCount: files,
  }

  return <div className="dialog-overlay" onKeyDown={e => e.key === 'Escape' && closeExportWizard()}>
    <div className="dialog export-wizard">
      {header}
      <div className="export-body">
        <aside className="export-rail">
          <div className="export-rail-ctx">
            <span className="chapter-section-label">{mode === 'edition' ? 'Exporting edition' : 'Exporting'}</span>
            <div className="export-rail-book">
              {edition && coverThumbURL(edition) && <img className="export-rail-cover" src={coverThumbURL(edition)} alt="" />}
              <span className="export-rail-text">
                <strong>{mode === 'edition' ? (edition?.label || 'Edition') : mode === 'reading' ? 'Reading copy' : 'Custom export'}</strong>
                <small>{mode === 'edition'
                  ? [edition?.year, `${edition?.formats.length ?? 0} formats`].filter(Boolean).join(' · ')
                  : 'Current draft · no edition'}</small>
              </span>
            </div>
          </div>
          <nav className="export-steps">
            {steps.map((id, i) => {
              const done = i < stepIndex
              const current = i === stepIndex
              return (
                <button type="button" key={id} className={`export-step${current ? ' active' : ''}${done ? ' done' : ''}`}
                  disabled={!done} onClick={() => done && setScreen(id)}>
                  <span className="export-step-num">{done ? <TickGlyph /> : i + 1}</span>
                  <span className="export-step-text">
                    <strong>{STEP_LABELS[id]}</strong>
                    <small>{stepSummary(id, summaryInput)}</small>
                  </span>
                </button>
              )
            })}
          </nav>
          <div className="export-rail-foot">
            <strong>Original protected</strong>
            <small>Exporting writes new files and never changes the open project.</small>
          </div>
        </aside>
        <main className="export-main">
          <div className="export-pane">
            <ExportSteps
              step={screen as FlowStep} mode={mode} edition={edition} title={title}
              offered={offered} selected={selected} onToggle={toggle} chosen={chosen}
              tab={activeItem?.id ?? ''} onTab={setTab} touched={touched}
              optionsFor={optionsFor} onField={setField} looseValue={looseValue} onLoose={setLoose}
              includeArt={includeArt} onIncludeArt={setIncludeArt} art={art}
              onChooseArt={id => void chooseArtwork(id)} artError={artError}
              lock={lock} onLock={setLock} review={review} words={stats.words}
            />
          </div>
          {footer(footerHint)}
        </main>
      </div>
    </div>
  </div>
}
