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
  // Index of the textblock (paragraph) this sentence lives in, counted over
  // the whole document walk. Optional because synthetic sentences (clause
  // splits) don't carry it; speaker attribution needs paragraph boundaries.
  block?: number
}

// collectSentences returns the spoken sentences of doc in order, as ProseMirror
// ranges plus their plain text. When a range is given, only sentences that
// overlap it are returned — a sentence straddling the range start is included
// from its true start, so playback from a mid-sentence cursor begins at the
// sentence's beginning.
export function collectSentences(doc: ProseMirrorNode, rangeFrom?: number, rangeTo?: number): DocSentence[] {
  const sentences: DocSentence[] = []
  let block = -1

  doc.descendants((node, nodePosition) => {
    if (!node.isTextblock) return
    // One id per textblock — every run inside it shares the paragraph.
    block++

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
        if (text) sentences.push({ from, to, text, block })
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

// A generation unit: what the synthesizer is fed. Ordinary sentences remain
// whole so Kokoro retains their cadence and clause-level prosody. Only an
// exceptionally long sentence is split as a tokenizer/latency safeguard;
// sentenceIndex maps each resulting unit back to the full highlighted range.
export interface GenerationUnit {
  text: string
  sentenceIndex: number
  // Cast mode only: true when this unit is quoted speech (character voice),
  // false for narration/tags/asides (narrator voice). Absent on units built
  // without cast splitting — those are voiced per sentence.
  quoted?: boolean
}

// This is intentionally high. Splitting normal prose at commas makes each
// clause sound like a fresh utterance and damages the author's intended pace.
const MAX_UNIT_WORDS = 80

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

// buildGenerationUnits flattens sentences into synthesis units. startIndex is
// retained in the API because callers construct queues relative to it; it no
// longer changes sentence phrasing.
export function buildGenerationUnits(sentences: DocSentence[], _startIndex: number): GenerationUnit[] {
  const units: GenerationUnit[] = []
  sentences.forEach((sentence, sentenceIndex) => {
    const pieces = splitClauses(sentence.text, MAX_UNIT_WORDS)
    for (const text of pieces) {
      units.push({ text, sentenceIndex })
    }
  })
  return units
}

// Minimum characters worth synthesizing on their own; smaller fragments are
// merged into a neighboring piece rather than spoken as a lone comma or dash.
const MIN_CAST_PIECE_CHARS = 4

// Splits a sentence's text along its quoted-speech char ranges (from the
// attribution scan) into alternating narrator/speech pieces. Slivers merge
// into their neighbor, adopting the larger side's voice.
export function splitByQuotedRanges(text: string, ranges: Array<[number, number]>): Array<{ text: string; quoted: boolean }> {
  if (!ranges.length) return [{ text, quoted: false }]
  const raw: Array<{ text: string; quoted: boolean }> = []
  let pos = 0
  for (const [start, end] of ranges) {
    if (start > pos) raw.push({ text: text.slice(pos, start), quoted: false })
    raw.push({ text: text.slice(start, end), quoted: true })
    pos = end
  }
  if (pos < text.length) raw.push({ text: text.slice(pos), quoted: false })

  const trimmed = raw.map(p => ({ text: p.text.trim(), quoted: p.quoted })).filter(p => p.text)
  const merged: Array<{ text: string; quoted: boolean }> = []
  for (const piece of trimmed) {
    const prev = merged[merged.length - 1]
    if (prev && piece.text.length < MIN_CAST_PIECE_CHARS) {
      prev.text += ' ' + piece.text
      continue
    }
    if (prev && prev.text.length < MIN_CAST_PIECE_CHARS) {
      merged[merged.length - 1] = { text: prev.text + ' ' + piece.text, quoted: piece.quoted }
      continue
    }
    merged.push({ ...piece })
  }
  return merged.length ? merged : [{ text, quoted: true }]
}

// Cast-mode unit builder: like buildGenerationUnits, but sentences split at
// quote boundaries first so dialogue can synthesize in the character's voice
// while tags and asides stay with the narrator. quotedRangesByFrom is keyed
// by DocSentence.from (the attribution join key).
export function buildCastGenerationUnits(
  sentences: DocSentence[],
  quotedRangesByFrom: Map<number, Array<[number, number]>>,
): GenerationUnit[] {
  const units: GenerationUnit[] = []
  sentences.forEach((sentence, sentenceIndex) => {
    const ranges = quotedRangesByFrom.get(sentence.from) ?? []
    for (const piece of splitByQuotedRanges(sentence.text, ranges)) {
      for (const text of splitClauses(piece.text, MAX_UNIT_WORDS)) {
        units.push({ text, sentenceIndex, quoted: piece.quoted })
      }
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
