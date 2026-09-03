// ToolsPanel - Main sidebar with section routing

import { useState, useEffect, useRef } from 'react'
import { useAppStore, type AppSettings } from '../store/appStore'
import { useReadAloudStore } from '../store/readAloudStore'

// Extracted types, constants, and components
import type { GlyphSection } from './tools/types'
import { SECTION_CONFIG } from './tools/constants'
import GlyphIcon from './tools/GlyphIcon'
import DashboardTab from './tools/Dashboard'
import AiStudioTab from './tools/AIStudio'
import CharacterQuickRef from './tools/CharacterQuickRef'
import SignalsPanel from './tools/Analysis/SignalsPanel'
import ProsePanel from './tools/Analysis/ProsePanel'
import PacingPanel from './tools/Analysis/PacingPanel'
import ChaptersPanel from './tools/Analysis/ChaptersPanel'
import ReviewPanel from './tools/Analysis/ReviewPanel'
import AIDetectPanel from './tools/Analysis/AIDetectPanel'
import { OPEN_TOOLS_SECTION_EVENT, useReviewCount } from './tools/Analysis/shared'

function isSectionEnabled(section: Exclude<GlyphSection, null>, settings: AppSettings): boolean {
  if (section === 'ai') return settings.ai_enabled
  if (section === 'characters') return settings.cast_enabled
  if (section === 'signals' || section === 'prose' || section === 'pacing' || section === 'chapters' || section === 'review') {
    return settings.analysis_enabled
  }
  return true
}

// Read Aloud lives at the very bottom of the rail, above the bottom-bar
// space — it is a player, not a sidebar pane, so it sits apart from the
// section glyphs. Not configured yet → deep-links to its settings section
// (the same pattern as the AI Studio CLI setup); configured → toggles the
// docked player bar.
function ReadAloudGlyph() {
  const openSettings = useAppStore(s => s.openSettings)
  const modelState = useReadAloudStore(s => s.modelState)
  const playerVisible = useReadAloudStore(s => s.playerVisible)
  const setPlayerVisible = useReadAloudStore(s => s.setPlayerVisible)
  const refreshModelStatus = useReadAloudStore(s => s.refreshModelStatus)

  useEffect(() => {
    void refreshModelStatus()
  }, [refreshModelStatus])

  const ready = modelState === 'ready'
  const handleClick = () => {
    if (!ready) {
      openSettings('readaloud')
      return
    }
    setPlayerVisible(!playerVisible)
  }

  return (
    <button
      className={`glyph-btn glyph-btn--readaloud${playerVisible && ready ? ' active' : ''}`}
      onClick={handleClick}
      title={ready ? 'Read aloud (Ctrl+Shift+L)' : 'Read aloud — needs configuration'}
    >
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4">
        <path d="M2.5 6.2v3.6h2.4L8.5 13V3L4.9 6.2H2.5z" fill="currentColor" stroke="none"/>
        <path d="M10.8 5.8a3 3 0 0 1 0 4.4"/>
        <path d="M12.6 4a5.6 5.6 0 0 1 0 8"/>
      </svg>
      {!ready && <span className="glyph-badge glyph-badge--setup">!</span>}
    </button>
  )
}

// ── Main panel ──────────────────────────────────────────────────────────────

export default function ToolsPanel() {
  const [activeSection, setActiveSection] = useState<GlyphSection>(null)
  const [panelWidth, setPanelWidth] = useState(350)
  const [isResizing, setIsResizing] = useState(false)
  const resizeRef = useRef<{ startX: number; startWidth: number } | null>(null)
  const restoredRef = useRef(false)
  const { settings, saveSettings, loaded } = useAppStore()
  const reviewCount = useReviewCount()

  // setSection persists open/closed + active pane so the sidebar survives
  // restarts and remounts (e.g. the Characters codex unmounting this panel).
  const setSection = (next: GlyphSection) => {
    setActiveSection(next)
    void saveSettings({ sidebar_active_section: next ?? '' })
  }

  // Restore the saved sidebar state once settings are loaded. An invalid or
  // plugin-disabled saved section falls back to the Dashboard rather than a
  // surprise-closed sidebar.
  useEffect(() => {
    if (!loaded || restoredRef.current) return
    restoredRef.current = true
    const saved = settings.sidebar_active_section
    if (!saved) return // deliberately closed
    const known = SECTION_CONFIG.some(s => s.id === saved)
    const section = (known ? saved : 'dashboard') as Exclude<GlyphSection, null>
    setActiveSection(isSectionEnabled(section, settings) ? section : 'dashboard')
  }, [loaded])

  // Load saved panel width from settings
  useEffect(() => {
    if (settings.sidebar_panel_width) {
      setPanelWidth(settings.sidebar_panel_width)
    }
  }, [settings.sidebar_panel_width])

  useEffect(() => {
    if (activeSection && !isSectionEnabled(activeSection, settings)) {
      setSection(null)
    }
  }, [
    activeSection,
    settings.ai_enabled,
    settings.cast_enabled,
    settings.analysis_enabled,
  ])

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
      // Save width to settings (serialized through the store's save chain)
      void saveSettings({ sidebar_panel_width: panelWidth })
    }

    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
    return () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }, [isResizing, panelWidth, saveSettings])

  const handleGlyphClick = (section: Exclude<GlyphSection, null>) => {
    setSection(activeSection === section ? null : section)
  }

  // Cross-panel deep links: any component can open a sidebar pane by
  // dispatching OPEN_TOOLS_SECTION_EVENT (see tools/Analysis/shared.ts).
  useEffect(() => {
    const handleOpenSection = (event: Event) => {
      const section = (event as CustomEvent<Exclude<GlyphSection, null>>).detail
      if (!section) return
      if (!SECTION_CONFIG.some(s => s.id === section)) return
      if (!isSectionEnabled(section, useAppStore.getState().settings)) return
      setSection(section)
    }
    window.addEventListener(OPEN_TOOLS_SECTION_EVENT, handleOpenSection)
    return () => window.removeEventListener(OPEN_TOOLS_SECTION_EVENT, handleOpenSection)
  }, [])

  const visibleSections = SECTION_CONFIG.filter(section => isSectionEnabled(section.id, settings))

  const activeSectionConfig = activeSection ? SECTION_CONFIG.find(s => s.id === activeSection) : null

  return (
    <div className="tools-sidebar">
      {/* Slide-out panel */}
      {activeSection && (
        <div className="slide-panel" style={{ width: panelWidth }}>
          <div className="resize-handle" onMouseDown={handleResizeStart} />
          <div className="slide-panel-header">
            <span className="slide-panel-title">{activeSectionConfig?.tooltip}</span>
            <button className="slide-panel-close" onClick={() => setSection(null)} title="Close panel">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>
          <div className="slide-panel-content">
            {activeSection === 'dashboard' && <DashboardTab />}
            {activeSection === 'characters' && <CharacterQuickRef />}
            {activeSection === 'signals' && <SignalsPanel />}
            {activeSection === 'prose' && <ProsePanel />}
            {activeSection === 'pacing' && <PacingPanel />}
            {activeSection === 'chapters' && <ChaptersPanel />}
            {activeSection === 'review' && <ReviewPanel />}
            {activeSection === 'aidetect' && <AIDetectPanel />}
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
            {section.id === 'review' && reviewCount > 0 && (
              <span className="glyph-badge">{reviewCount > 9 ? '9+' : reviewCount}</span>
            )}
          </button>
        ))}
        {settings.read_aloud_enabled && (
          <>
            <div className="glyph-bar-spacer" />
            <ReadAloudGlyph />
          </>
        )}
      </div>
    </div>
  )
}
