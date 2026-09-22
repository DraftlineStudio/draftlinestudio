import { useEffect, useState } from 'react'
import type { BookData } from '../types/draftline'
// bookKey lives beside the analysis panels because it also keys their stored
// history; it is imported rather than re-homed so that storage format cannot
// drift.
import { bookKey } from '../components/tools/Analysis/shared'

// How long the book has to stop changing before whole-manuscript work re-runs.
const SETTLE_MS = 2000

/**
 * A book snapshot that settles instead of following every keystroke.
 *
 * bookStore replaces the book object on every editor flush, roughly every
 * 150ms while typing, so anything derived from `book` by identity re-runs that
 * often. Whole-manuscript passes read this instead.
 *
 * A different project swaps in immediately rather than waiting out the delay,
 * so one book's figures are never shown against another.
 */
export function useSettledBook(book: BookData | null, delayMs = SETTLE_MS): BookData | null {
  const [settled, setSettled] = useState(book)
  useEffect(() => {
    setSettled(prev => (prev && book && bookKey(prev) === bookKey(book) ? prev : book))
    const timer = window.setTimeout(() => setSettled(book), delayMs)
    return () => window.clearTimeout(timer)
  }, [book, delayMs])
  return settled
}
