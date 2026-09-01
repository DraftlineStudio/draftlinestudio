import { useState, useEffect, useRef, useMemo } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../store/bookStore'
import { useAppStore } from '../store/appStore'
import { analyzeText, getScoreColor, getScoreLabel, type AIDetectionResult } from '../services/aiDetection'
import { countBookWords, getCurrentContent } from '../utils/textUtils'
import { useAnalysisStore } from '../store/analysisStore'

export default function StatusBar() {
  const { book, isDirty, isAutoSaving, currentSection, currentIndex } = useBookStore(useShallow(s => ({
    book: s.book,
    isDirty: s.isDirty,
    isAutoSaving: s.isAutoSaving,
    currentSection: s.currentSection,
    currentIndex: s.currentIndex,
  })))
  const statusMessage = useAppStore(s => s.statusMessage)
  // Whole-book word count is an HTML re-parse of every chapter; memoize it so it
  // only recomputes when the book content actually changes — not on every
  // isDirty / statusMessage / isAutoSaving toggle re-render.
  const words = useMemo(() => (book ? countBookWords(book) : 0), [book])
  const filePath = book?.file_path || null
  const fileName = filePath ? filePath.split(/[\\/]/).pop() : null
  const analysis = useAnalysisStore(useShallow(s => ({
    state: s.state,
    progress: s.progress,
    message: s.message,
    modules: s.modules,
    error: s.error,
    run: s.run,
  })))

  // Get current chapter content for AI detection
  const currentContent = getCurrentContent(book, currentSection, currentIndex)

  // Debounced AI analysis - only run 2 seconds after typing stops
  const [aiResult, setAiResult] = useState<AIDetectionResult | null>(null)
  const analysisTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    // Clear any pending analysis
    if (analysisTimer.current) clearTimeout(analysisTimer.current)

    // Don't analyze if content is too short
    if (!currentContent || currentContent.length < 100) {
      setAiResult(null)
      return
    }

    // Debounce: wait 2 seconds after typing stops
    analysisTimer.current = setTimeout(() => {
      setAiResult(analyzeText(currentContent))
    }, 2000)

    return () => {
      if (analysisTimer.current) clearTimeout(analysisTimer.current)
    }
  }, [currentContent])

  return (
    <div className="statusbar">
      <div className="statusbar-left">
        {isAutoSaving ? (
          <span className="statusbar-autosave" title="Auto-saving...">
            <svg className="autosave-spinner" width="12" height="12" viewBox="0 0 12 12">
              <circle cx="6" cy="6" r="5" fill="none" stroke="currentColor" strokeWidth="1.5" strokeDasharray="20" strokeLinecap="round">
                <animateTransform attributeName="transform" type="rotate" from="0 6 6" to="360 6 6" dur="0.8s" repeatCount="indefinite" />
              </circle>
            </svg>
          </span>
        ) : (
          <span className={`statusbar-dot ${isDirty ? 'dirty' : ''}`} title={isDirty ? 'Unsaved changes' : 'Saved'} />
        )}
        <span className="statusbar-file">{fileName ? fileName : 'Unsaved'}</span>
        {statusMessage && <span className="statusbar-message">{statusMessage}</span>}
      </div>
      {book && analysis.state === 'running' && (
        <div className="statusbar-analysis-progress" title={analysis.message}>
          <span>{analysis.message}</span>
          <div className="statusbar-analysis-track" aria-label={`${analysis.progress}% complete`}>
            <div style={{ width: `${analysis.progress}%` }} />
          </div>
          <span className="statusbar-analysis-percent">{Math.round(analysis.progress)}%</span>
        </div>
      )}
      <div className="statusbar-right">
        {book && analysis.state !== 'idle' && (
          <button
            className="statusbar-analysis-modules"
            onClick={() => void analysis.run()}
            disabled={analysis.state === 'running'}
            title={analysis.state === 'error' ? analysis.error : analysis.state === 'stale' ? 'Analysis is out of date; click to run now' : 'Story analysis is current'}
          >
            {(['characters', 'story', 'pacing'] as const).map(module => (
              <span className="statusbar-analysis-module" key={module}>
                <i className={`analysis-state-dot ${analysis.modules[module]}`} />
                {module === 'characters' ? 'Characters' : module === 'story' ? 'Story' : 'Pacing'}
              </span>
            ))}
          </button>
        )}
        {aiResult && (
          <div
            className="statusbar-ai-score"
            title={`AI Detection: ${aiResult.score}% - ${getScoreLabel(aiResult.score)} (${aiResult.confidence} confidence)`}
          >
            <span className="ai-score-label">AI</span>
            <div className="ai-score-bar">
              <div
                className="ai-score-fill"
                style={{
                  width: `${aiResult.score}%`,
                  background: getScoreColor(aiResult.score),
                }}
              />
            </div>
            <span className="ai-score-value" style={{ color: getScoreColor(aiResult.score) }}>
              {aiResult.score}%
            </span>
          </div>
        )}
        <span className="statusbar-words">{words.toLocaleString()} words</span>
      </div>
    </div>
  )
}
