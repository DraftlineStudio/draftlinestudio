import { useState, useRef, useEffect } from 'react'
import { useBookStore } from '../../store/bookStore'
import { CancelRewrite, GenerateInlineContent } from '../../../wailsjs/go/main/App'

interface Props {
  onInsert: (html: string) => void
  onCancel: () => void
  beforeContext: string
  afterContext: string
}

export default function InlinePrompt({ onInsert, onCancel, beforeContext, afterContext }: Props) {
  const { book, currentSection, currentIndex, closeInlinePrompt } = useBookStore()
  const [prompt, setPrompt] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const inputRef = useRef<HTMLTextAreaElement>(null)

  // Focus input on mount
  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const handleCancel = () => {
    if (loading) {
      CancelRewrite().catch(() => {})
    }
    onCancel()
    closeInlinePrompt()
  }

  // Handle escape key
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        handleCancel()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [loading, onCancel, closeInlinePrompt])

  const getChapterTitle = () => {
    if (!book) return ''
    if (currentSection === 'copyright') return 'Copyright'
    const arr = currentSection === 'front_matter' ? book.front_matter
      : currentSection === 'body' ? book.body
      : book.back_matter
    return arr[currentIndex]?.title || ''
  }

  const getCharacterNames = () => {
    if (!book?.story_bible?.characters) return []
    return book.story_bible.characters.map(c => c.name)
  }

  const handleSubmit = async () => {
    if (!prompt.trim()) return

    setLoading(true)
    setError('')

    try {
      const result = await GenerateInlineContent({
        instruction: prompt,
        before_context: beforeContext,
        after_context: afterContext,
        characters: getCharacterNames(),
        chapter_title: getChapterTitle(),
      })

      if (result.error) {
        setError(result.error)
        setLoading(false)
      } else {
        onInsert(result.result)
        closeInlinePrompt()
      }
    } catch (e) {
      setError(String(e))
      setLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault()
      handleSubmit()
    }
  }

  return (
    <div className="inline-prompt-container">
      <div className="inline-prompt-box">
        <div className="inline-prompt-header">
          <span className="inline-prompt-icon">✨</span>
          <span className="inline-prompt-label">Generate content</span>
          {loading && <span className="inline-prompt-spinner" />}
        </div>

        <textarea
          ref={inputRef}
          className="inline-prompt-input"
          value={prompt}
          onChange={e => setPrompt(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Describe what to write... (Ctrl+Enter to generate)"
          rows={2}
          disabled={loading}
        />

        {error && <div className="inline-prompt-error">{error}</div>}

        <div className="inline-prompt-footer">
          <span className="inline-prompt-hint">
            Ctrl+Enter to generate • Esc to cancel
          </span>
          <div className="inline-prompt-actions">
            <button
              className="inline-prompt-btn cancel"
              onClick={handleCancel}
            >
              Cancel
            </button>
            <button
              className="inline-prompt-btn generate"
              onClick={handleSubmit}
              disabled={loading || !prompt.trim()}
            >
              {loading ? 'Generating...' : 'Generate'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
