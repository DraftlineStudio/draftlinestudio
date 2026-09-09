import { useMemo, useState } from 'react'
import { useBookStore } from '../../store/bookStore'
import { useAppStore } from '../../store/appStore'
import { ExportEPUB, ExportDOCX, ExportPDF, ExportPrintPDF } from '../../../wailsjs/go/main/App'
import './ExportWizard.css'

type ExportFormat = 'epub' | 'docx' | 'pdf' | 'print-pdf'
type WizardStep = 'destination' | 'contents' | 'design' | 'review' | 'exporting'
type PrintPanel = 'page' | 'typography' | 'furniture' | 'title'

interface ExportOptions {
  includeCopyright: boolean
  includeFrontMatter: boolean
  includeBackMatter: boolean
}

interface EPUBOptions extends ExportOptions {
  fontFamily: 'reader' | 'merriweather' | 'lato'
  paragraphStyle: 'indented' | 'spaced'
  textAlign: 'reader' | 'left' | 'justify'
  chapterStyle: 'classic' | 'minimal'
  sceneBreakStyle: 'asterism' | 'rule' | 'space'
}

interface PDFOptions extends ExportOptions {
  pageSize: 'letter' | 'a4' | '6x9' | '5x8' | '5.5x8.5'
  fontFamily: 'merriweather' | 'lato'
  fontSize: 10 | 11 | 12 | 14
  lineHeight: 1.3 | 1.4 | 1.5 | 1.6
  paragraphIndent: string
  textAlign: 'justify' | 'left'
}

interface PrintPDFOptions extends PDFOptions {
  trimSize: '5x8' | '5.25x8' | '5.5x8.5' | '6x9' | 'custom'
  customWidth: string
  customHeight: string
  bleed: string
  gutterMargin: string
  outerMargin: string
  topMargin: string
  bottomMargin: string
  includeCropMarks: boolean
  chapterStartsRecto: boolean
  dropCap: boolean
  dropCapLines: 2 | 3 | 4
  runningHeaders: boolean
  headerStyle: 'smallcaps' | 'italic' | 'normal'
  pageNumberPosition: 'bottom-center' | 'bottom-outside' | 'top-outside'
  generateHalfTitle: boolean
  generateTOC: boolean
  mirroredMargins: boolean
  headingFont: 'body' | 'classic' | 'modern' | 'romance' | 'scifi' | 'fantasy'
  furnitureFont: 'body' | 'classic' | 'modern' | 'romance' | 'scifi' | 'fantasy'
  titlePageFont: 'body' | 'classic' | 'modern' | 'romance' | 'scifi' | 'fantasy'
  titlePageStyle: 'classic' | 'minimal' | 'dramatic'
  titlePageShowAuthor: boolean
  titlePageShowPublisher: boolean
}

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

function FormatIcon({ format }: { format: ExportFormat }) {
  if (format === 'epub') return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 5.5A2.5 2.5 0 016.5 3H11v16H6.5A2.5 2.5 0 004 21.5v-16zM20 5.5A2.5 2.5 0 0017.5 3H13v16h4.5a2.5 2.5 0 012.5 2.5v-16z" /></svg>
  if (format === 'docx') return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 2.75h8l4 4V21.25H6zM14 3v4h4M8.5 11l1.25 5 1.4-3.7 1.35 3.7 1.25-5" /></svg>
  if (format === 'pdf') return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 2.75h8l4 4V21.25H6zM14 3v4h4M8.5 16v-4h1.25a1.15 1.15 0 010 2.3H8.5M12.5 12h1.1a2 2 0 010 4h-1.1zM16 16v-4h2" /></svg>
  return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 8V3h12v5M6 18H4a2 2 0 01-2-2v-6a2 2 0 012-2h16a2 2 0 012 2v6a2 2 0 01-2 2h-2M6 14h12v7H6zM18.5 11h.01" /></svg>
}

function CheckIcon() {
  return <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4 10.5l3.5 3.5L16 5.5" /></svg>
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

export default function ExportWizard() {
  const { book } = useBookStore()
  const { closeExportWizard, setStatusMessage } = useAppStore()
  const [step, setStep] = useState<WizardStep>('destination')
  const [format, setFormat] = useState<ExportFormat | null>(null)
  const [printPanel, setPrintPanel] = useState<PrintPanel>('page')
  const [exporting, setExporting] = useState(false)
  const [exportError, setExportError] = useState('')
  const [exportSuccess, setExportSuccess] = useState(false)
  const [exportedPath, setExportedPath] = useState('')
  const [options, setOptions] = useState<ExportOptions>({ includeCopyright: true, includeFrontMatter: true, includeBackMatter: true })
  const [epubOptions, setEpubOptions] = useState<EPUBOptions>({ ...options, fontFamily: 'reader', paragraphStyle: 'indented', textAlign: 'reader', chapterStyle: 'classic', sceneBreakStyle: 'asterism' })
  const [pdfOptions, setPdfOptions] = useState<PDFOptions>({ ...options, pageSize: 'letter', fontFamily: 'merriweather', fontSize: 12, lineHeight: 1.5, paragraphIndent: '0.25', textAlign: 'left' })
  const [printOptions, setPrintOptions] = useState<PrintPDFOptions>({
    ...pdfOptions, pageSize: '5.5x8.5', trimSize: '5.5x8.5', customWidth: '5.5', customHeight: '8.5', bleed: '0',
    gutterMargin: '0.875', outerMargin: '0.625', topMargin: '0.75', bottomMargin: '0.625', includeCropMarks: false,
    fontFamily: 'merriweather', fontSize: 10, lineHeight: 1.4, paragraphIndent: '0.25', textAlign: 'left', chapterStartsRecto: true,
    dropCap: true, dropCapLines: 3, runningHeaders: true, headerStyle: 'smallcaps', pageNumberPosition: 'bottom-center',
    generateHalfTitle: true, generateTOC: true, mirroredMargins: true,
    headingFont: 'classic', furnitureFont: 'body', titlePageFont: 'classic', titlePageStyle: 'classic',
    titlePageShowAuthor: true, titlePageShowPublisher: true,
  })

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

  function updateSharedOption(key: keyof ExportOptions, value: boolean) {
    setOptions(current => ({ ...current, [key]: value }))
    setEpubOptions(current => ({ ...current, [key]: value }))
    setPdfOptions(current => ({ ...current, [key]: value }))
    setPrintOptions(current => ({ ...current, [key]: value }))
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
    else if (step === 'design') setStep('review')
  }

  function goToCompletedStep(target: Exclude<WizardStep, 'exporting'>) {
    const targetIndex = FLOW_STEPS.findIndex(item => item.id === target)
    if (targetIndex <= currentStepIndex || (step === 'review' && targetIndex < 3)) setStep(target)
  }

  async function handleExport() {
    if (!book || !format) return
    setStep('exporting')
    setExporting(true)
    setExportError('')
    setExportSuccess(false)
    try {
      let result: { success: boolean; file_path?: string; error?: string }
      if (format === 'epub') result = await ExportEPUB(book as any, epubOptions as any)
      else if (format === 'docx') result = await ExportDOCX(book as any, options as any)
      else if (format === 'pdf') result = await ExportPDF(book as any, pdfOptions as any)
      else result = await ExportPrintPDF(book as any, printOptions as any)
      if (result.success) {
        setExportSuccess(true)
        setExportedPath(result.file_path || '')
        setStatusMessage(`Exported to ${result.file_path}`)
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
      <div className="export-step-heading"><span className="export-eyebrow">Step 1 of 4</span><h2>Where is this book going?</h2><p>Choose the job this file needs to do. Draftline will shape the remaining choices around it.</p></div>
      <div className="export-destination-grid">
        {(Object.entries(FORMAT_INFO) as Array<[ExportFormat, typeof FORMAT_INFO[ExportFormat]]>).map(([id, info]) => (
          <button type="button" key={id} className={`export-destination-card${format === id ? ' selected' : ''}`} onClick={() => setFormat(id)} aria-pressed={format === id}>
            <span className="export-format-icon"><FormatIcon format={id} /></span>
            <span className="export-destination-copy"><span className="export-destination-topline"><strong>{info.intent}</strong></span><span>{info.description}</span><small>{info.label} {info.extension} · {info.detail}</small></span>
            <span className="export-card-check"><CheckIcon /></span>
          </button>
        ))}
      </div>
    </>
  }

  function renderContents() {
    const sectionCards = [
      { id: 'copyright', title: 'Copyright page', description: book?.copyright ? 'Rights, edition, and publication notice.' : 'No copyright content has been written.', count: stats.copyrightWords ? `${stats.copyrightWords.toLocaleString()} words` : 'Empty', checked: currentOptions.includeCopyright, locked: false, change: (value: boolean) => updateSharedOption('includeCopyright', value) },
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
      </div>
      <div className="export-setting-block"><h3>Page margins</h3><p>Interior breathing room and binding allowance.</p><div className="export-number-grid">
        {([['gutterMargin', 'Inside', printOptions.gutterMargin], ['outerMargin', 'Outside', printOptions.outerMargin], ['topMargin', 'Top', printOptions.topMargin], ['bottomMargin', 'Bottom', printOptions.bottomMargin]] as const).map(([key, label, value]) => <label key={key}>{label}<span><input value={value} onChange={event => setPrintOptions(current => ({ ...current, [key]: event.target.value }))} inputMode="decimal" /> in</span></label>)}
      </div></div>
      <div className="export-setting-block"><h3>Production page</h3><p>Binding and printer setup.</p><Toggle checked={printOptions.mirroredMargins} onChange={value => setPrintOptions(current => ({ ...current, mirroredMargins: value }))} label="Mirror inside margins" /><Toggle checked={printOptions.chapterStartsRecto} onChange={value => setPrintOptions(current => ({ ...current, chapterStartsRecto: value }))} label="Chapters begin on recto" note="Start on a right-hand page" /><div className="export-compact-field"><label>Bleed</label><span><input value={printOptions.bleed} onChange={event => setPrintOptions(current => ({ ...current, bleed: event.target.value }))} inputMode="decimal" /> in</span></div><Toggle checked={printOptions.includeCropMarks} onChange={value => setPrintOptions(current => ({ ...current, includeCropMarks: value }))} label="Include crop marks" /></div>
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
        <label>Type size<select value={printOptions.fontSize} onChange={event => setPrintOptions(current => ({ ...current, fontSize: Number(event.target.value) as PDFOptions['fontSize'] }))}><option value={10}>10 pt</option><option value={11}>11 pt</option><option value={12}>12 pt</option></select></label>
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
      {format === 'epub' && <div className="export-simple-design export-epub-design"><div className="export-reader-preview" aria-hidden="true"><div className={`${epubOptions.chapterStyle} ${epubOptions.paragraphStyle} ${epubOptions.fontFamily}`}><span>Chapter One</span><i /><i /><i /><b>{epubOptions.sceneBreakStyle === 'asterism' ? '⁂' : epubOptions.sceneBreakStyle === 'rule' ? '—' : ''}</b><i /><i /></div></div><div className="export-reading-controls"><h3>Responsive reader edition</h3><p>Choose the publisher defaults. Compatible readers can still override type, size, spacing, and theme for accessibility.</p><div className="export-select-grid"><label>Typeface<select value={epubOptions.fontFamily} onChange={event => setEpubOptions(current => ({ ...current, fontFamily: event.target.value as EPUBOptions['fontFamily'] }))}><option value="reader">Reader default (smallest file)</option><option value="merriweather">Embed Merriweather</option><option value="lato">Embed Lato</option></select></label><label>Paragraphs<select value={epubOptions.paragraphStyle} onChange={event => setEpubOptions(current => ({ ...current, paragraphStyle: event.target.value as EPUBOptions['paragraphStyle'] }))}><option value="indented">Book-style indents</option><option value="spaced">Space between paragraphs</option></select></label><label>Alignment<select value={epubOptions.textAlign} onChange={event => setEpubOptions(current => ({ ...current, textAlign: event.target.value as EPUBOptions['textAlign'] }))}><option value="reader">Reader default</option><option value="left">Left aligned</option><option value="justify">Justified</option></select></label><label>Chapter opening<select value={epubOptions.chapterStyle} onChange={event => setEpubOptions(current => ({ ...current, chapterStyle: event.target.value as EPUBOptions['chapterStyle'] }))}><option value="classic">Classic / lowered title</option><option value="minimal">Minimal / compact title</option></select></label><label>Scene breaks<select value={epubOptions.sceneBreakStyle} onChange={event => setEpubOptions(current => ({ ...current, sceneBreakStyle: event.target.value as EPUBOptions['sceneBreakStyle'] }))}><option value="asterism">Asterism</option><option value="rule">Short rule</option><option value="space">Open space</option></select></label></div><ul><li><CheckIcon />Linked contents and chapter navigation</li><li><CheckIcon />Storefront metadata from Book Details</li><li><CheckIcon />Valid reflowable XHTML, never raw editor markup</li></ul></div></div>}
      {format === 'docx' && <div className="export-simple-design"><div className="export-document-preview" aria-hidden="true"><div><i /><i /><i /><i /><i /><i /><i /></div></div><div><h3>Clean editable manuscript</h3><p>Draftline will favor familiar Word styles and editable structure over a locked visual design.</p><ul><li><CheckIcon />One heading per chapter</li><li><CheckIcon />Explicit page breaks</li><li><CheckIcon />Readable body-text defaults</li></ul></div></div>}
      {format === 'pdf' && <div className="export-simple-design export-pdf-design"><div className="export-document-preview pdf" aria-hidden="true"><div className={pdfOptions.fontFamily}><span>{book?.metadata.title || 'Untitled'}</span><i /><i /><i /><i /><i /></div></div><div className="export-reading-controls"><h3>Comfortable reading copy</h3><p>A fixed-layout edition for screens, reviewers, home printers, or archiving.</p><div className="export-select-grid"><label>Page size<select value={pdfOptions.pageSize} onChange={event => setPdfOptions(current => ({ ...current, pageSize: event.target.value as PDFOptions['pageSize'] }))}><option value="letter">US Letter</option><option value="a4">A4</option><option value="6x9">6 x 9 in</option><option value="5.5x8.5">5.5 x 8.5 in</option><option value="5x8">5 x 8 in</option></select></label><label>Typeface<select value={pdfOptions.fontFamily} onChange={event => setPdfOptions(current => ({ ...current, fontFamily: event.target.value as PDFOptions['fontFamily'] }))}><option value="merriweather">Merriweather</option><option value="lato">Lato</option></select></label><label>Type size<select value={pdfOptions.fontSize} onChange={event => setPdfOptions(current => ({ ...current, fontSize: Number(event.target.value) as PDFOptions['fontSize'] }))}><option value={11}>11 pt</option><option value={12}>12 pt</option><option value={14}>14 pt</option></select></label><label>Line spacing<select value={pdfOptions.lineHeight} onChange={event => setPdfOptions(current => ({ ...current, lineHeight: Number(event.target.value) as PDFOptions['lineHeight'] }))}><option value={1.3}>Tight / 1.3</option><option value={1.4}>Book / 1.4</option><option value={1.5}>Relaxed / 1.5</option><option value={1.6}>Open / 1.6</option></select></label><label>Alignment<select value={pdfOptions.textAlign} onChange={event => setPdfOptions(current => ({ ...current, textAlign: event.target.value as PDFOptions['textAlign'] }))}><option value="left">Left aligned</option><option value="justify">Justified</option></select></label><label>First-line indent<span className="export-field-with-unit"><input value={pdfOptions.paragraphIndent} onChange={event => setPdfOptions(current => ({ ...current, paragraphIndent: event.target.value }))} inputMode="decimal" /> in</span></label></div></div></div>}
      {format === 'print-pdf' && <><div className="export-design-tabs" role="tablist">{([['page', 'Page'], ['typography', 'Typography'], ['furniture', 'Furniture'], ['title', 'Title page']] as Array<[PrintPanel, string]>).map(([id, label]) => <button type="button" key={id} role="tab" aria-selected={printPanel === id} className={printPanel === id ? 'selected' : ''} onClick={() => setPrintPanel(id)}>{label}</button>)}</div>{printPanel === 'page' && renderPrintPagePanel()}{printPanel === 'typography' && renderPrintTypographyPanel()}{printPanel === 'furniture' && renderPrintFurniturePanel()}{printPanel === 'title' && renderPrintTitlePanel()}</>}
    </>
  }

  function renderReview() {
    if (!format) return null
    const included = [currentOptions.includeCopyright && book?.copyright ? 'Copyright page' : '', currentOptions.includeFrontMatter && stats.frontCount ? `${stats.frontCount} front matter ${stats.frontCount === 1 ? 'item' : 'items'}` : '', `${stats.bodyCount} manuscript ${stats.bodyCount === 1 ? 'chapter' : 'chapters'}`, currentOptions.includeBackMatter && stats.backCount ? `${stats.backCount} back matter ${stats.backCount === 1 ? 'item' : 'items'}` : ''].filter(Boolean)
    return <>
      <div className="export-step-heading"><span className="export-eyebrow">Step 4 of 4</span><h2>Review this edition</h2><p>One final check before choosing where to save the file.</p></div>
      <div className="export-review-layout">
        <div className="export-review-hero"><span className="export-format-icon"><FormatIcon format={format} /></span><div><small>{FORMAT_INFO[format].intent}</small><strong>{book?.metadata.title || 'Untitled'}</strong><span>{book?.metadata.author ? `by ${book.metadata.author}` : 'Author not set'}</span></div><em>{FORMAT_INFO[format].extension}</em></div>
        <div className="export-review-card"><div className="export-review-section"><span>Contents</span><strong>{selectedWords.toLocaleString()} words</strong><p>{included.join(' · ')}</p></div><button type="button" onClick={() => setStep('contents')}>Edit contents</button></div>
        <div className="export-review-card"><div className="export-review-section"><span>Design</span><strong>{format === 'print-pdf' ? `${TRIM_SIZES[printOptions.trimSize].label}${printOptions.trimSize === 'custom' ? ` (${printOptions.customWidth} x ${printOptions.customHeight})` : ''} / ${printOptions.fontFamily} ${printOptions.fontSize} pt` : format === 'pdf' ? `${pdfOptions.pageSize.toUpperCase()} / ${pdfOptions.fontFamily} ${pdfOptions.fontSize} pt` : format === 'epub' ? `${epubOptions.fontFamily === 'reader' ? 'Reader typography' : `Embedded ${epubOptions.fontFamily}`} / ${epubOptions.paragraphStyle}` : 'Editable manuscript layout'}</strong><p>{format === 'print-pdf' ? `${printOptions.textAlign === 'justify' ? 'Justified' : 'Left aligned'} / ${printOptions.runningHeaders ? 'Running headers' : 'No running headers'} / ${printOptions.generateTOC ? 'Contents page' : 'No contents page'}` : format === 'epub' ? `${epubOptions.chapterStyle} chapters / ${epubOptions.sceneBreakStyle} scene breaks / ${epubOptions.textAlign} alignment` : FORMAT_INFO[format].detail}</p></div><button type="button" onClick={() => setStep('design')}>Edit design</button></div>
      </div>
      <div className="export-save-note"><CheckIcon /><span>Draftline will open your system’s save dialog next. Your project file will not be changed.</span></div>
    </>
  }

  if (!book) return null
  if (step === 'exporting') return <div className="dialog-overlay" onKeyDown={event => event.key === 'Escape' && !exporting && closeExportWizard()}><div className="dialog export-wizard export-wizard-result">
    {exporting && <div className="export-result-state"><div className="export-progress-spinner" /><span className="export-eyebrow">Building edition</span><h2>Creating your {format ? FORMAT_INFO[format].label : 'export'}…</h2><p>Draftline is assembling the selected book sections.</p></div>}
    {exportError && <div className="export-result-state error"><span className="export-result-icon">!</span><span className="export-eyebrow">Export interrupted</span><h2>That file could not be created</h2><p>{exportError}</p><div className="dialog-actions"><button className="dialog-btn" onClick={() => setStep('review')}>Back to review</button><button className="dialog-btn primary" onClick={handleExport}>Try again</button></div></div>}
    {exportSuccess && <div className="export-result-state success"><span className="export-result-icon"><CheckIcon /></span><span className="export-eyebrow">Edition complete</span><h2>Your book is ready</h2><p>{exportedPath}</p><div className="dialog-actions"><button className="dialog-btn primary" onClick={closeExportWizard}>Done</button></div></div>}
  </div></div>

  return <div className="dialog-overlay" onKeyDown={event => event.key === 'Escape' && closeExportWizard()}><div className="dialog export-wizard">
    <header className="export-wizard-header"><div><span className="export-wizard-mark">D</span><strong>Create an edition</strong></div><button type="button" className="export-close" onClick={closeExportWizard} aria-label="Close export wizard">×</button></header>
    <nav className="export-stepper" aria-label="Export progress">{FLOW_STEPS.map((item, index) => { const complete = index < currentStepIndex; const active = item.id === step; return <button type="button" key={item.id} disabled={index > currentStepIndex} className={`${active ? 'active' : ''}${complete ? ' complete' : ''}`} onClick={() => goToCompletedStep(item.id)} aria-current={active ? 'step' : undefined}><span>{complete ? <CheckIcon /> : index + 1}</span><em>{item.label}</em></button> })}</nav>
    <div className="export-wizard-body">
      <aside className="export-edition-summary"><span className="export-summary-kicker">Current book</span><h3>{book.metadata.title || 'Untitled'}</h3><p>{book.metadata.author || 'Author not set'}</p><div className="export-summary-stats"><span><strong>{stats.bodyCount}</strong> chapters</span><span><strong>{stats.words.toLocaleString()}</strong> words</span></div>{format ? <div className="export-summary-format"><span className="export-format-icon"><FormatIcon format={format} /></span><span><small>Selected edition</small><strong>{FORMAT_INFO[format].label}</strong></span><button type="button" onClick={() => setStep('destination')}>Change</button></div> : <div className="export-summary-empty">Choose an export destination to begin.</div>}<div className="export-summary-foot"><span>Original protected</span><small>Exporting creates a separate file and never replaces the open Draftline project.</small></div></aside>
      <main className="export-wizard-main"><div className="export-step-content">{step === 'destination' && renderDestination()}{step === 'contents' && renderContents()}{step === 'design' && renderDesign()}{step === 'review' && renderReview()}</div><footer className="export-wizard-footer"><button type="button" className="dialog-btn" onClick={handleBack}>{step === 'destination' ? 'Cancel' : 'Back'}</button>{step !== 'review' ? <button type="button" className="dialog-btn primary" onClick={handleNext} disabled={step === 'destination' && !format}>Continue</button> : <button type="button" className="dialog-btn primary export-create-button" onClick={handleExport}>Choose location and export</button>}</footer></main>
    </div>
  </div></div>
}
