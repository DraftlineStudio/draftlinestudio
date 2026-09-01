import { describe, expect, it } from 'vitest'
import type { EvidenceRecord } from '../../types/draftline'
import { rankEvidenceForReview } from './evidenceReview'

const record = (overrides: Partial<EvidenceRecord>): EvidenceRecord => ({
  id: 'record', kind: 'fact', evidence_type: 'state', chapter_id: 'one', chapter_index: 0,
  section: 'body', section_index: 0, paragraph_index: 0, sentence_index: 0,
  start_offset: 0, end_offset: 10, text: 'A source sentence.', confidence: .8,
  rationale: 'test', status: 'detected', source: 'auto', ...overrides,
})

describe('evidence review ranking', () => {
  it('prioritizes discoveries and singleton named details over mundane states', () => {
    const ranked = rankEvidenceForReview([
      record({ id: 'mundane', named_entities: [{ text: 'IBM', label: 'ORG' }] }),
      record({ id: 'repeat', text: 'IBM remained open.', named_entities: [{ text: 'IBM', label: 'ORG' }] }),
      record({ id: 'discovery', evidence_type: 'discovery', text: 'Kyle found the key.', named_entities: [{ text: 'Kyle', label: 'PERSON' }] }),
    ])
    expect(ranked.map(item => item.record.id)).toEqual(['discovery'])
    expect(ranked[0].reasons).toContain('Named detail appears in one indexed passage')
  })

  it('keeps pinned records visible and excludes rejected records', () => {
    const ranked = rankEvidenceForReview([
      record({ id: 'pinned', pinned: true, status: 'confirmed' }),
      record({ id: 'rejected', evidence_type: 'discovery', status: 'rejected' }),
    ])
    expect(ranked.map(item => item.record.id)).toEqual(['pinned'])
  })
})
