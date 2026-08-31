// Prose panel — sentence rhythm, length distribution, readability, and word
// classes (design ref: Story Analysis Sidebars, section 3c "Prose expanded").
// The rhythm strip is computed live from the current chapter (no analysis run
// needed); the remaining blocks come from a whole-manuscript client-side pass
// plus the stored story analysis overview.

import { useEffect, useMemo, useState } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../../store/bookStore'
import { useAnalysisStore } from '../../../store/analysisStore'
import { useAppStore } from '../../../store/appStore'
import { getCurrentContent, htmlToText } from '../../../utils/textUtils'
import { aggregateWordClasses, bookKey } from './shared'
import './prose.css'

const LONG_SENTENCE_WORDS = 25
const SHORT_SENTENCE_WORDS = 8
const RHYTHM_SENTENCES = 60
const RHYTHM_DEBOUNCE_MS = 2000

const BUCKET_LABELS = ['1–5', '6–10', '11–15', '16–20', '21+'] as const

// Word count of every sentence in a plain-text string, in reading order.
function sentenceWordCounts(text: string): number[] {
  return text
    .split(/[.!?…]+/)
    .map(sentence => sentence.trim())
    .filter(sentence => sentence.length > 0)
    .map(sentence => sentence.split(/\s+/).length)
}

// "46%" for big shares, "2.7%" for small ones (matches the design mock).
function formatPercent(value: number): string {
  return value >= 10 || value === 0 ? `${Math.round(value)}%` : `${value.toFixed(1)}%`
}

interface ManuscriptSentences {
  buckets: number[]        // sentence counts per BUCKET_LABELS bucket
  total: number            // total sentence count
  shortPercent: number     // % of sentences under SHORT_SENTENCE_WORDS words
  longPercent: number      // % of sentences over LONG_SENTENCE_WORDS words
}

export default function ProsePanel() {
  const { book, currentSection, currentIndex } = useBookStore(
    useShallow(state => ({
      book: state.book,
      currentSection: state.currentSection,
      currentIndex: state.currentIndex,
    })),
  )
  const analysisState = useAnalysisStore(state => state.state)
  const runAnalysis = useAnalysisStore(state => state.run)
  const analysisEnabled = useAppStore(state => state.settings.analysis_enabled)

  // ── Debounced book snapshot ───────────────────────────────────────────────
  // bookStore replaces the book object on every editor flush (~150ms while
  // typing), so anything memoized on `book` would re-run per keystroke. The
  // rhythm strip and the whole-manuscript pass both read this 2s-debounced
  // snapshot instead; a *different* project (bookKey change, incl. open/close)
  // swaps in immediately so the previous book's stats never linger.
  const [debouncedBook, setDebouncedBook] = useState(book)
  useEffect(() => {
    setDebouncedBook(prev => (prev && book && bookKey(prev) === bookKey(book) ? prev : book))
    const timer = window.setTimeout(() => setDebouncedBook(book), RHYTHM_DEBOUNCE_MS)
    return () => window.clearTimeout(timer)
  }, [book])

  // ── Rhythm: last N sentences of the current chapter ───────────────────────
  // Chapter switches update instantly (section/index are not debounced); only
  // live typing waits out the debounce.
  const rhythm = useMemo(
    () =>
      sentenceWordCounts(
        htmlToText(getCurrentContent(debouncedBook, currentSection, currentIndex)),
      ).slice(-RHYTHM_SENTENCES),
    [debouncedBook, currentSection, currentIndex],
  )
  const rhythmMax = rhythm.length > 0 ? Math.max(...rhythm) : 1

  // ── Whole-manuscript sentence pass (histogram + short/long shares) ────────
  const manuscript = useMemo<ManuscriptSentences | null>(() => {
    if (!debouncedBook) return null
    const buckets = [0, 0, 0, 0, 0]
    let total = 0
    let short = 0
    let long = 0
    for (const sectionChapters of [debouncedBook.front_matter, debouncedBook.body, debouncedBook.back_matter]) {
      for (const chapter of sectionChapters || []) {
        for (const words of sentenceWordCounts(htmlToText(chapter.content || ''))) {
          total += 1
          if (words < SHORT_SENTENCE_WORDS) short += 1
          if (words > LONG_SENTENCE_WORDS) long += 1
          if (words <= 5) buckets[0] += 1
          else if (words <= 10) buckets[1] += 1
          else if (words <= 15) buckets[2] += 1
          else if (words <= 20) buckets[3] += 1
          else buckets[4] += 1
        }
      }
    }
    return {
      buckets,
      total,
      shortPercent: total > 0 ? (short / total) * 100 : 0,
      longPercent: total > 0 ? (long / total) * 100 : 0,
    }
  }, [debouncedBook])

  if (!book) {
    return <div className="tool-empty-state">Open a project to see analysis.</div>
  }

  const analysis = book.analysis?.story

  const rhythmBlock = (
    <div className="an-block">
      <div className="an-label-row">
        <span className="an-label">Rhythm</span>
        <span className="an-label-hint">last {RHYTHM_SENTENCES} sentences</span>
      </div>
      {rhythm.length > 0 ? (
        <>
          <div className="prose-rhythm-strip">
            {rhythm.map((words, i) => (
              <div
                key={i}
                className={`prose-rhythm-bar${words > LONG_SENTENCE_WORDS ? ' warn' : ''}`}
                style={{ height: `${Math.max(8, (words / rhythmMax) * 100)}%` }}
              />
            ))}
          </div>
          <div className="an-footnote prose-rhythm-note">
            Each bar is a sentence; amber runs past {LONG_SENTENCE_WORDS} words.
          </div>
        </>
      ) : (
        <div className="prose-rhythm-empty">
          No prose in this chapter yet — the rhythm strip fills in as you write.
        </div>
      )}
    </div>
  )

  if (!analysis) {
    return (
      <div className="an-panel prose-panel">
        {rhythmBlock}
        <div className="an-block">
          <div className="an-empty">
            <p>No story analysis yet.</p>
            <p>Draftline runs its private local analysis after 15 seconds of writing inactivity.</p>
            <button
              className="an-run-btn prose-empty-run"
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

  const overview = analysis.overview
  const wordClasses = aggregateWordClasses(analysis.chapters)

  const bucketPercents = manuscript && manuscript.total > 0
    ? manuscript.buckets.map(count => (count / manuscript.total) * 100)
    : BUCKET_LABELS.map(() => 0)
  const maxBucketPercent = Math.max(...bucketPercents, 1)

  const avgParagraph = overview.paragraph_count > 0
    ? `${(overview.sentence_count / overview.paragraph_count).toFixed(1)} sentences`
    : '—'

  return (
    <div className="an-panel prose-panel">
      {rhythmBlock}

      <div className="an-block">
        <div className="an-label-row">
          <span className="an-label">Sentence lengths</span>
          <span className="an-label-hint">{(manuscript?.total ?? 0).toLocaleString()} sentences</span>
        </div>
        <div className="prose-hist">
          {BUCKET_LABELS.map((label, i) => (
            <div className="prose-hist-row" key={label}>
              <span className="prose-hist-label">{label}</span>
              <div className="an-bar-track">
                <div
                  className="an-bar-fill"
                  style={{ width: `${(bucketPercents[i] / maxBucketPercent) * 100}%` }}
                />
              </div>
              <span className="prose-hist-value">{Math.round(bucketPercents[i])}%</span>
            </div>
          ))}
        </div>
      </div>

      <div className="an-block">
        <div className="an-label prose-block-title">Sentences</div>
        <StatRow label={`Short (< ${SHORT_SENTENCE_WORDS} words)`} value={formatPercent(manuscript?.shortPercent ?? 0)} />
        <StatRow label={`Long (> ${LONG_SENTENCE_WORDS} words)`} value={formatPercent(manuscript?.longPercent ?? 0)} />
        <StatRow label="Avg. paragraph" value={avgParagraph} />
        <StatRow label="Paragraphs" value={overview.paragraph_count.toLocaleString()} />
      </div>

      <div className="an-block">
        <div className="an-label prose-block-title">Readability</div>
        <StatRow label="Reading ease" value={overview.reading_ease.toFixed(1)} />
        <StatRow label="Grade level" value={overview.mean_grade_level.toFixed(1)} />
      </div>

      <div className="an-block">
        <div className="an-label prose-classes-title">Word classes</div>
        <div className="prose-class-bar">
          <div className="prose-class-seg verbs" style={{ width: `${wordClasses.verb}%` }} />
          <div className="prose-class-seg adj" style={{ width: `${wordClasses.adjective}%` }} />
          <div className="prose-class-seg adv" style={{ width: `${wordClasses.adverb}%` }} />
          <div className="prose-class-seg rest" />
        </div>
        <div className="prose-legend">
          <span><span className="prose-swatch verbs" />Verbs {wordClasses.verb.toFixed(1)}%</span>
          <span><span className="prose-swatch adj" />Adj {wordClasses.adjective.toFixed(1)}%</span>
          <span><span className="prose-swatch adv" />Adv {wordClasses.adverb.toFixed(1)}%</span>
        </div>
      </div>
    </div>
  )
}

function StatRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="prose-stat-row">
      <span className="prose-stat-label">{label}</span>
      <span className="prose-stat-value">{value}</span>
    </div>
  )
}
