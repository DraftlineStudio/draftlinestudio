// The full wraparound a printer needs. Draftline never makes one and never
// alters one: it keeps a copy or remembers where the file is, and hands it
// back at export exactly as it arrived.
export interface EditionWrap {
  file_name: string
  bytes?: number
  // Pixel dimensions when the file states them.
  width?: number
  height?: number
  // The physical size a printer works in, when the file states one
  // ("12.25 × 9.25 in · 300 dpi").
  size_label?: string
  // True when the bytes are kept inside the project. False means only
  // source_path is known and the file must still be on disk at export.
  stored: boolean
  // What keeping the copy costs, already worded ("about 18 MB").
  stored_label?: string
  // The archive member holding the bytes while stored. It is what the project
  // hands back when the copy has to be written out again.
  member?: string
  // Where the author's own file lives, always recorded.
  source_path?: string
  // Detects the file being replaced or moved since it was attached.
  source_checksum?: string
  attached?: string
  // A small copy the screen shows, so a panel never loads a print-resolution
  // file. Absent for a PDF or a PSD, which Go cannot rasterise.
  preview_file?: string
  preview_width?: number
  preview_height?: number
}
