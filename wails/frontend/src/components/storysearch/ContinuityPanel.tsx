import { useEffect, useMemo, useState } from 'react'
import { BuildContinuityReport } from '../../../wailsjs/go/main/App'
import { types } from '../../../wailsjs/go/models'
import { useBookStore } from '../../store/bookStore'
import type { BookData, Section } from '../../types/draftline'

type SignalFilter = 'all' | 'review' | 'observations' | 'decided'

interface Props {
  book: BookData
  onNavigate: (section: Section, sectionIndex: number, evidenceQuery: string) => void
  /** Reports outstanding counts so the bar header can show them. */
  onCounts?: (counts: { review: number; info: number } | null) => void
}

export default function ContinuityPanel({ book, onNavigate, onCounts }: Props) {
  const setDecision = useBookStore(state => state.setContinuityDecision)
  const [report, setReport] = useState<types.ContinuityReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [textFilter, setTextFilter] = useState('')
  const [filter, setFilter] = useState<SignalFilter>('all')
  const [selectedID, setSelectedID] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    void BuildContinuityReport(book as types.BookData)
      .then(value => { if (!cancelled) setReport(value) })
      .catch(reason => { if (!cancelled) setError(String(reason)) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [book.file_path, book.analysis?.evidence?.content_hash, book.analysis?.continuity?.decisions])

  useEffect(() => {
    onCounts?.(report ? { review: report.review_count, info: report.info_count } : null)
    return () => onCounts?.(null)
  }, [report, onCounts])

  const signals = useMemo(() => {
    const needle = textFilter.trim().toLowerCase()
    return (report?.signals ?? []).filter(signal => {
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
  }, [report, filter, textFilter])

  // Keep a valid selection as filters narrow the queue.
  const selected = signals.find(signal => signal.id === selectedID) ?? signals[0] ?? null
  const selectedIndex = selected ? signals.findIndex(signal => signal.id === selected.id) : -1

  if (loading) return <div className="continuity-state"><span className="story-search-spinner" />Comparing the story fingerprint…</div>
  if (error || (report && !report.success)) return <div className="continuity-state error">{error || report?.error}</div>
  if (!report) return null

  const decide = (status: 'reviewed' | 'dismissed') => {
    if (!selected) return
    // Advance to the next open question so the queue keeps moving.
    const next = signals.find(signal => signal.id !== selected.id && !signal.status)
    setDecision(selected.id, selected.status === status ? null : status)
    if (next && selected.status !== status) setSelectedID(next.id)
  }

  return (
    <section className="continuity-panel" aria-label="Continuity review">
      <div className="continuity-split">
        <div className="continuity-queue">
          <div className="continuity-queue-head">
            <div className="continuity-queue-search">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                <circle cx="11" cy="11" r="7" /><path d="M21 21l-4.3-4.3" />
              </svg>
              <input
                value={textFilter}
                onChange={event => setTextFilter(event.target.value)}
                placeholder="Filter questions"
                aria-label="Filter continuity questions"
              />
            </div>
            {([
              ['all', 'All'],
              ['review', 'Review'],
              ['observations', 'Observations'],
              ['decided', 'Decided'],
            ] as const).map(([value, label]) => (
              <button
                type="button"
                key={value}
                className={filter === value ? 'active' : ''}
                onClick={() => setFilter(value)}
              >
                {label}
              </button>
            ))}
          </div>

          <div className="continuity-queue-list">
            {signals.map(signal => (
              <button
                type="button"
                key={signal.id}
                className={`continuity-row ${selected?.id === signal.id ? 'selected' : ''} ${signal.status ? 'decided' : ''} ${signal.severity}`}
                onClick={() => setSelectedID(signal.id)}
                title={signal.title}
              >
                <i className={signal.severity}>{signal.severity === 'review' ? '!' : 'i'}</i>
                <span>{signal.title}</span>
                {signal.status && <b className={signal.status}>{signal.status === 'reviewed' ? 'Reviewed' : 'Dismissed'}</b>}
                <em>{categoryLabel(signal.category)}</em>
                <small>{Math.round(signal.confidence * 100)}%</small>
              </button>
            ))}
            {signals.length === 0 && (
              <div className="continuity-state">
                {report.signals.length === 0
                  ? 'No continuity questions were raised for this manuscript.'
                  : 'No questions match these filters.'}
              </div>
            )}
          </div>

          <div className="continuity-queue-foot">
            {report.chapters_checked} {report.chapters_checked === 1 ? 'chapter' : 'chapters'} checked · every conclusion is source-backed
          </div>
        </div>

        {selected ? (
          <div className="continuity-detail">
            <div className="continuity-detail-head">
              <span className={`continuity-sev ${selected.severity}`}>{selected.severity === 'review' ? 'Review' : 'Observation'}</span>
              <span className="continuity-tag">{categoryLabel(selected.category)}</span>
              <span className="continuity-tag">{kindLabel(selected.kind)}</span>
              {selected.status && <span className={`continuity-tag ${selected.status}`}>{selected.status === 'reviewed' ? 'Reviewed' : 'Dismissed'}</span>}
              <div className="continuity-detail-spacer" />
              <small>{Math.round(selected.confidence * 100)}% cue strength</small>
            </div>

            <h3 className="continuity-detail-title">{selected.title}</h3>
            <p className="continuity-detail-body">{selected.detail}</p>

            <div className="continuity-detail-sources">
              {!!selected.sources?.length && (
                <span className="story-graph-rail-label">
                  {selected.sources.length > 1 ? 'Paired sources' : 'Paired source'}
                </span>
              )}
              {(selected.sources ?? []).map((source, index) => (
                <button
                  type="button"
                  className="continuity-source"
                  key={`${source.evidence_id}-${source.chapter_index}-${index}`}
                  onClick={() => onNavigate(source.section as Section, source.section_index, source.text ?? '')}
                  title={`Open source in ${source.chapter_title}`}
                >
                  <span>{source.chapter_title}{(selected.sources?.length ?? 0) > 1 ? ` · source ${index + 1}` : ''}</span>
                  <em>“{source.text || `Open ${source.chapter_title}`}”</em>
                </button>
              ))}
            </div>

            <div className="continuity-detail-foot">
              <button
                type="button"
                className="continuity-primary"
                onClick={() => {
                  const source = selected.sources?.[0]
                  if (source) onNavigate(source.section as Section, source.section_index, source.text ?? '')
                }}
                disabled={!selected.sources?.length}
              >
                Open source
              </button>
              <button
                type="button"
                className={selected.status === 'reviewed' ? 'active' : ''}
                onClick={() => decide('reviewed')}
              >
                {selected.status === 'reviewed' ? 'Reviewed ✓' : 'Mark reviewed'}
              </button>
              <button
                type="button"
                className={selected.status === 'dismissed' ? 'active' : ''}
                onClick={() => decide('dismissed')}
              >
                {selected.status === 'dismissed' ? 'Dismissed ✓' : 'Dismiss'}
              </button>
              <div className="continuity-detail-spacer" />
              <small>Question {selectedIndex + 1} of {signals.length}</small>
            </div>
          </div>
        ) : (
          <div className="continuity-detail continuity-detail-empty">
            <span>Select a question to see its paired source.</span>
          </div>
        )}
      </div>
    </section>
  )
}

function categoryLabel(value: string): string {
  if (value === 'knowledge') return 'Who knows what'
  if (value === 'facts') return 'Story facts'
  if (value === 'chronology') return 'Chronology'
  if (value === 'structure') return 'Story structure'
  return 'Characters'
}

function kindLabel(value: string): string {
  return value.replace(/-/g, ' ').replace(/^./, letter => letter.toUpperCase())
}
