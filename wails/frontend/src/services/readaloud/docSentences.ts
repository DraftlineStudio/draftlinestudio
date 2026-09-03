// Maps sentence segmentation onto ProseMirror document positions. Follows the
// ChapterSearch pattern: walk textblocks, coalesce adjacent text nodes into
// runs (so bold/italic boundaries don't split a sentence), then segment each
// run. Block boundaries and hard breaks always end a sentence.

import type { Node as ProseMirrorNode } from '@tiptap/pm/model'
import { segmentText } from './segmentation'

export interface DocSentence {
  from: number
  to: number
  text: string
}

// collectSentences returns the spoken sentences of doc in order, as ProseMirror
// ranges plus their plain text. When a range is given, only sentences that
// overlap it are returned — a sentence straddling the range start is included
// from its true start, so playback from a mid-sentence cursor begins at the
// sentence's beginning.
export function collectSentences(doc: ProseMirrorNode, rangeFrom?: number, rangeTo?: number): DocSentence[] {
  const sentences: DocSentence[] = []

  doc.descendants((node, nodePosition) => {
    if (!node.isTextblock) return

    const runs: Array<{ text: string; from: number }> = []
    node.descendants((child, relativePosition) => {
      if (!child.isText || !child.text) return
      const from = nodePosition + 1 + relativePosition
      const previous = runs[runs.length - 1]
      if (previous && previous.from + previous.text.length === from) {
        previous.text += child.text
      } else {
        runs.push({ text: child.text, from })
      }
    })

    for (const run of runs) {
      for (const span of segmentText(run.text)) {
        const from = run.from + span.start
        const to = run.from + span.end
        if (rangeTo !== undefined && from >= rangeTo) continue
        if (rangeFrom !== undefined && to <= rangeFrom) continue
        const text = run.text.slice(span.start, span.end).trim()
        if (text) sentences.push({ from, to, text })
      }
    }
  })

  return sentences
}

// splitLeadClause splits a sentence at its first clause break (comma, dash,
// semicolon, colon) so playback can start on a short chunk instead of
// waiting for a long sentence to synthesize whole. Character offsets in the
// text map 1:1 onto document positions, so both halves keep exact highlight
// ranges. Returns the sentence unsplit when there is no break or either half
// would be too short to be worth speaking separately.
export function splitLeadClause(sentence: DocSentence): DocSentence[] {
  const match = /[,;:—–]\s+/.exec(sentence.text)
  if (!match) return [sentence]
  const cut = match.index + 1 // include the punctuation in the first chunk
  const restStart = match.index + match[0].length
  if (cut < 12 || sentence.text.length - restStart < 12) return [sentence]
  return [
    { from: sentence.from, to: sentence.from + cut, text: sentence.text.slice(0, cut) },
    { from: sentence.from + restStart, to: sentence.to, text: sentence.text.slice(restStart) },
  ]
}

// A generation unit: what the synthesizer is fed. Long sentences are split
// into clause-sized units so audio starts early and generation overlaps
// playback at fine grain; sentenceIndex maps every unit back to its
// sentence, whose FULL range the highlight covers while any of its units
// plays.
export interface GenerationUnit {
  text: string
  sentenceIndex: number
}

// Sentences longer than this many words are split at clause boundaries for
// generation. ~25 words ≈ 8s of speech — small enough that synthesis of one
// unit always fits inside the playback of its predecessors.
const MAX_UNIT_WORDS = 25
// The very first unit of a play is kept shorter still, so first audio lands
// fast.
const FIRST_UNIT_WORDS = 10

function wordCount(text: string): number {
  return text.split(/\s+/).filter(Boolean).length
}

// Splits one sentence's text into clause units of at most maxWords, cutting
// only at clause boundaries (comma, semicolon, colon, dash). A stretch with
// no boundary stays whole — a wrong-place cut sounds worse than latency.
function splitClauses(text: string, maxWords: number): string[] {
  if (wordCount(text) <= maxWords) return [text]
  const boundary = /[,;:—–]\s+/g
  const parts: string[] = []
  let start = 0
  let lastCut = 0
  let match: RegExpExecArray | null
  while ((match = boundary.exec(text)) !== null) {
    const candidateEnd = match.index + 1 // keep the punctuation
    if (wordCount(text.slice(start, candidateEnd)) >= maxWords) {
      parts.push(text.slice(start, lastCut > start ? lastCut : candidateEnd).trim() || text.slice(start, candidateEnd).trim())
      start = (lastCut > start ? lastCut : candidateEnd)
      while (start < text.length && /\s/.test(text[start])) start++
      lastCut = start
      boundary.lastIndex = start
    } else {
      lastCut = candidateEnd
    }
  }
  const tail = text.slice(start).trim()
  if (tail) parts.push(tail)
  return parts.length ? parts : [text]
}

// buildGenerationUnits flattens sentences into synthesis units. startIndex
// marks the first sentence that will actually be spoken — its first unit is
// cut extra short for sub-second first audio.
export function buildGenerationUnits(sentences: DocSentence[], startIndex: number): GenerationUnit[] {
  const units: GenerationUnit[] = []
  sentences.forEach((sentence, sentenceIndex) => {
    let pieces: string[]
    if (sentenceIndex === startIndex) {
      const lead = splitLeadClause(sentence)
      if (lead.length === 2 && wordCount(lead[0].text) <= FIRST_UNIT_WORDS + 4) {
        pieces = [lead[0].text, ...splitClauses(lead[1].text, MAX_UNIT_WORDS)]
      } else {
        pieces = splitClauses(sentence.text, MAX_UNIT_WORDS)
      }
    } else {
      pieces = splitClauses(sentence.text, MAX_UNIT_WORDS)
    }
    for (const text of pieces) {
      units.push({ text, sentenceIndex })
    }
  })
  return units
}

// firstUnitOfSentence returns the unit index where a sentence begins (for
// skip/jump, which operate on sentences while playback runs on units).
export function firstUnitOfSentence(units: GenerationUnit[], sentenceIndex: number): number {
  for (let i = 0; i < units.length; i++) {
    if (units[i].sentenceIndex >= sentenceIndex) return i
  }
  return -1
}

// sentenceIndexAt returns the index of the sentence containing pos, or the
// nearest following sentence; -1 when pos is after the last sentence.
export function sentenceIndexAt(sentences: DocSentence[], pos: number): number {
  for (let i = 0; i < sentences.length; i++) {
    if (pos < sentences[i].to) return i
  }
  return -1
}
