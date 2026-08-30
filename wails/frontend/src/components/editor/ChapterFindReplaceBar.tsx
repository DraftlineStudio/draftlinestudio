import type { Editor } from '@tiptap/react'
import { useCallback, useEffect, useRef, useState } from 'react'
import {
  getChapterSearchState,
  replaceAllChapterMatches,
  replaceCurrentChapterMatch,
  updateChapterSearch,
} from '../../extensions/ChapterSearch'

interface Props {
  editor: Editor | null
}

type SearchMode = 'find' | 'replace' | null

export default function ChapterFindReplaceBar({ editor }: Props) {
  const [mode, setMode] = useState<SearchMode>(null)
  const [query, setQuery] = useState('')
  const [replacement, setReplacement] = useState('')
  const [caseSensitive, setCaseSensitive] = useState(false)
  const [wholeWord, setWholeWord] = useState(false)
  const [matchCount, setMatchCount] = useState(0)
  const [activeIndex, setActiveIndex] = useState(0)
  const findInputRef = useRef<HTMLInputElement>(null)
  const replaceInputRef = useRef<HTMLInputElement>(null)

  const refreshStatus = useCallback(() => {
    if (!editor) return
    const state = getChapterSearchState(editor.view)
    setMatchCount(state?.matches.length ?? 0)
    setActiveIndex(state?.activeIndex ?? 0)
  }, [editor])

  const scrollToCurrent = useCallback(() => {
    window.requestAnimationFrame(() => {
      document.querySelector('.chapter-search-match-current')?.scrollIntoView({
        block: 'center',
        behavior: 'smooth',
      })
    })
  }, [])

  const openSearch = useCallback((nextMode: Exclude<SearchMode, null>) => {
    if (!editor) return

    let initialQuery = query
    const { from, to } = editor.state.selection
    if (from !== to) {
      const selectedText = editor.state.doc.textBetween(from, to, ' ')
      if (selectedText && selectedText.length <= 200 && !selectedText.includes('\n')) {
        initialQuery = selectedText
        setQuery(selectedText)
      }
    }

    setMode(current => nextMode === 'replace' || current === null ? nextMode : current)
    updateChapterSearch(editor.view, {
      query: initialQuery,
      caseSensitive,
      wholeWord,
      activeIndex: 0,
    })
    refreshStatus()
    window.requestAnimationFrame(() => {
      findInputRef.current?.focus()
      findInputRef.current?.select()
    })
  }, [caseSensitive, editor, query, refreshStatus, wholeWord])

  const closeSearch = useCallback(() => {
    if (editor) updateChapterSearch(editor.view, { query: '', activeIndex: 0 })
    setMode(null)
    setMatchCount(0)
    setActiveIndex(0)
    editor?.commands.focus()
  }, [editor])

  const moveMatch = useCallback((direction: 1 | -1) => {
    if (!editor || matchCount === 0) return
    const nextIndex = (activeIndex + direction + matchCount) % matchCount
    updateChapterSearch(editor.view, { activeIndex: nextIndex })
    refreshStatus()
    scrollToCurrent()
  }, [activeIndex, editor, matchCount, refreshStatus, scrollToCurrent])

  const replaceCurrent = useCallback(() => {
    if (!editor || !replaceCurrentChapterMatch(editor.view, replacement)) return
    refreshStatus()
    scrollToCurrent()
  }, [editor, refreshStatus, replacement, scrollToCurrent])

  const replaceAll = useCallback(() => {
    if (!editor) return
    replaceAllChapterMatches(editor.view, replacement)
    refreshStatus()
  }, [editor, refreshStatus, replacement])

  useEffect(() => {
    const handleShortcut = (event: KeyboardEvent) => {
      if (!(event.ctrlKey || event.metaKey)) return
      const key = event.key.toLocaleLowerCase()
      if (key !== 'f' && key !== 'h') return
      event.preventDefault()
      openSearch(key === 'h' ? 'replace' : 'find')
    }
    window.addEventListener('keydown', handleShortcut)
    return () => window.removeEventListener('keydown', handleShortcut)
  }, [openSearch])

  useEffect(() => {
    if (!editor || mode === null) return
    updateChapterSearch(editor.view, { query, caseSensitive, wholeWord, activeIndex: 0 })
    refreshStatus()
  }, [caseSensitive, editor, mode, query, refreshStatus, wholeWord])

  useEffect(() => {
    if (!editor || mode === null) return
    const handleTransaction = () => refreshStatus()
    editor.on('transaction', handleTransaction)
    return () => {
      editor.off('transaction', handleTransaction)
    }
  }, [editor, mode, refreshStatus])

  if (mode === null) return null

  return (
    <div className={`chapter-find-bar ${mode === 'replace' ? 'replace-mode' : ''}`} role="search" aria-label="Find in chapter">
      <div className="chapter-find-row">
        <span className="chapter-find-scope">Chapter</span>
        <div className="chapter-find-input-wrap">
          <input
            ref={findInputRef}
            className="chapter-find-input"
            value={query}
            onChange={event => setQuery(event.target.value)}
            onKeyDown={event => {
              if (event.key === 'Escape') closeSearch()
              if (event.key === 'Enter') {
                event.preventDefault()
                moveMatch(event.shiftKey ? -1 : 1)
              }
            }}
            placeholder="Find in current chapter"
            aria-label="Find text"
          />
          <span className="chapter-find-count">
            {query ? (matchCount ? `${activeIndex + 1} / ${matchCount}` : 'No matches') : ''}
          </span>
        </div>
        <button
          className={`chapter-find-option ${caseSensitive ? 'active' : ''}`}
          onClick={() => setCaseSensitive(value => !value)}
          title="Match case"
          aria-pressed={caseSensitive}
        >
          Aa
        </button>
        <button
          className={`chapter-find-option ${wholeWord ? 'active' : ''}`}
          onClick={() => setWholeWord(value => !value)}
          title="Whole words only"
          aria-pressed={wholeWord}
        >
          W
        </button>
        <button className="chapter-find-icon" onClick={() => moveMatch(-1)} disabled={!matchCount} title="Previous match (Shift+Enter)" aria-label="Previous match">
          ↑
        </button>
        <button className="chapter-find-icon" onClick={() => moveMatch(1)} disabled={!matchCount} title="Next match (Enter)" aria-label="Next match">
          ↓
        </button>
        {mode === 'find' && (
          <button className="chapter-find-text-button" onClick={() => { setMode('replace'); window.requestAnimationFrame(() => replaceInputRef.current?.focus()) }}>
            Replace
          </button>
        )}
        <button className="chapter-find-icon close" onClick={closeSearch} title="Close (Esc)" aria-label="Close find">
          ×
        </button>
      </div>
      {mode === 'replace' && (
        <div className="chapter-find-row chapter-replace-row">
          <span className="chapter-find-scope" aria-hidden="true" />
          <input
            ref={replaceInputRef}
            className="chapter-find-input chapter-replace-input"
            value={replacement}
            onChange={event => setReplacement(event.target.value)}
            onKeyDown={event => {
              if (event.key === 'Escape') closeSearch()
              if (event.key === 'Enter') {
                event.preventDefault()
                replaceCurrent()
              }
            }}
            placeholder="Replace with"
            aria-label="Replacement text"
          />
          <button className="chapter-find-text-button" onClick={replaceCurrent} disabled={!matchCount}>Replace</button>
          <button className="chapter-find-text-button" onClick={replaceAll} disabled={!matchCount}>Replace All</button>
        </div>
      )}
    </div>
  )
}
