// Story Analysis viewer (the Beat Sheet / Foreshadowing / Knowledge Matrix
// planning tools were retired in 0.16.02449; their data model remains in the
// .draftline format for compatibility).

import { useBookStore } from '../../../store/bookStore'
import { useAnalysisStore } from '../../../store/analysisStore'
import { useAppStore } from '../../../store/appStore'

// ── Issues / Story Analysis ─────────────────────────────────────────────────

export function IssuesSection() {
  const { book } = useBookStore()
  const analysisState = useAnalysisStore(state => state.state)
  const runAnalysis = useAnalysisStore(state => state.run)
  const analysisEnabled = useAppStore(state => state.settings.analysis_enabled)

  if (!book) {
    return <div className="tool-empty-state">Open a project to see analysis.</div>
  }

  const analysis = book.analysis?.story
  if (!analysis) {
    return (
      <div className="issues-section">
        <div className="tool-empty-state">
          <p>No story analysis yet.</p>
          <p style={{ fontSize: '11px', opacity: 0.7, marginTop: '8px' }}>
            Draftline runs its private local analysis after 15 seconds of writing inactivity.
          </p>
          <button className="bible-add-btn" disabled={!analysisEnabled || analysisState === 'running'} onClick={() => void runAnalysis()}>
            {!analysisEnabled ? 'Enable in Plugins' : analysisState === 'running' ? 'Analyzing…' : 'Analyze now'}
          </button>
        </div>
      </div>
    )
  }

  const overview = analysis.overview
  return (
    <div className="issues-section story-analysis">
      <div className="story-analysis-heading">
        <div>
          <div className="tool-label">Manuscript signals</div>
          <span>{analysis.engine} · {analysisState === 'stale' ? 'out of date' : 'current'}</span>
        </div>
        <button className="bible-add-btn" disabled={!analysisEnabled || analysisState === 'running'} onClick={() => void runAnalysis()}>
          {!analysisEnabled ? 'Disabled' : analysisState === 'running' ? 'Analyzing…' : 'Run again'}
        </button>
      </div>

      <div className="story-analysis-overview">
        <Metric label="Chapters" value={overview.chapter_count.toLocaleString()} />
        <Metric label="Avg. chapter" value={`${Math.round(overview.average_chapter_words).toLocaleString()} words`} />
        <Metric label="Avg. sentence" value={`${overview.average_sentence_words} words`} />
        <Metric label="Dialogue" value={`${overview.dialogue_percent}%`} />
        <Metric label="Reading ease" value={overview.reading_ease.toFixed(1)} />
        <Metric label="Tempo" value={`${overview.tempo_score}/100`} />
      </div>

      {(analysis.observations?.length ?? 0) > 0 && (
        <section className="story-analysis-section">
          <div className="tool-label">Worth reviewing</div>
          <p className="settings-hint">Measured differences, not errors or prescriptions.</p>
          <div className="story-observation-list">
            {analysis.observations!.map((observation, index) => (
              <article className="story-observation" key={`${observation.chapter_index}-${observation.kind}-${index}`}>
                <span>{observation.kind}</span>
                <strong>{observation.title}</strong>
                <p>{observation.detail}</p>
              </article>
            ))}
          </div>
        </section>
      )}

      <section className="story-analysis-section">
        <div className="tool-label">By chapter</div>
        <div className="chapter-analysis-list">
          {analysis.chapters.map(chapter => (
            <article className="chapter-analysis-card" key={chapter.chapter_id || chapter.chapter_index}>
              <header>
                <strong>{chapter.title || `Chapter ${chapter.chapter_index + 1}`}</strong>
                <span className={`tempo-label ${chapter.tempo_label}`}>{chapter.tempo_label}</span>
              </header>
              <div className="chapter-analysis-stats">
                <span>{chapter.word_count.toLocaleString()} words</span>
                <span>{chapter.average_sentence_words} words/sentence</span>
                <span>{chapter.dialogue_percent}% dialogue</span>
              </div>
              {(chapter.keywords?.length ?? 0) > 0 && (
                <div className="chapter-keywords">
                  {chapter.keywords!.map(keyword => <span key={keyword.term}>{keyword.term}</span>)}
                </div>
              )}
              {chapter.extractive_summary && <p className="chapter-extractive-summary">{chapter.extractive_summary}</p>}
            </article>
          ))}
        </div>
      </section>
    </div>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return <div className="story-analysis-metric"><span>{label}</span><strong>{value}</strong></div>
}
