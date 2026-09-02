// reviewDeskModel — pure derivation for the bottom bar Review tab.
//
// Turns fingerprint diagnostics into ordered "decision cards" and derives the
// id mappings the backend uses so author decisions round-trip:
//   * `stableID` (wails/internal/fingerprint/identity.go): sha256 of parts
//     joined by NUL, first 8 bytes hex, prefixed with the id kind.
//   * continuity signal ids (wails/internal/continuity/continuity.go signal()):
//     the folded copy of a diagnostic in the continuity report does NOT reuse
//     the diagnostic id — it hashes "fingerprint-<kind>", the title, and the
//     resolvable (evidence id, chapter id) pairs, all NUL-separated. We derive
//     that id too so a call made here also dims the same question on the
//     Continuity tab.
import { types } from '../../../wailsjs/go/models'
import type { ContinuityDecision } from '../../types/draftline'

/** NUL separator used by the Go backend when hashing stable ids. */
const NUL = String.fromCharCode(0)

export type ReviewKindColor = 'warning' | 'error' | 'muted'
export type DecisionStatus = 'reviewed' | 'dismissed'

export interface ReviewCard {
  id: string
  kind: string
  kindLabel: string
  kindColor: ReviewKindColor
  title: string
  detail: string
  /** Present only when the engine is less than 95% sure. */
  confidencePct?: number
  decided?: DecisionStatus
  /** Quotable text lifted from a near-duplicate diagnostic's detail. */
  excerpt?: string
  evidenceIds: string[]
  /** Concatenated front+body+back indexes for near-duplicates; see component. */
  chapterIndices: number[]
  /** Orphaned-correction cards: the author correction the diagnostic points at. */
  correctionId?: string
  dormantChapters?: number
  /** Ids to write author decisions under (diagnostic id + folded signal id). */
  decisionIds: string[]
  priority: number
}

export type ReviewActionKind =
  | 'reset'
  | 'flag_duplicate'
  | 'open_threads'
  | 'intentional'
  | 'discard_correction'
  | 'reattach'
  | 'goto'
  | 'review'
  | 'dismiss'
  | 'reopen'

export interface ReviewAction {
  action: ReviewActionKind
  label: string
  tone: 'confirm' | 'accent' | 'plain'
  disabled?: boolean
  title?: string
}

export interface ReviewDeskLayout {
  hero: ReviewCard | null
  columns: [ReviewCard[], ReviewCard[]]
}

type DiagnosticInput = Pick<
  types.FingerprintDiagnostic,
  'id' | 'kind' | 'title' | 'detail' | 'evidence_ids' | 'chapter_indices' | 'confidence'
>
type FingerprintInput = {
  diagnostics?: DiagnosticInput[]
  threads?: Pick<types.StoryThread, 'id' | 'state' | 'dormant_chapters'>[]
  author_model?: Pick<types.StoryAuthorModel, 'contexts' | 'checkpoints' | 'canon' | 'corrections' | 'profiles' | 'voice_notes'>
}
type EvidenceChapterLookup = ReadonlyMap<string, { readonly chapter_id: string }>

/** Undecided first, then by how urgently the kind needs an author call. */
const KIND_PRIORITY = [
  'near_duplicate_chapter',
  'canon_conflict',
  'unclosed_presence',
  'direction_conflict',
  'attribute_conflict',
  'identity_conflict',
  'orphaned_correction',
  'checkpoint_too_early',
  'orphan_character',
  'dormant_thread',
]

const KIND_COLORS: Record<string, ReviewKindColor> = {
  near_duplicate_chapter: 'warning',
  dormant_thread: 'warning',
  attribute_conflict: 'warning',
  direction_conflict: 'warning',
  checkpoint_too_early: 'warning',
  canon_conflict: 'error',
  orphaned_correction: 'error',
  unclosed_presence: 'error',
  identity_conflict: 'error',
  orphan_character: 'muted',
}

export function buildReviewCards(
  fingerprint: FingerprintInput | null | undefined,
  decisions: readonly ContinuityDecision[] | null | undefined,
  chapterIdByEvidence?: EvidenceChapterLookup,
): ReviewCard[] {
  if (!fingerprint) return []
  const diagnostics = fingerprint.diagnostics ?? []
  if (diagnostics.length === 0) return []

  const decisionStatus = new Map<string, DecisionStatus>()
  for (const decision of decisions ?? []) {
    if (decision.status === 'reviewed' || decision.status === 'dismissed') {
      decisionStatus.set(decision.signal_id, decision.status)
    }
  }

  // The dormant-thread diagnostic id hashes the thread id, so recompute the
  // hash per dormant thread to recover the chapter count for the label.
  const dormantByDiagnostic = new Map<string, number>()
  for (const thread of fingerprint.threads ?? []) {
    if (thread.state === 'dormant') {
      dormantByDiagnostic.set(stableDiagnosticId(thread.id, 'dormant'), thread.dormant_chapters ?? 0)
    }
  }
  // Same trick recovers which author correction an orphaned_correction card is
  // about: diagnostic id = stableID("diagnostic", correction.id, "orphaned").
  const correctionByDiagnostic = new Map<string, string>()
  for (const correction of fingerprint.author_model?.corrections ?? []) {
    correctionByDiagnostic.set(stableDiagnosticId(correction.id, 'orphaned'), correction.id)
  }

  const cards = diagnostics.map((diagnostic, order) => {
    const signalId = continuitySignalIdFor(diagnostic, chapterIdByEvidence)
    const decided = decisionStatus.get(diagnostic.id) ?? decisionStatus.get(signalId)
    const dormantChapters =
      diagnostic.kind === 'dormant_thread'
        ? dormantByDiagnostic.get(diagnostic.id) ?? parseDormantChapters(diagnostic.detail)
        : undefined
    const card: ReviewCard & { order: number } = {
      id: diagnostic.id,
      kind: diagnostic.kind,
      kindLabel: reviewKindLabel(diagnostic.kind, dormantChapters),
      kindColor: KIND_COLORS[diagnostic.kind] ?? 'muted',
      title: diagnostic.title,
      detail: diagnostic.detail,
      decided,
      dormantChapters,
      evidenceIds: [...(diagnostic.evidence_ids ?? [])],
      chapterIndices: [...(diagnostic.chapter_indices ?? [])],
      decisionIds: [diagnostic.id, signalId],
      priority: kindPriority(diagnostic.kind),
      order,
    }
    if (diagnostic.confidence < 0.95) card.confidencePct = Math.round(diagnostic.confidence * 100)
    if (diagnostic.kind === 'near_duplicate_chapter') {
      const excerpt = extractQuote(diagnostic.detail)
      if (excerpt) card.excerpt = excerpt
    }
    if (diagnostic.kind === 'orphaned_correction') {
      card.correctionId = correctionByDiagnostic.get(diagnostic.id)
    }
    return card
  })

  cards.sort((a, b) => {
    const decidedRank = Number(Boolean(a.decided)) - Number(Boolean(b.decided))
    return decidedRank || a.priority - b.priority || a.order - b.order
  })
  return cards.map(({ order: _order, ...card }) => card)
}

export function reviewKindLabel(kind: string, dormantChapters?: number): string {
  switch (kind) {
    case 'near_duplicate_chapter': return 'TEMPORAL ANOMALY · NEEDS A CALL'
    case 'unclosed_presence': return 'UNRESOLVED PRESENCE'
    case 'attribute_conflict': return 'ATTRIBUTE CONFLICT'
    case 'direction_conflict': return 'DIRECTIONAL CONFLICT'
    case 'canon_conflict': return 'CANON VIOLATION'
    case 'identity_conflict': return 'IDENTITY CONFLICT'
    case 'orphan_character': return 'FORGOTTEN INTRODUCTION'
    case 'orphaned_correction': return 'ORPHANED CORRECTION'
    case 'checkpoint_too_early': return 'CHECKPOINT TOO EARLY'
    case 'dormant_thread': {
      if (dormantChapters === undefined || dormantChapters <= 0) return 'DORMANT THREAD'
      return `DORMANT THREAD · ${dormantChapters} ${dormantChapters === 1 ? 'CHAPTER' : 'CHAPTERS'}`
    }
    default: return kind.replace(/_/g, ' ').toUpperCase()
  }
}

export function kindPriority(kind: string): number {
  const index = KIND_PRIORITY.indexOf(kind)
  return index === -1 ? KIND_PRIORITY.length : index
}

export function cardActions(card: ReviewCard): ReviewAction[] {
  if (card.decided) return [{ action: 'reopen', label: 'Reopen', tone: 'plain' }]
  switch (card.kind) {
    case 'near_duplicate_chapter':
      return [
        { action: 'reset', label: 'It’s a reset ↺', tone: 'confirm' },
        { action: 'flag_duplicate', label: 'Duplicate — flag it', tone: 'plain' },
      ]
    case 'dormant_thread':
      return [
        { action: 'open_threads', label: 'View thread', tone: 'accent' },
        { action: 'intentional', label: 'Intentional', tone: 'plain' },
      ]
    case 'orphaned_correction':
      return [
        { action: 'reattach', label: 'Reattach…', tone: 'accent', disabled: true, title: 'Reattaching corrections is coming soon' },
        card.correctionId
          ? { action: 'discard_correction', label: 'Discard', tone: 'plain' }
          : { action: 'dismiss', label: 'Dismiss', tone: 'plain' },
      ]
    default:
      return [
        { action: 'goto', label: 'Go to source', tone: 'accent', disabled: card.evidenceIds.length === 0 && card.chapterIndices.length === 0 },
        { action: 'review', label: 'Mark reviewed', tone: 'plain' },
        { action: 'dismiss', label: 'Dismiss', tone: 'plain' },
      ]
  }
}

/**
 * Highest-priority undecided card takes the wide first column; the rest flow
 * into two narrower columns. Cards arrive sorted, so decided ones naturally
 * sink to the bottoms of the columns.
 */
export function layoutReviewCards(cards: ReviewCard[]): ReviewDeskLayout {
  const hero = cards.length > 0 && !cards[0].decided ? cards[0] : null
  const rest = hero ? cards.slice(1) : cards
  const columns: [ReviewCard[], ReviewCard[]] = [[], []]
  rest.forEach((card, index) => columns[index % 2].push(card))
  return { hero, columns }
}

// --- Author-model drafts (plain data; callers convert with createFrom) -------

export interface AuthorModelDraft {
  contexts: types.StoryContext[]
  checkpoints: types.StoryCheckpoint[]
  canon: types.CanonRule[]
  corrections: types.FingerprintCorrection[]
  profiles: string[]
  voice_notes: types.CharacterVoiceNotes[]
}

export function cloneAuthorModel(model: FingerprintInput['author_model'] | null | undefined): AuthorModelDraft {
  return {
    contexts: [...(model?.contexts ?? [])],
    checkpoints: [...(model?.checkpoints ?? [])],
    canon: [...(model?.canon ?? [])],
    corrections: [...(model?.corrections ?? [])],
    profiles: [...(model?.profiles ?? [])],
    voice_notes: [...(model?.voice_notes ?? [])],
  }
}

export function loopContextId(diagnosticId: string): string {
  return 'context-loop-' + diagnosticId.replace(/^diagnostic-/, '')
}

/**
 * "It's a reset ↺": record the author's call as a simulation context in the
 * author model. Idempotent — a second call for the same diagnostic adds nothing.
 * Deliberately conservative: near-duplicate diagnostics carry no evidence ids,
 * so no `kind:"context"` corrections are written (no reliable target).
 */
export function withLoopContext(
  model: FingerprintInput['author_model'] | null | undefined,
  diagnosticId: string,
): { model: AuthorModelDraft; added: boolean; contextId: string } {
  const draft = cloneAuthorModel(model)
  const contextId = loopContextId(diagnosticId)
  if (draft.contexts.some(context => context.id === contextId)) {
    return { model: draft, added: false, contextId }
  }
  draft.contexts.push(types.StoryContext.createFrom({
    id: contextId,
    kind: 'simulation',
    label: 'Loop iteration',
    confidence: 1,
    source: 'author',
  }))
  return { model: draft, added: true, contextId }
}

export function withoutCorrection(
  model: FingerprintInput['author_model'] | null | undefined,
  correctionId: string,
): { model: AuthorModelDraft; removed: boolean } {
  const draft = cloneAuthorModel(model)
  const next = draft.corrections.filter(correction => correction.id !== correctionId)
  const removed = next.length !== draft.corrections.length
  draft.corrections = next
  return { model: draft, removed }
}

// --- Backend id derivations --------------------------------------------------

/** Mirrors fingerprint stableID("diagnostic", ...parts). */
export function stableDiagnosticId(...parts: string[]): string {
  return 'diagnostic-' + sha256Hex(parts.join(NUL)).slice(0, 16)
}

/**
 * Mirrors continuity.go: signal("fingerprint-"+kind, ..., title, sources, ...)
 * where sources are the diagnostic's evidence ids that resolve to records, in
 * order, each contributing NUL+evidenceID+NUL+chapterID to the hash key.
 */
export function continuitySignalIdFor(
  diagnostic: Pick<types.FingerprintDiagnostic, 'kind' | 'title' | 'evidence_ids'>,
  chapterIdByEvidence?: EvidenceChapterLookup,
): string {
  let key = 'fingerprint-' + diagnostic.kind + NUL + diagnostic.title
  for (const evidenceId of diagnostic.evidence_ids ?? []) {
    const record = chapterIdByEvidence?.get(evidenceId)
    if (record) key += NUL + evidenceId + NUL + record.chapter_id
  }
  return 'continuity-' + sha256Hex(key).slice(0, 16)
}

function parseDormantChapters(detail: string): number | undefined {
  const match = /for (\d+) chapters?/.exec(detail)
  return match ? Number(match[1]) : undefined
}

/** First “quoted” span in a detail string, if it is long enough to quote. */
export function extractQuote(detail: string): string | undefined {
  const match = /[“"]([^”"]{12,240})[”"]/.exec(detail)
  return match ? match[1] : undefined
}

// --- SHA-256 (sync, UTF-8) ---------------------------------------------------
// Needed because the backend derives every stable id from sha256 and the
// WebCrypto digest API is async; verified against NIST and Go-generated vectors.

const SHA_K = new Uint32Array([
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
  0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
  0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
  0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
  0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
  0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
  0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
  0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
])

function rotr(value: number, bits: number): number {
  return (value >>> bits) | (value << (32 - bits))
}

export function sha256Hex(text: string): string {
  const data = new TextEncoder().encode(text)
  const padded = new Uint8Array((((data.length + 8) >> 6) + 1) << 6)
  padded.set(data)
  padded[data.length] = 0x80
  const view = new DataView(padded.buffer)
  const bitLength = data.length * 8
  view.setUint32(padded.length - 8, Math.floor(bitLength / 0x100000000))
  view.setUint32(padded.length - 4, bitLength >>> 0)

  const state = new Uint32Array([0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19])
  const words = new Uint32Array(64)
  for (let offset = 0; offset < padded.length; offset += 64) {
    for (let i = 0; i < 16; i++) words[i] = view.getUint32(offset + i * 4)
    for (let i = 16; i < 64; i++) {
      const s0 = rotr(words[i - 15], 7) ^ rotr(words[i - 15], 18) ^ (words[i - 15] >>> 3)
      const s1 = rotr(words[i - 2], 17) ^ rotr(words[i - 2], 19) ^ (words[i - 2] >>> 10)
      words[i] = (words[i - 16] + s0 + words[i - 7] + s1) >>> 0
    }
    let [a, b, c, d, e, f, g, h] = state
    for (let i = 0; i < 64; i++) {
      const t1 = (h + (rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25)) + ((e & f) ^ (~e & g)) + SHA_K[i] + words[i]) >>> 0
      const t2 = ((rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22)) + ((a & b) ^ (a & c) ^ (b & c))) >>> 0
      h = g; g = f; f = e
      e = (d + t1) >>> 0
      d = c; c = b; b = a
      a = (t1 + t2) >>> 0
    }
    state[0] += a; state[1] += b; state[2] += c; state[3] += d
    state[4] += e; state[5] += f; state[6] += g; state[7] += h
  }
  let hex = ''
  for (const word of state) hex += word.toString(16).padStart(8, '0')
  return hex
}
