import { useCallback, useState, useEffect } from 'react'
import { useAppStore } from '../store/appStore'
import { types } from '../../wailsjs/go/models'
import ContextMenu, { ContextMenuItem } from './ContextMenu'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import { GetAppVersion } from '../../wailsjs/go/main/App'

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

export default function WelcomeScreen({ onNewBook, onNewUniverse, onOpenFile, onOpenRecent }: WelcomeScreenProps) {
  const { settings, openSettings, recentProjects, removeRecentProject, clearRecentProjects } = useAppStore()
  const [hoveredProject, setHoveredProject] = useState<string | null>(null)
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  const [appVersion, setAppVersion] = useState<string>('')

  useEffect(() => {
    GetAppVersion().then(setAppVersion).catch(() => setAppVersion('0.0.00000'))
  }, [])

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

  return (
    <div className="welcome-screen">
      <div className="welcome-header">
        <div className="welcome-logo">
          <div className="welcome-logo-mark" role="img" aria-label="Draftline" />
        </div>
        <div className="welcome-version">v{appVersion}</div>
      </div>

      <div className="welcome-actions">
        <button className="welcome-action-tile" onClick={onNewBook}>
          <div className="tile-icon">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <path d="M4 19.5A2.5 2.5 0 016.5 17H20" />
              <path d="M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z" />
              <line x1="12" y1="6" x2="12" y2="14" />
              <line x1="8" y1="10" x2="16" y2="10" />
            </svg>
          </div>
          <span className="tile-label">New Book</span>
        </button>

        <button className="welcome-action-tile" onClick={onNewUniverse}>
          <div className="tile-icon">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <circle cx="12" cy="12" r="10" />
              <circle cx="12" cy="12" r="4" />
              <line x1="12" y1="2" x2="12" y2="8" />
              <line x1="12" y1="16" x2="12" y2="22" />
              <line x1="2" y1="12" x2="8" y2="12" />
              <line x1="16" y1="12" x2="22" y2="12" />
            </svg>
          </div>
          <span className="tile-label">New Universe</span>
          <span className="tile-badge">Storiverse</span>
        </button>

        <button className="welcome-action-tile" onClick={onOpenFile}>
          <div className="tile-icon">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z" />
            </svg>
          </div>
          <span className="tile-label">Open...</span>
        </button>
      </div>

      <div className="welcome-recent">
        <div className="recent-header">
          <h3>Recent Projects</h3>
          {recentProjects.length > 0 && (
            <button className="recent-clear" onClick={clearRecentProjects}>Clear All</button>
          )}
        </div>

        <div className="recent-list">
          {recentProjects.length === 0 ? (
            <div className="recent-empty">
              <p>No recent projects</p>
              <p className="recent-empty-hint">Create a new book or open an existing file to get started.</p>
            </div>
          ) : (
            recentProjects.map((project) => (
              <div
                key={project.path}
                className={`recent-item ${hoveredProject === project.path ? 'hovered' : ''}`}
                onClick={() => onOpenRecent(project.path)}
                onContextMenu={(e) => handleContextMenu(e, project)}
                onMouseEnter={() => setHoveredProject(project.path)}
                onMouseLeave={() => setHoveredProject(null)}
              >
                <div className="recent-item-icon">
                  {project.type === 'universe' ? (
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                      <circle cx="12" cy="12" r="10" />
                      <circle cx="12" cy="12" r="4" />
                    </svg>
                  ) : (
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                      <path d="M4 19.5A2.5 2.5 0 016.5 17H20" />
                      <path d="M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z" />
                    </svg>
                  )}
                </div>
                <div className="recent-item-info">
                  <div className="recent-item-name">{project.name}</div>
                  <div className="recent-item-meta">
                    <span className="recent-item-type">
                      {project.type === 'universe' ? 'Universe' : 'Book'}
                    </span>
                    <span className="recent-item-dot">·</span>
                    {project.type === 'universe' && project.stats.books !== undefined && (
                      <>
                        <span>{project.stats.books} books</span>
                        <span className="recent-item-dot">·</span>
                      </>
                    )}
                    <span>{project.stats.chapters} chapters</span>
                    <span className="recent-item-dot">·</span>
                    <span>{formatWordCount(project.stats.words)}</span>
                  </div>
                  <div className="recent-item-date">Last opened: {formatRelativeDate(project.lastOpened)}</div>
                </div>
                {hoveredProject === project.path && (
                  <button
                    className="recent-item-remove"
                    onClick={(e) => {
                      e.stopPropagation()
                      void removeRecentProject(project.path)
                    }}
                    title="Remove from list"
                  >
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <line x1="18" y1="6" x2="6" y2="18" />
                      <line x1="6" y1="6" x2="18" y2="18" />
                    </svg>
                  </button>
                )}
              </div>
            ))
          )}
        </div>
      </div>

      <div className="welcome-footer">
        <button className="welcome-footer-btn" onClick={openSettings}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="3" />
            <path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83" />
          </svg>
          <span>Settings</span>
        </button>

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
