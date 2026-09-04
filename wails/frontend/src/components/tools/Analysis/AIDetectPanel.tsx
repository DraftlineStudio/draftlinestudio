// AI Analysis panel (design 3g) — heuristic AI-detection for the whole book.
// Gauge for the current chapter, per-chapter heat strip, highest-signal list,
// flagged passages, and anti-pattern counts. All local heuristics; no backend
// analysis run required (does not use book.analysis.story).

import { useEffect, useMemo, useRef, useState } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../../store/bookStore'
import {
  analyzeText,
  analyzeAntiPatterns,
  getScoreColor,
  getAntiPatternColor,
  type AIDetectionResult,
  type AntiPatternResult,
} from '../../../services/aiDetection'
import { useBookAIScan, scanPassages, type ChapterAIScore } from '../../../services/aiScan'
import { htmlToText, getCurrentContent } from '../../../utils/textUtils'
import { allChapters } from '../../characters/shared'
import { goToChapter } from './shared'

const GAUGE_ARC_LENGTH = 188.5 // length of the 150x84 semicircle path

type Threshold = 'success' | 'warning' | 'error'

function thresholdOf(score: number): Threshold {
  if (score < 40) return 'success'
  if (score <= 70) return 'warning'
  return 'error'
}

const VERDICTS: Record<Threshold, string> = {
  success: 'Reads human',
  warning: 'Mixed signals',
  error: 'Likely AI',
}

const COUNT_WORDS = ['No', 'One', 'Two', 'Three', 'Four', 'Five', 'Six', 'Seven', 'Eight', 'Nine', 'Ten']

function mixedThresholdNote(count: number): string {
  if (count === 0) return 'No chapter crosses the mixed threshold.'
  const word = COUNT_WORDS[count] ?? String(count)
  return count === 1
    ? 'One chapter crosses the mixed threshold.'
    : `${word} chapters cross the mixed threshold.`
}

function truncateExcerpt(text: string, max = 160): string {
  if (text.length <= max) return text
  const cut = text.slice(0, max).replace(/\s+\S*$/, '')
  return `${cut}…`
}

interface RankedScore extends ChapterAIScore {
  position: number // 1-based position in book order among scanned chapters
}

interface FlaggedPassage extends RankedScore {
  passageScore: number
  excerpt: string
}

export default function AIDetectPanel() {
  const { book, currentSection, currentIndex } = useBookStore(useShallow(s => ({
    book: s.book,
    currentSection: s.currentSection,
    currentIndex: s.currentIndex,
  })))

  const { scores, done } = useBookAIScan(book)

  const currentContent = useMemo(
    () => getCurrentContent(book, currentSection, currentIndex),
    [book, currentSection, currentIndex],
  )
  const currentTextLength = useMemo(() => htmlToText(currentContent).length, [currentContent])

  const currentTitle = useMemo(() => {
    if (!book) return ''
    if (currentSection === 'copyright') return 'Copyright'
    return book[currentSection]?.[currentIndex]?.title || 'Untitled chapter'
  }, [book, currentSection, currentIndex])

  // Debounced current-chapter analysis: instant on chapter switch, 2s after edits.
  const [aiResult, setAiResult] = useState<AIDetectionResult | null>(null)
  const [antiPatterns, setAntiPatterns] = useState<AntiPatternResult | null>(null)
  const lastLocation = useRef('')

  // Book identity (stable across keystrokes, different across books) so that
  // opening another book at the same section/index counts as a switch.
  const bookIdent = book ? `${book.file_path || ''}|${book.metadata?.created || ''}` : ''

  useEffect(() => {
    const location = `${bookIdent}|${currentSection}:${currentIndex}`
    const switched = lastLocation.current !== location
    lastLocation.current = location
    if (currentTextLength < 100) {
      setAiResult(null)
      setAntiPatterns(null)
      return
    }
    const timer = setTimeout(() => {
      setAiResult(analyzeText(currentContent))
      setAntiPatterns(analyzeAntiPatterns(currentContent))
    }, switched ? 0 : 2000)
    return () => clearTimeout(timer)
  }, [currentContent, currentTextLength, bookIdent, currentSection, currentIndex])

  // Book-order scores (ascending combined index) with 1-based positions.
  const ordered = useMemo<RankedScore[]>(
    () => [...scores]
      .sort((a, b) => a.globalIndex - b.globalIndex)
      .map((s, i) => ({ ...s, position: i + 1 })),
    [scores],
  )

  const mixedCount = useMemo(() => ordered.filter(s => s.score >= 40).length, [ordered])

  const topFive = useMemo(
    () => [...ordered].sort((a, b) => b.score - a.score).slice(0, 5),
    [ordered],
  )

  // Best passage from each of the two highest-scoring chapters at >= 40.
  const chapters = useMemo(() => (book ? allChapters(book) : []), [book])
  const flagged = useMemo<FlaggedPassage[]>(() => {
    if (!done) return []
    const out: FlaggedPassage[] = []
    const candidates = [...ordered].sort((a, b) => b.score - a.score).filter(s => s.score >= 40).slice(0, 2)
    for (const c of candidates) {
      const best = scanPassages(chapters[c.globalIndex]?.content || '')[0]
      if (best) {
        out.push({ ...c, passageScore: best.score, excerpt: truncateExcerpt(best.excerpt) })
      }
    }
    return out
  }, [done, ordered, chapters])

  if (!book) {
    return <div className="tool-empty-state">Open a project to see analysis.</div>
  }

  const score = aiResult?.score ?? 0
  const verdictClass = thresholdOf(score)

  return (
    <div className="an-panel ai-panel">
      {/* 1 — Current-chapter gauge */}
      <div className="an-block ai-gauge-block">
        {currentTextLength < 100 ? (
          <div className="ai-gauge-empty">Not enough text in this chapter to analyze.</div>
        ) : aiResult === null ? (
          <div className="ai-gauge-empty">Analyzing…</div>
        ) : (
          <>
            <div className="ai-gauge">
              <svg width="150" height="84" viewBox="0 0 150 84">
                <path
                  d="M15 78 A60 60 0 0 1 135 78"
                  stroke="var(--bg-surface-alt)"
                  strokeWidth="11"
                  fill="none"
                  strokeLinecap="round"
                />
                <path
                  d="M15 78 A60 60 0 0 1 135 78"
                  stroke={getScoreColor(score)}
                  strokeWidth="11"
                  fill="none"
                  strokeLinecap="round"
                  strokeDasharray={`${(score / 100) * GAUGE_ARC_LENGTH} 200`}
                />
              </svg>
              <div className="ai-gauge-readout">
                <div className="ai-gauge-score">{score}%</div>
                <div className="an-label">AI signal</div>
              </div>
            </div>
            <div className="ai-verdict" style={{ color: `var(--status-${verdictClass})` }}>
              {VERDICTS[verdictClass]}
            </div>
            <div className="ai-gauge-sub">{currentTitle} · current chapter</div>
          </>
        )}
        <div className="ai-legend">
          <span><span className="ai-legend-dot" style={{ background: 'var(--status-success)' }} />{'< 40 human'}</span>
          <span><span className="ai-legend-dot" style={{ background: 'var(--status-warning)' }} />{'40–70 mixed'}</span>
          <span><span className="ai-legend-dot" style={{ background: 'var(--status-error)' }} />{'> 70 likely AI'}</span>
        </div>
      </div>

      {/* 2 — Whole-book heat strip */}
      <div className="an-block">
        <div className="an-label-row">
          <span className="an-label">By chapter</span>
          <span className="an-label-hint">start → end</span>
        </div>
        <div className="an-heat-strip">
          {ordered.map(s => (
            <div
              key={s.globalIndex}
              className="an-heat-cell"
              title={`${s.title} — ${s.score}%`}
              style={{
                background: `var(--status-${thresholdOf(s.score)})`,
                opacity: 0.35 + (s.score / 100) * 0.65,
              }}
            />
          ))}
        </div>
        <div className="an-footnote ai-strip-note">
          {!done
            ? 'Scanning chapters…'
            : ordered.length === 0
              ? 'No chapters with enough text to analyze.'
              : mixedThresholdNote(mixedCount)}
        </div>
      </div>

      {/* 3 — Highest signal */}
      {topFive.length > 0 && (
        <div className="an-block ai-top-block">
          <div className="an-label-row">
            <span className="an-label">Highest signal</span>
          </div>
          <div className="ai-top-rows">
            {topFive.map(s => (
              <button
                key={s.globalIndex}
                type="button"
                className="an-chapter-row"
                onClick={() => goToChapter(s.globalIndex)}
              >
                <span className="an-row-n">{s.position}</span>
                <span className="an-row-title">{s.title}</span>
                <div className="an-bar-track ai-top-bar">
                  <div
                    className={`an-bar-fill ${thresholdOf(s.score)}`}
                    style={{ width: `${s.score}%` }}
                  />
                </div>
                <span className="an-row-value">{s.score}%</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* 4 — Flagged passages */}
      <div className="an-block">
        <div className="an-label-row">
          <span className="an-label">Flagged passages</span>
        </div>
        {!done ? (
          <div className="an-footnote">Scanning chapters…</div>
        ) : flagged.length === 0 ? (
          <div className="an-footnote">No passages currently read as AI-assisted.</div>
        ) : (
          <div className="an-card-list">
            {flagged.map(p => (
              <div key={p.globalIndex} className="an-card">
                <div className="ai-card-head">
                  <span className="ai-card-chapter">Chapter {p.position} · {p.title}</span>
                  <span className={`an-badge ${p.passageScore > 70 ? 'error' : 'warning'}`}>
                    {p.passageScore}%
                  </span>
                </div>
                <div className="an-serif">“{p.excerpt}”</div>
                <button type="button" className="an-link ai-go" onClick={() => goToChapter(p.globalIndex)}>
                  Go to chapter ›
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 5 — AI anti-patterns (current chapter) */}
      <div className="an-block">
        <div className="an-label-row">
          <span className="an-label">AI anti-patterns</span>
          <span className="an-label-hint">current chapter</span>
        </div>
        {antiPatterns && antiPatterns.patterns.length > 0 ? (
          <div className="ai-ap-list">
            {antiPatterns.patterns.map(p => (
              <div
                key={p.pattern}
                className="ai-ap-row"
                style={p.count > p.limitPerChapter
                  ? { borderLeftColor: getAntiPatternColor(p.severity) }
                  : undefined}
              >
                <span className="ai-ap-name">{p.displayName}</span>
                <span className="ai-ap-count">{p.count}× (limit {p.limitPerChapter}/ch)</span>
              </div>
            ))}
          </div>
        ) : (
          <div className="an-footnote">No overused AI patterns detected.</div>
        )}
      </div>

      {/* 6 — Disclaimer */}
      <div className="an-block">
        <div className="an-footnote">
          Detection is probabilistic — a high score marks prose worth a second look, not a
          verdict. Heuristic estimate only; for authoritative verification use a dedicated
          detector.
        </div>
      </div>
    </div>
  )
}
