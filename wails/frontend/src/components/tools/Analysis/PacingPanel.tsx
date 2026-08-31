// Pacing panel (analysis sidebar suite, design ref 3d) — tempo by chapter.
// A whole-book heat strip up top, then a per-chapter scan list switchable
// between tempo / dialogue / reading ease / length, with scene-break counts.

import { useMemo, useState } from 'react'
import { useBookStore } from '../../../store/bookStore'
import { useAnalysisStore } from '../../../store/analysisStore'
import { useAppStore } from '../../../store/appStore'
import { goToChapter } from './shared'
import type { ChapterAnalysis } from '../../../types/draftline'
import './pacing.css'

// Chapters this short carry no meaningful pacing signal (part dividers,
// epigraphs); they render faint and are excluded from means and outliers.
const MIN_ANALYZABLE_WORDS = 20

type MetricKey = 'tempo' | 'dialogue' | 'ease' | 'length'

interface MetricDef {
  key: MetricKey
  tab: string
  column: string
  value: (ch: ChapterAnalysis) => number
}

const METRICS: MetricDef[] = [
  { key: 'tempo', tab: 'Tempo', column: 'Tempo', value: ch => ch.tempo_score },
  { key: 'dialogue', tab: 'Dialogue', column: 'Dialogue', value: ch => ch.dialogue_percent },
  { key: 'ease', tab: 'Ease', column: 'Ease', value: ch => ch.reading_ease },
  { key: 'length', tab: 'Length', column: 'Length', value: ch => ch.word_count },
]

const isAnalyzable = (ch: ChapterAnalysis) => ch.word_count >= MIN_ANALYZABLE_WORDS

const chapterName = (ch: ChapterAnalysis, n: number) => ch.title || `Chapter ${n}`

// Lowest-mean contiguous window of 3+ analyzable chapters. Returns the sag
// sentence when that window sits at least 12 tempo points below the book mean.
function heatInsight(chapters: ChapterAnalysis[]): string {
  const rows = chapters
    .map((ch, i) => ({ n: i + 1, tempo: ch.tempo_score }))
    .filter((row, i) => isAnalyzable(chapters[i]) && row.tempo > 0)
  const base = 'Brighter is faster.'
  if (rows.length < 3) return base
  const bookMean = rows.reduce((sum, row) => sum + row.tempo, 0) / rows.length
  // Prefix sums make every window mean O(1).
  const prefix = [0]
  for (const row of rows) prefix.push(prefix[prefix.length - 1] + row.tempo)
  let best: { start: number; end: number; mean: number } | null = null
  for (let len = 3; len <= rows.length; len++) {
    for (let i = 0; i + len <= rows.length; i++) {
      const mean = (prefix[i + len] - prefix[i]) / len
      if (!best || mean < best.mean) {
        best = { start: rows[i].n, end: rows[i + len - 1].n, mean }
      }
    }
  }
  if (best && bookMean - best.mean >= 12) {
    return `${base} The pace sags around chapters ${best.start}–${best.end}.`
  }
  return base
}

export default function PacingPanel() {
  const book = useBookStore(state => state.book)
  const analysisState = useAnalysisStore(state => state.state)
  const runAnalysis = useAnalysisStore(state => state.run)
  const analysisEnabled = useAppStore(state => state.settings.analysis_enabled)
  const [metricKey, setMetricKey] = useState<MetricKey>('tempo')

  const analysis = book?.analysis?.story
  const chapters = analysis?.chapters

  const insight = useMemo(() => heatInsight(chapters ?? []), [chapters])

  const metric = METRICS.find(m => m.key === metricKey) ?? METRICS[0]

  // Normalization max plus outlier stats (mean/σ over analyzable chapters
  // only) for the active metric.
  const scan = useMemo(() => {
    const list = chapters ?? []
    const values = list.map(metric.value)
    const max = values.reduce((a, b) => Math.max(a, b), 0)
    const usable = list.filter(isAnalyzable).map(metric.value)
    const mean = usable.length ? usable.reduce((a, b) => a + b, 0) / usable.length : 0
    const variance = usable.length
      ? usable.reduce((sum, v) => sum + (v - mean) * (v - mean), 0) / usable.length
      : 0
    const sd = Math.sqrt(variance)
    return { values, max, mean, sd }
  }, [chapters, metric])

  if (!book) {
    return <div className="tool-empty-state">Open a project to see analysis.</div>
  }

  if (!analysis || !chapters || chapters.length === 0) {
    return (
      <div className="an-panel pacing-panel">
        <div className="an-empty">
          <p>No pacing analysis yet.</p>
          <p>Draftline analyzes your manuscript locally after 15 seconds of writing inactivity.</p>
          <button
            className="an-run-btn"
            disabled={!analysisEnabled || analysisState === 'running'}
            onClick={() => void runAnalysis()}
          >
            {analysisState === 'running' ? 'Analyzing…' : 'Analyze now'}
          </button>
        </div>
      </div>
    )
  }

  const totalBreaks = chapters.reduce((sum, ch) => sum + ch.scene_break_count, 0)

  return (
    <div className="an-panel pacing-panel">
      <div className="an-block">
        <div className="an-label-row">
          <span className="an-label">The whole book</span>
          <span className="an-label-hint">start → end</span>
        </div>
        <div className="an-heat-strip">
          {chapters.map((ch, i) => {
            const faint = ch.tempo_score === 0 || !isAnalyzable(ch)
            return (
              <div
                key={ch.chapter_id || `${ch.chapter_index}`}
                className="an-heat-cell"
                title={`${i + 1} · ${chapterName(ch, i + 1)} — ${ch.tempo_score}`}
                style={
                  faint
                    ? { background: 'var(--text-faint)' }
                    : {
                        background: 'var(--app-accent)',
                        opacity: 0.35 + (ch.tempo_score / 100) * 0.65,
                      }
                }
              />
            )
          })}
        </div>
        <div className="an-footnote pacing-heat-note">{insight}</div>
      </div>

      <div className="an-block an-tabs pacing-tabs-block">
        {METRICS.map(m => (
          <button
            key={m.key}
            className={`an-tab${m.key === metric.key ? ' active' : ''}`}
            onClick={() => setMetricKey(m.key)}
          >
            {m.tab}
          </button>
        ))}
      </div>

      <div className="pacing-col-head">
        <span className="pacing-col-n" />
        <span className="pacing-col-title">Chapter</span>
        <span className="pacing-col-metric">{metric.column}</span>
        <span className="pacing-col-brk">Brk</span>
      </div>

      <div className="pacing-rows">
        {chapters.map((ch, i) => {
          const value = scan.values[i]
          const width = scan.max > 0 ? Math.max((value / scan.max) * 100, 2) : 2
          const analyzable = isAnalyzable(ch)
          const outlier =
            analyzable && scan.sd > 0 && Math.abs(value - scan.mean) > 1.5 * scan.sd
          const fillClass = !analyzable
            ? 'an-bar-fill faint'
            : outlier
              ? 'an-bar-fill warning'
              : 'an-bar-fill'
          return (
            <button
              key={ch.chapter_id || `${ch.chapter_index}`}
              className="an-chapter-row"
              onClick={() => goToChapter(ch.chapter_index)}
            >
              <span className="an-row-n">{i + 1}</span>
              <span className="an-row-title">{chapterName(ch, i + 1)}</span>
              <div className="an-bar-track pacing-bar">
                <div className={fillClass} style={{ width: `${width}%` }} />
              </div>
              <span className="an-row-value">
                {analyzable ? ch.scene_break_count : '—'}
              </span>
            </button>
          )
        })}
      </div>

      <div className="an-block an-footnote pacing-footer">
        Brk = scene breaks · {totalBreaks.toLocaleString()} across the manuscript · amber
        bars are outliers
      </div>
    </div>
  )
}
