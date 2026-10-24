import affData from '../../public/dictionaries/en_US.aff?raw'
import dicData from '../../public/dictionaries/en_US.dic?raw'
import gbAffData from '../../public/dictionaries/en_GB.aff?raw'
import gbDicData from '../../public/dictionaries/en_GB.dic?raw'
import caAffData from '../../public/dictionaries/en_CA.aff?raw'
import caDicData from '../../public/dictionaries/en_CA.dic?raw'
import auAffData from '../../public/dictionaries/en_AU.aff?raw'
import auDicData from '../../public/dictionaries/en_AU.dic?raw'
import supplementData from '../../public/dictionaries/en-supplement.txt?raw'
import Typo from 'typo-js'
import { describe, expect, it, vi } from 'vitest'
import {
  checkWord,
  getCaseCorrection,
  getDictionaryRoot,
  getImmediateSuggestions,
  getSuggestions,
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
          : "5\ncouldn't\nwouldn't\nTuesday\nrose\nroses\n",
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

  it('keeps dictionary checks case-sensitive, whichever casing is seen first', async () => {
    await loadDictionary()
    // Checking the correct casing must not cache an answer for the wrong one,
    // and the wrong one must not poison the correct one afterwards.
    expect(checkWord('Tuesday')).toBe(true)
    expect(checkWord('tuesday')).toBe(false)
    expect(checkWord('Tuesday')).toBe(true)
    expect(checkWord('tuesday')).toBe(false)
  })

  it('flags a confirmed name wearing a bare s and offers the possessive first', async () => {
    await loadDictionary()
    setIgnoredWords(['Mara Ionescu'])
    try {
      expect(checkWord('Ionescu')).toBe(true)
      expect(checkWord('Ionescu’s')).toBe(true)
      expect(checkWord('Ionescus')).toBe(false)
      expect(getImmediateSuggestions('Ionescus')[0]).toBe("Ionescu's")
      expect(await getSuggestions('Ionescus')).toContain("Ionescu's")
    } finally {
      setIgnoredWords([])
    }
  })

  it('keeps a real word that happens to shadow a name plus s', async () => {
    await loadDictionary()
    setIgnoredWords(['Rose'])
    try {
      expect(checkWord('roses')).toBe(true)
      expect(checkWord('Roses')).toBe(true)
    } finally {
      setIgnoredWords([])
    }
  })

  it('offers the accepted casing as the first suggestion for a miscapitalized word', async () => {
    await loadDictionary()
    expect(getCaseCorrection('tuesday')).toBe('Tuesday')
    expect(getCaseCorrection('Tuesday')).toBe(null)
    expect(getImmediateSuggestions('tuesday')[0]).toBe('Tuesday')
    expect(await getSuggestions('tuesday')).toContain('Tuesday')
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

  it('is case-sensitive about proper nouns', () => {
    for (const word of ['Tuesday', 'January', 'Paris']) {
      expect(dictionary.check(word), word).toBe(true)
      expect(dictionary.check(word.toLocaleLowerCase()), word.toLocaleLowerCase()).toBe(false)
    }
  })
})

describe('bundled en_GB dictionary', () => {
  const dictionary = new Typo('en_GB', gbAffData, gbDicData)

  it('accepts British spellings, -ise and Oxford -ize both', () => {
    for (const word of [
      'colour', 'honour', 'travelling', 'fibre', 'centre', 'grey', 'whisky',
      'realise', 'organise', 'apologise',
      'realize', 'organize', 'apologize',
    ]) {
      expect(dictionary.check(word), word).toBe(true)
    }
  })

  it('flags American spellings and real misspellings', () => {
    for (const word of ['color', 'honor', 'center', 'traveling', 'fiber', 'recieve', 'definately']) {
      expect(dictionary.check(word), word).toBe(false)
    }
  })
})

describe('bundled en_CA dictionary', () => {
  const dictionary = new Typo('en_CA', caAffData, caDicData)

  it('accepts Canadian spellings: British-style forms with -ize endings', () => {
    for (const word of ['colour', 'honour', 'centre', 'travelled', 'grey', 'whisky', 'realize', 'organize']) {
      expect(dictionary.check(word), word).toBe(true)
    }
  })

  it('flags -ise endings and real misspellings', () => {
    for (const word of ['realise', 'organise', 'traveling', 'recieve', 'definately']) {
      expect(dictionary.check(word), word).toBe(false)
    }
  })
})

describe('bundled en_AU dictionary', () => {
  const dictionary = new Typo('en_AU', auAffData, auDicData)

  it('accepts Australian spellings: British-style forms with -ise endings', () => {
    for (const word of ['colour', 'honour', 'centre', 'travelled', 'grey', 'whisky', 'realise', 'organise']) {
      expect(dictionary.check(word), word).toBe(true)
    }
  })

  it('flags -ize endings and real misspellings', () => {
    for (const word of ['realize', 'organize', 'center', 'traveling', 'recieve', 'definately']) {
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
