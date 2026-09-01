import { useMemo, useState } from 'react'
import { useAnalysisStore } from '../../store/analysisStore'
import type { BookData, EvidenceRecord, Section } from '../../types/draftline'

type EvidenceFilter = 'all' | 'event' | 'fact'

interface Props {
  book: BookData
  onNavigate: (section: Section, sectionIndex: number, query: string) => void
}

export default function EvidenceIndexPanel({ book, onNavigate }: Props) {
  const [filter, setFilter] = useState<EvidenceFilter>('all')
  const runAnalysis = useAnalysisStore(state => state.run)
  const analysisState = useAnalysisStore(state => state.state)
  const evidence = book.analysis?.evidence
  const records = useMemo(
    () => (evidence?.records ?? []).filter(record => record.status !== 'rejected' && (filter === 'all' || record.kind === filter)),
    [evidence, filter],
  )
  const chapters = useMemo(() => [...book.front_matter, ...book.body, ...book.back_matter], [book])
  const eventCount = evidence?.records.filter(record => record.kind === 'event' && record.status !== 'rejected').length ?? 0
  const factCount = evidence?.records.filter(record => record.kind === 'fact' && record.status !== 'rejected').length ?? 0

  if (!evidence) {
    return (
      <div className="evidence-index-empty">
        <strong>The evidence index has not been built yet.</strong>
        <span>Draftline will build it during the next local analysis pass. It uses deterministic language cues and exact manuscript sentences—not generative AI.</span>
        <button type="button" onClick={() => void runAnalysis()} disabled={analysisState === 'running'}>
          {analysisState === 'running' ? 'Analyzing…' : 'Analyze now'}
        </button>
      </div>
    )
  }

  return (
    <div className="evidence-index">
      <div className="evidence-index-toolbar">
        <div className="evidence-index-summary">
          <strong>{eventCount + factCount}</strong> source-backed records
          <span>{eventCount} events · {factCount} facts</span>
          {evidence.truncated && <span className="evidence-index-warning">Index limit reached</span>}
        </div>
        <div className="evidence-index-filters" role="group" aria-label="Filter evidence records">
          {(['all', 'event', 'fact'] as const).map(value => (
            <button key={value} type="button" className={filter === value ? 'active' : ''} onClick={() => setFilter(value)}>
              {value === 'all' ? 'All' : value === 'event' ? 'Events' : 'Facts'}
            </button>
          ))}
        </div>
        <span className="evidence-index-built" title={evidence.last_analyzed}>Local index · {evidence.engine}</span>
      </div>

      {records.length === 0 ? (
        <div className="evidence-index-empty"><span>No {filter === 'all' ? '' : `${filter} `}records are currently indexed.</span></div>
      ) : (
        <div className="evidence-index-list">
          {records.map(record => (
            <EvidenceRow
              key={record.id}
              record={record}
              chapterTitle={chapters[record.chapter_index]?.title || `Chapter ${record.chapter_index + 1}`}
              onNavigate={onNavigate}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function EvidenceRow({ record, chapterTitle, onNavigate }: { record: EvidenceRecord; chapterTitle: string; onNavigate: Props['onNavigate'] }) {
  const cue = record.confidence >= .85 ? 'strong cue' : record.confidence >= .75 ? 'clear cue' : 'possible cue'
  const terms = [...(record.character_names ?? []), ...(record.named_entities ?? []).map(entity => entity.text)]
    .filter((term, index, all) => all.findIndex(value => value.toLocaleLowerCase() === term.toLocaleLowerCase()) === index)

  return (
    <button
      type="button"
      className="evidence-index-row"
      onClick={() => onNavigate(record.section as Section, record.section_index, record.text)}
      title={`Open evidence in ${chapterTitle}`}
    >
      <span className="evidence-index-location">
        <strong>{chapterTitle}</strong>
        <span>¶ {record.paragraph_index + 1} · sentence {record.sentence_index + 1}</span>
      </span>
      <span className="evidence-index-content">
        <span className="evidence-index-badges">
          <i className={`evidence-kind ${record.kind}`}>{record.kind}</i>
          <i>{labelEvidenceType(record.evidence_type)}</i>
          <i title={record.rationale}>{cue}</i>
          {record.status === 'confirmed' && <i className="confirmed">author confirmed</i>}
        </span>
        <span className="evidence-index-text">{record.text}</span>
        {terms.length > 0 && <span className="evidence-index-terms">{terms.slice(0, 6).join(' · ')}</span>}
      </span>
      <span className="evidence-index-open">Open →</span>
    </button>
  )
}

function labelEvidenceType(value: string): string {
  return value.replace(/_/g, ' ').replace(/^./, (letter: string) => letter.toUpperCase())
}
