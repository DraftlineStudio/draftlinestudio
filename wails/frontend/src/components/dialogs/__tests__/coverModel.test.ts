import { describe, expect, it } from 'vitest'
import type { Edition, EditionCover } from '../../../types/draftline'
import {
  COVER_EXTENSIONS, EDITION_COVER_CAVEAT, PRINT_COVER_CAVEAT, attachedLabel, coverAssetURL,
  coverFacts, coverFullURL, coverNotices, coverThumbURL, fileSize, firstArtworkPath,
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
  it('reports the pixels, the size and the encoding the search settled on', () => {
    const facts = coverFacts(cover())
    const byLabel = Object.fromEntries(facts.map(f => [f.label, f.value]))
    expect(byLabel['Pixels']).toBe('1600 × 2560')
    expect(byLabel['Size']).toBe('598 KB')
    expect(byLabel['Encoding']).toBe('JPEG, quality 88')
    expect(byLabel['Colour']).toBe('sRGB')
    expect(byLabel['From']).toBe('4000 × 6400 TIFF')
  })

  it('does not claim a JPEG quality for a PNG', () => {
    const facts = coverFacts(cover({ encoding: 'png', quality: 0, file: 'cover.png' }))
    expect(facts.find(f => f.label === 'Encoding')?.value).toBe('PNG')
  })

  it('says greyscale when the artwork carried no colour', () => {
    const facts = coverFacts(cover({ greyscale: true }))
    expect(facts.find(f => f.label === 'Colour')?.value).toBe('greyscale')
  })

  it('mentions the larger copy only when one was kept', () => {
    expect(coverFacts(cover()).some(f => f.label === 'Larger copy')).toBe(false)
    const big = coverFacts(cover({ large_file: 'cover_large.jpg', large_width: 2400, large_height: 3840, large_bytes: 1_500_000 }))
    expect(big.find(f => f.label === 'Larger copy')?.value).toBe('2400 × 3840 · 1.43 MB')
  })

  it('says plainly when no print-ready original was recorded', () => {
    const facts = coverFacts(cover({ source_path: '' }))
    expect(facts.find(f => f.label === 'Print-ready original')?.value).toBe('not recorded')
  })
})

describe('what the panel says a cover is not', () => {
  it('states the arithmetic that makes it not a print cover', () => {
    expect(PRINT_COVER_CAVEAT).toContain('no spine')
    expect(PRINT_COVER_CAVEAT).toContain('no bleed')
    expect(PRINT_COVER_CAVEAT).toContain('5.33 × 8.53')
    expect(PRINT_COVER_CAVEAT).toContain('6 × 9')
  })

  it('says the artwork belongs to the edition rather than to the book', () => {
    expect(EDITION_COVER_CAVEAT).toContain('edition')
    expect(EDITION_COVER_CAVEAT).toContain('ISBN')
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
