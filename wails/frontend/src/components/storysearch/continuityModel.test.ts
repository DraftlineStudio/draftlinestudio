import { describe, expect, it } from 'vitest'
import { types } from '../../../wailsjs/go/models'
import {
  chapterRange,
  evidenceCards,
  filterSignals,
  issueMeta,
  kindLabel,
  signalTone,
  sourceLoc,
  whereLabel,
} from './continuityModel'

function source(partial: Partial<types.ContinuitySource> = {}): types.ContinuitySource {
  return types.ContinuitySource.createFrom({
    evidence_id: 'ev-1',
    text: 'She was already gone.',
    chapter_index: 2,
    chapter_title: 'Chapter 3',
    section: 'body',
    section_index: 2,
    ...partial,
  })
}

function signal(partial: Partial<types.ContinuitySignal> = {}): types.ContinuitySignal {
  return types.ContinuitySignal.createFrom({
    id: 'continuity-1',
    kind: 'attribute-conflict',
    category: 'facts',
    severity: 'review',
    title: 'Ruiz hair color may conflict',
    detail: 'Two passages disagree.',
    confidence: 0.9,
    sources: [source(), source({ chapter_index: 6, chapter_title: 'Chapter 7', paragraph_index: 13 })],
    ...partial,
  })
}

describe('signalTone', () => {
  it('marks paired review signals as errors', () => {
    expect(signalTone(signal())).toBe('error')
  })
  it('marks single-source review signals as warnings', () => {
    expect(signalTone(signal({ sources: [source()] }))).toBe('warning')
  })
  it('mutes observations regardless of sources', () => {
    expect(signalTone(signal({ severity: 'info' }))).toBe('muted')
  })
})

describe('kindLabel', () => {
  it('humanizes hyphenated kinds', () => {
    expect(kindLabel('attribute-conflict')).toBe('Attribute conflict')
  })
  it('strips the fingerprint prefix and underscores', () => {
    expect(kindLabel('fingerprint-canon_conflict')).toBe('Canon conflict')
  })
})

describe('chapterRange / issueMeta / whereLabel', () => {
  it('renders a range across chapters', () => {
    expect(chapterRange(signal().sources)).toBe('Ch 3 → Ch 7')
    expect(issueMeta(signal())).toBe('attribute conflict · Ch 3 → Ch 7')
    expect(whereLabel(signal())).toBe('Chapter 3 · Ch 3 → Ch 7')
  })
  it('collapses a single chapter', () => {
    const one = signal({ sources: [source()] })
    expect(chapterRange(one.sources)).toBe('Ch 3')
    expect(whereLabel(one)).toBe('Chapter 3')
  })
  it('handles missing sources', () => {
    const none = signal({ sources: [] })
    expect(chapterRange(none.sources)).toBe('')
    expect(issueMeta(none)).toBe('attribute conflict')
    expect(whereLabel(none)).toBe('No linked passage')
  })
})

describe('sourceLoc', () => {
  it('includes the paragraph when known', () => {
    expect(sourceLoc(source({ chapter_index: 6, paragraph_index: 13 }))).toBe('Ch 7 ¶ 14')
  })
  it('omits the paragraph when unknown', () => {
    expect(sourceLoc(source())).toBe('Ch 3')
  })
})

describe('evidenceCards', () => {
  it('maps first source to claim and second to established', () => {
    const cards = evidenceCards(signal())
    expect(cards.map(card => card.role)).toEqual(['claim', 'canon'])
    expect(cards[0].label).toBe('The claim · Ch 3')
    expect(cards[1].label).toBe('Established · Ch 7 ¶ 14')
  })
  it('renders a lone source as a single card', () => {
    const cards = evidenceCards(signal({ sources: [source()] }))
    expect(cards).toHaveLength(1)
    expect(cards[0].role).toBe('single')
    expect(cards[0].label).toBe('Source · Ch 3')
  })
  it('labels sources beyond the pair as extras', () => {
    const three = signal({ sources: [source(), source(), source({ chapter_index: 9 })] })
    expect(evidenceCards(three).map(card => card.role)).toEqual(['claim', 'canon', 'extra'])
    expect(evidenceCards(three)[2].label).toBe('Source 3 · Ch 10')
  })
  it('returns nothing without sources', () => {
    expect(evidenceCards(signal({ sources: [] }))).toEqual([])
  })
})

describe('filterSignals', () => {
  const open = signal({ id: 'a' })
  const observation = signal({ id: 'b', severity: 'info', title: 'Thin chapter coverage' })
  const decided = signal({ id: 'c', status: 'dismissed' })
  const all = [open, observation, decided]

  it('keeps everything on all', () => {
    expect(filterSignals(all, 'all', '')).toHaveLength(3)
  })
  it('review keeps undecided review signals only', () => {
    expect(filterSignals(all, 'review', '').map(s => s.id)).toEqual(['a'])
  })
  it('observations keeps undecided info signals only', () => {
    expect(filterSignals(all, 'observations', '').map(s => s.id)).toEqual(['b'])
  })
  it('decided keeps decided signals only', () => {
    expect(filterSignals(all, 'decided', '').map(s => s.id)).toEqual(['c'])
  })
  it('text filter matches titles, details, names, and source text', () => {
    expect(filterSignals(all, 'all', 'thin chapter').map(s => s.id)).toEqual(['b'])
    expect(filterSignals(all, 'all', 'already gone')).toHaveLength(3)
    expect(filterSignals(all, 'all', 'zebra')).toHaveLength(0)
  })
})
