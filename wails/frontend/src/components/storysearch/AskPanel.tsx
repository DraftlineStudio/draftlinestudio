import { FormEvent, Fragment, useEffect, useMemo, useRef, useState } from 'react'
import { SearchStory } from '../../../wailsjs/go/main/App'
import { types } from '../../../wailsjs/go/models'
import type { BookData, Section } from '../../types/draftline'
import { isConfirmedCharacter } from '../../utils/characterStatus'

type SearchResult = types.StorySearchResult
type SearchMatch = types.StorySearchMatch
type SearchStarter = { label: string; detail: string; query: string }
type SearchStarterGroup = { id: string; label: string; starters: SearchStarter[] }
type SortOrder = 'earliest' | 'latest'

interface Props {
  book: BookData
  onNavigate: (section: Section, sectionIndex: number, evidenceQuery: string) => void
}

export default function AskPanel({ book, onNavigate }: Props) {
  const [query, setQuery] = useState('')
  const [result, setResult] = useState<SearchResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [chapterFilter, setChapterFilter] = useState<number | null>(null)
  const [order, setOrder] = useState<SortOrder>('earliest')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => { inputRef.current?.focus() }, [])
  useEffect(() => { setResult(null) }, [book?.file_path])

  async function runSearch(value: string) {
    const cleaned = value.trim()
    if (!book || !cleaned || loading) return
    setLoading(true)
    setChapterFilter(null)
    try {
      setResult(await SearchStory(book as types.BookData, types.StorySearchRequest.createFrom({ query: cleaned, limit: 150 })))
    } catch (error) {
      setResult(types.StorySearchResult.createFrom({ query: cleaned, matches: [], total: 0, error: String(error) }))
    } finally {
      setLoading(false)
    }
  }

  function submit(event?: FormEvent) {
    event?.preventDefault()
    void runSearch(query)
  }

  function navigate(match: SearchMatch) {
    onNavigate(match.section as Section, match.section_index, match.matched_terms[0] ?? '')
  }

  const aliases = result?.resolved_entities ?? []
  const insight = result?.insight
  const starterGroups = useMemo(() => buildSearchStarters(book), [book])

  const matches = useMemo(() => {
    const list = result?.matches ?? []
    const filtered = chapterFilter === null ? list : list.filter(match => match.chapter_index === chapterFilter)
    return order === 'latest' ? [...filtered].reverse() : filtered
  }, [result, chapterFilter, order])

  return (
    <div className="ask-panel">
      <form className="ask-panel-query" onSubmit={submit}>
        <div className="story-search-input-wrap">
          <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.3">
            <circle cx="5.5" cy="5.5" r="3.8" /><path d="M8.3 8.3 12 12" />
          </svg>
          <input
            ref={inputRef}
            value={query}
            onChange={event => setQuery(event.target.value)}
            placeholder="Ask a question about your story, or search for any detail"
            aria-label="Trace names, details, and phrases across the story"
          />
          {query && <button type="button" className="story-search-clear" onClick={() => { setQuery(''); setResult(null); inputRef.current?.focus() }} aria-label="Clear search">×</button>}
        </div>
        <button className="story-search-submit" type="submit" disabled={!query.trim() || loading}>
          {loading ? 'Searching…' : 'Ask'}
        </button>
        {aliases.length > 0 && (
          <div className="ask-panel-aliases">
            <span>Aliases included</span>
            {aliases.map(entity => (
              <span className="story-search-alias-chip" key={entity.id} title={entity.aliases.join(', ')}>
                {entity.canonical}<small>{entity.aliases.length} {entity.aliases.length === 1 ? 'name' : 'names'}</small>
              </span>
            ))}
          </div>
        )}
        {insight?.interpreted_query && <span className="ask-panel-interpreted">Interpreted as: {insight.interpreted_query}</span>}
      </form>

      {!result && !loading && <SearchStarterPanel groups={starterGroups} onSelect={value => { setQuery(value); void runSearch(value) }} />}
      {loading && <div className="story-search-empty"><span className="story-search-spinner" />Building the source trail…</div>}
      {result?.error && <div className="story-search-empty story-search-error">{result.error}</div>}
      {result && !result.error && result.total === 0 && (
        <div className="story-search-empty">
          <strong>No source trail connects all of those details.</strong>
          <span>Try fewer details, a confirmed alias, or put an exact phrase in quotation marks.</span>
        </div>
      )}

      {result && !result.error && result.total > 0 && (
        <div className="ask-panel-split">
          <aside className="ask-panel-answer">
            {!!insight?.signals?.length && (
              <>
                <span className="story-graph-rail-label">Answer</span>
                {insight.signals.map((signal, index) => (
                  <div className={`ask-panel-signal ${signal.kind}`} key={`${signal.title}-${index}`}>
                    <i>{signal.kind === 'answer' ? '✓' : signal.kind === 'attention' ? '!' : 'i'}</i>
                    <span><strong>{signal.title}</strong><small>{signal.detail}</small></span>
                  </div>
                ))}
              </>
            )}

            {!!insight?.knowledge_states?.length && (
              <>
                <span className="story-graph-rail-label">What Draftline can prove</span>
                <div className="ask-panel-proofs">
                  {insight.knowledge_states.map((state, index) => (
                    <button
                      type="button"
                      key={`${state.evidence_id}-${state.state}-${index}`}
                      onClick={() => onNavigate(state.section as Section, state.section_index, state.text)}
                      title={`Open source in ${state.chapter_title}`}
                    >
                      <i className={state.state}>{knowledgeIcon(state.state)}</i>
                      <span>{knowledgeHeadline(state)} — “{state.text}”</span>
                      <small>{state.chapter_title}</small>
                    </button>
                  ))}
                </div>
              </>
            )}

            {!!insight?.related_terms?.length && (
              <>
                <span className="story-graph-rail-label">Details traveling with it</span>
                <div className="ask-panel-related">
                  {insight.related_terms.map(term => (
                    <button type="button" key={`${term.label}-${term.text}`} title={`Trace ${term.text} · ${term.label}`} onClick={() => { setQuery(term.text); void runSearch(term.text) }}>
                      {term.text}<small>{term.count}</small>
                    </button>
                  ))}
                </div>
              </>
            )}

            <div className="ask-panel-answer-foot">
              <span>{trailHeadline(result.total, insight?.chapter_count ?? 0)}</span>
              {!!insight?.chapters?.length && (
                <small>
                  First mention {insight.chapters[0].chapter_title} · Last mention {insight.chapters[insight.chapters.length - 1].chapter_title}
                </small>
              )}
              {result.total > result.matches.length && (
                <small>Showing the first {result.matches.length} source scenes; counts cover all {result.total}.</small>
              )}
            </div>
          </aside>

          <div className="ask-panel-trail">
            <div className="ask-panel-trail-head">
              <span className="story-graph-rail-label">Source trail · manuscript order</span>
              <span />
              <button type="button" className={order === 'earliest' ? 'active' : ''} onClick={() => setOrder('earliest')}>Earliest</button>
              <button type="button" className={order === 'latest' ? 'active' : ''} onClick={() => setOrder('latest')}>Latest shown</button>
            </div>

            {!!insight?.chapters?.length && (
              <div className="ask-panel-coverage">
                {insight.chapters.map(chapter => (
                  <button
                    type="button"
                    key={chapter.chapter_index}
                    className={chapterFilter === chapter.chapter_index ? 'active' : ''}
                    onClick={() => setChapterFilter(chapterFilter === chapter.chapter_index ? null : chapter.chapter_index)}
                    title={`${chapter.evidence_count} indexed evidence records`}
                  >
                    {chapter.chapter_title}<small>{chapter.occurrences}</small>
                  </button>
                ))}
                {chapterFilter !== null && <button type="button" className="ask-panel-coverage-clear" onClick={() => setChapterFilter(null)}>Clear</button>}
              </div>
            )}

            <div className="ask-panel-occurrences">
              {matches.map(match => (
                <button
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
              {matches.length === 0 && (
                <div className="story-search-empty">
                  <span>No source scenes in that chapter.</span>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function SearchStarterPanel({ groups, onSelect }: { groups: SearchStarterGroup[]; onSelect: (query: string) => void }) {
  const [active, setActive] = useState(groups[0]?.id ?? '')
  useEffect(() => {
    if (!groups.some(group => group.id === active)) setActive(groups[0]?.id ?? '')
  }, [active, groups])
  const selected = groups.find(group => group.id === active) ?? groups[0]
  return (
    <div className="story-search-start">
      <div className="story-search-start-lede">
        <strong>What do you want to know about this story?</strong>
        <span>You do not need search syntax. Choose a trail Draftline found in this manuscript, or ask in your own words.</span>
      </div>
      <div className="story-search-start-tabs" role="tablist" aria-label="Story search suggestions">
        {groups.map(group => <button type="button" role="tab" aria-selected={group.id === selected?.id} className={group.id === selected?.id ? 'active' : ''} key={group.id} onClick={() => setActive(group.id)}>{group.label}</button>)}
      </div>
      <div className="story-search-starters">
        {selected?.starters.map((starter, index) => (
          <button type="button" key={`${starter.query}-${index}`} onClick={() => onSelect(starter.query)}>
            <strong>{starter.label}</strong>
            <span>{starter.detail}</span>
            <i>Explore →</i>
          </button>
        ))}
      </div>
    </div>
  )
}

function buildSearchStarters(book: BookData | null): SearchStarterGroup[] {
  if (!book) return []
  const characters = [...(book.story_bible?.characters ?? [])]
    .filter(isConfirmedCharacter)
    .sort((a, b) => (b.mention_count ?? 0) - (a.mention_count ?? 0))
    .slice(0, 6)
    .map(character => ({
      label: `Follow ${character.name}`,
      detail: `See where ${character.name} appears and which details travel with them.`,
      query: character.name,
    }))
  const records = book.analysis?.evidence?.records ?? []
  const knowledge: SearchStarter[] = []
  const discoveries: SearchStarter[] = []
  const seenKnowledge = new Set<string>()
  for (const record of records) {
    if (knowledge.length < 6) {
      for (const state of record.knowledge_states ?? []) {
        const name = state.character_names?.[0]
        if (!name) continue
        const key = `${name}:${state.state}`
        if (seenKnowledge.has(key)) continue
        seenKnowledge.add(key)
        knowledge.push({ label: knowledgeStarterLabel(name, state.state), detail: record.text, query: `"${record.text.replace(/"/g, '')}"` })
        if (knowledge.length >= 6) break
      }
    }
    if (discoveries.length < 6 && record.evidence_type === 'discovery') {
      discoveries.push({ label: `A discovery in ${chapterLabel(book, record.section, record.section_index)}`, detail: record.text, query: `"${record.text.replace(/"/g, '')}"` })
    }
    if (knowledge.length >= 6 && discoveries.length >= 6) break
  }
  return [
    { id: 'characters', label: 'Characters', starters: characters },
    { id: 'knowledge', label: 'Who knows what', starters: knowledge },
    { id: 'discoveries', label: 'Discoveries', starters: discoveries },
  ].filter(group => group.starters.length > 0)
}

function knowledgeStarterLabel(name: string, state: string): string {
  if (state === 'shared') return `What did ${name} share?`
  if (state === 'withheld') return `What did ${name} withhold?`
  if (state === 'learned') return `What did ${name} learn?`
  if (state === 'does_not_know') return `What didn't ${name} know?`
  if (state === 'believes' || state === 'does_not_believe') return `What did ${name} believe?`
  if (state === 'suspects' || state === 'does_not_suspect') return `What did ${name} suspect?`
  return `What did ${name} know?`
}

function chapterLabel(book: BookData, section: string, index: number): string {
  const chapters = section === 'front_matter' ? book.front_matter : section === 'back_matter' ? book.back_matter : book.body
  return chapters?.[index]?.title || `chapter ${index + 1}`
}

function labelEvidenceType(value: string): string {
  return value.replace(/_/g, ' ').replace(/^./, (letter: string) => letter.toUpperCase())
}

function knowledgeIcon(state: string): string {
  if (state === 'shared') return '→'
  if (state === 'withheld') return '×'
  if (state === 'learned') return '+'
  if (state === 'does_not_know') return '?'
  if (state === 'attempts_to_recall') return '…'
  if (state === 'believes' || state === 'suspects') return '~'
  return '•'
}

function knowledgeHeadline(state: types.StorySearchKnowledgeState): string {
  const characters = state.character_names?.join(' and ') || 'A confirmed character'
  const counterparties = state.counterparty_names?.join(' and ')
  if (state.state === 'shared') return counterparties ? `${characters} shares this with ${counterparties}` : `${characters} communicates this`
  if (state.state === 'withheld') return counterparties ? `${characters} withholds this from ${counterparties}` : `${characters} withholds this`
  if (state.state === 'learned') return counterparties ? `${characters} learns this from ${counterparties}` : `${characters} learns or realizes this`
  if (state.state === 'does_not_know') return `${characters} is explicitly shown not knowing this`
  if (state.state === 'attempts_to_recall') return `${characters} tries to remember this`
  if (state.state === 'believes') return `${characters} believes this`
  if (state.state === 'does_not_believe') return `${characters} is explicitly shown not believing this`
  if (state.state === 'suspects') return `${characters} suspects this`
  if (state.state === 'does_not_suspect') return `${characters} is explicitly shown not suspecting this`
  return `${characters} is shown knowing this`
}

function trailHeadline(total: number, chapters: number): string {
  if (total === 1) return 'One precise occurrence in the manuscript'
  if (chapters === 1) return `${total} connected occurrences in one chapter`
  return `${total} connected occurrences across ${chapters} chapters`
}

function HighlightedExcerpt({ text, terms }: { text: string; terms: string[] }) {
  const normalized = terms.filter(Boolean).sort((a, b) => b.length - a.length)
  if (normalized.length === 0) return <>{text}</>
  const escaped = normalized.map(term => term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
  const parts = text.split(new RegExp(`(${escaped.join('|')})`, 'gi'))
  const lower = new Set(normalized.map(term => term.toLowerCase()))
  return <>{parts.map((part, index) => lower.has(part.toLowerCase()) ? <mark key={index}>{part}</mark> : <Fragment key={index}>{part}</Fragment>)}</>
}
