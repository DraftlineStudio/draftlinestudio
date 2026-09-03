// Read Aloud sentence segmentation. Pure text in, character spans out — the
// ProseMirror position mapping lives in docSentences.ts so this layer stays
// trivially unit-testable.
//
// The rules favor prose fiction: sentence-final punctuation may hide inside
// closing quotes ("Go away." Then he left.), abbreviations and initials do not
// end sentences, ellipses continue a sentence when the text resumes in
// lowercase, and em dashes never terminate. A missed split merely reads two
// sentences in one breath, so ambiguous cases (etc. at a true sentence end)
// lean toward not splitting.

export interface SentenceSpan {
  start: number
  end: number
}

// Compared lowercase with internal dots removed, so "i.e." matches "ie".
// Only consulted when the terminator is a single period.
const ABBREVIATIONS = new Set([
  'mr', 'mrs', 'ms', 'dr', 'st', 'prof', 'sr', 'jr', 'vs', 'etc', 'al',
  'ie', 'eg', 'cf', 'mt', 'ave', 'blvd', 'capt', 'gen', 'lt', 'col', 'sgt',
  'fig', 'vol', 'ch', 'pp', 'approx', 'dept', 'est',
])

const TERMINATORS = '.!?…'
const CLOSERS = '"”’\')\\]»'

function isTerminator(ch: string): boolean {
  return TERMINATORS.includes(ch)
}

function isCloser(ch: string): boolean {
  return CLOSERS.includes(ch)
}

function isWhitespace(ch: string): boolean {
  return /\s/.test(ch)
}

function isLowercaseLetter(ch: string): boolean {
  return /\p{Ll}/u.test(ch)
}

function isDigit(ch: string): boolean {
  return ch >= '0' && ch <= '9'
}

// The word immediately before position, allowing internal periods ("e.g").
function tokenBefore(text: string, position: number): string {
  let start = position
  while (start > 0 && /[\p{L}.]/u.test(text[start - 1])) start--
  return text.slice(start, position)
}

// segmentText splits text into sentence spans. Span boundaries include the
// terminator run and any closing quotes/brackets; leading whitespace is
// excluded. Text without a final terminator still yields a final span.
export function segmentText(text: string): SentenceSpan[] {
  const spans: SentenceSpan[] = []
  let sentenceStart = 0
  let i = 0

  const pushSpan = (endExclusive: number) => {
    while (sentenceStart < endExclusive && isWhitespace(text[sentenceStart])) sentenceStart++
    if (sentenceStart >= endExclusive) {
      sentenceStart = endExclusive
      return
    }
    const body = text.slice(sentenceStart, endExclusive).trim()
    // Merge fragments too small to speak ("A.", stray closers) into the
    // previous sentence rather than emitting a blip of audio.
    const previous = spans[spans.length - 1]
    if (body.replace(/[^\p{L}\p{N}]/gu, '').length < 2 && previous) {
      previous.end = endExclusive
    } else {
      spans.push({ start: sentenceStart, end: endExclusive })
    }
    sentenceStart = endExclusive
  }

  while (i < text.length) {
    if (!isTerminator(text[i])) {
      i++
      continue
    }

    // Consume the whole terminator run (?!, !!!, ...).
    const runStart = i
    while (i < text.length && isTerminator(text[i])) i++
    const run = text.slice(runStart, i)

    // Then any closing quotes/brackets ("Go away." → boundary after the quote).
    let end = i
    while (end < text.length && isCloser(text[end])) end++

    const isEllipsis = run.includes('…') || /\.{2,}/.test(run)

    if (run === '.') {
      const before = tokenBefore(text, runStart)
      const compact = before.replace(/\./g, '').toLocaleLowerCase()
      // Single-capital initials: J. R. R. Tolkien.
      if (/^\p{Lu}$/u.test(before)) { i = end; continue }
      if (ABBREVIATIONS.has(compact)) { i = end; continue }
      // "No." is an abbreviation only in front of a number (No. 5).
      if (compact === 'no' && /^\s*\d/.test(text.slice(end))) { i = end; continue }
      // Decimals and versions: 3.50, 0.17.
      if (isDigit(text[runStart - 1] ?? '') && isDigit(text[runStart + 1] ?? '')) { i = end; continue }
    }

    // What follows decides. EOF always ends the sentence.
    if (end >= text.length) {
      pushSpan(end)
      i = end
      continue
    }
    // A boundary needs whitespace after the closers; "end.Next" is a typo we
    // leave unsplit rather than reading a false pause.
    if (!isWhitespace(text[end])) {
      i = end
      continue
    }
    let next = end
    while (next < text.length && isWhitespace(text[next])) next++
    // A trailing-off ellipsis continues when the text resumes in lowercase;
    // the same rule keeps "he said." inside dialogue attribution unsplit.
    if (next < text.length && isLowercaseLetter(text[next])) {
      i = end
      continue
    }
    pushSpan(end)
    i = end
  }

  pushSpan(text.length)
  return spans
}
