import { useMemo, useState } from 'react'
import { useAnalysisStore } from '../../store/analysisStore'
import { useBookStore } from '../../store/bookStore'
import type { BookData, EvidenceRecord, Section } from '../../types/draftline'
import { rankEvidenceForReview } from './evidenceReview'

type EvidenceFilter = 'review' | 'all' | 'event' | 'fact'

interface Props {
  book: BookData
  onNavigate: (section: Section, sectionIndex: number, query: string) => void
}

export default function EvidenceIndexPanel({ book, onNavigate }: Props) {
  const [filter, setFilter] = useState<EvidenceFilter>('review')
  const runAnalysis = useAnalysisStore(state => state.run)
  const analysisState = useAnalysisStore(state => state.state)
  const updateEvidenceRecord = useBookStore(state => state.updateEvidenceRecord)
  const evidence = book.analysis?.evidence
  const ranked = useMemo(() => rankEvidenceForReview(evidence?.records ?? []), [evidence])
  const reasons = useMemo(() => new Map(ranked.map(item => [item.record.id, item.reasons])), [ranked])
  const records = useMemo(() => {
    if (filter === 'review') return ranked.map(item => item.record)
    return (evidence?.records ?? []).filter(record => filter === 'all' || record.kind === filter)
  }, [evidence, filter, ranked])
  const chapters = useMemo(() => [...book.front_matter, ...book.body, ...book.back_matter], [book])
  const active = evidence?.records.filter(record => record.status !== 'rejected') ?? []
  const eventCount = active.filter(record => record.kind === 'event').length
  const factCount = active.filter(record => record.kind === 'fact').length
  const rejectedCount = evidence?.records.filter(record => record.status === 'rejected').length ?? 0
  const reviewedCount = evidence?.records.filter(record => record.status !== 'detected').length ?? 0

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
          <strong>{filter === 'review' ? ranked.length : records.length}</strong>
          {filter === 'review'
            ? <span>worth reviewing</span>
            : <span>{eventCount} active events · {factCount} active facts{rejectedCount > 0 ? ` · ${rejectedCount} rejected` : ''}</span>}
          {reviewedCount > 0 && <span>{reviewedCount} decisions saved</span>}
          {evidence.truncated && <span className="evidence-index-warning">Index limit reached</span>}
        </div>
        <div className="evidence-index-filters" role="group" aria-label="Filter evidence records">
          {(['review', 'all', 'event', 'fact'] as const).map(value => (
            <button key={value} type="button" className={filter === value ? 'active' : ''} onClick={() => setFilter(value)}>
              {value === 'review' ? 'Worth reviewing' : value === 'all' ? 'Everything' : value === 'event' ? 'Events' : 'Facts'}
            </button>
          ))}
        </div>
        <span className="evidence-index-built" title={evidence.last_analyzed}>Local index · {evidence.engine}</span>
      </div>

      {records.length === 0 ? (
        <div className="evidence-index-empty">
          <strong>{filter === 'review' ? 'Nothing needs your attention.' : 'No matching evidence.'}</strong>
          <span>{filter === 'review' ? 'The complete source-backed index remains available under Everything.' : 'Try another archive filter.'}</span>
        </div>
      ) : (
        <div className="evidence-index-list">
          {records.map(record => (
            <EvidenceRow
              key={record.id}
              record={record}
              reviewReasons={reasons.get(record.id) ?? []}
              chapterTitle={chapters[record.chapter_index]?.title || `Chapter ${record.chapter_index + 1}`}
              onNavigate={onNavigate}
              onUpdate={changes => updateEvidenceRecord(record.id, changes)}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function EvidenceRow({ record, reviewReasons, chapterTitle, onNavigate, onUpdate }: {
  record: EvidenceRecord
  reviewReasons: string[]
  chapterTitle: string
  onNavigate: Props['onNavigate']
  onUpdate: (changes: Partial<EvidenceRecord>) => void
}) {
  const [editing, setEditing] = useState(false)
  const [authorText, setAuthorText] = useState(record.author_text ?? '')
  const [note, setNote] = useState(record.author_note ?? '')
  const cue = record.confidence >= .85 ? 'strong cue' : record.confidence >= .75 ? 'clear cue' : 'possible cue'
  const terms = [...(record.character_names ?? []), ...(record.named_entities ?? []).map(entity => entity.text)]
    .filter((term, index, all) => all.findIndex(value => value.toLocaleLowerCase() === term.toLocaleLowerCase()) === index)
  const reviewedAt = () => new Date().toISOString()

  function saveReview() {
    onUpdate({
      author_text: authorText.trim() || undefined,
      author_note: note.trim() || undefined,
      status: 'confirmed',
      reviewed_at: reviewedAt(),
    })
    setEditing(false)
  }

  return (
    <article className={`evidence-index-row ${record.status} ${record.pinned ? 'pinned' : ''}`}>
      <button className="evidence-index-source" type="button" onClick={() => onNavigate(record.section as Section, record.section_index, record.text)} title={`Open evidence in ${chapterTitle}`}>
        <span className="evidence-index-location">
          <strong>{chapterTitle}</strong>
          <span>¶ {record.paragraph_index + 1} · sentence {record.sentence_index + 1}</span>
        </span>
        <span className="evidence-index-content">
          <span className="evidence-index-badges">
            <i className={`evidence-kind ${record.kind}`}>{record.kind}</i>
            <i>{labelEvidenceType(record.evidence_type)}</i>
            <i title={record.rationale}>{cue}</i>
            {record.status === 'confirmed' && <i className="confirmed">confirmed</i>}
            {record.status === 'rejected' && <i className="rejected">rejected</i>}
            {record.pinned && <i className="pinned">pinned</i>}
          </span>
          <span className="evidence-index-text">{record.text}</span>
          {record.author_text && <span className="evidence-index-author-text"><b>Author interpretation</b>{record.author_text}</span>}
          {terms.length > 0 && <span className="evidence-index-terms">{terms.slice(0, 6).join(' · ')}</span>}
          {reviewReasons.length > 0 && <span className="evidence-index-review-reason">{reviewReasons.slice(0, 2).join(' · ')}</span>}
          {record.author_note && !editing && <span className="evidence-index-note">Note: {record.author_note}</span>}
        </span>
        <span className="evidence-index-open">Open →</span>
      </button>

      <div className="evidence-review-actions">
        <button type="button" className={record.status === 'confirmed' ? 'active confirm' : ''} onClick={() => onUpdate({ status: 'confirmed', reviewed_at: reviewedAt() })}>Confirm</button>
        <button type="button" className={record.status === 'rejected' ? 'active reject' : ''} onClick={() => onUpdate({ status: 'rejected', reviewed_at: reviewedAt() })}>Reject</button>
        <button type="button" className={editing ? 'active' : ''} onClick={() => setEditing(value => !value)}>{editing ? 'Cancel' : 'Edit'}</button>
        <button type="button" className={record.pinned ? 'active pin' : ''} onClick={() => onUpdate({ pinned: !record.pinned, reviewed_at: reviewedAt() })}>{record.pinned ? 'Pinned' : 'Pin'}</button>
      </div>

      {editing && <div className="evidence-review-editor">
        <label>
          <span>Your interpretation <small>The source sentence remains unchanged</small></span>
          <input value={authorText} onChange={event => setAuthorText(event.target.value)} placeholder="What this establishes in the story" />
        </label>
        <label>
          <span>Author note</span>
          <input value={note} onChange={event => setNote(event.target.value)} placeholder="Why it matters, what to revisit, or what it pays off" />
        </label>
        <button type="button" onClick={saveReview}>Save and confirm</button>
      </div>}
    </article>
  )
}

function labelEvidenceType(value: string): string {
  return value.replace(/_/g, ' ').replace(/^./, (letter: string) => letter.toUpperCase())
}
