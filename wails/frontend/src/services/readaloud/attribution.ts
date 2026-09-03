// Per-sentence speaker attribution for Read Aloud cast mode. Pure module —
// no store imports, no editor types — so it unit-tests like segmentation.ts.
//
// Input is the full chapter's sentences (collectSentences output: text plus
// paragraph/block index); output attributes each sentence to the narrator, a
// roster character, or the unknown bucket. Everything is deterministic and
// local: quote-span tracking plus dialogue-tag/mention heuristics, no AI.
//
// Two phases:
//   A) a quote-span state machine that survives the segmenter's mid-quote
//      sentence splits ("Stop. Now." is two sentences) and multi-paragraph
//      speech (unclosed quote + next paragraph re-opening with a quote char);
//   B) per-sentence attribution rules, first hit wins: same-sentence tag,
//      pre-quote tag/action beat, quote continuation, nearby mention,
//      two-speaker alternation, unknown.
//
// Deliberate v1 scope limits (documented in docs/frontend/READ-ALOUD.md):
// em-dash and single-quote dialogue read as narration; a "mixed" sentence
// (quote + tag) is spoken whole in the character's voice; untagged
// three-plus-party exchanges land in the unknown bucket.

export type SpeakerKey = 'narrator' | 'unknown' | `char:${string}`

export interface RosterEntry {
  key: SpeakerKey // 'char:<lowercased canonical name>'
  name: string // canonical display name
  aliases: string[] // includes the name itself; matched longest-first
}

export type SentenceKind = 'narration' | 'dialogue' | 'mixed'
export type Confidence = 'tag' | 'pretag' | 'continuation' | 'nearby' | 'alternation' | 'none'

export interface AttributedSentence {
  from: number // join key back to DocSentence.from
  speaker: SpeakerKey
  kind: SentenceKind
  confidence: Confidence
}

export interface GenderEvidence {
  he: number
  she: number
  they: number
}

export interface AttributionResult {
  sentences: AttributedSentence[]
  // Dialogue+mixed sentence counts per speaker; the narrator's entry counts
  // narration sentences instead.
  lineCounts: Map<SpeakerKey, number>
  // Pronoun evidence accumulated from resolved "he said"-style tags.
  genderEvidence: Map<SpeakerKey, GenderEvidence>
}

export interface AttributionInput {
  from: number
  text: string
  block?: number
}

// Verbs that mark a dialogue tag ("...," she said). Lowercase; matched
// case-insensitively against the token next to a name or pronoun.
const TAG_VERBS = new Set([
  'said', 'asked', 'replied', 'whispered', 'shouted', 'muttered', 'murmured',
  'cried', 'called', 'answered', 'added', 'continued', 'snapped', 'hissed',
  'growled', 'breathed', 'began', 'insisted', 'admitted', 'agreed',
  'demanded', 'offered', 'warned', 'told', 'repeated', 'echoed', 'managed',
  'pressed', 'countered', 'blurted', 'stammered', 'sighed', 'laughed',
])

type Pronoun = 'he' | 'she' | 'they'

// A sentence at most this many unquoted non-space characters is pure
// dialogue; more unquoted text makes it "mixed" (quote plus tag/narration).
const MIXED_UNQUOTED_CHARS = 15

const OPENERS: Record<string, string> = { '“': '”', '«': '»', '"': '"' }

// A textblock containing only separator glyphs is a scene break.
const SCENE_SEPARATOR = /^[\s*\-–—~•#_.]+$/

function isStraightQuoteOpener(prev: string | null): boolean {
  return prev === null || /[\s([—–-]/.test(prev)
}

// ── Phase A: quote spans ─────────────────────────────────────────────────────

interface QuoteScan {
  kind: SentenceKind
  startsInsideQuote: boolean
  quoteOpensHere: boolean
  // Id of the quote span this sentence's speech belongs to (open → close,
  // including paragraph continuations); null for pure narration. A tag found
  // anywhere in a span names every sentence of it.
  spanId: number | null
  // Unquoted stretches of the sentence text (for tag matching), and the
  // unquoted prefix before the first quote (for pre-quote tag matching).
  unquoted: string
  unquotedBeforeQuote: string
}

function scanQuotes(sentences: AttributionInput[]): QuoteScan[] {
  const scans: QuoteScan[] = []
  let inQuote = false
  let closer = ''
  let nextSpanId = 0
  let activeSpanId = -1
  let prevBlock: number | undefined

  for (const sentence of sentences) {
    const text = sentence.text
    let index = 0

    if (sentence.block !== prevBlock && prevBlock !== undefined && inQuote) {
      // Convention: a speech spanning paragraphs leaves its quote unclosed
      // and re-opens with a quote char at the next paragraph start. If the
      // new paragraph does NOT re-open, the quote is over (or was a typo) —
      // close it so an unclosed straight quote can't swallow the chapter.
      const first = text.trimStart()[0] ?? ''
      if (OPENERS[first] && OPENERS[first] === closer) {
        // Continuation: consume the re-opening char as a marker, stay inside.
        index = text.indexOf(first) + 1
      } else {
        inQuote = false
        closer = ''
      }
    }
    prevBlock = sentence.block

    const startsInsideQuote = inQuote
    let spanId: number | null = startsInsideQuote ? activeSpanId : null
    let quoteOpensHere = false
    let quoted = 0
    const unquotedParts: string[] = []
    let current = ''
    let beforeQuote: string | null = null

    for (; index < text.length; index++) {
      const ch = text[index]
      if (!inQuote) {
        const isOpener = ch === '“' || ch === '«'
          || (ch === '"' && isStraightQuoteOpener(index === 0 ? null : text[index - 1]))
        if (isOpener) {
          inQuote = true
          closer = OPENERS[ch]
          quoteOpensHere = true
          activeSpanId = nextSpanId++
          if (spanId === null) spanId = activeSpanId
          if (beforeQuote === null) beforeQuote = current
          if (current.trim()) unquotedParts.push(current)
          current = ''
          continue
        }
        current += ch
      } else {
        if (ch === closer) {
          inQuote = false
          closer = ''
          continue
        }
        quoted++
      }
    }
    if (current.trim()) unquotedParts.push(current)

    const unquoted = unquotedParts.join(' ')
    const unquotedDense = unquoted.replace(/\s/g, '').length
    const kind: SentenceKind = quoted === 0 && !startsInsideQuote && !quoteOpensHere
      ? 'narration'
      : unquotedDense <= MIXED_UNQUOTED_CHARS ? 'dialogue' : 'mixed'

    scans.push({
      kind,
      startsInsideQuote,
      quoteOpensHere,
      spanId: kind === 'narration' ? null : spanId,
      unquoted,
      unquotedBeforeQuote: beforeQuote ?? '',
    })
  }
  return scans
}

// ── Roster matching ──────────────────────────────────────────────────────────

interface Matcher {
  key: SpeakerKey
  patterns: RegExp[]
}

function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function buildMatchers(roster: RosterEntry[]): Matcher[] {
  return roster.map(entry => ({
    key: entry.key,
    patterns: [...new Set([entry.name, ...entry.aliases])]
      .filter(alias => alias.trim().length > 1)
      .sort((a, b) => b.length - a.length)
      // Case-sensitive as written: "Grace" the character must not match
      // "grace" the noun.
      .map(alias => new RegExp(`(^|\\W)(${escapeRegExp(alias)})(?=\\W|$)`, 'g')),
  }))
}

interface RosterHit {
  key: SpeakerKey
  index: number
  length: number
}

function findRosterHits(text: string, matchers: Matcher[]): RosterHit[] {
  const hits: RosterHit[] = []
  for (const matcher of matchers) {
    for (const pattern of matcher.patterns) {
      pattern.lastIndex = 0
      let match: RegExpExecArray | null
      while ((match = pattern.exec(text)) !== null) {
        const index = match.index + match[1].length
        // Longest-alias-first within a matcher: skip shorter aliases that
        // land inside an already-recorded hit for the same character.
        if (!hits.some(h => h.key === matcher.key && index >= h.index && index < h.index + h.length)) {
          hits.push({ key: matcher.key, index, length: match[2].length })
        }
      }
    }
  }
  return hits.sort((a, b) => a.index - b.index)
}

// ── Phase B: attribution ─────────────────────────────────────────────────────

function emptyEvidence(): GenderEvidence {
  return { he: 0, she: 0, they: 0 }
}

function compatible(evidence: GenderEvidence | undefined, pronoun: Pronoun): boolean {
  if (!evidence) return true
  const total = evidence.he + evidence.she + evidence.they
  if (total === 0) return true
  const count = evidence[pronoun]
  return count >= evidence.he && count >= evidence.she && count >= evidence.they
}

// Finds "<Name> said" / "said <Name>" in unquoted text; returns the speaker.
function findNamedTag(unquoted: string, matchers: Matcher[]): SpeakerKey | null {
  const hits = findRosterHits(unquoted, matchers)
  for (const hit of hits) {
    const after = unquoted.slice(hit.index + hit.length).match(/^\s*,?\s*(?:had\s+|would\s+)?([A-Za-z']+)/)
    if (after && TAG_VERBS.has(after[1].toLowerCase())) return hit.key
    const before = unquoted.slice(0, hit.index).match(/([A-Za-z']+)\s*$/)
    if (before && TAG_VERBS.has(before[1].toLowerCase())) return hit.key
  }
  return null
}

function findPronounTag(unquoted: string): Pronoun | null {
  const match = /(^|\W)(he|she|they)\s+(?:had\s+|would\s+)?([A-Za-z']+)/i.exec(unquoted)
  if (match && TAG_VERBS.has(match[3].toLowerCase())) return match[2].toLowerCase() as Pronoun
  return null
}

export function attributeSpeakers(
  sentences: AttributionInput[],
  roster: RosterEntry[],
): AttributionResult {
  const scans = scanQuotes(sentences)
  const matchers = buildMatchers(roster)

  const out: AttributedSentence[] = []
  const lineCounts = new Map<SpeakerKey, number>()
  const genderEvidence = new Map<SpeakerKey, GenderEvidence>()

  // Scene-scoped context; reset at separator blocks.
  let lastSpeaker: SpeakerKey | null = null
  let prevParaSpeaker: SpeakerKey | null = null
  let currentParaSpeaker: SpeakerKey | null = null
  let recentSpeakers: SpeakerKey[] = [] // last two DISTINCT dialogue speakers, most recent first
  let lastNamedInNarration: { key: SpeakerKey; block: number | undefined } | null = null
  let sceneSpeakers = new Set<SpeakerKey>()

  const resetScene = () => {
    lastSpeaker = null
    prevParaSpeaker = null
    currentParaSpeaker = null
    recentSpeakers = []
    lastNamedInNarration = null
    sceneSpeakers = new Set()
  }

  const bumpEvidence = (key: SpeakerKey, pronoun: Pronoun) => {
    const evidence = genderEvidence.get(key) ?? emptyEvidence()
    evidence[pronoun]++
    genderEvidence.set(key, evidence)
  }

  let prevBlock: number | undefined

  for (let i = 0; i < sentences.length; i++) {
    const sentence = sentences[i]
    const scan = scans[i]
    const newParagraph = sentence.block !== prevBlock

    if (newParagraph) {
      prevParaSpeaker = currentParaSpeaker ?? prevParaSpeaker
      currentParaSpeaker = null
    }
    prevBlock = sentence.block

    if (SCENE_SEPARATOR.test(sentence.text)) {
      resetScene()
      out.push({ from: sentence.from, speaker: 'narrator', kind: 'narration', confidence: 'none' })
      continue
    }

    if (scan.kind === 'narration') {
      const hits = findRosterHits(sentence.text, matchers)
      if (hits.length) lastNamedInNarration = { key: hits[hits.length - 1].key, block: sentence.block }
      out.push({ from: sentence.from, speaker: 'narrator', kind: 'narration', confidence: 'none' })
      continue
    }

    // Dialogue or mixed — rules in order, first hit wins.
    let speaker: SpeakerKey | null = null
    let confidence: Confidence = 'none'

    // 1a/1b. Same-sentence named tag.
    const named = findNamedTag(scan.unquoted, matchers)
    if (named) {
      speaker = named
      confidence = 'tag'
    }

    // 1c. Same-sentence pronoun tag ("he said").
    if (!speaker) {
      const pronoun = findPronounTag(scan.unquoted)
      if (pronoun) {
        // A narration name only anchors a pronoun while it's fresh — this
        // paragraph or the one before.
        const fresh = lastNamedInNarration
          && (sentence.block === undefined || lastNamedInNarration.block === undefined
            || sentence.block - lastNamedInNarration.block <= 1)
        if (fresh && lastNamedInNarration && compatible(genderEvidence.get(lastNamedInNarration.key), pronoun)) {
          speaker = lastNamedInNarration.key
          confidence = 'tag'
        } else {
          const candidate = recentSpeakers.find(key => key !== 'unknown'
            && (genderEvidence.get(key)?.[pronoun] ?? 0) > 0
            && compatible(genderEvidence.get(key), pronoun))
          if (candidate) {
            speaker = candidate
            confidence = 'tag'
          }
        }
        if (speaker) bumpEvidence(speaker, pronoun)
      }
    }

    // 2. Pre-quote tag / action beat: name in the unquoted text before the
    //    quote opens, or in the immediately preceding narration sentence of
    //    the same paragraph.
    if (!speaker && scan.quoteOpensHere && !scan.startsInsideQuote) {
      const beforeHits = findRosterHits(scan.unquotedBeforeQuote, matchers)
      if (beforeHits.length) {
        speaker = beforeHits[beforeHits.length - 1].key
        confidence = 'pretag'
      } else if (i > 0 && scans[i - 1].kind === 'narration' && sentences[i - 1].block === sentence.block) {
        const prevHits = findRosterHits(sentences[i - 1].text, matchers)
        if (prevHits.length === 1) {
          speaker = prevHits[0].key
          confidence = 'pretag'
        }
      }
    }

    // 3. Quote continuation: the segmenter split inside the quote, or the
    //    speech continued across a paragraph boundary.
    if (!speaker && scan.startsInsideQuote && lastSpeaker) {
      speaker = lastSpeaker
      confidence = 'continuation'
    }

    // 4. Nearby mention: nearest earlier narration sentence in this
    //    paragraph, then the previous paragraph, holding exactly one name.
    if (!speaker) {
      outer: for (const scope of [sentence.block, (sentence.block ?? 0) - 1]) {
        for (let j = i - 1; j >= 0; j--) {
          if (sentences[j].block !== scope) continue
          if (scans[j].kind !== 'narration') continue
          const hits = findRosterHits(sentences[j].text, matchers)
          if (hits.length === 1) {
            speaker = hits[0].key
            confidence = 'nearby'
            break outer
          }
          if (hits.length > 1) break outer // ambiguous — don't guess farther
        }
      }
    }

    // 5. Two-speaker alternation: a fresh dialogue paragraph in a
    //    two-character scene answers the previous paragraph's speaker.
    if (!speaker && newParagraph && recentSpeakers.length === 2 && sceneSpeakers.size === 2
      && prevParaSpeaker && prevParaSpeaker === recentSpeakers[0]) {
      speaker = recentSpeakers[1]
      confidence = 'alternation'
    }

    // 5b. Monologue continuation: with exactly one speaker active in the
    //     scene, an untagged dialogue paragraph continues that speaker.
    if (!speaker && newParagraph && sceneSpeakers.size === 1
      && prevParaSpeaker && sceneSpeakers.has(prevParaSpeaker)) {
      speaker = prevParaSpeaker
      confidence = 'alternation'
    }

    // 6. Unknown.
    if (!speaker) {
      speaker = 'unknown'
      confidence = 'none'
    }

    // Tagged sentences name people in their unquoted parts too ("," Marcus
    // said.) — remember them for later pronoun tags.
    if (scan.unquoted) {
      const hits = findRosterHits(scan.unquoted, matchers)
      if (hits.length) lastNamedInNarration = { key: hits[hits.length - 1].key, block: sentence.block }
    }

    out.push({ from: sentence.from, speaker, kind: scan.kind, confidence })

    lastSpeaker = speaker
    currentParaSpeaker = speaker
    if (speaker !== 'unknown' && speaker !== 'narrator') {
      sceneSpeakers.add(speaker)
      if (recentSpeakers[0] !== speaker) {
        recentSpeakers = [speaker, ...recentSpeakers.filter(key => key !== speaker)].slice(0, 2)
      }
    }
  }

  // Backfill: a tag anywhere in a quote span names the whole span. The
  // segmenter often splits “Stop. Now,” he said. so the tag lands in a later
  // sentence than the speech's start.
  const spanSpeaker = new Map<number, SpeakerKey>()
  for (let i = 0; i < out.length; i++) {
    const spanId = scans[i].spanId
    if (spanId !== null && out[i].speaker !== 'unknown' && out[i].speaker !== 'narrator' && !spanSpeaker.has(spanId)) {
      spanSpeaker.set(spanId, out[i].speaker)
    }
  }
  for (let i = 0; i < out.length; i++) {
    const spanId = scans[i].spanId
    if (spanId === null || out[i].speaker !== 'unknown') continue
    const speaker = spanSpeaker.get(spanId)
    if (speaker) out[i] = { ...out[i], speaker, confidence: 'continuation' }
  }

  for (const attributed of out) {
    lineCounts.set(attributed.speaker, (lineCounts.get(attributed.speaker) ?? 0) + 1)
  }

  return { sentences: out, lineCounts, genderEvidence }
}

// Convenience for roster construction: the canonical speaker key for a
// character name.
export function speakerKeyFor(name: string): SpeakerKey {
  return `char:${name.trim().toLocaleLowerCase()}`
}
