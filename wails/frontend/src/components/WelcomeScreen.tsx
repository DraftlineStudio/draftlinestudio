import { useCallback, useState, useEffect } from 'react'
import { useAppStore } from '../store/appStore'
import { types } from '../../wailsjs/go/models'
import ContextMenu, { ContextMenuItem } from './ContextMenu'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import { GetAppVersion } from '../../wailsjs/go/main/App'
import { avatarColor } from '../utils/accentColor'

type RecentProject = types.RecentProject

interface ContextMenuState {
  x: number
  y: number
  project: RecentProject
}

interface WelcomeScreenProps {
  onNewBook: () => void
  onNewUniverse: () => void
  onOpenFile: () => void
  onOpenRecent: (path: string) => void
}

// Studio Glow launch screen: the last-opened book sets the scene — its
// avatar color drives the titlebar strip (via the shared applyAccent
// machinery in App) and the soft glow behind the "Continue Writing" hero.
// Actions live in a compact left rail; recents exclude the hero book.
export default function WelcomeScreen({ onNewBook, onNewUniverse, onOpenFile, onOpenRecent }: WelcomeScreenProps) {
  const { openSettings, recentProjects, removeRecentProject, clearRecentProjects } = useAppStore()
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  const [appVersion, setAppVersion] = useState<string>('')

  useEffect(() => {
    GetAppVersion().then(setAppVersion).catch(() => setAppVersion('0.0.00000'))
  }, [])

  const hero = recentProjects[0] ?? null
  const rest = recentProjects.slice(1)
  const heroAccent = hero ? avatarColor(hero.name) : null

  const formatRelativeDate = useCallback((isoDate: string) => {
    const date = new Date(isoDate)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

    if (diffDays === 0) {
      const hours = date.getHours()
      const minutes = date.getMinutes()
      const ampm = hours >= 12 ? 'PM' : 'AM'
      const h = hours % 12 || 12
      const m = minutes.toString().padStart(2, '0')
      return `Today at ${h}:${m} ${ampm}`
    }
    if (diffDays === 1) return 'Yesterday'
    if (diffDays < 7) return `${diffDays} days ago`
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
  }, [])

  const formatWordCount = useCallback((words: number) => {
    if (words >= 1000) return `${Math.round(words / 1000)}k words`
    return `${words} words`
  }, [])

  const metaLine = useCallback((project: RecentProject) => {
    const parts: string[] = [project.type === 'universe' ? 'Universe' : 'Book']
    if (project.type === 'universe' && project.stats.books !== undefined) {
      parts.push(`${project.stats.books} books`)
    }
    parts.push(`${project.stats.chapters} chapters`, formatWordCount(project.stats.words))
    return parts.join(' · ')
  }, [formatWordCount])

  const handleContextMenu = useCallback((e: React.MouseEvent, project: RecentProject) => {
    e.preventDefault()
    e.stopPropagation()
    setContextMenu({ x: e.clientX, y: e.clientY, project })
  }, [])

  const getContextMenuItems = useCallback((project: RecentProject): ContextMenuItem[] => {
    const folderPath = project.path.substring(0, project.path.lastIndexOf('\\') || project.path.lastIndexOf('/'))
    return [
      {
        label: 'Open',
        icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z" /></svg>,
        onClick: () => onOpenRecent(project.path),
      },
      {
        label: 'Show in Folder',
        icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" /></svg>,
        onClick: () => BrowserOpenURL(`file://${folderPath}`),
      },
      {
        label: 'Remove from List',
        icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>,
        onClick: () => removeRecentProject(project.path),
        danger: true,
      },
    ]
  }, [onOpenRecent, removeRecentProject])

  const books = recentProjects.filter(p => p.type !== 'universe')
  const totalWords = recentProjects.reduce((sum, p) => sum + (p.stats.words || 0), 0)

  return (
    <div className="sgw-screen">
      <div className="sgw-content">
        {/* Left rail: brand + actions */}
        <div className="sgw-rail">
          <div className="sgw-rail-logo" role="img" aria-label="Draftline" />
          <div className="sgw-rail-tag">Author's Studio</div>
          <div className="sgw-rail-actions">
            <button className="sgw-rail-row" onClick={onNewBook}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><path d="M4 19.5A2.5 2.5 0 016.5 17H20" /><path d="M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z" /><line x1="12" y1="6" x2="12" y2="14" /><line x1="8" y1="10" x2="16" y2="10" /></svg>
              <span className="sgw-rail-label">New Book</span>
              <kbd className="sgw-kbd">Ctrl+N</kbd>
            </button>
            <button className="sgw-rail-row" onClick={onNewUniverse}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10" /><circle cx="12" cy="12" r="4" /><line x1="12" y1="2" x2="12" y2="8" /><line x1="12" y1="16" x2="12" y2="22" /><line x1="2" y1="12" x2="8" y2="12" /><line x1="16" y1="12" x2="22" y2="12" /></svg>
              <span className="sgw-rail-label">New Universe</span>
              <span className="sgw-badge">Storiverse</span>
            </button>
            <button className="sgw-rail-row" onClick={onOpenFile}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z" /></svg>
              <span className="sgw-rail-label">Open Book…</span>
              <kbd className="sgw-kbd">Ctrl+O</kbd>
            </button>
            <div className="sgw-rail-divider" />
            <button className="sgw-rail-row" onClick={() => openSettings()}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 11-2.83 2.83l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 11-4 0v-.09a1.65 1.65 0 00-1-1.51 1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 11-2.83-2.83l.06-.06a1.65 1.65 0 00.33-1.82 1.65 1.65 0 00-1.51-1H3a2 2 0 110-4h.09a1.65 1.65 0 001.51-1 1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 112.83-2.83l.06.06a1.65 1.65 0 001.82.33h.01a1.65 1.65 0 001-1.51V3a2 2 0 114 0v.09a1.65 1.65 0 001 1.51h.01a1.65 1.65 0 001.82-.33l.06-.06a2 2 0 112.83 2.83l-.06.06a1.65 1.65 0 00-.33 1.82v.01a1.65 1.65 0 001.51 1H21a2 2 0 110 4h-.09a1.65 1.65 0 00-1.51 1z" /></svg>
              <span className="sgw-rail-label">Settings</span>
            </button>
          </div>
          <div className="sgw-rail-spacer" />
          <div className="sgw-rail-version">v{appVersion}</div>
        </div>

        {/* Main pane */}
        <div className="sgw-main">
          {hero ? (
            <>
              <div className="sgw-glow" style={{ '--sgw-book-accent': heroAccent! } as React.CSSProperties} />
              <div className="sgw-main-inner">
                <div className="sgw-slabel">Continue Writing</div>
                <div
                  className="sgw-hero"
                  style={{ '--sgw-book-accent': heroAccent! } as React.CSSProperties}
                  onClick={() => onOpenRecent(hero.path)}
                  onContextMenu={(e) => handleContextMenu(e, hero)}
                >
                  <div className="sgw-cover" style={{ background: heroAccent! }}>
                    <span>{hero.name}</span>
                  </div>
                  <div className="sgw-hero-text">
                    <div className="sgw-hero-title">{hero.name}</div>
                    <div className="sgw-hero-meta">{metaLine(hero)}</div>
                    <div className="sgw-hero-date">Last opened: {formatRelativeDate(hero.lastOpened)}</div>
                  </div>
                  <button
                    className="sgw-open-btn"
                    onClick={(e) => { e.stopPropagation(); onOpenRecent(hero.path) }}
                  >
                    {hero.type === 'universe' ? 'Open Universe' : 'Open Book'}
                  </button>
                </div>
                {rest.length > 0 && (
                  <>
                    <div className="sgw-recent-head">
                      <div className="sgw-slabel">Recent</div>
                      <button className="sgw-clear" onClick={clearRecentProjects}>Clear All</button>
                    </div>
                    <div className="sgw-list">
                      {rest.map((project) => (
                        <div
                          key={project.path}
                          className="sgw-row"
                          onClick={() => onOpenRecent(project.path)}
                          onContextMenu={(e) => handleContextMenu(e, project)}
                        >
                          <span className="sgw-mini" style={{ background: avatarColor(project.name) }}>
                            {project.name.charAt(0).toUpperCase()}
                          </span>
                          <span className="sgw-row-text">
                            <span className="sgw-row-title">{project.name}</span>
                            <span className="sgw-row-meta">{metaLine(project)}</span>
                          </span>
                          <span className="sgw-row-date">{formatRelativeDate(project.lastOpened)}</span>
                        </div>
                      ))}
                    </div>
                  </>
                )}
              </div>
            </>
          ) : (
            /* First run / cleared recents: a quiet literary column */
            <>
              <div className="sgw-glow sgw-glow-brand" />
              <div className="sgw-empty">
                <div className="sgw-empty-mark" role="img" aria-label="" />
                <div className="sgw-empty-line">Every book begins with a blank page.</div>
                <div className="sgw-empty-break">⁂</div>
                <div className="sgw-empty-hint">Create a new book or open an existing file to get started.</div>
                <div className="sgw-empty-keys">
                  <kbd className="sgw-kbd">Ctrl+N</kbd> New book
                  <span className="sgw-empty-dot">·</span>
                  <kbd className="sgw-kbd">Ctrl+O</kbd> Open
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Welcome status bar */}
      <div className="sgw-status">
        <span>Draftline</span>
        {books.length > 0 && (
          <span>{books.length} {books.length === 1 ? 'book' : 'books'} · {formatWordCount(totalWords)}</span>
        )}
      </div>

      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          items={getContextMenuItems(contextMenu.project)}
          onClose={() => setContextMenu(null)}
        />
      )}
    </div>
  )
}
