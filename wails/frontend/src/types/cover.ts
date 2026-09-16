// The record of an edition's cover art, and the answer to "is the print-ready
// original still there".
//
// This mirrors internal/types/cover.go and is a separate file for the same
// reason that one is: a cover record is entirely about bytes that are
// deliberately kept somewhere else. No image data is on it. The images live
// in the archive under editions/<edition id>/ and are shown through a
// same-origin URL this process serves; putting them here would send a
// megabyte of JPEG across the Wails bridge on every autosave.

export interface EditionCover {
  // Changes whenever the artwork does, so the display URL changes with it and
  // the webview cannot show the previous cover from its cache.
  id: string
  file: string
  thumb_file: string
  large_file?: string
  width: number
  height: number
  bytes: number
  thumb_width: number
  thumb_height: number
  thumb_bytes: number
  large_width?: number
  large_height?: number
  large_bytes?: number
  encoding: string
  quality: number
  greyscale?: boolean
  converted_from_cmyk?: boolean
  flattened_alpha?: boolean
  // The print-ready original, recorded rather than copied.
  source_path?: string
  source_checksum?: string
  source_bytes?: number
  source_width?: number
  source_height?: number
  source_modified?: string
  source_format?: string
  attached?: string
  // What was done to this artwork, in the author's terms, for the panel.
  notes?: string[]
}

export interface CoverSourceReport {
  // 'present' | 'moved' | 'changed'. Typed as a string because that is what the
  // generated binding produces, and because a value written by a later build
  // is a fact to show, not one to refuse.
  status: string
  message: string
  path?: string
}
