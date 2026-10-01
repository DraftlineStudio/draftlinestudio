// What a new chapter is called before the writer touches the field.
//
// The rule it has to hold: continue a numbering the book already uses, and
// never invent one. A chapter is called whatever the writer calls it, so a
// book of named chapters must not start growing numbers on its own.
//
// All titles here are invented for these tests.

import { describe, it, expect } from 'vitest'
import { nextChapterTitle } from '../chapters'

describe('nextChapterTitle', () => {
  it('continues a plain numbered run', () => {
    expect(nextChapterTitle(['Chapter 1', 'Chapter 2', 'Chapter 3'], 'Chapter')).toBe('Chapter 4')
  })

  it('starts from nothing when the section is empty', () => {
    expect(nextChapterTitle([], 'Chapter')).toBe('')
  })

  // The whole point of the ask: a book of named chapters gets no number.
  it('invents no number for named chapters', () => {
    expect(nextChapterTitle(['Prologue', 'The Lighthouse Weekend', 'After'], 'Chapter')).toBe('')
  })

  it('keeps the numbered label and drops the old name', () => {
    expect(nextChapterTitle(['Chapter 11 — The Quay'], 'Chapter')).toBe('Chapter 12')
    expect(nextChapterTitle(['Chapter 4: Low Water'], 'Chapter')).toBe('Chapter 5')
  })

  it('follows the book’s own wording, not the type’s', () => {
    expect(nextChapterTitle(['Ch 7'], 'Ch')).toBe('Ch 8')
    expect(nextChapterTitle(['1', '2'], 'Chapter')).toBe('3')
  })

  // Counting from the highest, not the last, so the Epilogue at the end of the
  // list does not reset the chapters and the suggestion is never a title that
  // already exists.
  it('counts past the highest number wherever it sits in the list', () => {
    expect(nextChapterTitle(['Chapter 1', 'Chapter 12', 'Epilogue'], 'Chapter')).toBe('Chapter 13')
    expect(nextChapterTitle(['Chapter 9', 'Chapter 3'], 'Chapter')).toBe('Chapter 10')
  })

  it('keeps zero padding', () => {
    expect(nextChapterTitle(['Chapter 01', 'Chapter 02'], 'Chapter')).toBe('Chapter 03')
    expect(nextChapterTitle(['Chapter 09'], 'Chapter')).toBe('Chapter 10')
  })

  // Adding a Prologue to a numbered book must not produce "Chapter 13".
  it('ignores numbering that belongs to another type', () => {
    expect(nextChapterTitle(['Chapter 1', 'Chapter 2'], 'Prologue')).toBe('')
    expect(nextChapterTitle(['Part 1', 'Part 2'], 'Part')).toBe('Part 3')
  })

  it('leaves a title alone when the number is not the label', () => {
    expect(nextChapterTitle(['12 Bells for the Harbour'], 'Chapter')).toBe('')
    expect(nextChapterTitle(['Part 2 Chapter 3'], 'Chapter')).toBe('')
  })

  it('tolerates blank and whitespace titles among real ones', () => {
    expect(nextChapterTitle(['', '  ', 'Chapter 2'], 'Chapter')).toBe('Chapter 3')
  })
})
