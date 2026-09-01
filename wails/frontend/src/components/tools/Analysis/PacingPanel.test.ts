import { describe, expect, it } from 'vitest'
import type { ChapterAnalysis } from '../../../types/draftline'
import { findTempoShifts, median, tempoBand } from './PacingPanel'

function chapter(index: number, tempo: number, words = 1_000): ChapterAnalysis {
  return {
    chapter_index: index,
    title: `Chapter ${index + 1}`,
    word_count: words,
    tempo_score: tempo,
  } as ChapterAnalysis
}

describe('pacing presentation model', () => {
  it('uses the same tempo boundaries as the backend', () => {
    expect(tempoBand(42)).toBe('measured')
    expect(tempoBand(43)).toBe('balanced')
    expect(tempoBand(67)).toBe('balanced')
    expect(tempoBand(68)).toBe('brisk')
  })

  it('uses a median chapter length that resists extreme chapters', () => {
    expect(median([800, 900, 1_000, 8_000])).toBe(950)
    expect(median([800, 1_000, 8_000])).toBe(1_000)
  })

  it('surfaces only noticeable neighboring tempo transitions', () => {
    const shifts = findTempoShifts([
      chapter(0, 50),
      chapter(1, 55),
      chapter(2, 72),
      chapter(3, 40),
      chapter(4, 90, 10), // divider: ignored
    ])

    expect(shifts).toHaveLength(2)
    expect(shifts[0].delta).toBe(-32)
    expect(shifts[0].to.chapter_index).toBe(3)
    expect(shifts[1].delta).toBe(17)
  })
})
