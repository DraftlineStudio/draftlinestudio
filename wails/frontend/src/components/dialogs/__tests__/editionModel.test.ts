// The Editions half of the Book & Editions screen.
//
// The copyright block is checked against internal/types/testdata/
// copyright_cases.json, the same table internal/types/copyright_test.go reads.
// That is the only thing keeping the page an author watches while typing and
// the page printed into the exported book from drifting apart, so the file is
// read from disk rather than copied in here.

import { describe, expect, it } from 'vitest'
// The shared table of worked examples, read from the Go side of the tree so
// there is exactly one copy of it. internal/types/copyright_test.go reads the
// same file.
import copyrightFixture from '../../../../../internal/types/testdata/copyright_cases.json?raw'
import type { Edition, EditionFormat, EditionIndex, Metadata } from '../../../types/draftline'
import {
  advancedFor, copyrightLines, derivedISBN10, duplicateAsNewEdition, editionBadge, emptyEditionIndex,
  formatBadge, isbnLocked, kindDot, newEdition, newFormat, priorYears, sectionsFor, spineWidthInches,
  spineWidthLabel, statusBadgeKind,
} from '../editionModel'

interface CopyrightCase {
  name: string
  metadata: Partial<Metadata>
  edition: Partial<Edition>
  format: Partial<EditionFormat>
  prior_years: string[]
  lines: string[]
}

const asEdition = (e: Partial<Edition>): Edition => ({ id: '', label: '', year: '', status: '', formats: [], ...e })
const asFormat = (f: Partial<EditionFormat>): EditionFormat => ({ id: '', kind: 'print', ...f })

describe('the generated copyright page agrees with the Go generator', () => {
  const cases = (JSON.parse(copyrightFixture) as { cases: CopyrightCase[] }).cases

  it('has the worked examples to check against', () => {
    expect(cases.length).toBeGreaterThan(0)
  })

  for (const tc of cases) {
    it(tc.name, () => {
      expect(copyrightLines(tc.metadata, asEdition(tc.edition), asFormat(tc.format), tc.prior_years))
        .toEqual(tc.lines)
    })
  }
})

describe('copyright years', () => {
  const index: EditionIndex = {
    version: 1,
    editions: [
      { id: 'a', label: 'First edition', year: '2026', status: 'Published', formats: [] },
      { id: 'b', label: 'Second edition', year: '2030', status: 'Published', previous_edition_id: 'a', formats: [] },
      { id: 'c', label: 'Third edition', year: '2033', status: 'Draft', previous_edition_id: 'b', formats: [] },
    ],
  }

  it('accumulate down the chain of editions, oldest first', () => {
    expect(priorYears(index, 'c')).toEqual(['2026', '2030'])
    expect(priorYears(index, 'a')).toEqual([])
  })

  it('never repeat a year two editions share', () => {
    const reissue: EditionIndex = {
      version: 1,
      editions: [
        { id: 'a', label: 'First edition', year: '2026', status: 'Published', formats: [] },
        { id: 'b', label: 'Second edition', year: '2026', status: 'Published', previous_edition_id: 'a', formats: [] },
      ],
    }
    const lines = copyrightLines(
      { title: 'Harbour Lights', author: 'Ines Marrow', publisher: 'Windlass and Co.' },
      reissue.editions[1],
      asFormat({ id: 'f', kind: 'print', format: 'Paperback', publication_date: '2026-11-02' }),
      priorYears(reissue, 'b'),
    )
    expect(lines[1]).toBe('Copyright © 2026 by Ines Marrow')
  })

  it('stop when a record names itself as its own predecessor', () => {
    const looped: EditionIndex = {
      version: 1,
      editions: [
        { id: 'a', label: 'A', year: '2026', status: '', previous_edition_id: 'b', formats: [] },
        { id: 'b', label: 'B', year: '2027', status: '', previous_edition_id: 'a', formats: [] },
      ],
    }
    expect(priorYears(looped, 'a')).toEqual(['2027'])
  })
})

// ── Derivations ────────────────────────────────────────────────────────────

describe('the derived ISBN-10', () => {
  it('follows a 978 number and answers nothing for a 979', () => {
    expect(derivedISBN10({ id: 'f', kind: 'print', isbn13: '978-1-9471345-1-5' })).toBe('1-94713451-5')
    expect(derivedISBN10({ id: 'f', kind: 'print', isbn13: '978-1-9471345-1-5' }, false)).toBe('1947134515')
    expect(derivedISBN10({ id: 'f', kind: 'ebook', isbn13: '979-8-1234567-1-2' })).toBe('')
  })

  it('answers nothing for a number with a mistyped check digit', () => {
    expect(derivedISBN10({ id: 'f', kind: 'print', isbn13: '978-1-9471345-1-6' })).toBe('')
    expect(derivedISBN10({ id: 'f', kind: 'print' })).toBe('')
  })
})

describe('spine width', () => {
  it('matches the stock and binding table to three decimals', () => {
    // 412 leaves of cream 55# at 0.0025 in, plus 0.007 in of glued cover.
    const paperback: EditionFormat = {
      id: 'pb', kind: 'print', page_count: '412', paper_stock: 'Cream, 55#', binding: 'Perfect bound',
    }
    expect(spineWidthLabel(paperback)).toBe('1.037 in')

    // 428 leaves of white 60# at 0.002252 in, plus 0.24 in of board.
    const hardcover: EditionFormat = {
      id: 'hc', kind: 'print', page_count: '428', paper_stock: 'White, 60#', binding: 'Case laminate',
    }
    expect(spineWidthInches(hardcover)).toBeCloseTo(1.203856, 9)
    expect(spineWidthLabel(hardcover)).toBe('1.204 in')
  })

  it('gives nothing a spine that is not a printed book with a page count', () => {
    expect(spineWidthLabel({ id: 'e', kind: 'ebook', page_count: '412', paper_stock: 'Cream, 55#' })).toBe('')
    expect(spineWidthLabel({ id: 'p', kind: 'print', page_count: '' })).toBe('')
    expect(spineWidthLabel({ id: 'p', kind: 'print', page_count: 'four hundred' })).toBe('')
    expect(spineWidthLabel({ id: 'p', kind: 'print', page_count: '0' })).toBe('')
    expect(spineWidthLabel({ id: 'p', kind: 'print', page_count: '-40' })).toBe('')
  })

  it('falls back to the commonest stock rather than to zero', () => {
    expect(spineWidthLabel({
      id: 'p', kind: 'print', page_count: '300', paper_stock: 'Recycled, 70#', binding: 'Saddle stitch',
    })).toBe('0.757 in')
  })
})

// ── The panel ──────────────────────────────────────────────────────────────

const printFormat: EditionFormat = {
  id: 'ed-1-pb', kind: 'print', format: 'Paperback', isbn13: '978-1-9471345-1-5',
  page_count: '412', paper_stock: 'Cream, 55#', binding: 'Perfect bound', status: 'Published',
}
const ebookFormat: EditionFormat = {
  id: 'ed-1-eb', kind: 'ebook', format: 'eBook', isbn13: '979-8-1234567-1-2', status: 'Registered',
}
const firstEdition: Edition = {
  id: 'ed-1', label: 'First edition', year: '2026', status: 'Published',
  formats: [ebookFormat, printFormat],
}

describe('the format panel', () => {
  it('gives a printed book a specification a printer needs and an ebook one it does not', () => {
    const print = sectionsFor(firstEdition, printFormat)
    const spec = print.find(s => s.label === 'Specification')
    expect(spec?.rows.map(r => r.label)).toEqual([
      'Trim size', 'Page count', 'Spine width', 'Paper stock', 'Binding', 'Bleed',
    ])
    expect(spec?.rows.find(r => r.label === 'Spine width')?.value).toBe('1.037 in — calculated')

    const ebook = sectionsFor(firstEdition, ebookFormat).find(s => s.label === 'Specification')
    expect(ebook?.rows.map(r => r.label)).toEqual(['EPUB version', 'Unique identifier'])
    expect(ebook?.rows[1].value).toBe('urn:isbn:9798123456712')
  })

  it('draws a calculated value as static text, with no field to type into', () => {
    const rows = sectionsFor(firstEdition, printFormat).flatMap(s => s.rows)
    const spine = rows.find(r => r.label === 'Spine width')
    expect(spine?.kind).toBe('static')
    expect(spine?.field).toBeUndefined()

    const ten = advancedFor(undefined, firstEdition, printFormat).find(r => r.label === 'ISBN-10')
    expect(ten?.kind).toBe('static')
    expect(ten?.field).toBeUndefined()
    expect(ten?.value).toBe('1-94713451-5')
  })

  it('says plainly that a 979 number has no ISBN-10', () => {
    const ten = advancedFor(undefined, firstEdition, ebookFormat).find(r => r.label === 'ISBN-10')
    expect(ten?.value).toBe('None — a 979 ISBN has no ISBN-10')
  })

  it('locks a published ISBN and leaves an unpublished one editable', () => {
    expect(isbnLocked(printFormat)).toBe(true)
    expect(isbnLocked(ebookFormat)).toBe(false)
    const locked = sectionsFor(firstEdition, printFormat)[0].rows[0]
    expect([locked.label, locked.locked]).toEqual(['ISBN-13', true])
    expect(locked.hint).toBe('Published. This number is fixed.')
    // A registered-but-unpublished number is still the author's to correct.
    expect(sectionsFor(firstEdition, ebookFormat)[0].rows[0].locked).toBe(false)
  })

  it('names what a later edition supersedes, by the number a reader looks it up by', () => {
    const second: Edition = {
      id: 'ed-2', label: 'Second edition', year: '2030', status: 'Draft', previous_edition_id: 'ed-1',
      formats: [{ id: 'ed-2-pb', kind: 'print', format: 'Paperback' }],
    }
    const index: EditionIndex = { version: 1, editions: [firstEdition, second] }
    const row = (edition: Edition, format: EditionFormat) =>
      advancedFor(index, edition, format).find(r => r.label === 'Previous edition')?.value

    expect(row(second, second.formats[0])).toBe('Supersedes 978-1-9471345-1-5')
    expect(row(firstEdition, printFormat)).toBe('None — original release')
  })

  it('gives print and ebook their own advanced rows', () => {
    const print = advancedFor(undefined, firstEdition, printFormat).map(r => r.label)
    expect(print).toContain('Interior')
    expect(print).toContain('Inside gutter')
    expect(print).not.toContain('DRM')

    const ebook = advancedFor(undefined, firstEdition, ebookFormat).map(r => r.label)
    expect(ebook).toContain('Layout')
    expect(ebook).toContain('DRM')
    expect(ebook).not.toContain('Interior')
  })
})

describe('badges and kind colours', () => {
  it('map a status to the kind of state it is', () => {
    expect(statusBadgeKind('Published')).toBe('ok')
    expect(statusBadgeKind('Registered')).toBe('accent')
    expect(statusBadgeKind('In progress')).toBe('warn')
    expect(statusBadgeKind('Out of print')).toBe('warn')
    expect(statusBadgeKind('Draft')).toBe('neutral')
    expect(statusBadgeKind('')).toBe('neutral')
  })

  it('call an unset status a draft rather than showing nothing', () => {
    expect(formatBadge({ id: 'f', kind: 'print' })).toEqual({ label: 'Draft', kind: 'neutral' })
    expect(editionBadge({ id: 'e', label: '', year: '', status: '', formats: [] }))
      .toEqual({ label: 'Draft', kind: 'neutral' })
  })

  it('colour a format row by kind, and never leave an unknown kind colourless', () => {
    expect(kindDot('ebook')).toBe('var(--section-front)')
    expect(kindDot('print')).toBe('var(--section-body)')
    expect(kindDot('audio')).toBe('var(--section-back)')
    expect(kindDot('hologram')).toBe('var(--text-muted)')
  })
})

// ── Making records ─────────────────────────────────────────────────────────

describe('adding editions and formats', () => {
  it('names each edition in order and links it to the one before', () => {
    let index = emptyEditionIndex()
    const first = newEdition(index, '2026')
    expect([first.id, first.label, first.previous_edition_id]).toEqual(['ed-1', 'First edition', undefined])

    index = { ...index, editions: [first] }
    const second = newEdition(index, '2030')
    expect([second.id, second.label, second.previous_edition_id]).toEqual(['ed-2', 'Second edition', 'ed-1'])
  })

  it('gives a new format the defaults its kind needs to compute anything', () => {
    const index = emptyEditionIndex()
    const print = newFormat(index, 'print')
    expect(print.format).toBe('Paperback')
    expect(spineWidthLabel({ ...print, page_count: '412' })).toBe('1.037 in')

    const ebook = newFormat(index, 'ebook')
    expect([ebook.format, ebook.epub_version, ebook.layout]).toEqual(['eBook', 'EPUB 3.3', 'Reflowable'])

    // Neither starts out claiming to be registered or published.
    expect([print.status, print.registration]).toEqual(['Draft', 'Not yet assigned'])
  })

  it('never issues an identifier already in the record', () => {
    const index: EditionIndex = {
      version: 1,
      editions: [{ id: 'ed-1', label: 'First edition', year: '2026', status: '', formats: [{ id: 'fmt-1', kind: 'print' }] }],
    }
    expect(newEdition(index, '2030').id).toBe('ed-2')
    expect(newFormat(index, 'ebook').id).toBe('fmt-2')
  })
})

describe('duplicating an edition', () => {
  const index: EditionIndex = {
    version: 1,
    editions: [{
      id: 'ed-1', label: 'First edition', year: '2026', status: 'Published',
      formats: [
        { id: 'fmt-1', kind: 'print', format: 'Paperback', isbn13: '978-1-9471345-1-5', trim: '6 × 9 in (trade)', page_count: '412', status: 'Published', lccn: '2026901447', publication_date: '2026-04-14' },
        { id: 'fmt-2', kind: 'ebook', format: 'eBook', isbn13: '978-1-9471345-0-8', asin: 'B0INVENTED1', status: 'Published' },
      ],
    }],
  }

  it('carries the specification over and deliberately leaves the ISBNs behind', () => {
    const next = duplicateAsNewEdition(index, 'ed-1', '2030')
    expect(next.editions).toHaveLength(2)
    const copy = next.editions[1]
    expect([copy.label, copy.year, copy.previous_edition_id]).toEqual(['Second edition', '2030', 'ed-1'])
    expect(copy.formats.map(f => f.format)).toEqual(['Paperback', 'eBook'])
    expect(copy.formats[0].trim).toBe('6 × 9 in (trade)')
    expect(copy.formats[0].page_count).toBe('412')

    // An ISBN identifies one object. A reissue is a different object.
    for (const format of copy.formats) {
      expect(format.isbn13).toBeUndefined()
      expect(format.status).toBe('Draft')
      expect(format.registration).toBe('Not yet assigned')
    }
    expect(copy.formats[0].lccn).toBeUndefined()
    expect(copy.formats[0].publication_date).toBeUndefined()
    expect(copy.formats[1].asin).toBeUndefined()
  })

  it('leaves the edition it copied untouched, ISBNs and all', () => {
    const next = duplicateAsNewEdition(index, 'ed-1', '2030')
    expect(next.editions[0]).toEqual(index.editions[0])
    expect(next.editions[0].formats[0].isbn13).toBe('978-1-9471345-1-5')
    // and gives the copy its own identifiers
    expect(next.editions[1].formats.map(f => f.id)).toEqual(['fmt-3', 'fmt-4'])
  })

  it('does nothing when asked to copy an edition that is not there', () => {
    expect(duplicateAsNewEdition(index, 'ed-9', '2030')).toBe(index)
  })
})
