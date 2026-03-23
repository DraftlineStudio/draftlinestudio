import { useState, useEffect, useRef, useMemo } from 'react'
import { useBookStore } from '../store/bookStore'
import { useAppStore } from '../store/appStore'
import { RewriteText, CancelRewrite } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import type { Character, CharacterRole, WritingStyleOptions } from '../types/draftline'
import { DEFAULT_STYLE_OPTIONS } from '../types/draftline'
import { analyzeText, getScoreColor, getScoreLabel, type AIDetectionResult } from '../services/aiDetection'

type Tab = 'dashboard' | 'bible' | 'ai'
type AIMode = 'line_edit' | 'expand' | 'smooth'
type AIState = 'idle' | 'loading' | 'voice' | 'error'
type BibleSection = 'characters' | 'plot' | 'timeline'

const AI_MODES: { id: AIMode; label: string; desc: string }[] = [
  { id: 'line_edit', label: 'Line Edit', desc: 'Prose rhythm and sentence variety' },
  { id: 'expand', label: 'Expand', desc: 'Add detail, texture, show vs. tell' },
  { id: 'smooth', label: 'Smooth', desc: 'Remove repetition, improve flow' },
]

// Style feature definitions for the mixer
const STYLE_FEATURES: { key: keyof WritingStyleOptions; label: string; desc: string }[] = [
  { key: 'metaphors', label: 'Metaphors', desc: 'Figurative comparisons' },
  { key: 'similes', label: 'Similes', desc: '"Like" and "as" comparisons' },
  { key: 'sensory_detail', label: 'Sensory Detail', desc: 'Sight, sound, smell, touch, taste' },
  { key: 'internal_thought', label: 'Internal Thought', desc: 'Character introspection' },
  { key: 'dialogue', label: 'Dialogue', desc: 'Conversation expansion' },
  { key: 'action', label: 'Action', desc: 'Physical beats, movement' },
  { key: 'description', label: 'Description', desc: 'Setting and atmosphere' },
  { key: 'pacing', label: 'Pacing', desc: 'Sentence rhythm variation' },
]

const INTENSITY_LABELS = ['Off', 'Subtle', 'Moderate', 'Heavy']

function genId(): string {
  return Math.random().toString(36).slice(2) + Date.now().toString(36)
}

function countWords(html: string): number {
  const div = document.createElement('div')
  div.innerHTML = html
  const text = div.textContent || div.innerText || ''
  return text.trim().split(/\s+/).filter((w) => w.length > 0).length
}

// ── Main panel ──────────────────────────────────────────────────────────────

export default function ToolsPanel() {
  const [activeTab, setActiveTab] = useState<Tab>('dashboard')
  const { rightPanelOpen, toggleRightPanel } = useBookStore()

  return (
    <div className={`tools-panel${rightPanelOpen ? '' : ' collapsed'}`}>
      <div className="tools-panel-inner">
        <div className="tools-tabs">
          <button className="panel-collapse-btn" onClick={toggleRightPanel} title="Collapse panel">
            <svg width="6" height="10" viewBox="0 0 6 10" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
              <path d="M1 1l4 4-4 4" />
            </svg>
          </button>
          <button className={`tools-tab${activeTab === 'dashboard' ? ' active' : ''}`} onClick={() => setActiveTab('dashboard')}>Dashboard</button>
          <button className={`tools-tab${activeTab === 'bible' ? ' active' : ''}`} onClick={() => setActiveTab('bible')}>Bible</button>
          <button className={`tools-tab${activeTab === 'ai' ? ' active' : ''}`} onClick={() => setActiveTab('ai')}>AI</button>
        </div>
        <div className="tools-content">
          {activeTab === 'dashboard' && <DashboardTab />}
          {activeTab === 'bible' && <StoryBibleTab />}
          {activeTab === 'ai' && <AiStudioTab />}
        </div>
      </div>
    </div>
  )
}

// ── Writing Dashboard ───────────────────────────────────────────────────────

function DashboardTab() {
  const { book, updateWritingGoals, currentSection, currentIndex } = useBookStore()
  const [sessionStart] = useState(() => Date.now())
  const [sessionStartWords] = useState(() => {
    if (!book) return 0
    return getTotalWords(book)
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
  const analysisTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (analysisTimer.current) clearTimeout(analysisTimer.current)
    if (!currentContent || currentContent.length < 100) {
      setAiResult(null)
      return
    }
    analysisTimer.current = setTimeout(() => {
      setAiResult(analyzeText(currentContent))
    }, 2000)
    return () => { if (analysisTimer.current) clearTimeout(analysisTimer.current) }
  }, [currentContent])

  // Calculate metrics
  const totalWords = getTotalWords(book)
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
  const chapterWordCounts = chapters.map(ch => countWords(ch.content || ''))
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
    </div>
  )
}

function getTotalWords(book: any): number {
  let total = 0
  total += countWords(book.copyright || '')
  for (const ch of book.front_matter || []) total += countWords(ch.content || '')
  for (const ch of book.body || []) total += countWords(ch.content || '')
  for (const ch of book.back_matter || []) total += countWords(ch.content || '')
  return total
}

// ── AI Studio ───────────────────────────────────────────────────────────────

function AiStudioTab() {
  const { book, currentSection, currentIndex, setPendingDiff, getStyleOptions, updateStyleOptions } = useBookStore()
  const { settings, openSettings } = useAppStore()

  const [aiMode, setAiMode] = useState<AIMode>('line_edit')
  const [aiState, setAiState] = useState<AIState>('idle')
  const [error, setError] = useState('')
  const [showImport, setShowImport] = useState(false)
  const [importText, setImportText] = useState('')
  const [showStyleMixer, setShowStyleMixer] = useState(false)

  const styleOptions = getStyleOptions()

  const aiConfigured = settings.ai_enabled && (
    settings.ai_mode === 'claudecode' ||
    (settings.ai_mode === 'api' && settings.ai_provider !== '' && settings.ai_api_key !== '') ||
    (settings.ai_mode === 'local' && settings.ai_local_endpoint !== '')
  )

  function getCurrentHTML(): string {
    if (!book) return ''
    if (currentSection === 'copyright') return book.copyright
    const arr = currentSection === 'front_matter' ? book.front_matter
      : currentSection === 'body' ? book.body
        : book.back_matter
    return arr[currentIndex]?.content ?? ''
  }

  function getAiLabel(): string {
    if (settings.ai_mode === 'claudecode') return 'Claude Code'
    if (settings.ai_mode === 'local') return settings.ai_local_model || 'Local AI'
    return settings.ai_provider === 'claude' ? 'Claude'
      : settings.ai_provider === 'openai' ? 'OpenAI'
        : 'AI'
  }

  const currentMode = AI_MODES.find(m => m.id === aiMode)!

  async function handleRun() {
    const html = getCurrentHTML()
    if (!html || html === '<p></p>') return
    setAiState('loading')
    setError('')
    try {
      // Pass style options for expand/smooth modes
      const useStyleOptions = aiMode === 'expand' || aiMode === 'smooth'
      const res = await RewriteText(html, aiMode, useStyleOptions ? JSON.stringify(styleOptions) : '')
      if (res.error) {
        setError(res.error)
        setAiState('error')
      } else {
        const { diffContent } = await import('../utils/diff')
        const d = diffContent(html, res.result)
        setPendingDiff({ diffs: d, originalHtml: html })
        setAiState('idle')
      }
    } catch (e) {
      setError(String(e))
      setAiState('error')
    }
  }

  function handleStyleChange(key: keyof WritingStyleOptions, value: number) {
    updateStyleOptions({ [key]: value })
  }

  function handleImportCompare() {
    const html = getCurrentHTML()
    if (!html || !importText.trim()) return
    const pasted = importText.trim().startsWith('<')
      ? importText
      : importText.split(/\n{2,}/).map(p => `<p>${p.trim()}</p>`).join('\n')
    import('../utils/diff').then(({ diffContent }) => {
      const d = diffContent(html, pasted)
      setPendingDiff({ diffs: d, originalHtml: html })
      setShowImport(false)
      setImportText('')
    })
  }

  if (!book) {
    return <div className="tool-empty-state">Open or create a project to use AI tools.</div>
  }

  // ── Render states ──

  if (aiState === 'loading') {
    return (
      <LoadingPane
        label={`${currentMode.label} · ${getAiLabel()}`}
        onCancel={() => {
          CancelRewrite()
          setAiState('idle')
        }}
      />
    )
  }

  if (aiState === 'error') {
    return (
      <div className="ai-error-pane">
        <div className="ai-error-msg">{error}</div>
        <button className="ai-link-btn" onClick={() => setAiState('idle')}>← Back</button>
      </div>
    )
  }

  // ── Idle state ──
  const showMixer = (aiMode === 'expand' || aiMode === 'smooth')

  return (
    <>
      {/* Mode selector */}
      <div className="ai-section-label">Editing Mode</div>
      <div className="ai-mode-list">
        {AI_MODES.map(m => (
          <button
            key={m.id}
            className={`ai-mode-row${aiMode === m.id ? ' selected' : ''}`}
            onClick={() => setAiMode(m.id)}
          >
            <span className="ai-mode-dot" />
            <span className="ai-mode-name">{m.label}</span>
            <span className="ai-mode-hint">{m.desc}</span>
          </button>
        ))}
      </div>

      {/* Style Mixer for Expand/Smooth modes */}
      {showMixer && (
        <div className="style-mixer-section">
          <button
            className="style-mixer-toggle"
            onClick={() => setShowStyleMixer(!showStyleMixer)}
          >
            <span>Style Options</span>
            <svg
              className={`style-mixer-chevron${showStyleMixer ? ' open' : ''}`}
              width="10"
              height="6"
              viewBox="0 0 10 6"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M1 1l4 4 4-4" />
            </svg>
          </button>
          {showStyleMixer && (
            <div className="style-mixer-content">
              {STYLE_FEATURES.map(feature => (
                <div key={feature.key} className="style-feature-row">
                  <div className="style-feature-header">
                    <span className="style-feature-label">{feature.label}</span>
                    <span className="style-feature-value">{INTENSITY_LABELS[styleOptions[feature.key]]}</span>
                  </div>
                  <div className="style-feature-slider">
                    <input
                      type="range"
                      min="0"
                      max="3"
                      value={styleOptions[feature.key]}
                      onChange={e => handleStyleChange(feature.key, parseInt(e.target.value))}
                      className="style-slider"
                    />
                    <div className="style-slider-marks">
                      {INTENSITY_LABELS.map((label, i) => (
                        <span
                          key={i}
                          className={`style-slider-mark${styleOptions[feature.key] === i ? ' active' : ''}`}
                        />
                      ))}
                    </div>
                  </div>
                </div>
              ))}
              <div className="style-mixer-hint">
                Customize how the AI enhances your writing. "Off" skips that feature entirely.
              </div>
            </div>
          )}
        </div>
      )}

      {/* Import pane */}
      {showImport ? (
        <div className="import-pane">
          <div className="import-pane-label">Paste edited version — plain text or HTML:</div>
          <textarea
            className="import-edits-textarea"
            value={importText}
            onChange={e => setImportText(e.target.value)}
            placeholder="Paste the edited chapter here…"
            autoFocus
          />
          <button className="ai-run-btn" onClick={handleImportCompare} disabled={!importText.trim()}>
            Compare with Current
          </button>
          <button className="ai-link-btn" onClick={() => { setShowImport(false); setImportText('') }}>Cancel</button>
        </div>
      ) : (
        <div className="ai-actions">
          {aiConfigured ? (
            <button className="ai-run-btn" onClick={handleRun}>
              Run {currentMode.label}
            </button>
          ) : (
            <button className="ai-run-btn configure" onClick={openSettings}>Configure AI ›</button>
          )}
          {aiConfigured && (
            <button className="ai-link-btn" onClick={() => setShowImport(true)}>
              or compare with imported draft ›
            </button>
          )}
        </div>
      )}
    </>
  )
}

// ── Loading pane ─────────────────────────────────────────────────────────────

function LoadingPane({ label, onCancel }: { label: string; onCancel: () => void }) {
  const [elapsed, setElapsed] = useState(0)
  const [logLines, setLogLines] = useState<string[]>([])
  const [streamText, setStreamText] = useState('')
  const logRef = useRef<HTMLDivElement>(null)
  const streamRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const t = setInterval(() => setElapsed(s => s + 1), 1000)
    return () => clearInterval(t)
  }, [])

  useEffect(() => {
    const handler = (line: string) => {
      setLogLines(prev => {
        const next = [...prev, line]
        return next.length > 20 ? next.slice(-20) : next
      })
    }
    EventsOn('ai:log', handler)
    return () => EventsOff('ai:log')
  }, [])

  useEffect(() => {
    const handler = (token: string) => setStreamText(prev => prev + token)
    EventsOn('ai:token', handler)
    return () => EventsOff('ai:token')
  }, [])

  useEffect(() => {
    if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight
  }, [logLines])

  useEffect(() => {
    if (streamRef.current) streamRef.current.scrollTop = streamRef.current.scrollHeight
  }, [streamText])

  const mins = Math.floor(elapsed / 60)
  const secs = elapsed % 60
  const timeStr = mins > 0 ? `${mins}m ${secs}s` : `${secs}s`

  return (
    <div className="ai-loading-pane">
      <div className="ai-loading-header">
        <div className="ai-spinner" />
        <span className="ai-loading-label">{label}</span>
        <span className="ai-loading-time">{timeStr}</span>
      </div>
      {streamText ? (
        <div className="ai-stream-box" ref={streamRef}>{streamText}</div>
      ) : logLines.length > 0 && (
        <div className="ai-log-box" ref={logRef}>
          {logLines.map((line, i) => <div key={i} className="ai-log-line">{line}</div>)}
        </div>
      )}
      <button className="ai-link-btn" style={{ marginTop: 10 }} onClick={onCancel}>
        Cancel
      </button>
    </div>
  )
}

// ── Story Bible ──────────────────────────────────────────────────────────────

function StoryBibleTab() {
  const [section, setSection] = useState<BibleSection>('characters')
  return (
    <>
      <div className="story-bible-subnav">
        <button className={`bible-nav-btn${section === 'characters' ? ' active' : ''}`} onClick={() => setSection('characters')}>Characters</button>
        <button className={`bible-nav-btn${section === 'plot' ? ' active' : ''}`} onClick={() => setSection('plot')}>Plot</button>
        <button className={`bible-nav-btn${section === 'timeline' ? ' active' : ''}`} onClick={() => setSection('timeline')}>Timeline</button>
      </div>
      {section === 'characters' && <CharactersSection />}
      {section === 'plot' && <PlotSection />}
      {section === 'timeline' && <TimelineSection />}
    </>
  )
}

function CharactersSection() {
  const { book, addCharacter, updateCharacter, deleteCharacter } = useBookStore()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [addingNew, setAddingNew] = useState(false)

  if (!book) return <div className="tool-empty-state">Open a project to manage characters.</div>

  const characters = book.story_bible?.characters ?? []

  return (
    <>
      {addingNew && (
        <CharacterForm char={null} onSave={c => { addCharacter(c); setAddingNew(false) }} onCancel={() => setAddingNew(false)} />
      )}
      {characters.map(char =>
        editingId === char.id ? (
          <CharacterForm key={char.id} char={char} onSave={c => { updateCharacter(c); setEditingId(null) }} onCancel={() => setEditingId(null)} />
        ) : (
          <CharacterCard key={char.id} char={char} onEdit={() => setEditingId(char.id)} onDelete={() => deleteCharacter(char.id)} />
        )
      )}
      {!addingNew && (
        <button className="tool-card-btn" style={{ marginTop: 4 }} onClick={() => setAddingNew(true)}>+ Add Character</button>
      )}
    </>
  )
}

function CharacterCard({ char, onEdit, onDelete }: { char: Character; onEdit: () => void; onDelete: () => void }) {
  const [expanded, setExpanded] = useState(false)
  return (
    <div className="character-card" onClick={() => setExpanded(v => !v)}>
      <div className="character-card-header">
        <span className="character-name">{char.name || 'Unnamed'}</span>
        <span className={`character-role ${char.role}`}>{char.role}</span>
      </div>
      {char.description && <div className="character-desc-preview">{char.description}</div>}
      {expanded && (
        <div className="character-expanded" onClick={e => e.stopPropagation()}>
          {char.appearance && <div className="char-field"><span className="char-field-label">Appearance</span><span>{char.appearance}</span></div>}
          {char.personality && <div className="char-field"><span className="char-field-label">Personality</span><span>{char.personality}</span></div>}
          {char.motivation && <div className="char-field"><span className="char-field-label">Motivation</span><span>{char.motivation}</span></div>}
          {char.notes && <div className="char-field"><span className="char-field-label">Notes</span><span>{char.notes}</span></div>}
          <div className="character-card-actions">
            <button className="tool-card-btn secondary" style={{ fontSize: 11, padding: '2px 10px' }} onClick={e => { e.stopPropagation(); onEdit() }}>Edit</button>
            <button className="tool-card-btn secondary" style={{ fontSize: 11, padding: '2px 10px', color: '#E06C75' }} onClick={e => { e.stopPropagation(); onDelete() }}>Delete</button>
          </div>
        </div>
      )}
    </div>
  )
}

function CharacterForm({ char, onSave, onCancel }: { char: Character | null; onSave: (c: Character) => void; onCancel: () => void }) {
  const [name, setName] = useState(char?.name ?? '')
  const [role, setRole] = useState<CharacterRole | string>(char?.role ?? 'supporting')
  const [desc, setDesc] = useState(char?.description ?? '')
  const [app, setApp] = useState(char?.appearance ?? '')
  const [persona, setPersona] = useState(char?.personality ?? '')
  const [motiv, setMotiv] = useState(char?.motivation ?? '')
  const [notes, setNotes] = useState(char?.notes ?? '')

  return (
    <div className="character-form">
      <div className="character-form-title">{char ? 'Edit Character' : 'New Character'}</div>
      <div className="character-form-row">
        <div className="dialog-field" style={{ margin: 0 }}>
          <label className="dialog-label">Name</label>
          <input className="dialog-input" value={name} onChange={e => setName(e.target.value)} placeholder="Character name" autoFocus />
        </div>
        <div className="dialog-field" style={{ margin: 0 }}>
          <label className="dialog-label">Role</label>
          <select className="dialog-select" value={role} onChange={e => setRole(e.target.value as CharacterRole)}>
            <option value="protagonist">Protagonist</option>
            <option value="antagonist">Antagonist</option>
            <option value="supporting">Supporting</option>
            <option value="minor">Minor</option>
            <option value="other">Other</option>
          </select>
        </div>
      </div>
      <div className="dialog-field" style={{ marginTop: 6 }}>
        <label className="dialog-label">Description</label>
        <textarea className="dialog-input" rows={2} value={desc} onChange={e => setDesc(e.target.value)} placeholder="Brief overview of this character's role" style={{ resize: 'vertical' }} />
      </div>
      <div className="dialog-field" style={{ marginTop: 4 }}>
        <label className="dialog-label">Appearance</label>
        <textarea className="dialog-input" rows={2} value={app} onChange={e => setApp(e.target.value)} placeholder="Physical description, distinguishing features" style={{ resize: 'vertical' }} />
      </div>
      <div className="character-form-row" style={{ marginTop: 4 }}>
        <div className="dialog-field" style={{ margin: 0 }}>
          <label className="dialog-label">Personality</label>
          <textarea className="dialog-input" rows={2} value={persona} onChange={e => setPersona(e.target.value)} placeholder="Traits, quirks, voice" style={{ resize: 'vertical' }} />
        </div>
        <div className="dialog-field" style={{ margin: 0 }}>
          <label className="dialog-label">Motivation</label>
          <textarea className="dialog-input" rows={2} value={motiv} onChange={e => setMotiv(e.target.value)} placeholder="Goals, fears, driving need" style={{ resize: 'vertical' }} />
        </div>
      </div>
      <div className="dialog-field" style={{ marginTop: 4 }}>
        <label className="dialog-label">Notes</label>
        <textarea className="dialog-input" rows={2} value={notes} onChange={e => setNotes(e.target.value)} placeholder="Arc beats, continuity flags, backstory" style={{ resize: 'vertical' }} />
      </div>
      <div className="ai-actions" style={{ marginTop: 8 }}>
        <button className="ai-run-btn" onClick={() => { if (name.trim()) onSave({ id: char?.id ?? genId(), name: name.trim(), role, description: desc, appearance: app, personality: persona, motivation: motiv, notes }) }} disabled={!name.trim()}>
          {char ? 'Save Changes' : 'Add Character'}
        </button>
        <button className="ai-link-btn" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}

function PlotSection() {
  const { book, updateStoryBibleText } = useBookStore()
  if (!book) return <div className="tool-empty-state">Open a project first.</div>
  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Plot Bible</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>Outline acts, subplots, themes, and unresolved threads.</p>
      <textarea
        className="bible-textarea"
        value={book.story_bible?.plot_notes ?? ''}
        onChange={e => updateStoryBibleText('plot_notes', e.target.value)}
        placeholder={"Act 1:\n  Opening image...\n  Inciting incident...\n\nAct 2:\n  ...\n\nThemes:\n  ..."}
      />
    </div>
  )
}

function TimelineSection() {
  const { book, updateStoryBibleText } = useBookStore()
  if (!book) return <div className="tool-empty-state">Open a project first.</div>
  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Story Timeline</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>Chronological events in story time. One event per line.</p>
      <textarea
        className="bible-textarea"
        value={book.story_bible?.timeline ?? ''}
        onChange={e => updateStoryBibleText('timeline', e.target.value)}
        placeholder={"Day 1 — Marcus arrives in the city (Ch. 1)\nDay 1, evening — He meets Clara (Ch. 2)\nDay 3 — The letter arrives (Ch. 4)"}
      />
    </div>
  )
}
