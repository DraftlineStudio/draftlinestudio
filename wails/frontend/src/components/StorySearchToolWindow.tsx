import { useCallback, useMemo, useState } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useAppStore } from '../store/appStore'
import { useBookStore } from '../store/bookStore'
import type { Section } from '../types/draftline'
import AskPanel from './storysearch/AskPanel'
import ContinuityPanel from './storysearch/ContinuityPanel'
import EvidenceIndexPanel from './storysearch/EvidenceIndexPanel'
import StoryMapPanel from './storysearch/StoryMapPanel'
import ThreadsPanel from './storysearch/ThreadsPanel'
import ReviewDeskPanel from './storysearch/ReviewDeskPanel'

type ToolView = 'map' | 'threads' | 'review' | 'continuity' | 'search' | 'evidence'

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
  const [activeView, setActiveView] = useState<ToolView>('map')
  const [continuityCounts, setContinuityCounts] = useState<{ review: number; info: number } | null>(null)
  const [reviewUndecided, setReviewUndecided] = useState<number | null>(null)
  // Stable identities: the panels report counts from effects, so a new
  // function each render would loop.
  const reportCounts = useCallback((counts: { review: number; info: number } | null) => setContinuityCounts(counts), [])
  const reportReviewCount = useCallback((undecided: number) => setReviewUndecided(undecided), [])
  const openThreadsTab = useCallback(() => setActiveView('threads'), [])

  const fingerprint = book?.analysis?.fingerprint

  // Tab badge before the Review panel has ever mounted: raw detection count
  // minus recorded decisions.
  const reviewBadge = useMemo(() => {
    if (reviewUndecided !== null) return reviewUndecided
    const diagnostics = fingerprint?.diagnostics ?? []
    const decided = new Set((book?.analysis?.continuity?.decisions ?? []).map(d => d.signal_id))
    return diagnostics.filter(d => !decided.has(d.id)).length
  }, [reviewUndecided, fingerprint, book?.analysis?.continuity?.decisions])

  const subtitle = useMemo(() => {
    switch (activeView) {
      case 'map': {
        const events = fingerprint?.events?.length ?? 0
        return events > 0
          ? `story time · ${events} events · the manuscript path weaves through it`
          : 'story time · builds after analysis runs'
      }
      case 'threads': {
        const threads = fingerprint?.threads ?? []
        if (threads.length === 0) return 'obligations the story has opened'
        const open = threads.filter(t => !['resolved', 'abandoned'].includes(t.state)).length
        const dormant = threads.filter(t => t.state === 'dormant').length
        return `${threads.length} threads · ${open} open${dormant ? ` · ${dormant} dormant` : ''}`
      }
      case 'review':
        return `${reviewBadge} detection${reviewBadge === 1 ? '' : 's'} · deterministic · sorted by severity`
      case 'continuity':
        return 'review questions · paired sources'
      case 'search':
        return 'answers assembled from evidence · source-backed · no AI'
      case 'evidence':
        return 'everything Draftline has indexed'
    }
  }, [activeView, fingerprint, reviewBadge])

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
    <section className="story-search-window" style={{ height: panelHeight }} aria-label="Story tools">
      <div className="story-search-resizer" onPointerDown={beginResize} />
      <header className="story-search-header">
        <Tab view="map" active={activeView} onSelect={setActiveView} label="Story Map">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M3 12h4l2-7 4 14 2-7h6" />
          </svg>
        </Tab>
        <Tab view="threads" active={activeView} onSelect={setActiveView} label="Threads">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M5 4v13M5 6h11l-2 3.5L16 13H5" />
          </svg>
        </Tab>
        <Tab view="review" active={activeView} onSelect={setActiveView} label="Review">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M12 3l9 5-9 5-9-5 9-5M3 13l9 5 9-5" />
          </svg>
          {reviewBadge > 0 && <small className="story-search-tab-badge">{reviewBadge > 99 ? '99+' : reviewBadge}</small>}
        </Tab>
        <Tab view="continuity" active={activeView} onSelect={setActiveView} label="Continuity">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="12" cy="12" r="9" /><path d="M8.5 12l2.5 2.5 4.5-5" />
          </svg>
        </Tab>
        <Tab view="search" active={activeView} onSelect={setActiveView} label="Ask Draftline">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="11" cy="11" r="7" /><path d="M21 21l-4.3-4.3" />
          </svg>
        </Tab>

        <span className="story-search-header-hint">{subtitle}</span>
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
          onClick={() => setActiveView(activeView === 'evidence' ? 'map' : 'evidence')}
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

      {activeView === 'map' && book && <StoryMapPanel book={book} onNavigate={navigateSource} />}
      {activeView === 'threads' && book && <ThreadsPanel book={book} onNavigate={navigateSource} />}
      {activeView === 'review' && book && <ReviewDeskPanel book={book} onNavigate={navigateSource} onCount={reportReviewCount} onOpenThreads={openThreadsTab} />}
      {activeView === 'continuity' && book && <ContinuityPanel book={book} onNavigate={navigateSource} onCounts={reportCounts} />}
      {activeView === 'search' && book && <AskPanel book={book} onNavigate={navigateSource} />}
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
