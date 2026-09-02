import type { types } from '../../../wailsjs/go/models'

/** Visual weight of a continuity signal in the issue list / detail chip. */
export type ContinuityTone = 'error' | 'warning' | 'muted'

export type ContinuityFilter = 'all' | 'review' | 'observations' | 'decided'

/**
 * Maps a signal to its severity tone. Paired-source review signals are hard
 * claim-vs-established conflicts (error); single-source review signals are
 * softer prompts (warning); observations stay muted.
 */
export function signalTone(signal: Pick<types.ContinuitySignal, 'severity' | 'sources'>): ContinuityTone {
  if (signal.severity !== 'review') return 'muted'
  return (signal.sources?.length ?? 0) >= 2 ? 'error' : 'warning'
}

export function categoryLabel(value: string): string {
  if (value === 'knowledge') return 'Who knows what'
  if (value === 'facts') return 'Story facts'
  if (value === 'chronology') return 'Chronology'
  if (value === 'structure') return 'Story structure'
  if (value === 'story') return 'Story fingerprint'
  return 'Characters'
}

/** "fingerprint-attribute_conflict" → "Attribute conflict". */
export function kindLabel(value: string): string {
  return value
    .replace(/^fingerprint[-_]/, '')
    .replace(/[-_]/g, ' ')
    .replace(/^./, letter => letter.toUpperCase())
}

/** "Ch 3 → Ch 7", "Ch 11", or "" when no source carries a chapter. */
export function chapterRange(sources: types.ContinuitySource[] | undefined): string {
  const indexes = (sources ?? [])
    .map(source => source.chapter_index)
    .filter(index => Number.isInteger(index) && index >= 0)
  if (indexes.length === 0) return ''
  const min = Math.min(...indexes)
  const max = Math.max(...indexes)
  return min === max ? `Ch ${min + 1}` : `Ch ${min + 1} → Ch ${max + 1}`
}

/** Issue-list meta line: "attribute conflict · Ch 2 → Ch 11". */
export function issueMeta(signal: Pick<types.ContinuitySignal, 'kind' | 'sources'>): string {
  const kind = kindLabel(signal.kind).toLowerCase()
  const range = chapterRange(signal.sources)
  return range ? `${kind} · ${range}` : kind
}

/** Compact card location: "Ch 7 ¶ 14" (paragraph shown when known). */
export function sourceLoc(source: Pick<types.ContinuitySource, 'chapter_index' | 'paragraph_index'>): string {
  const chapter = `Ch ${source.chapter_index + 1}`
  return typeof source.paragraph_index === 'number' && source.paragraph_index >= 0
    ? `${chapter} ¶ ${source.paragraph_index + 1}`
    : chapter
}

/** Detail header location line: "Chapter title · Ch 3 → Ch 7". */
export function whereLabel(signal: Pick<types.ContinuitySignal, 'sources'>): string {
  const first = signal.sources?.[0]
  if (!first) return 'No linked passage'
  const range = chapterRange(signal.sources)
  return range.includes('→') ? `${first.chapter_title} · ${range}` : first.chapter_title
}

export interface ContinuityEvidenceCard {
  role: 'claim' | 'canon' | 'single' | 'extra'
  label: string
  source: types.ContinuitySource
}

/**
 * First source is the claim under question, second is the established/canon
 * passage; a lone source renders as a single full-width card.
 */
export function evidenceCards(signal: Pick<types.ContinuitySignal, 'sources'>): ContinuityEvidenceCard[] {
  const sources = signal.sources ?? []
  if (sources.length === 0) return []
  if (sources.length === 1) {
    return [{ role: 'single', label: `Source · ${sourceLoc(sources[0])}`, source: sources[0] }]
  }
  return sources.map((source, index) => {
    if (index === 0) return { role: 'claim' as const, label: `The claim · ${sourceLoc(source)}`, source }
    if (index === 1) return { role: 'canon' as const, label: `Established · ${sourceLoc(source)}`, source }
    return { role: 'extra' as const, label: `Source ${index + 1} · ${sourceLoc(source)}`, source }
  })
}

/** Applies the status chip filter plus the free-text filter. */
export function filterSignals(
  signals: types.ContinuitySignal[],
  filter: ContinuityFilter,
  textFilter: string,
): types.ContinuitySignal[] {
  const needle = textFilter.trim().toLowerCase()
  return signals.filter(signal => {
    if (filter === 'review' && (signal.severity !== 'review' || signal.status)) return false
    if (filter === 'observations' && (signal.severity === 'review' || signal.status)) return false
    if (filter === 'decided' && !signal.status) return false
    if (needle) {
      const sourceText = signal.sources?.map(source => `${source.chapter_title} ${source.text}`).join(' ') ?? ''
      const haystack = `${signal.title} ${signal.detail} ${(signal.character_names ?? []).join(' ')} ${sourceText}`.toLowerCase()
      if (!haystack.includes(needle)) return false
    }
    return true
  })
}
