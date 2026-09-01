import { FormEvent, Fragment, useEffect, useMemo, useRef, useState } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { SearchStory } from '../../wailsjs/go/main/App'
import { types } from '../../wailsjs/go/models'
import { useAppStore } from '../store/appStore'
import { useBookStore } from '../store/bookStore'
import type { Section } from '../types/draftline'
import EvidenceIndexPanel from './storysearch/EvidenceIndexPanel'

type SearchResult = types.StorySearchResult
type SearchMatch = types.StorySearchMatch

export default function StorySearchToolWindow() {
  const { book, setCurrentChapter, setViewMode } = useBookStore(useShallow(s => ({
    book: s.book,
    setCurrentChapter: s.setCurrentChapter,
    setViewMode: s.setViewMode,
  })))
  const { height, close, saveHeight } = useAppStore(useShallow(s => ({
    height: s.bottomToolHeight,
    close: s.closeBottomTool,
    saveHeight: s.setBottomToolHeight,
  })))
  const [panelHeight, setPanelHeight] = useState(height)
  const [query, setQuery] = useState('')
  const [activeView, setActiveView] = useState<'search' | 'evidence'>('search')
  const [result, setResult] = useState<SearchResult | null>(null)
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const firstRef = useRef<HTMLButtonElement>(null)
  const lastRef = useRef<HTMLButtonElement>(null)

  useEffect(() => { if (activeView === 'search') inputRef.current?.focus() }, [activeView])
  useEffect(() => { setResult(null) }, [book?.file_path])

  async function submit(event?: FormEvent) {
    event?.preventDefault()
    const cleaned = query.trim()
    if (!book || !cleaned || loading) return
    setLoading(true)
    try {
      setResult(await SearchStory(book as types.BookData, types.StorySearchRequest.createFrom({ query: cleaned, limit: 150 })))
    } catch (error) {
      setResult(types.StorySearchResult.createFrom({ query: cleaned, matches: [], total: 0, error: String(error) }))
    } finally {
      setLoading(false)
    }
  }

  function beginResize(event: React.PointerEvent<HTMLDivElement>) {
    event.preventDefault()
    const startY = event.clientY
    const startHeight = panelHeight
    const clamp = (value: number) => Math.max(170, Math.min(560, value))
    const onMove = (move: PointerEvent) => setPanelHeight(clamp(startHeight + startY - move.clientY))
    const onUp = (up: PointerEvent) => {
      const finalHeight = clamp(startHeight + startY - up.clientY)
      setPanelHeight(finalHeight)
      saveHeight(finalHeight)
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
    }
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }

  function navigateSource(section: Section, sectionIndex: number, evidenceQuery: string) {
    setViewMode('editor')
    setCurrentChapter(section, sectionIndex)
    window.setTimeout(() => {
      window.dispatchEvent(new CustomEvent('draftline:find-story-evidence', { detail: { query: evidenceQuery } }))
    }, 0)
  }

  function navigate(match: SearchMatch) {
    navigateSource(match.section as Section, match.section_index, match.matched_terms[0] ?? '')
  }

  const aliases = useMemo(() => result?.resolved_entities ?? [], [result])
  const truncated = result ? result.total > result.matches.length : false

  return (
    <section className="story-search-window" style={{ height: panelHeight }} aria-label="Story Search">
      <div className="story-search-resizer" onPointerDown={beginResize} />
      <header className="story-search-header">
        <button type="button" className={`story-search-tab ${activeView === 'search' ? 'active' : ''}`} onClick={() => setActiveView('search')}>
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.25">
            <circle cx="5" cy="5" r="3.4" /><path d="M7.5 7.5 11 11" />
          </svg>
          Story Search
        </button>
        <button type="button" className={`story-search-tab ${activeView === 'evidence' ? 'active' : ''}`} onClick={() => setActiveView('evidence')}>
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.15">
            <path d="M2 1.5h8v9H2zM4 4h4M4 6h4M4 8h2.5" />
          </svg>
          Evidence Index
          {!!book?.analysis?.evidence?.records.length && <small>{book.analysis.evidence.records.length}</small>}
        </button>
        <span className="story-search-header-hint">Local analysis · no AI</span>
        <button className="story-search-close" onClick={close} title="Close Story Search" aria-label="Close Story Search">×</button>
      </header>

      {activeView === 'search' && <form className="story-search-form" onSubmit={submit}>
        <div className="story-search-input-wrap">
          <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.3">
            <circle cx="5.5" cy="5.5" r="3.8" /><path d="M8.3 8.3 12 12" />
          </svg>
          <input
            ref={inputRef}
            value={query}
            onChange={event => setQuery(event.target.value)}
            placeholder={'Search names, places, objects, or an exact phrase'}
            aria-label="Search names, details, and phrases across the story"
          />
          {query && <button type="button" className="story-search-clear" onClick={() => { setQuery(''); setResult(null); inputRef.current?.focus() }} aria-label="Clear search">×</button>}
        </div>
        <button className="story-search-submit" type="submit" disabled={!query.trim() || loading}>
          {loading ? 'Searching…' : 'Search'}
        </button>
      </form>}

      {activeView === 'search' && aliases.length > 0 && (
        <div className="story-search-aliases">
          <span>Aliases included</span>
          {aliases.map(entity => (
            <span className="story-search-alias-chip" key={entity.id} title={entity.aliases.join(', ')}>
              {entity.canonical}<small>{entity.aliases.length} names</small>
            </span>
          ))}
        </div>
      )}

      {activeView === 'search' ? <div className="story-search-body">
        {!result && !loading && (
          <div className="story-search-empty">
            <strong>Find story evidence without leaving the editor.</strong>
            <span>Use several terms to find scenes where those details converge. Confirmed character names automatically include their aliases.</span>
          </div>
        )}
        {loading && <div className="story-search-empty"><span className="story-search-spinner" />Searching the manuscript…</div>}
        {result?.error && <div className="story-search-empty story-search-error">{result.error}</div>}
        {result && !result.error && result.total === 0 && (
          <div className="story-search-empty">
            <strong>No scene contains all of those terms.</strong>
            <span>Try fewer terms, or put an exact phrase in quotation marks.</span>
          </div>
        )}
        {result && !result.error && result.total > 0 && (
          <>
            <div className="story-search-summary">
              <span><strong>{result.total}</strong> {result.total === 1 ? 'scene' : 'scenes'} in manuscript order{truncated ? ` · showing first ${result.matches.length}` : ''}</span>
              <div>
                <button type="button" onClick={() => firstRef.current?.scrollIntoView({ block: 'nearest' })}>First occurrence</button>
                <button type="button" onClick={() => lastRef.current?.scrollIntoView({ block: 'nearest' })}>Last occurrence</button>
              </div>
            </div>
            <div className="story-search-results">
              {result.matches.map((match, index) => (
                <button
                  ref={index === 0 ? firstRef : index === result.matches.length - 1 ? lastRef : undefined}
                  className="story-search-result"
                  type="button"
                  key={`${match.chapter_index}-${match.scene_index}`}
                  onClick={() => navigate(match)}
                  title={`Open ${match.chapter_title}`}
                >
                  <span className="story-search-result-location">
                    <strong>{match.chapter_title}</strong>
                    <span>Scene {match.scene_index + 1}</span>
                  </span>
                  <span className="story-search-result-excerpt">
                    {!!match.evidence?.length && (
                      <span className="story-search-evidence-badges">
                        {match.evidence.slice(0, 4).map(item => <i key={item.id} className={item.kind}>{labelEvidenceType(item.evidence_type)}</i>)}
                      </span>
                    )}
                    <HighlightedExcerpt text={match.excerpt} terms={match.matched_terms} />
                    {!!match.additional_hits && <small> +{match.additional_hits} more evidence {match.additional_hits === 1 ? 'paragraph' : 'paragraphs'}</small>}
                  </span>
                  <span className="story-search-result-open">Open chapter →</span>
                </button>
              ))}
            </div>
          </>
        )}
      </div> : book ? <EvidenceIndexPanel book={book} onNavigate={navigateSource} /> : null}
    </section>
  )
}

function labelEvidenceType(value: string): string {
  return value.replace(/_/g, ' ').replace(/^./, (letter: string) => letter.toUpperCase())
}

function HighlightedExcerpt({ text, terms }: { text: string; terms: string[] }) {
  const normalized = terms.filter(Boolean).sort((a, b) => b.length - a.length)
  if (normalized.length === 0) return <>{text}</>
  const escaped = normalized.map(term => term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
  const parts = text.split(new RegExp(`(${escaped.join('|')})`, 'gi'))
  const lower = new Set(normalized.map(term => term.toLowerCase()))
  return <>{parts.map((part, index) => lower.has(part.toLowerCase()) ? <mark key={index}>{part}</mark> : <Fragment key={index}>{part}</Fragment>)}</>
}
