// Pacing panel — an explainable view of prose tempo by chapter. Tempo is a
// rhythm signal built from sentence length, sentence variation, and dialogue;
// it deliberately does not claim to measure plot urgency or story quality.

import { useMemo, useState } from 'react'
import { useBookStore } from '../../../store/bookStore'
import { useAnalysisStore } from '../../../store/analysisStore'
import { useAppStore } from '../../../store/appStore'
import { goToChapter } from './shared'
import type { ChapterAnalysis } from '../../../types/draftline'

const MIN_ANALYZABLE_WORDS = 20
const NOTICEABLE_SHIFT = 10

type MetricKey = 'tempo' | 'dialogue' | 'length'
type TempoBand = 'measured' | 'balanced' | 'brisk'

interface MetricDef {
  key: MetricKey
  tab: string
  explanation: string
}

interface TempoShift {
  from: ChapterAnalysis
  to: ChapterAnalysis
  delta: number
}

const METRICS: MetricDef[] = [
  {
    key: 'tempo',
    tab: 'Tempo',
    explanation: 'Higher scores usually mean shorter sentences, more dialogue, or both.',
  },
  {
    key: 'dialogue',
    tab: 'Dialogue',
    explanation: 'The share of chapter words that appear inside quotation marks.',
  },
  {
    key: 'length',
    tab: 'Length',
    explanation: 'Chapter size compared with the typical analyzed chapter in this book.',
  },
]

const isAnalyzable = (chapter: ChapterAnalysis) => chapter.word_count >= MIN_ANALYZABLE_WORDS
const chapterName = (chapter: ChapterAnalysis, n: number) => chapter.title || `Chapter ${n}`

export function tempoBand(score: number): TempoBand {
  if (score <= 42) return 'measured'
  if (score >= 68) return 'brisk'
  return 'balanced'
}

export function median(values: number[]): number {
  if (values.length === 0) return 0
  const sorted = [...values].sort((a, b) => a - b)
  const middle = Math.floor(sorted.length / 2)
  return sorted.length % 2 === 0
    ? (sorted[middle - 1] + sorted[middle]) / 2
    : sorted[middle]
}

export function findTempoShifts(chapters: ChapterAnalysis[]): TempoShift[] {
  const shifts: TempoShift[] = []
  for (let index = 1; index < chapters.length; index += 1) {
    const from = chapters[index - 1]
    const to = chapters[index]
    if (!isAnalyzable(from) || !isAnalyzable(to)) continue
    const delta = to.tempo_score - from.tempo_score
    if (Math.abs(delta) >= NOTICEABLE_SHIFT) {
      shifts.push({ from, to, delta })
    }
  }
  return shifts.sort((a, b) => Math.abs(b.delta) - Math.abs(a.delta))
}

function plural(value: number, singular: string): string {
  return `${value.toLocaleString()} ${singular}${value === 1 ? '' : 's'}`
}

function dialogueLabel(value: number): string {
  if (value < 15) return 'sparse'
  if (value < 35) return 'moderate'
  return 'dialogue-heavy'
}

function lengthLabel(ratio: number): string {
  if (ratio < 0.65) return 'shorter than typical'
  if (ratio > 1.5) return 'longer than typical'
  return 'near typical length'
}

function shiftCopy(shift: TempoShift): string {
  const direction = shift.delta > 0 ? 'rises' : 'drops'
  return `${shift.to.title || 'The next chapter'} ${direction} ${Math.abs(Math.round(shift.delta))} points after ${shift.from.title || 'the previous chapter'}.`
}

export default function PacingPanel() {
  const book = useBookStore(state => state.book)
  const analysisState = useAnalysisStore(state => state.state)
  const runAnalysis = useAnalysisStore(state => state.run)
  const analysisEnabled = useAppStore(state => state.settings.analysis_enabled)
  const [metricKey, setMetricKey] = useState<MetricKey>('tempo')

  const analysis = book?.analysis?.story
  const chapters = analysis?.chapters
  const metric = METRICS.find(item => item.key === metricKey) ?? METRICS[0]

  const model = useMemo(() => {
    const list = chapters ?? []
    const usable = list.filter(isAnalyzable)
    const typicalWords = median(usable.map(chapter => chapter.word_count))
    const shifts = findTempoShifts(list)
    const bands = usable.reduce<Record<TempoBand, number>>(
      (counts, chapter) => {
        counts[tempoBand(chapter.tempo_score)] += 1
        return counts
      },
      { measured: 0, balanced: 0, brisk: 0 },
    )
    return { typicalWords, shifts, bands }
  }, [chapters])

  if (!book) {
    return <div className="tool-empty-state">Open a project to see analysis.</div>
  }

  if (!analysis || !chapters || chapters.length === 0) {
    return (
      <div className="an-panel pacing-panel">
        <div className="an-empty">
          <p>No prose-tempo analysis yet.</p>
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

  const overallScore = Math.round(analysis.overview.tempo_score)
  const overallBand = tempoBand(overallScore)
  const largestShift = model.shifts[0]

  return (
    <div className="an-panel pacing-panel">
      <div className="an-block pacing-overview">
        <div className="an-label">Prose tempo</div>
        <div className="pacing-score-line">
          <span className={`pacing-score ${overallBand}`}>{overallScore}</span>
          <span>
            <strong>{overallBand}</strong>
            <small>out of 100</small>
          </span>
        </div>
        <p className="pacing-definition">
          How quickly the prose reads based on sentence shape and dialogue. This is a rhythm
          measurement, not a judgment of plot urgency or quality.
        </p>
      </div>

      <div className="an-block pacing-flow">
        <div className="an-label-row">
          <span className="an-label">Flow through the book</span>
          <span className="an-label-hint">start → end</span>
        </div>
        <div className="pacing-flow-strip" aria-label="Chapter prose tempo from start to end">
          {chapters.map((chapter, index) => {
            const band = tempoBand(chapter.tempo_score)
            const inert = !isAnalyzable(chapter) || chapter.tempo_score === 0
            return (
              <button
                type="button"
                key={chapter.chapter_id || chapter.chapter_index}
                className={`pacing-flow-cell ${inert ? 'inert' : band}`}
                title={`${index + 1} · ${chapterName(chapter, index + 1)} — ${Math.round(chapter.tempo_score)}/100, ${band}`}
                aria-label={`Open ${chapterName(chapter, index + 1)}`}
                onClick={() => goToChapter(chapter.chapter_index)}
              />
            )
          })}
        </div>
        <div className="pacing-legend">
          <span><i className="measured" />Measured {model.bands.measured}</span>
          <span><i className="balanced" />Balanced {model.bands.balanced}</span>
          <span><i className="brisk" />Brisk {model.bands.brisk}</span>
        </div>
        <div className="pacing-flow-read">
          {largestShift
            ? `${model.shifts.length} noticeable ${model.shifts.length === 1 ? 'change' : 'changes'}. ${shiftCopy(largestShift)}`
            : 'Tempo remains fairly consistent between neighboring chapters.'}
        </div>
      </div>

      {model.shifts.length > 0 && (
        <div className="an-block pacing-shifts">
          <div className="an-label">Largest transitions</div>
          {model.shifts.slice(0, 3).map(shift => (
            <button
              type="button"
              className="pacing-shift-row"
              key={`${shift.from.chapter_index}-${shift.to.chapter_index}`}
              onClick={() => goToChapter(shift.to.chapter_index)}
            >
              <span className={shift.delta > 0 ? 'up' : 'down'}>
                {shift.delta > 0 ? '↑' : '↓'} {Math.abs(Math.round(shift.delta))}
              </span>
              <span>{shiftCopy(shift)}</span>
            </button>
          ))}
        </div>
      )}

      <div className="an-block pacing-metric-picker">
        <div className="an-tabs">
          {METRICS.map(item => (
            <button
              type="button"
              key={item.key}
              className={`an-tab${item.key === metricKey ? ' active' : ''}`}
              onClick={() => setMetricKey(item.key)}
            >
              {item.tab}
            </button>
          ))}
        </div>
        <div className="an-footnote">{metric.explanation}</div>
      </div>

      <div className="pacing-chapters">
        {chapters.map((chapter, index) => {
          const analyzable = isAnalyzable(chapter)
          const band = tempoBand(chapter.tempo_score)
          const lengthRatio = model.typicalWords > 0 ? chapter.word_count / model.typicalWords : 0
          const metricValue = metricKey === 'tempo'
            ? chapter.tempo_score
            : metricKey === 'dialogue'
              ? chapter.dialogue_percent
              : Math.min(100, lengthRatio * 50)
          const value = metricKey === 'tempo'
            ? `${Math.round(chapter.tempo_score)}/100 · ${band}`
            : metricKey === 'dialogue'
              ? `${chapter.dialogue_percent.toFixed(1)}% · ${dialogueLabel(chapter.dialogue_percent)}`
              : `${chapter.word_count.toLocaleString()} words`
          const detail = metricKey === 'tempo'
            ? `${chapter.average_sentence_words.toFixed(1)} words/sentence · ${chapter.dialogue_percent.toFixed(0)}% dialogue`
            : metricKey === 'dialogue'
              ? `${plural(chapter.sentence_count, 'sentence')} · ${plural(chapter.scene_break_count, 'scene break')}`
              : `${lengthRatio.toFixed(1)}× typical · ${lengthLabel(lengthRatio)}`

          return (
            <button
              type="button"
              key={chapter.chapter_id || chapter.chapter_index}
              className={`pacing-chapter-card${analyzable ? '' : ' inert'}`}
              onClick={() => goToChapter(chapter.chapter_index)}
            >
              <span className="pacing-chapter-number">{index + 1}</span>
              <span className="pacing-chapter-body">
                <span className="pacing-chapter-head">
                  <span className="pacing-chapter-title">{chapterName(chapter, index + 1)}</span>
                  <span className="pacing-chapter-value">{analyzable ? value : 'Not enough prose'}</span>
                </span>
                {analyzable && (
                  <>
                    <span className="pacing-chapter-track">
                      <span
                        className={`pacing-chapter-fill ${metricKey === 'tempo' ? band : metricKey}`}
                        style={{ width: `${Math.max(2, Math.min(100, metricValue))}%` }}
                      />
                    </span>
                    <span className="pacing-chapter-detail">{detail}</span>
                  </>
                )}
              </span>
            </button>
          )
        })}
      </div>

      <div className="an-block an-footnote pacing-footer">
        Typical chapter: {Math.round(model.typicalWords).toLocaleString()} words · select any chapter to open it
      </div>
    </div>
  )
}
