// A format's typeset template: the settings an export of this ISBN starts
// from, edited where the ISBN lives rather than only inside the export wizard.
//
// This is the join between the two screens. The template IS the export
// wizard's answers — the same WizardOptions object, stored in the same
// `export_settings` block on the format record — so anything set here prefills
// the wizard, and anything the wizard saves back shows up here. There is one
// template per format and one place it is written.
//
// Two kinds of setting live on a format and they are kept apart on purpose:
//
//   the specification   Trim, paper, binding, bleed, page count, EPUB version.
//                       These describe the published object and are fields on
//                       the record, because a printer and a storefront read
//                       them, not an exporter.
//   the template        Margins, type, furniture, contents. These are how the
//                       file is set, and they are the export's options.
//
// Where a control has no field in either — a print scene break, a narration
// aid — it is offered and held, and says so, rather than being hidden.

import type { Edition, EditionFormat } from '../../types/draftline'
import { hardcoverSpineRequiresTemplate, spineWidthLabel } from './editionModel'
import {
  defaultWizardOptions, exportSettingsBlock, wizardOptionsForFormat,
  type PrintPDFOptions, type WizardOptions,
} from './exportSource'
import {
  DISPLAY_FACE_CHOICES, EPUB_VERSION_CHOICES, LINE_HEIGHT_CHOICES,
  PRINT_TYPE, PRINT_TYPE_SIZES, TRIM_PRESETS,
} from './publishingDefaults'

export type FormatKind = 'print' | 'ebook' | 'audio'

export function kindOf(format: EditionFormat): FormatKind {
  if (format.kind === 'ebook') return 'ebook'
  if (format.kind === 'audio') return 'audio'
  return 'print'
}

/** Which of the wizard's option sets this format's template lives in. */
export function groupOf(kind: FormatKind): 'epub' | 'pdf' | 'print' {
  if (kind === 'ebook') return 'epub'
  if (kind === 'audio') return 'pdf'
  return 'print'
}

export function templateFor(edition: Edition, format: EditionFormat): WizardOptions {
  return wizardOptionsForFormat(edition, format)
}

// ── Standard or custom ─────────────────────────────────────────────────────

export type Typesetting = 'standard' | 'custom'

/**
 * Which way this format is set.
 *
 * It is read off the record rather than worked out by comparing settings
 * against the defaults. That comparison was the bug: applying the standard and
 * reading it back disagreed, and a format whose standard happened to equal the
 * defaults could never be moved off it. Empty means standard, so a record
 * written before this existed is on the standard, which it was.
 */
export function typesettingOf(format: EditionFormat): Typesetting {
  return (format.typesetting ?? '').trim().toLowerCase() === 'custom' ? 'custom' : 'standard'
}

/**
 * Going back to the standard: the whole template, as it ships.
 *
 * The trim is carried across. It is the physical size of the book rather than
 * a typesetting decision, it is chosen on its own card above, and quietly
 * turning a 6 × 9 paperback back into 5.5 × 8.5 because somebody clicked
 * "Industry standard" would be a worse surprise than any it prevents.
 */
export function standardPatch(current: WizardOptions): Partial<EditionFormat> {
  const fresh = defaultWizardOptions()
  fresh.print = {
    ...fresh.print,
    trimSize: current.print.trimSize,
    customWidth: current.print.customWidth,
    customHeight: current.print.customHeight,
    pageSize: current.print.pageSize,
  }
  return { typesetting: 'standard', export_settings: exportSettingsBlock(fresh) }
}

/** Moving to custom. It changes no setting; it unlocks them. */
export function customPatch(): Partial<EditionFormat> {
  return { typesetting: 'custom' }
}

/**
 * Saving an edit made in Advanced.
 *
 * Reaching into Advanced and changing a margin is what "custom" means, so it
 * is recorded with the edit rather than left for the author to declare
 * separately.
 */
export function templateEditPatch(options: WizardOptions): Partial<EditionFormat> {
  return { typesetting: 'custom', export_settings: exportSettingsBlock(options) }
}

/**
 * Saving a preference: one of the cards above the Advanced line.
 *
 * A print size, a scene-break mark, a chapter opening, what the running head
 * says — these configure the book without leaving the industry standard. A
 * paperback set at 6 x 9 with a short rule between scenes is still set the
 * standard way, and telling an author they have gone Custom for choosing one
 * is both wrong and the reason those cards felt like traps.
 */
export function preferencePatch(options: WizardOptions): Partial<EditionFormat> {
  return { export_settings: exportSettingsBlock(options) }
}

// ── The choice cards ───────────────────────────────────────────────────────

export interface TemplateOption {
  id: string
  label: string
  sub?: string
  glyph?: string
}

// One card on the settings screen.
//
// A choice carries both how it is read and how it is written. Keeping `set` on
// the choice rather than in a separate switch means a card that has nowhere to
// store its answer cannot be declared at all, so the compiler catches it
// instead of the card rendering and quietly ignoring every click.
export interface TemplateChoice {
  id: string
  label: string
  hint: string
  value: string
  options: TemplateOption[]
  /** Applies one of `options` and returns the answers to save. */
  set: (options: WizardOptions, value: string) => WizardOptions
}

const TRIM_OPTIONS: TemplateOption[] = [
  { id: '5x8', label: '5 × 8 in', sub: 'Mass market' },
  { id: '5.25x8', label: '5.25 × 8 in', sub: 'Digest' },
  { id: '5.5x8.5', label: '5.5 × 8.5 in', sub: 'Small trade' },
  { id: '6x9', label: '6 × 9 in', sub: 'Trade · most common' },
  { id: 'custom', label: 'Custom…', sub: 'Set in Advanced' },
]

// What choosing the standard actually means, per kind. It is said once, beside
// the one control that decides it.
export const STANDARD_HINT: Record<FormatKind, string> = {
  print: 'Mirrored margins, 0.875 in gutter, justified text, chapters on the right-hand page, running headers.',
  ebook: 'Book-style indents, reader-controlled type, linked contents, valid reflowable XHTML.',
  audio: 'Open leading, ragged right, a slate before each chapter.',
}

const DROPCAP_OPTIONS: TemplateOption[] = [
  { id: 'on', label: 'On', glyph: 'T' },
  { id: 'off', label: 'Off' },
]

const ON_OFF: TemplateOption[] = [{ id: 'on', label: 'On' }, { id: 'off', label: 'Off' }]

const SCENE_OPTIONS: TemplateOption[] = [
  { id: 'asterism', label: 'Asterism', glyph: '⁂' },
  { id: 'rule', label: 'Short rule', glyph: '—' },
  { id: 'space', label: 'Whitespace', sub: 'Blank line' },
]

const CHAPTER_OPTIONS: TemplateOption[] = [
  { id: 'classic', label: 'Classic', sub: 'Lowered title, generous space' },
  { id: 'compact', label: 'Compact', sub: 'Title tight to the text' },
]

export function choicesFor(format: EditionFormat, options: WizardOptions): TemplateChoice[] {
  const kind = kindOf(format)
  const out: TemplateChoice[] = []

  if (kind === 'print') {
    out.push({
      id: 'trim', label: 'Print size', hint: 'Trade paperback is 6 × 9.',
      value: options.print.trimSize, options: TRIM_OPTIONS,
      set: (o, value) => {
        const preset = TRIM_PRESETS.find(one => one.id === value)
        return {
          ...o,
          print: {
            ...o.print,
            trimSize: value as PrintPDFOptions['trimSize'],
            ...(preset ? { customWidth: String(preset.width), customHeight: String(preset.height) } : {}),
          },
        }
      },
    })
  }

  // Drop caps. An ebook has them too — ::first-letter is ordinary CSS that
  // Kindle honours — which is why this is no longer a print-only card.
  if (kind === 'print') {
    out.push({
      id: 'dropcap', label: 'Drop caps', hint: 'First letter of each chapter.',
      value: options.print.dropCap ? 'on' : 'off', options: DROPCAP_OPTIONS,
      set: (o, value) => ({ ...o, print: { ...o.print, dropCap: value === 'on' } }),
    })
  }
  if (kind === 'ebook') {
    out.push({
      id: 'dropcap', label: 'Drop caps', hint: 'First letter of each chapter.',
      value: options.epub.dropCap ? 'on' : 'off', options: DROPCAP_OPTIONS,
      set: (o, value) => ({ ...o, epub: { ...o.epub, dropCap: value === 'on' } }),
    })
  }

  // Scene breaks. A script says [PAUSE] out loud, because an asterism is
  // silent and a narrator cannot act on it.
  if (kind === 'print') {
    out.push({
      id: 'scene', label: 'Scene breaks', hint: '',
      value: options.print.sceneBreakStyle, options: SCENE_OPTIONS,
      set: (o, value) => ({
        ...o, print: { ...o.print, sceneBreakStyle: value as 'asterism' | 'rule' | 'space' },
      }),
    })
  }
  if (kind === 'ebook') {
    out.push({
      id: 'scene', label: 'Scene breaks', hint: '',
      value: options.epub.sceneBreakStyle, options: SCENE_OPTIONS,
      set: (o, value) => ({
        ...o, epub: { ...o.epub, sceneBreakStyle: value as 'asterism' | 'rule' | 'space' },
      }),
    })
  }
  if (kind === 'audio') {
    out.push({
      id: 'scene', label: 'Scene breaks', hint: 'A narrator cannot say an asterism.',
      value: options.audio.pauseBreaks ? 'pause' : 'space',
      options: [
        { id: 'pause', label: '[PAUSE]', sub: 'Narration cue' },
        { id: 'space', label: 'Whitespace', sub: 'Blank line' },
      ],
      set: (o, value) => ({ ...o, audio: { ...o.audio, pauseBreaks: value === 'pause' } }),
    })
  }

  // Chapter openings. For a script this is the slate: the chapter on a page of
  // its own, which is the cue a take is recorded against.
  if (kind === 'print') {
    out.push({
      id: 'chapter', label: 'Chapter headers', hint: '',
      value: options.print.chapterStyle, options: CHAPTER_OPTIONS,
      set: (o, value) => ({
        ...o, print: { ...o.print, chapterStyle: value === 'compact' ? 'compact' : 'classic' },
      }),
    })
  }
  if (kind === 'ebook') {
    out.push({
      id: 'chapter', label: 'Chapter headers', hint: '',
      value: options.epub.chapterStyle === 'minimal' ? 'compact' : 'classic',
      options: CHAPTER_OPTIONS,
      set: (o, value) => ({
        ...o, epub: { ...o.epub, chapterStyle: value === 'compact' ? 'minimal' : 'classic' },
      }),
    })
  }
  if (kind === 'audio') {
    out.push({
      id: 'chapter', label: 'Chapter openings', hint: 'What a take is recorded against.',
      value: options.audio.slatePage ? 'slate' : 'classic',
      options: [
        { id: 'slate', label: 'Slate page', sub: 'Chapter on its own page' },
        { id: 'classic', label: 'Running', sub: 'Chapter title above the text' },
      ],
      set: (o, value) => ({ ...o, audio: { ...o.audio, slatePage: value === 'slate' } }),
    })
  }

  // What a reader sees along the top of a spread, and where the folio sits.
  if (kind === 'print') {
    out.push({
      id: 'heads', label: 'Running heads', hint: 'Along the top of every page of the manuscript.',
      value: options.print.runningHeaders ? options.print.headerContent : 'none',
      options: [
        { id: 'author-title', label: 'Author & title', sub: 'Author on left pages, title on right' },
        { id: 'title-chapter', label: 'Title & chapter', sub: 'Title on left pages, chapter on right' },
        { id: 'chapter', label: 'Chapter only', sub: 'Chapter on both pages' },
        { id: 'none', label: 'None', sub: 'Page numbers only' },
      ],
      // "None" is the absence of a running head rather than a fourth thing one
      // can say, so it turns them off and leaves the wording alone.
      set: (o, value) => value === 'none'
        ? { ...o, print: { ...o.print, runningHeaders: false } }
        : {
          ...o,
          print: {
            ...o.print,
            runningHeaders: true,
            headerContent: value as PrintPDFOptions['headerContent'],
          },
        },
    })
    out.push({
      id: 'folio', label: 'Page numbers', hint: '',
      value: options.print.pageNumberPosition === 'top-outside' ? 'top-outside' : 'bottom-center',
      options: [
        { id: 'top-outside', label: 'Top outside', sub: 'On the running head’s line' },
        { id: 'bottom-center', label: 'Bottom centre', sub: 'Under the text block' },
      ],
      set: (o, value) => ({
        ...o,
        print: { ...o.print, pageNumberPosition: value as PrintPDFOptions['pageNumberPosition'] },
      }),
    })
  }

  // The narrator's own aids. Each one changes the page it is printed on.
  if (kind === 'audio') {
    out.push({
      id: 'numbers', label: 'Paragraph numbers',
      hint: 'So a retake is asked for by number, not by reading the line back.',
      value: options.audio.numberParagraphs ? 'on' : 'off', options: ON_OFF,
      set: (o, value) => ({ ...o, audio: { ...o.audio, numberParagraphs: value === 'on' } }),
    })
    out.push({
      id: 'pronunciation', label: 'Pronunciation column',
      hint: 'Leaves the outer margin wide to write names into.',
      value: options.audio.pronunciationColumn ? 'on' : 'off', options: ON_OFF,
      set: (o, value) => ({ ...o, audio: { ...o.audio, pronunciationColumn: value === 'on' } }),
    })
  }

  return out
}

/** Taking one of the cards above. Returns the same object when nothing moved. */
export function chooseTemplate(
  options: WizardOptions, format: EditionFormat, choiceID: string, value: string,
): WizardOptions {
  const choice = choicesFor(format, options).find(one => one.id === choiceID)
  return choice ? choice.set(options, value) : options
}


/**
 * Whether taking this card is a deviation rather than a preference.
 *
 * Only one is: a custom trim is a page size no print-on-demand service offers
 * as standard, and the two measurements that define it live in Advanced. So it
 * moves the format to Custom and opens the drawer where it can be finished.
 * Everything else on those cards configures the book and leaves it standard.
 */
export function choiceGoesCustom(choiceID: string, value: string): boolean {
  return choiceID === 'trim' && value === 'custom'
}

// ── Advanced ───────────────────────────────────────────────────────────────

// A row either edits the record (a specification a printer reads) or the
// template (how the file is set). `record` names the field on EditionFormat;
// `field` names the key inside the template's own option group.
export type AdvancedRow =
  | { kind: 'select'; id: string; label: string; hint?: string; value: string; options: string[]; record?: keyof EditionFormat; field?: string; numeric?: boolean }
  | { kind: 'text'; id: string; label: string; hint?: string; value: string; record?: keyof EditionFormat; field?: string; mono?: boolean }
  | { kind: 'toggle'; id: string; label: string; hint?: string; value: boolean; field?: string }
  | { kind: 'static'; id: string; label: string; hint?: string; value: string }

export interface AdvancedGroup { label: string; rows: AdvancedRow[] }

/**
 * Whether a row describes the published object rather than how it is set.
 *
 * A paper stock and a page count are read by a printer, not by an exporter,
 * and they stay editable whether the format is on the standard or not:
 * recording what a book is made of is not a typesetting decision.
 */
export function isSpecification(row: AdvancedRow): boolean {
  return 'record' in row && !!row.record
}

const PAPER_STOCKS = ['Cream, 55#', 'White, 60#', 'White, 50#', 'Groundwood, 45#']
const BINDINGS = ['Perfect bound', 'Case laminate', 'Cloth with jacket']
const BLEEDS = ['No bleed', 'Bleed 0.125 in']

const faces = DISPLAY_FACE_CHOICES.map(choice => choice.label)
const leadings = LINE_HEIGHT_CHOICES.map(choice => choice.label)

function leadingLabel(value: number): string {
  return LINE_HEIGHT_CHOICES.find(choice => choice.value === value)?.label ?? String(value)
}

function faceLabel(value: string): string {
  return DISPLAY_FACE_CHOICES.find(choice => choice.value === value)?.label ?? value
}

export function advancedFor(format: EditionFormat, options: WizardOptions): AdvancedGroup[] {
  const kind = kindOf(format)
  const groups: AdvancedGroup[] = []

  if (kind === 'print') {
    const print = options.print
    groups.push({
      label: 'Book block',
      rows: [
        { kind: 'select', id: 'paper', label: 'Paper stock', value: format.paper_stock ?? '', options: PAPER_STOCKS, record: 'paper_stock' },
        { kind: 'select', id: 'binding', label: 'Binding', value: format.binding ?? '', options: BINDINGS, record: 'binding' },
        { kind: 'select', id: 'bleed', label: 'Bleed', value: format.bleed ?? '', options: BLEEDS, record: 'bleed' },
        { kind: 'text', id: 'pages', label: 'Page count', hint: 'From the last typeset pass.', value: format.page_count ?? '', record: 'page_count', mono: true },
        { kind: 'static', id: 'spine', label: 'Spine width', value: hardcoverSpineRequiresTemplate(format)
          ? 'Use the printer’s hardcover template' : spineWidthLabel(format) || 'Set a page count' },
      ],
    })
    groups.push({
      label: 'Margins',
      rows: [
        { kind: 'text', id: 'gutter', label: 'Gutter (in)', value: print.gutterMargin, field: 'gutterMargin', mono: true },
        { kind: 'text', id: 'outer', label: 'Outer (in)', value: print.outerMargin, field: 'outerMargin', mono: true },
        { kind: 'text', id: 'top', label: 'Top (in)', value: print.topMargin, field: 'topMargin', mono: true },
        { kind: 'text', id: 'bottom', label: 'Bottom (in)', value: print.bottomMargin, field: 'bottomMargin', mono: true },
        { kind: 'toggle', id: 'mirrored', label: 'Mirrored margins', value: print.mirroredMargins, field: 'mirroredMargins' },
        { kind: 'toggle', id: 'crop', label: 'Crop marks', value: print.includeCropMarks, field: 'includeCropMarks' },
      ],
    })
    groups.push({
      label: 'Type',
      rows: [
        { kind: 'select', id: 'face', label: 'Body typeface', value: print.fontFamily === 'lato' ? 'Lato' : 'Merriweather', options: ['Merriweather', 'Lato'], field: 'fontFamily' },
        { kind: 'select', id: 'size', label: 'Type size', value: `${print.fontSize} pt`, options: PRINT_TYPE_SIZES.map(n => `${n} pt`), field: 'fontSize', numeric: true },
        { kind: 'select', id: 'leading', label: 'Line spacing', value: leadingLabel(print.lineHeight), options: leadings, field: 'lineHeight', numeric: true },
        { kind: 'select', id: 'align', label: 'Alignment', value: print.textAlign === 'justify' ? 'Justified' : 'Left aligned', options: ['Justified', 'Left aligned'], field: 'textAlign' },
        { kind: 'text', id: 'indent', label: 'First-line indent (in)', value: print.paragraphIndent, field: 'paragraphIndent', mono: true },
        { kind: 'select', id: 'heading', label: 'Heading font', value: faceLabel(print.headingFont), options: faces, field: 'headingFont' },
      ],
    })
    groups.push({
      label: 'Chapters & furniture',
      rows: [
        { kind: 'toggle', id: 'recto', label: 'Chapters start recto', value: print.chapterStartsRecto, field: 'chapterStartsRecto' },
        { kind: 'select', id: 'dropLines', label: 'Drop cap depth', value: `${print.dropCapLines} lines`, options: ['2 lines', '3 lines', '4 lines'], field: 'dropCapLines', numeric: true },
        { kind: 'toggle', id: 'headers', label: 'Running headers', value: print.runningHeaders, field: 'runningHeaders' },
        { kind: 'select', id: 'headerStyle', label: 'Header style', value: headerWord(print.headerStyle), options: ['Small caps', 'Italic', 'Normal'], field: 'headerStyle' },
        { kind: 'select', id: 'headerContent', label: 'Running head says', value: headContentWord(print.headerContent), options: ['Author & title', 'Title & chapter', 'Chapter only'], field: 'headerContent' },
        { kind: 'select', id: 'folios', label: 'Page numbers', value: folioWord(print.pageNumberPosition), options: ['Bottom center', 'Bottom outside', 'Top outside'], field: 'pageNumberPosition' },
        { kind: 'toggle', id: 'halfTitle', label: 'Half title', value: print.generateHalfTitle, field: 'generateHalfTitle' },
        { kind: 'toggle', id: 'toc', label: 'Contents page', value: print.generateTOC, field: 'generateTOC' },
        { kind: 'select', id: 'titleStyle', label: 'Title page style', value: titleWord(print.titlePageStyle), options: ['Classic', 'Minimal', 'Dramatic'], field: 'titlePageStyle' },
      ],
    })
  }

  if (kind === 'ebook') {
    const epub = options.epub
    const digits = (format.isbn13 ?? '').replace(/[^0-9Xx]/g, '')
    groups.push({
      label: 'Package',
      rows: [
        { kind: 'select', id: 'epubVersion', label: 'EPUB version', value: format.epub_version ?? EPUB_VERSION_CHOICES[0], options: EPUB_VERSION_CHOICES, record: 'epub_version' },
        { kind: 'static', id: 'identifier', label: 'Unique identifier', value: digits ? `urn:isbn:${digits}` : 'A generated identifier', hint: digits ? '' : 'An export without an ISBN carries an identifier Draftline generates. Nothing is withheld for it.' },
      ],
    })
    groups.push({
      label: 'Reader defaults',
      rows: [
        { kind: 'select', id: 'face', label: 'Typeface', value: epubFaceWord(epub.fontFamily), options: ['Reader default', 'Embed Merriweather', 'Embed Lato'], field: 'fontFamily' },
        { kind: 'select', id: 'align', label: 'Alignment', value: epubAlignWord(epub.textAlign), options: ['Reader default', 'Left aligned', 'Justified'], field: 'textAlign' },
        { kind: 'select', id: 'paragraphs', label: 'Paragraphs', value: epub.paragraphStyle === 'spaced' ? 'Space between paragraphs' : 'Book-style indents', options: ['Book-style indents', 'Space between paragraphs'], field: 'paragraphStyle' },
      ],
    })
  }

  if (kind === 'audio') {
    const pdf = options.pdf
    groups.push({
      label: 'Script layout',
      rows: [
        { kind: 'select', id: 'face', label: 'Typeface', value: pdf.fontFamily === 'lato' ? 'Lato' : 'Merriweather', options: ['Lato', 'Merriweather'], field: 'fontFamily' },
        { kind: 'select', id: 'size', label: 'Type size', value: `${pdf.fontSize} pt`, options: ['12 pt', '14 pt'], field: 'fontSize', numeric: true },
        { kind: 'select', id: 'leading', label: 'Line spacing', value: leadingLabel(pdf.lineHeight), options: leadings, field: 'lineHeight', numeric: true },
      ],
    })
  }

  const group = groupOf(kind)
  const contents = options[group] as unknown as Record<string, unknown>
  groups.push({
    label: 'Contents',
    rows: [
      { kind: 'toggle', id: 'copyright', label: 'Copyright page', value: contents.includeCopyright === true, field: 'includeCopyright' },
      { kind: 'toggle', id: 'front', label: 'Front matter', value: contents.includeFrontMatter === true, field: 'includeFrontMatter' },
      { kind: 'toggle', id: 'back', label: 'Back matter', value: contents.includeBackMatter === true, field: 'includeBackMatter' },
    ],
  })
  return groups
}

/** Writing one Advanced row back into the template. */
export function setAdvanced(
  options: WizardOptions, kind: FormatKind, row: AdvancedRow, value: string | boolean,
): WizardOptions {
  const field = 'field' in row ? row.field : undefined
  if (!field) return options
  const group = groupOf(kind)
  const bag = options[group] as unknown as Record<string, unknown>
  return { ...options, [group]: { ...bag, [field]: decodeAdvanced(field, value) } } as WizardOptions
}

// The screen speaks in words and the options hold values. This is the one
// place the two vocabularies meet.
function decodeAdvanced(field: string, value: string | boolean): unknown {
  if (typeof value === 'boolean') return value
  switch (field) {
    case 'fontSize': return Number.parseInt(value, 10)
    case 'dropCapLines': return Number.parseInt(value, 10)
    case 'lineHeight': return LINE_HEIGHT_CHOICES.find(choice => choice.label === value)?.value ?? PRINT_TYPE.lineHeight
    case 'fontFamily':
      if (value === 'Lato' || value === 'Embed Lato') return 'lato'
      if (value === 'Merriweather' || value === 'Embed Merriweather') return 'merriweather'
      if (value === 'Reader default') return 'reader'
      return value.toLowerCase()
    case 'textAlign':
      if (value === 'Justified') return 'justify'
      if (value === 'Left aligned') return 'left'
      if (value === 'Reader default') return 'reader'
      return value.toLowerCase()
    case 'paragraphStyle': return value.startsWith('Space') ? 'spaced' : 'indented'
    case 'headingFont': return DISPLAY_FACE_CHOICES.find(choice => choice.label === value)?.value ?? 'body'
    case 'headerStyle': return value === 'Small caps' ? 'smallcaps' : value.toLowerCase()
    case 'pageNumberPosition':
      if (value === 'Bottom outside') return 'bottom-outside'
      if (value === 'Top outside') return 'top-outside'
      return 'bottom-center'
    case 'titlePageStyle': return value.toLowerCase()
    case 'headerContent':
      if (value === 'Title & chapter') return 'title-chapter'
      if (value === 'Chapter only') return 'chapter'
      return 'author-title'
    default: return value
  }
}

function headContentWord(value: string): string {
  if (value === 'title-chapter') return 'Title & chapter'
  if (value === 'chapter') return 'Chapter only'
  return 'Author & title'
}

function headerWord(value: string): string {
  if (value === 'italic') return 'Italic'
  if (value === 'normal') return 'Normal'
  return 'Small caps'
}

function folioWord(value: string): string {
  if (value === 'bottom-outside') return 'Bottom outside'
  if (value === 'top-outside') return 'Top outside'
  return 'Bottom center'
}

function titleWord(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1)
}

function epubFaceWord(value: string): string {
  if (value === 'merriweather') return 'Embed Merriweather'
  if (value === 'lato') return 'Embed Lato'
  return 'Reader default'
}

function epubAlignWord(value: string): string {
  if (value === 'left') return 'Left aligned'
  if (value === 'justify') return 'Justified'
  return 'Reader default'
}

/** The one line under a format in the Formats list and the rail. */
export function templateSummary(format: EditionFormat, options: WizardOptions): string {
  const kind = kindOf(format)
  if (kind === 'print') {
    const preset = TRIM_PRESETS.find(one => one.id === options.print.trimSize)
    const trim = (format.trim ?? '').trim() || preset?.label || 'custom trim'
    return `${trim} · ${faceWord(options.print.fontFamily)} ${options.print.fontSize} pt · ${options.print.runningHeaders ? 'running headers' : 'no headers'}`
  }
  if (kind === 'ebook') {
    return `${(format.epub_version ?? 'EPUB 3.3').trim()} · reflowable · ${options.epub.fontFamily === 'reader' ? 'reader type' : 'embedded type'}`
  }
  return 'Letter · narration script'
}

function faceWord(value: string): string {
  return value === 'lato' ? 'Lato' : 'Merriweather'
}
