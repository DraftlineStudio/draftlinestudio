import { useEffect, useMemo, useState } from 'react'
import { BuildContinuityReport } from '../../../wailsjs/go/main/App'
import { types } from '../../../wailsjs/go/models'
import type { BookData, Section } from '../../types/draftline'

interface Props {
  book: BookData
  onNavigate: (section: Section, sectionIndex: number, evidenceQuery: string) => void
}

export default function ContinuityPanel({ book, onNavigate }: Props) {
  const [report, setReport] = useState<types.ContinuityReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [textFilter, setTextFilter] = useState('')
  const [severity, setSeverity] = useState('')
  const [category, setCategory] = useState('')
  const [character, setCharacter] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    void BuildContinuityReport(book as types.BookData)
      .then(value => { if (!cancelled) setReport(value) })
      .catch(reason => { if (!cancelled) setError(String(reason)) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [book.file_path, book.analysis?.evidence?.content_hash])

  const signals = useMemo(() => {
    const needle = textFilter.trim().toLowerCase()
    return (report?.signals ?? []).filter(signal => {
      if (severity && signal.severity !== severity) return false
      if (category && signal.category !== category) return false
      if (character && !signal.character_ids?.includes(character)) return false
      if (needle) {
        const sourceText = signal.sources?.map(source => `${source.chapter_title} ${source.text}`).join(' ') ?? ''
        const haystack = `${signal.title} ${signal.detail} ${(signal.character_names ?? []).join(' ')} ${sourceText}`.toLowerCase()
        if (!haystack.includes(needle)) return false
      }
      return true
    })
  }, [category, character, report, severity, textFilter])

  const activeFilters = !!(textFilter || severity || category || character)
  function clearFilters() {
    setTextFilter('')
    setSeverity('')
    setCategory('')
    setCharacter('')
  }

  if (loading) return <div className="continuity-state"><span className="story-search-spinner" />Comparing the story fingerprint…</div>
  if (error || (report && !report.success)) return <div className="continuity-state error">{error || report?.error}</div>
  if (!report) return null

  return (
    <section className="continuity-panel" aria-label="Continuity review">
      <header className="continuity-overview">
        <div>
          <span className="continuity-eyebrow">Continuity review</span>
          <strong>{report.review_count ? `${report.review_count} ${report.review_count === 1 ? 'question' : 'questions'} worth reviewing` : 'No strong continuity concerns found'}</strong>
          <small>{report.chapters_checked} chapters checked · every conclusion is source-backed and intentionally phrased as a review prompt</small>
        </div>
        <div className="continuity-summary">
          <span className="review"><strong>{report.review_count}</strong> review</span>
          <span><strong>{report.info_count}</strong> observations</span>
        </div>
      </header>

      <div className="continuity-filters">
        <input value={textFilter} onChange={event => setTextFilter(event.target.value)} placeholder="Filter by character, detail, or source text" aria-label="Filter continuity signals" />
        <select value={severity} onChange={event => setSeverity(event.target.value)} aria-label="Filter by importance">
          <option value="">Any importance</option>
          <option value="review">Worth reviewing</option>
          <option value="info">Observations</option>
        </select>
        <ContinuitySelect value={category} onChange={setCategory} label="All categories" options={report.categories} />
        <ContinuitySelect value={character} onChange={setCharacter} label="All characters" options={report.characters} />
        {activeFilters && <button type="button" className="continuity-clear" onClick={clearFilters}>Clear</button>}
        <span className="continuity-count">{signals.length} shown</span>
      </div>

      <div className="continuity-signals">
        {signals.map(signal => (
          <article className={`continuity-signal ${signal.severity}`} key={signal.id}>
            <span className={`continuity-signal-icon ${signal.kind}`}>{signal.severity === 'review' ? '!' : 'i'}</span>
            <div className="continuity-signal-main">
              <span className="continuity-signal-labels">
                <i>{categoryLabel(signal.category)}</i>
                <i>{kindLabel(signal.kind)}</i>
                <i className="confidence">{Math.round(signal.confidence * 100)}% cue strength</i>
              </span>
              <strong>{signal.title}</strong>
              <p>{signal.detail}</p>
              {!!signal.sources?.length && <div className="continuity-sources">
                {(signal.sources ?? []).map((source, index) => (
                  <button type="button" key={`${source.evidence_id}-${source.chapter_index}-${index}`} onClick={() => onNavigate(source.section as Section, source.section_index, source.text ?? '')} title={`Open source in ${source.chapter_title}`}>
                    <span>{(signal.sources?.length ?? 0) > 1 ? `Source ${index + 1}` : 'Source'} · {source.chapter_title}</span>
                    <strong>{source.text || `Open ${source.chapter_title}`}</strong>
                    <i>Open →</i>
                  </button>
                ))}
              </div>}
            </div>
          </article>
        ))}
        {signals.length === 0 && <div className="continuity-state">No continuity signals match these filters. {activeFilters && <button type="button" onClick={clearFilters}>Clear filters</button>}</div>}
      </div>
    </section>
  )
}

function ContinuitySelect({ value, onChange, label, options }: { value: string; onChange: (value: string) => void; label: string; options: types.ContinuityFacet[] }) {
  if (!options.length) return null
  return <select value={value} onChange={event => onChange(event.target.value)} aria-label={label}>
    <option value="">{label}</option>
    {options.map(option => <option value={option.id} key={option.id}>{option.label} ({option.count})</option>)}
  </select>
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
