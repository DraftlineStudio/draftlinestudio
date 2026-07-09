// ToolsPanel - Main sidebar with section routing

import { useState, useEffect, useRef } from 'react'
import { useAppStore } from '../store/appStore'

// Extracted types, constants, and components
import type { GlyphSection } from './tools/types'
import { SECTION_CONFIG } from './tools/constants'
import GlyphIcon from './tools/GlyphIcon'
import DashboardTab from './tools/Dashboard'
import AiStudioTab from './tools/AIStudio'
import { CharactersSection, PlotSection, TimelineSection } from './tools/StoryBible'
import { BeatsSection, ForeshadowingSection, KnowledgeSection, IssuesSection } from './tools/PlotWalker'

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
