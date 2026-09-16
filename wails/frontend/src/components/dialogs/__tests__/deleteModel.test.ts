import { describe, expect, it } from 'vitest'
import type { Edition, EditionFormat, EditionIndex, EditionSnapshot } from '../../../types/draftline'
import {
  deleteButtonLabel, deleteConfirmed, deleteSteps, editionDeletion, formatDeletion,
} from '../deleteModel'

const format = (patch: Partial<EditionFormat> = {}): EditionFormat =>
  ({ id: 'f1', kind: 'print', format: 'Paperback', ...patch })

const edition = (patch: Partial<Edition> = {}): Edition =>
  ({ id: 'ed1', label: 'First edition', year: '2026', status: 'Draft', formats: [format()], ...patch })

const index = (one: Edition, snapshots: EditionSnapshot[] = []): EditionIndex =>
  ({ version: 1, editions: [one], snapshots })

const snapshot = (id: string): EditionSnapshot =>
  ({ id, frozen: '2026-04-02T11:42:00Z', word_count: 73873, sections: 25, members: 26, bytes: 1, title: 'x' } as EditionSnapshot)

describe('how hard a delete is to confirm', () => {
  it('asks once for a draft and twice for something published', () => {
    const draft = edition()
    expect(deleteSteps(editionDeletion(index(draft), draft))).toBe(1)

    const live = edition({ status: 'Published' })
    expect(deleteSteps(editionDeletion(index(live), live))).toBe(2)

    const published = format({ status: 'Published', isbn13: '978-1-7371829-1-1' })
    const owner = edition({ formats: [published] })
    expect(deleteSteps(formatDeletion(index(owner), owner, published))).toBe(2)
  })

  it('takes only the record’s own name, never a reflex', () => {
    const one = edition({ status: 'Published' })
    const deletion = editionDeletion(index(one), one)
    expect(deleteConfirmed(deletion, 'First edition')).toBe(true)
    expect(deleteConfirmed(deletion, '  FIRST EDITION ')).toBe(true)
    expect(deleteConfirmed(deletion, 'yes')).toBe(false)
    expect(deleteConfirmed(deletion, '')).toBe(false)
    expect(deleteConfirmed(null, '')).toBe(false)
  })

  it('names what the last button will actually do', () => {
    const draft = edition()
    const soft = editionDeletion(index(draft), draft)
    expect(deleteButtonLabel(soft, 1)).toBe('Delete edition')

    const live = edition({ status: 'Published' })
    const hard = editionDeletion(index(live), live)
    expect(deleteButtonLabel(hard, 1)).toBe('Yes, continue')
    expect(deleteButtonLabel(hard, 2)).toBe('Delete edition')

    const published = format({ status: 'Published' })
    const owner = edition({ formats: [published] })
    expect(deleteButtonLabel(formatDeletion(index(owner), owner, published), 2)).toBe('Delete format')
  })
})

describe('what an author is told goes with it', () => {
  it('counts the formats, the numbers and the artwork on an edition', () => {
    const one = edition({
      cover: { id: 'c1', file: 'cover.jpg', thumb_file: 't.jpg', width: 1600, height: 2560, bytes: 1 } as never,
      formats: [
        format({ id: 'a', format: 'Paperback', isbn13: '978-1-7371829-1-1', wrap: { file_name: 'wrap.pdf', stored: false } }),
        format({ id: 'b', format: 'eBook', kind: 'ebook' }),
      ],
    })
    const losses = editionDeletion(index(one), one).losses.join(' ')
    expect(losses).toContain('2 formats: Paperback, eBook.')
    expect(losses).toContain('1 ISBN')
    expect(losses).toContain('cover artwork')
    expect(losses).toContain('1 print-ready wrap')
  })

  it('says plainly when an edition has nothing under it yet', () => {
    const empty = edition({ formats: [] })
    expect(editionDeletion(index(empty), empty).losses[0]).toContain('no formats yet')
  })

  it('tells a format’s delete that the rest of the edition is untouched', () => {
    const one = edition({ formats: [format({ id: 'a', isbn13: '978-1-7371829-1-1' }), format({ id: 'b', format: 'eBook' })] })
    const deletion = formatDeletion(index(one), one, one.formats[0])
    expect(deletion.title).toBe('Delete the paperback of First edition?')
    expect(deletion.losses.join(' ')).toContain('978-1-7371829-1-1')
    expect(deletion.losses.join(' ')).toContain('Nothing else on First edition is touched.')
  })

  it('says a locked manuscript survives when another format still stands on it', () => {
    const one = edition({
      formats: [
        format({ id: 'a', snapshot_id: 's1' }),
        format({ id: 'b', format: 'eBook', kind: 'ebook', snapshot_id: 's1' }),
      ],
    })
    const idx = index(one, [snapshot('s1')])
    expect(formatDeletion(idx, one, one.formats[0]).note).toContain('stays')
  })

  it('says a locked manuscript goes when nothing else stands on it', () => {
    const one = edition({ formats: [format({ id: 'a', snapshot_id: 's1' })] })
    const idx = index(one, [snapshot('s1')])
    const note = formatDeletion(idx, one, one.formats[0]).note
    expect(note).toContain('goes with it')
    expect(note).toContain('73,873 words')
  })

  it('says nothing about a snapshot when there is none', () => {
    const one = edition()
    expect(formatDeletion(index(one), one, one.formats[0]).note).toBe('')
  })
})
