// The print-ready wrap: the full wraparound artwork a printer needs, which is
// back, spine and front in one piece, at print resolution, with bleed.
//
// Draftline never makes one and never alters one. It is the author's file,
// handed back at export exactly as it arrived. Two things can be true of it:
// the project keeps a copy, or the project remembers where it lives. Both
// record the same facts; only one costs disk and sync.

import type { EditionFormat, EditionWrap } from '../../types/draftline'

// What a printer asks for and what an author recognises. Not a constraint:
// Draftline accepts whatever is handed to it and reports what it saw.
export const WRAP_EXTENSIONS = ['.pdf', '.tif', '.tiff', '.png', '.jpg', '.jpeg', '.psd', '.ai', '.eps']

export function looksLikeWrap(path: string): boolean {
  const lower = path.toLowerCase()
  return WRAP_EXTENSIONS.some(ext => lower.endsWith(ext))
}

export function firstWrapPath(paths: string[]): string {
  return paths.find(looksLikeWrap) ?? ''
}

export function fileSizeLabel(bytes: number): string {
  if (!bytes) return ''
  if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
  if (bytes >= 1024 * 1024) return `${Math.round(bytes / (1024 * 1024))} MB`
  return `${Math.max(1, Math.round(bytes / 1024))} KB`
}

// The line beside the heading: the physical size a printer works in when the
// file states one, and the pixel size otherwise. A wrap with neither is still
// a wrap; it just has nothing to say here.
export function wrapSizeLabel(wrap: EditionWrap): string {
  const physical = (wrap.size_label ?? '').trim()
  if (physical) return physical
  if (wrap.width && wrap.height) return `${wrap.width} × ${wrap.height}`
  return fileSizeLabel(wrap.bytes ?? 0)
}

// The warning, with the real number rather than a vague caution. The file is
// kept exactly as supplied, which is the point of keeping it and also why it
// costs what it costs.
export function keepInProjectWarning(wrap: EditionWrap | undefined): string {
  if (!wrap) return ''
  return `The total size of this project will increase by ${fileSizeLabel(wrap.bytes ?? 0)}, as this is stored in lossless print-ready format.`
}

// Where the screen fetches the small copy from. Same route the covers use, and
// keyed on what was attached so replacing the artwork shows the new picture.
export function wrapPreviewURL(editionID: string, wrap: EditionWrap | undefined): string {
  if (!editionID || !wrap?.preview_file) return ''
  const stamp = encodeURIComponent(wrap.attached ?? wrap.file_name)
  return `/editions/${encodeURIComponent(editionID)}/${encodeURIComponent(wrap.preview_file)}?v=${stamp}`
}

// Whether a format can carry a wrap at all. Only a printed object has one.
export const takesWrap = (format: EditionFormat): boolean => format.kind === 'print'
