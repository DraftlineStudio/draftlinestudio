import { useMemo } from 'react'
import { useBookStore } from '../store/bookStore'
import { analyzeText, getScoreColor, getScoreLabel } from '../services/aiDetection'

function countWords(html: string): number {
  const div = document.createElement('div')
  div.innerHTML = html
  const text = div.textContent || div.innerText || ''
  return text.trim().split(/\s+/).filter((w) => w.length > 0).length
}

function totalWords(book: ReturnType<typeof useBookStore.getState>['book']): number {
  if (!book) return 0
  const allContent = [
    book.copyright,
    ...book.front_matter.map((c) => c.content),
    ...book.body.map((c) => c.content),
    ...book.back_matter.map((c) => c.content),
  ]
  return allContent.reduce((sum, html) => sum + countWords(html || ''), 0)
}

function getCurrentContent(book: ReturnType<typeof useBookStore.getState>['book'], section: string, index: number): string {
  if (!book) return ''
  if (section === 'copyright') return book.copyright || ''
  if (section === 'front_matter') return book.front_matter[index]?.content || ''
  if (section === 'body') return book.body[index]?.content || ''
  if (section === 'back_matter') return book.back_matter[index]?.content || ''
  return ''
}

export default function StatusBar() {
  const { book, isDirty, statusMessage, currentSection, currentIndex } = useBookStore()
  const words = totalWords(book)
  const filePath = book?.file_path || null
  const fileName = filePath ? filePath.split(/[\\/]/).pop() : null

  // Get current chapter content for AI detection
  const currentContent = getCurrentContent(book, currentSection, currentIndex)

  // Memoize AI analysis to avoid recalculating on every render
  const aiResult = useMemo(() => {
    if (!currentContent || currentContent.length < 50) return null
    return analyzeText(currentContent)
  }, [currentContent])

  return (
    <div className="statusbar">
      <div className="statusbar-left">
        <span className={`statusbar-dot ${isDirty ? 'dirty' : ''}`} title={isDirty ? 'Unsaved changes' : 'Saved'} />
        <span className="statusbar-file">{fileName ? fileName : 'Unsaved'}</span>
        {statusMessage && <span>{statusMessage}</span>}
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
