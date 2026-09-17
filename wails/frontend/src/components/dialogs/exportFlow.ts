// The export flow, as the flow itself rather than as four fixed steps.
//
// An export now starts from what the author is exporting — an edition, a
// reading copy, or a set of formats picked by hand — and the rest of the
// wizard follows from that answer. An edition export asks about locking the
// text at the end; a reading copy never does. Which steps exist, which formats
// are on offer, which settings each format shows, and what the review says are
// all consequences of that first choice, so they are computed here rather than
// branched over inside the screen.
//
// No React and no bindings. Everything below is a function of the publishing
// record and the wizard's own answers.

import type { Edition, EditionFormat, EditionIndex } from '../../types/draftline'
import { spineWidthLabel } from './editionModel'
import { coverThumbURL } from './coverModel'
import { wrapPreviewURL } from './wrapModel'
import type { ExportFormat, WizardOptions } from './exportSource'
import {
  AUDIO_SCRIPT, AUDIO_TYPE_SIZES, CUSTOM_TRIM_LABEL, DISPLAY_FACE_CHOICES,
  EPUB_VERSION_CHOICES, LINE_HEIGHT_CHOICES, PAGE_SIZE_CHOICES, PRINT_TYPE_SIZES,
  READING_TYPE_SIZES, TRIM_PRESETS,
} from './publishingDefaults'

// ── What can come out ──────────────────────────────────────────────────────

// The six objects the flow can produce. Four of them are files Draftline
// already writes; a hardcover interior is a print interior at a different
// spine width, and an audiobook script is a reading copy laid out for someone
// reading it aloud. Draftline writes no audio and never claims to.
export type FlowOutput = 'epub' | 'docx' | 'pdf' | 'print-pdf' | 'hc' | 'audio'

export interface OutputInfo {
  label: string
  desc: string
  meta: string
}

export const OUTPUT_INFO: Record<FlowOutput, OutputInfo> = {
  epub: { label: 'EPUB', desc: 'Reflowable ebook for storefronts and e-readers.', meta: '.epub' },
  docx: { label: 'DOCX', desc: 'Editable manuscript for editors, agents, or Word.', meta: '.docx' },
  pdf: { label: 'Reading copy PDF', desc: 'Fixed layout for reviewers, screens, or home printing.', meta: '.pdf' },
  'print-pdf': { label: 'Print-ready PDF', desc: 'Interior for the printer: trim, mirrored margins, folios.', meta: '.pdf' },
  hc: { label: 'Hardcover interior', desc: 'Same interior, case-laminate spine width.', meta: '.pdf' },
  audio: { label: 'Audiobook script', desc: 'Narration script: wide paragraph spacing, chapter slates, ebook cover on the opening page.', meta: '.pdf' },
}

// What a from-scratch export may choose from, in the order the screen offers
// them: the two everyday files first.
export const SCRATCH_OUTPUTS: FlowOutput[] = ['docx', 'pdf', 'print-pdf', 'epub', 'audio']

// Which exporter actually writes the file. A hardcover interior and a
// paperback interior are the same exporter at different measurements, and a
// narration script is the reading-copy exporter.
export function exportFormatFor(output: FlowOutput): ExportFormat {
  if (output === 'hc') return 'print-pdf'
  if (output === 'audio') return 'pdf'
  return output as ExportFormat
}

// Which set of the wizard's answers a format reads. Contents toggles are per
// format because the author sets them per format tab.
export type OptionGroup = 'shared' | 'epub' | 'pdf' | 'print' | 'audio' | 'docx'

export function optionGroupFor(output: FlowOutput): OptionGroup {
  if (output === 'epub') return 'epub'
  if (output === 'docx') return 'docx'
  if (output === 'audio') return 'audio'
  if (output === 'pdf') return 'pdf'
  return 'print'
}

// ── The shape of the flow ──────────────────────────────────────────────────

export type FlowMode = 'edition' | 'reading' | 'custom'
export type FlowStep = 'formats' | 'settings' | 'artwork' | 'finalize' | 'review'

// A reading copy is one file with no artwork and nothing to lock, so it skips
// three of the five steps. An edition ends at the lock decision, which is the
// only step in the flow the author cannot pass without answering.
export function stepsFor(mode: FlowMode): FlowStep[] {
  if (mode === 'reading') return ['settings', 'review']
  // Artwork belongs to an edition, so an export that belongs to none has no
  // cover and no wrap to carry and is not asked about them.
  if (mode === 'custom') return ['formats', 'settings', 'review']
  return ['formats', 'settings', 'artwork', 'finalize']
}

export const STEP_LABELS: Record<FlowStep, string> = {
  formats: 'Formats', settings: 'Settings', artwork: 'Artwork',
  finalize: 'Finalize', review: 'Review',
}

// ── What is being exported ─────────────────────────────────────────────────

// One selectable thing. For an edition it is a registered format, identified
// by the record's own id so that two paperbacks are two rows; for a
// from-scratch export the output kind is its own identity.
export interface FlowItem {
  id: string
  output: FlowOutput
  label: string
  desc: string
  meta: string
  /** The registered number, when this row has one. Never a condition. */
  isbn: string
  badge: string
  /** The record behind this row, absent on a from-scratch export. */
  record?: EditionFormat
}

// What kind of file a registered format produces. A hardcover is told from a
// paperback by the word on the record, which is the only place that
// distinction is written down.
export function outputForRecord(format: EditionFormat): FlowOutput {
  const word = (format.format ?? '').trim().toLowerCase()
  if (format.kind === 'ebook') return 'epub'
  if (format.kind === 'audio') return 'audio'
  if (word.startsWith('hardcover')) return 'hc'
  return 'print-pdf'
}

export function editionItems(edition: Edition | undefined): FlowItem[] {
  if (!edition) return []
  return edition.formats.map(format => {
    const output = outputForRecord(format)
    const isbn = (format.isbn13 ?? '').trim()
    return {
      id: format.id,
      output,
      label: (format.format ?? '').trim() || OUTPUT_INFO[output].label,
      desc: OUTPUT_INFO[output].desc,
      meta: OUTPUT_INFO[output].meta,
      isbn,
      // A format with no number exports exactly as well as one with a number.
      // The badge says the record is incomplete, not that the export is.
      badge: isbn ? '' : 'No ISBN',
      record: format,
    }
  })
}

export function scratchItems(): FlowItem[] {
  return SCRATCH_OUTPUTS.map(output => ({
    id: output,
    output,
    label: OUTPUT_INFO[output].label,
    desc: OUTPUT_INFO[output].desc,
    meta: OUTPUT_INFO[output].meta,
    isbn: '',
    badge: '',
  }))
}

export function findEdition(index: EditionIndex | undefined, editionID: string): Edition | undefined {
  return index?.editions.find(edition => edition.id === editionID)
}

// ── The settings each format shows ─────────────────────────────────────────

// A field on the wizard's answers. A row with no field is a control the design
// calls for and the exporter does not read yet; it holds its value on screen
// and changes nothing in the file, which is better than pretending the option
// is not there.
export interface OptionField { group: OptionGroup; key: string }

export interface Choice { label: string; value: string | number }

export type SettingRow =
  | { kind: 'select'; id: string; label: string; field: OptionField | null; choices: Choice[] }
  | { kind: 'text'; id: string; label: string; field: OptionField | null }
  | { kind: 'toggle'; id: string; label: string; field: OptionField | null; invert?: boolean }
  | { kind: 'static'; id: string; label: string; value: string }

export interface SettingGroup {
  label: string
  note: string
  rows: SettingRow[]
}

const sel = (id: string, label: string, field: OptionField | null, choices: Choice[]): SettingRow =>
  ({ kind: 'select', id, label, field, choices })
const txt = (id: string, label: string, field: OptionField | null): SettingRow =>
  ({ kind: 'text', id, label, field })
const tog = (id: string, label: string, field: OptionField | null): SettingRow =>
  ({ kind: 'toggle', id, label, field })

// A toggle whose field is stored the other way round. "Title page" is shown
// ticked when omitTitlePage is false, because absence has to keep meaning the
// page is made.
const togNot = (id: string, label: string, field: OptionField): SettingRow =>
  ({ kind: 'toggle', id, label, field, invert: true })
const stat = (id: string, label: string, value: string): SettingRow =>
  ({ kind: 'static', id, label, value })

const plain = (values: string[]): Choice[] => values.map(label => ({ label, value: label }))

// The contents group is the same question for every format — what goes in the
// file — asked against that format's own answers.
function contentsGroup(group: OptionGroup, extra: SettingRow[] = []): SettingGroup {
  return {
    label: 'Contents',
    note: 'What goes in the file.',
    rows: [
      togNot('titlePage', 'Title page', { group, key: 'omitTitlePage' }),
      tog('includeCopyright', 'Copyright page', { group, key: 'includeCopyright' }),
      tog('includeFrontMatter', 'Front matter', { group, key: 'includeFrontMatter' }),
      tog('includeBackMatter', 'Back matter', { group, key: 'includeBackMatter' }),
      ...extra,
    ],
  }
}

// Every list below is the one in publishingDefaults.ts. Retuning the defaults
// from real Reedsy and Atticus output changes what these offer, in one place.
const TRIM_CHOICES: Choice[] = [
  ...TRIM_PRESETS.map(preset => ({ label: preset.label, value: preset.id as string })),
  { label: CUSTOM_TRIM_LABEL, value: 'custom' },
]

const LINE_HEIGHTS: Choice[] = LINE_HEIGHT_CHOICES
const DISPLAY_FACES: Choice[] = DISPLAY_FACE_CHOICES

const PT = (sizes: number[]): Choice[] => sizes.map(n => ({ label: `${n} pt`, value: n }))

function printGroups(record: EditionFormat | undefined, custom: boolean): SettingGroup[] {
  const pages = (record?.page_count ?? '').trim()
  const spine = record ? spineWidthLabel(record) : ''
  const page: SettingRow[] = [
    sel('trimSize', 'Trim size', { group: 'print', key: 'trimSize' }, TRIM_CHOICES),
  ]
  // The custom trim's two measurements only exist because the list above
  // offers Custom…; they are the same question continued, not a second one.
  if (custom) {
    page.push(txt('customWidth', 'Custom width (in)', { group: 'print', key: 'customWidth' }))
    page.push(txt('customHeight', 'Custom height (in)', { group: 'print', key: 'customHeight' }))
  }
  page.push(
    txt('bleed', 'Bleed (in)', { group: 'print', key: 'bleed' }),
    txt('gutterMargin', 'Gutter margin (in)', { group: 'print', key: 'gutterMargin' }),
    txt('outerMargin', 'Outer margin (in)', { group: 'print', key: 'outerMargin' }),
    txt('topMargin', 'Top margin (in)', { group: 'print', key: 'topMargin' }),
    txt('bottomMargin', 'Bottom margin (in)', { group: 'print', key: 'bottomMargin' }),
    tog('mirroredMargins', 'Mirrored margins', { group: 'print', key: 'mirroredMargins' }),
    tog('includeCropMarks', 'Crop marks', { group: 'print', key: 'includeCropMarks' }),
  )
  return [
    { label: 'Page', note: 'Trim and margins drive the cover wrap.', rows: page },
    { label: 'Type', note: '', rows: [
      sel('fontFamily', 'Body typeface', { group: 'print', key: 'fontFamily' }, [
        { label: 'Merriweather', value: 'merriweather' }, { label: 'Lato', value: 'lato' }]),
      sel('fontSize', 'Type size', { group: 'print', key: 'fontSize' }, PT(PRINT_TYPE_SIZES)),
      sel('lineHeight', 'Line spacing', { group: 'print', key: 'lineHeight' }, LINE_HEIGHTS),
      sel('textAlign', 'Alignment', { group: 'print', key: 'textAlign' }, [
        { label: 'Justified', value: 'justify' }, { label: 'Left aligned', value: 'left' }]),
      txt('paragraphIndent', 'First-line indent (in)', { group: 'print', key: 'paragraphIndent' }),
      sel('headingFont', 'Heading font', { group: 'print', key: 'headingFont' }, DISPLAY_FACES),
    ] },
    { label: 'Chapters & furniture', note: '', rows: [
      tog('chapterStartsRecto', 'Chapters start recto', { group: 'print', key: 'chapterStartsRecto' }),
      tog('dropCap', 'Drop cap', { group: 'print', key: 'dropCap' }),
      sel('dropCapLines', 'Drop cap depth', { group: 'print', key: 'dropCapLines' }, [
        { label: '2 lines', value: 2 }, { label: '3 lines', value: 3 }, { label: '4 lines', value: 4 }]),
      tog('runningHeaders', 'Running headers', { group: 'print', key: 'runningHeaders' }),
      sel('headerStyle', 'Header style', { group: 'print', key: 'headerStyle' }, [
        { label: 'Small caps', value: 'smallcaps' }, { label: 'Italic', value: 'italic' },
        { label: 'Normal', value: 'normal' }]),
      sel('pageNumberPosition', 'Page numbers', { group: 'print', key: 'pageNumberPosition' }, [
        { label: 'Bottom center', value: 'bottom-center' }, { label: 'Bottom outside', value: 'bottom-outside' },
        { label: 'Top outside', value: 'top-outside' }]),
      tog('generateHalfTitle', 'Half title', { group: 'print', key: 'generateHalfTitle' }),
      tog('generateTOC', 'Contents page', { group: 'print', key: 'generateTOC' }),
      sel('titlePageStyle', 'Title page style', { group: 'print', key: 'titlePageStyle' }, [
        { label: 'Classic', value: 'classic' }, { label: 'Minimal', value: 'minimal' },
        { label: 'Dramatic', value: 'dramatic' }]),
    ] },
    contentsGroup('print', pages || spine
      ? [stat('estimate', 'Estimated pages', [pages, spine ? `spine ${spine}` : ''].filter(Boolean).join(' · '))]
      : []),
  ]
}

// settingGroups is every control one format tab shows, in the order the design
// puts them.
export function settingGroups(
  output: FlowOutput, record: EditionFormat | undefined, options: WizardOptions,
): SettingGroup[] {
  if (output === 'print-pdf' || output === 'hc') {
    return printGroups(record, options.print.trimSize === 'custom')
  }
  if (output === 'epub') {
    return [
      { label: 'Reader defaults', note: 'Readers can override type for accessibility.', rows: [
        sel('fontFamily', 'Typeface', { group: 'epub', key: 'fontFamily' }, [
          { label: 'Reader default (smallest file)', value: 'reader' },
          { label: 'Embed Merriweather', value: 'merriweather' },
          { label: 'Embed Lato', value: 'lato' }]),
        sel('paragraphStyle', 'Paragraphs', { group: 'epub', key: 'paragraphStyle' }, [
          { label: 'Book-style indents', value: 'indented' },
          { label: 'Space between paragraphs', value: 'spaced' }]),
        sel('textAlign', 'Alignment', { group: 'epub', key: 'textAlign' }, [
          { label: 'Reader default', value: 'reader' }, { label: 'Left aligned', value: 'left' },
          { label: 'Justified', value: 'justify' }]),
        sel('chapterStyle', 'Chapter opening', { group: 'epub', key: 'chapterStyle' }, [
          { label: 'Classic · lowered title', value: 'classic' },
          { label: 'Minimal · compact title', value: 'minimal' }]),
        sel('sceneBreakStyle', 'Scene breaks', { group: 'epub', key: 'sceneBreakStyle' }, [
          { label: 'Asterism ⁂', value: 'asterism' }, { label: 'Short rule', value: 'rule' },
          { label: 'Open space', value: 'space' }]),
        sel('version', 'EPUB version', { group: 'epub', key: 'version' }, plain(EPUB_VERSION_CHOICES)),
      ] },
      contentsGroup('epub'),
    ]
  }
  if (output === 'pdf') {
    return [
      { label: 'Page & type', note: '', rows: [
        sel('pageSize', 'Page size', { group: 'pdf', key: 'pageSize' }, PAGE_SIZE_CHOICES),
        sel('fontFamily', 'Typeface', { group: 'pdf', key: 'fontFamily' }, [
          { label: 'Merriweather', value: 'merriweather' }, { label: 'Lato', value: 'lato' }]),
        sel('fontSize', 'Type size', { group: 'pdf', key: 'fontSize' }, PT(READING_TYPE_SIZES)),
        sel('lineHeight', 'Line spacing', { group: 'pdf', key: 'lineHeight' }, LINE_HEIGHTS),
        sel('textAlign', 'Alignment', { group: 'pdf', key: 'textAlign' }, [
          { label: 'Left aligned', value: 'left' }, { label: 'Justified', value: 'justify' }]),
        txt('paragraphIndent', 'First-line indent (in)', { group: 'pdf', key: 'paragraphIndent' }),
      ] },
      { label: 'Reading copy', note: '', rows: [
        togNot('pageNumbers', 'Page numbers', { group: 'pdf', key: 'hideFolios' }),
        tog('draftWatermark', '“Draft” watermark', { group: 'pdf', key: 'draftWatermark' }),
      ] },
      contentsGroup('pdf'),
    ]
  }
  if (output === 'audio') {
    // A narration script is its own document with its own options. Every one
    // of these reaches the renderer; they were all dead controls before.
    const a = (key: string): OptionField => ({ group: 'audio', key })
    return [
      { label: 'Script layout', note: 'Built for reading aloud, not for print.', rows: [
        sel('pageSize', 'Page size', a('pageSize'), [
          { label: 'US Letter', value: 'letter' }, { label: 'A4', value: 'a4' }]),
        sel('fontFamily', 'Typeface', a('fontFamily'), [
          { label: 'Lato', value: 'lato' }, { label: 'Merriweather', value: 'merriweather' }]),
        sel('fontSize', 'Type size', a('fontSize'), PT(AUDIO_TYPE_SIZES)),
        sel('lineHeight', 'Line spacing', a('lineHeight'), [
          { label: 'Relaxed · 1.5', value: 1.5 }, { label: 'Open · 1.8', value: 1.8 },
          { label: 'Double · 2.0', value: 2 }]),
        sel('paragraphSpacing', 'Paragraph spacing', a('paragraphSpacing'), [
          { label: 'Half line between', value: 'half line between' },
          { label: 'One line between', value: 'one line between' },
          { label: 'Two lines between', value: 'two lines between' }]),
        // Ragged right is not a preference here. Justification moves words
        // between takes, and a narrator reading a line twice must see the
        // same line, so the script has no alignment to choose.
        stat('align', 'Alignment', 'Left aligned, ragged'),
      ] },
      { label: 'Narration aids', note: '', rows: [
        tog('slatePage', 'Chapter slate page', a('slatePage')),
        tog('numberParagraphs', 'Number every paragraph', a('numberParagraphs')),
        tog('pauseBreaks', 'Scene breaks as [PAUSE]', a('pauseBreaks')),
        tog('pronunciationColumn', 'Pronunciation notes column', a('pronunciationColumn')),
        tog('coverPage', 'Opening page with ebook cover', a('coverPage')),
        tog('chapterWordCount', 'Chapter length on the slate', a('chapterWordCount')),
      ] },
      contentsGroup('audio'),
    ]
  }
  return [
    { label: 'Word document', note: 'Familiar styles over locked design.', rows: [
      sel('bodyStyle', 'Body style', { group: 'docx', key: 'bodyStyle' }, [
        { label: 'Normal · 12 pt proportional', value: 'normal' },
        { label: 'Manuscript · 12 pt Courier, double spaced', value: 'manuscript' }]),
      sel('chapterBreak', 'Chapter headings', { group: 'docx', key: 'chapterBreak' }, [
        { label: 'Start a new page', value: 'page' },
        { label: 'Run on', value: 'run' }]),
      tog('trackChanges', 'Track changes ready', { group: 'docx', key: 'trackChanges' }),
      tog('hashSceneBreaks', 'Scene breaks as #', { group: 'docx', key: 'hashSceneBreaks' }),
    ] },
    contentsGroup('docx'),
  ]
}

// ── Reading and writing one control ────────────────────────────────────────

type Bag = Record<string, unknown>

export function readField(options: WizardOptions, field: OptionField): string | number | boolean {
  const bag = options[field.group] as unknown as Bag
  const value = bag[field.key]
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') return value
  return ''
}

export function writeField(
  options: WizardOptions, field: OptionField, value: string | number | boolean,
): WizardOptions {
  const bag = options[field.group] as unknown as Bag
  return { ...options, [field.group]: { ...bag, [field.key]: value } } as WizardOptions
}

// ── The preview beside the settings ────────────────────────────────────────

export interface Preview { ratio: string; caption: string }

// The page shape and the one line under it. Both are read off the answers the
// author has actually given, so changing the trim changes the picture.
export function previewFor(output: FlowOutput, options: WizardOptions): Preview {
  if (output === 'print-pdf' || output === 'hc') {
    const print = options.print
    const size = print.trimSize === 'custom'
      ? { w: Number(print.customWidth) || 6, h: Number(print.customHeight) || 9 }
      : trimInches(print.trimSize)
    return {
      ratio: `${size.w}/${size.h}`,
      caption: [
        `${size.w} × ${size.h} in`,
        `${faceName(print.fontFamily)} ${print.fontSize} pt`,
        print.runningHeaders ? 'running headers' : 'no running headers',
      ].join(' · '),
    }
  }
  if (output === 'epub') {
    return {
      ratio: '1/1.5',
      caption: options.epub.fontFamily === 'reader'
        ? 'Reflowable · reader controls type'
        : `Reflowable · ${faceName(options.epub.fontFamily)} embedded`,
    }
  }
  if (output === 'audio') {
    return { ratio: '8.5/11', caption: 'US Letter · Lato 14 pt · open spacing · slates' }
  }
  if (output === 'pdf') {
    const pdf = options.pdf
    return { ratio: pageRatio(pdf.pageSize), caption: `${pageName(pdf.pageSize)} · ${faceName(pdf.fontFamily)} ${pdf.fontSize} pt` }
  }
  return { ratio: '8.5/11', caption: 'Word styles · Heading 1 per chapter' }
}

function trimInches(trim: string): { w: number; h: number } {
  switch (trim) {
    case '5x8': return { w: 5, h: 8 }
    case '5.25x8': return { w: 5.25, h: 8 }
    case '6x9': return { w: 6, h: 9 }
    default: return { w: 5.5, h: 8.5 }
  }
}

function pageRatio(size: string): string {
  switch (size) {
    case 'a4': return '210/297'
    case '6x9': return '6/9'
    case '5.5x8.5': return '5.5/8.5'
    case '5x8': return '5/8'
    default: return '8.5/11'
  }
}

function pageName(size: string): string {
  switch (size) {
    case 'a4': return 'A4'
    case '6x9': return '6 × 9 in'
    case '5.5x8.5': return '5.5 × 8.5 in'
    case '5x8': return '5 × 8 in'
    default: return 'US Letter'
  }
}

function faceName(face: string): string {
  if (face === 'lato') return 'Lato'
  if (face === 'merriweather') return 'Merriweather'
  return face
}

// ── The artwork step ───────────────────────────────────────────────────────

// One row per format that has any artwork at all. A DOCX and a reading copy
// have none, so they are not asked about.
export interface ArtRow {
  id: string
  label: string
  /** A print format shows the wrap; an ebook or a script shows the cover. */
  wide: boolean
  thumbURL: string
  thumbText: string
  badge: string
  badgeKind: 'ok' | 'warn'
  file: string
  spec: string
  canRegen: boolean
}

export function artworkRows(
  items: FlowItem[], edition: Edition | undefined, title: string,
): ArtRow[] {
  return items.filter(item => item.output !== 'docx' && item.output !== 'pdf').map(item => {
    const print = item.output === 'print-pdf' || item.output === 'hc'
    const wrap = item.record?.wrap
    const cover = edition?.cover
    const held = print ? !!wrap : !!cover
    const thumbURL = print
      ? (edition && wrap ? wrapPreviewURL(edition.id, wrap) : '')
      : (edition ? coverThumbURL(edition) : '')
    return {
      id: item.id,
      label: item.label,
      wide: print,
      thumbURL,
      thumbText: print ? 'back · spine · front' : title,
      badge: held ? (print ? 'From this format' : 'From edition') : 'Not set',
      badgeKind: held ? 'ok' as const : 'warn' as const,
      file: print
        ? (wrap?.file_name || 'Choose a wrap file…')
        : (cover?.file || 'Choose a cover file…'),
      spec: artSpec(item, edition),
      canRegen: print && !!item.record,
    }
  })
}

function artSpec(item: FlowItem, edition: Edition | undefined): string {
  if (item.output === 'print-pdf' || item.output === 'hc') {
    const wrap = item.record?.wrap
    if (wrap?.size_label) return `${wrap.size_label} wrap`
    if (wrap?.width && wrap.height) return `${wrap.width} × ${wrap.height} wrap`
    const spine = item.record ? spineWidthLabel(item.record) : ''
    return spine ? `Wrap sized from trim and page count · spine ${spine}` : 'Wrap sized from trim and page count'
  }
  if (item.output === 'audio') return 'Ebook cover, placed on the script’s opening page'
  const cover = edition?.cover
  return cover ? `${cover.width} × ${cover.height} · ${cover.greyscale ? 'greyscale' : 'sRGB'}` : '1600 × 2560 · sRGB'
}

// ── The review ─────────────────────────────────────────────────────────────

export interface ReviewRow { k: string; v: string }

export interface ReviewInput {
  mode: FlowMode
  edition?: Edition
  items: FlowItem[]
  includeArt: boolean
  art: ArtRow[]
  lock: boolean | null
  words: number
  fileCount: number
  /** The book's title, for naming the archive an edition export produces. */
  title?: string
}

export function reviewRows(input: ReviewInput): ReviewRow[] {
  const rows: ReviewRow[] = [
    { k: 'Source', v: input.edition ? [input.edition.label, input.edition.year].filter(Boolean).join(' · ') : 'Current draft' },
    { k: 'Formats', v: input.items.map(item => item.label).join(' · ') || 'None' },
    { k: 'Artwork', v: input.mode === 'reading' ? 'None' : input.includeArt ? (input.art.map(a => a.file).join(' · ') || 'None attached') : 'Interior only' },
  ]
  if (input.mode === 'edition') {
    rows.push({ k: 'Text', v: input.lock ? `Locked snapshot · ${input.words.toLocaleString()} words` : 'Current draft, not locked' })
    // An edition goes out as one publication-ready archive rather than as a
    // handful of files, so the review says what will actually be on disk.
    rows.push({ k: 'Bundle', v: `${bundleName(input.title ?? '', input.edition)}.zip` })
    rows.push({
      k: 'Files',
      v: `${input.fileCount} ${input.fileCount === 1 ? 'file' : 'files'} in ${input.items.length} ${input.items.length === 1 ? 'folder' : 'folders'}`,
    })
    return rows
  }
  rows.push({ k: 'Files', v: `${input.fileCount} ${input.fileCount === 1 ? 'file' : 'files'}` })
  return rows
}

// The archive's name, and the name of the one folder inside it. It mirrors
// export.SafeName in Go so the review says what the save dialog will offer.
export function bundleName(title: string, edition: Edition | undefined): string {
  const book = title.trim() || 'Untitled'
  const label = (edition?.label ?? '').trim()
  return safeName(label ? `${book} — ${label}` : book)
}

function safeName(name: string): string {
  const cleaned = name.replace(/[/\\:*?"<>|]/g, '-').replace(/[\u0000-\u001f]/g, ' ')
  const tidied = cleaned.split(/\s+/).filter(Boolean).join(' ').replace(/[. ]+$/, '')
  return tidied.slice(0, 120).trim() || 'Untitled'
}

// ── Handing an edition export to Go ────────────────────────────────────────

// Every format's answers carry the edition and the format they were made for.
// The exporter reads the frozen text and the cover out of the record with
// them, so an item that arrives unstamped exports the wrong book.
export function stamped(options: WizardOptions, editionID: string, formatID: string): WizardOptions {
  const id = { editionID, formatID }
  return {
    shared: { ...options.shared, ...id },
    epub: { ...options.epub, ...id },
    pdf: { ...options.pdf, ...id },
    print: { ...options.print, ...id },
    audio: { ...options.audio, ...id },
    docx: { ...options.docx, ...id },
  }
}

export interface BundleItemPayload {
  format_id: string
  output: FlowOutput
  shared: WizardOptions['shared']
  epub: WizardOptions['epub']
  pdf: WizardOptions['pdf']
  audio: WizardOptions['audio']
  docx: WizardOptions['docx']
  print: WizardOptions['print']
}

export interface BundleRequestPayload {
  edition_id: string
  include_artwork: boolean
  items: BundleItemPayload[]
}

export function bundleRequest(
  editionID: string, includeArt: boolean, items: FlowItem[],
  optionsFor: (item: FlowItem) => WizardOptions,
): BundleRequestPayload {
  return {
    edition_id: editionID,
    include_artwork: includeArt,
    items: items.map(item => {
      const options = stamped(optionsFor(item), editionID, item.id)
      return {
        format_id: item.id,
        output: item.output,
        shared: options.shared,
        epub: options.epub,
        pdf: options.pdf,
        audio: options.audio,
        docx: options.docx,
        print: options.print,
      }
    }),
  }
}

// How many files the export writes: one per format, plus the artwork carried
// beside the print and ebook ones.
export function fileCount(items: FlowItem[], includeArt: boolean, art: ArtRow[]): number {
  return items.length + (includeArt ? art.length : 0)
}

// ── The one-line summary under each step in the rail ───────────────────────

export interface StepSummaryInput {
  items: FlowItem[]
  touched: boolean
  fromEdition: boolean
  includeArt: boolean
  lock: boolean | null
  fileCount: number
}

export function stepSummary(step: FlowStep, input: StepSummaryInput): string {
  switch (step) {
    case 'formats': return input.items.map(item => item.label).join(', ') || 'None selected'
    case 'settings': return input.touched ? 'Edited' : `From ${input.fromEdition ? 'edition template' : 'defaults'}`
    case 'artwork': return input.includeArt ? 'Included' : 'Interior only'
    case 'finalize': return input.lock === null ? 'Lock decision required' : input.lock ? 'Text locked' : 'Current draft'
    case 'review': return `${input.fileCount} ${input.fileCount === 1 ? 'file' : 'files'}`
  }
}
