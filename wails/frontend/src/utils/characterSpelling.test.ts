import { describe, expect, it, vi } from 'vitest'
import type { Character } from '../types/draftline'
import { characterSpellingNames } from './characterSpelling'
import { checkWord, loadDictionary, setCustomWords, setIgnoredWords } from '../services/spellCheck'

const person = (overrides: Partial<Character> = {}): Character => ({
  id: 'person', name: 'Nelwin Brindle', role: 'minor', description: '', appearance: '',
  personality: '', motivation: '', notes: '', is_auto_detected: true, detection_status: 'accepted',
  aliases: ['Nelwin Brindls', 'Nell Brindle'], spelling_aliases: ['Nell Brindle'], ...overrides,
})

describe('character spelling exceptions', () => {
  it('includes accepted names and safe aliases without suspected spellings', () => {
    expect(characterSpellingNames([person()])).toEqual(['Nelwin Brindle', 'Nell Brindle'])
  })

  it('excludes review and rejected characters regardless of score', () => {
    expect(characterSpellingNames([
      person({ detection_status: 'review', detection_score: 1 }),
      person({ detection_status: 'rejected' }),
    ])).toEqual([])
  })

  it('trusts writer-created aliases and keeps legacy automatic aliases out until re-indexing', () => {
    expect(characterSpellingNames([person({ is_auto_detected: false })]))
      .toEqual(['Nelwin Brindle', 'Nelwin Brindls', 'Nell Brindle'])
    expect(characterSpellingNames([person({ spelling_aliases: undefined })])).toEqual(['Nelwin Brindle'])
    expect(characterSpellingNames([person({ spelling_aliases: [] })])).toEqual(['Nelwin Brindle'])
  })

  it('replaces the exception list when characters are removed or names change', () => {
    expect(characterSpellingNames([])).toEqual([])
    expect(characterSpellingNames([person({ name: 'Nelwin', spelling_aliases: [] })])).toEqual(['Nelwin'])
  })
})

it('keeps suspected spellings flagged and removes book exceptions without removing personal words', async () => {
  vi.stubGlobal('fetch', vi.fn(async (input: string) => ({
    ok: true,
    text: async () => input.endsWith('.aff') ? 'SET UTF-8\n' : input.endsWith('.dic') ? '1\nordinary\n' : '',
  })))
  try {
    await loadDictionary()
    setCustomWords(['Zelmora'])
    setIgnoredWords(characterSpellingNames([person()]))
    expect(checkWord('Brindle’s')).toBe(true)
    expect(checkWord('Nell')).toBe(true)
    expect(checkWord('Brindls')).toBe(false)
    setIgnoredWords(characterSpellingNames([person({ detection_status: 'rejected' })]))
    expect(checkWord('Brindle')).toBe(false)
    expect(checkWord('Zelmora')).toBe(true)
  } finally {
    setIgnoredWords([])
    setCustomWords([])
    vi.unstubAllGlobals()
  }
})
