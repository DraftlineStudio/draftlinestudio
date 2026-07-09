/**
 * Text utility functions
 *
 * Centralized utilities for text manipulation and analysis.
 * These should be used throughout the app to avoid duplication.
 */

import type { BookData } from '../types/draftline'

/**
 * Extract plain text from HTML content
 */
export function htmlToText(html: string): string {
  const div = document.createElement('div')
  div.innerHTML = html
  return div.textContent || div.innerText || ''
}

/**
 * Count words in HTML content
 */
export function countWords(html: string): number {
  const text = htmlToText(html)
  return text.trim().split(/\s+/).filter((w) => w.length > 0).length
}

/**
 * Count total words across all book sections
 */
export function countBookWords(book: BookData): number {
  let total = 0
  total += countWords(book.copyright || '')
  for (const ch of book.front_matter || []) total += countWords(ch.content || '')
  for (const ch of book.body || []) total += countWords(ch.content || '')
  for (const ch of book.back_matter || []) total += countWords(ch.content || '')
  return total
}

/**
 * Count characters (excluding spaces) in HTML content
 */
export function countCharacters(html: string, excludeSpaces = true): number {
  const text = htmlToText(html)
  return excludeSpaces ? text.replace(/\s/g, '').length : text.length
}

/**
 * Count sentences in HTML content
 */
export function countSentences(html: string): number {
  const text = htmlToText(html)
  return text.split(/[.!?]+/).filter(s => s.trim().length > 0).length
}

/**
 * Count paragraphs in HTML content
 */
export function countParagraphs(html: string): number {
  // Count <p> tags or double newlines
  const pMatches = html.match(/<p[^>]*>/gi)
  if (pMatches && pMatches.length > 0) {
    return pMatches.length
  }
  // Fallback: count double newlines
  const text = htmlToText(html)
  return text.split(/\n\s*\n/).filter(p => p.trim().length > 0).length
}

/**
 * Estimate reading time in minutes
 * Uses average reading speed of 200 words per minute
 */
export function estimateReadingTime(html: string, wordsPerMinute = 200): number {
  const words = countWords(html)
  return Math.ceil(words / wordsPerMinute)
}

/**
 * Get content for a specific section and index in a book
 */
export function getCurrentContent(book: BookData | null, section: string, index: number): string {
  if (!book) return ''
  if (section === 'copyright') return book.copyright || ''
  if (section === 'front_matter') return book.front_matter[index]?.content || ''
  if (section === 'body') return book.body[index]?.content || ''
  if (section === 'back_matter') return book.back_matter[index]?.content || ''
  return ''
}
