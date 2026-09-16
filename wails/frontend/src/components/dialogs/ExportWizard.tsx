import { useMemo, useState } from 'react'
import { useBookStore } from '../../store/bookStore'
import { useAppStore } from '../../store/appStore'
import { ExportEPUB, ExportDOCX, ExportPDF, ExportPrintPDF, FreezeSnapshot } from '../../../wailsjs/go/main/App'
import { exportTextNote, frozenLabel, sharedWith, snapshotFor, withFrozenSnapshot } from './snapshotModel'
import {
  customTrimError, editionCards, exportSourceSummary, findFormat, isbnRegistrationError,
  patchChangesFormat, prefillChanges, registrableKind, registrationPatch,
  wizardOptionsForFormat, writeBackPatch, defaultWizardOptions, OUTPUT_LABELS,
  type EditionCard, type EPUBOptions, type ExportFormat, type ExportOptions,
  type PDFOptions, type PrefillChange, type PrintPDFOptions, type WizardOptions,
} from './exportSource'

type WizardStep = 'destination' | 'contents' | 'design' | 'review' | 'exporting'
type PrintPanel = 'page' | 'typography' | 'furniture' | 'title'

const FLOW_STEPS: Array<{ id: Exclude<WizardStep, 'exporting'>; label: string }> = [
  { id: 'destination', label: 'Destination' },
  { id: 'contents', label: 'Contents' },
  { id: 'design', label: 'Design' },
  { id: 'review', label: 'Review' },
]

const FORMAT_INFO: Record<ExportFormat, {
  label: string
  intent: string
  description: string
  detail: string
  extension: string
}> = {
  epub: { label: 'EPUB', intent: 'Publish an ebook', description: 'A responsive edition for e-readers and storefronts.', detail: 'Reader-controlled type · Linked contents', extension: '.epub' },
  docx: { label: 'DOCX', intent: 'Send an editable manuscript', description: 'For editors, agents, collaborators, or another word processor.', detail: 'Editable text · Microsoft Word compatible', extension: '.docx' },
  pdf: { label: 'PDF', intent: 'Share a reading copy', description: 'A fixed-layout copy for reviewers, beta readers, or archiving.', detail: 'Letter page · Fixed appearance', extension: '.pdf' },
  'print-pdf': { label: 'Print PDF', intent: 'Prepare a printed book', description: 'Book trim, mirrored margins, folios, and print typography.', detail: 'Production layout · Printer-oriented', extension: '.pdf' },
}

const TRIM_SIZES: Record<PrintPDFOptions['trimSize'], { label: string; w: string; h: string }> = {
  '5x8': { label: '5 × 8', w: '5', h: '8' },
  '5.25x8': { label: '5.25 × 8', w: '5.25', h: '8' },
  '5.5x8.5': { label: '5.5 × 8.5', w: '5.5', h: '8.5' },
  '6x9': { label: '6 × 9', w: '6', h: '9' },
  custom: { label: 'Custom', w: '', h: '' },
}

const DISPLAY_FONTS: Array<{ id: PrintPDFOptions['headingFont']; label: string; use: string }> = [
  { id: 'body', label: 'Body font', use: 'Unified' },
  { id: 'classic', label: 'EB Garamond', use: 'Classic' },
  { id: 'modern', label: 'Lato', use: 'Modern' },
  { id: 'romance', label: 'Great Vibes', use: 'Romance' },
  { id: 'scifi', label: 'Orbitron', use: 'Science fiction' },
  { id: 'fantasy', label: 'Cinzel Decorative', use: 'Fantasy' },
]

// Which text an export will actually contain. The sentence comes from the
// record rather than from this file, because a format with a frozen manuscript
// and a format without one are two different promises, and the author has to
// be told which they are getting before the file is written rather than after.

function FormatIcon({ format }: { format: ExportFormat }) {
  if (format === 'epub') return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 5.5A2.5 2.5 0 016.5 3H11v16H6.5A2.5 2.5 0 004 21.5v-16zM20 5.5A2.5 2.5 0 0017.5 3H13v16h4.5a2.5 2.5 0 012.5 2.5v-16z" /></svg>
  if (format === 'docx') return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 2.75h8l4 4V21.25H6zM14 3v4h4M8.5 11l1.25 5 1.4-3.7 1.35 3.7 1.25-5" /></svg>
  if (format === 'pdf') return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 2.75h8l4 4V21.25H6zM14 3v4h4M8.5 16v-4h1.25a1.15 1.15 0 010 2.3H8.5M12.5 12h1.1a2 2 0 010 4h-1.1zM16 16v-4h2" /></svg>
  return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 8V3h12v5M6 18H4a2 2 0 01-2-2v-6a2 2 0 012-2h16a2 2 0 012 2v6a2 2 0 01-2 2h-2M6 14h12v7H6zM18.5 11h.01" /></svg>
}

function CheckIcon() {
  return <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4 10.5l3.5 3.5L16 5.5" /></svg>
}

// CoverChip is the picture on an edition card and in the sidebar. A cover that
// exists is fetched from the same-origin address this process serves; an
// edition with no artwork gets the drawn placeholder the design uses, with the
// book's title on it, rather than an empty rectangle.
function CoverChip({ src, title, className }: { src: string; title: string; className: string }) {
  if (src) return <img className={className} src={src} alt="" />
  return <span className={`${className} placeholder`} aria-hidden="true"><em>{title}</em></span>
}

function Toggle({ checked, onChange, label, note }: { checked: boolean; onChange: (checked: boolean) => void; label: string; note?: string }) {
  return (
    <label className="export-toggle-row">
      <span><strong>{label}</strong>{note && <small>{note}</small>}</span>
      <input type="checkbox" checked={checked} onChange={event => onChange(event.target.checked)} />
      <span className="export-switch" aria-hidden="true" />
    </label>
  )
}

type Updater<T> = T | ((current: T) => T)
const applyUpdate = <T,>(current: T, next: Updater<T>): T =>
  (typeof next === 'function' ? (next as (value: T) => T)(current) : next)

export default function ExportWizard() {
  const { book, updateFormat, updateEdition, addEdition, addFormat, freezeFormat } = useBookStore()
  const { closeExportWizard, setStatusMessage } = useAppStore()
  const [step, setStep] = useState<WizardStep>('destination')
  const [format, setFormat] = useState<ExportFormat | null>(null)
  // The registered format this export is made against, or '' for a
  // from-scratch export. It is the one piece of wizard state the backend also
  // reads, through the options it travels in.
  const [formatID, setFormatID] = useState('')
  const [printPanel, setPrintPanel] = useState<PrintPanel>('page')
  const [exporting, setExporting] = useState(false)
  const [exportError, setExportError] = useState('')
  const [exportSuccess, setExportSuccess] = useState(false)
  const [exportedPath, setExportedPath] = useState('')
  const [pendingChanges, setPendingChanges] = useState<PrefillChange[]>([])
  const [savedBack, setSavedBack] = useState(false)
  const [offerISBN, setOfferISBN] = useState(false)
  const [isbnDraft, setIsbnDraft] = useState('')
  const [isbnError, setIsbnError] = useState('')
  const [registered, setRegistered] = useState('')
  const [frozenNote, setFrozenNote] = useState('')
  const [wizard, setWizard] = useState<WizardOptions>(defaultWizardOptions())

  // The four setters the design panels below use, with the shapes they had
  // when each option set was its own piece of component state. The state is
  // one object now — it is what gets written to the edition record — and this
  // keeps the panels reading as the settings they edit rather than as paths
  // into a structure.
  const options = wizard.shared
  const epubOptions = wizard.epub
  const pdfOptions = wizard.pdf
  const printOptions = wizard.print
  const setOptions = (next: Updater<ExportOptions>) => setWizard(current => ({ ...current, shared: applyUpdate(current.shared, next) }))
  const setEpubOptions = (next: Updater<EPUBOptions>) => setWizard(current => ({ ...current, epub: applyUpdate(current.epub, next) }))
  const setPdfOptions = (next: Updater<PDFOptions>) => setWizard(current => ({ ...current, pdf: applyUpdate(current.pdf, next) }))
  const setPrintOptions = (next: Updater<PrintPDFOptions>) => setWizard(current => ({ ...current, print: applyUpdate(current.print, next) }))

  const title = book?.metadata.title || 'Untitled'
  const cards = useMemo(() => editionCards(book?.editions, title), [book?.editions, title])
  const source = useMemo(() => exportSourceSummary(book?.editions, formatID, title), [book?.editions, formatID, title])
  const chosenFormat = useMemo(() => findFormat(book?.editions, formatID)?.format, [book?.editions, formatID])
  const textNote = exportTextNote(book?.editions, chosenFormat)

  const stats = useMemo(() => {
    const countWords = (html: string) => {
      const text = html.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
      return text ? text.split(' ').length : 0
    }
    const countGroup = (items: Array<{ content: string }> = []) => items.reduce((sum, item) => sum + countWords(item.content), 0)
    if (!book) return { chapters: 0, words: 0, frontCount: 0, bodyCount: 0, backCount: 0, copyrightWords: 0, frontWords: 0, bodyWords: 0, backWords: 0 }
    const copyrightWords = countWords(book.copyright || '')
    const frontWords = countGroup(book.front_matter)
    const bodyWords = countGroup(book.body)
    const backWords = countGroup(book.back_matter)
    return { chapters: book.front_matter.length + book.body.length + book.back_matter.length, words: copyrightWords + frontWords + bodyWords + backWords, frontCount: book.front_matter.length, bodyCount: book.body.length, backCount: book.back_matter.length, copyrightWords, frontWords, bodyWords, backWords }
  }, [book])

  const currentOptions = format === 'print-pdf' ? printOptions : format === 'pdf' ? pdfOptions : format === 'epub' ? epubOptions : options
  const selectedWords = stats.bodyWords + (currentOptions.includeCopyright ? stats.copyrightWords : 0) + (currentOptions.includeFrontMatter ? stats.frontWords : 0) + (currentOptions.includeBackMatter ? stats.backWords : 0)
  const currentStepIndex = FLOW_STEPS.findIndex(item => item.id === step)
  const trimError = format === 'print-pdf' ? customTrimError(printOptions) : ''

  function updateSharedOption(key: keyof ExportOptions, value: boolean) {
    setWizard(current => ({
      shared: { ...current.shared, [key]: value },
      epub: { ...current.epub, [key]: value },
      pdf: { ...current.pdf, [key]: value },
      print: { ...current.print, [key]: value },
    }))
  }

  // Choosing a registered edition fills the rest of the wizard from the
  // record: the trim it is bound at, the gutter it is bound with, and whatever
  // a previous export of this same format saved back.
  function chooseEdition(card: EditionCard, output: ExportFormat = card.outputFormat) {
    const found = findFormat(book?.editions, card.formatID)
    if (!found) return
    setFormat(output)
    setFormatID(card.formatID)
    setWizard(wizardOptionsForFormat(found.edition, found.format))
  }

  // Starting from scratch really does start from scratch: the defaults, and no
  // edition attached to what comes out.
  function chooseScratch(id: ExportFormat) {
    setFormat(id)
    setFormatID('')
    setWizard(defaultWizardOptions())
  }

  function handleBack() {
    if (step === 'destination') return closeExportWizard()
    if (step === 'contents') setStep('destination')
    if (step === 'design') setStep('contents')
    if (step === 'review') setStep('design')
  }

  function handleNext() {
    if (step === 'destination' && format) setStep('contents')
    else if (step === 'contents') setStep('design')
    else if (step === 'design' && !trimError) setStep('review')
  }

  function goToCompletedStep(target: Exclude<WizardStep, 'exporting'>) {
    const targetIndex = FLOW_STEPS.findIndex(item => item.id === target)
    if (targetIndex <= currentStepIndex || (step === 'review' && targetIndex < 3)) setStep(target)
  }

  // settleWithEdition runs after the file exists. Two different things can
  // happen, and neither of them touches the record without being asked:
  //
  //   - the export came from a registered format, and something the record
  //     prefilled was changed, so the author is asked whether the record
  //     should learn it. Nothing changed means only the wizard's own answers
  //     are remembered: the trim, the gutter and the bleed are the record's
  //     words for the published object and are never rewritten unasked;
  //   - the export came from scratch, in which case the design brief says
  //     nothing is written back at all. That became an offer rather than a
  //     rule: a file with an ISBN on it is an edition, and the offer saves
  //     typing the whole record in again.
  function settleWithEdition(chosen: ExportFormat) {
    const found = findFormat(book?.editions, formatID)
    if (found) {
      const changes = prefillChanges(found.edition, found.format, wizard, chosen)
      if (changes.length) {
        setPendingChanges(changes)
        return
      }
      const prefilled = wizardOptionsForFormat(found.edition, found.format)
      const patch = writeBackPatch(wizard, chosen, prefilled)
      if (patchChangesFormat(found.format, patch)) updateFormat(formatID, patch)
      return
    }
    if (registrableKind(chosen)) setOfferISBN(true)
  }

  function saveBackToEdition() {
    if (!format || !formatID) return
    const found = findFormat(book?.editions, formatID)
    if (!found) return
    updateFormat(formatID, writeBackPatch(wizard, format, wizardOptionsForFormat(found.edition, found.format)))
    setPendingChanges([])
    setSavedBack(true)
    setStatusMessage('Saved back to the edition record')
  }

  // Registering a from-scratch export makes a one-format edition out of it, so
  // it appears in the Book & Editions rail like any other.
  function registerExportAsEdition() {
    if (!book || !format) return
    const kind = registrableKind(format)
    if (!kind) return
    const problem = isbnRegistrationError(isbnDraft, book.editions)
    if (problem) return setIsbnError(problem)
    const editionID = addEdition(String(new Date().getFullYear()))
    if (!editionID) return setIsbnError('That edition could not be created.')
    // A new edition record normally names the one before it as what it
    // supersedes, because that is what pressing "New edition" on the Editions
    // screen means. An export made from scratch says nothing of the kind, and
    // claiming it replaces the last edition would be an invention printed on
    // the copyright page as a revision history.
    updateEdition(editionID, { previous_edition_id: undefined })
    const newFormatID = addFormat(editionID, kind)
    if (!newFormatID) return setIsbnError('That format could not be created.')
    updateFormat(newFormatID, registrationPatch(isbnDraft, format, wizard, book.metadata))
    setFormatID(newFormatID)
    setOfferISBN(false)
    setRegistered(isbnDraft.trim())
    setStatusMessage(`Registered ${isbnDraft.trim()} as a new edition`)
  }

  // freezeIfNeeded is what makes "as it was" true.
  //
  // The first export of a registered format freezes the manuscript and stamps
  // the format with it; every export after that reads the frozen words instead
  // of the draft. Freezing before the file is written, rather than after, means
  // the book handed to the exporter is already the book the record names — so
  // the file that comes out and the text the ISBN stands for cannot disagree,
  // even if the save that stores them is still five seconds away.
  //
  // It returns the book to export, which is the book on screen with the new
  // stamp on it. The store is updated too, by the same pure transform.
  async function freezeIfNeeded(): Promise<typeof book | null> {
    if (!book) return null
    if (!formatID || !book.editions) return book
    const found = findFormat(book.editions, formatID)
    if (!found) return book
    const already = snapshotFor(book.editions, found.format)
    if (already) {
      const when = frozenLabel(already.frozen)
      setFrozenNote(`Exported the text frozen for this ISBN${when ? ` on ${when}` : ''}, not the draft on screen.`)
      return book
    }
    const result = await FreezeSnapshot(book as any, formatID)
    if (!result.success || !result.snapshot) {
      setExportError(result.error || 'The text for this edition could not be frozen.')
      return null
    }
    freezeFormat(formatID, result.snapshot)
    const editions = withFrozenSnapshot(book.editions, formatID, result.snapshot)
    const shared = sharedWith(editions, result.snapshot.id, formatID)
    setFrozenNote(result.reused && shared.length
      ? `These are the same words already frozen for ${shared.join(' and ')}. Your project file stores them once and both ISBNs point at them.`
      : `Froze ${result.snapshot.word_count.toLocaleString()} words as the text this ISBN stands for. Exporting it again gives you these words, however far the book moves on.`)
    return { ...book, editions }
  }

  async function handleExport() {
    if (!book || !format) return
    setStep('exporting')
    setExporting(true)
    setExportError('')
    setExportSuccess(false)
    setFrozenNote('')
    try {
      const outgoing = await freezeIfNeeded()
      if (!outgoing) return
      let result: { success: boolean; file_path?: string; error?: string }
      if (format === 'epub') result = await ExportEPUB(outgoing as any, epubOptions as any)
      else if (format === 'docx') result = await ExportDOCX(outgoing as any, options as any)
      else if (format === 'pdf') result = await ExportPDF(outgoing as any, pdfOptions as any)
      else result = await ExportPrintPDF(outgoing as any, printOptions as any)
      if (result.success) {
        setExportSuccess(true)
        setExportedPath(result.file_path || '')
        setStatusMessage(`Exported to ${result.file_path}`)
        settleWithEdition(format)
      } else if (result.error === 'cancelled') setStep('review')
      else setExportError(result.error || 'Export failed')
    } catch (error) {
      setExportError(String(error))
    } finally {
      setExporting(false)
    }
  }

  function renderDestination() {
    return <>
      <div className="export-step-heading"><span className="export-eyebrow">Step 1 of 4</span><h2>Where is this book going?</h2><p>Editions you have registered already carry an ISBN, a trim size, and their own cover art. Pick one and Draftline fills in the remaining steps.</p></div>
      {cards.length > 0 && <section className="export-ed-section">
        <div className="export-ed-section-head"><span className="export-eyebrow">Registered editions</span><small>{cards.length} {cards.length === 1 ? 'template' : 'templates'} from this project’s ISBNs</small></div>
        <div className="export-ed-grid">
          {cards.map(card => (
            <div key={card.formatID} className="export-ed-cell">
              <button type="button" className={`export-ed-card${formatID === card.formatID && format === card.outputFormat ? ' selected' : ''}`} onClick={() => chooseEdition(card)} aria-pressed={formatID === card.formatID && format === card.outputFormat}>
                <CoverChip src={card.thumbURL} title={card.title} className="export-ed-cover" />
                <span className="export-ed-copy">
                  <span className="export-ed-topline"><strong>{card.format}</strong><em>{card.edition}</em></span>
                  <span className="export-ed-isbn">{card.isbn13}</span>
                  <span className="export-ed-spec">{card.spec}</span>
                  <span className="export-ed-foot"><i className={`export-ed-badge ${card.badgeKind}`}>{card.badge}</i><small>{card.out}</small></span>
                </span>
              </button>
              {card.altOutput && <button type="button" className={`export-ed-alt${formatID === card.formatID && format === card.altOutput ? ' selected' : ''}`} onClick={() => chooseEdition(card, card.altOutput!)} aria-pressed={formatID === card.formatID && format === card.altOutput}>{card.altLabel}<em>Same ISBN, cover and copyright page — for reviewers</em></button>}
            </div>
          ))}
        </div>
      </section>}
      <section className="export-ed-section">
        <div className="export-ed-section-head"><span className="export-eyebrow">{cards.length > 0 ? 'Or start from scratch' : 'Choose a destination'}</span><small>No ISBN attached — Draftline will offer to register one afterwards.</small></div>
        <div className="export-destination-grid">
          {(Object.entries(FORMAT_INFO) as Array<[ExportFormat, typeof FORMAT_INFO[ExportFormat]]>).map(([id, info]) => (
            <button type="button" key={id} className={`export-destination-card${format === id && !formatID ? ' selected' : ''}`} onClick={() => chooseScratch(id)} aria-pressed={format === id && !formatID}>
              <span className="export-format-icon"><FormatIcon format={id} /></span>
              <span className="export-destination-copy"><span className="export-destination-topline"><strong>{info.intent}</strong></span><span>{info.description}</span><small>{info.label} {info.extension} · {info.detail}</small></span>
              <span className="export-card-check"><CheckIcon /></span>
            </button>
          ))}
        </div>
      </section>
    </>
  }

  function renderContents() {
    const copyrightNote = source
      ? 'Generated from this edition’s record, with your own copyright page kept underneath it.'
      : book?.copyright ? 'Rights, edition, and publication notice.' : 'No copyright content has been written.'
    const sectionCards = [
      { id: 'copyright', title: 'Copyright page', description: copyrightNote, count: source ? 'Generated' : stats.copyrightWords ? `${stats.copyrightWords.toLocaleString()} words` : 'Empty', checked: currentOptions.includeCopyright, locked: false, change: (value: boolean) => updateSharedOption('includeCopyright', value) },
      { id: 'front', title: 'Front matter', description: 'Dedication, acknowledgements, preface, and other opening material.', count: `${stats.frontCount} ${stats.frontCount === 1 ? 'item' : 'items'}`, checked: currentOptions.includeFrontMatter, locked: false, change: (value: boolean) => updateSharedOption('includeFrontMatter', value) },
      { id: 'body', title: 'Manuscript', description: 'The complete body of the book in manuscript order.', count: `${stats.bodyCount} ${stats.bodyCount === 1 ? 'chapter' : 'chapters'}`, checked: true, locked: true, change: () => undefined },
      { id: 'back', title: 'Back matter', description: 'Afterword, notes, bibliography, and other closing material.', count: `${stats.backCount} ${stats.backCount === 1 ? 'item' : 'items'}`, checked: currentOptions.includeBackMatter, locked: false, change: (value: boolean) => updateSharedOption('includeBackMatter', value) },
    ]
    return <>
      <div className="export-step-heading"><span className="export-eyebrow">Step 2 of 4</span><h2>What belongs in this edition?</h2><p>The manuscript always travels with the export. Choose which surrounding book sections belong with it.</p></div>
      <div className="export-content-grid">
        {sectionCards.map(section => (
          <button type="button" key={section.id} className={`export-content-card${section.checked ? ' selected' : ''}`} disabled={section.locked} onClick={() => section.change(!section.checked)} aria-pressed={section.checked}>
            <span className="export-content-check"><CheckIcon /></span><span><strong>{section.title}</strong><small>{section.description}</small></span><em>{section.count}</em>
          </button>
        ))}
      </div>
      <div className="export-selection-total"><span>{selectedWords.toLocaleString()} words selected</span><span>{FORMAT_INFO[format!].label} edition</span></div>
    </>
  }

  function renderPrintPagePanel() {
    return <div className="export-design-panel">
      <div className="export-setting-block export-setting-block-wide"><h3>Trim size</h3><p>The finished dimensions of the bound book.</p>
        <div className="export-choice-strip export-trim-choices">
          {(Object.entries(TRIM_SIZES) as Array<[PrintPDFOptions['trimSize'], { label: string; w: string; h: string }]>).map(([id, size]) => <button type="button" key={id} className={printOptions.trimSize === id ? 'selected' : ''} onClick={() => setPrintOptions(current => ({ ...current, trimSize: id, customWidth: size.w || current.customWidth, customHeight: size.h || current.customHeight }))}>{size.label}{id !== 'custom' && <small>inches</small>}</button>)}
        </div>
        {printOptions.trimSize === 'custom' && <div className="export-inline-fields"><label>Width <input value={printOptions.customWidth} onChange={event => setPrintOptions(current => ({ ...current, customWidth: event.target.value }))} inputMode="decimal" /></label><span>×</span><label>Height <input value={printOptions.customHeight} onChange={event => setPrintOptions(current => ({ ...current, customHeight: event.target.value }))} inputMode="decimal" /></label><span>inches</span></div>}
        {trimError && <p className="export-trim-error" role="alert">{trimError}</p>}
      </div>
      <div className="export-setting-block"><h3>Page margins</h3><p>Interior breathing room and binding allowance.</p><div className="export-number-grid">
        {([['gutterMargin', 'Inside', printOptions.gutterMargin], ['outerMargin', 'Outside', printOptions.outerMargin], ['topMargin', 'Top', printOptions.topMargin], ['bottomMargin', 'Bottom', printOptions.bottomMargin]] as const).map(([key, label, value]) => <label key={key}>{label}<span><input value={value} onChange={event => setPrintOptions(current => ({ ...current, [key]: event.target.value }))} inputMode="decimal" /> in</span></label>)}
      </div></div>
      <div className="export-setting-block"><h3>Production page</h3><p>Binding and printer setup.</p><Toggle checked={printOptions.mirroredMargins} onChange={value => setPrintOptions(current => ({ ...current, mirroredMargins: value }))} label="Mirror inside margins" /><Toggle checked={printOptions.chapterStartsRecto} onChange={value => setPrintOptions(current => ({ ...current, chapterStartsRecto: value }))} label="Chapters begin on recto" note="Right-hand starts may insert an unnumbered blank that still counts in pagination" /><div className="export-compact-field"><label>Bleed</label><span><input value={printOptions.bleed} onChange={event => setPrintOptions(current => ({ ...current, bleed: event.target.value }))} inputMode="decimal" /> in</span></div><Toggle checked={printOptions.includeCropMarks} onChange={value => setPrintOptions(current => ({ ...current, includeCropMarks: value }))} label="Include crop marks" /></div>
    </div>
  }

  function renderPrintTypographyPanel() {
    return <div className="export-design-panel">
      <div className="export-setting-block export-setting-block-wide"><h3>Body typeface</h3><p>Choose the reading character of the printed page.</p><div className="export-font-choices">
        {(['merriweather', 'lato'] as const).map(font => <button type="button" key={font} className={`${font}${printOptions.fontFamily === font ? ' selected' : ''}`} onClick={() => setPrintOptions(current => ({ ...current, fontFamily: font }))}><span>Aa</span><small>{font === 'merriweather' ? 'Merriweather' : 'Lato'}</small></button>)}
      </div></div>
      <div className="export-setting-block export-setting-block-wide"><h3>Chapter heading typeface</h3><p>A display face for chapter titles and generated Contents headings. Body text remains in the reading face above.</p><div className="export-display-font-choices">
        {DISPLAY_FONTS.map(font => <button type="button" key={font.id} className={`${font.id}${printOptions.headingFont === font.id ? ' selected' : ''}`} onClick={() => setPrintOptions(current => ({ ...current, headingFont: font.id }))}><span>Aa</span><strong>{font.label}</strong><small>{font.use}</small></button>)}
      </div></div>
      <div className="export-setting-block"><h3>Composition</h3><div className="export-select-grid">
        <label>Type size<select value={printOptions.fontSize} onChange={event => setPrintOptions(current => ({ ...current, fontSize: Number(event.target.value) as PDFOptions['fontSize'] }))}><option value={9}>9 pt</option><option value={10}>10 pt</option><option value={11}>11 pt</option><option value={12}>12 pt</option></select></label>
        <label>Line spacing<select value={printOptions.lineHeight} onChange={event => setPrintOptions(current => ({ ...current, lineHeight: Number(event.target.value) as PrintPDFOptions['lineHeight'] }))}><option value={1.3}>Tight · 1.3</option><option value={1.4}>Book · 1.4</option><option value={1.5}>Relaxed · 1.5</option><option value={1.6}>Open · 1.6</option></select></label>
        <label>Alignment<select value={printOptions.textAlign} onChange={event => setPrintOptions(current => ({ ...current, textAlign: event.target.value as PrintPDFOptions['textAlign'] }))}><option value="left">Left aligned</option><option value="justify">Justified</option></select></label>
        <label>First-line indent<span className="export-field-with-unit"><input value={printOptions.paragraphIndent} onChange={event => setPrintOptions(current => ({ ...current, paragraphIndent: event.target.value }))} inputMode="decimal" /> in</span></label>
      </div></div>
      <div className="export-setting-block"><h3>Chapter opening</h3><Toggle checked={printOptions.dropCap} onChange={value => setPrintOptions(current => ({ ...current, dropCap: value }))} label="Opening drop cap" />{printOptions.dropCap && <div className="export-compact-field"><label>Depth</label><select value={printOptions.dropCapLines} onChange={event => setPrintOptions(current => ({ ...current, dropCapLines: Number(event.target.value) as PrintPDFOptions['dropCapLines'] }))}><option value={2}>2 lines</option><option value={3}>3 lines</option><option value={4}>4 lines</option></select></div>}</div>
    </div>
  }

  function renderPrintFurniturePanel() {
    return <div className="export-design-panel">
      <div className="export-setting-block"><h3>Running furniture</h3><p>Quiet navigation around the manuscript.</p><div className="export-compact-field"><label>Header and folio font</label><select value={printOptions.furnitureFont} onChange={event => setPrintOptions(current => ({ ...current, furnitureFont: event.target.value as PrintPDFOptions['furnitureFont'] }))}>{DISPLAY_FONTS.map(font => <option key={font.id} value={font.id}>{font.label}</option>)}</select></div><Toggle checked={printOptions.runningHeaders} onChange={value => setPrintOptions(current => ({ ...current, runningHeaders: value }))} label="Running headers" note="Book title on verso, chapter on recto" />{printOptions.runningHeaders && <div className="export-compact-field"><label>Header style</label><select value={printOptions.headerStyle} onChange={event => setPrintOptions(current => ({ ...current, headerStyle: event.target.value as PrintPDFOptions['headerStyle'] }))}><option value="smallcaps">Small caps</option><option value="italic">Italic</option><option value="normal">Normal</option></select></div>}<div className="export-compact-field"><label>Page numbers</label><select value={printOptions.pageNumberPosition} onChange={event => setPrintOptions(current => ({ ...current, pageNumberPosition: event.target.value as PrintPDFOptions['pageNumberPosition'] }))}><option value="bottom-center">Bottom center</option><option value="bottom-outside">Bottom outside</option><option value="top-outside">Top outside</option></select></div></div>
      <div className="export-setting-block"><h3>Generated pages</h3><p>Pages Draftline composes around your manuscript.</p><Toggle checked={printOptions.generateHalfTitle} onChange={value => setPrintOptions(current => ({ ...current, generateHalfTitle: value }))} label="Half-title page" /><Toggle checked={printOptions.generateTOC} onChange={value => setPrintOptions(current => ({ ...current, generateTOC: value }))} label="Table of contents" /></div>
      <div className="export-setting-block export-page-sample" aria-hidden="true"><span className={`export-page-sample-header display-${printOptions.furnitureFont}`}>{book?.metadata.title || 'BOOK TITLE'}</span><div className={`export-page-sample-title display-${printOptions.headingFont}`}>Chapter One</div><div className="export-page-sample-lines"><i /><i /><i /><i /><i /><i /></div><span className={`export-page-sample-number display-${printOptions.furnitureFont}`}>17</span></div>
    </div>
  }

  function renderPrintTitlePanel() {
    return <div className="export-design-panel">
      <div className={`export-setting-block export-title-page-sample ${printOptions.titlePageStyle} ${printOptions.titlePageFont}`} aria-hidden="true"><div><strong>{book?.metadata.title || 'Untitled'}</strong>{printOptions.titlePageShowAuthor && <span>by {book?.metadata.author || 'Author'}</span>}{printOptions.titlePageShowPublisher && <small>{book?.metadata.publisher || 'Publisher'}</small>}</div></div>
      <div className="export-setting-block"><h3>Title page composition</h3><p>Set the tone of the first full title page.</p><div className="export-compact-field"><label>Display typeface</label><select value={printOptions.titlePageFont} onChange={event => setPrintOptions(current => ({ ...current, titlePageFont: event.target.value as PrintPDFOptions['titlePageFont'] }))}>{DISPLAY_FONTS.map(font => <option key={font.id} value={font.id}>{font.label} / {font.use}</option>)}</select></div><label className="export-option-label">Placement</label><div className="export-choice-strip">{(['classic', 'minimal', 'dramatic'] as const).map(style => <button type="button" key={style} className={printOptions.titlePageStyle === style ? 'selected' : ''} onClick={() => setPrintOptions(current => ({ ...current, titlePageStyle: style }))}>{style[0].toUpperCase() + style.slice(1)}</button>)}</div><Toggle checked={printOptions.titlePageShowAuthor} onChange={value => setPrintOptions(current => ({ ...current, titlePageShowAuthor: value }))} label="Show author" /><Toggle checked={printOptions.titlePageShowPublisher} onChange={value => setPrintOptions(current => ({ ...current, titlePageShowPublisher: value }))} label="Show publisher or imprint" /></div>
    </div>
  }

  function renderDesign() {
    if (!format) return null
    const info = FORMAT_INFO[format]
    return <>
      <div className="export-step-heading export-step-heading-row"><div><span className="export-eyebrow">Step 3 of 4</span><h2>Shape the {info.label} edition</h2><p>{format === 'print-pdf' ? 'Make a few deliberate book-design decisions; everything else receives sensible defaults.' : 'Choose how this edition should behave after it leaves Draftline.'}</p></div><span className="export-heading-format"><FormatIcon format={format} />{info.label}</span></div>
      {format === 'epub' && <div className="export-simple-design export-epub-design"><div className="export-reader-preview" aria-hidden="true"><div className={`${epubOptions.chapterStyle} ${epubOptions.paragraphStyle} ${epubOptions.fontFamily}`}><span>Chapter One</span><i /><i /><i /><b>{epubOptions.sceneBreakStyle === 'asterism' ? '⁂' : epubOptions.sceneBreakStyle === 'rule' ? '—' : ''}</b><i /><i /></div></div><div className="export-reading-controls"><h3>Responsive reader edition</h3><p>Choose the publisher defaults. Compatible readers can still override type, size, spacing, and theme for accessibility.</p><div className="export-select-grid"><label>Typeface<select value={epubOptions.fontFamily} onChange={event => setEpubOptions(current => ({ ...current, fontFamily: event.target.value as EPUBOptions['fontFamily'] }))}><option value="reader">Reader default (smallest file)</option><option value="merriweather">Embed Merriweather</option><option value="lato">Embed Lato</option></select></label><label>Paragraphs<select value={epubOptions.paragraphStyle} onChange={event => setEpubOptions(current => ({ ...current, paragraphStyle: event.target.value as EPUBOptions['paragraphStyle'] }))}><option value="indented">Book-style indents</option><option value="spaced">Space between paragraphs</option></select></label><label>Alignment<select value={epubOptions.textAlign} onChange={event => setEpubOptions(current => ({ ...current, textAlign: event.target.value as EPUBOptions['textAlign'] }))}><option value="reader">Reader default</option><option value="left">Left aligned</option><option value="justify">Justified</option></select></label><label>Chapter opening<select value={epubOptions.chapterStyle} onChange={event => setEpubOptions(current => ({ ...current, chapterStyle: event.target.value as EPUBOptions['chapterStyle'] }))}><option value="classic">Classic / lowered title</option><option value="minimal">Minimal / compact title</option></select></label><label>Scene breaks<select value={epubOptions.sceneBreakStyle} onChange={event => setEpubOptions(current => ({ ...current, sceneBreakStyle: event.target.value as EPUBOptions['sceneBreakStyle'] }))}><option value="asterism">Asterism</option><option value="rule">Short rule</option><option value="space">Open space</option></select></label></div><ul><li><CheckIcon />Linked contents and chapter navigation</li><li><CheckIcon />{source ? `Identifier, cover and copyright page from ${source.name}` : 'Storefront metadata from Book Details'}</li><li><CheckIcon />Valid reflowable XHTML, never raw editor markup</li></ul></div></div>}
      {format === 'docx' && <div className="export-simple-design"><div className="export-document-preview" aria-hidden="true"><div><i /><i /><i /><i /><i /><i /><i /></div></div><div><h3>Clean editable manuscript</h3><p>Draftline will favor familiar Word styles and editable structure over a locked visual design.</p><ul><li><CheckIcon />One heading per chapter</li><li><CheckIcon />Explicit page breaks</li><li><CheckIcon />Readable body-text defaults</li></ul></div></div>}
      {format === 'pdf' && <div className="export-simple-design export-pdf-design"><div className="export-document-preview pdf" aria-hidden="true"><div className={pdfOptions.fontFamily}><span>{book?.metadata.title || 'Untitled'}</span><i /><i /><i /><i /><i /></div></div><div className="export-reading-controls"><h3>Comfortable reading copy</h3><p>A fixed-layout edition for screens, reviewers, home printers, or archiving.</p><div className="export-select-grid"><label>Page size<select value={pdfOptions.pageSize} onChange={event => setPdfOptions(current => ({ ...current, pageSize: event.target.value as PDFOptions['pageSize'] }))}><option value="letter">US Letter</option><option value="a4">A4</option><option value="6x9">6 x 9 in</option><option value="5.5x8.5">5.5 x 8.5 in</option><option value="5x8">5 x 8 in</option></select></label><label>Typeface<select value={pdfOptions.fontFamily} onChange={event => setPdfOptions(current => ({ ...current, fontFamily: event.target.value as PDFOptions['fontFamily'] }))}><option value="merriweather">Merriweather</option><option value="lato">Lato</option></select></label><label>Type size<select value={pdfOptions.fontSize} onChange={event => setPdfOptions(current => ({ ...current, fontSize: Number(event.target.value) as PDFOptions['fontSize'] }))}><option value={11}>11 pt</option><option value={12}>12 pt</option><option value={14}>14 pt</option></select></label><label>Line spacing<select value={pdfOptions.lineHeight} onChange={event => setPdfOptions(current => ({ ...current, lineHeight: Number(event.target.value) as PDFOptions['lineHeight'] }))}><option value={1.3}>Tight / 1.3</option><option value={1.4}>Book / 1.4</option><option value={1.5}>Relaxed / 1.5</option><option value={1.6}>Open / 1.6</option></select></label><label>Alignment<select value={pdfOptions.textAlign} onChange={event => setPdfOptions(current => ({ ...current, textAlign: event.target.value as PDFOptions['textAlign'] }))}><option value="left">Left aligned</option><option value="justify">Justified</option></select></label><label>First-line indent<span className="export-field-with-unit"><input value={pdfOptions.paragraphIndent} onChange={event => setPdfOptions(current => ({ ...current, paragraphIndent: event.target.value }))} inputMode="decimal" /> in</span></label></div></div></div>}
      {format === 'print-pdf' && <><div className="export-design-tabs" role="tablist">{([['page', 'Page'], ['typography', 'Typography'], ['furniture', 'Furniture'], ['title', 'Title page']] as Array<[PrintPanel, string]>).map(([id, label]) => <button type="button" key={id} role="tab" aria-selected={printPanel === id} className={printPanel === id ? 'selected' : ''} onClick={() => setPrintPanel(id)}>{label}</button>)}</div>{printPanel === 'page' && renderPrintPagePanel()}{printPanel === 'typography' && renderPrintTypographyPanel()}{printPanel === 'furniture' && renderPrintFurniturePanel()}{printPanel === 'title' && renderPrintTitlePanel()}</>}
    </>
  }

  function renderReview() {
    if (!format) return null
    const included = [currentOptions.includeCopyright && (source || book?.copyright) ? 'Copyright page' : '', currentOptions.includeFrontMatter && stats.frontCount ? `${stats.frontCount} front matter ${stats.frontCount === 1 ? 'item' : 'items'}` : '', `${stats.bodyCount} manuscript ${stats.bodyCount === 1 ? 'chapter' : 'chapters'}`, currentOptions.includeBackMatter && stats.backCount ? `${stats.backCount} back matter ${stats.backCount === 1 ? 'item' : 'items'}` : ''].filter(Boolean)
    return <>
      <div className="export-step-heading"><span className="export-eyebrow">Step 4 of 4</span><h2>Review this edition</h2><p>One final check before choosing where to save the file.</p></div>
      <div className="export-review-layout">
        <div className="export-review-hero"><span className="export-format-icon"><FormatIcon format={format} /></span><div><small>{FORMAT_INFO[format].intent}</small><strong>{book?.metadata.title || 'Untitled'}</strong><span>{book?.metadata.author ? `by ${book.metadata.author}` : 'Author not set'}</span></div><em>{FORMAT_INFO[format].extension}</em></div>
        {source
          ? <div className="export-review-card"><div className="export-review-section"><span>Edition</span><strong>{source.name}</strong><p>{source.isbn13} · identifier, cover art and copyright page come from this record</p></div><button type="button" onClick={() => setStep('destination')}>Change edition</button></div>
          : <div className="export-review-card"><div className="export-review-section"><span>Edition</span><strong>Not registered</strong><p>Exported from scratch. Draftline will offer to attach an ISBN afterwards.</p></div><button type="button" onClick={() => setStep('destination')}>Change</button></div>}
        <div className="export-review-card"><div className="export-review-section"><span>Contents</span><strong>{selectedWords.toLocaleString()} words</strong><p>{included.join(' · ')}</p></div><button type="button" onClick={() => setStep('contents')}>Edit contents</button></div>
        <div className="export-review-card"><div className="export-review-section"><span>Design</span><strong>{format === 'print-pdf' ? `${TRIM_SIZES[printOptions.trimSize].label}${printOptions.trimSize === 'custom' ? ` (${printOptions.customWidth} x ${printOptions.customHeight})` : ''} / ${printOptions.fontFamily} ${printOptions.fontSize} pt` : format === 'pdf' ? `${pdfOptions.pageSize.toUpperCase()} / ${pdfOptions.fontFamily} ${pdfOptions.fontSize} pt` : format === 'epub' ? `${epubOptions.fontFamily === 'reader' ? 'Reader typography' : `Embedded ${epubOptions.fontFamily}`} / ${epubOptions.paragraphStyle}` : 'Editable manuscript layout'}</strong><p>{format === 'print-pdf' ? `${printOptions.textAlign === 'justify' ? 'Justified' : 'Left aligned'} / ${printOptions.runningHeaders ? 'Running headers' : 'No running headers'} / ${printOptions.generateTOC ? 'Contents page' : 'No contents page'}` : format === 'epub' ? `${epubOptions.chapterStyle} chapters / ${epubOptions.sceneBreakStyle} scene breaks / ${epubOptions.textAlign} alignment` : FORMAT_INFO[format].detail}</p></div><button type="button" onClick={() => setStep('design')}>Edit design</button></div>
      </div>
      <div className="export-save-note"><CheckIcon /><span>Draftline will open your system’s save dialog next. Your writing is not changed. {textNote}</span></div>
    </>
  }

  // The two things that can be offered once the file exists.
  function renderAftermath() {
    return <>
      {frozenNote && <p className="export-ed-frozen">{frozenNote}</p>}
      {pendingChanges.length > 0 && <div className="export-ed-writeback">
        <strong>Save these back to {source ? source.name : 'the edition'}?</strong>
        <ul>{pendingChanges.map(change => <li key={change.label}><span>{change.label}</span><em>{change.from} → {change.to}</em></li>)}</ul>
        <p>An ISBN never changes. These are the settings this edition is exported with, not the number it is sold under.</p>
        <div className="dialog-actions"><button className="dialog-btn" onClick={() => setPendingChanges([])}>Not this time</button><button className="dialog-btn primary" onClick={saveBackToEdition}>Save to the edition</button></div>
      </div>}
      {savedBack && <p className="export-ed-settled">Saved to the edition record.</p>}
      {offerISBN && <div className="export-ed-writeback">
        <strong>Does this file have an ISBN?</strong>
        <p>Attaching one registers it as an edition, so the number, the specification you just used and the copyright page stay with the book. Leave it blank if it has none.</p>
        <label className="export-ed-isbn-field">ISBN<input value={isbnDraft} onChange={event => { setIsbnDraft(event.target.value); setIsbnError('') }} placeholder="978-…" inputMode="numeric" /></label>
        {isbnError && <p className="export-trim-error" role="alert">{isbnError}</p>}
        <div className="dialog-actions"><button className="dialog-btn" onClick={() => setOfferISBN(false)}>No ISBN</button><button className="dialog-btn primary" onClick={registerExportAsEdition}>Register this edition</button></div>
      </div>}
      {registered && <p className="export-ed-settled">Registered {registered}. It is on the Book &amp; Editions screen now.</p>}
    </>
  }

  if (!book) return null
  if (step === 'exporting') return <div className="dialog-overlay" onKeyDown={event => event.key === 'Escape' && !exporting && closeExportWizard()}><div className="dialog export-wizard export-wizard-result">
    {exporting && <div className="export-result-state"><div className="export-progress-spinner" /><span className="export-eyebrow">Building edition</span><h2>Creating your {format ? FORMAT_INFO[format].label : 'export'}…</h2><p>Draftline is assembling the selected book sections.</p></div>}
    {exportError && <div className="export-result-state error"><span className="export-result-icon">!</span><span className="export-eyebrow">Export interrupted</span><h2>That file could not be created</h2><p>{exportError}</p><div className="dialog-actions"><button className="dialog-btn" onClick={() => setStep('review')}>Back to review</button><button className="dialog-btn primary" onClick={handleExport}>Try again</button></div></div>}
    {exportSuccess && <div className="export-result-state success"><span className="export-result-icon"><CheckIcon /></span><span className="export-eyebrow">Edition complete</span><h2>Your book is ready</h2><p>{exportedPath}</p>{renderAftermath()}<div className="dialog-actions"><button className="dialog-btn primary" onClick={closeExportWizard}>Done</button></div></div>}
  </div></div>

  return <div className="dialog-overlay" onKeyDown={event => event.key === 'Escape' && closeExportWizard()}><div className="dialog export-wizard">
    <header className="export-wizard-header"><div><span className="export-wizard-mark">D</span><strong>Create an edition</strong></div><button type="button" className="export-close" onClick={closeExportWizard} aria-label="Close export wizard">×</button></header>
    <nav className="export-stepper" aria-label="Export progress">{FLOW_STEPS.map((item, index) => { const complete = index < currentStepIndex; const active = item.id === step; return <button type="button" key={item.id} disabled={index > currentStepIndex} className={`${active ? 'active' : ''}${complete ? ' complete' : ''}`} onClick={() => goToCompletedStep(item.id)} aria-current={active ? 'step' : undefined}><span>{complete ? <CheckIcon /> : index + 1}</span><em>{item.label}</em></button> })}</nav>
    <div className="export-wizard-body">
      <aside className="export-edition-summary"><span className="export-summary-kicker">Current book</span><h3>{title}</h3><p>{book.metadata.author || 'Author not set'}</p><div className="export-summary-stats"><span><strong>{stats.bodyCount}</strong> chapters</span><span><strong>{stats.words.toLocaleString()}</strong> words</span></div>
        {source ? <div className="export-ed-source">
          <span className="export-summary-kicker">Using edition</span>
          <div className="export-ed-source-head"><CoverChip src={source.thumbURL} title={source.title} className="export-ed-source-cover" /><span><strong>{source.name}</strong><em>{source.isbn13}</em></span></div>
          <ul className="export-ed-prefill">{source.prefill.map(line => <li key={line.k}><CheckIcon /><span><i>{line.k}</i> {line.v}</span></li>)}</ul>
          {source.note && <small className="export-ed-source-note">{source.note}</small>}
          <small className="export-ed-source-text">{textNote}</small>
          <small className="export-ed-source-foot">{source.footnote}</small>
        </div>
        : format ? <div className="export-summary-format"><span className="export-format-icon"><FormatIcon format={format} /></span><span><small>Selected edition</small><strong>{OUTPUT_LABELS[format]}</strong></span><button type="button" onClick={() => setStep('destination')}>Change</button></div>
          : <div className="export-summary-empty">Pick a registered edition, or start from scratch, to begin.</div>}
        <div className="export-summary-foot"><span>Original protected</span><small>Exporting creates a separate file and never replaces the open Draftline project.</small></div></aside>
      <main className="export-wizard-main"><div className="export-step-content">{step === 'destination' && renderDestination()}{step === 'contents' && renderContents()}{step === 'design' && renderDesign()}{step === 'review' && renderReview()}</div><footer className="export-wizard-footer"><span className="export-ed-footer-hint">{format ? (source ? 'Using registered metadata. Nothing is written to the manuscript.' : '') : 'Choose an edition or a format to continue.'}</span><span className="export-ed-footer-actions"><button type="button" className="dialog-btn" onClick={handleBack}>{step === 'destination' ? 'Cancel' : 'Back'}</button>{step !== 'review' ? <button type="button" className="dialog-btn primary" onClick={handleNext} disabled={(step === 'destination' && !format) || !!trimError}>Continue</button> : <button type="button" className="dialog-btn primary export-create-button" onClick={handleExport}>Choose location and export</button>}</span></footer></main>
    </div>
  </div></div>
}
