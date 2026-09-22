// Text measurement over chapter HTML. One copy of each of these, used
// everywhere, so two panels never disagree about a book's length.

import type { BookData } from '../types/draftline'

/**
 * Parsed into an inert document rather than assigned to a live element, so
 * markup carrying a remote reference cannot make the parse fetch it.
 */
export function htmlToText(html: string): string {
  return new DOMParser().parseFromString(html, 'text/html').body.textContent || ''
}

export function countWords(html: string): number {
  const text = htmlToText(html)
  return text.trim().split(/\s+/).filter((w) => w.length > 0).length
}

/** Includes the copyright page, which the Go-side count does not. */
export function countBookWords(book: BookData): number {
  let total = 0
  total += countWords(book.copyright || '')
  for (const ch of book.front_matter || []) total += countWords(ch.content || '')
  for (const ch of book.body || []) total += countWords(ch.content || '')
  for (const ch of book.back_matter || []) total += countWords(ch.content || '')
  return total
}

export function countCharacters(html: string, excludeSpaces = true): number {
  const text = htmlToText(html)
  return excludeSpaces ? text.replace(/\s/g, '').length : text.length
}

export function countSentences(html: string): number {
  const text = htmlToText(html)
  return text.split(/[.!?]+/).filter(s => s.trim().length > 0).length
}

export function countParagraphs(html: string): number {
  const pMatches = html.match(/<p[^>]*>/gi)
  if (pMatches && pMatches.length > 0) {
    return pMatches.length
  }
  const text = htmlToText(html)
  return text.split(/\n\s*\n/).filter(p => p.trim().length > 0).length
}

export function estimateReadingTime(html: string, wordsPerMinute = 200): number {
  const words = countWords(html)
  return Math.ceil(words / wordsPerMinute)
}

export function getCurrentContent(book: BookData | null, section: string, index: number): string {
  if (!book) return ''
  if (section === 'copyright') return book.copyright || ''
  if (section === 'front_matter') return book.front_matter[index]?.content || ''
  if (section === 'body') return book.body[index]?.content || ''
  if (section === 'back_matter') return book.back_matter[index]?.content || ''
  return ''
}
