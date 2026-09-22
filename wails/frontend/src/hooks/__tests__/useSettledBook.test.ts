// The settle rule, tested without React: bookStore replaces the book object on
// every editor flush, so whole-manuscript work must not key off that identity.
// A different project still has to swap in at once, or one book's figures show
// against another.
//
// Environment: plain node, no jsdom, so this exercises the decision the hook
// makes rather than rendering it.
import { describe, it, expect } from 'vitest'
import { bookKey } from '../../components/tools/Analysis/shared'
import type { BookData } from '../../types/draftline'

function bookAt(path: string, created: string, text: string): BookData {
  return {
    file_path: path,
    metadata: { created },
    body: [{ content: text }],
  } as unknown as BookData
}

// What the hook's state updater does when `book` changes: keep the previous
// snapshot while it is the same project, otherwise swap immediately.
const settle = (prev: BookData | null, next: BookData | null) =>
  prev && next && bookKey(prev) === bookKey(next) ? prev : next

describe('the settled book snapshot', () => {
  it('keeps the old snapshot while the writer is typing in the same book', () => {
    const first = bookAt('C:/books/a.draftline', '2026-01-01', 'one')
    const afterKeystroke = bookAt('C:/books/a.draftline', '2026-01-01', 'one two')

    expect(settle(first, afterKeystroke)).toBe(first)
  })

  it('swaps immediately when a different book is opened', () => {
    const first = bookAt('C:/books/a.draftline', '2026-01-01', 'one')
    const other = bookAt('C:/books/b.draftline', '2026-01-01', 'other')

    expect(settle(first, other)).toBe(other)
  })

  it('swaps immediately for a new book at the same path', () => {
    const first = bookAt('C:/books/a.draftline', '2026-01-01', 'one')
    const replaced = bookAt('C:/books/a.draftline', '2026-06-01', 'one')

    expect(settle(first, replaced)).toBe(replaced)
  })

  it('takes the first book, and a close, immediately', () => {
    const first = bookAt('C:/books/a.draftline', '2026-01-01', 'one')
    expect(settle(null, first)).toBe(first)
    expect(settle(first, null)).toBeNull()
  })
})
