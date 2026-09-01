import { useState, useRef, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { useBookStore } from '../store/bookStore'
import { useAppStore } from '../store/appStore'
import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime'
import { avatarColor, hexToRgba } from '../utils/accentColor'

interface TitleBarProps {
  minimal?: boolean
}

export default function TitleBar({ minimal = false }: TitleBarProps) {
  const { book, currentSection, newBook, openBook, saveBook, saveBookAs, closeProject, setViewMode } = useBookStore()
  const { settings, openSettings, openMetadataDialog, openExportWizard, openChapterHistory, openStorySearch } = useAppStore()
  const [dropOpen, setDropOpen] = useState(false)
  const [dropPos, setDropPos] = useState({ top: 0, left: 0 })
  const btnRef = useRef<HTMLButtonElement>(null)
  const dropRef = useRef<HTMLDivElement>(null)
  const title = book?.metadata.title || 'Untitled'
  const color = avatarColor(title)

  function openDrop() {
    if (btnRef.current) {
      const r = btnRef.current.getBoundingClientRect()
      setDropPos({ top: r.bottom + 6, left: r.left })
    }
    setDropOpen(v => !v)
  }

  // Close on outside click or Escape
  useEffect(() => {
    if (!dropOpen) return
    function onDown(e: MouseEvent) {
      const t = e.target as Node
      if (btnRef.current?.contains(t)) return
      if (dropRef.current?.contains(t)) return
      setDropOpen(false)
    }
    function onKey(e: KeyboardEvent) { if (e.key === 'Escape') setDropOpen(false) }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [dropOpen])

  function run(fn: () => void) { setDropOpen(false); fn() }
  function showStorySearch() { setViewMode('editor'); openStorySearch() }

  const dropdown = dropOpen && createPortal(
    <div
      ref={dropRef}
      className="titlebar-dropdown"
      style={{ top: dropPos.top, left: dropPos.left } as React.CSSProperties}
    >
      <button className="titlebar-dropdown-item" onClick={() => run(newBook)}>
        <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.3">
          <rect x="1" y="1" width="8" height="11" rx="1.2" />
          <line x1="9" y1="3.5" x2="12" y2="3.5" /><line x1="12" y1="3.5" x2="12" y2="12" />
          <line x1="12" y1="12" x2="5.5" y2="12" /><line x1="5.5" y1="12" x2="5.5" y2="9" />
        </svg>
        <span>New Book</span><kbd>Ctrl+N</kbd>
      </button>
      <button className="titlebar-dropdown-item" onClick={() => run(openBook)}>
        <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.3">
          <path d="M1 4.5h3.5l1.5 2H12v5H1z" /><path d="M1 4.5V2.5h2.5" />
        </svg>
        <span>Open Book…</span><kbd>Ctrl+O</kbd>
      </button>

      <div className="titlebar-dropdown-sep" />

      <button className="titlebar-dropdown-item" onClick={() => run(saveBook)} disabled={!book}>
        <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.3">
          <rect x="1" y="1" width="11" height="11" rx="1.2" />
          <rect x="2.5" y="1" width="5.5" height="4.5" /><rect x="2" y="7.5" width="9" height="4" />
          <line x1="7" y1="2" x2="7" y2="5" />
        </svg>
        <span>Save</span><kbd>Ctrl+S</kbd>
      </button>
      <button className="titlebar-dropdown-item" onClick={() => run(saveBookAs)} disabled={!book}>
        <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.3">
          <rect x="1" y="1" width="11" height="11" rx="1.2" />
          <rect x="2.5" y="1" width="5.5" height="4.5" /><rect x="2" y="7.5" width="9" height="4" />
          <line x1="7" y1="2" x2="7" y2="5" />
          <line x1="9.5" y1="9" x2="11.5" y2="9" /><line x1="10.5" y1="8" x2="10.5" y2="10" />
        </svg>
        <span>Save As…</span><kbd>Ctrl+Shift+S</kbd>
      </button>

      <button className="titlebar-dropdown-item" onClick={() => run(openChapterHistory)} disabled={!book || currentSection === 'copyright'}>
        <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.3">
          <path d="M2.2 4.2A5 5 0 1 1 1.5 7"/><path d="M1 2v3.5h3.5"/><path d="M6.5 3.5V7l2.2 1.3"/>
        </svg>
        <span>Chapter History…</span>
      </button>

      <button className="titlebar-dropdown-item" onClick={() => run(showStorySearch)} disabled={!book}>
        <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.3">
          <circle cx="5.5" cy="5.5" r="3.8" /><path d="M8.3 8.3 12 12" />
        </svg>
        <span>Search Story…</span><kbd>Ctrl+Shift+F</kbd>
      </button>

      <div className="titlebar-dropdown-sep" />

      <button className="titlebar-dropdown-item" onClick={() => run(openMetadataDialog)} disabled={!book}>
        <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.3">
          <circle cx="6.5" cy="6.5" r="5.5" />
          <line x1="6.5" y1="5.5" x2="6.5" y2="10" />
          <circle cx="6.5" cy="3.5" r="0.6" fill="currentColor" stroke="none" />
        </svg>
        <span>Book Info…</span>
      </button>

      <button className="titlebar-dropdown-item" onClick={() => run(openExportWizard)} disabled={!book}>
        <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.3">
          <path d="M2 9.5v2h9v-2" />
          <path d="M6.5 1v7" />
          <path d="M3.5 5l3 3 3-3" />
        </svg>
        <span>Export…</span>
      </button>

      <div className="titlebar-dropdown-sep" />

      <button className="titlebar-dropdown-item" onClick={() => run(closeProject)} disabled={!book}>
        <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.3">
          <rect x="1" y="1" width="8" height="11" rx="1.2" />
          <path d="M6 5l3 3M6 8l3-3" strokeLinecap="round" />
        </svg>
        <span>Close Project</span>
      </button>
    </div>,
    document.body
  )

  // Minimal mode for welcome screen - just branding and window controls
  if (minimal) {
    return (
      <div className="titlebar titlebar-minimal">
        <div className="titlebar-brand">
          <div className="titlebar-logo" role="img" aria-label="Draftline" />
        </div>

        <div className="titlebar-spacer" />

        <div className="titlebar-actions">
          <button className="titlebar-winbtn" onClick={WindowMinimise} title="Minimize">
            <svg width="10" height="1" viewBox="0 0 10 1"><line x1="0" y1="0.5" x2="10" y2="0.5" stroke="currentColor" strokeWidth="1.5"/></svg>
          </button>
          <button className="titlebar-winbtn" onClick={WindowToggleMaximise} title="Maximize">
            <svg width="10" height="10" viewBox="0 0 10 10" fill="none"><rect x="0.75" y="0.75" width="8.5" height="8.5" stroke="currentColor" strokeWidth="1.5"/></svg>
          </button>
          <button className="titlebar-winbtn close" onClick={Quit} title="Close">
            <svg width="10" height="10" viewBox="0 0 10 10" fill="none"><line x1="1" y1="1" x2="9" y2="9" stroke="currentColor" strokeWidth="1.5"/><line x1="9" y1="1" x2="1" y2="9" stroke="currentColor" strokeWidth="1.5"/></svg>
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="titlebar">
      <div className="titlebar-brand">
        <div className="titlebar-logo" role="img" aria-label="Draftline" />
      </div>

      <div className="titlebar-sep" />

      {/* Project switcher — glow stays inside titlebar via overflow:hidden */}
      <div
        className="titlebar-project"
        style={{ '--avatar-glow': hexToRgba(color, 0.2) } as React.CSSProperties}
      >
        <button
          ref={btnRef}
          className={`titlebar-project-btn ${dropOpen ? 'open' : ''}`}
          onClick={openDrop}
          title="Switch book"
          style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
        >
          <span className="titlebar-project-avatar" style={{ background: color }}>{title.charAt(0).toUpperCase()}</span>
          <span className="titlebar-project-name">{title}</span>
          <svg className="titlebar-project-chevron" width="10" height="6" viewBox="0 0 10 6">
            <path d="M1 1l4 4 4-4" stroke="currentColor" strokeWidth="1.5" fill="none" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        </button>
      </div>

      {dropdown}

      <div className="titlebar-spacer" />

      <div className="titlebar-actions">
        <button
          className="titlebar-author-btn"
          onClick={openSettings}
          title="App Settings"
          style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
        >
          {settings.default_author
            ? <span className="titlebar-author-name">{settings.default_author}</span>
            : <svg width="13" height="13" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <circle cx="7" cy="7" r="5.5"/>
                <circle cx="7" cy="5.5" r="1.8"/>
                <path d="M3 11.5c0-2.2 1.8-3.5 4-3.5s4 1.3 4 3.5" strokeLinecap="round"/>
              </svg>
          }
          <svg className="titlebar-author-gear" width="10" height="10" viewBox="0 0 14 14" fill="currentColor">
            <path d="M7 9a2 2 0 1 0 0-4 2 2 0 0 0 0 4zm4.2-1.1.9-.5a5 5 0 0 0-.5-1.2l-1 .1a3 3 0 0 0-.7-.7l.1-1a5 5 0 0 0-1.2-.5l-.5.9a3 3 0 0 0-1 0l-.5-.9a5 5 0 0 0-1.2.5l.1 1a3 3 0 0 0-.7.7l-1-.1a5 5 0 0 0-.5 1.2l.9.5a3 3 0 0 0 0 1l-.9.5a5 5 0 0 0 .5 1.2l1-.1c.2.3.4.5.7.7l-.1 1a5 5 0 0 0 1.2.5l.5-.9a3 3 0 0 0 1 0l.5.9a5 5 0 0 0 1.2-.5l-.1-1c.3-.2.5-.4.7-.7l1 .1a5 5 0 0 0 .5-1.2l-.9-.5a3 3 0 0 0 0-1z"/>
          </svg>
        </button>
        <button className="titlebar-winbtn" onClick={WindowMinimise} title="Minimize">
          <svg width="10" height="1" viewBox="0 0 10 1"><line x1="0" y1="0.5" x2="10" y2="0.5" stroke="currentColor" strokeWidth="1.5"/></svg>
        </button>
        <button className="titlebar-winbtn" onClick={WindowToggleMaximise} title="Maximize">
          <svg width="10" height="10" viewBox="0 0 10 10" fill="none"><rect x="0.75" y="0.75" width="8.5" height="8.5" stroke="currentColor" strokeWidth="1.5"/></svg>
        </button>
        <button className="titlebar-winbtn close" onClick={Quit} title="Close">
          <svg width="10" height="10" viewBox="0 0 10 10" fill="none"><line x1="1" y1="1" x2="9" y2="9" stroke="currentColor" strokeWidth="1.5"/><line x1="9" y1="1" x2="1" y2="9" stroke="currentColor" strokeWidth="1.5"/></svg>
        </button>
      </div>
    </div>
  )
}
