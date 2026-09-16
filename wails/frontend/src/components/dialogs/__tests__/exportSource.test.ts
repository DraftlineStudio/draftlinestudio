// Export step one: the registered editions, the prefill, and the write-back.
//
// Everything here is the wizard's arithmetic rather than its markup — which
// trim '148 × 210 mm (A5)' is, what a gutter of '0.875 in' becomes, what
// changed since the record prefilled it, and what a save-back actually writes.

import { describe, expect, it } from 'vitest'
import type { Edition, EditionFormat, EditionIndex } from '../../../types/draftline'
import {
  bleedFromRecord, bleedLabel, customTrimError, defaultWizardOptions, editionCards,
  exportSourceSummary, findFormat, fixedLayoutNote, inchesText, isbnRegistrationError,
  outputFormatFor, patchChangesFormat, prefillChanges, printPrefill, readingCopyFor,
  registrableKind, registrationPatch, SOURCE_FOOTNOTE,
  trimFromRecord, trimRecordWords, wizardOptionsForFormat, writeBackPatch,
  type PrintPDFOptions,
} from '../exportSource'

// An invented book's publishing record. The names and the imprint are made up;
// the ISBNs carry correct check digits so they behave like real ones.
function index(): EditionIndex {
  const ebook: EditionFormat = {
    id: 'fmt-ebook', kind: 'ebook', format: 'eBook',
    isbn13: '978-1-9471345-1-5', epub_version: 'EPUB 3.3', layout: 'Reflowable',
    status: 'Published', publication_date: '2026-04-14',
  }
  const paperback: EditionFormat = {
    id: 'fmt-paper', kind: 'print', format: 'Paperback',
    isbn13: '978-1-9471345-2-2', trim: '6 × 9 in (trade)', page_count: '412',
    paper_stock: 'Cream, 55#', binding: 'Perfect bound', bleed: 'No bleed',
    gutter: '0.9 in', status: 'Registered',
  }
  const unnumbered: EditionFormat = {
    id: 'fmt-hard', kind: 'print', format: 'Hardcover', page_count: '428',
    binding: 'Case laminate', status: 'Draft',
  }
  const audio: EditionFormat = { id: 'fmt-audio', kind: 'audio', format: 'Audiobook', isbn13: '978-1-9471345-3-9' }
  const first: Edition = {
    id: 'ed-1', label: 'First edition', year: '2026', status: 'Published',
    formats: [ebook, paperback, unnumbered, audio],
  }
  return { version: 1, editions: [first] }
}

const paperbackOf = (idx: EditionIndex) => findFormat(idx, 'fmt-paper')!

describe('the registered-edition cards', () => {
  it('offers one card per format that has an ISBN and a file Draftline makes', () => {
    const cards = editionCards(index(), 'The Quiet Ledger')
    expect(cards.map(c => c.formatID)).toEqual(['fmt-ebook', 'fmt-paper'])
  })

  it('leaves out a format with no ISBN, because there is nothing to export it as', () => {
    const cards = editionCards(index(), 'The Quiet Ledger')
    expect(cards.some(c => c.formatID === 'fmt-hard')).toBe(false)
  })

  it('leaves out audio, which Draftline does not produce a file for', () => {
    const cards = editionCards(index(), 'The Quiet Ledger')
    expect(cards.some(c => c.formatID === 'fmt-audio')).toBe(false)
  })

  it('says what comes out of each one', () => {
    const cards = editionCards(index(), 'The Quiet Ledger')
    expect(cards[0].out).toBe('EPUB')
    expect(cards[1].out).toBe('Print PDF')
  })

  it('reads the specification a buyer would use to tell two paperbacks apart', () => {
    const cards = editionCards(index(), 'The Quiet Ledger')
    expect(cards[1].spec).toBe('6 × 9 in (trade) · 412 pp · perfect bound')
    expect(cards[0].spec).toBe('Reflowable EPUB 3.3 · linked contents')
  })

  it('badges a published format as published and everything else as ready', () => {
    const cards = editionCards(index(), 'The Quiet Ledger')
    expect([cards[0].badge, cards[0].badgeKind]).toEqual(['Published', 'ok'])
    expect([cards[1].badge, cards[1].badgeKind]).toEqual(['Template ready', 'accent'])
  })

  it('offers nothing at all for a book with no editions', () => {
    expect(editionCards(undefined, 'Untitled')).toEqual([])
    expect(editionCards({ version: 1, editions: [] }, 'Untitled')).toEqual([])
  })
})

describe('the sidebar summary', () => {
  it('names five things a printed edition fills in', () => {
    const summary = exportSourceSummary(index(), 'fmt-paper', 'The Quiet Ledger')!
    expect(summary.prefill.map(p => p.k)).toEqual(['Trim', 'Margins', 'Cover wrap', 'Copyright page', 'Identifier'])
    expect(summary.prefill[0].v).toBe('6 × 9 in (trade)')
    expect(summary.prefill[1].v).toBe('mirrored, 0.9 in gutter')
    expect(summary.prefill[4].v).toBe('978-1-9471345-2-2')
  })

  it('gives the spine width but does not claim to make a cover wrap', () => {
    const summary = exportSourceSummary(index(), 'fmt-paper', 'The Quiet Ledger')!
    expect(summary.prefill[2].v).toBe('1.037 in spine — interior only, no wrap')
  })

  it('names five different things an ebook fills in, identifier included', () => {
    const summary = exportSourceSummary(index(), 'fmt-ebook', 'The Quiet Ledger')!
    expect(summary.prefill.map(p => p.k)).toEqual(['Package', 'Cover', 'Identifier', 'Copyright page', 'Contents'])
    expect(summary.prefill[2].v).toBe('urn:isbn:9781947134515')
    expect(summary.prefill[1].v).toBe('none attached')
  })

  it('says that changing something will be offered back to the record', () => {
    const summary = exportSourceSummary(index(), 'fmt-ebook', 'The Quiet Ledger')!
    expect(summary.footnote).toBe(SOURCE_FOOTNOTE)
  })

  it('is nothing at all when no edition was chosen', () => {
    expect(exportSourceSummary(index(), '', 'The Quiet Ledger')).toBeNull()
    expect(exportSourceSummary(index(), 'fmt-gone', 'The Quiet Ledger')).toBeNull()
  })
})

describe('what the record prefills', () => {
  it('sets the page up at the trim the edition is bound at', () => {
    const { edition, format } = paperbackOf(index())
    const options = wizardOptionsForFormat(edition, format)
    expect(options.print.trimSize).toBe('6x9')
    expect(options.print.gutterMargin).toBe('0.9')
    expect(options.print.bleed).toBe('0')
  })

  it('carries the edition and format identity into every option set', () => {
    const { edition, format } = paperbackOf(index())
    const options = wizardOptionsForFormat(edition, format)
    for (const set of [options.shared, options.epub, options.pdf, options.print]) {
      expect(set.editionID).toBe('ed-1')
      expect(set.formatID).toBe('fmt-paper')
    }
  })

  it('reads the trims the Editions screen offers, millimetres included', () => {
    expect(trimFromRecord('5 × 8 in')).toEqual({ trimSize: '5x8', width: '5', height: '8' })
    expect(trimFromRecord('6 × 9 in (trade)')).toEqual({ trimSize: '6x9', width: '6', height: '9' })
    expect(trimFromRecord('148 × 210 mm (A5)')).toEqual({ trimSize: 'custom', width: '5.83', height: '8.27' })
    expect(trimFromRecord('7 × 10 in')).toEqual({ trimSize: 'custom', width: '7', height: '10' })
  })

  it('keeps its default rather than inventing a page from a trim it cannot read', () => {
    expect(trimFromRecord('')).toBeNull()
    expect(trimFromRecord('whatever the printer says')).toBeNull()
    expect(trimFromRecord('99 × 99 in')).toBeNull()
    const { edition, format } = paperbackOf(index())
    const options = wizardOptionsForFormat(edition, { ...format, trim: '99 × 99 in' })
    expect(options.print.trimSize).toBe('5.5x8.5')
  })

  it('reads a measurement however the author typed it', () => {
    expect(inchesText('0.875')).toBe('0.875')
    expect(inchesText('0.875 in')).toBe('0.875')
    expect(inchesText('0.9in')).toBe('0.9')
    expect(inchesText('')).toBe('')
    expect(inchesText('a bit')).toBe('')
  })

  it('turns the record’s bleed phrase into inches and back', () => {
    expect(bleedFromRecord('No bleed')).toBe('0')
    expect(bleedFromRecord('Bleed 0.125 in')).toBe('0.125')
    expect(bleedFromRecord('')).toBeNull()
    expect(bleedLabel('0')).toBe('No bleed')
    expect(bleedLabel('0.125')).toBe('Bleed 0.125 in')
  })

  it('leaves a print format alone when its record says nothing', () => {
    expect(printPrefill({ id: 'x', kind: 'print' })).toEqual({})
  })
})

describe('what a previous export left behind', () => {
  it('starts where the last export of this format finished', () => {
    const { edition, format } = paperbackOf(index())
    const stored = {
      ...format,
      export_settings: { 'print-pdf': { gutterMargin: '1.05', fontSize: 11, generateTOC: false } },
    }
    const options = wizardOptionsForFormat(edition, stored)
    expect(options.print.gutterMargin).toBe('1.05')
    expect(options.print.fontSize).toBe(11)
    expect(options.print.generateTOC).toBe(false)
  })

  it('ignores a stored value of the wrong type rather than laying out a page with it', () => {
    const { edition, format } = paperbackOf(index())
    const stored = {
      ...format,
      export_settings: { 'print-pdf': { gutterMargin: 42, fontSize: 'large', invented: true } },
    }
    const options = wizardOptionsForFormat(edition, stored)
    expect(options.print.gutterMargin).toBe('0.9')
    expect(options.print.fontSize).toBe(9)
    expect('invented' in options.print).toBe(false)
  })

  it('ignores an export_settings that is not an object at all', () => {
    const { edition, format } = paperbackOf(index())
    const stored = { ...format, export_settings: 'nonsense' as unknown as Record<string, unknown> }
    expect(wizardOptionsForFormat(edition, stored).print.gutterMargin).toBe('0.9')
  })
})

describe('offering the change back to the record', () => {
  it('offers nothing when nothing was changed', () => {
    const { edition, format } = paperbackOf(index())
    const options = wizardOptionsForFormat(edition, format)
    expect(prefillChanges(edition, format, options, 'print-pdf')).toEqual([])
  })

  it('names the gutter, and says what it was and what it became', () => {
    const { edition, format } = paperbackOf(index())
    const options = wizardOptionsForFormat(edition, format)
    options.print = { ...options.print, gutterMargin: '1.05' }
    const changes = prefillChanges(edition, format, options, 'print-pdf')
    expect(changes[0]).toEqual({ label: 'Inside gutter', from: '0.9 in', to: '1.05 in' })
  })

  it('names a changed trim in the words the Editions screen uses', () => {
    const { edition, format } = paperbackOf(index())
    const options = wizardOptionsForFormat(edition, format)
    options.print = { ...options.print, trimSize: '5.5x8.5' }
    const changes = prefillChanges(edition, format, options, 'print-pdf')
    expect(changes[0]).toEqual({ label: 'Trim size', from: '6 × 9 in (trade)', to: '5.5 × 8.5 in' })
  })

  it('still offers when only a design choice moved', () => {
    const { edition, format } = paperbackOf(index())
    const options = wizardOptionsForFormat(edition, format)
    options.print = { ...options.print, titlePageStyle: 'dramatic' }
    expect(prefillChanges(edition, format, options, 'print-pdf')).toHaveLength(1)
  })
})

describe('saving back', () => {
  it('writes the gutter onto the format record, not only into the settings', () => {
    const { edition, format } = paperbackOf(index())
    const options = wizardOptionsForFormat(edition, format)
    options.print = { ...options.print, gutterMargin: '1.05', trimSize: '5x8', bleed: '0.125' }
    const patch = writeBackPatch(options, 'print-pdf')
    expect(patch.gutter).toBe('1.05 in')
    expect(patch.trim).toBe('5 × 8 in')
    expect(patch.bleed).toBe('Bleed 0.125 in')
  })

  it('stores every option set, so reopening the wizard offers them rather than the defaults', () => {
    const { edition, format } = paperbackOf(index())
    const options = wizardOptionsForFormat(edition, format)
    options.print = { ...options.print, gutterMargin: '1.05' }
    const patch = writeBackPatch(options, 'print-pdf')
    const reopened = wizardOptionsForFormat(edition, { ...format, ...patch })
    expect(reopened.print.gutterMargin).toBe('1.05')
  })

  it('leaves an ebook’s record fields alone: an ebook has no trim to change', () => {
    const { edition, format } = findFormat(index(), 'fmt-ebook')!
    const patch = writeBackPatch(wizardOptionsForFormat(edition, format), 'epub')
    expect(patch.trim).toBeUndefined()
    expect(patch.gutter).toBeUndefined()
    expect(patch.export_settings).toBeTruthy()
  })
})

describe('the custom trim, bounded at both ends', () => {
  const withTrim = (width: string, height: string): PrintPDFOptions =>
    ({ ...defaultWizardOptions().print, trimSize: 'custom', customWidth: width, customHeight: height })

  it('accepts a page anybody prints', () => {
    expect(customTrimError(withTrim('6', '9'))).toBe('')
    expect(customTrimError(withTrim('3', '12'))).toBe('')
  })

  it('refuses a ninety-nine inch page and says what a usable one is', () => {
    const message = customTrimError(withTrim('99', '9'))
    expect(message).toContain('99')
    expect(message).toContain('3')
    expect(message).toContain('12')
  })

  it('refuses a page too small to bind, an empty field, and a word', () => {
    expect(customTrimError(withTrim('0.2', '9'))).not.toBe('')
    expect(customTrimError(withTrim('', '9'))).not.toBe('')
    expect(customTrimError(withTrim('six', '9'))).not.toBe('')
    expect(customTrimError(withTrim('6', '120'))).not.toBe('')
  })

  it('never refuses one of Draftline’s own named trims', () => {
    const named = { ...defaultWizardOptions().print, trimSize: '6x9' as const, customWidth: '99', customHeight: '99' }
    expect(customTrimError(named)).toBe('')
  })

  it('writes a custom trim back in inches', () => {
    expect(trimRecordWords(withTrim('7', '10'))).toBe('7 × 10 in')
  })
})

describe('registering a from-scratch export', () => {
  it('offers a number only for the two exports that are saleable objects', () => {
    expect(registrableKind('epub')).toBe('ebook')
    expect(registrableKind('print-pdf')).toBe('print')
    expect(registrableKind('docx')).toBeNull()
    expect(registrableKind('pdf')).toBeNull()
  })

  it('refuses a number whose check digit does not match', () => {
    expect(isbnRegistrationError('978-1-9471345-1-4')).toContain('check digit')
    expect(isbnRegistrationError('12345')).toContain('10 or 13 digits')
    expect(isbnRegistrationError('')).toContain('skip')
    expect(isbnRegistrationError('978-1-9471345-1-5')).toBe('')
  })

  it('refuses a number the record already carries, however it is punctuated', () => {
    const idx = index()
    expect(isbnRegistrationError('978-1-9471345-1-5', idx)).toContain('already on')
    expect(isbnRegistrationError('9781947134515', idx)).toContain('already on')
    expect(isbnRegistrationError('978-1-9471345-1-5', idx)).toContain('ebook')
    // A number nowhere in the record is still fine.
    expect(isbnRegistrationError('978-1-9471345-4-6', idx)).toBe('')
  })

  it('makes a record that describes the file that exists, and claims nothing else', () => {
    const options = defaultWizardOptions()
    options.print = { ...options.print, trimSize: '6x9', gutterMargin: '0.95' }
    const patch = registrationPatch('978-1-9471345-1-5', 'print-pdf', options, { imprint: 'Bellwether House' })
    expect(patch.isbn13).toBe('978-1-9471345-1-5')
    expect(patch.format).toBe('Paperback')
    // A file exists and a number is attached to it. Neither of those is a book
    // being on sale, and where the number came from only the author knows.
    expect(patch.status).toBe('Registered')
    expect(patch.registration).toBeUndefined()
    expect(patch.trim).toBe('6 × 9 in (trade)')
    expect(patch.gutter).toBe('0.95 in')
    expect(patch.imprint_of_record).toBe('Bellwether House')
    expect(patch.publication_date).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })

  it('registers an EPUB as an ebook', () => {
    const patch = registrationPatch('978-1-9471345-1-5', 'epub', defaultWizardOptions(), {})
    expect(patch.format).toBe('eBook')
    expect(patch.trim).toBeUndefined()
  })
})

describe('which file a format produces', () => {
  it('is one file per kind, and nothing for audio', () => {
    expect(outputFormatFor({ id: 'a', kind: 'ebook' })).toBe('epub')
    expect(outputFormatFor({ id: 'b', kind: 'print' })).toBe('print-pdf')
    expect(outputFormatFor({ id: 'c', kind: 'audio' })).toBeNull()
  })

  it('also offers a reading copy of anything it makes a file for', () => {
    expect(readingCopyFor({ id: 'a', kind: 'ebook' })).toBe('pdf')
    expect(readingCopyFor({ id: 'b', kind: 'print' })).toBe('pdf')
    expect(readingCopyFor({ id: 'c', kind: 'audio' })).toBeNull()
  })

  it('puts the reading copy on every card, so the cover page is reachable', () => {
    const cards = editionCards(index(), 'The Quiet Ledger')
    expect(cards.map(c => c.altOutput)).toEqual(['pdf', 'pdf'])
    expect(cards[0].altLabel).toContain('Reading copy')
  })
})

// ── What the screen may and may not promise ────────────────────────────────

describe('a record that says fixed layout', () => {
  const fixed = (): EditionIndex => {
    const idx = index()
    idx.editions[0].formats[0] = { ...idx.editions[0].formats[0], layout: 'Fixed layout' }
    return idx
  }

  it('is named as a caveat rather than repeated as a promise', () => {
    expect(fixedLayoutNote({ id: 'a', kind: 'ebook', layout: 'Fixed layout' })).toContain('reflowable')
    expect(fixedLayoutNote({ id: 'a', kind: 'ebook', layout: 'Reflowable' })).toBe('')
    expect(fixedLayoutNote({ id: 'a', kind: 'ebook' })).toBe('')
  })

  it('never makes the card claim a fixed-layout export', () => {
    const card = editionCards(fixed(), 'The Quiet Ledger')[0]
    expect(card.spec).toBe('Reflowable EPUB 3.3 · linked contents')
    expect(card.spec).not.toContain('Fixed')
  })

  it('makes the sidebar say what the file will actually be', () => {
    const summary = exportSourceSummary(fixed(), 'fmt-ebook', 'The Quiet Ledger')!
    expect(summary.prefill.find(line => line.k === 'Package')!.v).toBe('EPUB 3.3, reflowable')
    expect(summary.note).toContain('reflowable')
    const plain = exportSourceSummary(index(), 'fmt-ebook', 'The Quiet Ledger')!
    expect(plain.note).toBe('')
  })
})

// ── The record's own words are not rewritten by an export ──────────────────

describe('a no-change export', () => {
  // The A5 case is the one that bites: the wizard reads 148 × 210 mm as 5.83 by
  // 8.27 inches and has no way to spell millimetres again, so rewriting an
  // untouched trim would change a printer's specification behind the author.
  const a5 = (): EditionIndex => {
    const idx = index()
    idx.editions[0].formats[1] = { ...idx.editions[0].formats[1], trim: '148 × 210 mm (A5)' }
    return idx
  }

  it('reports nothing changed', () => {
    const idx = a5()
    const { edition, format } = paperbackOf(idx)
    const options = wizardOptionsForFormat(edition, format)
    expect(prefillChanges(edition, format, options, 'print-pdf')).toEqual([])
  })

  it('leaves the record’s trim, gutter and bleed exactly as the record spells them', () => {
    const idx = a5()
    const { edition, format } = paperbackOf(idx)
    const options = wizardOptionsForFormat(edition, format)
    const patch = writeBackPatch(options, 'print-pdf', wizardOptionsForFormat(edition, format))
    expect(patch.trim).toBeUndefined()
    expect(patch.gutter).toBeUndefined()
    expect(patch.bleed).toBeUndefined()
    expect(Object.keys(patch)).toEqual(['export_settings'])
  })

  it('writes a trim back only once the author has actually moved it', () => {
    const idx = a5()
    const { edition, format } = paperbackOf(idx)
    const prefilled = wizardOptionsForFormat(edition, format)
    const options = { ...prefilled, print: { ...prefilled.print, trimSize: '6x9' as const, pageSize: '6x9' as const } }
    const patch = writeBackPatch(options, 'print-pdf', prefilled)
    expect(patch.trim).toBe('6 × 9 in (trade)')
    expect(patch.gutter).toBeUndefined()
  })

  it('does not dirty the book when the settings already stored are the settings used', () => {
    const idx = index()
    const { edition, format } = paperbackOf(idx)
    const options = wizardOptionsForFormat(edition, format)
    const patch = writeBackPatch(options, 'print-pdf', options)
    expect(patchChangesFormat(format, patch)).toBe(true)
    const stored = { ...format, ...patch }
    expect(patchChangesFormat(stored, patch)).toBe(false)
  })
})
