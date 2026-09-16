import { describe, expect, it } from 'vitest'
import type { Edition, EditionCover } from '../../../types/draftline'
import {
  COVER_EXTENSIONS, EDITION_COVER_NOTE, LARGE_COPY_HINT, attachedLabel, coverAssetURL,
  coverFacts, coverFullURL, coverNotices, coverPixels, coverThumbURL, fileSize, firstArtworkPath,
  sourceLabel,
  looksLikeArtwork, sourceStatusTone,
} from '../coverModel'

function cover(patch: Partial<EditionCover> = {}): EditionCover {
  return {
    id: 'cov-1700000000000',
    file: 'cover.jpg',
    thumb_file: 'cover_thumb.jpg',
    width: 1600,
    height: 2560,
    bytes: 612_340,
    thumb_width: 236,
    thumb_height: 378,
    thumb_bytes: 10_240,
    encoding: 'jpeg',
    quality: 88,
    source_path: 'D:/Artwork/harbour-lights-wrap.tif',
    source_checksum: 'a'.repeat(64),
    source_width: 4000,
    source_height: 6400,
    source_format: 'tiff',
    attached: '2026-09-16T10:30:00Z',
    ...patch,
  }
}

function edition(patch: Partial<Edition> = {}): Edition {
  return { id: 'ed-1', label: 'First edition', year: '2026', status: 'Draft', formats: [], ...patch }
}

describe('the address a cover is shown from', () => {
  it('is same-origin and carries no image data', () => {
    const url = coverThumbURL(edition({ cover: cover() }))
    expect(url.startsWith('/editions/')).toBe(true)
    expect(url).not.toContain('data:')
    expect(url).not.toContain('base64')
  })

  it('changes when the artwork changes, so a replaced cover is not the old one', () => {
    const first = coverThumbURL(edition({ cover: cover({ id: 'cov-1' }) }))
    const second = coverThumbURL(edition({ cover: cover({ id: 'cov-2' }) }))
    expect(first).not.toEqual(second)
    // The path is the same either way; only the identifier moves. That is the
    // point: the archive member keeps its name and the browser still refetches.
    expect(first.split('?')[0]).toEqual(second.split('?')[0])
  })

  it('addresses the full cover and the thumbnail separately', () => {
    const ed = edition({ cover: cover() })
    expect(coverFullURL(ed)).toContain('cover.jpg')
    expect(coverThumbURL(ed)).toContain('cover_thumb.jpg')
  })

  it('keeps a PNG cover named as a PNG', () => {
    const ed = edition({ cover: cover({ file: 'cover.png', thumb_file: 'cover_thumb.png', encoding: 'png', quality: 0 }) })
    expect(coverFullURL(ed)).toContain('cover.png')
  })

  it('is empty for an edition with no cover, so nothing is fetched', () => {
    expect(coverThumbURL(edition())).toBe('')
    expect(coverFullURL(edition())).toBe('')
    expect(coverAssetURL('ed-1', '', 'cov-1')).toBe('')
    expect(coverAssetURL('', 'cover.jpg', 'cov-1')).toBe('')
  })

  it('escapes anything odd in the names rather than putting it in a path', () => {
    expect(coverAssetURL('ed 1', 'cover.jpg', 'cov/1')).toBe('/editions/ed%201/cover.jpg?v=cov%2F1')
  })
})

describe('the facts beside the picture', () => {
  it('reports what it is, what it costs, and what it was made from', () => {
    const byLabel = Object.fromEntries(coverFacts(cover()).map(f => [f.label, f.value]))
    expect(byLabel['Format']).toBe('JPEG, quality 88')
    expect(byLabel['Stored']).toBe('598 KB')
    expect(byLabel['Print-ready original']).toBe('4000 × 6400 TIFF')
  })

  // The file name and the archive path are not facts an author can act on: the
  // cover lives inside the .draftline. The original's location is on a button.
  it('names neither the stored file nor a path', () => {
    const text = coverFacts(cover()).map(f => `${f.label} ${f.value}`).join(' ')
    expect(text).not.toContain('cover.jpg')
    expect(text).not.toContain('/')
  })

  it('does not claim a JPEG quality for a PNG', () => {
    expect(coverFacts(cover({ encoding: 'png', quality: 0, file: 'cover.png' }))
      .find(f => f.label === 'Format')?.value).toBe('PNG')
  })

  it('says greyscale when the artwork carried no colour', () => {
    expect(coverFacts(cover({ greyscale: true }))
      .find(f => f.label === 'Format')?.value).toContain('greyscale')
  })

  it('counts the larger copy in what is stored, only when one was kept', () => {
    expect(coverFacts(cover()).find(f => f.label === 'Stored')?.value).toBe('598 KB')
    const big = coverFacts(cover({ large_file: 'cover_large.jpg', large_width: 2400, large_height: 3840, large_bytes: 1_500_000 }))
    expect(big.find(f => f.label === 'Stored')?.value).toBe('598 KB · 1.43 MB larger copy')
  })

  it('says the original is unrecorded rather than inventing a specification', () => {
    expect(sourceLabel(cover({ source_width: 0, source_height: 0, source_format: '', source_path: '' })))
      .toBe('not recorded')
    expect(coverPixels(cover())).toBe('1600 × 2560')
  })
})

describe('what the card says, and how little of it', () => {
  // The card carries one sentence. Everything else Draftline does to a cover
  // belongs in the docs, not on a modal the author is trying to work in.
  // An edition is a catalogue entry, not a number. The line says where the
  // artwork lives and stops; making the ISBN the point of it was wrong.
  it('says the artwork belongs to the edition, in one short line', () => {
    expect(EDITION_COVER_NOTE).toContain('edition')
    expect(EDITION_COVER_NOTE).not.toContain('ISBN')
    expect(EDITION_COVER_NOTE.split(' ').length).toBeLessThan(16)
  })

  it('keeps the reason for the larger copy on a tooltip rather than the card', () => {
    expect(LARGE_COPY_HINT).toContain('Kobo')
    expect(LARGE_COPY_HINT).toContain('every copy sold')
  })

  it('shows the conversion notes the backend produced, unaltered', () => {
    const notes = ['This artwork was in CMYK…', 'This artwork had transparent areas…']
    expect(coverNotices(cover({ notes }))).toEqual(notes)
    expect(coverNotices(cover({ notes: undefined }))).toEqual([])
  })
})

describe('the print-ready original', () => {
  it('reads a moved or changed original as something to act on', () => {
    expect(sourceStatusTone('present')).toBe('ok')
    expect(sourceStatusTone('moved')).toBe('warn')
    expect(sourceStatusTone('changed')).toBe('warn')
    expect(sourceStatusTone('something later')).toBe('neutral')
  })
})

describe('dropping artwork on the card', () => {
  it('takes the first image out of whatever was dropped', () => {
    expect(firstArtworkPath(['/notes/outline.md', '/art/cover.TIF'])).toBe('/art/cover.TIF')
    expect(firstArtworkPath(['/art/a.png', '/art/b.jpg'])).toBe('/art/a.png')
  })

  it('refuses a drop with no artwork in it rather than sending it to be refused', () => {
    expect(firstArtworkPath(['/manuscript.draftline', '/notes.txt'])).toBe('')
    expect(firstArtworkPath([])).toBe('')
  })

  it('accepts every extension the file dialog offers, and no others', () => {
    for (const ext of COVER_EXTENSIONS) {
      expect(looksLikeArtwork(`/art/cover${ext}`)).toBe(true)
      expect(looksLikeArtwork(`/art/cover${ext.toUpperCase()}`)).toBe(true)
    }
    expect(looksLikeArtwork('/art/cover.psd')).toBe(false)
    expect(looksLikeArtwork('/art/cover.pdf')).toBe(false)
    expect(looksLikeArtwork('/art/cover')).toBe(false)
  })
})

describe('the small print on the card', () => {
  it('writes sizes for a person rather than in bytes', () => {
    expect(fileSize(0)).toBe('—')
    expect(fileSize(940)).toBe('940 bytes')
    expect(fileSize(11_500)).toBe('11 KB')
    expect(fileSize(1_258_291)).toBe('1.20 MB')
  })

  it('shows when a cover was attached, and nothing at all when that is unknown', () => {
    expect(attachedLabel(cover())).not.toBe('')
    expect(attachedLabel(cover({ attached: '' }))).toBe('')
    expect(attachedLabel(cover({ attached: 'not a date' }))).toBe('')
  })
})
