// Signals — hub overview of the local manuscript analysis (design 3b).
// Freshness row, six stat tiles with since-last-run deltas, a one-line
// plain-language read, and jump rows into the other analysis panels.

import type { ReactNode } from 'react'
import { useBookStore } from '../../../store/bookStore'
import { useAnalysisStore } from '../../../store/analysisStore'
import { useAppStore } from '../../../store/appStore'
import {
  formatAnalyzedStamp,
  formatDelta,
  openToolsSection,
  usePreviousOverview,
  useReviewCount,
} from './shared'
import './signals.css'

function ChevronRight() {
  return (
    <svg
      width="12"
      height="12"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <polyline points="9 18 15 12 9 6" />
    </svg>
  )
}

function tempoDescriptor(score: number): 'measured' | 'balanced' | 'brisk' {
  if (score < 40) return 'measured'
  if (score < 70) return 'balanced'
  return 'brisk'
}

function sentenceClause(averageSentenceWords: number): string {
  if (averageSentenceWords < 10) return 'Short sentences'
  if (averageSentenceWords < 18) return 'Varied sentences'
  return 'Long sentences'
}

function dialogueClause(dialoguePercent: number): string {
  if (dialoguePercent < 15) return 'sparse dialogue'
  if (dialoguePercent < 35) return 'moderate dialogue'
  return 'dialogue-heavy'
}

export function buildSignalSummary(
  averageSentenceWords: number,
  dialoguePercent: number,
  tempo: 'measured' | 'balanced' | 'brisk',
): string {
  return `${sentenceClause(averageSentenceWords)}, ${dialogueClause(dialoguePercent)}, ${tempo} prose tempo.`
}

interface TileProps {
  label: string
  value: ReactNode
  note: string
}

function Tile({ label, value, note }: TileProps) {
  return (
    <div className="an-tile">
      <div className="an-tile-label">{label}</div>
      <div className="an-tile-value">{value}</div>
      <div className="an-tile-note">{note}</div>
    </div>
  )
}

export default function SignalsPanel() {
  const book = useBookStore(state => state.book)
  const analysisState = useAnalysisStore(state => state.state)
  const runAnalysis = useAnalysisStore(state => state.run)
  const analysisEnabled = useAppStore(state => state.settings.analysis_enabled)
  const previous = usePreviousOverview(book)
  const reviewCount = useReviewCount()

  if (!book) {
    return <div className="tool-empty-state">Open a project to see analysis.</div>
  }

  const analysis = book.analysis?.story
  const running = analysisState === 'running'

  if (!analysis) {
    return (
      <div className="an-panel sg-panel">
        <div className="an-empty">
          <p>No analysis yet.</p>
          <p>
            Draftline analyzes your manuscript locally after 15 seconds of
            writing inactivity — nothing leaves your machine.
          </p>
          <button
            className="an-run-btn"
            disabled={running || !analysisEnabled}
            onClick={() => void runAnalysis()}
          >
            {running ? 'Analyzing…' : 'Analyze now'}
          </button>
        </div>
      </div>
    )
  }

  const overview = analysis.overview
  const dotClass =
    analysisState === 'current' ? '' : analysisState === 'error' ? ' error' : ' stale'
  const freshnessText = running
    ? 'Analyzing…'
    : analysisState === 'stale'
      ? 'Analysis out of date'
      : formatAnalyzedStamp(analysis.last_analyzed)

  // The backend rounds overview metrics to one decimal; round tempo to an
  // integer for display and derive the descriptor from the same value so the
  // tile, its note, and the one-line read can never disagree at a boundary.
  const tempoScore = Math.round(overview.tempo_score)
  const tempo = tempoDescriptor(tempoScore)
  const oneLine = buildSignalSummary(
    overview.average_sentence_words,
    overview.dialogue_percent,
    tempo,
  )

  const deltaNote = (current: number, prev: number | undefined, digits: number) =>
    formatDelta(current, prev, digits) ?? 'first run'

  return (
    <div className="an-panel sg-panel">
      <div className="an-freshness">
        <div className="an-freshness-status">
          <span className={`an-freshness-dot${dotClass}`} />
          <span className="sg-freshness-text">{freshnessText}</span>
        </div>
        <button
          className="an-run-btn"
          disabled={running || !analysisEnabled}
          onClick={() => void runAnalysis()}
        >
          Run again
        </button>
      </div>

      <div className="an-block sg-tiles-block">
        <div className="an-tiles">
          <Tile
            label="Chapters"
            value={overview.chapter_count.toLocaleString()}
            note={deltaNote(overview.chapter_count, previous?.chapter_count, 0)}
          />
          <Tile
            label="Avg. chapter"
            value={
              <>
                {Math.round(overview.average_chapter_words).toLocaleString()} <small>words</small>
              </>
            }
            note={deltaNote(overview.average_chapter_words, previous?.average_chapter_words, 0)}
          />
          <Tile
            label="Avg. sentence"
            value={
              <>
                {overview.average_sentence_words.toFixed(1)} <small>words</small>
              </>
            }
            note={deltaNote(overview.average_sentence_words, previous?.average_sentence_words, 1)}
          />
          <Tile
            label="Dialogue"
            value={`${overview.dialogue_percent.toFixed(1)}%`}
            note={deltaNote(overview.dialogue_percent, previous?.dialogue_percent, 1)}
          />
          <Tile
            label="Manuscript"
            value={overview.word_count.toLocaleString()}
            note={deltaNote(overview.word_count, previous?.word_count, 0)}
          />
          <Tile
            label="Tempo"
            value={
              <>
                {tempoScore}
                <small>/100</small>
              </>
            }
            note={`${tempo} overall`}
          />
        </div>
      </div>

      <div className="sg-oneline">
        <div className="an-label">In one line</div>
        <div className="an-serif">{oneLine}</div>
      </div>

      <div className="sg-jumps">
        <button className="an-jump-row" onClick={() => openToolsSection('prose')}>
          <span>Prose — sentences &amp; word classes</span>
          <span className="an-jump-meta">
            <ChevronRight />
          </span>
        </button>
        <button className="an-jump-row" onClick={() => openToolsSection('pacing')}>
          <span>Pacing — tempo by chapter</span>
          <span className="an-jump-meta">
            <ChevronRight />
          </span>
        </button>
        <button className="an-jump-row" onClick={() => openToolsSection('chapters')}>
          <span>Chapters — keywords &amp; summaries</span>
          <span className="an-jump-meta">
            <ChevronRight />
          </span>
        </button>
        <button className="an-jump-row" onClick={() => openToolsSection('review')}>
          <span>Worth reviewing</span>
          <span className="an-jump-meta">
            {reviewCount > 0 && <span className="an-badge">{reviewCount}</span>}
            <ChevronRight />
          </span>
        </button>
      </div>
    </div>
  )
}
