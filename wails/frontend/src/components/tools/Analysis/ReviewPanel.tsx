// Worth Reviewing panel (analysis sidebar suite, design 3f) — dismissable
// observation cards with a kind filter. Dismissals persist per book via
// shared.ts (localStorage) so the list can be worked down to zero.

import { useState } from 'react'
import { useBookStore } from '../../../store/bookStore'
import { useAnalysisStore } from '../../../store/analysisStore'
import { useAppStore } from '../../../store/appStore'
import {
  activeObservations,
  dismissObservation,
  goToChapter,
  observationKey,
  restoreAllObservations,
  useDismissedSet,
} from './shared'
import './review.css'

function capitalize(word: string): string {
  return word.charAt(0).toUpperCase() + word.slice(1)
}

export default function ReviewPanel() {
  const book = useBookStore(state => state.book)
  const analysisState = useAnalysisStore(state => state.state)
  const runAnalysis = useAnalysisStore(state => state.run)
  const analysisEnabled = useAppStore(state => state.settings.analysis_enabled)
  const dismissed = useDismissedSet(book)
  const [filter, setFilter] = useState<string | null>(null)

  if (!book) {
    return <div className="tool-empty-state">Open a project to see analysis.</div>
  }

  const analysis = book.analysis?.story
  if (!analysis) {
    return (
      <div className="an-panel rv-panel">
        <div className="an-empty">
          No analysis yet. Draftline analyzes your manuscript locally after 15 seconds
          of writing inactivity.
          <div>
            <button
              className="an-run-btn rv-empty-action"
              disabled={!analysisEnabled || analysisState === 'running'}
              onClick={() => void runAnalysis()}
            >
              {!analysisEnabled ? 'Enable in Plugins' : analysisState === 'running' ? 'Analyzing…' : 'Analyze now'}
            </button>
          </div>
        </div>
      </div>
    )
  }

  const observations = analysis.observations ?? []
  const active = activeObservations(analysis, dismissed)

  const kindCounts = new Map<string, number>()
  for (const observation of active) {
    kindCounts.set(observation.kind, (kindCounts.get(observation.kind) ?? 0) + 1)
  }
  // Fall back to "All" when the selected kind's last card gets dismissed.
  const effectiveFilter = filter !== null && kindCounts.has(filter) ? filter : null
  const visible = effectiveFilter === null ? active : active.filter(o => o.kind === effectiveFilter)

  return (
    <div className="an-panel rv-panel">
      {active.length > 0 && (
        <div className="rv-filter">
          <div className="an-tabs rv-tabs">
            <button
              className={`an-tab${effectiveFilter === null ? ' active' : ''}`}
              onClick={() => setFilter(null)}
            >
              All {active.length}
            </button>
            {[...kindCounts.keys()].map(kind => (
              <button
                key={kind}
                className={`an-tab${effectiveFilter === kind ? ' active' : ''}`}
                onClick={() => setFilter(kind)}
              >
                {capitalize(kind)} {kindCounts.get(kind)}
              </button>
            ))}
          </div>
          <div className="an-footnote">Measured differences, not errors or prescriptions.</div>
        </div>
      )}

      {observations.length === 0 ? (
        <div className="an-empty">Nothing worth flagging — the manuscript reads evenly.</div>
      ) : active.length === 0 ? (
        <div className="an-empty">
          All observations dismissed.
          <div>
            <button
              className="an-run-btn rv-empty-action"
              onClick={() => restoreAllObservations(book)}
            >
              Restore dismissed
            </button>
          </div>
        </div>
      ) : (
        <div className="an-card-list rv-cards">
          {visible.map((observation, index) => {
            const key = observationKey(observation)
            const chapterIndex = observation.chapter_index
            const kindClass =
              observation.kind === 'structure' ? ' structure'
              : observation.kind === 'pacing' ? ' pacing'
              : ''
            return (
              <div className="an-card rv-card" key={`${key}#${index}`}>
                <div className={`rv-kind${kindClass}`}>{capitalize(observation.kind)}</div>
                <div className="rv-title">{observation.title}</div>
                <div className="rv-detail">{observation.detail}</div>
                <div className="rv-actions">
                  {chapterIndex !== undefined && (
                    <button className="an-link" onClick={() => goToChapter(chapterIndex)}>
                      Go to chapter ›
                    </button>
                  )}
                  <button className="rv-dismiss" onClick={() => dismissObservation(book, key)}>
                    Dismiss
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
