import type { EvidenceRecord } from '../../types/draftline'

export interface RankedEvidence {
  record: EvidenceRecord
  score: number
  reasons: string[]
}

const reviewLimit = 100

export function rankEvidenceForReview(records: EvidenceRecord[]): RankedEvidence[] {
  const termFrequency = new Map<string, number>()
  for (const record of records) {
    if (record.status === 'rejected') continue
    for (const term of recordTerms(record)) {
      termFrequency.set(term, (termFrequency.get(term) ?? 0) + 1)
    }
  }

  return records
    .filter(record => record.status === 'detected' || record.pinned)
    .map(record => scoreRecord(record, termFrequency))
    .filter(candidate => candidate.record.pinned || candidate.score >= 35)
    .sort((a, b) => b.score - a.score || a.record.chapter_index - b.record.chapter_index || a.record.start_offset - b.record.start_offset)
    .slice(0, reviewLimit)
}

function scoreRecord(record: EvidenceRecord, termFrequency: Map<string, number>): RankedEvidence {
  let score = 0
  const reasons: string[] = []
  const add = (points: number, reason: string) => { score += points; reasons.push(reason) }

  if (record.pinned) add(100, 'Pinned by author')
  switch (record.evidence_type) {
    case 'discovery': add(60, 'Possible discovery or revelation'); break
    case 'introduction': add(50, 'First confirmed character appearance'); break
    case 'time_reference': add(40, 'Explicit story-time reference'); break
    case 'interaction': add(35, 'Character interaction'); break
    case 'transition': add(12, 'Arrival, departure, or movement'); break
    default: add(5, 'Candidate story fact')
  }
  if ((record.character_ids?.length ?? 0) >= 2) add(15, 'Several confirmed characters share this moment')
  if ((record.knowledge_states?.length ?? 0) > 0) add(45, 'Establishes a character knowledge state')
  if (record.confidence < .75) add(10, 'Lower-confidence rule match')
  if (recordTerms(record).some(term => termFrequency.get(term) === 1)) add(25, 'Named detail appears in one indexed passage')

  return { record, score, reasons }
}

function recordTerms(record: EvidenceRecord): string[] {
  const values = [...(record.character_names ?? []), ...(record.named_entities ?? []).map(entity => entity.text)]
  return [...new Set(values.map(value => value.trim().toLocaleLowerCase()).filter(Boolean))]
}
