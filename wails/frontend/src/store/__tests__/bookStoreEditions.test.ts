// The publishing record on the book store.
//
// The thing worth asserting is not that an action exists but that an edition
// rides the same funnel a chapter does: typing an ISBN dirties the book and
// the ordinary five-second autosave writes it. Before this, export and
// edition settings lived in component state and were thrown away when the
// dialog closed.
//
// Environment: plain node. The only boundary mocked is the generated Wails
// binding module.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { BookData } from '../../types/draftline'

const mocks = vi.hoisted(() => ({
  NewBook: vi.fn(),
  OpenBookDialog: vi.fn(),
  PickBookPath: vi.fn(),
  SaveBook: vi.fn(),
  SaveBookAs: vi.fn(),
  SaveBookSnapshots: vi.fn(),
  OpenRecentProject: vi.fn(),
  AddRecentProject: vi.fn(),
  IndexBook: vi.fn(),
  MergeEntities: vi.fn(),
  SplitEntity: vi.fn(),
  ImportEPUB: vi.fn(),
  ImportDOCX: vi.fn(),
  ShowInfoDialog: vi.fn(),
  CloseBookFile: vi.fn().mockResolvedValue(undefined),
  AnalyzeBook: vi.fn(),
  LoadSettings: vi.fn(),
  SaveSettings: vi.fn(),
  BrowseForDirectory: vi.fn(),
  GetRecentProjects: vi.fn(),
  RemoveRecentProject: vi.fn(),
  ClearRecentProjects: vi.fn(),
}))

vi.mock('../../../wailsjs/go/main/App', () => mocks)

async function flushMicrotasks() {
  for (let i = 0; i < 20; i++) await Promise.resolve()
}

function makeBook(overrides: Partial<BookData> = {}): BookData {
  return {
    version: '2.0',
    metadata: { title: 'Harbour Lights', author: 'Ines Marrow', isbn: '', publisher: 'Windlass and Co.', created: '', modified: '' },
    copyright: '',
    front_matter: [],
    body: [{ title: 'Chapter 1', type: 'chapter', content: '<p>original</p>' }],
    back_matter: [],
    file_path: 'C:/tmp/test.draftline',
    ...overrides,
  }
}

let bookStoreMod: typeof import('../bookStore')
const store = () => bookStoreMod.useBookStore.getState()
const record = () => bookStoreMod.useBookStore.getState().book?.editions

beforeEach(async () => {
  vi.resetModules()
  for (const fn of Object.values(mocks)) fn.mockReset()
  mocks.AddRecentProject.mockResolvedValue(undefined)
  mocks.SaveBook.mockResolvedValue({ success: true, file_path: 'C:/tmp/test.draftline' })
  vi.useFakeTimers()
  // Each test re-imports the store, so a timer scheduled by the previous
  // test still points at the previous module instance. Left pending, it fires
  // inside the next test and saves a book that test never wrote.
  vi.clearAllTimers()
  bookStoreMod = await import('../bookStore')
})

describe('registering editions', () => {
  it('starts a publishing record on a book that has none, and autosaves it', async () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    expect(record()).toBeUndefined()

    const editionID = store().addEdition('2026')
    expect(editionID).toBe('ed-1')
    expect(record()?.version).toBe(1)
    expect(record()?.editions).toHaveLength(1)
    expect(store().isDirty).toBe(true)

    await vi.advanceTimersByTimeAsync(5_000)
    await flushMicrotasks()
    expect(mocks.SaveBook).toHaveBeenCalledTimes(1)
    const saved: BookData = mocks.SaveBook.mock.calls[0][0]
    expect(saved.editions?.editions[0].label).toBe('First edition')
  })

  it('adds a format under the edition it was asked for, and nowhere else', () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    const first = store().addEdition('2026')
    const second = store().addEdition('2030')

    const paperback = store().addFormat(first, 'print')
    const ebook = store().addFormat(second, 'ebook')

    expect(record()?.editions[0].formats.map(f => f.id)).toEqual([paperback])
    expect(record()?.editions[1].formats.map(f => f.id)).toEqual([ebook])
    expect(record()?.editions[0].formats[0].format).toBe('Paperback')
    expect(record()?.editions[1].formats[0].format).toBe('eBook')
  })

  it('refuses to add a format to an edition that is not there', () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    store().addEdition('2026')
    expect(store().addFormat('ed-9', 'print')).toBe('')
    expect(record()?.editions[0].formats).toHaveLength(0)
  })

  it('does nothing at all when no book is open', () => {
    bookStoreMod.useBookStore.setState({ book: null, isDirty: false })
    expect(store().addEdition('2026')).toBe('')
    expect(store().addFormat('ed-1', 'print')).toBe('')
    store().updateFormat('fmt-1', { isbn13: '978-1-9471345-1-5' })
    expect(store().book).toBeNull()
  })
})

describe('editing what an edition holds', () => {
  beforeEach(() => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    const edition = store().addEdition('2026')
    store().addFormat(edition, 'print')
    bookStoreMod.useBookStore.setState({ isDirty: false })
  })

  it('writes an ISBN onto one format and dirties the book', async () => {
    store().updateFormat('fmt-1', { isbn13: '978-1-9471345-1-5', status: 'Registered' })
    expect(record()?.editions[0].formats[0].isbn13).toBe('978-1-9471345-1-5')
    expect(store().isDirty).toBe(true)

    await vi.advanceTimersByTimeAsync(5_000)
    await flushMicrotasks()
    expect(mocks.SaveBook).toHaveBeenCalledTimes(1)
    const saved: BookData = mocks.SaveBook.mock.calls[0][0]
    expect(saved.editions?.editions[0].formats[0].isbn13).toBe('978-1-9471345-1-5')
  })

  it('leaves the rest of the format alone when one field changes', () => {
    store().updateFormat('fmt-1', { page_count: '412' })
    store().updateFormat('fmt-1', { isbn13: '978-1-9471345-1-5' })
    const format = record()!.editions[0].formats[0]
    expect([format.page_count, format.isbn13, format.paper_stock])
      .toEqual(['412', '978-1-9471345-1-5', 'Cream, 55#'])
  })

  it('changes an edition without disturbing its formats', () => {
    store().updateEdition('ed-1', { label: 'First edition, revised', revision_note: 'Typesetting corrected.' })
    expect(record()?.editions[0].label).toBe('First edition, revised')
    expect(record()?.editions[0].formats).toHaveLength(1)
  })
})

describe('removing editions and formats', () => {
  it('removes one format and leaves its neighbours', () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    const edition = store().addEdition('2026')
    const first = store().addFormat(edition, 'print')
    const second = store().addFormat(edition, 'ebook')

    store().removeFormat(first)
    expect(record()?.editions[0].formats.map(f => f.id)).toEqual([second])
  })

  // Removing a draft edition must not rewrite the copyright years of a book
  // already published: the edition that pointed at it is relinked to what it
  // pointed at, so the chain still reaches the first edition's year.
  it('relinks the chain when an edition in the middle is removed', () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    store().addEdition('2026')
    store().addEdition('2030')
    store().addEdition('2033')
    expect(record()?.editions[2].previous_edition_id).toBe('ed-2')

    store().removeEdition('ed-2')
    expect(record()?.editions.map(e => e.id)).toEqual(['ed-1', 'ed-3'])
    expect(record()?.editions[1].previous_edition_id).toBe('ed-1')
  })

  it('ignores a removal of something that is not there', () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    store().addEdition('2026')
    bookStoreMod.useBookStore.setState({ isDirty: false })

    store().removeEdition('ed-9')
    expect(record()?.editions).toHaveLength(1)
    expect(store().isDirty).toBe(false)
  })
})

describe('duplicating an edition through the store', () => {
  it('registers the copy and selects it by returning its identifier', () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    const edition = store().addEdition('2026')
    const format = store().addFormat(edition, 'print')
    store().updateFormat(format, { isbn13: '978-1-9471345-1-5', status: 'Published' })

    const copy = store().duplicateEdition(edition, '2030')
    expect(copy).toBe('ed-2')
    expect(record()?.editions[1].previous_edition_id).toBe('ed-1')
    // The published number stays with the edition it was issued for.
    expect(record()?.editions[0].formats[0].isbn13).toBe('978-1-9471345-1-5')
    expect(record()?.editions[1].formats[0].isbn13).toBeUndefined()
  })

  it('answers nothing when the edition to copy is not there', () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    expect(store().duplicateEdition('ed-9', '2030')).toBe('')
  })
})

// ── Frozen manuscripts ─────────────────────────────────────────────────────
//
// The text a published ISBN stands for rides the same funnel: freezing dirties
// the book and the ordinary autosave writes the catalogue. The words
// themselves never reach the store — only the record that names them.

const snapshotRecord = (id: string) => ({
  id, frozen: '2026-04-14T09:00:00Z', title: 'Harbour Lights',
  word_count: 91_400, sections: 34, members: 36, bytes: 512_000,
})

describe('freezing the text an ISBN stands for', () => {
  it('stamps the format, catalogues the text, and autosaves both', async () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    const edition = store().addEdition('2026')
    const format = store().addFormat(edition, 'ebook')

    store().freezeFormat(format, snapshotRecord('aaa'))
    expect(record()?.editions[0].formats[0].snapshot_id).toBe('aaa')
    expect(record()?.snapshots).toHaveLength(1)
    expect(store().isDirty).toBe(true)

    await vi.advanceTimersByTimeAsync(5_000)
    await flushMicrotasks()
    const saved: BookData = mocks.SaveBook.mock.calls[0][0]
    expect(saved.editions?.snapshots?.[0].word_count).toBe(91_400)
  })

  it('stores the same words once for two formats', () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    const edition = store().addEdition('2026')
    const ebook = store().addFormat(edition, 'ebook')
    const paperback = store().addFormat(edition, 'print')

    store().freezeFormat(ebook, snapshotRecord('aaa'))
    store().freezeFormat(paperback, snapshotRecord('aaa'))
    expect(record()?.snapshots).toHaveLength(1)
    expect(record()?.editions[0].formats.map(f => f.snapshot_id)).toEqual(['aaa', 'aaa'])
  })

  it('releasing one of two formats leaves the words the other went out with', () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    const edition = store().addEdition('2026')
    const ebook = store().addFormat(edition, 'ebook')
    const paperback = store().addFormat(edition, 'print')
    store().freezeFormat(ebook, snapshotRecord('aaa'))
    store().freezeFormat(paperback, snapshotRecord('aaa'))

    store().releaseFormatSnapshot(ebook)
    expect(record()?.snapshots).toHaveLength(1)
    expect(record()?.editions[0].formats[0].snapshot_id).toBeUndefined()
    expect(record()?.editions[0].formats[1].snapshot_id).toBe('aaa')

    store().releaseFormatSnapshot(paperback)
    expect(record()?.snapshots).toEqual([])
  })

  it('removing a format lets go of the words nothing else points at', () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    const edition = store().addEdition('2026')
    const ebook = store().addFormat(edition, 'ebook')
    const paperback = store().addFormat(edition, 'print')
    store().freezeFormat(ebook, snapshotRecord('aaa'))
    store().freezeFormat(paperback, snapshotRecord('bbb'))

    store().removeFormat(ebook)
    expect(record()?.snapshots?.map(s => s.id)).toEqual(['bbb'])
    expect(record()?.editions[0].formats).toHaveLength(1)
  })

  it('removing a format that shares its words keeps them for the other', () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    const edition = store().addEdition('2026')
    const ebook = store().addFormat(edition, 'ebook')
    const paperback = store().addFormat(edition, 'print')
    store().freezeFormat(ebook, snapshotRecord('aaa'))
    store().freezeFormat(paperback, snapshotRecord('aaa'))

    store().removeFormat(ebook)
    expect(record()?.snapshots?.map(s => s.id)).toEqual(['aaa'])
    expect(record()?.editions[0].formats[0].snapshot_id).toBe('aaa')
  })

  it('a duplicated edition starts with nothing frozen', () => {
    bookStoreMod.useBookStore.setState({ book: makeBook(), isDirty: false })
    const edition = store().addEdition('2026')
    const format = store().addFormat(edition, 'print')
    store().freezeFormat(format, snapshotRecord('aaa'))

    store().duplicateEdition(edition, '2030')
    expect(record()?.editions[1].formats[0].snapshot_id).toBeUndefined()
    expect(record()?.editions[0].formats[0].snapshot_id).toBe('aaa')
  })
})
