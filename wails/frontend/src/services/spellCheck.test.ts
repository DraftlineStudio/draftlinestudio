import affData from '../../public/dictionaries/en_US.aff?raw'
import dicData from '../../public/dictionaries/en_US.dic?raw'
import supplementData from '../../public/dictionaries/en_US-supplement.txt?raw'
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
  parseSupplement,
  setIgnoredWords,
} from './spellCheck'
import { buildWordBuckets } from './spellSuggestions'

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
        : String(input).endsWith('.txt')
          ? '# supplement\nGlock\n'
          : "2\ncouldn't\nwouldn't\n",
    })) as unknown as typeof fetch

    try {
      await loadDictionary()
      setIgnoredWords(['Mara Ionescu'])
      expect(checkWord('couldn’t')).toBe(true)
      expect(checkWord('wouldn‘t')).toBe(true)
      expect(checkWord('Mara')).toBe(true)
      expect(checkWord('Ionescu’s')).toBe(true)
      // Bundled supplement words are accepted in any case, possessives included.
      expect(checkWord('Glock')).toBe(true)
      expect(checkWord('glock’s')).toBe(true)
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

describe('bundled dictionary supplement', () => {
  const dictionary = new Typo('en_US', affData, dicData)
  const supplement = parseSupplement(supplementData)

  it('carries the contemporary vocabulary SCOWL lacks', () => {
    for (const word of ['glock', 'lockpick', 'lockpicks', 'deco', 'flashbang', 'passcode', 'wi', 'fi', 'eotech', 'holosun', 'sightmark', 'primaryarms']) {
      expect(supplement, word).toContain(word)
    }
  })

  it('lists only words the base dictionary rejects, so upgrades prune it honestly', () => {
    const redundant = supplement.filter(word => dictionary.check(word))
    expect(redundant).toEqual([])
  })
})

// The suggestion index is built in a worker, so it mirrors the tokenizer's
// apostrophe set rather than importing it. A word the tokenizer can produce
// but the index rejects is a word that gets underlined and then offered
// nothing — and the editor inserts the curly apostrophe, not the straight one.
describe('the suggestion index accepts what the tokenizer produces', () => {
  const tokenize = (text: string) => text.match(/\p{Script=Latin}+(?:['’‘ʼ＇]\p{Script=Latin}+)*/gu) ?? []

  it('indexes words carrying every apostrophe the tokenizer allows', () => {
    const words = ["couldn't", 'couldn’t', 'couldn‘t', 'couldnʼt', 'couldn＇t']
    for (const word of words) {
      expect(tokenize(word)).toEqual([word])
      const buckets = buildWordBuckets([word])
      expect([...buckets.values()].flat(), `${word} was dropped from the index`)
        .toEqual([word.toLocaleLowerCase()])
    }
  })

  it('still indexes hyphenated and plain entries', () => {
    for (const word of ['mother-in-law', 'tesseract']) {
      const buckets = buildWordBuckets([word])
      expect([...buckets.values()].flat()).toEqual([word])
    }
  })

  it('still refuses entries that are not words', () => {
    for (const entry of ['', 'a', '123', '<p>', 'two words']) {
      expect([...buildWordBuckets([entry]).values()].flat()).toEqual([])
    }
  })
})
