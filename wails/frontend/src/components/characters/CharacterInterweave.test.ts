import { describe, expect, it } from 'vitest'
import { clampChapter, contiguousRuns, crossingFraction, positionOfParagraph } from './CharacterInterweave'

describe('contiguousRuns', () => {
  it('keeps a single unbroken appearance as one run', () => {
    expect(contiguousRuns([0, 1, 2, 3])).toEqual([[0, 3]])
  })

  it('splits on a gap so an absence stays visible', () => {
    // The old rail drew one line from first to last chapter, which made a
    // character who vanishes for six chapters look continuously present.
    expect(contiguousRuns([0, 1, 8, 9, 10])).toEqual([[0, 1], [8, 10]])
  })

  it('handles isolated single-chapter appearances', () => {
    expect(contiguousRuns([2, 5, 9])).toEqual([[2, 2], [5, 5], [9, 9]])
  })

  it('returns nothing for a character with no recorded chapters', () => {
    expect(contiguousRuns([])).toEqual([])
  })
})

describe('crossingFraction', () => {
  it('centres a lone crossing in its chapter', () => {
    expect(crossingFraction(0, 1)).toBe(0.5)
  })

  it('separates crossings that share a chapter', () => {
    const positions = [0, 1, 2].map(index => crossingFraction(index, 3))
    expect(new Set(positions).size).toBe(3)
    expect(positions).toEqual([...positions].sort((a, b) => a - b))
  })

  it('keeps every crossing inside the column', () => {
    for (const total of [1, 2, 5, 12]) {
      for (let index = 0; index < total; index++) {
        const fraction = crossingFraction(index, total)
        expect(fraction).toBeGreaterThanOrEqual(0.2)
        expect(fraction).toBeLessThanOrEqual(0.8)
      }
    }
  })
})

describe('clampChapter', () => {
  it('keeps an out-of-range chapter on the canvas', () => {
    expect(clampChapter(-3, 10)).toBe(0)
    expect(clampChapter(99, 10)).toBe(9)
  })

  it('survives a missing or non-numeric chapter index', () => {
    expect(clampChapter(Number.NaN, 10)).toBe(0)
    expect(clampChapter(Number.POSITIVE_INFINITY, 10)).toBe(0)
  })
})

describe('beat placement within a chapter', () => {
  it('keeps beats inside the chapter and ordered by paragraph', () => {
    for (const paragraph of [0, 1, 12, 300, 4000]) {
      const fraction = positionOfParagraph(paragraph)
      expect(fraction).toBeGreaterThanOrEqual(0.12)
      expect(fraction).toBeLessThanOrEqual(0.86)
    }
    expect(positionOfParagraph(60)).toBeGreaterThan(positionOfParagraph(2))
  })
})
