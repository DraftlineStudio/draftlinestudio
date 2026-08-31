// Writing Dashboard - word counts, goals, AI detection

import { useState, useEffect, useRef, useMemo } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../../store/bookStore'
import { analyzeText, getScoreColor, getScoreLabel, analyzeAntiPatterns, getAntiPatternColor, type AIDetectionResult, type AntiPatternResult } from '../../../services/aiDetection'
import { countWords, countBookWords } from '../../../utils/textUtils'

export default function DashboardTab() {
  const { book, updateWritingGoals, currentSection, currentIndex } = useBookStore(useShallow(s => ({
    book: s.book,
    updateWritingGoals: s.updateWritingGoals,
    currentSection: s.currentSection,
    currentIndex: s.currentIndex,
  })))
  const [sessionStart] = useState(() => Date.now())
  const [sessionStartWords] = useState(() => {
    if (!book) return 0
    return countBookWords(book)
  })
  const [editingTarget, setEditingTarget] = useState(false)
  const [editingDaily, setEditingDaily] = useState(false)
  const [targetInput, setTargetInput] = useState('')
  const [dailyInput, setDailyInput] = useState('')

  if (!book) {
    return <div className="tool-empty-state">Open or create a project to see your writing dashboard.</div>
  }

  // Get current chapter content for AI analysis
  const currentContent = useMemo(() => {
    if (currentSection === 'copyright') return book.copyright || ''
    const arr = currentSection === 'front_matter' ? book.front_matter
      : currentSection === 'body' ? book.body
      : book.back_matter
    return arr[currentIndex]?.content || ''
  }, [book, currentSection, currentIndex])

  // Debounced AI Detection - only run 2 seconds after content stops changing
  const [aiResult, setAiResult] = useState<AIDetectionResult | null>(null)
  const [antiPatternResult, setAntiPatternResult] = useState<AntiPatternResult | null>(null)
  const analysisTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (analysisTimer.current) clearTimeout(analysisTimer.current)
    if (!currentContent || currentContent.length < 100) {
      setAiResult(null)
      setAntiPatternResult(null)
      return
    }
    analysisTimer.current = setTimeout(() => {
      setAiResult(analyzeText(currentContent))
      setAntiPatternResult(analyzeAntiPatterns(currentContent))
    }, 2000)
    return () => { if (analysisTimer.current) clearTimeout(analysisTimer.current) }
  }, [currentContent])

  // Calculate metrics. Whole-book count re-parses every chapter's HTML, so
  // memoize on the book reference — it must not re-run on every unrelated store
  // update that re-renders this tab.
  const totalWords = useMemo(() => countBookWords(book), [book])
  const targetWords = book.writing_goals?.target_word_count || 0
  const dailyGoal = book.writing_goals?.daily_word_goal || 0
  const todayWords = book.writing_goals?.words_today || 0
  const lastWritingDate = book.writing_goals?.last_writing_date || ''

  // Check if it's a new day
  const today = new Date().toISOString().split('T')[0]
  const isNewDay = lastWritingDate !== today

  // Session stats
  const sessionWords = Math.max(0, totalWords - sessionStartWords)
  const sessionMinutes = Math.floor((Date.now() - sessionStart) / 60000)

  // Chapter stats
  const chapters = book.body || []
  const chapterWordCounts = useMemo(() => chapters.map(ch => countWords(ch.content || '')), [chapters])
  const avgWordsPerChapter = chapters.length > 0
    ? Math.round(chapterWordCounts.reduce((a, b) => a + b, 0) / chapters.length)
    : 0
  const shortestChapter = chapterWordCounts.length > 0 ? Math.min(...chapterWordCounts) : 0
  const longestChapter = chapterWordCounts.length > 0 ? Math.max(...chapterWordCounts) : 0

  // Progress percentages
  const targetProgress = targetWords > 0 ? Math.min(100, (totalWords / targetWords) * 100) : 0
  const dailyProgress = dailyGoal > 0 ? Math.min(100, ((isNewDay ? 0 : todayWords) / dailyGoal) * 100) : 0

  function handleSetTarget() {
    const num = parseInt(targetInput.replace(/,/g, ''), 10)
    if (!isNaN(num) && num > 0) {
      updateWritingGoals({ target_word_count: num })
    }
    setEditingTarget(false)
    setTargetInput('')
  }

  function handleSetDaily() {
    const num = parseInt(dailyInput.replace(/,/g, ''), 10)
    if (!isNaN(num) && num > 0) {
      updateWritingGoals({ daily_word_goal: num })
    }
    setEditingDaily(false)
    setDailyInput('')
  }

  return (
    <div className="dashboard-content">
      {/* Manuscript Progress */}
      <div className="dashboard-section">
        <div className="dashboard-section-header">
          <span className="dashboard-section-title">Manuscript Progress</span>
        </div>
        <div className="dashboard-big-number">
          <span className="big-number">{totalWords.toLocaleString()}</span>
          <span className="big-number-label">words</span>
        </div>
        {targetWords > 0 ? (
          <div className="progress-bar-container">
            <div className="progress-bar">
              <div className="progress-bar-fill" style={{ width: `${targetProgress}%` }} />
            </div>
            <div className="progress-bar-labels">
              <span>{Math.round(targetProgress)}%</span>
              <span className="progress-target" onClick={() => { setTargetInput(targetWords.toString()); setEditingTarget(true) }}>
                {targetWords.toLocaleString()} target
              </span>
            </div>
          </div>
        ) : editingTarget ? (
          <div className="goal-input-row">
            <input
              type="text"
              className="goal-input"
              value={targetInput}
              onChange={e => setTargetInput(e.target.value)}
              placeholder="e.g. 80000"
              autoFocus
              onKeyDown={e => { if (e.key === 'Enter') handleSetTarget(); if (e.key === 'Escape') setEditingTarget(false) }}
            />
            <button className="goal-input-btn" onClick={handleSetTarget}>Set</button>
          </div>
        ) : (
          <button className="set-goal-btn" onClick={() => setEditingTarget(true)}>+ Set target word count</button>
        )}
      </div>

      {/* Daily Goal */}
      <div className="dashboard-section">
        <div className="dashboard-section-header">
          <span className="dashboard-section-title">Today's Writing</span>
        </div>
        {dailyGoal > 0 ? (
          <>
            <div className="dashboard-stat-row">
              <span className="stat-value">{(isNewDay ? sessionWords : todayWords).toLocaleString()}</span>
              <span className="stat-label">/ {dailyGoal.toLocaleString()} words</span>
            </div>
            <div className="progress-bar-container small">
              <div className="progress-bar">
                <div
                  className={`progress-bar-fill ${dailyProgress >= 100 ? 'complete' : ''}`}
                  style={{ width: `${dailyProgress}%` }}
                />
              </div>
            </div>
            <div className="daily-goal-edit" onClick={() => { setDailyInput(dailyGoal.toString()); setEditingDaily(true) }}>
              Edit daily goal
            </div>
          </>
        ) : editingDaily ? (
          <div className="goal-input-row">
            <input
              type="text"
              className="goal-input"
              value={dailyInput}
              onChange={e => setDailyInput(e.target.value)}
              placeholder="e.g. 1000"
              autoFocus
              onKeyDown={e => { if (e.key === 'Enter') handleSetDaily(); if (e.key === 'Escape') setEditingDaily(false) }}
            />
            <button className="goal-input-btn" onClick={handleSetDaily}>Set</button>
          </div>
        ) : (
          <button className="set-goal-btn" onClick={() => setEditingDaily(true)}>+ Set daily word goal</button>
        )}
      </div>

      {/* Session Stats */}
      <div className="dashboard-section">
        <div className="dashboard-section-header">
          <span className="dashboard-section-title">This Session</span>
        </div>
        <div className="session-stats-grid">
          <div className="session-stat">
            <span className="session-stat-value">{sessionWords.toLocaleString()}</span>
            <span className="session-stat-label">words written</span>
          </div>
          <div className="session-stat">
            <span className="session-stat-value">{sessionMinutes < 60 ? `${sessionMinutes}m` : `${Math.floor(sessionMinutes/60)}h ${sessionMinutes%60}m`}</span>
            <span className="session-stat-label">time writing</span>
          </div>
        </div>
      </div>

      {/* Chapter Stats */}
      <div className="dashboard-section">
        <div className="dashboard-section-header">
          <span className="dashboard-section-title">Chapter Stats</span>
        </div>
        <div className="chapter-stats-list">
          <div className="chapter-stat-item">
            <span className="chapter-stat-label">Chapters</span>
            <span className="chapter-stat-value">{chapters.length}</span>
          </div>
          <div className="chapter-stat-item">
            <span className="chapter-stat-label">Avg. length</span>
            <span className="chapter-stat-value">{avgWordsPerChapter.toLocaleString()} words</span>
          </div>
          <div className="chapter-stat-item">
            <span className="chapter-stat-label">Shortest</span>
            <span className="chapter-stat-value">{shortestChapter.toLocaleString()} words</span>
          </div>
          <div className="chapter-stat-item">
            <span className="chapter-stat-label">Longest</span>
            <span className="chapter-stat-value">{longestChapter.toLocaleString()} words</span>
          </div>
        </div>
      </div>

      {/* Quick Chapter Overview */}
      <div className="dashboard-section">
        <div className="dashboard-section-header">
          <span className="dashboard-section-title">Chapter Breakdown</span>
        </div>
        <div className="chapter-bars">
          {chapters.map((ch, i) => {
            const words = chapterWordCounts[i]
            const barWidth = longestChapter > 0 ? (words / longestChapter) * 100 : 0
            return (
              <div key={i} className="chapter-bar-row" title={`${ch.title}: ${words.toLocaleString()} words`}>
                <span className="chapter-bar-label">{ch.title.length > 15 ? ch.title.slice(0, 15) + '...' : ch.title}</span>
                <div className="chapter-bar-track">
                  <div className="chapter-bar-fill" style={{ width: `${barWidth}%` }} />
                </div>
                <span className="chapter-bar-count">{words.toLocaleString()}</span>
              </div>
            )
          })}
        </div>
      </div>

      {/* AI Detection Score */}
      <div className="dashboard-section">
        <div className="dashboard-section-header">
          <span className="dashboard-section-title">AI Detection</span>
          <span className="dashboard-section-hint">Current chapter</span>
        </div>
        {aiResult ? (
          <>
            <div className="ai-detection-score">
              <div className="ai-detection-gauge">
                <svg viewBox="0 0 100 50" className="ai-gauge-svg">
                  <path
                    d="M 10 45 A 40 40 0 0 1 90 45"
                    fill="none"
                    stroke="var(--bg-secondary)"
                    strokeWidth="8"
                    strokeLinecap="round"
                  />
                  <path
                    d="M 10 45 A 40 40 0 0 1 90 45"
                    fill="none"
                    stroke={getScoreColor(aiResult.score)}
                    strokeWidth="8"
                    strokeLinecap="round"
                    strokeDasharray={`${aiResult.score * 1.26} 126`}
                  />
                </svg>
                <div className="ai-gauge-value" style={{ color: getScoreColor(aiResult.score) }}>
                  {aiResult.score}%
                </div>
                <div className="ai-gauge-label">{getScoreLabel(aiResult.score)}</div>
              </div>
            </div>
            <div className="ai-detection-breakdown">
              <div className="ai-metric-row">
                <span className="ai-metric-label">Burstiness</span>
                <div className="ai-metric-bar">
                  <div className="ai-metric-fill" style={{ width: `${aiResult.breakdown.burstiness}%`, background: getScoreColor(aiResult.breakdown.burstiness) }} />
                </div>
              </div>
              <div className="ai-metric-row">
                <span className="ai-metric-label">Vocabulary</span>
                <div className="ai-metric-bar">
                  <div className="ai-metric-fill" style={{ width: `${aiResult.breakdown.vocabularyRichness}%`, background: getScoreColor(aiResult.breakdown.vocabularyRichness) }} />
                </div>
              </div>
              <div className="ai-metric-row">
                <span className="ai-metric-label">Repetition</span>
                <div className="ai-metric-bar">
                  <div className="ai-metric-fill" style={{ width: `${aiResult.breakdown.repetition}%`, background: getScoreColor(aiResult.breakdown.repetition) }} />
                </div>
              </div>
              <div className="ai-metric-row">
                <span className="ai-metric-label">Variety</span>
                <div className="ai-metric-bar">
                  <div className="ai-metric-fill" style={{ width: `${aiResult.breakdown.sentenceVariety}%`, background: getScoreColor(aiResult.breakdown.sentenceVariety) }} />
                </div>
              </div>
              <div className="ai-metric-row">
                <span className="ai-metric-label">AI Phrases</span>
                <div className="ai-metric-bar">
                  <div className="ai-metric-fill" style={{ width: `${aiResult.breakdown.transitionPatterns}%`, background: getScoreColor(aiResult.breakdown.transitionPatterns) }} />
                </div>
              </div>
            </div>
            {aiResult.flags.length > 0 && (
              <div className="ai-detection-flags">
                {aiResult.flags.slice(0, 3).map((flag, i) => (
                  <div key={i} className="ai-flag">{flag}</div>
                ))}
              </div>
            )}
            <div className="ai-detection-disclaimer">
              <strong>Estimate only — not a guarantee.</strong> This heuristic analysis may produce false positives or negatives. For authoritative verification, use Pangram or GPTZero. Draftline makes no claims about the accuracy of this score.
            </div>
          </>
        ) : (
          <div className="ai-detection-empty">
            Not enough text to analyze. Write more to see AI detection score.
          </div>
        )}
      </div>

      {/* Anti-Pattern Checker */}
      {antiPatternResult && antiPatternResult.patterns.length > 0 && (
        <div className="dashboard-section">
          <div className="dashboard-section-header">
            <span className="dashboard-section-title">AI Anti-Patterns</span>
            <span className="dashboard-section-hint">
              {antiPatternResult.overLimit.length > 0
                ? `${antiPatternResult.overLimit.length} over limit`
                : 'All within limits'}
            </span>
          </div>
          <div className="anti-pattern-list">
            {antiPatternResult.patterns.map((p, i) => (
              <div
                key={i}
                className={`anti-pattern-item ${p.severity}`}
                style={{ borderLeftColor: getAntiPatternColor(p.severity) }}
              >
                <span className="anti-pattern-name">{p.displayName}</span>
                <span className="anti-pattern-count">
                  {p.count}x <span className="anti-pattern-limit">(limit: {p.limitPerChapter}/ch)</span>
                </span>
              </div>
            ))}
          </div>
          <p className="settings-hint" style={{ marginTop: 8, fontSize: 10 }}>
            Patterns like "half-step", "eyes widened", etc. that AI tends to overuse.
          </p>
        </div>
      )}
    </div>
  )
}
