import { useEffect, useMemo, useState } from 'react'
import { BuildContinuityReport } from '../../../wailsjs/go/main/App'
import { types } from '../../../wailsjs/go/models'
import { useBookStore } from '../../store/bookStore'
import type { BookData, Section } from '../../types/draftline'
import {
  categoryLabel,
  evidenceCards,
  filterSignals,
  issueMeta,
  kindLabel,
  signalTone,
  whereLabel,
  type ContinuityFilter,
} from './continuityModel'

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
  const [filter, setFilter] = useState<ContinuityFilter>('all')
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

  const signals = useMemo(
    () => filterSignals(report?.signals ?? [], filter, textFilter),
    [report, filter, textFilter],
  )

  // Keep a valid selection as filters narrow the queue.
  const selected = signals.find(signal => signal.id === selectedID) ?? signals[0] ?? null
  const selectedIndex = selected ? signals.findIndex(signal => signal.id === selected.id) : -1
  const cards = useMemo(() => (selected ? evidenceCards(selected) : []), [selected])

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

  const goToSource = (source: types.ContinuitySource) =>
    onNavigate(source.section as Section, source.section_index, source.text ?? '')

  return (
    <section className="continuity-panel" aria-label="Continuity review">
      <div className="continuity-split">
        <div className="continuity-list">
          <div className="continuity-filters">
            <div className="continuity-search">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                <circle cx="11" cy="11" r="7" /><path d="M21 21l-4.3-4.3" />
              </svg>
              <input
                value={textFilter}
                onChange={event => setTextFilter(event.target.value)}
                placeholder="Filter issues"
                aria-label="Filter continuity issues"
              />
            </div>
            <div className="continuity-chips">
              {([
                ['all', 'All'],
                ['review', 'Review'],
                ['observations', 'Observations'],
                ['decided', 'Decided'],
              ] as const).map(([value, label]) => (
                <button
                  type="button"
                  key={value}
                  className={`continuity-chip ${filter === value ? 'active' : ''}`}
                  onClick={() => setFilter(value)}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>

          <div className="continuity-issues">
            {signals.map(signal => (
              <button
                type="button"
                key={signal.id}
                className={`continuity-issue ${selected?.id === signal.id ? 'selected' : ''} ${signal.status ? 'decided' : ''}`}
                onClick={() => setSelectedID(signal.id)}
                title={signal.title}
              >
                <span className={`continuity-dot tone-${signalTone(signal)}`} />
                <span className="continuity-issue-text">
                  <span className="continuity-issue-title">{signal.title}</span>
                  <span className="continuity-issue-meta">{issueMeta(signal)}</span>
                </span>
                {signal.status && (
                  <b className={`continuity-issue-status ${signal.status}`}>
                    {signal.status === 'reviewed' ? 'Reviewed' : 'Dismissed'}
                  </b>
                )}
              </button>
            ))}
            {signals.length === 0 && (
              <div className="continuity-state">
                {report.signals.length === 0
                  ? 'No continuity questions were raised for this manuscript.'
                  : 'No issues match these filters.'}
              </div>
            )}
          </div>

          <div className="continuity-list-foot">
            {report.chapters_checked} {report.chapters_checked === 1 ? 'chapter' : 'chapters'} checked · every conclusion is source-backed
          </div>
        </div>

        {selected ? (
          <div className="continuity-detail">
            <div className="continuity-detail-head">
              <span className={`continuity-kind tone-${signalTone(selected)}`}>{kindLabel(selected.kind)}</span>
              <span className="continuity-where">{whereLabel(selected)} · {categoryLabel(selected.category)}</span>
              {selected.status && (
                <span className={`continuity-decided-chip ${selected.status}`}>
                  {selected.status === 'reviewed' ? 'Reviewed' : 'Dismissed'}
                </span>
              )}
              <div className="continuity-detail-spacer" />
              <small>{Math.round(selected.confidence * 100)}% cue strength</small>
            </div>

            <div className="continuity-detail-desc">{selected.title}</div>
            <p className="continuity-detail-body">{selected.detail}</p>

            {cards.length > 0 && (
              <div className="continuity-evidence">
                {cards.map((card, index) => (
                  <button
                    type="button"
                    key={`${card.source.evidence_id}-${card.source.chapter_index}-${index}`}
                    className={`continuity-card ${card.role}`}
                    onClick={() => goToSource(card.source)}
                    title={`Open source in ${card.source.chapter_title}`}
                  >
                    <span className="continuity-card-label">{card.label}</span>
                    <em className="continuity-card-quote">“{card.source.text || `Open ${card.source.chapter_title}`}”</em>
                  </button>
                ))}
              </div>
            )}

            <div className="continuity-detail-spacer" />

            <div className="continuity-actions">
              <button
                type="button"
                className="continuity-primary"
                onClick={() => { if (cards[0]) goToSource(cards[0].source) }}
                disabled={cards.length === 0}
              >
                Go to source
              </button>
              <button
                type="button"
                className={`continuity-action ${selected.status === 'reviewed' ? 'active' : ''}`}
                onClick={() => decide('reviewed')}
              >
                {selected.status === 'reviewed' ? 'Reviewed ✓' : 'Mark reviewed'}
              </button>
              <button
                type="button"
                className={`continuity-action ${selected.status === 'dismissed' ? 'active' : ''}`}
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
            <span>Select an issue to compare its claim against what the story established.</span>
          </div>
        )}
      </div>
    </section>
  )
}
