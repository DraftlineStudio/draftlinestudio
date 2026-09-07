import affData from '../../public/dictionaries/en_US.aff?raw'
import dicData from '../../public/dictionaries/en_US.dic?raw'
import Typo from 'typo-js'
import { describe, expect, it, vi } from 'vitest'
import {
  checkWord,
  getDictionaryRoot,
  getImmediateSuggestions,
  loadDictionary,
  normalizeCustomDictionary,
  normalizeIgnoredWords,
  normalizeSpellWord,
  setIgnoredWords,
} from './spellCheck'

describe('spell-check word normalization', () => {
  it('canonicalizes typographic apostrophes before dictionary lookup', () => {
    expect(normalizeSpellWord('couldn’t')).toBe("couldn't")
    expect(normalizeSpellWord('wouldn‘t')).toBe("wouldn't")
    expect(normalizeSpellWord('didnʼt')).toBe("didn't")
    expect(normalizeSpellWord('isn＇t')).toBe("isn't")
  })

  it('keeps dictionary roots independent of apostrophe typography', () => {
    expect(getDictionaryRoot('FleetCom’s')).toBe('FleetCom')
    expect(getDictionaryRoot("FleetCom's")).toBe('FleetCom')
    expect(normalizeCustomDictionary(['FleetCom’s', "FleetCom's"])).toEqual(['FleetCom'])
  })

  it('treats accented Latin words as whole words, not split at the diacritic', () => {
    expect(getDictionaryRoot('café’s')).toBe('café')
    expect(getDictionaryRoot('"naïve,"')).toBe('naïve')
    expect(normalizeIgnoredWords(['Señora Peña'])).toEqual(['Peña', 'Señora'])
  })

  it('expands confirmed multi-word names and aliases into transient words', () => {
    expect(new Set(normalizeIgnoredWords([
      'Mara Ionescu',
      'Detective Hanlon’s',
      'O’Brien',
    ]))).toEqual(new Set(['Mara', 'Ionescu', 'Detective', 'Hanlon', "O'Brien"]))
  })

  it('checks smart-apostrophe contractions and project names against canonical forms', async () => {
    const originalFetch = globalThis.fetch
    globalThis.fetch = vi.fn(async (input: string | URL | Request) => ({
      ok: true,
      text: async () => String(input).endsWith('.aff')
        ? 'SET UTF-8\n'
        : "2\ncouldn't\nwouldn't\n",
    })) as unknown as typeof fetch

    try {
      await loadDictionary()
      setIgnoredWords(['Mara Ionescu'])
      expect(checkWord('couldn’t')).toBe(true)
      expect(checkWord('wouldn‘t')).toBe(true)
      expect(checkWord('Mara')).toBe(true)
      expect(checkWord('Ionescu’s')).toBe(true)
      expect(getImmediateSuggestions('couldn’t')).not.toContain("couldn't")
    } finally {
      globalThis.fetch = originalFetch
      setIgnoredWords([])
    }
  })
})

describe('bundled dictionary vocabulary', () => {
  const dictionary = new Typo('en_US', affData, dicData)

  it('accepts professional and common-variant vocabulary (SCOWL size 70, variant 2)', () => {
    for (const word of [
      'responder', 'responders', 'lockdown', 'takedown', 'bystanders',
      'grey', 'whisky', 'cancelled', 'towards',
      'café', 'naïve', 'fiancée',
    ]) {
      expect(dictionary.check(word), word).toBe(true)
    }
  })

  it('still flags real misspellings', () => {
    for (const word of ['recieve', 'teh', 'definately', 'xyzzt']) {
      expect(dictionary.check(word), word).toBe(false)
    }
  })
})
