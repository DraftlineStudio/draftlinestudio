// Per-book daily writing history, persisted in localStorage, powering the
// Writing Dashboard goal-streak row. Dates are LOCAL calendar days in
// YYYY-MM-DD form — a streak square must flip at the writer's midnight,
// not UTC's (toISOString would shift evening sessions onto tomorrow).

import type { BookData } from '../../../types/draftline'
import { bookKey } from '../Analysis/shared'

export interface DayWords {
  date: string // ISO date YYYY-MM-DD
  words: number
}

type HistoryMap = Record<string, number>

const KEEP_DAYS = 60

const storageKey = (book: BookData) => `draftline.writing-history.${bookKey(book)}`

function toISODate(d: Date): string {
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${month}-${day}`
}

export function todayISO(): string {
  return toISODate(new Date())
}

function isoDaysAgo(days: number): string {
  const d = new Date()
  d.setDate(d.getDate() - days) // Date handles month/DST rollover
  return toISODate(d)
}

function readHistory(book: BookData): HistoryMap {
  try {
    const raw = localStorage.getItem(storageKey(book))
    if (!raw) return {}
    const parsed: unknown = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {}
    const map: HistoryMap = {}
    for (const [date, words] of Object.entries(parsed as Record<string, unknown>)) {
      if (typeof words === 'number' && isFinite(words) && words >= 0) map[date] = words
    }
    return map
  } catch {
    return {}
  }
}

function writeHistory(book: BookData, map: HistoryMap) {
  try {
    localStorage.setItem(storageKey(book), JSON.stringify(map))
  } catch {
    // Non-fatal: the streak row just won't persist.
  }
}

// Upsert today's word count and prune entries older than KEEP_DAYS. An
// existing entry is never lowered — words_today can transiently read 0
// around the day rollover before the store catches up.
export function recordTodayWords(book: BookData, words: number) {
  const map = readHistory(book)
  const today = todayISO()
  const cutoff = isoDaysAgo(KEEP_DAYS)
  let changed = false
  for (const date of Object.keys(map)) {
    if (date < cutoff) {
      delete map[date]
      changed = true
    }
  }
  const next = Math.max(map[today] ?? 0, Math.max(0, Math.floor(words)))
  if (next > 0 && next !== map[today]) {
    map[today] = next
    changed = true
  }
  if (changed) writeHistory(book, map)
}

// Last n calendar days, oldest → newest, days without an entry as 0 words.
export function getLastNDays(book: BookData, n: number): DayWords[] {
  const map = readHistory(book)
  const out: DayWords[] = []
  for (let i = n - 1; i >= 0; i--) {
    const date = isoDaysAgo(i)
    out.push({ date, words: map[date] ?? 0 })
  }
  return out
}
