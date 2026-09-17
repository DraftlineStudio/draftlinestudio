// Where an export comes from, and what it carries over.
//
// The export wizard used to ask one question on its first step — what kind of
// file? — and hold every answer after that in component state, which was
// thrown away when the dialog closed. An author who had spent ten minutes
// setting a trim, a gutter and a title page did it again the next time.
//
// Step one now offers the editions the book is actually registered in. Picking
// one prefills the rest of the wizard from the record, and anything changed
// afterwards can be written back to it. This file is that translation, in both
// directions, as pure functions:
//
//   record  → cards, sidebar summary, prefilled options   (what the author sees)
//   options → what changed, and the patch that saves it   (what the record gets)
//
// It is deliberately free of React and of the Wails bindings so that the
// arithmetic — which trim is 148 × 210 mm, what a gutter of "0.875 in" is as a
// number — can be tested as arithmetic.

import type { Edition, EditionFormat, EditionIndex, EditionKind, Metadata } from '../../types/draftline'
import { coverThumbURL } from './coverModel'
import { spineWidthLabel } from './editionModel'
import { normalizeISBN, validISBN } from './bookInfoModel'
import {
  CONTENTS_DEFAULTS, defaultWizardOptions, MAX_TRIM_INCHES, MIN_TRIM_INCHES, TRIM_PRESETS,
} from './publishingDefaults'

// ── The option shapes ──────────────────────────────────────────────────────
// These mirror internal/types/export.go field for field. They live here rather
// than in the wizard because they are now stored on the edition record, and a
// stored shape that drifts from the shape the exporter reads is a silent
// corruption rather than a type error.

export type ExportFormat = 'epub' | 'docx' | 'pdf' | 'print-pdf'

export interface ExportOptions {
  includeCopyright: boolean
  includeFrontMatter: boolean
  includeBackMatter: boolean
  // Stated the negative way round on purpose: every export has always made a
  // title page, so an absent value has to keep meaning "make one".
  omitTitlePage?: boolean
  // The registered format this export is made against. The backend reads the
  // edition record itself from these, which is how the ISBN, the cover and the
  // copyright page reach the file without any of them crossing the bridge.
  editionID?: string
  formatID?: string
}

export interface EPUBOptions extends ExportOptions {
  fontFamily: 'reader' | 'merriweather' | 'lato'
  paragraphStyle: 'indented' | 'spaced'
  textAlign: 'reader' | 'left' | 'justify'
  chapterStyle: 'classic' | 'minimal'
  sceneBreakStyle: 'asterism' | 'rule' | 'space'
  // A raised initial on the first paragraph of each chapter. Ordinary
  // ::first-letter CSS, which Kindle and other reading systems honour.
  dropCap: boolean
}

export interface PDFOptions extends ExportOptions {
  pageSize: 'letter' | 'a4' | '6x9' | '5x8' | '5.5x8.5'
  fontFamily: 'merriweather' | 'lato'
  fontSize: 9 | 10 | 11 | 12 | 14
  lineHeight: 1.3 | 1.4 | 1.5 | 1.6
  paragraphIndent: string
  textAlign: 'justify' | 'left'
}

// DOCXOptions is the editable manuscript: the file that goes to an editor and
// comes back marked up. Its settings are about that round trip.
export interface DOCXOptions extends ExportOptions {
  bodyStyle: 'normal' | 'manuscript'
  chapterBreak: 'page' | 'run'
  trackChanges: boolean
  hashSceneBreaks: boolean
}

// AudioOptions is the narration script: the file a voice actor reads from. It
// is not the reading copy renamed, which is what it used to be, and why none
// of its settings could be changed.
export interface AudioOptions extends ExportOptions {
  pageSize: 'letter' | 'a4'
  fontFamily: 'lato' | 'merriweather'
  fontSize: 12 | 14 | 16
  lineHeight: 1.5 | 1.8 | 2
  paragraphSpacing: 'half line between' | 'one line between' | 'two lines between'
  slatePage: boolean
  numberParagraphs: boolean
  pauseBreaks: boolean
  pronunciationColumn: boolean
  coverPage: boolean
  chapterWordCount: boolean
}

export type TrimSize = '5x8' | '5.25x8' | '5.5x8.5' | '6x9' | 'custom'

export interface PrintPDFOptions extends PDFOptions {
  trimSize: TrimSize
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
  // What the running head says on each side of the spread. With folios set
  // top-outside, 'author-title' reads across as
  // "137 | A. Marsh        Wide Water | 138".
  headerContent: 'author-title' | 'title-chapter' | 'chapter'
  // The same two choices an ebook offers, asked of a printed page.
  sceneBreakStyle: 'asterism' | 'rule' | 'space'
  chapterStyle: 'classic' | 'compact'
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

// WizardOptions is every answer the wizard holds, in one object, because that
// is what gets frozen onto the record.
export interface WizardOptions {
  shared: ExportOptions
  epub: EPUBOptions
  pdf: PDFOptions
  print: PrintPDFOptions
  audio: AudioOptions
  docx: DOCXOptions
}

// Every number a new export starts from is in publishingDefaults.ts, which is
// the one file to edit when the defaults turn out to be wrong. They are
// re-exported here because this is where the rest of the app already asks for
// them, and moving the callers would only spread the knowledge again.
export const defaultSharedOptions = (): ExportOptions => ({ ...CONTENTS_DEFAULTS })
export { defaultWizardOptions, MAX_TRIM_INCHES, MIN_TRIM_INCHES }

// ── Which file a format produces ───────────────────────────────────────────

// A registered format is a saleable object, and there is exactly one kind of
// file Draftline makes for each: an ebook is an EPUB, a printed book is a
// print-ready interior. An audiobook is neither, which is why the cards leave
// it out.
export function outputFormatFor(format: EditionFormat): ExportFormat | null {
  if (format.kind === 'print') return 'print-pdf'
  if (format.kind === 'ebook') return 'epub'
  // An audiobook record catalogues the ISBN, the artwork, the narrators and
  // the channels; Draftline writes no audio. What it can give that edition is
  // the file a narrator actually reads from, which is the reading copy.
  if (format.kind === 'audio') return 'pdf'
  return 'pdf'
}

// The other file a registered edition can produce: a reading copy.
//
// It is the same edition — the same ISBN on the copyright page, the same cover
// on a page of its own — as an ordinary PDF anyone can open, which is what
// goes to a reviewer, a blurb writer or a sensitivity reader. It is not the
// print interior: that file is for a printer and deliberately carries no
// cover, because a cover bound into the interior becomes page one of the
// printed block.
export function readingCopyFor(format: EditionFormat): ExportFormat | null {
  // An audiobook's main output is already the reading copy, so it gets no
  // second button offering the same file twice.
  if (format.kind === 'audio') return null
  return outputFormatFor(format) ? 'pdf' : null
}

export const OUTPUT_LABELS: Record<ExportFormat, string> = {
  epub: 'EPUB', docx: 'DOCX', pdf: 'PDF', 'print-pdf': 'Print PDF',
}

// The kind of object an ISBN attached to a from-scratch export describes. Only
// two of the four exports are saleable objects: a manuscript sent to an editor
// and a reading copy sent to a reviewer are neither, and neither gets a number.
export function registrableKind(format: ExportFormat): EditionKind | null {
  if (format === 'epub') return 'ebook'
  if (format === 'print-pdf') return 'print'
  return null
}

// ── Step one: the registered editions ──────────────────────────────────────

export interface EditionCard {
  editionID: string
  formatID: string
  /** The printed word for the format: Paperback, eBook. */
  format: string
  /** 'Second edition · 2030'. */
  edition: string
  isbn13: string
  /** The one-line specification under the ISBN. */
  spec: string
  badge: string
  badgeKind: 'ok' | 'accent'
  /** What comes out: 'EPUB' or 'Print PDF'. */
  out: string
  outputFormat: ExportFormat
  /** The reading copy this edition can also produce, or null. */
  altOutput: ExportFormat | null
  altLabel: string
  /** Same-origin thumbnail, or '' when this edition has no cover. */
  thumbURL: string
  /** The book title, for the drawn placeholder when there is no artwork. */
  title: string
}

// editionCards walks every format of every edition. An edition the author
// configured is an edition they can export: the ISBN is a field on the record,
// not a permission to use it. A format with no ISBN exports perfectly well and
// the package simply carries the generated identifier it always carried.
//
// The one format left out is audio, because Draftline writes no audio file and
// a card that cannot produce anything is not an option.
export function editionCards(index: EditionIndex | undefined, title: string): EditionCard[] {
  if (!index) return []
  const cards: EditionCard[] = []
  for (const edition of index.editions) {
    for (const format of edition.formats) {
      const isbn = (format.isbn13 ?? '').trim()
      const output = outputFormatFor(format)
      if (!output) continue
      const published = (format.status ?? '').trim().toLowerCase() === 'published'
      cards.push({
        editionID: edition.id,
        formatID: format.id,
        format: (format.format ?? '').trim() || 'Format',
        edition: [edition.label, edition.year].map(v => (v ?? '').trim()).filter(Boolean).join(' · '),
        isbn13: isbn,
        spec: cardSpec(format),
        badge: published ? 'Published' : 'Template ready',
        badgeKind: published ? 'ok' : 'accent',
        out: format.kind === 'audio' ? 'Narrator script (PDF)' : OUTPUT_LABELS[output],
        outputFormat: output,
        altOutput: readingCopyFor(format),
        altLabel: 'Reading copy (PDF)',
        thumbURL: coverThumbURL(edition),
        title,
      })
    }
  }
  return cards
}

function cardSpec(format: EditionFormat): string {
  if (format.kind === 'print') {
    const pages = (format.page_count ?? '').trim()
    return [
      (format.trim ?? '').trim(),
      pages ? `${pages} pp` : '',
      (format.binding ?? '').trim().toLowerCase(),
    ].filter(Boolean).join(' · ')
  }
  const version = (format.epub_version ?? '').trim() || 'EPUB 3.3'
  // The card says what the export will be, not what the record wishes it were.
  // Draftline writes a reflowable package; see fixedLayoutNote.
  return `Reflowable ${version} · linked contents`
}

// Draftline has one EPUB renderer and it makes a reflowable book. A record set
// to fixed layout is not wrong — an author may well have a fixed-layout file
// made elsewhere under that ISBN — but the export made here is not it, and the
// screen says so rather than printing the word "Fixed layout" over a file that
// reflows.
export function fixedLayoutNote(format: EditionFormat): string {
  if ((format.layout ?? '').trim().toLowerCase() !== 'fixed layout') return ''
  return 'This record says fixed layout. Draftline exports a reflowable package.'
}

// ── Step one: the sidebar ──────────────────────────────────────────────────

export interface PrefillLine { k: string; v: string }

export interface ExportSourceSummary {
  editionID: string
  formatID: string
  /** 'Paperback · Second edition'. */
  name: string
  isbn13: string
  thumbURL: string
  title: string
  prefill: PrefillLine[]
  footnote: string
  /** A plain caveat about this record, or '' when there is nothing to warn about. */
  note: string
}

export const SOURCE_FOOTNOTE =
  'Steps 2–4 are prefilled from this edition. Change anything and Draftline asks whether to save it back to the edition record.'

// exportSourceSummary is the design's `src`: the five lines that say what
// picking this edition actually did, so the author can see the prefill rather
// than having to go and look for it.
export function exportSourceSummary(index: EditionIndex | undefined, formatID: string, title: string): ExportSourceSummary | null {
  const found = findFormat(index, formatID)
  if (!found) return null
  const { edition, format } = found
  const isbn = (format.isbn13 ?? '').trim()
  const identifier = normalizeISBN(isbn) ? `urn:isbn:${normalizeISBN(isbn)}` : 'a generated identifier'
  const copyright = [edition.label, edition.year].map(v => (v ?? '').trim()).filter(Boolean).join(', ').toLowerCase()

  const prefill: PrefillLine[] = format.kind === 'print'
    ? [
      { k: 'Trim', v: (format.trim ?? '').trim() || 'not set' },
      { k: 'Margins', v: `mirrored, ${gutterInches(format)} in gutter` },
      { k: 'Cover wrap', v: spineLine(format) },
      { k: 'Copyright page', v: copyright || 'from this edition' },
      { k: 'Identifier', v: isbn || 'not yet assigned' },
    ]
    : [
      { k: 'Package', v: `${(format.epub_version ?? 'EPUB 3.3').trim()}, reflowable` },
      { k: 'Cover', v: coverLine(edition) },
      { k: 'Identifier', v: identifier },
      { k: 'Copyright page', v: copyright || 'from this edition' },
      { k: 'Contents', v: 'chapters + front matter' },
    ]

  const layoutNote = format.kind === 'ebook' ? fixedLayoutNote(format) : ''
  if (layoutNote) prefill.push({ k: 'Layout', v: 'record says fixed layout — this export reflows' })

  return {
    editionID: edition.id,
    formatID: format.id,
    name: [(format.format ?? '').trim() || 'Format', (edition.label ?? '').trim()].filter(Boolean).join(' · '),
    isbn13: isbn,
    thumbURL: coverThumbURL(edition),
    title,
    prefill,
    footnote: SOURCE_FOOTNOTE,
    note: layoutNote,
  }
}

function coverLine(edition: Edition): string {
  const cover = edition.cover
  if (!cover) return 'none attached'
  return `${(edition.label ?? 'edition').trim()} art, ${cover.width} × ${cover.height}`
}

// Draftline exports the interior of a printed book, never the wrap. The spine
// width is what a cover designer needs and is derived from the record, so the
// line gives it and then says plainly what the export will and will not carry.
function spineLine(format: EditionFormat): string {
  const spine = spineWidthLabel(format)
  if (!spine) return 'set a page count for the spine width'
  return `${spine} spine — interior only, no wrap`
}

// ── Prefill: the record becomes the wizard's answers ────────────────────────

// wizardOptionsForFormat layers three sources, least specific first: the
// defaults every export starts from, then what the edition record says, then
// whatever was saved back to this format last time. The last export of this
// edition therefore wins over the record, and the record wins over the
// defaults — which is the order an author would expect.
export function wizardOptionsForFormat(edition: Edition, format: EditionFormat): WizardOptions {
  const options = defaultWizardOptions()
  const identify = { editionID: edition.id, formatID: format.id }
  options.shared = { ...options.shared, ...identify }
  options.epub = { ...options.epub, ...identify }
  options.pdf = { ...options.pdf, ...identify }
  options.print = { ...options.print, ...identify, ...printPrefill(format) }
  return applyStoredSettings(options, format)
}

// printPrefill is what a printed format's specification means to the page
// layout: the trim it is bound at, the gutter it is bound with, and whether
// the artwork bleeds off the edge.
export function printPrefill(format: EditionFormat): Partial<PrintPDFOptions> {
  const patch: Partial<PrintPDFOptions> = {}
  const trim = trimFromRecord(format.trim ?? '')
  if (trim) {
    patch.trimSize = trim.trimSize
    patch.customWidth = trim.width
    patch.customHeight = trim.height
    if (trim.trimSize !== 'custom') patch.pageSize = trim.trimSize as PDFOptions['pageSize']
  }
  const gutter = inchesText(format.gutter ?? '')
  if (gutter) patch.gutterMargin = gutter
  const bleed = bleedFromRecord(format.bleed ?? '')
  if (bleed !== null) patch.bleed = bleed
  return patch
}

// applyStoredSettings reads back what a previous export of this format saved.
// It is defensive on purpose: export_settings is a free-form map in the
// archive, so a project file edited by hand — or written by a later Draftline
// — must not be able to put a number where the wizard expects a string.
export function applyStoredSettings(options: WizardOptions, format: EditionFormat): WizardOptions {
  const stored = format.export_settings
  if (!stored || typeof stored !== 'object') return options
  const merge = <T extends object>(base: T, key: ExportFormat | 'shared' | 'audio'): T => {
    const saved = (stored as Record<string, unknown>)[key]
    if (!saved || typeof saved !== 'object') return base
    const out = { ...base } as Record<string, unknown>
    for (const [field, value] of Object.entries(saved as Record<string, unknown>)) {
      if (!(field in out)) continue
      if (typeof value !== typeof out[field]) continue
      out[field] = value
    }
    return out as T
  }
  return {
    shared: merge(options.shared, 'shared'),
    epub: merge(options.epub, 'epub'),
    pdf: merge(options.pdf, 'pdf'),
    print: merge(options.print, 'print-pdf'),
    audio: merge(options.audio, 'audio'),
    docx: merge(options.docx, 'docx'),
  }
}

// ── Write-back: the wizard's answers become the record ──────────────────────

export interface PrefillChange {
  label: string
  from: string
  to: string
}

// prefillChanges is what the write-back prompt lists. It compares what the
// author now has against what the record would have prefilled, so the prompt
// names the fields that actually moved rather than offering to save
// everything every time.
export function prefillChanges(edition: Edition, format: EditionFormat, options: WizardOptions, output: ExportFormat): PrefillChange[] {
  const prefilled = wizardOptionsForFormat(edition, format)
  const changes: PrefillChange[] = []

  if (output === 'print-pdf') {
    const was = prefilled.print
    const now = options.print
    if (trimLabel(was) !== trimLabel(now)) changes.push({ label: 'Trim size', from: trimLabel(was), to: trimLabel(now) })
    if (was.gutterMargin !== now.gutterMargin) changes.push({ label: 'Inside gutter', from: `${was.gutterMargin} in`, to: `${now.gutterMargin} in` })
    if (was.bleed !== now.bleed) changes.push({ label: 'Bleed', from: bleedLabel(was.bleed), to: bleedLabel(now.bleed) })
  }

  const key = output === 'print-pdf' ? 'print' : output === 'epub' ? 'epub' : 'pdf'
  const nothingElse = shallowEqual(
    withoutIdentity(prefilled[key] as unknown as Record<string, unknown>),
    withoutIdentity(options[key] as unknown as Record<string, unknown>),
  )
  if (!nothingElse && !changes.length) {
    changes.push({ label: 'Design and contents', from: 'the defaults', to: 'your choices' })
  } else if (!nothingElse) {
    changes.push({ label: 'Design and contents', from: 'as prefilled', to: 'as you set them' })
  }
  return changes
}

// exportSettingsBlock is the wizard's own memory: every answer, so the next
// export of this format starts where this one finished. It is not a statement
// about the published object, which is why it is written without asking while
// the record's own fields are not.
export function exportSettingsBlock(options: WizardOptions): Record<string, unknown> {
  return {
    shared: { ...options.shared },
    epub: { ...options.epub },
    pdf: { ...options.pdf },
    'print-pdf': { ...options.print },
    audio: { ...options.audio },
    docx: { ...options.docx },
  }
}

// writeBackPatch is what saving back actually writes.
//
// Two things can go onto the record: the wizard's remembered answers, and the
// fields the record has words of its own for — a trim, a gutter, a bleed, all
// of which a cover designer and a printer read off the edition record rather
// than out of an export.
//
// The second kind is written ONLY for a field the author actually moved.
// `prefilled` is what this record put into the wizard when it was picked, and
// a value that still matches it is left exactly as the record spells it. That
// matters beyond tidiness: the record may hold '148 × 210 mm (A5)', which the
// wizard reads as 5.83 by 8.27 inches and cannot spell in millimetres again.
// Rewriting an untouched trim would quietly turn A5 into a rounded inch
// measurement the trim list does not even offer — a printer's specification
// changed by an export that changed nothing.
export function writeBackPatch(
  options: WizardOptions, output: ExportFormat, prefilled?: WizardOptions,
): Partial<EditionFormat> {
  const patch: Partial<EditionFormat> = { export_settings: exportSettingsBlock(options) }
  if (output !== 'print-pdf') return patch
  const was = prefilled?.print
  const now = options.print
  if (!was || trimRecordWords(was) !== trimRecordWords(now)) patch.trim = trimRecordWords(now)
  if (!was || was.gutterMargin !== now.gutterMargin) patch.gutter = `${now.gutterMargin} in`
  if (!was || was.bleed !== now.bleed) patch.bleed = bleedLabel(now.bleed)
  return patch
}

// patchChangesFormat says whether writing this patch would alter the record at
// all. A successful export marks the project dirty and schedules an autosave,
// and an export that settled on exactly the settings already stored has
// nothing to save.
export function patchChangesFormat(format: EditionFormat, patch: Partial<EditionFormat>): boolean {
  const current = format as unknown as Record<string, unknown>
  for (const [key, value] of Object.entries(patch)) {
    if (JSON.stringify(current[key] ?? null) !== JSON.stringify(value ?? null)) return true
  }
  return false
}

// ── Registering a from-scratch export ──────────────────────────────────────

// The design brief says a from-scratch export writes nothing back. The author
// asked for one exception: if the file that just came out has an ISBN, it is
// an edition, and saying so here saves typing the whole record in again on the
// Book & Editions screen. It is an offer, not a rule — an export with no
// number given is exactly the export the brief describes.
export function isbnRegistrationError(isbn: string, index?: EditionIndex): string {
  const trimmed = isbn.trim()
  if (!trimmed) return 'Type an ISBN, or skip this.'
  const digits = normalizeISBN(trimmed)
  if (digits.length !== 10 && digits.length !== 13) return 'An ISBN is 10 or 13 digits.'
  if (!validISBN(trimmed)) return 'That ISBN’s check digit does not match the digits before it.'
  const holder = formatHolding(index, digits)
  if (holder) {
    // An ISBN identifies one object. Two records carrying the same number is a
    // publishing record that contradicts itself, and it is the kind of mistake
    // nothing later in the chain catches.
    return `${digits} is already on ${holder}. An ISBN belongs to one format only.`
  }
  return ''
}

// formatHolding names the record an ISBN is already on, for the message above.
export function formatHolding(index: EditionIndex | undefined, digits: string): string {
  if (!index || !digits) return ''
  for (const edition of index.editions) {
    for (const format of edition.formats) {
      if (normalizeISBN(format.isbn13 ?? '') !== digits) continue
      const word = (format.format ?? '').trim() || 'a format'
      const label = (edition.label ?? '').trim()
      return label ? `the ${word.toLowerCase()} of the ${label.toLowerCase()}` : `the ${word.toLowerCase()}`
    }
  }
  return ''
}

// registrationPatch is the format record a registered from-scratch export
// becomes: the number, and the specification the export was actually made
// with, so the edition describes the file that exists rather than a blank
// template.
//
// It deliberately does not claim more than the wizard knows. Where the number
// came from — an agency block, a retailer's free number — is something only
// the author can say, so `registration` keeps the new record's own 'Not yet
// assigned' until they set it. And the status is 'Registered', not
// 'Published': a file has been made and a number attached to it, which is not
// the same as a book being on sale.
export function registrationPatch(
  isbn: string, output: ExportFormat, options: WizardOptions, meta: Partial<Metadata>,
): Partial<EditionFormat> {
  const patch: Partial<EditionFormat> = {
    isbn13: isbn.trim(),
    status: 'Registered',
    publication_date: today(),
    imprint_of_record: (meta.imprint ?? '').trim() || (meta.publisher ?? '').trim() || undefined,
    ...writeBackPatch(options, output),
  }
  if (output === 'print-pdf') patch.format = 'Paperback'
  if (output === 'epub') patch.format = 'eBook'
  return patch
}

function today(): string {
  return new Date().toISOString().slice(0, 10)
}

// ── Trim arithmetic ────────────────────────────────────────────────────────


const MM_PER_INCH = 25.4

// trimFromRecord reads the Editions screen's own wording — '6 × 9 in (trade)',
// '148 × 210 mm (A5)' — and returns the wizard's answer. Millimetres are
// converted; anything it cannot read at all returns null, and the wizard keeps
// its default rather than inventing a page size.
export function trimFromRecord(trim: string): { trimSize: TrimSize; width: string; height: string } | null {
  const text = trim.trim()
  if (!text) return null
  const numbers = text.match(/\d+(?:\.\d+)?/g)
  if (!numbers || numbers.length < 2) return null
  let width = Number.parseFloat(numbers[0])
  let height = Number.parseFloat(numbers[1])
  if (!Number.isFinite(width) || !Number.isFinite(height)) return null
  if (/mm/i.test(text)) {
    width = round2(width / MM_PER_INCH)
    height = round2(height / MM_PER_INCH)
  }
  if (width < MIN_TRIM_INCHES || width > MAX_TRIM_INCHES) return null
  if (height < MIN_TRIM_INCHES || height > MAX_TRIM_INCHES) return null
  const named = TRIM_PRESETS.find(t => t.width === width && t.height === height)
  return {
    trimSize: named ? named.id : 'custom',
    width: String(width),
    height: String(height),
  }
}

// trimRecordWords is the reverse: the words the Editions screen would use for
// the trim the author has set, so a write-back does not fill the record with a
// vocabulary of its own.
export function trimRecordWords(options: PrintPDFOptions): string {
  const preset = TRIM_PRESETS.find(one => one.id === options.trimSize)
  if (preset) return preset.recordWords
  return `${options.customWidth} × ${options.customHeight} in`
}

export function trimLabel(options: PrintPDFOptions): string {
  return trimRecordWords(options)
}

// customTrimError is the wizard's half of the bound the exporter enforces. It
// says the same thing before the author reaches the end of the wizard, because
// finding out at the save dialog that a page cannot be printed is a worse way
// to be told.
export function customTrimError(options: PrintPDFOptions): string {
  if (options.trimSize !== 'custom') return ''
  return trimSideError('width', options.customWidth) || trimSideError('height', options.customHeight)
}

function trimSideError(side: string, value: string): string {
  const text = value.trim()
  if (!text) return `Give a trim ${side} between ${MIN_TRIM_INCHES} and ${MAX_TRIM_INCHES} inches.`
  const parsed = Number(text)
  if (!Number.isFinite(parsed)) return `“${text}” is not a trim ${side}. Give a measurement in inches.`
  if (parsed < MIN_TRIM_INCHES || parsed > MAX_TRIM_INCHES) {
    return `A trim ${side} of ${parsed} inches cannot be printed. Give one between ${MIN_TRIM_INCHES} and ${MAX_TRIM_INCHES} inches.`
  }
  return ''
}

// ── Small readings of the record's words ───────────────────────────────────

// inchesText pulls the number out of a measurement an author typed: '0.875',
// '0.875 in', '0.9in'. It returns '' when there is no number, so the caller
// keeps its default rather than laying out a page with a gutter of NaN.
export function inchesText(value: string): string {
  const match = value.trim().match(/-?\d+(?:\.\d+)?/)
  if (!match) return ''
  const parsed = Number.parseFloat(match[0])
  if (!Number.isFinite(parsed) || parsed < 0) return ''
  return String(parsed)
}

function gutterInches(format: EditionFormat): string {
  return inchesText(format.gutter ?? '') || '0.875'
}

// The record holds a bleed as a phrase, because that is how it reads in a
// dropdown. The page layout needs it as inches.
export function bleedFromRecord(value: string): string | null {
  const text = value.trim()
  if (!text) return null
  if (/^no\b/i.test(text)) return '0'
  const inches = inchesText(text)
  return inches || null
}

export function bleedLabel(value: string): string {
  const inches = Number(value)
  if (!Number.isFinite(inches) || inches <= 0) return 'No bleed'
  return `Bleed ${inches} in`
}

function round2(value: number): number {
  return Math.round(value * 100) / 100
}

function withoutIdentity(value: Record<string, unknown>): Record<string, unknown> {
  const { editionID: _edition, formatID: _format, ...rest } = value
  return rest
}

function shallowEqual(a: Record<string, unknown>, b: Record<string, unknown>): boolean {
  const keys = new Set([...Object.keys(a), ...Object.keys(b)])
  for (const key of keys) {
    if (a[key] !== b[key]) return false
  }
  return true
}

export function findFormat(index: EditionIndex | undefined, formatID: string): { edition: Edition; format: EditionFormat } | null {
  if (!index || !formatID) return null
  for (const edition of index.editions) {
    for (const format of edition.formats) {
      if (format.id === formatID) return { edition, format }
    }
  }
  return null
}
