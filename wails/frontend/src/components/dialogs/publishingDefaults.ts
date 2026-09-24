// ═══════════════════════════════════════════════════════════════════════════
// INDUSTRY DEFAULTS
//
// Every number a new export starts from lives here and nowhere else. This is
// the file to edit when the defaults turn out to be wrong — and they will,
// because they are currently an educated guess rather than a measurement.
//
// The plan for making them a measurement: export the same short story from
// Reedsy and from Atticus in every format, every trade trim, hardcover and
// softcover, then read the page geometry and typography back out of those PDFs
// and set these to match what the two tools most authors actually use are
// doing. Nothing else in the codebase should need to change when that happens.
//
// What is NOT here: the arithmetic that reads a trim out of an edition record
// (exportSource.ts), and the page construction the Go renderer does from the
// options these produce (internal/export/pdf_spec.go). Those are behaviour.
// This is the starting position.
// ═══════════════════════════════════════════════════════════════════════════

import type {
  AudioOptions, DOCXOptions, EPUBOptions, ExportOptions, PDFOptions, PrintPDFOptions,
  TrimSize, WizardOptions,
} from './exportSource'

// ── Trim sizes ─────────────────────────────────────────────────────────────

export interface TrimPreset {
  id: TrimSize
  /** What the settings screen calls it. */
  label: string
  /** What gets written onto an edition record, which reads as a specification. */
  recordWords: string
  width: number
  height: number
}

// The four trade sizes every print-on-demand service offers, plus the escape
// hatch. Order is the order the dropdown shows.
export const TRIM_PRESETS: TrimPreset[] = [
  { id: '5x8', label: '5 × 8 in', recordWords: '5 × 8 in', width: 5, height: 8 },
  { id: '5.25x8', label: '5.25 × 8 in', recordWords: '5.25 × 8 in', width: 5.25, height: 8 },
  { id: '5.5x8.5', label: '5.5 × 8.5 in', recordWords: '5.5 × 8.5 in', width: 5.5, height: 8.5 },
  { id: '6x9', label: '6 × 9 in', recordWords: '6 × 9 in (trade)', width: 6, height: 9 },
]

export const CUSTOM_TRIM_LABEL = 'Custom…'

// The bounds a page has to fall inside to be printable at all. They mirror
// internal/export/pdf_spec.go: a page smaller than the smallest mass-market
// paperback or larger than anything a print-on-demand service will bind is a
// typing mistake rather than a page, and is refused on both sides.
export const MIN_TRIM_INCHES = 3
export const MAX_TRIM_INCHES = 12

// ── Print interior ─────────────────────────────────────────────────────────

// Margins in inches. The gutter is wider than the outer margin because the
// binding takes some of it back; how much depends on the page count, which is
// why a long book needs more than this.
export const PRINT_MARGINS = {
  gutter: '0.875',
  outer: '0.625',
  top: '0.75',
  bottom: '0.625',
}

export const PRINT_TYPE = {
  fontFamily: 'merriweather' as PrintPDFOptions['fontFamily'],
  fontSize: 9 as PrintPDFOptions['fontSize'],
  lineHeight: 1.4 as PrintPDFOptions['lineHeight'],
  // Justified, which is how a printed book is set and what the industry
  // standard on the Book & Editions screen promises. A reading copy is left
  // aligned; see READING_PDF.
  textAlign: 'justify' as PrintPDFOptions['textAlign'],
  paragraphIndent: '0.25',
  headingFont: 'classic' as PrintPDFOptions['headingFont'],
  furnitureFont: 'body' as PrintPDFOptions['furnitureFont'],
}

export const PRINT_FURNITURE = {
  // The author on the verso and the book on the recto, with the folio on the
  // outside of the same line: "137 | A. Marsh        Wide Water | 138". It
  // is what a trade paperback does and it is what a reader who opens the book
  // in the middle needs.
  headerContent: 'author-title' as PrintPDFOptions['headerContent'],
  sceneBreakStyle: 'asterism' as PrintPDFOptions['sceneBreakStyle'],
  chapterStyle: 'classic' as PrintPDFOptions['chapterStyle'],
  chapterStartsRecto: true,
  dropCap: true,
  dropCapLines: 3 as PrintPDFOptions['dropCapLines'],
  runningHeaders: true,
  headerStyle: 'smallcaps' as PrintPDFOptions['headerStyle'],
  pageNumberPosition: 'top-outside' as PrintPDFOptions['pageNumberPosition'],
  generateHalfTitle: true,
  generateTOC: true,
  mirroredMargins: true,
  includeCropMarks: false,
  skipKDPChecks: false,
  bleed: '0',
  titlePageFont: 'classic' as PrintPDFOptions['titlePageFont'],
  titlePageStyle: 'classic' as PrintPDFOptions['titlePageStyle'],
  titlePageShowAuthor: true,
  titlePageShowPublisher: true,
}

/** The trim a print export starts at when the record does not say. */
export const DEFAULT_TRIM: TrimSize = '5.5x8.5'

// ── Reading copy ───────────────────────────────────────────────────────────

// A file for a screen and a home printer, not for a printer's press: bigger
// type, looser leading, a page size the reader already has paper for.
export const READING_PDF = {
  pageSize: 'letter' as PDFOptions['pageSize'],
  fontFamily: 'merriweather' as PDFOptions['fontFamily'],
  fontSize: 12 as PDFOptions['fontSize'],
  lineHeight: 1.5 as PDFOptions['lineHeight'],
  textAlign: 'left' as PDFOptions['textAlign'],
  paragraphIndent: '0.25',
}

// ── Ebook ──────────────────────────────────────────────────────────────────

// Reader default is deliberate: it makes the smallest file and leaves
// typography to the reading app, which is what an accessible EPUB does.
export const EPUB_DEFAULTS = {
  fontFamily: 'reader' as EPUBOptions['fontFamily'],
  paragraphStyle: 'indented' as EPUBOptions['paragraphStyle'],
  textAlign: 'reader' as EPUBOptions['textAlign'],
  chapterStyle: 'classic' as EPUBOptions['chapterStyle'],
  sceneBreakStyle: 'asterism' as EPUBOptions['sceneBreakStyle'],
  dropCap: false,
  version: 'EPUB 3.3',
}

// ── Narration script ───────────────────────────────────────────────────────

// What a narrator reads from: large type, open leading, no justification, and
// a slate before each chapter. None of this reaches an exporter yet.
// The narration script. Large type and open leading because it is read aloud
// from a music stand; ragged right because justification moves the words
// between takes; numbered paragraphs because a retake is asked for by number.
export const AUDIO_SCRIPT = {
  pageSize: 'letter' as AudioOptions['pageSize'],
  fontFamily: 'lato' as AudioOptions['fontFamily'],
  fontSize: 14 as AudioOptions['fontSize'],
  lineHeight: 1.8 as AudioOptions['lineHeight'],
  paragraphSpacing: 'one line between' as AudioOptions['paragraphSpacing'],
  slatePage: true,
  numberParagraphs: true,
  pauseBreaks: true,
  pronunciationColumn: false,
  coverPage: true,
  chapterWordCount: true,
}

// ── Editable manuscript ────────────────────────────────────────────────────

// What an editor gets. A readable proportional page by default; the Courier
// double-spaced manuscript format is still what some submissions desks ask
// for. Revision recording is on, because a document sent to an editor is sent
// to be marked up.
export const DOCX_DEFAULTS = {
  bodyStyle: 'normal' as DOCXOptions['bodyStyle'],
  chapterBreak: 'page' as DOCXOptions['chapterBreak'],
  trackChanges: true,
  hashSceneBreaks: false,
}

// ── What goes in the file ──────────────────────────────────────────────────

export const CONTENTS_DEFAULTS: Omit<ExportOptions, 'editionID' | 'formatID'> = {
  includeCopyright: true,
  includeFrontMatter: true,
  includeBackMatter: true,
}

// ── The choice lists the settings screen offers ────────────────────────────

export const PRINT_TYPE_SIZES = [9, 10, 11, 12]
export const READING_TYPE_SIZES = [11, 12, 14]
export const AUDIO_TYPE_SIZES = [12, 14, 16]

export const LINE_HEIGHT_CHOICES: Array<{ label: string; value: number }> = [
  { label: 'Tight · 1.3', value: 1.3 },
  { label: 'Book · 1.4', value: 1.4 },
  { label: 'Relaxed · 1.5', value: 1.5 },
  { label: 'Open · 1.6', value: 1.6 },
]

// The display faces Draftline ships for chapter openings and furniture. The
// second word is what the face is for, not a genre claim about the book.
export const DISPLAY_FACE_CHOICES: Array<{ label: string; value: string }> = [
  { label: 'Body', value: 'body' },
  { label: 'Classic', value: 'classic' },
  { label: 'Modern', value: 'modern' },
  { label: 'Romance', value: 'romance' },
  { label: 'Sci-fi', value: 'scifi' },
  { label: 'Fantasy', value: 'fantasy' },
]

export const PAGE_SIZE_CHOICES: Array<{ label: string; value: string }> = [
  { label: 'US Letter', value: 'letter' },
  { label: 'A4', value: 'a4' },
  { label: '6 × 9 in', value: '6x9' },
  { label: '5.5 × 8.5 in', value: '5.5x8.5' },
  { label: '5 × 8 in', value: '5x8' },
]

export const EPUB_VERSION_CHOICES = ['EPUB 3.3', 'EPUB 3.0', 'EPUB 2.0.1']

// ── The whole starting position, in one object ─────────────────────────────

// This is what every new export begins from. An edition record and whatever a
// previous export of that format saved are layered over it in exportSource.ts;
// nothing else invents a number of its own.
export function defaultWizardOptions(): WizardOptions {
  const shared: ExportOptions = { ...CONTENTS_DEFAULTS }
  const trim = TRIM_PRESETS.find(preset => preset.id === DEFAULT_TRIM) ?? TRIM_PRESETS[2]
  const pdf: PDFOptions = { ...shared, ...READING_PDF }
  return {
    shared,
    epub: {
      ...shared,
      fontFamily: EPUB_DEFAULTS.fontFamily,
      paragraphStyle: EPUB_DEFAULTS.paragraphStyle,
      textAlign: EPUB_DEFAULTS.textAlign,
      chapterStyle: EPUB_DEFAULTS.chapterStyle,
      sceneBreakStyle: EPUB_DEFAULTS.sceneBreakStyle,
      dropCap: EPUB_DEFAULTS.dropCap,
    },
    pdf,
    audio: { ...shared, ...AUDIO_SCRIPT },
    docx: { ...shared, ...DOCX_DEFAULTS },
    print: {
      ...pdf,
      pageSize: trim.id as PDFOptions['pageSize'],
      trimSize: trim.id,
      customWidth: String(trim.width),
      customHeight: String(trim.height),
      gutterMargin: PRINT_MARGINS.gutter,
      outerMargin: PRINT_MARGINS.outer,
      topMargin: PRINT_MARGINS.top,
      bottomMargin: PRINT_MARGINS.bottom,
      fontFamily: PRINT_TYPE.fontFamily,
      fontSize: PRINT_TYPE.fontSize,
      lineHeight: PRINT_TYPE.lineHeight,
      textAlign: PRINT_TYPE.textAlign,
      paragraphIndent: PRINT_TYPE.paragraphIndent,
      headingFont: PRINT_TYPE.headingFont,
      furnitureFont: PRINT_TYPE.furnitureFont,
      ...PRINT_FURNITURE,
    },
  }
}
