// Chapters panel (analysis sidebar suite) — per-chapter keywords and
// extractive summaries, extracted at the last local analysis run.

import { useMemo, useState } from 'react'
import { useBookStore } from '../../../store/bookStore'
import { useAnalysisStore } from '../../../store/analysisStore'
import { useAppStore } from '../../../store/appStore'
import { goToChapter } from './shared'
import type { ChapterAnalysis } from '../../../types/draftline'

const VISIBLE_LIMIT = 8
const KEYWORD_LIMIT = 6

interface ChapterCard {
  n: number
  chapter: ChapterAnalysis
  keywords: string[]
}

function plural(count: number, noun: string): string {
  return `${count} ${noun}${count === 1 ? '' : 's'}`
}

export default function ChaptersPanel() {
  const book = useBookStore(state => state.book)
  const analysisState = useAnalysisStore(state => state.state)
  const runAnalysis = useAnalysisStore(state => state.run)
  const analysisEnabled = useAppStore(state => state.settings.analysis_enabled)
  const [expanded, setExpanded] = useState(false)

  const analysis = book?.analysis?.story

  // Number cards by position among analyzable chapters, then drop the ones
  // with nothing to show (part dividers etc.) so numbering stays stable.
  const cards = useMemo<ChapterCard[]>(() => {
    if (!analysis) return []
    return analysis.chapters
      .map((chapter, index) => ({
        n: index + 1,
        chapter,
        keywords: (chapter.keywords ?? []).slice(0, KEYWORD_LIMIT).map(k => k.term),
      }))
      .filter(card => card.keywords.length > 0 || Boolean(card.chapter.extractive_summary))
  }, [analysis])

  if (!book) {
    return <div className="tool-empty-state">Open a project to see analysis.</div>
  }

  if (!analysis) {
    return (
      <div className="an-panel chp-panel">
        <div className="an-empty">
          <p>No analysis yet.</p>
          <p>Draftline runs its private local analysis after 15 seconds of writing inactivity.</p>
          <button
            type="button"
            className="an-run-btn"
            disabled={!analysisEnabled || analysisState === 'running'}
            onClick={() => void runAnalysis()}
          >
            {!analysisEnabled ? 'Enable in Plugins' : analysisState === 'running' ? 'Analyzing…' : 'Analyze now'}
          </button>
        </div>
      </div>
    )
  }

  const visible = expanded ? cards : cards.slice(0, VISIBLE_LIMIT)

  return (
    <div className="an-panel chp-panel">
      <div className="chp-intro">
        Keywords and a one-line summary per chapter, extracted at last analysis.
      </div>
      {cards.length === 0 ? (
        <div className="an-empty">The last analysis produced no keywords or summaries for any chapter.</div>
      ) : (
        <div className="an-block">
          <div className="an-card-list">
            {visible.map(({ n, chapter, keywords }) => (
              <button
                key={chapter.chapter_id || chapter.chapter_index}
                type="button"
                className="an-card clickable chp-card"
                onClick={() => goToChapter(chapter.chapter_index)}
              >
                <div className="chp-card-head">
                  <span className="chp-card-n">{n}</span>
                  <span className="chp-card-title">{chapter.title}</span>
                </div>
                <div className="chp-card-meta">
                  {chapter.word_count.toLocaleString()} words
                  {' · '}
                  {plural(chapter.scene_break_count + 1, 'scene')}
                  {' · '}
                  {plural(chapter.scene_break_count, 'break')}
                </div>
                {chapter.extractive_summary && (
                  <div className="an-serif chp-card-summary">{chapter.extractive_summary}</div>
                )}
                {keywords.length > 0 && (
                  <div className="an-chip-row">
                    {keywords.map(term => (
                      <span key={term} className="an-chip">{term}</span>
                    ))}
                  </div>
                )}
              </button>
            ))}
            {cards.length > VISIBLE_LIMIT && (
              <button type="button" className="an-text-action" onClick={() => setExpanded(current => !current)}>
                {expanded ? 'Show fewer' : `Show all ${cards.length} chapters`}
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
