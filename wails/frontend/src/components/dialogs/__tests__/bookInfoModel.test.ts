// The Book & editions form's rules. The ISBN half is the frontend copy of
// internal/types/isbn.go and is checked against the same numbers, so the two
// cannot drift into disagreeing about what a valid ISBN is.

import { describe, expect, it } from 'vitest'
import type { ISBNEntry, Metadata } from '../../../types/draftline'
import { blockingProblems, checkBook, isbnRows, metadataPatch, normalizeISBN, validISBN } from '../bookInfoModel'

const meta = (over: Partial<Metadata> = {}): Metadata => ({
  title: 'A Lantern', author: 'R. Vance', isbn: '', publisher: '', created: '', modified: '', ...over,
})

describe('ISBN checking matches the backend', () => {
  it('accepts well-formed numbers in either length and any separators', () => {
    for (const value of ['978-0-306-40615-7', '9780306406157', '0-306-40615-2', '0306406152', '080442957X']) {
      expect([value, validISBN(value)]).toEqual([value, true])
    }
  })

  it('rejects a mistyped digit, a wrong length, and a misplaced X', () => {
    for (const value of ['', '978-0-306-40615-8', '0306406153', '97803064061', '97803064061570', '978030640615X', 'abc']) {
      expect([value, validISBN(value)]).toEqual([value, false])
    }
  })

  it('strips separators and upper-cases the check X', () => {
    expect(normalizeISBN('  978-0 306.40615_7 ')).toBe('9780306406157')
    expect(normalizeISBN('08044-2957x')).toBe('080442957X')
  })
})

describe('what the form refuses and what it merely warns about', () => {
  it('blocks only a book with no title', () => {
    const problems = checkBook(meta({ title: '   ' }), [])
    expect(blockingProblems(problems).map(p => p.field)).toEqual(['title'])
  })

  it('warns about an ISBN that does not check out, without blocking the save', () => {
    const problems = checkBook(meta(), [{ format: 'ebook', value: '978-0-306-40615-8' }])
    expect(problems.map(p => p.field)).toEqual(['isbn-0'])
    expect(blockingProblems(problems)).toEqual([])
  })

  it('warns when one format is given two ISBNs, because only the first is used', () => {
    const rows: ISBNEntry[] = [
      { format: 'ebook', value: '9780306406157' },
      { format: 'ebook', value: '0306406152' },
    ]
    const problems = checkBook(meta(), rows)
    expect(problems.map(p => p.field)).toEqual(['isbn-1'])
    expect(problems[0].message).toContain('Only the first is used')
  })

  it('says nothing about a blank row or two different formats', () => {
    const rows: ISBNEntry[] = [
      { format: 'ebook', value: '9780306406157' },
      { format: 'paperback', value: '0306406152' },
      { format: '', value: '  ' },
    ]
    expect(checkBook(meta(), rows)).toEqual([])
  })
})

describe('what a save writes', () => {
  it('trims every field, drops blank ISBN rows, and mirrors the legacy field', () => {
    const patch = metadataPatch(
      { title: '  A Lantern  ', author: ' R. Vance ', language: ' en-GB ', short_description: ' A keeper. ' },
      [{ format: 'ebook', value: ' 9780306406157 ' }, { format: '', value: '   ' }],
    )
    expect(patch.title).toBe('A Lantern')
    expect(patch.author).toBe('R. Vance')
    expect(patch.language).toBe('en-GB')
    expect(patch.short_description).toBe('A keeper.')
    expect(patch.isbns).toEqual([{ format: 'ebook', value: '9780306406157' }])
    expect(patch.isbn).toBe('9780306406157')
  })

  it('leaves a book with no ISBNs with an empty legacy field rather than a stale one', () => {
    expect(metadataPatch({ title: 'A Lantern' }, []).isbn).toBe('')
  })
})

describe('what the form starts with', () => {
  it('uses the book’s own list when it has one', () => {
    const rows = isbnRows(meta({ isbns: [{ format: 'ebook', value: '9780306406157' }], isbn: '9780306406157' }))
    expect(rows).toEqual([{ format: 'ebook', value: '9780306406157' }])
  })

  it('seeds a row from a legacy book that only ever had the single field', () => {
    expect(isbnRows(meta({ isbn: '9780306406157' }))).toEqual([{ format: '', value: '9780306406157' }])
  })

  it('starts empty for a book with no ISBN at all', () => {
    expect(isbnRows(meta())).toEqual([])
  })
})
