import { useState, useEffect, useRef } from 'react'
import { useBookStore } from '../store/bookStore'
import { analyzeText, getScoreColor, getScoreLabel, type AIDetectionResult } from '../services/aiDetection'
import { countBookWords, getCurrentContent } from '../utils/textUtils'

export default function StatusBar() {
  const { book, isDirty, isAutoSaving, statusMessage, currentSection, currentIndex } = useBookStore()
  const words = book ? countBookWords(book) : 0
  const filePath = book?.file_path || null
  const fileName = filePath ? filePath.split(/[\\/]/).pop() : null

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
      <div className="statusbar-right">
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
