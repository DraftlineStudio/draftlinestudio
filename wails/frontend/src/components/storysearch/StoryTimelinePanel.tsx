import { useEffect, useMemo, useState } from 'react'
import { BuildStoryTimeline } from '../../../wailsjs/go/main/App'
import { types } from '../../../wailsjs/go/models'
import type { BookData, Section } from '../../types/draftline'

interface Props {
  book: BookData
  onNavigate: (section: Section, sectionIndex: number, evidenceQuery: string) => void
}

export default function StoryTimelinePanel({ book, onNavigate }: Props) {
  const [result, setResult] = useState<types.StoryTimelineResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [textFilter, setTextFilter] = useState('')
  const [character, setCharacter] = useState('')
  const [location, setLocation] = useState('')
  const [eventType, setEventType] = useState('')
  const [timing, setTiming] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    void BuildStoryTimeline(book as types.BookData)
      .then(value => { if (!cancelled) setResult(value) })
      .catch(reason => { if (!cancelled) setError(String(reason)) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [book.file_path, book.analysis?.evidence?.content_hash])

  const events = useMemo(() => {
    const needle = textFilter.trim().toLowerCase()
    return (result?.events ?? []).filter(event => {
      if (character && !event.character_ids?.includes(character)) return false
      if (location && !event.locations?.some(item => item.text.toLowerCase() === location)) return false
      if (eventType && !event.event_types.includes(eventType)) return false
      if (timing === 'explicit' && event.time_kind === 'manuscript') return false
      if (timing && timing !== 'explicit' && event.time_kind !== timing) return false
      if (needle) {
        const haystack = [event.text, event.chapter_title, ...(event.character_names ?? []), ...(event.locations ?? []).map(item => item.text), ...(event.time_expressions ?? [])].join(' ').toLowerCase()
        if (!haystack.includes(needle)) return false
      }
      return true
    })
  }, [character, eventType, location, result, textFilter, timing])

  const activeFilters = !!(textFilter || character || location || eventType || timing)
  const maxChapterEvents = Math.max(1, ...(result?.chapters ?? []).map(chapter => chapter.event_count))

  function clearFilters() {
    setTextFilter('')
    setCharacter('')
    setLocation('')
    setEventType('')
    setTiming('')
  }

  function jumpToChapter(chapterIndex: number) {
    document.getElementById(`story-timeline-chapter-${chapterIndex}`)?.scrollIntoView({ block: 'start', behavior: 'smooth' })
  }

  if (loading) return <div className="story-timeline-state"><span className="story-search-spinner" />Assembling the source-backed timeline…</div>
  if (error || (result && !result.success)) return <div className="story-timeline-state error">{error || result?.error}</div>
  if (!result?.events.length) return <div className="story-timeline-state">No timeline events are indexed yet. Let the on-open story analysis finish, then return here.</div>

  let previousChapter = -1
  return (
    <section className="story-timeline" aria-label="Automatic story timeline">
      <header className="story-timeline-overview">
        <div>
          <span className="story-timeline-eyebrow">Automatic story timeline</span>
          <strong>{result.events.length} source-backed events across {result.chapters.length} {result.chapters.length === 1 ? 'chapter' : 'chapters'}</strong>
          <small>Manuscript order is preserved. Relative time is labeled but never converted into a guessed date.</small>
        </div>
        <div className="story-timeline-time-summary">
          <span><strong>{result.explicit_time_count}</strong> explicit time markers</span>
          <span><strong>{result.relative_time_count}</strong> relative transitions</span>
        </div>
      </header>

      <div className="story-timeline-density" aria-label="Event density by chapter">
        {result.chapters.map(chapter => (
          <button type="button" key={chapter.chapter_index} onClick={() => jumpToChapter(chapter.chapter_index)} title={`${chapter.chapter_title}: ${chapter.event_count} events, ${chapter.explicit_time_count} time markers`}>
            <i style={{ height: `${Math.max(12, chapter.event_count / maxChapterEvents * 100)}%` }} />
            <span>{chapter.chapter_title}</span>
          </button>
        ))}
      </div>

      <div className="story-timeline-filters">
        <input value={textFilter} onChange={event => setTextFilter(event.target.value)} placeholder="Filter events by any detail" aria-label="Filter timeline by text" />
        <TimelineSelect value={character} onChange={setCharacter} label="All characters" options={result.characters} />
        <TimelineSelect value={location} onChange={setLocation} label="All locations" options={result.locations.map(item => types.StoryTimelineFacet.createFrom({ ...item, id: item.label.toLowerCase() }))} />
        <TimelineSelect value={eventType} onChange={setEventType} label="All event types" options={result.event_types} />
        <select value={timing} onChange={event => setTiming(event.target.value)} aria-label="Filter by time certainty">
          <option value="">Any timing</option>
          <option value="explicit">Has an explicit time</option>
          <option value="anchored">Explicit time reference</option>
          <option value="relative">Relative timing</option>
          <option value="manuscript">Manuscript order only</option>
        </select>
        {activeFilters && <button type="button" className="story-timeline-clear" onClick={clearFilters}>Clear</button>}
        <span className="story-timeline-count">{events.length} shown</span>
      </div>

      <div className="story-timeline-events">
        {events.map(event => {
          const startsChapter = previousChapter !== event.chapter_index
          previousChapter = event.chapter_index
          return (
            <div className="story-timeline-event-wrap" id={startsChapter ? `story-timeline-chapter-${event.chapter_index}` : undefined} key={event.id}>
              {startsChapter && <div className="story-timeline-chapter-marker"><strong>{event.chapter_title}</strong><span>Chapter trail</span></div>}
              <button type="button" className="story-timeline-event" onClick={() => onNavigate(event.section as Section, event.section_index, event.source_text)} title={`Open source in ${event.chapter_title}`}>
                <span className={`story-timeline-node ${event.time_kind}`} />
                <span className="story-timeline-event-main">
                  <span className="story-timeline-event-labels">
                    <i className={event.primary_type}>{eventTypeLabel(event.primary_type)}</i>
                    <i className={`time ${event.time_kind}`}>{event.time_label}</i>
                    {event.status === 'confirmed' && <i className="confirmed">Confirmed</i>}
                    {event.pinned && <i className="pinned">Pinned</i>}
                  </span>
                  <strong>{event.text}</strong>
                  <small>{[...(event.character_names ?? []), ...(event.locations ?? []).map(item => item.text)].join(' · ') || 'No confirmed character or location attached'}</small>
                </span>
                <span className="story-timeline-open">Open source →</span>
              </button>
            </div>
          )
        })}
        {events.length === 0 && <div className="story-timeline-state">No events match these filters. <button type="button" onClick={clearFilters}>Clear filters</button></div>}
      </div>
    </section>
  )
}

function TimelineSelect({ value, onChange, label, options }: { value: string; onChange: (value: string) => void; label: string; options: types.StoryTimelineFacet[] }) {
  if (!options.length) return null
  return <select value={value} onChange={event => onChange(event.target.value)} aria-label={label}>
    <option value="">{label}</option>
    {options.map(option => <option value={option.id} key={option.id}>{option.label} ({option.count})</option>)}
  </select>
}

function eventTypeLabel(value: string): string {
  return value.replace(/_/g, ' ').replace(/^./, letter => letter.toUpperCase())
}
