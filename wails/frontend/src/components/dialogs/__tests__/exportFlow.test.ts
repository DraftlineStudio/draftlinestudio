import { describe, expect, it } from 'vitest'
import type { Edition, EditionFormat } from '../../../types/draftline'
import {
  artworkRows, bundleName, bundleRequest, editionItems, exportFormatFor, fileCount,
  optionGroupFor, outputForRecord, previewFor, readField, reviewRows, scratchItems,
  settingGroups, stamped, stepSummary, stepsFor, writeField,
} from '../exportFlow'
import { defaultWizardOptions } from '../exportSource'

const format = (patch: Partial<EditionFormat>): EditionFormat =>
  ({ id: 'f1', kind: 'print', format: 'Paperback', ...patch })

const edition = (formats: EditionFormat[]): Edition =>
  ({ id: 'ed1', label: 'First edition', year: '2026', status: 'Published', formats })

describe('what a registered format produces', () => {
  it('tells a hardcover from a paperback by the word on the record', () => {
    expect(outputForRecord(format({ format: 'Paperback' }))).toBe('print-pdf')
    expect(outputForRecord(format({ format: 'Hardcover' }))).toBe('hc')
  })

  it('reads an ebook and an audiobook off the kind', () => {
    expect(outputForRecord(format({ kind: 'ebook', format: 'eBook' }))).toBe('epub')
    expect(outputForRecord(format({ kind: 'audio', format: 'Audiobook' }))).toBe('audio')
  })

  it('sends a hardcover and an audiobook to the exporters that exist', () => {
    expect(exportFormatFor('hc')).toBe('print-pdf')
    expect(exportFormatFor('audio')).toBe('pdf')
    expect(exportFormatFor('epub')).toBe('epub')
  })
})

describe('the formats on offer', () => {
  it('offers every registered format, with or without a number', () => {
    const items = editionItems(edition([
      format({ id: 'a', isbn13: '978-1-7371829-1-1' }),
      format({ id: 'b', kind: 'ebook', format: 'eBook', isbn13: '' }),
    ]))
    expect(items.map(i => i.id)).toEqual(['a', 'b'])
    // The badge says the record is incomplete; nothing is withheld for it.
    expect(items[1].badge).toBe('No ISBN')
    expect(items[0].badge).toBe('')
  })

  it('offers five files to an export that belongs to no edition', () => {
    expect(scratchItems().map(i => i.output)).toEqual(['docx', 'pdf', 'print-pdf', 'epub', 'audio'])
  })
})

describe('the shape of the flow', () => {
  it('gives a reading copy two steps and an edition four, ending at the lock', () => {
    expect(stepsFor('reading')).toEqual(['settings', 'review'])
    // Artwork belongs to an edition, so an a la carte export has no such step.
    expect(stepsFor('custom')).toEqual(['formats', 'settings', 'review'])
    expect(stepsFor('edition')).toEqual(['formats', 'settings', 'artwork', 'finalize'])
  })

  it('says a lock decision is still required until one is taken', () => {
    const base = { items: [], touched: false, fromEdition: true, includeArt: true, fileCount: 2 }
    expect(stepSummary('finalize', { ...base, lock: null })).toBe('Lock decision required')
    expect(stepSummary('finalize', { ...base, lock: true })).toBe('Text locked')
    expect(stepSummary('finalize', { ...base, lock: false })).toBe('Current draft')
  })
})

describe('the settings each format shows', () => {
  const options = defaultWizardOptions()

  it('asks a print format about its page, type, furniture and contents', () => {
    const groups = settingGroups('print-pdf', format({ page_count: '412' }), options)
    expect(groups.map(g => g.label)).toEqual(['Page', 'Type', 'Chapters & furniture', 'Contents'])
  })

  it('offers the custom measurements only once Custom is chosen', () => {
    const plain = settingGroups('print-pdf', undefined, options)
    expect(plain[0].rows.some(r => r.id === 'customWidth')).toBe(false)
    const custom = defaultWizardOptions()
    custom.print = { ...custom.print, trimSize: 'custom' }
    expect(settingGroups('print-pdf', undefined, custom)[0].rows.some(r => r.id === 'customWidth')).toBe(true)
  })

  it('binds a contents toggle to the format asking it, not to a shared answer', () => {
    const epub = settingGroups('epub', undefined, options)
    const contents = epub[epub.length - 1].rows.find(r => r.id === 'includeCopyright')
    expect(contents && 'field' in contents && contents.field).toEqual({ group: 'epub', key: 'includeCopyright' })
  })

  it('leaves a control no exporter reads yet without a field', () => {
    const epub = settingGroups('epub', undefined, options)
    const version = epub[0].rows.find(r => r.id === 'epubVersion')
    expect(version && 'field' in version && version.field).toBeNull()
  })

  // A narration script is its own document. It used to borrow the reading
  // copy's answers, which is why none of its settings could be changed.
  it('gives a narration script its own answers, not the reading copy’s', () => {
    expect(optionGroupFor('audio')).toBe('audio')
    expect(optionGroupFor('pdf')).toBe('pdf')
    expect(optionGroupFor('docx')).toBe('docx')
  })

  it('binds every narration row to a field the renderer reads', () => {
    const groups = settingGroups('audio', undefined, defaultWizardOptions())
    expect(groups.map(g => g.label)).toEqual(['Script layout', 'Narration aids', 'Contents'])
    const live = groups.flatMap(g => g.rows).filter(r => r.kind !== 'static')
    expect(live.length).toBeGreaterThan(0)
    for (const row of live) {
      expect('field' in row && row.field, `${row.id} reaches no exporter`).toBeTruthy()
    }
  })
})

describe('reading and writing one control', () => {
  it('writes a value into its own group and leaves the others alone', () => {
    const before = defaultWizardOptions()
    const after = writeField(before, { group: 'print', key: 'fontSize' }, 11)
    expect(readField(after, { group: 'print', key: 'fontSize' })).toBe(11)
    expect(readField(after, { group: 'pdf', key: 'fontSize' })).toBe(before.pdf.fontSize)
  })
})

describe('the preview beside the settings', () => {
  it('draws the page the author actually chose', () => {
    const options = defaultWizardOptions()
    options.print = { ...options.print, trimSize: '6x9', fontSize: 10, runningHeaders: false }
    const preview = previewFor('print-pdf', options)
    expect(preview.ratio).toBe('6/9')
    expect(preview.caption).toContain('6 × 9 in')
    expect(preview.caption).toContain('Merriweather 10 pt')
    expect(preview.caption).toContain('no running headers')
  })
})

describe('the artwork step', () => {
  const items = editionItems(edition([
    format({ id: 'p', format: 'Paperback', page_count: '412' }),
    format({ id: 'e', kind: 'ebook', format: 'eBook' }),
    format({ id: 'd', kind: 'print', format: 'Paperback' }),
  ]))

  it('asks about the formats that carry artwork and no others', () => {
    const rows = artworkRows(items, edition([]), 'The Book')
    expect(rows.map(r => r.id)).toEqual(['p', 'e', 'd'])
    expect(rows[0].wide).toBe(true)
    expect(rows[1].wide).toBe(false)
  })

  it('says a print format has nothing attached rather than showing the cover as its wrap', () => {
    const rows = artworkRows(items, edition([]), 'The Book')
    expect(rows[0].badge).toBe('Not set')
    expect(rows[0].file).toBe('Choose a wrap file…')
  })

  it('names the wrap once one is attached to that format alone', () => {
    const withWrap = editionItems(edition([
      format({ id: 'p', wrap: { file_name: 'wrap.pdf', stored: false, size_label: '12.25 × 9.25 in · 300 dpi' } }),
      format({ id: 'q', format: 'Hardcover' }),
    ]))
    const rows = artworkRows(withWrap, edition([]), 'The Book')
    expect(rows[0].file).toBe('wrap.pdf')
    expect(rows[0].spec).toContain('12.25 × 9.25 in')
    expect(rows[1].file).toBe('Choose a wrap file…')
  })
})

describe('the review', () => {
  const items = editionItems(edition([format({ id: 'p' }), format({ id: 'e', kind: 'ebook', format: 'eBook' })]))

  it('counts the artwork among the files when it travels', () => {
    const art = artworkRows(items, edition([]), 'The Book')
    expect(fileCount(items, true, art)).toBe(4)
    expect(fileCount(items, false, art)).toBe(2)
  })

  it('says what the text will be only when there is an edition to lock it to', () => {
    const art = artworkRows(items, edition([]), 'The Book')
    const base = { items, includeArt: true, art, words: 73873, fileCount: 4 }
    const loose = reviewRows({ ...base, mode: 'custom', lock: null })
    expect(loose.find(r => r.k === 'Text')).toBeUndefined()
    const bound = reviewRows({ ...base, mode: 'edition', edition: edition([]), lock: true })
    expect(bound.find(r => r.k === 'Text')?.v).toContain('73,873 words')
    const draft = reviewRows({ ...base, mode: 'edition', edition: edition([]), lock: false })
    expect(draft.find(r => r.k === 'Text')?.v).toBe('Current draft, not locked')
  })

  it('calls a reading copy’s artwork none rather than interior only', () => {
    const rows = reviewRows({ mode: 'reading', items, includeArt: false, art: [], lock: null, words: 10, fileCount: 1 })
    expect(rows.find(r => r.k === 'Artwork')?.v).toBe('None')
  })
})

describe('an edition goes out as one archive', () => {
  const items = editionItems(edition([
    format({ id: 'p', format: 'Paperback' }),
    format({ id: 'e', kind: 'ebook', format: 'eBook' }),
  ]))

  it('names the archive for the book and the edition, safely', () => {
    expect(bundleName('Wide Water', edition([]))).toBe('Wide Water — First edition')
    expect(bundleName('Hope: A Novel', undefined)).toBe('Hope- A Novel')
    expect(bundleName('   ', undefined)).toBe('Untitled')
  })

  it('says a bundle and its folders on the review, not a pile of files', () => {
    const art = artworkRows(items, edition([]), 'Wide Water')
    const rows = reviewRows({
      mode: 'edition', edition: edition([]), items, includeArt: true, art,
      lock: false, words: 100, fileCount: 4, title: 'Wide Water',
    })
    expect(rows.find(r => r.k === 'Bundle')?.v).toBe('Wide Water — First edition.zip')
    expect(rows.find(r => r.k === 'Files')?.v).toBe('4 files in 2 folders')
  })

  it('still counts loose files for an export that belongs to no edition', () => {
    const rows = reviewRows({
      mode: 'custom', items, includeArt: false, art: [], lock: null,
      words: 100, fileCount: 2, title: 'Wide Water',
    })
    expect(rows.find(r => r.k === 'Bundle')).toBeUndefined()
    expect(rows.find(r => r.k === 'Files')?.v).toBe('2 files')
  })

  it('stamps every format’s answers with the edition and format they were made for', () => {
    const marked = stamped(defaultWizardOptions(), 'ed1', 'p')
    for (const group of [marked.shared, marked.epub, marked.pdf, marked.print]) {
      expect(group.editionID).toBe('ed1')
      expect(group.formatID).toBe('p')
    }
  })

  it('sends one request carrying every chosen format', () => {
    const request = bundleRequest('ed1', true, items, () => defaultWizardOptions())
    expect(request.edition_id).toBe('ed1')
    expect(request.include_artwork).toBe(true)
    expect(request.items.map(i => [i.format_id, i.output])).toEqual([['p', 'print-pdf'], ['e', 'epub']])
    expect(request.items[0].print.formatID).toBe('p')
    expect(request.items[1].epub.editionID).toBe('ed1')
  })
})
