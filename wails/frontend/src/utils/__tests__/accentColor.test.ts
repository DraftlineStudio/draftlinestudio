import { describe, expect, it } from 'vitest'
import { avatarColor, colorKeyLetter } from '../accentColor'

describe('book accent color keying', () => {
  it('files a title under the first word after a leading "The"', () => {
    expect(colorKeyLetter('The Lantern')).toBe('L')
    expect(colorKeyLetter('the harbour')).toBe('H')
    expect(colorKeyLetter('  The   Long Stair')).toBe('L')
    expect(avatarColor('The Lantern')).toBe(avatarColor('Lantern'))
  })

  it('leaves other titles alone, including ones that merely start with "The" as a word fragment', () => {
    expect(colorKeyLetter('Shale')).toBe('S')
    expect(colorKeyLetter('Theatre of Tides')).toBe('T')
    expect(colorKeyLetter('A Tower of Glass')).toBe('A')
  })

  it('never returns an undefined color for empty or bare-article titles', () => {
    expect(avatarColor('')).toBeTypeOf('string')
    expect(avatarColor('The')).toBe(avatarColor('T'))
  })
})
