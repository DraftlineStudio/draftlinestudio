import { useState, useMemo } from 'react'
import { useBookStore } from '../../store/bookStore'
import { useAppStore } from '../../store/appStore'
import { ExportEPUB, ExportDOCX, ExportPDF, ExportPrintPDF } from '../../../wailsjs/go/main/App'

type ExportFormat = 'epub' | 'docx' | 'pdf' | 'print-pdf'
type WizardStep = 'format' | 'options' | 'preview' | 'exporting'

interface ExportOptions {
  includeCopyright: boolean
  includeFrontMatter: boolean
  includeBackMatter: boolean
}

interface PDFOptions extends ExportOptions {
  pageSize: 'letter' | 'a4' | '6x9' | '5x8' | '5.25x8' | '5.5x8.5'
  fontSize: 11 | 12 | 14
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
  // Typography
  fontFamily: 'garamond' | 'palatino' | 'times' | 'georgia'
  lineHeight: 1.3 | 1.4 | 1.5 | 1.6
  paragraphIndent: string
  textAlign: 'justify' | 'left'
  // Chapter styling
  chapterStartsRecto: boolean  // Chapters always start on odd (right) page
  dropCap: boolean
  dropCapLines: 2 | 3 | 4
  // Headers & footers
  runningHeaders: boolean
  headerStyle: 'smallcaps' | 'italic' | 'normal'
  pageNumberPosition: 'bottom-center' | 'bottom-outside' | 'top-outside'
  // Front matter
  generateHalfTitle: boolean
  generateTOC: boolean
  mirroredMargins: boolean  // Critical: swap gutter/outer for odd/even pages
}

const FORMAT_INFO: Record<ExportFormat, { label: string; desc: string }> = {
  'epub': { label: 'EPUB', desc: 'E-book format for readers' },
  'docx': { label: 'DOCX', desc: 'Microsoft Word document' },
  'pdf': { label: 'PDF', desc: 'Portable document for sharing' },
  'print-pdf': { label: 'Print-Ready PDF', desc: 'For professional printing' },
}

const TRIM_SIZES: Record<string, { label: string; w: string; h: string }> = {
  '5x8': { label: '5" x 8"', w: '5', h: '8' },
  '5.25x8': { label: '5.25" x 8"', w: '5.25', h: '8' },
  '5.5x8.5': { label: '5.5" x 8.5"', w: '5.5', h: '8.5' },
  '6x9': { label: '6" x 9"', w: '6', h: '9' },
  'custom': { label: 'Custom', w: '', h: '' },
}

export default function ExportWizard() {
  const { book } = useBookStore()
  const { closeExportWizard, setStatusMessage } = useAppStore()

  const [step, setStep] = useState<WizardStep>('format')
  const [format, setFormat] = useState<ExportFormat | null>(null)
  const [exporting, setExporting] = useState(false)
  const [exportError, setExportError] = useState('')
  const [exportSuccess, setExportSuccess] = useState(false)
  const [exportedPath, setExportedPath] = useState('')

  // Common options
  const [options, setOptions] = useState<ExportOptions>({
    includeCopyright: true,
    includeFrontMatter: true,
    includeBackMatter: true,
  })

  // PDF-specific options
  const [pdfOptions, setPdfOptions] = useState<PDFOptions>({
    ...options,
    pageSize: '6x9',
    fontSize: 12,
  })

  // Print PDF-specific options
  const [printOptions, setPrintOptions] = useState<PrintPDFOptions>({
    ...pdfOptions,
    trimSize: '5.5x8.5',
    customWidth: '5.5',
    customHeight: '8.5',
    bleed: '0.125',
    gutterMargin: '0.875',
    outerMargin: '0.625',
    topMargin: '0.75',
    bottomMargin: '0.625',
    includeCropMarks: true,
    // Typography defaults (based on Reedsy analysis)
    fontFamily: 'garamond',
    lineHeight: 1.4,
    paragraphIndent: '0.25',
    textAlign: 'justify',
    // Chapter styling
    chapterStartsRecto: true,
    dropCap: true,
    dropCapLines: 3,
    // Headers & footers
    runningHeaders: true,
    headerStyle: 'smallcaps',
    pageNumberPosition: 'bottom-center',
    // Front matter
    generateHalfTitle: true,
    generateTOC: true,
    mirroredMargins: true,
  })

  // Calculate stats
  const stats = useMemo(() => {
    if (!book) return { chapters: 0, words: 0, frontCount: 0, bodyCount: 0, backCount: 0 }

    const countWords = (html: string) => {
      const text = html.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
      return text ? text.split(' ').length : 0
    }

    let words = 0
    const allChapters = [...book.front_matter, ...book.body, ...book.back_matter]
    for (const ch of allChapters) {
      words += countWords(ch.content)
    }
    if (book.copyright) {
      words += countWords(book.copyright)
    }

    return {
      chapters: allChapters.length,
      words,
      frontCount: book.front_matter.length,
      bodyCount: book.body.length,
      backCount: book.back_matter.length,
    }
  }, [book])

  function handleFormatSelect(f: ExportFormat) {
    setFormat(f)
    setStep('options')
  }

  function handleBack() {
    if (step === 'options') setStep('format')
    else if (step === 'preview') setStep('options')
  }

  function handleNext() {
    if (step === 'options') setStep('preview')
  }

  async function handleExport() {
    if (!book || !format) return

    setStep('exporting')
    setExporting(true)
    setExportError('')

    try {
      let result: { success: boolean; file_path?: string; error?: string }

      const baseOptions = format === 'print-pdf' ? printOptions :
                          format === 'pdf' ? pdfOptions : options

      switch (format) {
        case 'epub':
          result = await ExportEPUB(book as any, baseOptions as any)
          break
        case 'docx':
          result = await ExportDOCX(book as any, baseOptions as any)
          break
        case 'pdf':
          result = await ExportPDF(book as any, pdfOptions as any)
          break
        case 'print-pdf':
          result = await ExportPrintPDF(book as any, printOptions as any)
          break
        default:
          throw new Error('Unknown format')
      }

      if (result.success) {
        setExportSuccess(true)
        setExportedPath(result.file_path || '')
        setStatusMessage(`Exported to ${result.file_path}`)
      } else if (result.error !== 'cancelled') {
        setExportError(result.error || 'Export failed')
      } else {
        // User cancelled - go back to preview
        setStep('preview')
      }
    } catch (e) {
      setExportError(String(e))
    }
    setExporting(false)
  }

  function handleClose() {
    closeExportWizard()
  }

  function handleKey(e: React.KeyboardEvent) {
    if (e.key === 'Escape') handleClose()
  }

  // Get current options for the selected format
  function getCurrentOptions() {
    if (format === 'print-pdf') return printOptions
    if (format === 'pdf') return pdfOptions
    return options
  }

  // Step 1: Format Selection
  if (step === 'format') {
    return (
      <div className="dialog-overlay" onKeyDown={handleKey}>
        <div className="dialog export-wizard">
          <div className="dialog-title">Export Book</div>
          <p className="dialog-subtitle">Choose an export format</p>

          <div className="wizard-choice-grid export-format-grid">
            <button className="wizard-choice-tile" onClick={() => handleFormatSelect('epub')}>
              <div className="wizard-choice-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <path d="M4 19.5A2.5 2.5 0 016.5 17H20" />
                  <path d="M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z" />
                  <path d="M8 7h8M8 11h8M8 15h5" />
                </svg>
              </div>
              <div className="wizard-choice-label">EPUB</div>
              <div className="wizard-choice-desc">E-book format for readers</div>
            </button>

            <button className="wizard-choice-tile" onClick={() => handleFormatSelect('docx')}>
              <div className="wizard-choice-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
                  <polyline points="14 2 14 8 20 8" />
                  <line x1="16" y1="13" x2="8" y2="13" />
                  <line x1="16" y1="17" x2="8" y2="17" />
                  <line x1="10" y1="9" x2="8" y2="9" />
                </svg>
              </div>
              <div className="wizard-choice-label">DOCX</div>
              <div className="wizard-choice-desc">Microsoft Word document</div>
            </button>

            <button className="wizard-choice-tile" onClick={() => handleFormatSelect('pdf')}>
              <div className="wizard-choice-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
                  <polyline points="14 2 14 8 20 8" />
                  <path d="M9 15v-2h2a1 1 0 110 2H9z" />
                  <path d="M9 15v2" />
                </svg>
              </div>
              <div className="wizard-choice-label">PDF</div>
              <div className="wizard-choice-desc">Portable document for sharing</div>
            </button>

            <button className="wizard-choice-tile" onClick={() => handleFormatSelect('print-pdf')}>
              <div className="wizard-choice-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <rect x="5" y="8" width="14" height="10" rx="1" />
                  <path d="M5 11V6a1 1 0 011-1h12a1 1 0 011 1v5" />
                  <path d="M8 18v3h8v-3" />
                  <circle cx="17" cy="11" r="1" fill="currentColor" />
                </svg>
              </div>
              <div className="wizard-choice-label">Print-Ready PDF</div>
              <div className="wizard-choice-desc">For professional printing</div>
            </button>
          </div>

          <div className="dialog-actions">
            <button className="dialog-btn" onClick={handleClose}>Cancel</button>
          </div>
        </div>
      </div>
    )
  }

  // Step 2: Options
  if (step === 'options' && format) {
    const isPrint = format === 'print-pdf'

    return (
      <div className="dialog-overlay" onKeyDown={handleKey}>
        <div className="dialog export-wizard">
          <div className="dialog-title">Export Options</div>
          <div className="export-format-badge">
            <span className="export-format-name">{FORMAT_INFO[format].label}</span>
          </div>

          <div className="export-options-section">
            <label className="dialog-label">Include Sections</label>

            <label className="export-checkbox">
              <input
                type="checkbox"
                checked={getCurrentOptions().includeCopyright}
                onChange={e => {
                  const v = e.target.checked
                  setOptions(o => ({ ...o, includeCopyright: v }))
                  setPdfOptions(o => ({ ...o, includeCopyright: v }))
                  setPrintOptions(o => ({ ...o, includeCopyright: v }))
                }}
              />
              <span>Copyright page</span>
              <span className="export-checkbox-note">{book?.copyright ? 'Has content' : 'Empty'}</span>
            </label>

            <label className="export-checkbox">
              <input
                type="checkbox"
                checked={getCurrentOptions().includeFrontMatter}
                onChange={e => {
                  const v = e.target.checked
                  setOptions(o => ({ ...o, includeFrontMatter: v }))
                  setPdfOptions(o => ({ ...o, includeFrontMatter: v }))
                  setPrintOptions(o => ({ ...o, includeFrontMatter: v }))
                }}
              />
              <span>Front matter</span>
              <span className="export-checkbox-note">{stats.frontCount} items</span>
            </label>

            <label className="export-checkbox">
              <input
                type="checkbox"
                checked={getCurrentOptions().includeBackMatter}
                onChange={e => {
                  const v = e.target.checked
                  setOptions(o => ({ ...o, includeBackMatter: v }))
                  setPdfOptions(o => ({ ...o, includeBackMatter: v }))
                  setPrintOptions(o => ({ ...o, includeBackMatter: v }))
                }}
              />
              <span>Back matter</span>
              <span className="export-checkbox-note">{stats.backCount} items</span>
            </label>
          </div>

          {format === 'pdf' && (
            <div className="export-options-section">
              <label className="dialog-label">Page Settings</label>
              <p className="export-options-note">Standard Letter size (8.5" x 11")</p>

              <div className="export-field-row">
                <label className="export-field-label">Font size</label>
                <select
                  className="dialog-select"
                  value={pdfOptions.fontSize}
                  onChange={e => {
                    const v = Number(e.target.value) as 11 | 12 | 14
                    setPdfOptions(o => ({ ...o, fontSize: v }))
                  }}
                >
                  <option value={11}>11pt</option>
                  <option value={12}>12pt</option>
                  <option value={14}>14pt</option>
                </select>
              </div>
            </div>
          )}

          {isPrint && (
            <>
              {/* Page Size & Margins */}
              <div className="export-options-section">
                <label className="dialog-label">Page Size</label>

                <div className="export-field-row">
                  <label className="export-field-label">Trim size</label>
                  <select
                    className="dialog-select"
                    value={printOptions.trimSize}
                    onChange={e => {
                      const v = e.target.value as PrintPDFOptions['trimSize']
                      const size = TRIM_SIZES[v]
                      setPrintOptions(o => ({
                        ...o,
                        trimSize: v,
                        customWidth: size?.w || o.customWidth,
                        customHeight: size?.h || o.customHeight,
                      }))
                    }}
                  >
                    {Object.entries(TRIM_SIZES).map(([k, v]) => (
                      <option key={k} value={k}>{v.label}</option>
                    ))}
                  </select>
                </div>

                {printOptions.trimSize === 'custom' && (
                  <div className="export-field-row">
                    <label className="export-field-label">Dimensions</label>
                    <div className="export-dimension-inputs">
                      <input
                        type="text"
                        className="dialog-input"
                        value={printOptions.customWidth}
                        onChange={e => setPrintOptions(o => ({ ...o, customWidth: e.target.value }))}
                        placeholder="Width"
                      />
                      <span className="export-dimension-x">x</span>
                      <input
                        type="text"
                        className="dialog-input"
                        value={printOptions.customHeight}
                        onChange={e => setPrintOptions(o => ({ ...o, customHeight: e.target.value }))}
                        placeholder="Height"
                      />
                      <span className="export-dimension-unit">inches</span>
                    </div>
                  </div>
                )}

                <div className="export-field-row">
                  <label className="export-field-label">Bleed</label>
                  <div className="export-input-with-unit">
                    <input
                      type="text"
                      className="dialog-input"
                      value={printOptions.bleed}
                      onChange={e => setPrintOptions(o => ({ ...o, bleed: e.target.value }))}
                    />
                    <span className="export-unit">in</span>
                  </div>
                </div>
              </div>

              {/* Margins */}
              <div className="export-options-section">
                <label className="dialog-label">Margins</label>

                <label className="export-checkbox">
                  <input
                    type="checkbox"
                    checked={printOptions.mirroredMargins}
                    onChange={e => setPrintOptions(o => ({ ...o, mirroredMargins: e.target.checked }))}
                  />
                  <span>Mirrored margins</span>
                  <span className="export-checkbox-note">Swap gutter for spine binding</span>
                </label>

                <div className="export-margins-grid">
                  <div className="export-field-row">
                    <label className="export-field-label">Gutter (inside)</label>
                    <div className="export-input-with-unit">
                      <input
                        type="text"
                        className="dialog-input"
                        value={printOptions.gutterMargin}
                        onChange={e => setPrintOptions(o => ({ ...o, gutterMargin: e.target.value }))}
                      />
                      <span className="export-unit">in</span>
                    </div>
                  </div>
                  <div className="export-field-row">
                    <label className="export-field-label">Outside</label>
                    <div className="export-input-with-unit">
                      <input
                        type="text"
                        className="dialog-input"
                        value={printOptions.outerMargin}
                        onChange={e => setPrintOptions(o => ({ ...o, outerMargin: e.target.value }))}
                      />
                      <span className="export-unit">in</span>
                    </div>
                  </div>
                  <div className="export-field-row">
                    <label className="export-field-label">Top</label>
                    <div className="export-input-with-unit">
                      <input
                        type="text"
                        className="dialog-input"
                        value={printOptions.topMargin}
                        onChange={e => setPrintOptions(o => ({ ...o, topMargin: e.target.value }))}
                      />
                      <span className="export-unit">in</span>
                    </div>
                  </div>
                  <div className="export-field-row">
                    <label className="export-field-label">Bottom</label>
                    <div className="export-input-with-unit">
                      <input
                        type="text"
                        className="dialog-input"
                        value={printOptions.bottomMargin}
                        onChange={e => setPrintOptions(o => ({ ...o, bottomMargin: e.target.value }))}
                      />
                      <span className="export-unit">in</span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Typography */}
              <div className="export-options-section">
                <label className="dialog-label">Typography</label>

                <div className="export-field-row">
                  <label className="export-field-label">Font</label>
                  <select
                    className="dialog-select"
                    value={printOptions.fontFamily}
                    onChange={e => setPrintOptions(o => ({ ...o, fontFamily: e.target.value as any }))}
                  >
                    <option value="garamond">Garamond</option>
                    <option value="palatino">Palatino</option>
                    <option value="times">Times New Roman</option>
                    <option value="georgia">Georgia</option>
                  </select>
                </div>

                <div className="export-field-row">
                  <label className="export-field-label">Size</label>
                  <select
                    className="dialog-select"
                    value={printOptions.fontSize}
                    onChange={e => setPrintOptions(o => ({ ...o, fontSize: Number(e.target.value) as any }))}
                  >
                    <option value={10}>10pt</option>
                    <option value={11}>11pt</option>
                    <option value={12}>12pt</option>
                  </select>
                </div>

                <div className="export-field-row">
                  <label className="export-field-label">Line height</label>
                  <select
                    className="dialog-select"
                    value={printOptions.lineHeight}
                    onChange={e => setPrintOptions(o => ({ ...o, lineHeight: Number(e.target.value) as any }))}
                  >
                    <option value={1.3}>1.3 (tight)</option>
                    <option value={1.4}>1.4 (normal)</option>
                    <option value={1.5}>1.5 (relaxed)</option>
                    <option value={1.6}>1.6 (spacious)</option>
                  </select>
                </div>

                <div className="export-field-row">
                  <label className="export-field-label">Alignment</label>
                  <select
                    className="dialog-select"
                    value={printOptions.textAlign}
                    onChange={e => setPrintOptions(o => ({ ...o, textAlign: e.target.value as any }))}
                  >
                    <option value="justify">Justified</option>
                    <option value="left">Left-aligned</option>
                  </select>
                </div>

                <div className="export-field-row">
                  <label className="export-field-label">Paragraph indent</label>
                  <div className="export-input-with-unit">
                    <input
                      type="text"
                      className="dialog-input"
                      value={printOptions.paragraphIndent}
                      onChange={e => setPrintOptions(o => ({ ...o, paragraphIndent: e.target.value }))}
                    />
                    <span className="export-unit">in</span>
                  </div>
                </div>
              </div>

              {/* Chapter Styling */}
              <div className="export-options-section">
                <label className="dialog-label">Chapter Styling</label>

                <label className="export-checkbox">
                  <input
                    type="checkbox"
                    checked={printOptions.chapterStartsRecto}
                    onChange={e => setPrintOptions(o => ({ ...o, chapterStartsRecto: e.target.checked }))}
                  />
                  <span>Chapters start on right page</span>
                </label>

                <label className="export-checkbox">
                  <input
                    type="checkbox"
                    checked={printOptions.dropCap}
                    onChange={e => setPrintOptions(o => ({ ...o, dropCap: e.target.checked }))}
                  />
                  <span>Drop cap</span>
                </label>

                {printOptions.dropCap && (
                  <div className="export-field-row">
                    <label className="export-field-label">Drop cap lines</label>
                    <select
                      className="dialog-select"
                      value={printOptions.dropCapLines}
                      onChange={e => setPrintOptions(o => ({ ...o, dropCapLines: Number(e.target.value) as any }))}
                    >
                      <option value={2}>2 lines</option>
                      <option value={3}>3 lines</option>
                      <option value={4}>4 lines</option>
                    </select>
                  </div>
                )}
              </div>

              {/* Headers & Page Numbers */}
              <div className="export-options-section">
                <label className="dialog-label">Headers & Page Numbers</label>

                <label className="export-checkbox">
                  <input
                    type="checkbox"
                    checked={printOptions.runningHeaders}
                    onChange={e => setPrintOptions(o => ({ ...o, runningHeaders: e.target.checked }))}
                  />
                  <span>Running headers</span>
                  <span className="export-checkbox-note">Title on left, chapter on right</span>
                </label>

                {printOptions.runningHeaders && (
                  <div className="export-field-row">
                    <label className="export-field-label">Header style</label>
                    <select
                      className="dialog-select"
                      value={printOptions.headerStyle}
                      onChange={e => setPrintOptions(o => ({ ...o, headerStyle: e.target.value as any }))}
                    >
                      <option value="smallcaps">Small Caps</option>
                      <option value="italic">Italic</option>
                      <option value="normal">Normal</option>
                    </select>
                  </div>
                )}

                <div className="export-field-row">
                  <label className="export-field-label">Page numbers</label>
                  <select
                    className="dialog-select"
                    value={printOptions.pageNumberPosition}
                    onChange={e => setPrintOptions(o => ({ ...o, pageNumberPosition: e.target.value as any }))}
                  >
                    <option value="bottom-center">Bottom center</option>
                    <option value="bottom-outside">Bottom outside</option>
                    <option value="top-outside">Top outside</option>
                  </select>
                </div>
              </div>

              {/* Front Matter Generation */}
              <div className="export-options-section">
                <label className="dialog-label">Auto-Generated Pages</label>

                <label className="export-checkbox">
                  <input
                    type="checkbox"
                    checked={printOptions.generateHalfTitle}
                    onChange={e => setPrintOptions(o => ({ ...o, generateHalfTitle: e.target.checked }))}
                  />
                  <span>Half-title page</span>
                </label>

                <label className="export-checkbox">
                  <input
                    type="checkbox"
                    checked={printOptions.generateTOC}
                    onChange={e => setPrintOptions(o => ({ ...o, generateTOC: e.target.checked }))}
                  />
                  <span>Table of contents</span>
                </label>

                <label className="export-checkbox">
                  <input
                    type="checkbox"
                    checked={printOptions.includeCropMarks}
                    onChange={e => setPrintOptions(o => ({ ...o, includeCropMarks: e.target.checked }))}
                  />
                  <span>Crop marks</span>
                </label>
              </div>
            </>
          )}

          <div className="dialog-actions">
            <button className="dialog-btn" onClick={handleBack}>Back</button>
            <button className="dialog-btn primary" onClick={handleNext}>Next</button>
          </div>
        </div>
      </div>
    )
  }

  // Step 3: Preview
  if (step === 'preview' && format) {
    const currentOpts = getCurrentOptions()
    const includedSections = []
    if (currentOpts.includeCopyright && book?.copyright) includedSections.push('Copyright')
    if (currentOpts.includeFrontMatter && stats.frontCount > 0) includedSections.push(`Front matter (${stats.frontCount})`)
    includedSections.push(`Body (${stats.bodyCount} chapters)`)
    if (currentOpts.includeBackMatter && stats.backCount > 0) includedSections.push(`Back matter (${stats.backCount})`)

    return (
      <div className="dialog-overlay" onKeyDown={handleKey}>
        <div className="dialog export-wizard">
          <div className="dialog-title">Export Preview</div>

          <div className="export-preview-card">
            <div className="export-preview-format">
              <span className="export-format-name">{FORMAT_INFO[format].label}</span>
            </div>

            <div className="export-preview-meta">
              <div className="export-preview-title">{book?.metadata.title || 'Untitled'}</div>
              {book?.metadata.author && (
                <div className="export-preview-author">by {book.metadata.author}</div>
              )}
            </div>

            <div className="export-preview-stats">
              <div className="export-stat">
                <span className="export-stat-value">{stats.chapters}</span>
                <span className="export-stat-label">Chapters</span>
              </div>
              <div className="export-stat">
                <span className="export-stat-value">{stats.words.toLocaleString()}</span>
                <span className="export-stat-label">Words</span>
              </div>
            </div>

            <div className="export-preview-sections">
              <label className="dialog-label">Included sections</label>
              <ul className="export-sections-list">
                {includedSections.map((s, i) => (
                  <li key={i}>{s}</li>
                ))}
              </ul>
            </div>

            {format === 'print-pdf' && (
              <div className="export-preview-print">
                <label className="dialog-label">Print settings</label>
                <div className="export-print-summary">
                  <span>Trim: {printOptions.trimSize === 'custom'
                    ? `${printOptions.customWidth}" x ${printOptions.customHeight}"`
                    : TRIM_SIZES[printOptions.trimSize].label}</span>
                  <span>Margins: {printOptions.gutterMargin}" / {printOptions.outerMargin}"</span>
                  {printOptions.mirroredMargins && <span>Mirrored</span>}
                  <span>Font: {printOptions.fontFamily} {printOptions.fontSize}pt</span>
                  {printOptions.dropCap && <span>Drop caps</span>}
                  {printOptions.runningHeaders && <span>Headers</span>}
                  {printOptions.generateTOC && <span>TOC</span>}
                </div>
              </div>
            )}
          </div>

          <div className="dialog-actions">
            <button className="dialog-btn" onClick={handleBack}>Back</button>
            <button className="dialog-btn primary" onClick={handleExport}>Export</button>
          </div>
        </div>
      </div>
    )
  }

  // Step 4: Exporting / Result
  if (step === 'exporting') {
    return (
      <div className="dialog-overlay" onKeyDown={handleKey}>
        <div className="dialog export-wizard">
          {exporting && (
            <>
              <div className="dialog-title">Exporting...</div>
              <div className="export-progress">
                <div className="export-progress-spinner" />
                <p>Creating {FORMAT_INFO[format!].label} file...</p>
              </div>
            </>
          )}

          {exportError && (
            <>
              <div className="dialog-title">Export Failed</div>
              <div className="export-error">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="15" y1="9" x2="9" y2="15" />
                  <line x1="9" y1="9" x2="15" y2="15" />
                </svg>
                <p>{exportError}</p>
              </div>
              <div className="dialog-actions">
                <button className="dialog-btn" onClick={() => setStep('preview')}>Back</button>
                <button className="dialog-btn primary" onClick={handleExport}>Retry</button>
              </div>
            </>
          )}

          {exportSuccess && (
            <>
              <div className="dialog-title">Export Complete</div>
              <div className="export-success">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--green)" strokeWidth="2">
                  <circle cx="12" cy="12" r="10" />
                  <path d="M8 12l2.5 2.5L16 9" />
                </svg>
                <p>Your book has been exported successfully.</p>
                {exportedPath && (
                  <p className="export-path">{exportedPath}</p>
                )}
              </div>
              <div className="dialog-actions">
                <button className="dialog-btn primary" onClick={handleClose}>Done</button>
              </div>
            </>
          )}
        </div>
      </div>
    )
  }

  return null
}
