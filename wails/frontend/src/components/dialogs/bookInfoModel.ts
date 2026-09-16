// What the Book & Editions screen edits, and the rules it checks. Pure
// functions over metadata so the form stays a form: no React, no store.

import type { ISBNEntry, Metadata } from '../../types/draftline'

export const ISBN_FORMATS: { value: string; label: string }[] = [
  { value: '', label: 'Format…' },
  { value: 'hardcover', label: 'Hardcover' },
  { value: 'paperback', label: 'Paperback' },
  { value: 'ebook', label: 'eBook' },
  { value: 'audiobook', label: 'Audiobook' },
  { value: 'large_print', label: 'Large Print' },
  { value: 'other', label: 'Other' },
]

// Offered because they cover most of what Draftline is written in; the field
// accepts any BCP 47 tag typed by hand.
export const LANGUAGES: { value: string; label: string }[] = [
  { value: 'en-US', label: 'English (United States)' },
  { value: 'en-GB', label: 'English (United Kingdom)' },
  { value: 'en-AU', label: 'English (Australia)' },
  { value: 'en-CA', label: 'English (Canada)' },
  { value: 'fr-FR', label: 'French' },
  { value: 'de-DE', label: 'German' },
  { value: 'es-ES', label: 'Spanish' },
  { value: 'it-IT', label: 'Italian' },
  { value: 'pt-BR', label: 'Portuguese (Brazil)' },
  { value: 'nl-NL', label: 'Dutch' },
  { value: 'sv-SE', label: 'Swedish' },
  { value: 'ja-JP', label: 'Japanese' },
]

export const AUDIENCES: { value: string; label: string }[] = [
  { value: '', label: 'Not set' },
  { value: 'adult-trade', label: 'Adult — trade' },
  { value: 'young-adult', label: 'Young adult' },
  { value: 'middle-grade', label: 'Middle grade' },
  { value: 'academic', label: 'Academic' },
]

// ── ISBN ───────────────────────────────────────────────────────────────────
// The last digit of an ISBN is a checksum over the ones before it, so a
// mistyped digit is detectable rather than silently wrong. This is the
// frontend half of internal/types/isbn.go; both must agree, and
// __tests__/bookInfoModel.test.ts checks them against the same numbers.

export function normalizeISBN(value: string): string {
  return [...value].filter(c => (c >= '0' && c <= '9') || c === 'x' || c === 'X')
    .map(c => (c === 'x' ? 'X' : c)).join('')
}

export function validISBN(value: string): boolean {
  const d = normalizeISBN(value)
  if (d.length === 13) {
    let sum = 0
    for (let i = 0; i < 13; i++) {
      const n = d.charCodeAt(i) - 48
      if (n < 0 || n > 9) return false
      sum += i % 2 === 1 ? n * 3 : n
    }
    return sum % 10 === 0
  }
  if (d.length === 10) {
    let sum = 0
    for (let i = 0; i < 10; i++) {
      const c = d[i]
      let n: number
      if (c >= '0' && c <= '9') n = c.charCodeAt(0) - 48
      else if (c === 'X' && i === 9) n = 10
      else return false
      sum += n * (10 - i)
    }
    return sum % 11 === 0
  }
  return false
}

// isbn10 is the ISBN-10 form of an ISBN-13, or '' when there is none. Only
// the 978 range converts: a 979 ISBN has no ISBN-10 at all, which is a fact
// about the number rather than a failure, so the screen shows nothing.
// Mirrors ISBN10 in internal/types/isbn.go.
export function isbn10(value: string): string {
  const d = normalizeISBN(value)
  if (d.length === 10) return validISBN(d) ? d : ''
  if (d.length !== 13 || !validISBN(d) || !d.startsWith('978')) return ''
  const body = d.slice(3, 12)
  let sum = 0
  for (let i = 0; i < 9; i++) sum += (body.charCodeAt(i) - 48) * (10 - i)
  const check = (11 - (sum % 11)) % 11
  return body + (check === 10 ? 'X' : String(check))
}

// ── Validation ─────────────────────────────────────────────────────────────

export interface FieldProblem {
  field: string
  message: string
  // A warning is worth saying and never blocks a save. Only a blocking
  // problem stops the writer, and the only blocking problem is a book with
  // no title, because the title names the file and the window.
  blocking: boolean
}

export function checkBook(meta: Partial<Metadata>, isbns: ISBNEntry[]): FieldProblem[] {
  const problems: FieldProblem[] = []
  if (!(meta.title ?? '').trim()) {
    problems.push({ field: 'title', message: 'A book needs a title.', blocking: true })
  }
  const seen = new Map<string, number>()
  isbns.forEach((entry, index) => {
    const value = entry.value.trim()
    if (!value) return
    if (!validISBN(value)) {
      problems.push({
        field: `isbn-${index}`,
        message: 'That is not a valid ISBN. Check the last digit.',
        blocking: false,
      })
    }
    if (entry.format) {
      const first = seen.get(entry.format)
      if (first !== undefined) {
        problems.push({
          field: `isbn-${index}`,
          message: 'A second ISBN for the same format. Only the first is used.',
          blocking: false,
        })
      } else {
        seen.set(entry.format, index)
      }
    }
  })
  return problems
}

export const blockingProblems = (problems: FieldProblem[]): FieldProblem[] => problems.filter(p => p.blocking)

// The ISBN rows a book starts editing with: its own list, or a single row
// seeded from the legacy field so an older book does not look empty.
export function isbnRows(meta: Metadata | undefined): ISBNEntry[] {
  if (meta?.isbns?.length) return meta.isbns.map(e => ({ ...e }))
  if (meta?.isbn) return [{ format: '', value: meta.isbn }]
  return []
}

// What a save writes: trimmed text, blank ISBN rows dropped, and the legacy
// single field mirroring the first entry for format compatibility.
export function metadataPatch(draft: Partial<Metadata>, isbns: ISBNEntry[]): Partial<Metadata> {
  const cleaned = isbns
    .map(entry => ({ format: entry.format, value: entry.value.trim() }))
    .filter(entry => entry.value !== '')
  const text = (value: string | undefined) => (value ?? '').trim()
  return {
    title: text(draft.title),
    subtitle: text(draft.subtitle),
    author: text(draft.author),
    series_name: text(draft.series_name),
    series_number: text(draft.series_number),
    publisher: text(draft.publisher),
    imprint: text(draft.imprint),
    language: text(draft.language),
    copyright_holder: text(draft.copyright_holder),
    bisac_1: text(draft.bisac_1),
    bisac_2: text(draft.bisac_2),
    audience: text(draft.audience),
    keywords: text(draft.keywords),
    short_description: text(draft.short_description),
    contributors: text(draft.contributors),
    isbns: cleaned,
    isbn: cleaned[0]?.value ?? '',
  }
}
