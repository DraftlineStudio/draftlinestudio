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

// coverFacts is the rows beside the picture. The pixel size is under the
// picture itself (coverPixels), not repeated here.
export function coverFacts(cover: EditionCover): CoverFact[] {
  // Three facts and no more. The file name and the archive path are not among
  // them: the cover lives inside the .draftline and there is nothing an author
  // can do with either. What is worth knowing is what it is, what it costs,
  // and where the print-ready original they made it from actually is.
  const stored = cover.large_file
    ? `${fileSize(cover.bytes)} · ${fileSize(cover.large_bytes ?? 0)} larger copy`
    : fileSize(cover.bytes)
  return [
    { label: 'Format', value: coverFormat(cover) },
    { label: 'Stored', value: stored },
    { label: 'Print-ready original', value: sourceLabel(cover) },
  ]
}

/**
 * The original's own location, on a line of its own.
 *
 * A path is long and a fact row is two columns, so putting one in the value
 * column wrapped it into a ribbon a dozen characters wide. It gets the full
 * width, and it is the last thing on the card because it is the least often
 * needed.
 */
export function sourcePathLine(cover: EditionCover): string {
  return (cover.source_path ?? '').trim()
}

// What the author handed Draftline, said as a specification rather than as a
// path. The path is on the button beside it, which is the only thing anyone
// actually wants to do with it.
export function sourceLabel(cover: EditionCover): string {
  const size = cover.source_width && cover.source_height
    ? `${cover.source_width} × ${cover.source_height}`
    : ''
  const format = (cover.source_format ?? '').trim().toUpperCase()
  const spec = [size, format].filter(Boolean).join(' ')
  if (spec) return spec
  return cover.source_path ? 'recorded' : 'not recorded'
}

/** What the stored cover is: its encoding, and whether it carries colour. */
export function coverFormat(cover: EditionCover): string {
  return `${encodingLabel(cover)}${cover.greyscale ? ' · greyscale' : ''}`
}

/** The pixel size under the picture, and nothing else. */
export function coverPixels(cover: EditionCover): string {
  return `${cover.width} × ${cover.height}`
}

// The two facts the design canvas puts on a format: an ebook publishes pixels,
// a print format publishes a wrap. Two rows, because that is what fits beside
// a thumbnail and it is all a format needs to know about the artwork.
export function formatCoverFacts(cover: EditionCover, format: { kind?: string }): CoverFact[] {
  if (format.kind === 'print') {
    return [
      { label: 'Full wrap', value: cover.source_path ? baseName(cover.source_path) : 'not supplied' },
      { label: 'Front cover', value: `${cover.width} × ${cover.height}` },
    ]
  }
  return [
    { label: 'Cover file', value: cover.file },
    { label: 'Pixels', value: `${cover.width} × ${cover.height}` },
  ]
}

const baseName = (path: string): string => path.split(/[\/]/).pop() || path

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
// The one sentence the card carries. Everything else Draftline does to a
// cover — the resizing, the colour conversion, why it is not a print cover —
// lives in docs/frontend/EDITIONS.md and on the tooltips, not on the screen.
// The author is registering an edition, not reading a manual.
export const EDITION_COVER_NOTE = 'Stored in the project, with this edition.'

// The tooltip on the larger-copy checkbox. Off by default because Amazon
// charges a delivery fee per megabyte on every sale.
export const LARGE_COPY_HINT =
  'For Kobo, which asks for 2400 on the short edge. Off by default: a bigger cover costs a little on every copy sold.'

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
