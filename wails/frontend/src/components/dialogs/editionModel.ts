// What the Editions half of the Book & Editions screen shows, and the rules
// behind it. Pure functions over the publishing record: no React, no store, so
// the copyright page and the spine width can be tested as arithmetic rather
// than as a rendered component.
//
// Two of these are halves of a pair with Go:
//   copyrightLines  ↔ internal/types/copyright.go
//   spineWidthInches ↔ internal/types/editions.go
// The copyright halves are checked against one shared table of worked
// examples, internal/types/testdata/copyright_cases.json, so the preview the
// author watches and the page printed into the exported book cannot drift.

import type { Edition, EditionFormat, EditionIndex, EditionKind, Metadata } from '../../types/draftline'
import { isbn10, normalizeISBN } from './bookInfoModel'

// ── The words the record offers ────────────────────────────────────────────
// These are the design brief's own lists. A field still accepts anything
// stored in it, because a record written elsewhere is not wrong just because
// this build does not offer its wording.

export const FORMAT_WORDS = ['eBook', 'Paperback', 'Hardcover', 'Large print', 'Audiobook']

// The format word a writer picks decides what kind of thing the record is.
// Nothing should be a print record that says "eBook" on it, or an ebook that
// says "Hardcover": the word is the choice, and the kind follows it.
export function kindForFormat(word: string): EditionKind {
  switch (word.trim().toLowerCase()) {
    case 'ebook':
      return 'ebook'
    case 'audiobook':
      return 'audio'
    default:
      return 'print'
  }
}
export const REGISTRATIONS = ['Registered — agency', 'Free retailer ISBN', 'Not yet assigned']
export const FORMAT_STATUSES = ['Draft', 'Registered', 'Published', 'Out of print']
export const EDITION_STATUSES = ['Draft', 'In progress', 'Published', 'Out of print']
export const TRIMS = ['5 × 8 in', '5.25 × 8 in', '5.5 × 8.5 in', '6 × 9 in (trade)', '7 × 10 in', '148 × 210 mm (A5)']
export const PAPER_STOCKS = ['Cream, 55#', 'White, 60#', 'White, 50#', 'Groundwood, 45#']
export const BINDINGS = ['Perfect bound', 'Case laminate', 'Cloth with jacket']
export const BLEEDS = ['No bleed', 'Bleed 0.125 in']
export const INTERIORS = ['Black and white', 'Standard colour', 'Premium colour']
export const EPUB_VERSIONS = ['EPUB 3.3', 'EPUB 3.0', 'EPUB 2.0.1']
export const LAYOUTS = ['Reflowable', 'Fixed layout']
export const DRM_CHOICES = ['None', 'Retailer default']
export const TERRITORIES = ['World', 'World English', 'North America', 'Excluding UK and Commonwealth']
export const RIGHTS_NOTICES = ['All rights reserved', 'CC BY-NC-ND 4.0']

// The kind a format is decides which specification it has, and which colour
// marks it in the rail.
export const KIND_DOT: Record<EditionKind, string> = {
  ebook: 'var(--section-front)',
  print: 'var(--section-body)',
  audio: 'var(--section-back)',
}

// A kind read back out of a project file is only a string. One this build does
// not know gets the muted colour rather than no colour at all, so the row is
// still a row.
export const kindDot = (kind: string): string =>
  KIND_DOT[kind as EditionKind] ?? 'var(--text-muted)'

export type BadgeKind = 'ok' | 'warn' | 'accent' | 'neutral'

// A status is a state of the world, and the badge says which kind of state:
// settled (ok), under way (accent), not started (neutral), or needing the
// author's attention because the book can no longer be bought (warn).
export function statusBadgeKind(status: string): BadgeKind {
  switch (status.trim().toLowerCase()) {
    case 'published': return 'ok'
    case 'registered': return 'accent'
    case 'in progress': return 'warn'
    case 'out of print': return 'warn'
    default: return 'neutral'
  }
}

export function editionBadge(edition: Edition): { label: string; kind: BadgeKind } {
  const label = edition.status?.trim() || 'Draft'
  return { label, kind: statusBadgeKind(label) }
}

export function formatBadge(format: EditionFormat): { label: string; kind: BadgeKind } {
  const label = format.status?.trim() || 'Draft'
  return { label, kind: statusBadgeKind(label) }
}

// ── Derivations ────────────────────────────────────────────────────────────
// Never stored. A derived value written down is a derived value that goes
// stale against the field it came from, and on a copyright page that is a
// reprint with the wrong number on it.

export function derivedISBN10(format: EditionFormat, hyphenate = true): string {
  const ten = isbn10(format.isbn13 ?? '')
  if (!ten || !hyphenate) return ten
  return `${ten.slice(0, 1)}-${ten.slice(1, 9)}-${ten.slice(9)}`
}

// KDP publishes paperback calipers per page. A case-laminate cover also has
// hinges, boards and a wrap, so its dimensions must come from the printer's
// generated template rather than a paperback-style multiplier.
export function spinePerPage(stock: string): number {
  switch (normalizeSpec(stock)) {
    case 'white,60#': case 'white60#': case 'white,50#': case 'white50#': return 0.002252
    case 'groundwood,45#': case 'groundwood45#': return 0.00235
    case 'color': case 'colorpaper': case 'white,color': return 0.002347
    default: return 0.0025
  }
}

// 0 for anything that is not a printed book with a page count, which is the
// honest answer: an ebook has no spine.
export function spineWidthInches(format: EditionFormat): number {
  if (format.kind !== 'print') return 0
  let pages = Number.parseInt((format.page_count ?? '').trim(), 10)
  if (!Number.isFinite(pages) || pages <= 0) return 0
  if (!/^\d+$/.test((format.page_count ?? '').trim())) return 0
  if (pages % 2 !== 0) pages++
  if (hardcoverSpineRequiresTemplate(format)) return 0
  const interior = (format.interior ?? '').toLowerCase()
  const caliper = interior.includes('color') || interior.includes('colour') ? 0.002347 : spinePerPage(format.paper_stock ?? '')
  return pages * caliper
}

export function hardcoverSpineRequiresTemplate(format: EditionFormat): boolean {
  const binding = normalizeSpec(format.binding ?? '')
  return binding === 'caselaminate' || binding === 'clothwithjacket' || (format.format ?? '').toLowerCase().includes('hardcover')
}

export function spineWidthLabel(format: EditionFormat): string {
  const width = spineWidthInches(format)
  return width > 0 ? `${width.toFixed(3)} in` : ''
}

function normalizeSpec(value: string): string {
  return value.toLowerCase().replace(/[ \t]/g, '')
}

// ── The copyright page ─────────────────────────────────────────────────────

// priorYears is the copyright years of the editions this one supersedes,
// oldest first. The walk stops at an edition it has already seen, so a record
// that names itself as its own predecessor yields a finite list rather than
// hanging the screen.
export function priorYears(index: EditionIndex | undefined, editionID: string): string[] {
  if (!index) return []
  const seen = new Set<string>([editionID])
  const years: string[] = []
  let current = index.editions.find(e => e.id === editionID)
  while (current && (current.previous_edition_id ?? '').trim()) {
    const previousID = (current.previous_edition_id ?? '').trim()
    if (seen.has(previousID)) break
    seen.add(previousID)
    current = index.editions.find(e => e.id === previousID)
    if (!current) break
    const year = current.year?.trim()
    if (year) years.unshift(year)
  }
  return years
}

// copyrightLines is the Go generator, line for line. A line whose substance is
// missing is left out rather than printed empty: a book with no ISBN yet gets
// a page without an ISBN line, not one reading 'ISBN  (paperback)'.
export function copyrightLines(
  meta: Partial<Metadata>, edition: Edition, format: EditionFormat, prior: string[],
): string[] {
  const head: string[] = []
  const title = (meta.title ?? '').trim()
  if (title) head.push(title)
  const holder = (meta.copyright_holder ?? '').trim() || (meta.author ?? '').trim()
  if (holder) {
    const years = copyrightYears(prior, edition.year ?? '')
    if (years) head.push(`Copyright © ${years} by ${holder}`)
  }
  head.push(rightsSentence(format))

  const body: string[] = []
  const statement = editionStatement(edition, format)
  if (statement) body.push(statement)
  const imprint = copyrightImprint(meta, format)
  if (imprint) body.push(`Published by ${imprint}`)
  const isbn = (format.isbn13 ?? '').trim()
  if (isbn) body.push(`ISBN ${isbn} (${(format.format ?? '').trim().toLowerCase()})`)
  const lccn = (format.lccn ?? '').trim()
  if (lccn) body.push(`Library of Congress Control Number: ${lccn}`)
  // A revision note belongs on a later edition. Printing 'Original release.'
  // under a first edition's own ISBN says nothing a reader did not know.
  const note = (edition.revision_note ?? '').trim()
  if (note && (edition.previous_edition_id ?? '').trim()) body.push(note)

  if (!head.length) return body
  if (!body.length) return head
  return [...head, '', ...body]
}

// The cumulative year list a copyright line carries: every year an edition of
// this book established, in order, repeats dropped. A second edition of 2030
// off a first of 2026 prints '2026, 2030'.
function copyrightYears(prior: string[], year: string): string {
  const seen = new Set<string>()
  const years: string[] = []
  for (const candidate of [...prior, year]) {
    const trimmed = (candidate ?? '').trim()
    if (!trimmed || seen.has(trimmed)) continue
    seen.add(trimmed)
    years.push(trimmed)
  }
  return years.join(', ')
}

// The record holds the rights notice as a phrase, because that is how a list
// of rights reads in a dropdown. The page needs it as a sentence.
function rightsSentence(format: EditionFormat): string {
  const notice = (format.rights_notice ?? '').trim() || 'All rights reserved'
  return notice.endsWith('.') ? notice : `${notice}.`
}

// The edition line with its month and year: 'First edition, April 2026'. The
// words come from the edition's own label, which is what a printed copyright
// page says; the format's edition statement is the longer title-page wording
// and stands in only when an edition has no label.
function editionStatement(edition: Edition, format: EditionFormat): string {
  const statement = (edition.label ?? '').trim() || (format.edition_statement ?? '').trim()
  if (!statement) return ''
  const when = monthAndYear(format.publication_date ?? '')
  return when ? `${statement}, ${when}` : statement
}

function copyrightImprint(meta: Partial<Metadata>, format: EditionFormat): string {
  for (const candidate of [format.imprint_of_record, meta.imprint, meta.publisher]) {
    const trimmed = (candidate ?? '').trim()
    if (trimmed) return trimmed
  }
  return ''
}

const MONTHS = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
]

// Dates are stored as an ISO day because that is what a date field yields. A
// page that is not sure of the month prints the year alone rather than
// guessing one.
function monthAndYear(date: string): string {
  const trimmed = date.trim()
  if (trimmed.length < 4 || !/^\d{4}/.test(trimmed)) return ''
  const year = trimmed.slice(0, 4)
  if (trimmed.length < 7 || trimmed[4] !== '-') return year
  const month = Number.parseInt(trimmed.slice(5, 7), 10)
  if (!Number.isFinite(month) || month < 1 || month > 12) return year
  return `${MONTHS[month - 1]} ${year}`
}

// A publication date is free text, because a contract that says "Spring 2027"
// is a real answer. An exported EPUB can only declare a date in the form the
// specification allows — a year, a year and month, or a calendar day — so a
// date it cannot state is left out of the file entirely, and the row says so
// rather than letting the author find out from a retailer's validator.
// The rule is the same one internal/export/edition.go applies.
export function publicationDateHint(date: string | undefined): string {
  const text = (date ?? '').trim()
  if (!text) return ''
  if (/^\d{4}(-\d{2}(-\d{2}([T ].*)?)?)?$/.test(text) && isoPartsValid(text)) return ''
  return 'An exported ebook can only declare YYYY, YYYY-MM or YYYY-MM-DD. This wording stays on the record but will not be in the file.'
}

function isoPartsValid(text: string): boolean {
  const month = text.length >= 7 ? Number.parseInt(text.slice(5, 7), 10) : 1
  const day = text.length >= 10 ? Number.parseInt(text.slice(8, 10), 10) : 1
  return month >= 1 && month <= 12 && day >= 1 && day <= 31
}

// ── The format panel ───────────────────────────────────────────────────────

export type EditionRowKind = 'text' | 'select' | 'static' | 'textarea'

export interface EditionRow {
  label: string
  kind: EditionRowKind
  // The field this row edits. A static row has none: it is calculated, which
  // is the interface saying 'derived, never stored'.
  field?: keyof EditionFormat
  value: string
  options?: string[]
  hint?: string
  placeholder?: string
  mono?: boolean
  // A registered ISBN is fixed. The row says so rather than quietly ignoring
  // what the author types into it.
  locked?: boolean
}

export interface EditionSection {
  label: string
  note: string
  rows: EditionRow[]
}

// An ISBN is fixed once it is registered: a published format's number cannot
// be edited, because the number is how every retailer and library already
// refers to that object. Changing it means a new edition record.
export function isbnLocked(format: EditionFormat): boolean {
  return (format.status ?? '').trim().toLowerCase() === 'published' && !!(format.isbn13 ?? '').trim()
}

const text = (v: string | undefined) => (v ?? '')

export function sectionsFor(edition: Edition, format: EditionFormat): EditionSection[] {
  const out: EditionSection[] = []

  out.push({
    label: 'Identity',
    note: 'Locked once the ISBN is registered.',
    rows: [
      {
        label: 'ISBN-13', kind: 'text', field: 'isbn13', value: text(format.isbn13), mono: true,
        placeholder: '978-…', locked: isbnLocked(format),
        hint: isbnLocked(format) ? 'Published. This number is fixed.' : undefined,
      },
      { label: 'Registration', kind: 'select', field: 'registration', value: text(format.registration), options: REGISTRATIONS },
      { label: 'Format', kind: 'select', field: 'format', value: text(format.format), options: FORMAT_WORDS },
      {
        label: 'Edition statement', kind: 'text', field: 'edition_statement', value: text(format.edition_statement),
        placeholder: edition.label, hint: 'Printed on the title and copyright pages.',
      },
    ],
  })

  if (format.kind === 'print') {
    out.push({
      label: 'Specification',
      // Read by an export of this edition, which is the first step of the
      // export wizard. Draftline lays out the interior, never the wrap.
      note: 'The trim and gutter an export of this edition uses. Draftline exports the interior, not the cover wrap.',
      rows: [
        { label: 'Trim size', kind: 'select', field: 'trim', value: text(format.trim), options: TRIMS },
        { label: 'Page count', kind: 'text', field: 'page_count', value: text(format.page_count), mono: true, hint: 'From the last typeset pass.' },
        {
          label: 'Spine width', kind: 'static', mono: true,
          value: hardcoverSpineRequiresTemplate(format)
            ? 'Use the printer’s hardcover template'
            : spineWidthLabel(format) ? `${spineWidthLabel(format)} — calculated` : 'Set a page count',
          hint: hardcoverSpineRequiresTemplate(format)
            ? 'Case wrap, hinge and board dimensions vary by printer.'
            : 'Page count × the printer’s paper caliper.',
        },
        { label: 'Paper stock', kind: 'select', field: 'paper_stock', value: text(format.paper_stock), options: PAPER_STOCKS },
        { label: 'Binding', kind: 'select', field: 'binding', value: text(format.binding), options: BINDINGS },
        { label: 'Bleed', kind: 'select', field: 'bleed', value: text(format.bleed), options: BLEEDS },
      ],
    })
  }

  if (format.kind === 'ebook') {
    const digits = normalizeISBN(format.isbn13 ?? '')
    out.push({
      label: 'Specification',
      note: 'What an export of this edition applies.',
      rows: [
        { label: 'EPUB version', kind: 'select', field: 'epub_version', value: text(format.epub_version), options: EPUB_VERSIONS },
        {
          label: 'Unique identifier', kind: 'static', mono: true,
          value: digits ? `urn:isbn:${digits}` : 'Set an ISBN',
          hint: 'What the package document declares when this edition is exported.',
        },
      ],
    })
  }

  out.push({
    label: 'Release',
    note: '',
    rows: [
      {
        label: 'Publication date', kind: 'text', field: 'publication_date',
        value: text(format.publication_date), mono: true, placeholder: 'YYYY-MM-DD',
        hint: publicationDateHint(format.publication_date),
      },
      { label: 'List price', kind: 'text', field: 'list_price', value: text(format.list_price), mono: true },
      { label: 'Status', kind: 'select', field: 'status', value: text(format.status), options: FORMAT_STATUSES },
      { label: 'Channels', kind: 'text', field: 'channels', value: text(format.channels), placeholder: 'Where this format is sold' },
      // Which words this ISBN prints. Exporting freezes the manuscript the
      // first time and reads the frozen text afterwards, so the format has to
      // be able to say what it stands for. One row, not a panel.
      {
        label: 'Text', kind: 'static', field: 'snapshot_id',
        value: format.snapshot_id ? 'frozen when this ISBN was first exported' : 'not frozen yet',
        hint: format.snapshot_id
          ? 'Exports of this ISBN print these words, however far the book moves on.'
          : 'The first export of this ISBN freezes the manuscript as it stands then.',
      },
    ],
  })

  return out
}

// advancedFor takes the whole index as well, because one of its rows names
// what this format supersedes, and that fact lives in the edition before this
// one rather than in either record passed alongside.
export function advancedFor(index: EditionIndex | undefined, edition: Edition, format: EditionFormat): EditionRow[] {
  const ten = derivedISBN10(format)
  const rows: EditionRow[] = [
    {
      label: 'ISBN-10', kind: 'static', mono: true,
      value: ten || (format.isbn13?.trim() ? 'None — a 979 ISBN has no ISBN-10' : '—'),
      hint: 'Calculated from the ISBN-13.',
    },
    { label: 'Imprint of record', kind: 'text', field: 'imprint_of_record', value: text(format.imprint_of_record) },
    { label: 'Territory rights', kind: 'select', field: 'territory_rights', value: text(format.territory_rights), options: TERRITORIES },
    { label: 'Rights notice', kind: 'select', field: 'rights_notice', value: text(format.rights_notice), options: RIGHTS_NOTICES },
    {
      label: 'LCCN', kind: 'text', field: 'lccn', value: text(format.lccn), mono: true,
      placeholder: 'Library of Congress number, if applied for',
    },
    { label: 'Previous edition', kind: 'static', value: supersedes(index, edition, format) },
  ]

  if (format.kind === 'print') {
    rows.push(
      { label: 'Interior', kind: 'select', field: 'interior', value: text(format.interior), options: INTERIORS },
      { label: 'Inside gutter', kind: 'text', field: 'gutter', value: text(format.gutter), mono: true, placeholder: '0.875 in' },
    )
  }
  if (format.kind === 'ebook') {
    rows.push(
      {
        label: 'Layout', kind: 'select', field: 'layout', value: text(format.layout), options: LAYOUTS,
        hint: 'Draftline exports a reflowable package. A fixed-layout file for this ISBN is made elsewhere.',
      },
      { label: 'Retailer ASIN', kind: 'text', field: 'asin', value: text(format.asin), mono: true },
      { label: 'DRM', kind: 'select', field: 'drm', value: text(format.drm), options: DRM_CHOICES },
    )
  }
  return rows
}

// What this format replaces: the same kind of format in the edition before
// this one, named by its ISBN because that is what a reader or a distributor
// looks it up by.
function supersedes(index: EditionIndex | undefined, edition: Edition, format: EditionFormat): string {
  const previousID = (edition.previous_edition_id ?? '').trim()
  if (!previousID || !index) return 'None — original release'
  const previous = index.editions.find(e => e.id === previousID)
  if (!previous) return 'None — original release'
  const match = previous.formats.find(f => f.format === format.format) ?? previous.formats.find(f => f.kind === format.kind)
  const isbn = match?.isbn13?.trim()
  return isbn ? `Supersedes ${isbn}` : `Supersedes ${previous.label || previousID}`
}

// The one-line summary under the panel heading.
export function formatSubtitle(edition: Edition, format: EditionFormat): string {
  const parts = [edition.year, format.imprint_of_record]
  if (format.kind === 'print') parts.push(format.trim)
  parts.push(format.channels)
  return parts.map(p => (p ?? '').trim()).filter(Boolean).join(' · ')
}

export function formatTitle(edition: Edition, format: EditionFormat): string {
  const word = (format.format ?? '').trim() || 'Format'
  const label = (edition.label ?? '').trim()
  return label ? `${word} — ${label}` : word
}

// ── Making records ─────────────────────────────────────────────────────────

export const emptyEditionIndex = (): EditionIndex => ({ version: 1, editions: [] })

const ORDINALS = [
  'First', 'Second', 'Third', 'Fourth', 'Fifth',
  'Sixth', 'Seventh', 'Eighth', 'Ninth', 'Tenth',
]

// Identifiers are derived from what the record already holds rather than from
// a clock or a random source, so that the same sequence of actions produces
// the same file and a test can name what it made.
function nextID(taken: Set<string>, stem: string): string {
  for (let n = 1; ; n++) {
    const candidate = `${stem}-${n}`
    if (!taken.has(candidate)) return candidate
  }
}

const editionIDs = (index: EditionIndex) => new Set(index.editions.map(e => e.id))
const formatIDs = (index: EditionIndex) => new Set(index.editions.flatMap(e => e.formats.map(f => f.id)))

export function newEdition(index: EditionIndex, year: string): Edition {
  const ordinal = ORDINALS[index.editions.length] ?? `Edition ${index.editions.length + 1}`
  const previous = index.editions[index.editions.length - 1]
  return {
    id: nextID(editionIDs(index), 'ed'),
    label: `${ordinal} edition`,
    year,
    status: 'Draft',
    previous_edition_id: previous?.id,
    formats: [],
  }
}

// A new format starts with the defaults its kind needs to compute anything at
// all: a paperback with no stock has no spine width, and 'blank' is not a
// safer answer than 'the commonest one'.
export function newFormat(index: EditionIndex, kind: EditionKind, word?: string): EditionFormat {
  const base: EditionFormat = {
    id: nextID(formatIDs(index), 'fmt'),
    kind,
    status: 'Draft',
    registration: 'Not yet assigned',
    rights_notice: 'All rights reserved',
    territory_rights: 'World',
  }
  if (kind === 'print') {
    // The word the writer picked, so a Hardcover does not arrive labelled
    // Paperback. The rest are starting points they can change.
    return { ...base, format: word || 'Paperback', trim: '6 × 9 in (trade)', paper_stock: 'Cream, 55#', binding: word === 'Hardcover' ? 'Case laminate' : 'Perfect bound', bleed: 'No bleed', interior: 'Black and white' }
  }
  if (kind === 'ebook') {
    return { ...base, format: word || 'eBook', epub_version: 'EPUB 3.3', layout: 'Reflowable', drm: 'None' }
  }
  return { ...base, format: 'Audiobook' }
}

// Duplicating an edition is how a second edition begins: the same formats,
// the same specification, and deliberately NOT the same ISBNs. An ISBN
// identifies one object; a reissue with a new trim or a new cover is a
// different object and needs its own number.
export function duplicateAsNewEdition(index: EditionIndex, editionID: string, year: string): EditionIndex {
  const source = index.editions.find(e => e.id === editionID)
  if (!source) return index
  const draft = newEdition(index, year)
  draft.previous_edition_id = source.id
  const taken = formatIDs(index)
  draft.formats = source.formats.map(format => {
    const id = nextID(taken, 'fmt')
    taken.add(id)
    const copy: EditionFormat = { ...format, id, status: 'Draft', registration: 'Not yet assigned' }
    delete copy.isbn13
    delete copy.asin
    delete copy.lccn
    delete copy.publication_date
    delete copy.snapshot_id
    delete copy.export_settings
    return copy
  })
  return { ...index, editions: [...index.editions, draft] }
}
