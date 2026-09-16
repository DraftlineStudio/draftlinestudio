// The frozen-manuscript model: what the screen says about the text behind an
// ISBN, and the two transforms that are the whole retention policy.
//
// Every book, name and number here is invented.

import { describe, it, expect } from 'vitest'
import type { EditionIndex, EditionSnapshot } from '../../../types/draftline'
import {
  byteLabel, exportTextNote, frozenLabel, referenceCount, sharedWith,
  runExportWithFreeze, snapshotFacts, snapshotFootprint, snapshotFor,
  withFrozenSnapshot, withReleasedSnapshot,
} from '../snapshotModel'

const snapshot = (id: string, over: Partial<EditionSnapshot> = {}): EditionSnapshot => ({
  id,
  frozen: '2026-04-14T09:00:00Z',
  title: 'The Salt Lantern',
  word_count: 91_400,
  sections: 34,
  members: 36,
  bytes: 512_000,
  ...over,
})

function index(): EditionIndex {
  return {
    version: 1,
    editions: [{
      id: 'ed-1', label: 'First edition', year: '2026', status: 'Published',
      formats: [
        { id: 'fmt-ebook', kind: 'ebook', format: 'eBook', isbn13: '978-1-9471345-0-8' },
        { id: 'fmt-paper', kind: 'print', format: 'Paperback', isbn13: '978-1-9471345-1-5' },
        { id: 'fmt-hard', kind: 'print', format: 'Hardcover' },
      ],
    }],
    snapshots: [],
  }
}

const formatIn = (i: EditionIndex, id: string) =>
  i.editions.flatMap(e => e.formats).find(f => f.id === id)!

describe('reading the record', () => {
  it('finds the frozen manuscript a format was published from', () => {
    let i = index()
    i = withFrozenSnapshot(i, 'fmt-ebook', snapshot('aaa'))
    expect(snapshotFor(i, formatIn(i, 'fmt-ebook'))?.id).toBe('aaa')
    expect(snapshotFor(i, formatIn(i, 'fmt-paper'))).toBeUndefined()
  })

  it('answers nothing for a format whose record names words the book has lost', () => {
    const i = index()
    i.editions[0].formats[0].snapshot_id = 'gone'
    expect(snapshotFor(i, formatIn(i, 'fmt-ebook'))).toBeUndefined()
  })

  it('names the other formats published from the same words', () => {
    let i = index()
    i = withFrozenSnapshot(i, 'fmt-ebook', snapshot('aaa'))
    i = withFrozenSnapshot(i, 'fmt-paper', snapshot('aaa'))
    expect(sharedWith(i, 'aaa', 'fmt-ebook')).toEqual(['Paperback — First edition'])
    expect(sharedWith(i, 'aaa', 'fmt-paper')).toEqual(['eBook — First edition'])
    expect(referenceCount(i, 'aaa')).toBe(2)
    expect(referenceCount(i, 'bbb')).toBe(0)
  })
})

describe('saying it on screen', () => {
  it('reports sizes an author can read', () => {
    expect(byteLabel(0)).toBe('0 KB')
    expect(byteLabel(512_000)).toBe('500 KB')
    expect(byteLabel(4_500_000)).toBe('4.3 MB')
  })

  it('reports when the text was frozen, and does not invent a date it cannot read', () => {
    expect(frozenLabel('2026-04-14T09:00:00Z')).toMatch(/2026/)
    expect(frozenLabel('')).toBe('')
    expect(frozenLabel('some time last spring')).toBe('some time last spring')
  })

  it('describes the frozen text as length, date and size, and names the details frozen with it', () => {
    const facts = snapshotFacts(snapshot('abcdef0123456789'))
    expect(facts.map(f => f.label)).toEqual(['Frozen', 'Book details', 'Length', 'In this project file', 'Reference'])
    expect(facts[1].value).toContain('The Salt Lantern')
    expect(facts[2].value).toContain('91,400 words')
    expect(facts[2].value).toContain('34 sections')
    expect(facts[3].value).toBe('500 KB')
    expect(facts[4].value).toBe('abcdef012345')
    expect(snapshotFacts(undefined)).toEqual([])
  })

  it('adds up what every frozen manuscript occupies', () => {
    let i = index()
    i = withFrozenSnapshot(i, 'fmt-ebook', snapshot('aaa'))
    i = withFrozenSnapshot(i, 'fmt-paper', snapshot('bbb', { members: 12, bytes: 200_000 }))
    const footprint = snapshotFootprint(i)
    expect(footprint.count).toBe(2)
    expect(footprint.members).toBe(48)
    expect(footprint.bytes).toBe(712_000)
    expect(footprint.label).toContain('2 frozen manuscripts')
    expect(snapshotFootprint(index()).label).toContain('No text is frozen')
  })
})

describe('what an export is about to contain', () => {
  it('says plainly that a format with no frozen text uses the draft, and will freeze it', () => {
    const i = index()
    const note = exportTextNote(i, formatIn(i, 'fmt-ebook'))
    expect(note).toContain('as it stands today')
    expect(note).toContain('freezes it')
  })

  it('says plainly that a format with frozen text uses those words and not the draft', () => {
    let i = index()
    i = withFrozenSnapshot(i, 'fmt-ebook', snapshot('aaa'))
    const note = exportTextNote(i, formatIn(i, 'fmt-ebook'))
    expect(note).toContain('91,400 words')
    expect(note).toContain('not the draft')
    expect(note).toMatch(/2026/)
  })

  // The note used to claim that everything but the text came from the record
  // as it stands now. The title, the author and the publisher are inside the
  // frozen manuscript, so an author who corrects a misspelled name and
  // re-exports a published ISBN still gets the misspelling; the record's own
  // fields do reach the file. Saying the two apart is the whole point.
  it('does not claim that corrected book details reach an edition already frozen', () => {
    let i = index()
    i = withFrozenSnapshot(i, 'fmt-ebook', snapshot('aaa'))
    const note = exportTextNote(i, formatIn(i, 'fmt-ebook'))
    expect(note).not.toContain('Everything else')
    expect(note).toContain('the ones it was frozen with')
    expect(note).toContain('imprint of record')
  })

  it('says what an export attached to no edition does', () => {
    expect(exportTextNote(index(), undefined)).toContain('not attached to an edition')
  })
})

describe('freezing', () => {
  it('stamps the format and catalogues the text in one change', () => {
    const i = withFrozenSnapshot(index(), 'fmt-ebook', snapshot('aaa'))
    expect(formatIn(i, 'fmt-ebook').snapshot_id).toBe('aaa')
    expect(i.snapshots).toHaveLength(1)
    expect(formatIn(i, 'fmt-paper').snapshot_id).toBeUndefined()
  })

  it('stores the same words once however many ISBNs point at them', () => {
    let i = withFrozenSnapshot(index(), 'fmt-ebook', snapshot('aaa'))
    i = withFrozenSnapshot(i, 'fmt-paper', snapshot('aaa', { frozen: '2031-01-01T00:00:00Z' }))
    expect(i.snapshots).toHaveLength(1)
    // The moment kept is when this text FIRST went out, not the second freeze.
    expect(i.snapshots![0].frozen).toBe('2026-04-14T09:00:00Z')
    expect(formatIn(i, 'fmt-paper').snapshot_id).toBe('aaa')
  })

  it('re-freezing one format moves only that format', () => {
    let i = withFrozenSnapshot(index(), 'fmt-ebook', snapshot('aaa'))
    i = withFrozenSnapshot(i, 'fmt-paper', snapshot('aaa'))
    i = withFrozenSnapshot(i, 'fmt-paper', snapshot('bbb'))
    expect(formatIn(i, 'fmt-ebook').snapshot_id).toBe('aaa')
    expect(formatIn(i, 'fmt-paper').snapshot_id).toBe('bbb')
    expect(i.snapshots?.map(s => s.id)).toEqual(['aaa', 'bbb'])
  })
})

describe('releasing', () => {
  it('leaves the words when another format is still published from them', () => {
    let i = withFrozenSnapshot(index(), 'fmt-ebook', snapshot('aaa'))
    i = withFrozenSnapshot(i, 'fmt-paper', snapshot('aaa'))

    const after = withReleasedSnapshot(i, 'fmt-ebook')
    expect(formatIn(after, 'fmt-ebook').snapshot_id).toBeUndefined()
    expect(formatIn(after, 'fmt-paper').snapshot_id).toBe('aaa')
    expect(after.snapshots).toHaveLength(1)
  })

  it('lets the words go when the last format that pointed at them does', () => {
    let i = withFrozenSnapshot(index(), 'fmt-ebook', snapshot('aaa'))
    i = withFrozenSnapshot(i, 'fmt-paper', snapshot('bbb'))

    let after = withReleasedSnapshot(i, 'fmt-ebook')
    expect(after.snapshots?.map(s => s.id)).toEqual(['bbb'])
    after = withReleasedSnapshot(after, 'fmt-paper')
    expect(after.snapshots).toEqual([])
  })

  it('does nothing to a format that had nothing frozen', () => {
    const i = withFrozenSnapshot(index(), 'fmt-ebook', snapshot('aaa'))
    expect(withReleasedSnapshot(i, 'fmt-hard')).toBe(i)
  })
})

// ── Freezing around an export ──────────────────────────────────────────────
//
// Pressing Export opens the system's save dialog inside the exporter, and
// dismissing it is an ordinary thing to do. What must not happen then is a
// published ISBN quietly bound to the words that were on screen at the moment
// of the click: the author who cancels, writes for a week and exports again
// would get the week-old text, with nothing on any screen having said so.

describe('exporting a registered edition', () => {
  const record = snapshot('frozen-1')

  function run(over: {
    formatID?: string
    editions?: EditionIndex
    freeze?: () => Promise<{ success: boolean; snapshot?: EditionSnapshot; reused?: boolean; error?: string }>
    write?: (book: { editions?: EditionIndex }) => Promise<{ success: boolean; file_path?: string; error?: string }>
  } = {}) {
    const committed: EditionSnapshot[] = []
    const discarded: string[] = []
    const written: Array<{ editions?: EditionIndex }> = []
    const freezes: number[] = []
    const call = runExportWithFreeze({
      book: { editions: over.editions ?? index() },
      formatID: over.formatID ?? 'fmt-ebook',
      freeze: over.freeze ?? (async () => { freezes.push(1); return { success: true, snapshot: record } }),
      write: over.write ?? (async book => { written.push(book); return { success: true, file_path: '/tmp/out.epub' } }),
      commit: r => committed.push(r),
      discard: id => discarded.push(id),
    })
    return { call, committed, discarded, written, freezes }
  }

  it('writes the file from the words it froze, so the two cannot disagree', async () => {
    const r = run()
    const result = await r.call
    expect(result.ok).toBe(true)
    expect(r.written).toHaveLength(1)
    const format = r.written[0].editions!.editions[0].formats[0]
    expect(format.snapshot_id).toBe('frozen-1')
    expect(r.written[0].editions!.snapshots).toHaveLength(1)
    expect(r.committed).toEqual([record])
    expect(result.note).toContain('91,400 words')
  })

  it('freezes nothing when the author dismisses the save dialog', async () => {
    const r = run({ write: async () => ({ success: false, error: 'cancelled' }) })
    const result = await r.call
    expect(result.ok).toBe(false)
    expect(result.cancelled).toBe(true)
    expect(result.error).toBe('')
    expect(r.committed).toEqual([])
    expect(r.discarded).toEqual(['frozen-1'])
    expect(result.note).toContain('nothing was frozen')
  })

  it('freezes nothing when the export fails outright, and says why', async () => {
    const r = run({ write: async () => ({ success: false, error: 'the disk is full' }) })
    const result = await r.call
    expect(result.error).toBe('the disk is full')
    expect(r.committed).toEqual([])
    expect(r.discarded).toEqual(['frozen-1'])
  })

  it('does not write a file when the text could not be frozen', async () => {
    const r = run({ freeze: async () => ({ success: false, error: 'there is nothing written to freeze yet' }) })
    const result = await r.call
    expect(result.error).toBe('there is nothing written to freeze yet')
    expect(r.written).toEqual([])
    expect(r.committed).toEqual([])
    expect(r.discarded).toEqual([])
  })

  it('freezes nothing a second time, and exports the words already frozen', async () => {
    let i = index()
    i = withFrozenSnapshot(i, 'fmt-ebook', record)
    const r = run({ editions: i })
    const result = await r.call
    expect(result.ok).toBe(true)
    expect(r.freezes).toEqual([])
    expect(r.committed).toEqual([])
    expect(result.note).toContain('not the draft on screen')
  })

  it('freezes nothing for an export made from scratch', async () => {
    const r = run({ formatID: '' })
    const result = await r.call
    expect(result.ok).toBe(true)
    expect(r.freezes).toEqual([])
    expect(r.committed).toEqual([])
    expect(result.note).toBe('')
  })

  it('names the other ISBN when two formats are published from the same words', async () => {
    let i = index()
    i = withFrozenSnapshot(i, 'fmt-paper', record)
    const r = run({
      editions: i,
      freeze: async () => ({ success: true, snapshot: record, reused: true }),
    })
    const result = await r.call
    expect(result.note).toContain('Paperback — First edition')
    expect(r.written[0].editions!.snapshots).toHaveLength(1)
  })
})
