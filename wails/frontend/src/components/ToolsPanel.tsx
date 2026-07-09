import { useState, useEffect, useRef, useMemo } from 'react'
import { useBookStore, getGlobalChapterIndex } from '../store/bookStore'
import { useAppStore } from '../store/appStore'
import { RewriteText, RewriteTextCustom, CancelRewrite, CheckClaudeCode } from '../../wailsjs/go/main/App'
import type { types } from '../../wailsjs/go/models'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import type { Character, CharacterRole, WritingStyleOptions, Beat, BeatType, ForeshadowingItem, ForeshadowingStatus, SecretInfo, KnowledgeEntry } from '../types/draftline'
import { DEFAULT_STYLE_OPTIONS } from '../types/draftline'
import { analyzeText, getScoreColor, getScoreLabel, analyzeAntiPatterns, getAntiPatternColor, type AIDetectionResult, type AntiPatternResult } from '../services/aiDetection'
import { countWords, countBookWords } from '../utils/textUtils'

// Extracted types, constants, and components
import type { GlyphSection, AIMode, AIState } from './tools/types'
import { genId } from './tools/types'
import { AI_MODES, STYLE_FEATURES, INTENSITY_LABELS, SECTION_CONFIG, BEAT_TYPES } from './tools/constants'
import GlyphIcon from './tools/GlyphIcon'

// ── Main panel ──────────────────────────────────────────────────────────────

export default function ToolsPanel() {
  const [activeSection, setActiveSection] = useState<GlyphSection>(null)
  const [panelWidth, setPanelWidth] = useState(350)
  const [isResizing, setIsResizing] = useState(false)
  const resizeRef = useRef<{ startX: number; startWidth: number } | null>(null)
  const { settings, saveSettings } = useAppStore()

  // Load saved panel width from settings
  useEffect(() => {
    if (settings.sidebar_panel_width) {
      setPanelWidth(settings.sidebar_panel_width)
    }
  }, [settings.sidebar_panel_width])

  // Handle resize
  const handleResizeStart = (e: React.MouseEvent) => {
    e.preventDefault()
    setIsResizing(true)
    resizeRef.current = { startX: e.clientX, startWidth: panelWidth }
    document.body.style.cursor = 'ew-resize'
    document.body.style.userSelect = 'none'
  }

  useEffect(() => {
    if (!isResizing) return

    const handleMouseMove = (e: MouseEvent) => {
      if (!resizeRef.current) return
      const delta = resizeRef.current.startX - e.clientX
      const newWidth = Math.min(500, Math.max(280, resizeRef.current.startWidth + delta))
      setPanelWidth(newWidth)
    }

    const handleMouseUp = () => {
      setIsResizing(false)
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
      // Save width to settings
      saveSettings({ sidebar_panel_width: panelWidth })
    }

    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
    return () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }, [isResizing, panelWidth, saveSettings])

  const handleGlyphClick = (section: Exclude<GlyphSection, null>) => {
    setActiveSection(prev => prev === section ? null : section)
  }

  const visibleSections = SECTION_CONFIG.filter(s => {
    if (s.id === 'ai') return settings.show_ai_tab
    return true
  })

  const activeSectionConfig = activeSection ? SECTION_CONFIG.find(s => s.id === activeSection) : null

  return (
    <div className="tools-sidebar">
      {/* Slide-out panel */}
      {activeSection && (
        <div className="slide-panel" style={{ width: panelWidth }}>
          <div className="resize-handle" onMouseDown={handleResizeStart} />
          <div className="slide-panel-header">
            <span className="slide-panel-title">{activeSectionConfig?.tooltip}</span>
            <button className="slide-panel-close" onClick={() => setActiveSection(null)} title="Close panel">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>
          <div className="slide-panel-content">
            {activeSection === 'dashboard' && <DashboardTab />}
            {activeSection === 'characters' && <CharactersSection />}
            {activeSection === 'plot' && <PlotSection />}
            {activeSection === 'timeline' && <TimelineSection />}
            {activeSection === 'beats' && <BeatsSection />}
            {activeSection === 'foreshadow' && <ForeshadowingSection />}
            {activeSection === 'knowledge' && <KnowledgeSection />}
            {activeSection === 'issues' && <IssuesSection />}
            {activeSection === 'ai' && <AiStudioTab />}
          </div>
        </div>
      )}

      {/* Glyph bar */}
      <div className="glyph-bar">
        {visibleSections.map(section => (
          <button
            key={section.id}
            className={`glyph-btn${activeSection === section.id ? ' active' : ''}`}
            onClick={() => handleGlyphClick(section.id)}
            title={section.tooltip}
          >
            <GlyphIcon section={section.id} />
          </button>
        ))}
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

  // Calculate metrics
  const totalWords = countBookWords(book)
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

// ── AI Studio ───────────────────────────────────────────────────────────────

function AiStudioTab() {
  const { book, currentSection, currentIndex, setPendingDiff, getStyleOptions, updateStyleOptions, getEditorSelection } = useBookStore()
  const { settings, openSettings } = useAppStore()

  const [aiMode, setAiMode] = useState<AIMode>('line_edit')
  const [aiState, setAiState] = useState<AIState>('idle')
  const [error, setError] = useState('')
  const [showImport, setShowImport] = useState(false)
  const [importText, setImportText] = useState('')
  const [showStyleMixer, setShowStyleMixer] = useState(false)
  const [customPrompt, setCustomPrompt] = useState('')
  const [ccStatus, setCcStatus] = useState<types.ClaudeCodeStatus | null>(null)
  const [ccChecking, setCcChecking] = useState(false)

  const styleOptions = getStyleOptions()
  const selection = getEditorSelection()

  // Check Claude Code status on mount
  useEffect(() => {
    if (!ccStatus && !ccChecking) {
      setCcChecking(true)
      CheckClaudeCode()
        .then(setCcStatus)
        .catch(() => {})
        .finally(() => setCcChecking(false))
    }
  }, [])

  // Determine if AI is configured based on mode
  const aiConfigured = settings.ai_enabled && (
    (settings.ai_mode === 'claudecode' && ccStatus?.installed && ccStatus?.authenticated) ||
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
    const providerLabels: Record<string, string> = {
      claude: 'Claude',
      openai: 'OpenAI',
      gemini: 'Gemini',
      grok: 'Grok',
    }
    return providerLabels[settings.ai_provider] || 'AI'
  }

  const currentMode = AI_MODES.find(m => m.id === aiMode)!

  async function handleRun() {
    const fullHtml = getCurrentHTML()
    if (!fullHtml || fullHtml === '<p></p>') return

    // Determine what text to process: selection or full chapter
    const currentSelection = getEditorSelection()
    const hasSelection = currentSelection && currentSelection.text.trim().length > 0

    // For custom mode, we need a prompt
    if (aiMode === 'custom') {
      // Look for @ai commands in the text, or use the custom prompt input
      const textToSearch = hasSelection ? currentSelection.text : fullHtml.replace(/<[^>]*>/g, ' ')
      const aiMatch = textToSearch.match(/@ai\s+(.+?)(?:\n|$)/i)
      const prompt = customPrompt.trim() || (aiMatch ? aiMatch[1].trim() : '')

      if (!prompt) {
        setError('Enter a prompt below or use @ai in your text (e.g., "@ai expand this scene")')
        setAiState('error')
        return
      }
    }

    setAiState('loading')
    setError('')
    try {
      let res: { result: string; error?: string }
      let originalHtml: string

      if (aiMode === 'custom') {
        // Custom mode: use the prompt
        const textToProcess = hasSelection ? currentSelection.text : fullHtml
        const textToSearch = hasSelection ? currentSelection.text : fullHtml.replace(/<[^>]*>/g, ' ')
        const aiMatch = textToSearch.match(/@ai\s+(.+?)(?:\n|$)/i)
        const prompt = customPrompt.trim() || (aiMatch ? aiMatch[1].trim() : '')

        // Remove the @ai command from the text if present
        let cleanText = textToProcess
        if (aiMatch) {
          cleanText = textToProcess.replace(/@ai\s+.+?(?:\n|$)/gi, '')
        }

        res = await RewriteTextCustom(cleanText, prompt)
        originalHtml = hasSelection ? currentSelection.text : fullHtml
      } else {
        // Standard modes: use selection if available
        const textToProcess = hasSelection ? wrapSelectionAsHtml(currentSelection.text) : fullHtml
        const useStyleOptions = aiMode === 'expand' || aiMode === 'smooth'
        res = await RewriteText(textToProcess, aiMode, useStyleOptions ? JSON.stringify(styleOptions) : '')
        originalHtml = textToProcess
      }

      if (res.error) {
        setError(res.error)
        setAiState('error')
      } else {
        const { diffContent } = await import('../utils/diff')
        const d = diffContent(originalHtml, res.result)
        setPendingDiff({ diffs: d, originalHtml })
        setAiState('idle')
      }
    } catch (e) {
      setError(String(e))
      setAiState('error')
    }
  }

  // Helper to wrap plain text selection as HTML paragraphs
  function wrapSelectionAsHtml(text: string): string {
    if (text.trim().startsWith('<')) return text
    return text.split(/\n\n+/).map(p => `<p>${p.trim()}</p>`).join('\n')
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
    const isConnectionError = error.toLowerCase().includes('connection') ||
      error.toLowerCase().includes('network') ||
      error.toLowerCase().includes('failed to fetch') ||
      error.toLowerCase().includes('econnrefused')
    const isTimeoutError = error.toLowerCase().includes('timeout') ||
      error.toLowerCase().includes('timed out')
    const isAuthError = error.toLowerCase().includes('auth') ||
      error.toLowerCase().includes('401') ||
      error.toLowerCase().includes('403') ||
      error.toLowerCase().includes('invalid api key')
    const isNotInstalled = error.toLowerCase().includes('not installed')

    return (
      <div className="ai-error-pane">
        <div className="ai-error-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
        </div>
        <div className="ai-error-title">
          {isConnectionError ? 'Connection Failed' :
           isTimeoutError ? 'Request Timed Out' :
           isAuthError ? 'Authentication Error' :
           isNotInstalled ? 'Setup Required' :
           'Something Went Wrong'}
        </div>
        <div className="ai-error-msg">{error}</div>
        <div className="ai-error-actions">
          <button className="ai-run-btn" onClick={() => { setError(''); setAiState('idle') }}>
            Try Again
          </button>
          {(isAuthError || isNotInstalled) && (
            <button className="ai-link-btn" onClick={openSettings}>
              Open Settings
            </button>
          )}
          {!isAuthError && !isNotInstalled && (
            <button className="ai-link-btn" onClick={() => setAiState('idle')}>
              ← Back
            </button>
          )}
        </div>
        {isConnectionError && (
          <div className="ai-error-hint">
            Check your internet connection or verify the AI service is running.
          </div>
        )}
        {isTimeoutError && (
          <div className="ai-error-hint">
            The request took too long. Try selecting a smaller portion of text.
          </div>
        )}
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

      {/* Custom mode prompt input */}
      {aiMode === 'custom' && (
        <div className="custom-prompt-section">
          <div className="ai-section-label">Custom Prompt</div>
          <textarea
            className="custom-prompt-input"
            value={customPrompt}
            onChange={e => setCustomPrompt(e.target.value)}
            placeholder="Enter your instruction (e.g., 'expand this scene with more sensory detail') or use @ai in your text..."
            rows={3}
          />
          <div className="custom-prompt-hint">
            Tip: You can also write <code>@ai your instruction</code> directly in your text.
          </div>
        </div>
      )}

      {/* Selection indicator */}
      {selection && (
        <div className="ai-selection-indicator">
          <span className="selection-icon">✓</span>
          <span>Selection active ({selection.text.length} chars) — AI will target only selected text</span>
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
      ) : aiConfigured ? (
        <div className="ai-actions">
          <button className="ai-run-btn" onClick={handleRun}>
            Run {currentMode.label}
          </button>
          <button className="ai-link-btn" onClick={() => setShowImport(true)}>
            or compare with imported draft ›
          </button>
        </div>
      ) : (
        <AiSetupGuidance
          ccStatus={ccStatus}
          ccChecking={ccChecking}
          settings={settings}
          onOpenSettings={openSettings}
        />
      )}
    </>
  )
}

// ── AI Setup Guidance ────────────────────────────────────────────────────────

interface AiSetupGuidanceProps {
  ccStatus: types.ClaudeCodeStatus | null
  ccChecking: boolean
  settings: { ai_mode: string; ai_provider: string; ai_api_key: string; ai_local_endpoint: string }
  onOpenSettings: () => void
}

function AiSetupGuidance({ ccStatus, ccChecking, settings, onOpenSettings }: AiSetupGuidanceProps) {
  // Determine what's configured
  const ccInstalled = ccStatus?.installed
  const ccAuthenticated = ccStatus?.authenticated
  const hasApiKey = settings.ai_provider !== '' && settings.ai_api_key !== ''
  const hasLocalEndpoint = settings.ai_local_endpoint !== ''

  // If Claude Code mode is selected but not set up
  if (settings.ai_mode === 'claudecode') {
    if (ccChecking) {
      return (
        <div className="ai-setup-pane">
          <div className="ai-setup-checking">
            <div className="ai-spinner" />
            <span>Checking Claude Code...</span>
          </div>
        </div>
      )
    }

    if (!ccInstalled) {
      return (
        <div className="ai-setup-pane">
          <div className="ai-setup-icon">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <path d="M12 2v4m0 12v4M4.93 4.93l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"/>
            </svg>
          </div>
          <div className="ai-setup-title">Claude Code Not Installed</div>
          <p className="ai-setup-desc">
            Claude Code is a standalone AI coding assistant that powers the editing features.
          </p>
          <button className="ai-run-btn" onClick={onOpenSettings}>
            Set Up Claude Code
          </button>
          <p className="ai-setup-alt">
            Or <button className="ai-text-btn" onClick={onOpenSettings}>use an API key</button> instead
          </p>
        </div>
      )
    }

    if (!ccAuthenticated) {
      return (
        <div className="ai-setup-pane">
          <div className="ai-setup-icon installed">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <path d="M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2zm0 18a8 8 0 1 1 8-8 8 8 0 0 1-8 8z"/>
              <path d="M12 6v6l4 2"/>
            </svg>
          </div>
          <div className="ai-setup-title">Authentication Required</div>
          <p className="ai-setup-desc">
            Claude Code is installed but needs to be authenticated with your Anthropic account.
          </p>
          <button className="ai-run-btn" onClick={onOpenSettings}>
            Authenticate →
          </button>
        </div>
      )
    }
  }

  // If API mode is selected but no key
  if (settings.ai_mode === 'api' && !hasApiKey) {
    return (
      <div className="ai-setup-pane">
        <div className="ai-setup-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
        </div>
        <div className="ai-setup-title">API Key Required</div>
        <p className="ai-setup-desc">
          Add your API key to use {settings.ai_provider ? settings.ai_provider.charAt(0).toUpperCase() + settings.ai_provider.slice(1) : 'AI'} services.
        </p>
        <button className="ai-run-btn" onClick={onOpenSettings}>
          Add API Key
        </button>
        {ccInstalled && ccAuthenticated && (
          <p className="ai-setup-alt">
            Or <button className="ai-text-btn" onClick={onOpenSettings}>switch to Claude Code</button>
          </p>
        )}
      </div>
    )
  }

  // If Local mode is selected but no endpoint
  if (settings.ai_mode === 'local' && !hasLocalEndpoint) {
    return (
      <div className="ai-setup-pane">
        <div className="ai-setup-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
            <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
            <path d="M8 21h8m-4-4v4"/>
          </svg>
        </div>
        <div className="ai-setup-title">Local AI Not Configured</div>
        <p className="ai-setup-desc">
          Configure your local AI endpoint (e.g., Ollama, LM Studio) in Settings.
        </p>
        <button className="ai-run-btn" onClick={onOpenSettings}>
          Configure Endpoint
        </button>
      </div>
    )
  }

  // Generic fallback - nothing configured at all
  return (
    <div className="ai-setup-pane">
      <div className="ai-setup-icon">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
          <circle cx="12" cy="12" r="3"/>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
        </svg>
      </div>
      <div className="ai-setup-title">AI Features Ready</div>
      <p className="ai-setup-desc">
        Configure AI-assisted editing in Settings. Choose from Claude Code, cloud APIs, or local models.
      </p>
      <button className="ai-run-btn" onClick={onOpenSettings}>
        Open Settings
      </button>
    </div>
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

// ── Characters Section ───────────────────────────────────────────────────────

function CharactersSection() {
  const {
    book, addCharacter, updateCharacter, deleteCharacter, mergeCharacters,
    currentSection, currentIndex, setViewMode
  } = useBookStore()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [addingNew, setAddingNew] = useState(false)
  const [sortBy, setSortBy] = useState<'name' | 'mentions' | 'first'>('mentions')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [mergeMode, setMergeMode] = useState(false)
  const [showMergeDialog, setShowMergeDialog] = useState(false)

  if (!book) return <div className="tool-empty-state">Open a project to manage characters.</div>

  const allCharacters = book.story_bible?.characters ?? []

  // Get current chapter content for searching
  const getCurrentChapterContent = (): string => {
    if (currentSection === 'copyright') return book.copyright || ''
    const arr = currentSection === 'front_matter' ? book.front_matter
      : currentSection === 'body' ? book.body
      : book.back_matter
    return arr[currentIndex]?.content || ''
  }

  // Filter to characters appearing in current chapter by searching content
  // This respects aliases - if "Ruiz" is an alias, it will find mentions of "Ruiz" too
  const chapterContent = getCurrentChapterContent().toLowerCase()
  const characters = allCharacters.filter(char => {
    // Check if character name or any alias appears in chapter
    const namesToCheck = [char.name, ...(char.aliases || [])]
    return namesToCheck.some(name => {
      // Use word boundary check (simple version)
      const lowerName = name.toLowerCase()
      const idx = chapterContent.indexOf(lowerName)
      if (idx === -1) return false
      // Check it's a word boundary (not part of a longer word)
      const before = idx > 0 ? chapterContent[idx - 1] : ' '
      const after = idx + lowerName.length < chapterContent.length ? chapterContent[idx + lowerName.length] : ' '
      const isWordBoundary = !/[a-z]/.test(before) && !/[a-z]/.test(after)
      return isWordBoundary
    })
  })

  // Sort characters
  const sortedCharacters = [...characters].sort((a, b) => {
    if (sortBy === 'mentions') return (b.mention_count || 0) - (a.mention_count || 0)
    if (sortBy === 'first') return (a.first_chapter || 999) - (b.first_chapter || 999)
    return a.name.localeCompare(b.name)
  })

  const autoCount = characters.filter(c => c.is_auto_detected).length

  const toggleSelect = (id: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleMerge = (primaryId: string) => {
    const mergeIds = Array.from(selectedIds).filter(id => id !== primaryId)
    mergeCharacters(primaryId, mergeIds)
    setSelectedIds(new Set())
    setMergeMode(false)
    setShowMergeDialog(false)
  }

  const cancelMergeMode = () => {
    setMergeMode(false)
    setSelectedIds(new Set())
    setShowMergeDialog(false)
  }

  const selectedCharacters = characters.filter(c => selectedIds.has(c.id))

  return (
    <>
      {/* Merge dialog */}
      {showMergeDialog && selectedCharacters.length >= 2 && (
        <div className="merge-dialog">
          <div className="merge-dialog-header">Select Primary Name</div>
          <div className="merge-dialog-hint">Other names will become aliases of the selected character.</div>
          <div className="merge-options">
            {selectedCharacters.map(char => (
              <button key={char.id} className="merge-option" onClick={() => handleMerge(char.id)}>
                <span className="merge-option-name">{char.name}</span>
                {char.mention_count !== undefined && <span className="merge-option-count">({char.mention_count} mentions)</span>}
              </button>
            ))}
          </div>
          <button className="ai-link-btn" onClick={() => setShowMergeDialog(false)}>Cancel</button>
        </div>
      )}

      {/* Header with stats and actions */}
      <div className="characters-header">
        <div className="characters-stats">
          <span className="characters-count">{characters.length} in chapter</span>
          {autoCount > 0 && <span className="characters-auto-badge">{autoCount} auto</span>}
        </div>
        <div className="characters-actions">
          {mergeMode ? (
            <>
              <button
                className="tool-card-btn"
                onClick={() => setShowMergeDialog(true)}
                disabled={selectedIds.size < 2}
                title="Merge selected characters"
              >
                Merge ({selectedIds.size})
              </button>
              <button className="tool-card-btn secondary" onClick={cancelMergeMode}>Cancel</button>
            </>
          ) : (
            <>
              <button
                className="tool-card-btn"
                onClick={() => setViewMode('codex')}
                title="Open full character codex"
              >
                Codex
              </button>
              <button
                className="tool-card-btn secondary"
                onClick={() => setMergeMode(true)}
                disabled={characters.length < 2}
                title="Select characters to merge"
              >
                Merge
              </button>
            </>
          )}
        </div>
      </div>

      {/* Merge mode hint */}
      {mergeMode && (
        <div className="merge-mode-hint">
          Select 2+ characters that are the same person, then click Merge.
        </div>
      )}

      {/* Sort controls */}
      {characters.length > 0 && !mergeMode && (
        <div className="characters-sort">
          <span className="sort-label">Sort:</span>
          <button className={`sort-btn${sortBy === 'mentions' ? ' active' : ''}`} onClick={() => setSortBy('mentions')}>Mentions</button>
          <button className={`sort-btn${sortBy === 'first' ? ' active' : ''}`} onClick={() => setSortBy('first')}>First Appearance</button>
          <button className={`sort-btn${sortBy === 'name' ? ' active' : ''}`} onClick={() => setSortBy('name')}>Name</button>
        </div>
      )}

      {/* Character list */}
      {addingNew && (
        <CharacterForm char={null} onSave={c => { addCharacter(c); setAddingNew(false) }} onCancel={() => setAddingNew(false)} />
      )}
      {sortedCharacters.map(char =>
        editingId === char.id ? (
          <CharacterForm key={char.id} char={char} onSave={c => { updateCharacter(c); setEditingId(null) }} onCancel={() => setEditingId(null)} />
        ) : (
          <CharacterCard
            key={char.id}
            char={char}
            onEdit={() => setEditingId(char.id)}
            onDelete={() => deleteCharacter(char.id)}
            mergeMode={mergeMode}
            selected={selectedIds.has(char.id)}
            onToggleSelect={() => toggleSelect(char.id)}
          />
        )
      )}
      {!addingNew && !mergeMode && (
        <button className="tool-card-btn" style={{ marginTop: 4 }} onClick={() => setAddingNew(true)}>+ Add Character</button>
      )}

      {/* Empty state with link to codex */}
      {characters.length === 0 && !addingNew && (
        <div className="characters-empty">
          <p>No characters in this chapter.</p>
          <p className="hint">
            <button className="ai-link-btn" onClick={() => setViewMode('codex')}>Open Codex</button>
            {' '}to see all characters or re-index the book.
          </p>
        </div>
      )}
    </>
  )
}

function CharacterCard({ char, onEdit, onDelete, mergeMode, selected, onToggleSelect }: {
  char: Character
  onEdit: () => void
  onDelete: () => void
  mergeMode?: boolean
  selected?: boolean
  onToggleSelect?: () => void
}) {
  const { book, highlightedCharacterId, setHighlightedCharacter } = useBookStore()
  const [expanded, setExpanded] = useState(false)

  const isHighlighted = highlightedCharacterId === char.id

  // Get chapter name from index
  const getChapterName = (index: number) => {
    if (!book) return `Chapter ${index + 1}`
    const allChapters = [...(book.front_matter || []), ...(book.body || []), ...(book.back_matter || [])]
    return allChapters[index]?.title || `Chapter ${index + 1}`
  }

  const handleClick = () => {
    if (mergeMode && onToggleSelect) {
      onToggleSelect()
    } else {
      setExpanded(v => !v)
    }
  }

  const handleHighlightToggle = (e: React.MouseEvent) => {
    e.stopPropagation()
    setHighlightedCharacter(isHighlighted ? null : char.id)
  }

  return (
    <div
      className={`character-card${char.is_auto_detected ? ' auto-detected' : ''}${mergeMode ? ' merge-mode' : ''}${selected ? ' selected' : ''}${isHighlighted ? ' highlighting' : ''}`}
      onClick={handleClick}
    >
      <div className="character-card-header">
        {mergeMode && (
          <span className={`merge-checkbox${selected ? ' checked' : ''}`}>
            {selected ? '☑' : '☐'}
          </span>
        )}
        <span className="character-name">{char.name || 'Unnamed'}</span>
        <div className="character-badges">
          {char.is_auto_detected && <span className="character-auto-badge" title="Auto-detected">✨</span>}
          <span className={`character-role ${char.role}`}>{char.role}</span>
          {!mergeMode && (
            <button
              className={`character-highlight-btn${isHighlighted ? ' active' : ''}`}
              onClick={handleHighlightToggle}
              title={isHighlighted ? 'Stop highlighting in text' : 'Highlight in text'}
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="11" cy="11" r="8" />
                <path d="M21 21l-4.35-4.35" />
              </svg>
            </button>
          )}
        </div>
      </div>

      {/* Quick stats row */}
      {(char.mention_count !== undefined || char.first_chapter !== undefined) && (
        <div className="character-stats-row">
          {char.mention_count !== undefined && (
            <span className="character-stat" title="Total mentions">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z" />
              </svg>
              {char.mention_count}
            </span>
          )}
          {char.first_chapter !== undefined && (
            <span className="character-stat" title={`First appears in ${getChapterName(char.first_chapter)}`}>
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M4 19.5A2.5 2.5 0 016.5 17H20" />
                <path d="M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z" />
              </svg>
              Ch {char.first_chapter + 1}
            </span>
          )}
        </div>
      )}

      {/* Auto-detected attributes */}
      {char.attributes && Object.keys(char.attributes).length > 0 && (
        <div className="character-attributes">
          {char.attributes.eye_color && <span className="char-attr">Eyes: {char.attributes.eye_color}</span>}
          {char.attributes.hair_color && <span className="char-attr">Hair: {char.attributes.hair_color}</span>}
          {char.attributes.age && <span className="char-attr">Age: {char.attributes.age}</span>}
        </div>
      )}

      {char.description && <div className="character-desc-preview">{char.description}</div>}

      {expanded && (
        <div className="character-expanded" onClick={e => e.stopPropagation()}>
          {char.appearance && <div className="char-field"><span className="char-field-label">Appearance</span><span>{char.appearance}</span></div>}
          {char.personality && <div className="char-field"><span className="char-field-label">Personality</span><span>{char.personality}</span></div>}
          {char.motivation && <div className="char-field"><span className="char-field-label">Motivation</span><span>{char.motivation}</span></div>}
          {char.notes && <div className="char-field"><span className="char-field-label">Notes</span><span>{char.notes}</span></div>}

          {/* Chapter mentions breakdown */}
          {char.chapter_mentions && Object.keys(char.chapter_mentions).length > 0 && (
            <div className="char-field">
              <span className="char-field-label">Appears in</span>
              <div className="chapter-mentions">
                {Object.entries(char.chapter_mentions)
                  .sort(([a], [b]) => Number(a) - Number(b))
                  .map(([chIdx, count]) => (
                    <span key={chIdx} className="chapter-mention-badge">
                      {getChapterName(Number(chIdx))} ({count})
                    </span>
                  ))}
              </div>
            </div>
          )}

          {/* Aliases */}
          {char.aliases && char.aliases.length > 0 && (
            <div className="char-field">
              <span className="char-field-label">Also known as</span>
              <span>{char.aliases.join(', ')}</span>
            </div>
          )}

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

// ── Beat Sheet ───────────────────────────────────────────────────────────────

function BeatsSection() {
  const { book, addBeat, updateBeat, deleteBeat, currentSection, currentIndex } = useBookStore()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [addingNew, setAddingNew] = useState(false)

  if (!book) return <div className="tool-empty-state">Open a project first.</div>

  const beats = book.beat_sheet?.beats ?? []
  const globalChapterIndex = getGlobalChapterIndex(book, currentSection, currentIndex)
  const allChapters = [...book.front_matter, ...book.body, ...book.back_matter]

  // Group beats by chapter
  const beatsByChapter: Record<number, Beat[]> = {}
  beats.forEach(beat => {
    if (!beatsByChapter[beat.chapter_index]) beatsByChapter[beat.chapter_index] = []
    beatsByChapter[beat.chapter_index].push(beat)
  })

  const handleAddBeat = (chapterIdx: number, beatType: BeatType | string, description: string, notes: string) => {
    addBeat({
      id: genId(),
      chapter_index: chapterIdx,
      beat_type: beatType,
      description,
      notes: notes || undefined,
    })
    setAddingNew(false)
  }

  const handleUpdateBeat = (beat: Beat, updates: Partial<Beat>) => {
    updateBeat({ ...beat, ...updates })
    setEditingId(null)
  }

  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Beat Sheet</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>
        Track story beats per chapter. Based on Save the Cat! methodology.
      </p>

      {addingNew ? (
        <BeatForm
          chapters={allChapters}
          currentChapter={globalChapterIndex}
          onSubmit={handleAddBeat}
          onCancel={() => setAddingNew(false)}
        />
      ) : (
        <button className="bible-add-btn" onClick={() => setAddingNew(true)}>+ Add Beat</button>
      )}

      <div className="beats-list" style={{ marginTop: 12 }}>
        {Object.keys(beatsByChapter).sort((a, b) => Number(a) - Number(b)).map(chIdxStr => {
          const chIdx = Number(chIdxStr)
          const chapterBeats = beatsByChapter[chIdx]
          const chapter = allChapters[chIdx]
          return (
            <div key={chIdx} className="beat-chapter-group">
              <div className="beat-chapter-header">
                Ch. {chIdx + 1}: {chapter?.title || 'Untitled'}
              </div>
              {chapterBeats.map(beat => (
                editingId === beat.id ? (
                  <BeatForm
                    key={beat.id}
                    chapters={allChapters}
                    currentChapter={beat.chapter_index}
                    initialBeat={beat}
                    onSubmit={(chIdx, beatType, desc, notes) => handleUpdateBeat(beat, { chapter_index: chIdx, beat_type: beatType, description: desc, notes: notes || undefined })}
                    onCancel={() => setEditingId(null)}
                  />
                ) : (
                  <div key={beat.id} className="beat-card">
                    <div className="beat-card-header">
                      <span className="beat-type-badge">{BEAT_TYPES.find(t => t.value === beat.beat_type)?.label || beat.beat_type}</span>
                      <div className="beat-card-actions">
                        <button className="beat-edit-btn" onClick={() => setEditingId(beat.id)}>Edit</button>
                        <button className="beat-delete-btn" onClick={() => deleteBeat(beat.id)}>Delete</button>
                      </div>
                    </div>
                    <p className="beat-description">{beat.description}</p>
                    {beat.notes && <p className="beat-notes">{beat.notes}</p>}
                  </div>
                )
              ))}
            </div>
          )
        })}
        {beats.length === 0 && !addingNew && (
          <div className="tool-empty-state" style={{ marginTop: 12 }}>No beats defined yet.</div>
        )}
      </div>
    </div>
  )
}

function BeatForm({ chapters, currentChapter, initialBeat, onSubmit, onCancel }: {
  chapters: { title: string }[]
  currentChapter: number
  initialBeat?: Beat
  onSubmit: (chapterIdx: number, beatType: BeatType | string, description: string, notes: string) => void
  onCancel: () => void
}) {
  const [chapterIdx, setChapterIdx] = useState(initialBeat?.chapter_index ?? currentChapter)
  const [beatType, setBeatType] = useState<BeatType | string>(initialBeat?.beat_type ?? 'catalyst')
  const [description, setDescription] = useState(initialBeat?.description ?? '')
  const [notes, setNotes] = useState(initialBeat?.notes ?? '')

  return (
    <div className="beat-form">
      <div className="beat-form-row">
        <label>Chapter:</label>
        <select value={chapterIdx} onChange={e => setChapterIdx(Number(e.target.value))}>
          {chapters.map((ch, i) => (
            <option key={i} value={i}>Ch. {i + 1}: {ch.title || 'Untitled'}</option>
          ))}
        </select>
      </div>
      <div className="beat-form-row">
        <label>Beat Type:</label>
        <select value={beatType} onChange={e => setBeatType(e.target.value)}>
          {BEAT_TYPES.map(bt => (
            <option key={bt.value} value={bt.value}>{bt.label}</option>
          ))}
        </select>
      </div>
      <div className="beat-form-row">
        <label>Description:</label>
        <textarea
          value={description}
          onChange={e => setDescription(e.target.value)}
          placeholder="What happens in this beat..."
          rows={2}
        />
      </div>
      <div className="beat-form-row">
        <label>Notes:</label>
        <textarea
          value={notes}
          onChange={e => setNotes(e.target.value)}
          placeholder="Optional notes..."
          rows={1}
        />
      </div>
      <div className="beat-form-actions">
        <button className="bible-save-btn" onClick={() => onSubmit(chapterIdx, beatType, description, notes)}>
          {initialBeat ? 'Update' : 'Add'}
        </button>
        <button className="bible-cancel-btn" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}

// ── Foreshadowing Ledger ─────────────────────────────────────────────────────

function ForeshadowingSection() {
  const { book, addForeshadowingItem, updateForeshadowingItem, deleteForeshadowingItem } = useBookStore()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [addingNew, setAddingNew] = useState(false)

  if (!book) return <div className="tool-empty-state">Open a project first.</div>

  const items = book.foreshadowing?.items ?? []
  const allChapters = [...book.front_matter, ...book.body, ...book.back_matter]

  const handleAdd = (item: Omit<ForeshadowingItem, 'id'>) => {
    addForeshadowingItem({ ...item, id: genId() })
    setAddingNew(false)
  }

  const handleUpdate = (item: ForeshadowingItem) => {
    updateForeshadowingItem(item)
    setEditingId(null)
  }

  const getStatusColor = (status: ForeshadowingStatus | string) => {
    switch (status) {
      case 'planted': return '#fb923c'
      case 'active': return '#3b82f6'
      case 'resolved': return '#4ade80'
      default: return '#888888'
    }
  }

  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Foreshadowing Ledger</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>
        Track plant → reinforce → payoff chains across chapters.
      </p>

      {addingNew ? (
        <ForeshadowingForm
          chapters={allChapters}
          onSubmit={handleAdd}
          onCancel={() => setAddingNew(false)}
        />
      ) : (
        <button className="bible-add-btn" onClick={() => setAddingNew(true)}>+ Add Foreshadowing</button>
      )}

      <div className="foreshadow-list" style={{ marginTop: 12 }}>
        {items.map(item => (
          editingId === item.id ? (
            <ForeshadowingForm
              key={item.id}
              chapters={allChapters}
              initialItem={item}
              onSubmit={(i) => handleUpdate(i as ForeshadowingItem)}
              onCancel={() => setEditingId(null)}
            />
          ) : (
            <div key={item.id} className="foreshadow-card">
              <div className="foreshadow-card-header">
                <span className="foreshadow-name">{item.name}</span>
                <span className="foreshadow-status" style={{ background: `${getStatusColor(item.status)}22`, color: getStatusColor(item.status) }}>
                  {item.status}
                </span>
              </div>
              <p className="foreshadow-description">{item.description}</p>
              <div className="foreshadow-chapters">
                <span><strong>Plant:</strong> Ch. {item.plant_chapter + 1}</span>
                {(item.reinforce_chapters?.length ?? 0) > 0 && (
                  <span><strong>Reinforce:</strong> {item.reinforce_chapters?.map(c => `Ch. ${c + 1}`).join(', ')}</span>
                )}
                {item.payoff_chapter !== undefined && (
                  <span><strong>Payoff:</strong> Ch. {item.payoff_chapter + 1}</span>
                )}
              </div>
              {item.notes && <p className="foreshadow-notes">{item.notes}</p>}
              <div className="foreshadow-card-actions">
                <button className="beat-edit-btn" onClick={() => setEditingId(item.id)}>Edit</button>
                <button className="beat-delete-btn" onClick={() => deleteForeshadowingItem(item.id)}>Delete</button>
              </div>
            </div>
          )
        ))}
        {items.length === 0 && !addingNew && (
          <div className="tool-empty-state" style={{ marginTop: 12 }}>No foreshadowing items yet.</div>
        )}
      </div>
    </div>
  )
}

function ForeshadowingForm({ chapters, initialItem, onSubmit, onCancel }: {
  chapters: { title: string }[]
  initialItem?: ForeshadowingItem
  onSubmit: (item: ForeshadowingItem | Omit<ForeshadowingItem, 'id'>) => void
  onCancel: () => void
}) {
  const [name, setName] = useState(initialItem?.name ?? '')
  const [description, setDescription] = useState(initialItem?.description ?? '')
  const [plantChapter, setPlantChapter] = useState(initialItem?.plant_chapter ?? 0)
  const [reinforceChapters, setReinforceChapters] = useState(initialItem?.reinforce_chapters?.join(', ') ?? '')
  const [payoffChapter, setPayoffChapter] = useState<string>(initialItem?.payoff_chapter?.toString() ?? '')
  const [status, setStatus] = useState<ForeshadowingStatus | string>(initialItem?.status ?? 'planted')
  const [notes, setNotes] = useState(initialItem?.notes ?? '')

  const handleSubmit = () => {
    const reinforce = reinforceChapters
      .split(',')
      .map(s => parseInt(s.trim()) - 1)
      .filter(n => !isNaN(n) && n >= 0)
    const payoff = payoffChapter ? parseInt(payoffChapter) - 1 : undefined

    const item = {
      ...(initialItem ? { id: initialItem.id } : {}),
      name,
      description,
      plant_chapter: plantChapter,
      reinforce_chapters: reinforce,
      payoff_chapter: payoff !== undefined && payoff >= 0 ? payoff : undefined,
      status,
      notes: notes || undefined,
    }
    onSubmit(item as ForeshadowingItem)
  }

  return (
    <div className="foreshadow-form">
      <div className="beat-form-row">
        <label>Name:</label>
        <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="Short label..." />
      </div>
      <div className="beat-form-row">
        <label>Description:</label>
        <textarea value={description} onChange={e => setDescription(e.target.value)} placeholder="What is being foreshadowed..." rows={2} />
      </div>
      <div className="beat-form-row">
        <label>Plant Chapter:</label>
        <select value={plantChapter} onChange={e => setPlantChapter(Number(e.target.value))}>
          {chapters.map((ch, i) => (
            <option key={i} value={i}>Ch. {i + 1}: {ch.title || 'Untitled'}</option>
          ))}
        </select>
      </div>
      <div className="beat-form-row">
        <label>Reinforce (comma-sep):</label>
        <input type="text" value={reinforceChapters} onChange={e => setReinforceChapters(e.target.value)} placeholder="e.g. 5, 8, 12" />
      </div>
      <div className="beat-form-row">
        <label>Payoff Chapter:</label>
        <input type="text" value={payoffChapter} onChange={e => setPayoffChapter(e.target.value)} placeholder="e.g. 15 (leave empty if unresolved)" />
      </div>
      <div className="beat-form-row">
        <label>Status:</label>
        <select value={status} onChange={e => setStatus(e.target.value as ForeshadowingStatus)}>
          <option value="planted">Planted</option>
          <option value="active">Active</option>
          <option value="resolved">Resolved</option>
        </select>
      </div>
      <div className="beat-form-row">
        <label>Notes:</label>
        <textarea value={notes} onChange={e => setNotes(e.target.value)} placeholder="Optional notes..." rows={1} />
      </div>
      <div className="beat-form-actions">
        <button className="bible-save-btn" onClick={handleSubmit}>{initialItem ? 'Update' : 'Add'}</button>
        <button className="bible-cancel-btn" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}

// ── Knowledge Matrix ─────────────────────────────────────────────────────────

function KnowledgeSection() {
  const { book, addSecret, updateSecret, deleteSecret, setKnowledgeEntry } = useBookStore()
  const [addingSecret, setAddingSecret] = useState(false)
  const [editingSecretId, setEditingSecretId] = useState<string | null>(null)

  if (!book) return <div className="tool-empty-state">Open a project first.</div>

  const secrets = book.knowledge_matrix?.secrets ?? []
  const entries = book.knowledge_matrix?.entries ?? []
  const characters = book.story_bible?.characters ?? []

  // Build lookup: secretId -> characterId -> entry
  const entryLookup = useMemo(() => {
    const lookup: Record<string, Record<string, KnowledgeEntry>> = {}
    entries.forEach(e => {
      if (!lookup[e.secret_id]) lookup[e.secret_id] = {}
      lookup[e.secret_id][e.character_id] = e
    })
    return lookup
  }, [entries])

  const handleAddSecret = (name: string, description: string) => {
    addSecret({ id: genId(), name, description })
    setAddingSecret(false)
  }

  const handleUpdateSecret = (secret: SecretInfo) => {
    updateSecret(secret)
    setEditingSecretId(null)
  }

  const handleCellChange = (secretId: string, charId: string, field: 'learns' | 'suspected', value: string) => {
    const existing = entryLookup[secretId]?.[charId]
    const numVal = value ? parseInt(value) - 1 : undefined

    setKnowledgeEntry({
      secret_id: secretId,
      character_id: charId,
      learns_chapter: field === 'learns' ? (numVal !== undefined && numVal >= 0 ? numVal : undefined) : existing?.learns_chapter,
      suspected_chapter: field === 'suspected' ? (numVal !== undefined && numVal >= 0 ? numVal : undefined) : existing?.suspected_chapter,
    })
  }

  if (characters.length === 0) {
    return (
      <div>
        <div className="tool-label" style={{ marginBottom: 4 }}>Knowledge Matrix</div>
        <p className="settings-hint" style={{ marginBottom: 8 }}>
          Track which characters know which secrets, and when they learn them.
        </p>
        <div className="tool-empty-state" style={{ marginTop: 12 }}>
          Add characters first in the Characters tab.
        </div>
      </div>
    )
  }

  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Knowledge Matrix</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>
        Track which characters know which secrets. Enter chapter numbers.
      </p>

      {addingSecret ? (
        <SecretForm onSubmit={handleAddSecret} onCancel={() => setAddingSecret(false)} />
      ) : (
        <button className="bible-add-btn" onClick={() => setAddingSecret(true)}>+ Add Secret</button>
      )}

      {secrets.length > 0 && (
        <div className="knowledge-matrix" style={{ marginTop: 12 }}>
          <table>
            <thead>
              <tr>
                <th>Secret / Info</th>
                {characters.slice(0, 5).map(char => (
                  <th key={char.id} title={char.name}>{char.name.slice(0, 8)}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {secrets.map(secret => (
                <tr key={secret.id}>
                  <td className="knowledge-secret-cell">
                    {editingSecretId === secret.id ? (
                      <SecretForm
                        initialSecret={secret}
                        onSubmit={(name, desc) => handleUpdateSecret({ ...secret, name, description: desc })}
                        onCancel={() => setEditingSecretId(null)}
                        inline
                      />
                    ) : (
                      <div className="knowledge-secret-name">
                        <span title={secret.description}>{secret.name}</span>
                        <div className="knowledge-secret-actions">
                          <button onClick={() => setEditingSecretId(secret.id)}>Edit</button>
                          <button onClick={() => deleteSecret(secret.id)}>Del</button>
                        </div>
                      </div>
                    )}
                  </td>
                  {characters.slice(0, 5).map(char => {
                    const entry = entryLookup[secret.id]?.[char.id]
                    return (
                      <td key={char.id} className="knowledge-cell">
                        <input
                          type="text"
                          className="knowledge-cell-input"
                          placeholder="—"
                          value={entry?.learns_chapter !== undefined ? entry.learns_chapter + 1 : ''}
                          onChange={e => handleCellChange(secret.id, char.id, 'learns', e.target.value)}
                          title={`When ${char.name} learns this`}
                        />
                      </td>
                    )
                  })}
                </tr>
              ))}
            </tbody>
          </table>
          {characters.length > 5 && (
            <p className="settings-hint" style={{ marginTop: 8 }}>
              Showing first 5 characters. Full matrix available in Codex view.
            </p>
          )}
        </div>
      )}

      {secrets.length === 0 && !addingSecret && (
        <div className="tool-empty-state" style={{ marginTop: 12 }}>No secrets/info items yet.</div>
      )}
    </div>
  )
}

// ── Issues / Story Analysis ─────────────────────────────────────────────────

function IssuesSection() {
  const { book } = useBookStore()

  if (!book) {
    return <div className="tool-empty-state">Open a project to see analysis.</div>
  }

  return (
    <div className="issues-section">
      <div className="tool-empty-state">
        <p>Story analysis coming soon.</p>
        <p style={{ fontSize: '11px', opacity: 0.7, marginTop: '8px' }}>
          This panel will provide continuity checking and story structure analysis.
        </p>
      </div>
    </div>
  )
}

function SecretForm({ initialSecret, onSubmit, onCancel, inline }: {
  initialSecret?: SecretInfo
  onSubmit: (name: string, description: string) => void
  onCancel: () => void
  inline?: boolean
}) {
  const [name, setName] = useState(initialSecret?.name ?? '')
  const [description, setDescription] = useState(initialSecret?.description ?? '')

  if (inline) {
    return (
      <div className="secret-form-inline">
        <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="Name" />
        <button onClick={() => onSubmit(name, description)}>OK</button>
        <button onClick={onCancel}>X</button>
      </div>
    )
  }

  return (
    <div className="secret-form">
      <div className="beat-form-row">
        <label>Name:</label>
        <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="Secret/info name..." />
      </div>
      <div className="beat-form-row">
        <label>Description:</label>
        <textarea value={description} onChange={e => setDescription(e.target.value)} placeholder="What is this secret..." rows={2} />
      </div>
      <div className="beat-form-actions">
        <button className="bible-save-btn" onClick={() => onSubmit(name, description)}>{initialSecret ? 'Update' : 'Add'}</button>
        <button className="bible-cancel-btn" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}
