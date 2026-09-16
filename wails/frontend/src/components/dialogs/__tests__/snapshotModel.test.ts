// The frozen-manuscript model: what the screen says about the text behind an
// ISBN, and the two transforms that are the whole retention policy.
//
// Every book, name and number here is invented.

import { describe, it, expect } from 'vitest'
import type { EditionIndex, EditionSnapshot } from '../../../types/draftline'
import {
  byteLabel, exportTextNote, frozenLabel, referenceCount, sharedWith,
  snapshotFacts, snapshotFootprint, snapshotFor, withFrozenSnapshot, withReleasedSnapshot,
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

  it('describes the frozen text as length, date and size', () => {
    const facts = snapshotFacts(snapshot('abcdef0123456789'))
    expect(facts.map(f => f.label)).toEqual(['Frozen', 'Length', 'In this project file', 'Reference'])
    expect(facts[1].value).toContain('91,400 words')
    expect(facts[1].value).toContain('34 sections')
    expect(facts[2].value).toBe('500 KB')
    expect(facts[3].value).toBe('abcdef012345')
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
