// Frozen manuscripts, mirroring internal/types/snapshot.go.
//
// The split from draftline.ts follows the Go split for the same reason: these
// are records ABOUT text, never the text. A snapshot's words are a whole novel
// and they deliberately do not cross the Wails bridge — the format record
// names one, the backend reads it out of the project file at export time, and
// nothing in the frontend ever holds it.

/** The catalogue record of one frozen manuscript. */
export interface EditionSnapshot {
  /** The SHA-256 of the frozen content, which is also where it is filed. */
  id: string
  /** When this text was FIRST frozen. Freezing the same words again keeps it. */
  frozen: string
  /** The book's title as it stood, so a record reads without opening it. */
  title?: string
  word_count: number
  sections: number
  /** What this snapshot occupies inside the project file. */
  members: number
  bytes: number
}

/** What the backend's FreezeSnapshot hands back. */
export interface SnapshotResult {
  success: boolean
  error?: string
  snapshot?: EditionSnapshot
  /**
   * True when this exact text had already been frozen, so the project file
   * holds one copy that two formats point at rather than two copies.
   */
  reused?: boolean
}
