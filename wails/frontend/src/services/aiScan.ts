/**
 * Whole-book AI-detection scanner.
 *
 * Wraps services/aiDetection.ts `analyzeText` with an async, batched scan of
 * every chapter (front matter + body + back matter) so the AI Analysis panel
 * can show per-chapter scores without ever blocking the UI thread. Results are
 * cached per book object (WeakMap) so reopening the panel is instant, plus per
 * book identity (file path + created stamp) so an edit shows the previous
 * complete scan while the debounced re-scan runs.
 */

import { useEffect, useState } from 'react'
import { analyzeText } from './aiDetection'
import { htmlToText } from '../utils/textUtils'
import type { BookData, ChapterItem } from '../types/draftline'

export interface ChapterAIScore {
  globalIndex: number // combined index across front_matter + body + back_matter
  title: string
  score: number // 0-100 AI signal from analyzeText
  wordCount: number
}

export interface BookAIScanState {
  scores: ChapterAIScore[] // book order (ascending globalIndex), partial while running
  done: boolean
}

/** Chapters shorter than this (plain text) are skipped — matches analyzeText's floor. */
const MIN_TEXT_CHARS = 100
/** Chapters analyzed per setTimeout tick. */
const BATCH_SIZE = 3
/** Edits create a new book object; wait this long before re-scanning. */
const RESCAN_DEBOUNCE_MS = 2000

// Per-object cache: instant when the same book object is seen again.
const scanCache = new WeakMap<BookData, BookAIScanState>()
// Per-identity cache: last COMPLETE scan, shown as stale data during re-scans.
const lastCompleteByKey = new Map<string, BookAIScanState>()
// Book identities that have ever started a scan — a repeat sighting of a key on
// a NEW book object means the user is editing, so the re-scan is debounced.
const startedKeys = new Set<string>()

function scanKey(book: BookData): string {
  return `${book.file_path || ''}|${book.metadata?.created || ''}`
}

function combinedChapters(book: BookData): ChapterItem[] {
  return [...(book.front_matter || []), ...(book.body || []), ...(book.back_matter || [])]
}

/**
 * Scan every analyzable chapter of `book` with analyzeText, a few chapters per
 * event-loop tick. Returns partial results as they arrive; `done` flips true
 * once every eligible chapter has been scored.
 */
export function useBookAIScan(book: BookData | null): BookAIScanState {
  const [state, setState] = useState<BookAIScanState>({ scores: [], done: false })

  useEffect(() => {
    if (!book) {
      setState({ scores: [], done: true })
      return
    }

    const cached = scanCache.get(book)
    if (cached && cached.done) {
      setState(cached)
      return
    }

    const key = scanKey(book)
    const stale = lastCompleteByKey.get(key)
    // Keep the previous complete scan on screen while the re-scan runs.
    setState(stale ? { scores: stale.scores, done: false } : { scores: [], done: false })

    // Eligibility (>= MIN_TEXT_CHARS of plain text) is checked inside the
    // batched tick — parsing every chapter's HTML up front here would block
    // the main thread on large books, which is what the batching exists to
    // prevent.
    const chapters = combinedChapters(book)

    // First scan of a book starts immediately, as does a remount that
    // interrupted one (partial per-object cache); a repeat of the same book
    // identity on a NEW object means the user is typing — debounce that.
    const delay = startedKeys.has(key) && !cached ? RESCAN_DEBOUNCE_MS : 0
    startedKeys.add(key)

    let cancelled = false
    let next = 0
    const scores: ChapterAIScore[] = []
    let timer: ReturnType<typeof setTimeout>

    const tick = () => {
      if (cancelled) return
      const end = Math.min(next + BATCH_SIZE, chapters.length)
      for (; next < end; next++) {
        const chapter = chapters[next]
        const content = chapter.content || ''
        const text = htmlToText(content)
        if (text.length < MIN_TEXT_CHARS) continue
        scores.push({
          globalIndex: next,
          title: chapter.title || `Chapter ${next + 1}`,
          score: analyzeText(content).score,
          wordCount: text.trim().split(/\s+/).filter(w => w.length > 0).length,
        })
      }
      const done = next >= chapters.length
      const snapshot: BookAIScanState = { scores: [...scores], done }
      scanCache.set(book, snapshot)
      if (done) lastCompleteByKey.set(key, snapshot)
      setState(snapshot)
      if (!done) timer = setTimeout(tick, 0)
    }

    timer = setTimeout(tick, delay)
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [book])

  return state
}

export interface PassageScore {
  excerpt: string // plain-text paragraph
  score: number // 0-100 AI signal
}

/**
 * Score a single chapter's paragraphs. Paragraphs shorter than
 * `analyzeThreshold` plain-text characters are skipped (too short for the
 * heuristics to be meaningful). Returns passages sorted by score descending.
 */
export function scanPassages(html: string, analyzeThreshold = 150): PassageScore[] {
  const div = document.createElement('div')
  div.innerHTML = html
  let paragraphs = Array.from(div.querySelectorAll('p')).map(p => (p.textContent || '').trim())
  if (paragraphs.length === 0) {
    paragraphs = (div.textContent || '').split(/\n\s*\n/).map(p => p.trim())
  }
  return paragraphs
    .filter(p => p.length >= analyzeThreshold)
    .map(p => ({ excerpt: p, score: analyzeText(p).score }))
    .sort((a, b) => b.score - a.score)
}
