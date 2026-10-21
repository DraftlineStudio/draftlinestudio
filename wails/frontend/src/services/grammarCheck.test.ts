import { describe, expect, it } from 'vitest'
import { analyzeGrammar, type GrammarIssue } from './grammarCheck'

function byRule(text: string, rule: string): GrammarIssue[] {
  return analyzeGrammar(text).filter(issue => issue.rule === rule)
}

function flaggedSpans(text: string, rule: string): string[] {
  return byRule(text, rule).map(issue => text.slice(issue.from, issue.to))
}

describe('sentence capitalization', () => {
  it('flags a lower-case word opening a paragraph', () => {
    const issues = byRule('the ferry left without her.', 'sentence-capitalization')
    expect(issues).toHaveLength(1)
    expect(issues[0].from).toBe(0)
    expect(issues[0].to).toBe(3)
    expect(issues[0].replacements).toEqual(['The'])
  })

  it('flags a lower-case word after a sentence-ending mark', () => {
    expect(flaggedSpans('The ferry left. she watched it go.', 'sentence-capitalization')).toEqual(['she'])
    expect(flaggedSpans('Was it too late? probably.', 'sentence-capitalization')).toEqual(['probably'])
    expect(flaggedSpans('Stop! come back here.', 'sentence-capitalization')).toEqual(['come'])
  })

  it('offers the capitalized word as the fix', () => {
    const [issue] = byRule('The ferry left. she watched it go.', 'sentence-capitalization')
    expect(issue.replacements).toEqual(['She'])
  })

  it('leaves dialogue tags alone', () => {
    expect(byRule('“Come back!” she shouted across the pier.', 'sentence-capitalization')).toEqual([])
    expect(byRule('“Where now?” he asked.', 'sentence-capitalization')).toEqual([])
  })

  it('leaves a pause after an ellipsis alone', () => {
    expect(byRule('She waited... nothing happened.', 'sentence-capitalization')).toEqual([])
  })

  it('leaves abbreviations and initials alone', () => {
    expect(byRule('They packed pens, ink, etc. and left at dawn.', 'sentence-capitalization')).toEqual([])
    expect(byRule('The train leaves at 6 p.m. tonight.', 'sentence-capitalization')).toEqual([])
    expect(byRule('The office sat at 4 Hemlock Ave. beside the bakery.', 'sentence-capitalization')).toEqual([])
  })

  it('leaves camel-case brands and alphanumeric words alone', () => {
    expect(byRule('She sold the lamp online. eBay took a cut.', 'sentence-capitalization')).toEqual([])
    expect(byRule('The player died. mp3s were obsolete anyway.', 'sentence-capitalization')).toEqual([])
  })

  it('leaves the pronoun to the pronoun rule, contractions included', () => {
    expect(byRule('The door opened. i stepped inside.', 'sentence-capitalization')).toEqual([])
    expect(flaggedSpans('i’m not ready.', 'pronoun-i')).toEqual(['i'])
    expect(byRule('i’m not ready.', 'sentence-capitalization')).toEqual([])
  })
})

describe('article agreement', () => {
  it('flags the wrong article before a vowel or consonant sound', () => {
    expect(byRule('She kept a umbrella by the door.', 'article-agreement')).toHaveLength(1)
    expect(byRule('He was an builder of small boats.', 'article-agreement')).toHaveLength(1)
  })

  it('treats a silent h as a vowel sound in both spelling conventions', () => {
    expect(byRule('It was an honor to serve.', 'article-agreement')).toEqual([])
    expect(byRule('It was an honour to serve.', 'article-agreement')).toEqual([])
    expect(byRule('They waited an hour in an honest queue.', 'article-agreement')).toEqual([])
  })
})

describe('pronoun capitalization', () => {
  it('flags a standalone lower-case i and offers I', () => {
    const issues = byRule('She said i could stay until morning.', 'pronoun-i')
    expect(issues).toHaveLength(1)
    expect(issues[0].replacements).toEqual(['I'])
    expect(flaggedSpans('She said i could stay until morning.', 'pronoun-i')).toEqual(['i'])
  })

  it('flags the i that opens a contraction', () => {
    expect(flaggedSpans('Tomorrow i’ll pack, and i’ve said so.', 'pronoun-i')).toEqual(['i', 'i'])
    expect(flaggedSpans("Honestly i'm done, i'd say.", 'pronoun-i')).toEqual(['i', 'i'])
  })

  it('flags i at the end of a sentence or before punctuation', () => {
    expect(flaggedSpans('It was i.', 'pronoun-i')).toEqual(['i'])
    expect(flaggedSpans('Can i come along, or should i wait?', 'pronoun-i')).toEqual(['i', 'i'])
  })

  it('leaves i inside words, references, and hyphenations alone', () => {
    expect(byRule('The argument held, i.e. nobody moved.', 'pronoun-i')).toEqual([])
    expect(byRule('The i-beam carried the whole floor.', 'pronoun-i')).toEqual([])
    expect(byRule('A pi symbol marked the page.', 'pronoun-i')).toEqual([])
  })
})
