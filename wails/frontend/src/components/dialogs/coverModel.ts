// What the cover panel says about an attached cover.
//
// Pure functions, no React and no bindings, because most of what this screen
// has to get right is wording and arithmetic: the address the image is fetched
// from, the facts beside it, and the several things the author has to be told
// about a cover that Draftline made rather than received.
//
// The statements below are not decoration. A 1600 x 2560 front cover looks
// like a finished cover on screen and is not one: it has no spine, no back and
// no bleed, and at 300 dpi it is smaller than the trim size the same project
// prints at. An author who does not know that finds out from a printer.

import type { Edition, EditionCover } from '../../types/draftline'

// The URL space this process serves cover art from, same-origin. See
// wails/cover.go and the middleware in wails/plugins.go.
const EDITION_ASSETS = '/editions/'

// coverAssetURL addresses one of an edition's images.
//
// The cover's identifier rides along as a query. It changes every time artwork
// is attached, which is what makes replacing a cover actually change the
// picture: without it the address editions/ed-1/cover_thumb.jpg would be the
// same before and after, and the webview would go on showing the old one.
export function coverAssetURL(editionID: string, file: string, coverID: string): string {
  if (!editionID || !file) return ''
  return `${EDITION_ASSETS}${encodeURIComponent(editionID)}/${encodeURIComponent(file)}?v=${encodeURIComponent(coverID)}`
}

export function coverThumbURL(edition: Edition): string {
  const cover = edition.cover
  if (!cover) return ''
  return coverAssetURL(edition.id, cover.thumb_file, cover.id)
}

export function coverFullURL(edition: Edition): string {
  const cover = edition.cover
  if (!cover) return ''
  return coverAssetURL(edition.id, cover.file, cover.id)
}

export interface CoverFact {
  label: string
  value: string
}

// coverFacts is the rows beside the picture, in the design's Cover file /
// Pixels shape, with what Draftline did to the artwork added underneath.
export function coverFacts(cover: EditionCover): CoverFact[] {
  const facts: CoverFact[] = [
    { label: 'Cover file', value: cover.file },
    { label: 'Pixels', value: `${cover.width} × ${cover.height}` },
    { label: 'Size', value: fileSize(cover.bytes) },
    { label: 'Encoding', value: encodingLabel(cover) },
  ]
  if (cover.large_file) {
    facts.push({ label: 'Larger copy', value: `${cover.large_width} × ${cover.large_height} · ${fileSize(cover.large_bytes ?? 0)}` })
  }
  facts.push({ label: 'Colour', value: cover.greyscale ? 'greyscale' : 'sRGB' })
  if (cover.source_width && cover.source_height) {
    facts.push({ label: 'From', value: `${cover.source_width} × ${cover.source_height}${cover.source_format ? ` ${cover.source_format.toUpperCase()}` : ''}` })
  }
  facts.push({ label: 'Print-ready original', value: cover.source_path || 'not recorded' })
  return facts
}

function encodingLabel(cover: EditionCover): string {
  if (cover.encoding === 'png') return 'PNG'
  return cover.quality ? `JPEG, quality ${cover.quality}` : 'JPEG'
}

// fileSize is for a person reading a panel, not for arithmetic.
export function fileSize(bytes: number): string {
  if (!bytes || bytes < 0) return '—'
  if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
  if (bytes >= 1024) return `${Math.round(bytes / 1024)} KB`
  return `${bytes} bytes`
}

// PRINT_COVER_CAVEAT is on screen whenever a cover is attached, every time,
// because the mistake it prevents is expensive and only shows up at a printer.
//
// The arithmetic in it is real: 1600 / 300 = 5.33 inches, 2560 / 300 = 8.53,
// against the 6 by 9 trim this same project offers in the export wizard.
export const PRINT_COVER_CAVEAT =
  'This is the ebook cover: the front only, with no spine, no back and no bleed. ' +
  'At 300 dots per inch 1600 × 2560 is 5.33 × 8.53 inches, smaller than the 6 × 9 trim ' +
  'Draftline prints at, so it is not a print cover and Draftline will not make one from it. ' +
  'The print-ready artwork stays where it is on your disk; Draftline remembers the file and ' +
  'checks it is still the same one.'

export const EDITION_COVER_CAVEAT =
  'Cover art belongs to the edition, not to the book, so the artwork of a first edition stays ' +
  'with the first edition’s ISBNs when a second edition is reset with new art.'

// coverNotices are the sentences about THIS artwork: what the conversion
// changed. They come from the backend, which is where the conversion happened,
// so the screen never has to guess at what was done.
export function coverNotices(cover: EditionCover): string[] {
  return cover.notes ?? []
}

// sourceStatusTone maps the answer about the print-ready original onto how
// loudly to say it. A moved original is not an error - the cover is fine - but
// it is the thing that will bite at export time.
export function sourceStatusTone(status: string): 'ok' | 'warn' | 'neutral' {
  switch (status) {
    case 'present':
      return 'ok'
    case 'changed':
    case 'moved':
      return 'warn'
    default:
      return 'neutral'
  }
}

// attachedLabel is the date under the thumbnail.
export function attachedLabel(cover: EditionCover): string {
  if (!cover.attached) return ''
  const when = new Date(cover.attached)
  if (Number.isNaN(when.getTime())) return ''
  return when.toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' })
}

// coverExtensions is what the drop target accepts, and it matches the file
// dialog's filter and internal/coverart's own list. A drop of anything else is
// refused here rather than sent to the backend to be refused there, so the
// author gets the answer without a round trip.
export const COVER_EXTENSIONS = ['.jpg', '.jpeg', '.png', '.tif', '.tiff', '.webp', '.bmp']

export function looksLikeArtwork(path: string): boolean {
  const lower = path.toLowerCase()
  return COVER_EXTENSIONS.some(ext => lower.endsWith(ext))
}

// firstArtworkPath picks what a drop meant. Dropping a folder of artwork, or a
// mixed selection, should attach the one image rather than refuse the lot.
export function firstArtworkPath(paths: string[]): string {
  return paths.find(looksLikeArtwork) ?? ''
}
