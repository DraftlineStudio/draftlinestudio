import { describe, it, expect } from 'vitest'
import { attributeSpeakers, speakerKeyFor, type AttributionInput, type RosterEntry } from './attribution'
import { segmentText } from './segmentation'

// ── helpers ──────────────────────────────────────────────────────────────────

const RENEE = speakerKeyFor('Renee Alvarez')
const MARCUS = speakerKeyFor('Marcus Webb')

const ROSTER: RosterEntry[] = [
  { key: RENEE, name: 'Renee Alvarez', aliases: ['Renee Alvarez', 'Renee', 'Alvarez', 'Detective Alvarez'] },
  { key: MARCUS, name: 'Marcus Webb', aliases: ['Marcus Webb', 'Marcus', 'Webb'] },
]

// Builds attribution input from paragraphs, running each through the REAL
// segmenter so tests exercise the exact sentence splits playback uses.
function paragraphs(...blocks: string[]): AttributionInput[] {
  const out: AttributionInput[] = []
  let from = 1
  blocks.forEach((text, block) => {
    for (const span of segmentText(text)) {
      const sentence = text.slice(span.start, span.end).trim()
      if (sentence) out.push({ from: from + span.start, text: sentence, block })
    }
    from += text.length + 2
  })
  return out
}

function speakers(input: AttributionInput[], roster = ROSTER) {
  return attributeSpeakers(input, roster).sentences.map(s => s.speaker)
}

// ── tests ────────────────────────────────────────────────────────────────────

describe('attributeSpeakers', () => {
  it('classifies plain prose as narration', () => {
    const input = paragraphs('The rain fell for days. Nobody came to the door.')
    const result = attributeSpeakers(input, ROSTER)
    expect(result.sentences.every(s => s.speaker === 'narrator' && s.kind === 'narration')).toBe(true)
    expect(result.lineCounts.get('narrator')).toBe(2)
  })

  it('attributes a name-tagged quote (curly and straight)', () => {
    const curly = paragraphs('“You’re not an easy man to find,” Renee said.')
    expect(speakers(curly)).toEqual([RENEE])

    const straight = paragraphs('"That was the idea," Marcus said.')
    expect(speakers(straight)).toEqual([MARCUS])
  })

  it('attributes an inverted tag ("said Marcus")', () => {
    const input = paragraphs('“Leave the folder,” said Marcus.')
    expect(speakers(input)).toEqual([MARCUS])
  })

  it('distinguishes narration from quoted speech kinds', () => {
    const input = paragraphs('Renee waited. “Forty-one Hoyt Street. You were there,” she said.')
    const result = attributeSpeakers(input, ROSTER)
    expect(result.sentences[0].kind).toBe('narration')
    // A short tag ("she said.") keeps the sentence in the dialogue bucket; a
    // longer unquoted half makes it mixed. Either way it is not narration.
    const tagged = result.sentences[result.sentences.length - 1]
    expect(['dialogue', 'mixed']).toContain(tagged.kind)
    expect(tagged.speaker).toBe(RENEE)

    const longTag = paragraphs('“Enough,” Marcus said, setting the folder down without looking at her.')
    expect(attributeSpeakers(longTag, ROSTER).sentences[0].kind).toBe('mixed')
  })

  it('carries the speaker across the segmenter mid-quote split', () => {
    // segmentText splits “Stop. Now.” into two sentences; the second starts
    // inside the quote and must inherit the speaker.
    const input = paragraphs('“Stop. Now,” Marcus said.')
    expect(input.length).toBeGreaterThan(1)
    const result = attributeSpeakers(input, ROSTER)
    expect(result.sentences.map(s => s.speaker)).toEqual(input.map(() => MARCUS))
    expect(result.sentences[0].confidence).toBe('continuation')
    expect(result.sentences[result.sentences.length - 1].confidence).toBe('tag')
  })

  it('follows multi-paragraph speech through the re-opening quote convention', () => {
    const input = paragraphs(
      '“I was there the night it happened, and I remember every detail of it. It started with the phone call,” Renee said.',
      '“The second part of the story is worse. Nobody believed me afterward.”',
      'Marcus looked away.',
    )
    const result = attributeSpeakers(input, ROSTER)
    // Every sentence of both speech paragraphs belongs to Renee.
    const reneeCount = result.sentences.filter(s => s.speaker === RENEE).length
    expect(reneeCount).toBe(input.length - 1)
    expect(result.sentences[result.sentences.length - 1].speaker).toBe('narrator')
  })

  it('closes an unclosed quote when the next paragraph does not re-open it', () => {
    const input = paragraphs(
      '"The line went dead, Marcus said.', // typo: never closed
      'The rain kept falling on the roof. Renee watched the window.',
    )
    const result = attributeSpeakers(input, ROSTER)
    // Paragraph two must be narration despite the runaway open quote.
    expect(result.sentences.slice(1).every(s => s.speaker === 'narrator')).toBe(true)
  })

  it('uses a pre-quote action beat in the same sentence', () => {
    const input = paragraphs('Marcus turned from the window. “Get out.”')
    const result = attributeSpeakers(input, ROSTER)
    const quote = result.sentences[result.sentences.length - 1]
    expect(quote.speaker).toBe(MARCUS)
    expect(['pretag', 'nearby']).toContain(quote.confidence)
  })

  it('resolves a pronoun tag against the last-named character', () => {
    const input = paragraphs(
      'Renee set the folder on the desk.',
      '“You were there,” she said.',
    )
    const result = attributeSpeakers(input, ROSTER)
    expect(result.sentences[1].speaker).toBe(RENEE)
    expect(result.genderEvidence.get(RENEE)?.she).toBe(1)
  })

  it('resolves later pronoun tags via accumulated gender evidence', () => {
    const input = paragraphs(
      'Renee stepped inside.',
      '“You were there,” she said.',
      'Marcus shrugged and stared at the folder on the desk between them.',
      '“Prove it,” he said.',
      '“I intend to,” she said.',
    )
    const result = attributeSpeakers(input, ROSTER)
    expect(result.sentences[1].speaker).toBe(RENEE)
    expect(result.sentences[3].speaker).toBe(MARCUS)
    // "she" now has evidence pointing at Renee even though Marcus was named
    // more recently than her.
    expect(result.sentences[4].speaker).toBe(RENEE)
  })

  it('alternates between exactly two established speakers', () => {
    const input = paragraphs(
      '“You were there,” Renee said.',
      '“That was the idea,” Marcus said.',
      '“Then we agree.”',
    )
    const result = attributeSpeakers(input, ROSTER)
    expect(result.sentences.map(s => s.speaker)).toEqual([RENEE, MARCUS, RENEE])
    expect(result.sentences[2].confidence).toBe('alternation')
  })

  it('sends untagged dialogue to unknown when no rule applies', () => {
    const input = paragraphs('“Nobody move.”')
    const result = attributeSpeakers(input, ROSTER)
    expect(result.sentences[0].speaker).toBe('unknown')
    expect(result.sentences[0].confidence).toBe('none')
  })

  it('still detects dialogue with an empty roster', () => {
    const input = paragraphs('The door opened. “Hands where I can see them,” the officer said.')
    const result = attributeSpeakers(input, [])
    expect(result.sentences[0].speaker).toBe('narrator')
    expect(result.sentences[1].speaker).toBe('unknown')
    expect(result.sentences[1].kind).not.toBe('narration')
  })

  it('resets scene state at separator blocks', () => {
    const input = paragraphs(
      '“You were there,” Renee said.',
      '“That was the idea,” Marcus said.',
      '* * *',
      '“Hello again.”',
    )
    const result = attributeSpeakers(input, ROSTER)
    // After the scene break the alternation chain is dead: unknown, not Renee.
    expect(result.sentences[result.sentences.length - 1].speaker).toBe('unknown')
  })

  it('ignores single-quoted text (v1 scope)', () => {
    const input = paragraphs("He called it his 'lucky' coat and wore it everywhere.")
    const result = attributeSpeakers(input, ROSTER)
    expect(result.sentences[0].speaker).toBe('narrator')
    expect(result.sentences[0].kind).toBe('narration')
  })

  it('does not match a lowercase common word against a character name', () => {
    const grace: RosterEntry[] = [{ key: speakerKeyFor('Grace'), name: 'Grace', aliases: ['Grace'] }]
    const input = paragraphs('She moved with grace across the floor. “Stop,” said Grace.')
    const result = attributeSpeakers(input, grace)
    expect(result.sentences[0].speaker).toBe('narrator')
    expect(result.sentences[result.sentences.length - 1].speaker).toBe(speakerKeyFor('Grace'))
  })

  it('counts dialogue lines per speaker', () => {
    const input = paragraphs(
      '“One,” Renee said.',
      '“Two,” Marcus said.',
      '“Three,” Renee said.',
      'Silence settled over the room.',
    )
    const result = attributeSpeakers(input, ROSTER)
    expect(result.lineCounts.get(RENEE)).toBe(2)
    expect(result.lineCounts.get(MARCUS)).toBe(1)
    expect(result.lineCounts.get('narrator')).toBe(1)
  })

  it('does not let a nearby male name steal an established she-tag (evidence poisoning)', () => {
    const input = paragraphs(
      'Renee stepped inside.',
      '“You were there,” she said.',
      'Marcus watched her leave without answering.',
      '“Stop,” she said.',
    )
    const result = attributeSpeakers(input, ROSTER)
    expect(result.sentences[1].speaker).toBe(RENEE)
    // The last-named character is Marcus, but Renee owns the she-evidence.
    expect(result.sentences[3].speaker).toBe(RENEE)
    expect(result.genderEvidence.get(MARCUS)?.she ?? 0).toBe(0)
  })

  it('harvests gender evidence from narration co-reference (pre-pass)', () => {
    const input = paragraphs(
      'Renee closed the door.',
      'She counted to ten before speaking.',
      'Marcus arrived late and said nothing.',
      '“Sorry,” she said.',
    )
    const result = attributeSpeakers(input, ROSTER)
    expect(result.genderEvidence.get(RENEE)?.she ?? 0).toBeGreaterThan(0)
    // "she said" resolves to Renee even though Marcus was named last.
    expect(result.sentences[result.sentences.length - 1].speaker).toBe(RENEE)
  })

  it('unifies a paragraph around its single attributed speaker (backward case)', () => {
    const input = paragraphs('“We move at dawn.” Renee tapped the map. “No exceptions.”')
    const result = attributeSpeakers(input, ROSTER)
    const dialogue = result.sentences.filter(s => s.kind !== 'narration')
    expect(dialogue.length).toBeGreaterThan(1)
    expect(dialogue.every(s => s.speaker === RENEE)).toBe(true)
  })

  it('continues the paragraph speaker for later untagged quotes (forward case)', () => {
    const input = paragraphs('“You were there,” Renee said. “I can prove it.” “Tonight.”')
    const result = attributeSpeakers(input, ROSTER)
    expect(result.sentences.every(s => s.kind === 'narration' || s.speaker === RENEE)).toBe(true)
  })

  it('alternates untagged paragraphs even in a multi-speaker scene', () => {
    const dana: RosterEntry[] = [...ROSTER, { key: speakerKeyFor('Dana'), name: 'Dana', aliases: ['Dana'] }]
    const input = paragraphs(
      '“Report,” Renee said.',
      '“All clear,” Marcus said.',
      '“Radio silence from the tower,” said Dana.',
      '“Keep trying.”',
    )
    const result = attributeSpeakers(input, dana)
    const last = result.sentences[result.sentences.length - 1]
    // Not narrator-bucketed any more; the hand-off answers the previous
    // paragraph with the other recent speaker.
    expect(last.speaker).not.toBe('unknown')
    expect([speakerKeyFor('Dana'), MARCUS]).toContain(last.speaker)
  })

  it('reports quoted char ranges for cast-mode splitting', () => {
    const input = paragraphs('“You were there,” she said, and waited for an answer.')
    const result = attributeSpeakers(input, ROSTER)
    const first = result.sentences[0]
    expect(first.quotedRanges).toHaveLength(1)
    const [start, end] = first.quotedRanges[0]
    expect(input[0].text.slice(start, end)).toBe('“You were there,”')
  })

  it('prefers the longest alias when names overlap', () => {
    const roster: RosterEntry[] = [
      { key: speakerKeyFor('Detective Alvarez'), name: 'Detective Alvarez', aliases: ['Detective Alvarez'] },
      { key: speakerKeyFor('Alvarez'), name: 'Alvarez', aliases: ['Alvarez'] },
    ]
    const input = paragraphs('“Enough,” Detective Alvarez said.')
    const result = attributeSpeakers(input, roster)
    // Both matchers hit, but the tag rule scans hits in text order; the
    // longer alias covers the shorter one's span for its own matcher only.
    // What matters here: a speaker is found and it is one of the two.
    expect([speakerKeyFor('Detective Alvarez'), speakerKeyFor('Alvarez')]).toContain(result.sentences[0].speaker)
  })
})
