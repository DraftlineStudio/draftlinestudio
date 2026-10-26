import { describe, expect, it } from 'vitest'
import { analyzeGrammar, analyzeTypography, setGrammarCheckEnabled, setStyleCheckEnabled, type GrammarIssue } from './grammarCheck'

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

describe('missing apostrophes', () => {
  it('flags apostrophe-less contractions, the dictionary words im and ive included', () => {
    expect(flaggedSpans('If you fall im going too.', 'missing-apostrophe')).toEqual(['im'])
    expect(flaggedSpans('Honestly ive had worse mornings.', 'missing-apostrophe')).toEqual(['ive'])
    expect(flaggedSpans('He dont care and youre next.', 'missing-apostrophe')).toEqual(['dont', 'youre'])
  })

  it('preserves the typed capitalization in the fix', () => {
    expect(byRule('Dont wait up.', 'missing-apostrophe')[0].replacements).toEqual(["Don’t"])
    expect(byRule('im not ready.', 'missing-apostrophe')[0].replacements).toEqual(["I’m"])
  })

  it('leaves acronyms, shouting, and containing words alone', () => {
    expect(byRule('She sent an IM from the lobby.', 'missing-apostrophe')).toEqual([])
    expect(byRule('The sign read DONT WALK.', 'missing-apostrophe')).toEqual([])
    expect(byRule('A slim grim rim.', 'missing-apostrophe')).toEqual([])
  })
})

describe('its versus it’s', () => {
  it('flags its before a determiner or verb form as grammar', () => {
    const issues = byRule('I think its been a long day.', 'its-contraction')
    expect(issues).toHaveLength(1)
    expect(issues[0].kind).toBe('grammar')
    expect(issues[0].replacements).toEqual(["it’s"])
    expect(flaggedSpans('Its not the heat.', 'its-contraction')).toEqual(['Its'])
    expect(byRule('Its not the heat.', 'its-contraction')[0].replacements).toEqual(["It’s"])
  })

  it('flags its before a gerund as a softer style question', () => {
    const issues = byRule('Maybe its raining there too.', 'its-contraction')
    expect(issues).toHaveLength(1)
    expect(issues[0].kind).toBe('style')
  })

  it('leaves the possessive alone', () => {
    expect(byRule('The dog wagged its tail.', 'its-contraction')).toEqual([])
    expect(byRule('We admired its opening scene and feared its sting.', 'its-contraction')).toEqual([])
    expect(byRule('The charity kept its not-for-profit status.', 'its-contraction')).toEqual([])
  })
})

describe('were versus we’re', () => {
  it('flags sentence-opening Were before not or gonna', () => {
    expect(flaggedSpans('Were not lost. Were gonna be fine.', 'were-contraction')).toEqual(['Were', 'Were'])
    expect(byRule('Were not lost.', 'were-contraction')[0].replacements).toEqual(["We’re"])
    expect(flaggedSpans('“Were not done here,” she warned.', 'were-contraction')).toEqual(['Were'])
  })

  it('leaves questions, inversions, and the plain past tense alone', () => {
    expect(byRule('Were you there when it happened?', 'were-contraction')).toEqual([])
    expect(byRule('Were all the doors locked?', 'were-contraction')).toEqual([])
    expect(byRule('They were not ready for the storm.', 'were-contraction')).toEqual([])
  })
})

describe('dialogue tag punctuation', () => {
  it('turns the period before a tag into a comma', () => {
    const issues = byRule('“Leave the lantern.” Marcus said.', 'dialogue-tag-punctuation')
    expect(issues).toHaveLength(1)
    expect(issues[0].replacements).toEqual([',” Marcus said'])
  })

  it('lowercases a bare pronoun in the tag', () => {
    expect(byRule('“Leave it.” He said.', 'dialogue-tag-punctuation')[0].replacements)
      .toEqual([',” he said'])
    expect(byRule('“Leave it!” He shouted.', 'dialogue-tag-punctuation')[0].replacements)
      .toEqual(['!” he shouted'])
  })

  it('leaves a new sentence after the quote alone', () => {
    expect(byRule('“Go home.” Marcus said nothing more that night.', 'dialogue-tag-punctuation')).toEqual([])
    expect(byRule('“Go home.” He asked the ferryman for passage.', 'dialogue-tag-punctuation')).toEqual([])
  })

  it('leaves a correctly punctuated tag alone', () => {
    expect(byRule('“Leave the lantern,” Marcus said.', 'dialogue-tag-punctuation')).toEqual([])
    expect(byRule('“Where now?” he asked.', 'dialogue-tag-punctuation')).toEqual([])
  })
})

describe('capitalization after a comma', () => {
  it('flags an adverb that is never a proper noun', () => {
    const issues = byRule('Relax, Just breathe through it.', 'comma-capitalization')
    expect(issues).toHaveLength(1)
    expect(issues[0].replacements).toEqual(['just'])
  })

  it('leaves names and ordinary commas alone', () => {
    expect(byRule('Later, Marcus found the trail.', 'comma-capitalization')).toEqual([])
    expect(byRule('Relax, just breathe through it.', 'comma-capitalization')).toEqual([])
  })
})

describe('straight quotes', () => {
  it('curls double quotes by direction', () => {
    const issues = byRule('"Morning," she said. "Already?"', 'straight-quote')
    expect(issues.map(issue => issue.replacements[0])).toEqual(['“', '”', '“', '”'])
  })

  it('treats a quote after an opening bracket or dash as opening', () => {
    expect(byRule('He wrote ("loosely") in the margin.', 'straight-quote')
      .map(issue => issue.replacements[0])).toEqual(['“', '”'])
  })

  it('curls apostrophes inside and after words', () => {
    expect(byRule("It wasn't the dogs' fault.", 'straight-quote')
      .map(issue => issue.replacements[0])).toEqual(['’', '’'])
  })

  it('knows elisions and decades from opening single quotes', () => {
    expect(byRule("Back in '92 they called 'em heroes.", 'straight-quote')
      .map(issue => issue.replacements[0])).toEqual(['’', '’'])
    expect(byRule("She hummed 'some old tune' twice.", 'straight-quote')
      .map(issue => issue.replacements[0])).toEqual(['‘', '’'])
  })

  it('leaves curly quotes alone', () => {
    expect(byRule('“Morning,” she said. It wasn’t late.', 'straight-quote')).toEqual([])
  })
})

describe('spacing', () => {
  it('flags doubled spaces between words', () => {
    const issues = byRule('The door  closed.', 'double-space')
    expect(issues).toHaveLength(1)
    expect(issues[0].replacements).toEqual([' '])
  })

  it('flags a space before punctuation and offers removal', () => {
    const issues = byRule('Wait , what ?', 'space-before-punctuation')
    expect(issues).toHaveLength(2)
    expect(issues[0].replacements).toEqual([''])
  })

  it('leaves spaced ellipses and single spaces alone', () => {
    expect(byRule('She waited . . . then knocked.', 'space-before-punctuation')).toEqual([])
    expect(byRule('The door closed.', 'double-space')).toEqual([])
  })
})

describe('double hyphens', () => {
  it('offers an em dash for a typed double hyphen', () => {
    const issues = byRule('He paused--then ran.', 'double-hyphen')
    expect(issues).toHaveLength(1)
    expect(issues[0].replacements).toEqual(['—'])
  })

  it('leaves longer rules and single hyphens alone', () => {
    expect(byRule('A well-known divider: ----', 'double-hyphen')).toEqual([])
  })
})

describe('analyzeTypography', () => {
  it('returns only typography fixes, in order', () => {
    const issues = analyzeTypography('He paused--then "stopped" and waited...')
    expect(issues.map(issue => issue.rule))
      .toEqual(['double-hyphen', 'straight-quote', 'straight-quote', 'ellipsis'])
  })

  it('works while the live grammar check is off', () => {
    setGrammarCheckEnabled(false)
    try {
      expect(analyzeGrammar('its been "quiet" here').map(issue => issue.kind))
        .toEqual(['style', 'style'])
      expect(analyzeTypography('"quiet"')).toHaveLength(2)
    } finally {
      setGrammarCheckEnabled(true)
    }
  })
})

describe('the style toggle', () => {
  it('silences style findings while grammar findings survive, and back', () => {
    const text = 'its been "quiet" here'
    setStyleCheckEnabled(false)
    try {
      const kinds = analyzeGrammar(text).map(issue => issue.kind)
      expect(kinds).toContain('grammar')
      expect(kinds).not.toContain('style')
    } finally {
      setStyleCheckEnabled(true)
    }
    expect(analyzeGrammar(text).map(issue => issue.kind)).toContain('style')
  })

  it('returns nothing when both checks are off', () => {
    setGrammarCheckEnabled(false)
    setStyleCheckEnabled(false)
    try {
      expect(analyzeGrammar('its been "quiet" here')).toEqual([])
    } finally {
      setGrammarCheckEnabled(true)
      setStyleCheckEnabled(true)
    }
  })
})

describe('end-mark conflicts and ellipses', () => {
  it('flags a period trailing a stronger mark', () => {
    const issues = byRule('You are going to regret this!.', 'conflicting-end-marks')
    expect(issues).toHaveLength(1)
    expect(issues[0].replacements).toEqual(['!'])
    expect(flaggedSpans('Really?.', 'conflicting-end-marks')).toEqual(['?.'])
  })

  it('leaves an ellipsis after a mark alone', () => {
    expect(byRule('No!... not like this.', 'conflicting-end-marks')).toEqual([])
  })

  it('offers the ellipsis character for typed periods, as style', () => {
    const issues = byRule('She waited... nothing happened.', 'ellipsis')
    expect(issues).toHaveLength(1)
    expect(issues[0].kind).toBe('style')
    expect(issues[0].replacements).toEqual(['…'])
    expect(byRule('She waited… nothing happened.', 'ellipsis')).toEqual([])
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
