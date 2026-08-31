// Writing Dashboard — writing activity only (progress, goals, streak,
// session rate, chapter breakdown). Analysis lives in the analysis sidebars;
// AI detection shrinks to a jump row into its own panel.

import { useState, useEffect, useRef, useMemo } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../../../store/bookStore'
import { analyzeText, getScoreColor } from '../../../services/aiDetection'
import { countWords, countBookWords, getCurrentContent, htmlToText } from '../../../utils/textUtils'
import { bookKey, openToolsSection } from '../Analysis/shared'
import { recordTodayWords, getLastNDays, todayISO } from './history'
import './dashboard.css'

interface BreakdownRow {
  key: string
  label: string
  words: number
  muted: boolean
}

// "target 100k" / "target 800"
function formatTargetHint(target: number): string {
  return target >= 1000 ? `${Math.round(target / 1000)}k` : target.toString()
}

export default function DashboardTab() {
  const { book, updateWritingGoals, currentSection, currentIndex } = useBookStore(useShallow(s => ({
    book: s.book,
    updateWritingGoals: s.updateWritingGoals,
    currentSection: s.currentSection,
    currentIndex: s.currentIndex,
  })))

  // All hooks run unconditionally, before any early return.
  const [editingTarget, setEditingTarget] = useState(false)
  const [editingDaily, setEditingDaily] = useState(false)
  const [targetInput, setTargetInput] = useState('')
  const [dailyInput, setDailyInput] = useState('')
  const [showAllBreakdown, setShowAllBreakdown] = useState(false)

  // Re-render every 30s so "time writing" / "words / min" stay current.
  const [, setClockTick] = useState(0)
  useEffect(() => {
    const id = setInterval(() => setClockTick(t => t + 1), 30_000)
    return () => clearInterval(id)
  }, [])

  // Current chapter content for the AI-detection jump row.
  const currentContent = useMemo(
    () => getCurrentContent(book, currentSection, currentIndex),
    [book, currentSection, currentIndex],
  )

  // Debounced AI detection score — runs 2s after content stops changing.
  const [aiScore, setAiScore] = useState<number | null>(null)
  const analysisTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  useEffect(() => {
    if (analysisTimer.current) clearTimeout(analysisTimer.current)
    if (!currentContent || currentContent.length < 100) {
      setAiScore(null)
      return
    }
    analysisTimer.current = setTimeout(() => {
      // analyzeText needs ~100 chars of PLAIN text; HTML length overshoots
      // (a short chapter full of tags would otherwise show a junk 50%).
      const textLength = htmlToText(currentContent).trim().length
      setAiScore(textLength < 100 ? null : analyzeText(currentContent).score)
    }, 2000)
    return () => { if (analysisTimer.current) clearTimeout(analysisTimer.current) }
  }, [currentContent])

  // Whole-book count re-parses every chapter's HTML, so memoize on the book
  // reference — it must not re-run on unrelated store updates.
  const totalWords = useMemo(() => (book ? countBookWords(book) : 0), [book])

  // Session baseline — captured when a book first appears in this panel (and
  // re-captured when a DIFFERENT book is opened), not at mount time: mounting
  // with no book open must not make the whole manuscript later count as
  // "written this session" (which would also poison the streak history).
  const session = useRef<{ key: string; startTime: number; startWords: number } | null>(null)
  if (book) {
    const key = bookKey(book)
    if (session.current?.key !== key) {
      session.current = { key, startTime: Date.now(), startWords: totalWords }
    }
  }
  const sessionStart = session.current?.startTime ?? Date.now()
  const sessionStartWords = session.current?.startWords ?? 0

  const sectionCounts = useMemo(() => {
    const bodyCounts = (book?.body ?? []).map(ch => countWords(ch.content || ''))
    const frontWords = (book?.front_matter ?? []).reduce((sum, ch) => sum + countWords(ch.content || ''), 0)
    const backWords = (book?.back_matter ?? []).reduce((sum, ch) => sum + countWords(ch.content || ''), 0)
    return { bodyCounts, frontWords, backWords }
  }, [book])

  const targetWords = book?.writing_goals?.target_word_count || 0
  const dailyGoal = book?.writing_goals?.daily_word_goal || 0
  const todayWords = book?.writing_goals?.words_today || 0
  const lastWritingDate = book?.writing_goals?.last_writing_date || ''

  // words_today resets on a new day (tracked by last_writing_date); until the
  // store rolls it over, this session's words are today's words.
  const today = todayISO()
  const isNewDay = lastWritingDate !== today

  const sessionWords = Math.max(0, totalWords - sessionStartWords)
  const sessionMinutes = Math.floor((Date.now() - sessionStart) / 60000)
  const displayedToday = isNewDay ? sessionWords : todayWords

  // Persist today's count for the goal-streak row.
  useEffect(() => {
    if (!book) return
    recordTodayWords(book, displayedToday)
  }, [book, displayedToday, today])

  const streakDays = useMemo(() => {
    if (!book) return []
    const days = getLastNDays(book, 7)
    const last = days[days.length - 1]
    if (last) last.words = Math.max(last.words, displayedToday)
    return days
  }, [book, displayedToday, today])

  if (!book) {
    return <div className="tool-empty-state">Open or create a project to see your writing dashboard.</div>
  }

  const { bodyCounts, frontWords, backWords } = sectionCounts
  const chapters = book.body || []

  // Chapter stats (body chapters only).
  const avgWordsPerChapter = chapters.length > 0
    ? Math.round(bodyCounts.reduce((a, b) => a + b, 0) / chapters.length)
    : 0
  const shortestChapter = bodyCounts.length > 0 ? Math.min(...bodyCounts) : 0
  const longestChapter = bodyCounts.length > 0 ? Math.max(...bodyCounts) : 0

  const targetProgress = targetWords > 0 ? Math.min(100, (totalWords / targetWords) * 100) : 0
  const wordsToGo = Math.max(0, targetWords - totalWords)
  const dailyProgress = dailyGoal > 0 ? Math.min(100, (displayedToday / dailyGoal) * 100) : 0

  const streakCount = streakDays.filter(d => d.words > 0).length

  const sessionTime = sessionMinutes < 60
    ? `${sessionMinutes}m`
    : `${Math.floor(sessionMinutes / 60)}h ${sessionMinutes % 60}m`
  const wordsPerMinute = (sessionWords / Math.max(1, sessionMinutes)).toFixed(1)
  const sessionStartLabel = new Date(sessionStart).toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })

  // Chapter breakdown: aggregate front/back matter rows around body chapters.
  const breakdownRows: BreakdownRow[] = []
  if ((book.front_matter || []).length > 0) {
    breakdownRows.push({ key: 'front', label: 'Front matter', words: frontWords, muted: true })
  }
  chapters.forEach((ch, i) => {
    breakdownRows.push({ key: `body-${i}`, label: `${i + 1} · ${ch.title || 'Untitled'}`, words: bodyCounts[i], muted: false })
  })
  if ((book.back_matter || []).length > 0) {
    breakdownRows.push({ key: 'back', label: 'Back matter', words: backWords, muted: true })
  }
  const visibleRows = showAllBreakdown ? breakdownRows : breakdownRows.slice(0, 10)

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

  function openTargetEditor() {
    setTargetInput(targetWords > 0 ? targetWords.toString() : '')
    setEditingTarget(true)
  }

  function openDailyEditor() {
    setDailyInput(dailyGoal > 0 ? dailyGoal.toString() : '')
    setEditingDaily(true)
  }

  return (
    <div className="dashboard-content dash-panel">
      {/* Manuscript */}
      <div className="dashboard-section">
        <div className="dashboard-section-header">
          <span className="dashboard-section-title">Manuscript</span>
          {targetWords > 0 && (
            <button className="dash-hint-btn" onClick={openTargetEditor}>target {formatTargetHint(targetWords)}</button>
          )}
        </div>
        <div className="dashboard-big-number">
          <span className="big-number">{totalWords.toLocaleString()}</span>
          <span className="big-number-label">words</span>
        </div>
        {editingTarget ? (
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
        ) : targetWords > 0 ? (
          <div className="progress-bar-container">
            <div className="progress-bar">
              <div className="progress-bar-fill" style={{ width: `${targetProgress}%` }} />
            </div>
            <div className="progress-bar-labels">
              <span>{Math.round(targetProgress)}%</span>
              <button className="progress-target dash-linklike" onClick={openTargetEditor}>
                {wordsToGo.toLocaleString()} to go
              </button>
            </div>
          </div>
        ) : (
          <button className="set-goal-btn" onClick={openTargetEditor}>+ Set target word count</button>
        )}
      </div>

      {/* Today */}
      <div className="dashboard-section">
        <div className="dashboard-section-header">
          <span className="dashboard-section-title">Today</span>
          {dailyGoal > 0 && (
            <button className="dash-hint-btn" onClick={openDailyEditor}>goal {dailyGoal.toLocaleString()}</button>
          )}
        </div>
        {editingDaily ? (
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
        ) : dailyGoal > 0 ? (
          <>
            <div className="dashboard-big-number">
              <span className="big-number">{displayedToday.toLocaleString()}</span>
              <span className="big-number-label">of {dailyGoal.toLocaleString()} words</span>
            </div>
            <div className="progress-bar-container small">
              <div className="progress-bar">
                <div
                  className={`progress-bar-fill ${dailyProgress >= 100 ? 'complete' : ''}`}
                  style={{ width: `${dailyProgress}%` }}
                />
              </div>
            </div>
          </>
        ) : (
          <button className="set-goal-btn" onClick={openDailyEditor}>+ Set daily word goal</button>
        )}
        <div className="dash-streak">
          {streakDays.map(day => {
            const isToday = day.date === today
            const cls = isToday ? 'today' : day.words > 0 ? 'written' : ''
            return (
              <span
                key={day.date}
                className={`dash-streak-day ${cls}`}
                title={`${day.date}: ${day.words.toLocaleString()} words`}
              />
            )
          })}
          <span className="dash-streak-caption">{streakCount} of last 7 days</span>
        </div>
      </div>

      {/* This session */}
      <div className="dashboard-section">
        <div className="dashboard-section-header">
          <span className="dashboard-section-title">This session</span>
          <span className="dashboard-section-hint">started {sessionStartLabel}</span>
        </div>
        <div className="dash-session-row">
          <div className="dash-session-stat">
            <div className="dash-session-value">{sessionWords.toLocaleString()}</div>
            <div className="dash-session-label">words written</div>
          </div>
          <div className="dash-session-stat">
            <div className="dash-session-value">{sessionTime}</div>
            <div className="dash-session-label">time writing</div>
          </div>
          <div className="dash-session-stat">
            <div className="dash-session-value">{wordsPerMinute}</div>
            <div className="dash-session-label">words / min</div>
          </div>
        </div>
      </div>

      {/* Chapter stats */}
      <div className="dashboard-section">
        <div className="dashboard-section-header">
          <span className="dashboard-section-title">Chapter stats</span>
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

      {/* Chapter breakdown */}
      <div className="dashboard-section">
        <div className="dashboard-section-header">
          <span className="dashboard-section-title">Chapter breakdown</span>
          <span className="dashboard-section-hint">all sections</span>
        </div>
        <div className="dash-bd-list">
          {visibleRows.map(row => {
            const barWidth = longestChapter > 0 ? Math.min(100, (row.words / longestChapter) * 100) : 0
            return (
              <div key={row.key} className="chapter-bar-row" title={`${row.label}: ${row.words.toLocaleString()} words`}>
                <span className={`chapter-bar-label dash-bd-label${row.muted ? ' dash-bd-muted' : ''}`}>{row.label}</span>
                <div className="chapter-bar-track">
                  <div className="chapter-bar-fill" style={{ width: `${barWidth}%` }} />
                </div>
                <span className="chapter-bar-count">{row.words.toLocaleString()}</span>
              </div>
            )
          })}
          {breakdownRows.length > 10 && (
            <button className="dash-show-toggle" onClick={() => setShowAllBreakdown(v => !v)}>
              {showAllBreakdown ? 'Show fewer' : `Show all ${breakdownRows.length}`}
            </button>
          )}
        </div>
      </div>

      {/* AI detection jump row */}
      <button className="dash-jump" onClick={() => openToolsSection('aidetect')}>
        <span className="dash-jump-label">AI detection — current chapter</span>
        <span className="dash-jump-meta">
          {aiScore !== null && (
            <span className="dash-jump-score" style={{ color: getScoreColor(aiScore) }}>{aiScore}%</span>
          )}
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="9 18 15 12 9 6" />
          </svg>
        </span>
      </button>
    </div>
  )
}
