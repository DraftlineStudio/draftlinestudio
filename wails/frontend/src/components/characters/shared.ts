// Shared helpers for the character codex and the in-editor character sidebar.

import type { BookData, Section } from '../../types/draftline'

export function allChapters(book: BookData) {
  return [...(book.front_matter || []), ...(book.body || []), ...(book.back_matter || [])]
}

export function chapterName(book: BookData, index: number): string {
  return allChapters(book)[index]?.title || `Chapter ${index + 1}`
}

// Discrete alpha ramp for presence cells: "none / few / some / many / lots".
export function cellAlpha(v: number): number {
  return v <= 0 ? 0 : v <= 2 ? 0.3 : v <= 4 ? 0.55 : v <= 6 ? 0.78 : 0.95
}

// Maps a combined-chapter index (front_matter + body + back_matter) back to a
// navigable (section, index) pair.
export function chapterLocation(book: BookData, index: number): { section: Section; index: number } {
  const front = book.front_matter?.length || 0
  const body = book.body?.length || 0
  if (index < front) return { section: 'front_matter', index }
  if (index < front + body) return { section: 'body', index: index - front }
  return { section: 'back_matter', index: index - front - body }
}

// One-shot handoff: the sidebar's "Open in Characters" pre-selects a
// character in the codex without a store round-trip. Consumed on view mount.
let pendingFocusId: string | null = null

export function requestCharacterFocus(id: string) {
  pendingFocusId = id
}

export function takeCharacterFocus(): string | null {
  const id = pendingFocusId
  pendingFocusId = null
  return id
}
