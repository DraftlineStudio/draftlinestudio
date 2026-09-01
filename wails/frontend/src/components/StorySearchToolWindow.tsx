import { useCallback, useState } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useAppStore } from '../store/appStore'
import { useBookStore } from '../store/bookStore'
import type { Section } from '../types/draftline'
import AskPanel from './storysearch/AskPanel'
import ContinuityPanel from './storysearch/ContinuityPanel'
import EvidenceIndexPanel from './storysearch/EvidenceIndexPanel'
import StoryGraphPanel from './storysearch/StoryGraphPanel'

type ToolView = 'search' | 'graph' | 'continuity' | 'evidence'

const HINTS: Record<ToolView, string> = {
  search: 'Source-backed manuscript trails · no AI',
  graph: 'Derived threads and beats · manuscript order',
  continuity: 'Review questions · paired sources',
  evidence: 'Everything Draftline has indexed',
}

/** Collapsed height of the bar; the expand toggle swaps between this and tall. */
const TALL_HEIGHT = 560

export default function StorySearchToolWindow() {
  const { book, setCurrentChapter, setViewMode } = useBookStore(useShallow(s => ({
    book: s.book,
    setCurrentChapter: s.setCurrentChapter,
    setViewMode: s.setViewMode,
  })))
  const { height, close, saveHeight } = useAppStore(useShallow(s => ({
    height: s.bottomToolHeight,
    close: s.closeBottomTool,
    saveHeight: s.setBottomToolHeight,
  })))
  const [panelHeight, setPanelHeight] = useState(height)
  const [restoreHeight, setRestoreHeight] = useState(height)
  const [activeView, setActiveView] = useState<ToolView>('search')
  const [continuityCounts, setContinuityCounts] = useState<{ review: number; info: number } | null>(null)
  // Stable identity: the panel reports counts from an effect, so a new function
  // each render would loop.
  const reportCounts = useCallback((counts: { review: number; info: number } | null) => setContinuityCounts(counts), [])

  function beginResize(event: React.PointerEvent<HTMLDivElement>) {
    event.preventDefault()
    const startY = event.clientY
    const startHeight = panelHeight
    const clamp = (value: number) => Math.max(170, Math.min(TALL_HEIGHT, value))
    const onMove = (move: PointerEvent) => setPanelHeight(clamp(startHeight + startY - move.clientY))
    const onUp = (up: PointerEvent) => {
      const finalHeight = clamp(startHeight + startY - up.clientY)
      setPanelHeight(finalHeight)
      saveHeight(finalHeight)
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
    }
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }

  function toggleExpand() {
    if (panelHeight >= TALL_HEIGHT - 4) {
      const target = Math.max(170, Math.min(TALL_HEIGHT, restoreHeight))
      setPanelHeight(target)
      saveHeight(target)
      return
    }
    setRestoreHeight(panelHeight)
    setPanelHeight(TALL_HEIGHT)
    saveHeight(TALL_HEIGHT)
  }

  function navigateSource(section: Section, sectionIndex: number, evidenceQuery: string) {
    setViewMode('editor')
    setCurrentChapter(section, sectionIndex)
    window.setTimeout(() => {
      window.dispatchEvent(new CustomEvent('draftline:find-story-evidence', { detail: { query: evidenceQuery } }))
    }, 0)
  }

  const expanded = panelHeight >= TALL_HEIGHT - 4
  const evidenceCount = book?.analysis?.evidence?.records.length ?? 0

  return (
    <section className="story-search-window" style={{ height: panelHeight }} aria-label="Ask Draftline and Story Graph">
      <div className="story-search-resizer" onPointerDown={beginResize} />
      <header className="story-search-header">
        <Tab view="search" active={activeView} onSelect={setActiveView} label="Ask Draftline">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="11" cy="11" r="7" /><path d="M21 21l-4.3-4.3" />
          </svg>
        </Tab>
        <Tab view="graph" active={activeView} onSelect={setActiveView} label="Story Graph">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="5" cy="6" r="2.2" /><circle cx="12" cy="12" r="2.2" /><circle cx="19" cy="6" r="2.2" /><path d="M7 7.5l3 3M14 10.5l3-3" />
          </svg>
        </Tab>
        <Tab view="continuity" active={activeView} onSelect={setActiveView} label="Continuity">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="12" cy="12" r="9" /><path d="M8.5 12l2.5 2.5 4.5-5" />
          </svg>
        </Tab>

        <span className="story-search-header-hint">{HINTS[activeView]}</span>
        <div className="story-search-header-spacer" />

        {activeView === 'continuity' && continuityCounts && (
          <>
            {continuityCounts.review > 0 && <span className="story-search-badge review">{continuityCounts.review} review</span>}
            <span className="story-search-badge">{continuityCounts.info} {continuityCounts.info === 1 ? 'observation' : 'observations'}</span>
          </>
        )}

        <button
          type="button"
          className={`story-search-icon-btn ${activeView === 'evidence' ? 'active' : ''}`}
          onClick={() => setActiveView(activeView === 'evidence' ? 'search' : 'evidence')}
          title={`All deterministic detections${evidenceCount ? ` · ${evidenceCount} records` : ''}`}
          aria-label="Open all deterministic detections"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <path d="M12 10a2 2 0 0 0-2 2c0 2.5-.6 4.5-1.5 6" />
            <path d="M12 6a6 6 0 0 0-6 6c0 1.4-.2 2.7-.6 4" />
            <path d="M12 2a10 10 0 0 0-8 4" />
            <path d="M16 3.3A10 10 0 0 1 22 12c0 .8 0 1.6-.1 2.3" />
            <path d="M16 12a4 4 0 0 0-8 0c0 2-.3 3.8-.9 5.4" />
            <path d="M16 12c0 3-.5 5.8-1.4 8.3" />
            <path d="M12 14c0 2.8-.5 5.4-1.3 7.8" />
          </svg>
          {!!evidenceCount && <small>{evidenceCount}</small>}
        </button>

        <button
          type="button"
          className="story-search-icon-btn"
          onClick={toggleExpand}
          title={expanded ? 'Restore panel height' : 'Expand panel'}
          aria-label={expanded ? 'Restore panel height' : 'Expand panel'}
        >
          {expanded ? (
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
              <path d="M10 20v-6H4M14 4v6h6M4 20l6-6M20 4l-6 6" />
            </svg>
          ) : (
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
              <path d="M4 14v6h6M20 10V4h-6M4 20l6-6M20 4l-6 6" />
            </svg>
          )}
        </button>

        <button type="button" className="story-search-icon-btn" onClick={close} title="Close bottom tool" aria-label="Close bottom tool">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round">
            <path d="M5 5l14 14M19 5L5 19" />
          </svg>
        </button>
      </header>

      {activeView === 'search' && book && <AskPanel book={book} onNavigate={navigateSource} />}
      {activeView === 'graph' && book && <StoryGraphPanel book={book} onNavigate={navigateSource} onOpenCodex={() => setViewMode('cast')} />}
      {activeView === 'continuity' && book && <ContinuityPanel book={book} onNavigate={navigateSource} onCounts={reportCounts} />}
      {activeView === 'evidence' && book && <EvidenceIndexPanel book={book} onNavigate={navigateSource} />}
    </section>
  )
}

function Tab({ view, active, onSelect, label, children }: {
  view: ToolView
  active: ToolView
  onSelect: (view: ToolView) => void
  label: string
  children: React.ReactNode
}) {
  return (
    <button type="button" className={`story-search-tab ${active === view ? 'active' : ''}`} onClick={() => onSelect(view)}>
      {children}
      {label}
    </button>
  )
}
