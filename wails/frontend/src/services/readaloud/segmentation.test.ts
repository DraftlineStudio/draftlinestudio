import { describe, it, expect } from 'vitest'
import { Schema } from '@tiptap/pm/model'
import { segmentText } from './segmentation'
import { collectSentences, sentenceIndexAt, splitLeadClause, buildGenerationUnits, firstUnitOfSentence } from './docSentences'

function sentencesOf(text: string): string[] {
  return segmentText(text).map(s => text.slice(s.start, s.end).trim())
}

describe('segmentText', () => {
  it('splits plain declarative sentences', () => {
    expect(sentencesOf('The rain stopped. The street glistened. Nobody moved.')).toEqual([
      'The rain stopped.', 'The street glistened.', 'Nobody moved.',
    ])
  })

  it('keeps text without a terminator as one sentence', () => {
    expect(sentencesOf('A fragment without an ending')).toEqual(['A fragment without an ending'])
  })

  it('handles question and exclamation clusters', () => {
    expect(sentencesOf('You did what?! Never again. Really?')).toEqual([
      'You did what?!', 'Never again.', 'Really?',
    ])
  })

  describe('dialogue quotes', () => {
    it('keeps attribution after a quoted exclamation in one sentence', () => {
      expect(sentencesOf('"Hello!" she said.')).toEqual(['"Hello!" she said.'])
    })

    it('splits after a quote-final period when a new sentence follows', () => {
      expect(sentencesOf('She said, "Go away." Then he left.')).toEqual([
        'She said, "Go away."', 'Then he left.',
      ])
    })

    it('keeps attribution after a quoted question in one sentence', () => {
      expect(sentencesOf('"Where are you going?" asked Tom.')).toEqual([
        '"Where are you going?" asked Tom.',
      ])
    })

    it('handles curly quotes', () => {
      expect(sentencesOf('“It’s over.” He believed it.')).toEqual([
        '“It’s over.”', 'He believed it.',
      ])
    })

    it('splits between a closing quote and a new quoted sentence', () => {
      expect(sentencesOf('"Stop." "Why should I?"')).toEqual(['"Stop."', '"Why should I?"'])
    })
  })

  describe('abbreviations and initials', () => {
    it('does not split after honorifics', () => {
      expect(sentencesOf("Mr. Darcy went to St. James's Park.")).toEqual([
        "Mr. Darcy went to St. James's Park.",
      ])
    })

    it('does not split inside initials', () => {
      expect(sentencesOf('J. R. R. Tolkien wrote it. Everyone read it.')).toEqual([
        'J. R. R. Tolkien wrote it.', 'Everyone read it.',
      ])
    })

    it('does not split after i.e. and e.g.', () => {
      expect(sentencesOf('Use a colon, i.e. like this. Fine.')).toEqual([
        'Use a colon, i.e. like this.', 'Fine.',
      ])
    })

    it('does not split No. before a number but does split the word no', () => {
      expect(sentencesOf('See No. 5 on the list. It matters.')).toEqual([
        'See No. 5 on the list.', 'It matters.',
      ])
      expect(sentencesOf('The answer was no. She left.')).toEqual([
        'The answer was no.', 'She left.',
      ])
    })

    it('does not split decimals', () => {
      expect(sentencesOf('It cost 3.50 dollars. Cheap.')).toEqual([
        'It cost 3.50 dollars.', 'Cheap.',
      ])
    })
  })

  describe('ellipses', () => {
    it('continues when an ellipsis trails into lowercase', () => {
      expect(sentencesOf('He paused… then kept walking.')).toEqual([
        'He paused… then kept walking.',
      ])
    })

    it('splits when an ellipsis is followed by a capital', () => {
      expect(sentencesOf('He paused… Then he ran.')).toEqual(['He paused…', 'Then he ran.'])
    })

    it('treats three dots like an ellipsis', () => {
      expect(sentencesOf('Maybe... maybe not. Decide.')).toEqual([
        'Maybe... maybe not.', 'Decide.',
      ])
    })
  })

  describe('em dashes', () => {
    it('never ends a sentence at an em dash', () => {
      expect(sentencesOf('She reached for the door—the handle was gone. Panic.')).toEqual([
        'She reached for the door—the handle was gone.', 'Panic.',
      ])
    })

    it('keeps an interrupted line together until real punctuation', () => {
      expect(sentencesOf('"I only meant—" "Enough!"')).toEqual(['"I only meant—" "Enough!"'])
    })
  })

  it('does not split on a missing space after a period', () => {
    expect(sentencesOf('End.Next still one sentence. Fine.')).toEqual([
      'End.Next still one sentence.', 'Fine.',
    ])
  })

  it('attaches a lone capital-and-period forward as an initial', () => {
    expect(sentencesOf('It was late. A. Very late indeed.')).toEqual([
      'It was late.', 'A. Very late indeed.',
    ])
  })

  it('merges unspeakably short fragments into the previous sentence', () => {
    expect(sentencesOf('Wait. 5. Go on now.')).toEqual(['Wait. 5.', 'Go on now.'])
  })

  it('returns character spans that index the original string', () => {
    const text = '  One.  Two.'
    const spans = segmentText(text)
    expect(spans).toHaveLength(2)
    expect(text.slice(spans[0].start, spans[0].end)).toBe('One.')
    expect(text.slice(spans[1].start, spans[1].end)).toBe('Two.')
  })
})

// A minimal schema mirroring the editor's block/text structure.
const schema = new Schema({
  nodes: {
    doc: { content: 'block+' },
    paragraph: { group: 'block', content: 'inline*' },
    text: { group: 'inline' },
  },
  marks: {
    bold: {},
  },
})

function paragraph(...children: Array<ReturnType<typeof schema.text>>) {
  return schema.node('paragraph', null, children)
}

describe('collectSentences', () => {
  it('maps sentences to ProseMirror positions across marks', () => {
    // "A bold word here. Another one." with "bold" marked — the mark boundary
    // must not split the sentence.
    const doc = schema.node('doc', null, [
      paragraph(
        schema.text('A '),
        schema.text('bold', [schema.mark('bold')]),
        schema.text(' word here. Another one.'),
      ),
    ])
    const sentences = collectSentences(doc)
    expect(sentences.map(s => s.text)).toEqual(['A bold word here.', 'Another one.'])
    // Positions must slice the doc back to the same text.
    for (const s of sentences) {
      expect(doc.textBetween(s.from, s.to)).toBe(s.text)
    }
  })

  it('ends sentences at block boundaries even without punctuation', () => {
    const doc = schema.node('doc', null, [
      paragraph(schema.text('A heading without punctuation')),
      paragraph(schema.text('Body text follows. It continues.')),
    ])
    expect(collectSentences(doc).map(s => s.text)).toEqual([
      'A heading without punctuation',
      'Body text follows.',
      'It continues.',
    ])
  })

  it('clamps to a range but keeps a straddling sentence whole', () => {
    const doc = schema.node('doc', null, [
      paragraph(schema.text('First sentence here. Second sentence here. Third one.')),
    ])
    const all = collectSentences(doc)
    expect(all).toHaveLength(3)
    // Range starting mid-second-sentence: second and third included, second whole.
    const mid = all[1].from + 5
    const fromMid = collectSentences(doc, mid)
    expect(fromMid.map(s => s.text)).toEqual(['Second sentence here.', 'Third one.'])
    expect(fromMid[0].from).toBe(all[1].from)
    // Range covering only the first sentence.
    const firstOnly = collectSentences(doc, undefined, all[0].to)
    expect(firstOnly.map(s => s.text)).toEqual(['First sentence here.'])
  })

  it('splits a long lead sentence at its first clause with exact positions', () => {
    const text = 'When the rain finally stopped, the whole town came out to see the damage.'
    const sentence = { from: 100, to: 100 + text.length, text }
    const parts = splitLeadClause(sentence)
    expect(parts).toHaveLength(2)
    expect(parts[0].text).toBe('When the rain finally stopped,')
    expect(parts[1].text).toBe('the whole town came out to see the damage.')
    expect(parts[0].from).toBe(100)
    expect(parts[0].to).toBe(100 + 'When the rain finally stopped,'.length)
    expect(parts[1].to).toBe(sentence.to)
    // Positions still index the same characters.
    expect(text.slice(parts[1].from - 100, parts[1].to - 100)).toBe(parts[1].text)
  })

  it('leaves short or unbreakable sentences whole', () => {
    expect(splitLeadClause({ from: 0, to: 9, text: 'Run, now!' })).toHaveLength(1)
    expect(splitLeadClause({ from: 0, to: 26, text: 'No clause breaks in here..' })).toHaveLength(1)
  })

  it('keeps ordinary multi-clause sentences whole for natural prosody', () => {
    const long = 'When the rain finally stopped falling over the ruined harbor town, the people came out slowly from their basements and doorways, blinking at the grey morning light, and nobody said a single word about the night before.'
    const short = 'Nobody slept.'
    const sentences = [
      { from: 0, to: long.length, text: long },
      { from: long.length + 1, to: long.length + 1 + short.length, text: short },
    ]
    const units = buildGenerationUnits(sentences, 0)
    // Splitting normal prose at commas makes Kokoro reset its cadence.
    const firstSentenceUnits = units.filter(u => u.sentenceIndex === 0)
    expect(firstSentenceUnits).toEqual([{ text: long, sentenceIndex: 0 }])
    expect(units.filter(u => u.sentenceIndex === 1)).toHaveLength(1)
    // Unit lookup for sentence-level skip/jump.
    expect(firstUnitOfSentence(units, 0)).toBe(0)
    expect(firstUnitOfSentence(units, 1)).toBe(firstSentenceUnits.length)
    expect(firstUnitOfSentence(units, 2)).toBe(-1)
  })

  it('does not sacrifice the opening sentence cadence for a faster fragment', () => {
    const text = 'When the rain stopped, the whole town came out to see what was left of the harbor and the boats.'
    const units = buildGenerationUnits([{ from: 0, to: text.length, text }], 0)
    expect(units).toEqual([{ text, sentenceIndex: 0 }])
  })

  it('leaves an unbreakable long sentence whole rather than cutting mid-clause', () => {
    const text = 'word '.repeat(40).trim() + '.'
    const units = buildGenerationUnits([{ from: 0, to: text.length, text }], 5)
    expect(units).toHaveLength(1)
  })

  it('finds the sentence containing a position', () => {
    const doc = schema.node('doc', null, [
      paragraph(schema.text('One here. Two here. Three here.')),
    ])
    const sentences = collectSentences(doc)
    expect(sentenceIndexAt(sentences, sentences[1].from + 2)).toBe(1)
    expect(sentenceIndexAt(sentences, 0)).toBe(0)
    expect(sentenceIndexAt(sentences, sentences[2].to + 1)).toBe(-1)
  })
})
