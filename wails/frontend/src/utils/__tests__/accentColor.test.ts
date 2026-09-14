import { describe, expect, it } from 'vitest'
import { avatarColor, colorKeyLetter } from '../accentColor'

describe('book accent color keying', () => {
  it('files a title under the first word after a leading "The"', () => {
    expect(colorKeyLetter('The Rookery')).toBe('R')
    expect(colorKeyLetter('the undertow')).toBe('U')
    expect(colorKeyLetter('  The   Long Walk')).toBe('L')
    expect(avatarColor('The Rookery')).toBe(avatarColor('Rookery'))
  })

  it('leaves other titles alone, including ones that merely start with "The" as a word fragment', () => {
    expect(colorKeyLetter('Skin')).toBe('S')
    expect(colorKeyLetter('Theatre of Blood')).toBe('T')
    expect(colorKeyLetter('A Storm of Swords')).toBe('A')
  })

  it('never returns an undefined color for empty or bare-article titles', () => {
    expect(avatarColor('')).toBeTypeOf('string')
    expect(avatarColor('The')).toBe(avatarColor('T'))
  })
})
