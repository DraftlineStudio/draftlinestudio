import { lazy, Suspense, useEffect, useRef, useCallback, useState } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from './store/bookStore'
import { useAppStore } from './store/appStore'
import { startUpdateNag } from './services/updateNag'
import { useReadAloudStore } from './store/readAloudStore'
import { TakePendingOpenPath } from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { applyAccent, clearAccent } from './utils/accentColor'
import { useAutoTheme } from './hooks/useAutoTheme'
import ThemeTransitionOverlay from './components/ThemeTransitionOverlay'
import WelcomeScreen from './components/WelcomeScreen'
import TitleBar from './components/TitleBar'
import ChapterPanel from './components/ChapterPanel'
import EditorPanel from './components/EditorPanel'
import ErrorBoundary from './components/ErrorBoundary'
import BackendErrorNotice from './components/BackendErrorNotice'
import { firstProjectPath, subscribeFileDrop } from './services/fileDrop'
import PlannerPanel from './components/planner/PlannerPanel'
import ToolsPanel from './components/ToolsPanel'
import StatusBar from './components/StatusBar'
import AnalysisCoordinator from './components/AnalysisCoordinator'
import StorySearchToolWindow from './components/StorySearchToolWindow'
import NewChapterDialog from './components/dialogs/NewChapterDialog'
import NewBookWizard from './components/dialogs/NewBookWizard'
import UnsavedChangesDialog from './components/dialogs/UnsavedChangesDialog'
import BookLockDialog from './components/dialogs/BookLockDialog'

// Screens and dialogs that are not on screen at startup. Each one is a
// parse-and-compile cost the writer would otherwise pay before the editor
// appears, for something they may never open this session.
const ChapterHistoryDialog = lazy(() => import('./components/dialogs/ChapterHistoryDialog'))
const AppSettingsDialog = lazy(() => import('./components/dialogs/settings'))
const ExportWizard = lazy(() => import('./components/dialogs/ExportWizard'))
const BookInfoDialog = lazy(() => import('./components/dialogs/BookInfoDialog'))
const CharactersView = lazy(() => import('./components/characters/CharactersView'))
const PlannerView = lazy(() => import('./components/planner/PlannerView'))

export default function App() {
  const { hasBook, bookTitle, bookFilePath, newBook, openBook, openRecentBook, saveBook, saveBookAs, dialogs, initBook, viewMode, setViewMode, isOpening } = useBookStore(useShallow(s => ({
    hasBook: s.book !== null,
    isOpening: s.isOpening,
    bookTitle: s.book?.metadata.title,
    bookFilePath: s.book?.file_path,
    newBook: s.newBook,
    openBook: s.openBook,
    openRecentBook: s.openRecentBook,
    saveBook: s.saveBook,
    saveBookAs: s.saveBookAs,
    dialogs: s.dialogs,
    initBook: s.initBook,
    viewMode: s.viewMode,
    setViewMode: s.setViewMode,
  })))
  const { loadSettings, settings, showSettings, showWelcome, setShowWelcome, loadRecentProjects, recentProjects, toggleLeftPanel, showMetadata, showNewChapter, newChapterSection, showExportWizard, showChapterHistory, bottomToolOpen, openStorySearch, closeBottomTool } = useAppStore()
  const prevThemeRef = useRef<'light' | 'dark' | null>(null)
  const [isTransitioning, setIsTransitioning] = useState(false)
  const [targetTheme, setTargetTheme] = useState<'light' | 'dark'>('dark')
  // Incremented per transition so the overlay restarts its phase timers
  // even when a new transition begins while one is already showing.
  const [transitionNonce, setTransitionNonce] = useState(0)

  // Initialise on first load + load persisted settings and recent projects
  useEffect(() => {
    void loadSettings().then(() => {
      startUpdateNag()
      // Plugins activate after settings so enabled flags are authoritative.
      // Imported here rather than at the top so the whole plugin host stays
      // out of the startup bundle; nothing needs it before this point.
      void import('./services/plugins/loader').then(m => m.initPlugins())
    })
    void loadRecentProjects()
  }, [])

  // OS file associations / "Open with": the file the app was launched with,
  // plus files forwarded from second instances (single-instance lock) and
  // macOS open-file events. Routed through the normal open/import flows so
  // the unsaved-changes dialog is respected.
  useEffect(() => {
    const open = (path: string) => {
      if (path) void useBookStore.getState().openExternalFile(path)
    }
    void TakePendingOpenPath().then(open)
    return EventsOn('file:open', (path: string) => open(path))
  }, [])

  // Handle new book from welcome screen
  const handleNewBook = useCallback(() => {
    newBook()
  }, [newBook])

  // Handle open file from welcome screen
  const handleOpenFile = useCallback(async () => {
    await openBook()
    // If a book was opened, hide welcome screen
    const currentBook = useBookStore.getState().book
    if (currentBook?.file_path) {
      setShowWelcome(false)
    }
  }, [openBook, setShowWelcome])

  // Handle open recent from welcome screen
  const handleOpenRecent = useCallback(async (path: string) => {
    await openRecentBook(path)
    // Leave the welcome screen only once a book is actually loaded. A failed
    // open (the book is locked by another window, the archive is unreadable)
    // reports through the status bar and must not strand the writer on an
    // empty editor with no project to close.
    if (useBookStore.getState().book) setShowWelcome(false)
  }, [openRecentBook, setShowWelcome])

  // Dropping a .draftline on the window opens it, the way dropping one on
  // the application icon does.
  //
  // Lowest priority, so a cover dropped on the cover card is claimed there
  // first and only what nothing else wanted arrives here. Registered for the
  // life of the app: with nothing subscribed, Wails removes its drop listeners
  // and WebView2 downloads the file instead of opening it.
  useEffect(() => subscribeFileDrop((_x, _y, paths) => {
    const path = firstProjectPath(paths)
    if (!path) return false
    void handleOpenRecent(path)
    return true
  }, 0), [handleOpenRecent])

  // Handle theme transition with smooth fade animation and sky overlay.
  // Retriggering mid-flight (rapid theme toggling, or the auto-theme hook and
  // the theme effect both firing) must RESTART the clocks, or a first
  // transition's timers fire into a second transition's animation.
  const themeTimersRef = useRef<number[]>([])
  const handleThemeTransition = useCallback((newTheme: 'light' | 'dark') => {
    themeTimersRef.current.forEach(clearTimeout)
    themeTimersRef.current = []

    // Trigger the sky animation overlay; the nonce restarts its phase
    // timers even when a transition is already showing.
    setTargetTheme(newTheme)
    setIsTransitioning(true)
    setTransitionNonce(n => n + 1)

    // Add transition class for smooth color animation
    document.documentElement.classList.add('theme-transitioning')

    themeTimersRef.current.push(
      // Reset transition state after UI colors finish (0.8s)
      window.setTimeout(() => {
        document.documentElement.classList.remove('theme-transitioning')
      }, 900),
      // Keep sky animation going a bit longer for the eye candy.
      // Must be longer than 3200ms (hideTimer in ThemeTransitionOverlay)
      window.setTimeout(() => {
        setIsTransitioning(false)
      }, 3500),
    )
  }, [])

  // Use auto theme hook for automatic dawn/dusk transitions
  const { computedTheme } = useAutoTheme({
    themeMode: settings.theme_mode,
    manualDawn: settings.auto_theme_dawn,
    manualDusk: settings.auto_theme_dusk,
    useManualTimes: settings.auto_theme_use_manual,
    onTransition: handleThemeTransition
  })

  // Apply theme changes (from auto mode or manual settings)
  useEffect(() => {
    // Handle smooth transition when theme changes (not on initial load)
    if (prevThemeRef.current !== null && prevThemeRef.current !== computedTheme) {
      handleThemeTransition(computedTheme)
    }

    document.documentElement.setAttribute('data-theme', computedTheme)
    prevThemeRef.current = computedTheme
  }, [computedTheme])

  // Drive accent color from the open book's title — or, on the welcome
  // screen, from the last-opened book so the titlebar strip and Studio Glow
  // set the scene (falls back to the brand accent with no recents).
  const accentTitle = bookTitle ?? recentProjects[0]?.name ?? null
  useEffect(() => {
    if (accentTitle) applyAccent(accentTitle)
    else clearAccent()
  }, [accentTitle])

  // Global keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (!e.ctrlKey) return
      switch (e.key.toLocaleLowerCase()) {
        case 'n': e.preventDefault(); void newBook(); break
        case 'o': e.preventDefault(); void openBook(); break
        case '[': e.preventDefault(); toggleLeftPanel(); break
        case 'f':
          if (e.shiftKey && hasBook) {
            e.preventDefault()
            setViewMode('editor')
            openStorySearch()
          }
          break
        case 's':
          e.preventDefault()
          if (e.shiftKey) void saveBookAs()
          else void saveBook()
          break
        case 'l':
          // Read Aloud: start from selection/cursor, or toggle pause while active.
          if (e.shiftKey && hasBook && useAppStore.getState().settings.read_aloud_enabled) {
            e.preventDefault()
            const readAloud = useReadAloudStore.getState()
            if (readAloud.status === 'idle') readAloud.playSelection()
            else readAloud.togglePause()
          }
          break
        // Shift+period / Shift+comma report as '>' and '<' on most layouts.
        case '.':
        case '>':
        case ',':
        case '<':
          if (e.shiftKey && useReadAloudStore.getState().status !== 'idle') {
            e.preventDefault()
            useReadAloudStore.getState().skip(e.key === '.' || e.key === '>' ? 1 : -1)
          }
          break
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [hasBook, newBook, openBook, openStorySearch, saveBook, saveBookAs, setViewMode, toggleLeftPanel])

  // When book is opened from NewBookWizard, hide welcome screen
  useEffect(() => {
    if (hasBook && !showWelcome) return
    if (hasBook && (bookFilePath || !dialogs.showNewBookWizard)) {
      // Book was created/opened, hide welcome
      setShowWelcome(false)
    }
  }, [hasBook, bookFilePath, dialogs.showNewBookWizard, showWelcome, setShowWelcome])

  useEffect(() => {
    if (!settings.cast_enabled && viewMode === 'cast') setViewMode('editor')
  }, [settings.cast_enabled, viewMode, setViewMode])

  // Enabling the Read Aloud plugin runs the full-hash model verification
  // (playback stays locked until it passes); disabling unloads everything —
  // playback stops, the synthesis worker (and the model in its memory) is
  // terminated.
  useEffect(() => {
    if (settings.read_aloud_enabled) void useReadAloudStore.getState().verifyModel()
    else useReadAloudStore.getState().shutdown()
  }, [settings.read_aloud_enabled])

  // Device/thread configuration is read at worker start; tearing the worker
  // down on change makes the next playback session pick it up.
  useEffect(() => {
    useReadAloudStore.getState().shutdown()
  }, [settings.read_aloud_threads])

  useEffect(() => {
    if (!hasBook && bottomToolOpen) closeBottomTool()
  }, [bottomToolOpen, closeBottomTool, hasBook])

  // Full-screen overlay while a book archive loads: feedback for large
  // books, and an input shield so a second open cannot race the first.
  // It appears after a short delay (instant opens never flash it) and, on
  // completion, releases input at once but crossfades out over ~500ms so
  // the book surfaces through it.
  const [overlayPhase, setOverlayPhase] = useState<'hidden' | 'open' | 'closing'>('hidden')
  useEffect(() => {
    if (isOpening) {
      setOverlayPhase('open')
      return
    }
    setOverlayPhase(phase => (phase === 'open' ? 'closing' : phase))
    const timer = setTimeout(() => setOverlayPhase('hidden'), 550)
    return () => clearTimeout(timer)
  }, [isOpening])
  const openingOverlay = overlayPhase !== 'hidden' && (
    <div className={`opening-overlay${overlayPhase === 'closing' ? ' closing' : ''}`}>
      <div className="opening-spinner" />
      <div className="opening-label">Opening book…</div>
    </div>
  )

  // Show welcome screen. Having no book is always the welcome screen: the
  // editor layout has nothing to show without a project, and its project
  // menu cannot close what is not open.
  if (showWelcome || !hasBook) {
    return (
      <div className="app">
        <ThemeTransitionOverlay isTransitioning={isTransitioning} targetTheme={targetTheme} nonce={transitionNonce} />
        <TitleBar minimal />
        <WelcomeScreen
          onNewBook={handleNewBook}
          onOpenFile={handleOpenFile}
          onOpenRecent={handleOpenRecent}
        />
        {dialogs.showNewBookWizard && <NewBookWizard onCreated={() => setShowWelcome(false)} />}
        {dialogs.bookLockWarning && <BookLockDialog />}
        {showSettings && <Suspense fallback={null}><AppSettingsDialog /></Suspense>}
        {openingOverlay}
        <BackendErrorNotice />
      </div>
    )
  }

  return (
    <div className="app">
      <ThemeTransitionOverlay isTransitioning={isTransitioning} targetTheme={targetTheme} nonce={transitionNonce} />
      <TitleBar />
      <div className="main-layout">
        {/* The character codex is its own workspace: the manuscript rail does
            nothing there, so it gives the codex the width instead. */}
        {viewMode !== 'cast' && <ChapterPanel />}
        {viewMode === 'cast' ? (
          <Suspense fallback={null}><CharactersView /></Suspense>
        ) : viewMode === 'planner' ? (
          <ErrorBoundary name="Planner">
            <Suspense fallback={null}><PlannerView /></Suspense>
          </ErrorBoundary>
        ) : (
          <ErrorBoundary name="Editor">
            <EditorPanel />
          </ErrorBoundary>
        )}
        {viewMode === 'planner' && <PlannerPanel />}
        {viewMode === 'editor' && <ToolsPanel />}
      </div>
      {bottomToolOpen && viewMode === 'editor' && <StorySearchToolWindow />}
      <StatusBar />
      <AnalysisCoordinator />
      {showMetadata && <Suspense fallback={null}><BookInfoDialog /></Suspense>}
      {showNewChapter && newChapterSection && (
        <NewChapterDialog section={newChapterSection} />
      )}
      {dialogs.showNewBookWizard && <NewBookWizard />}
      {dialogs.showUnsavedWarning && <UnsavedChangesDialog />}
      {dialogs.bookLockWarning && <BookLockDialog />}
      {showExportWizard && <Suspense fallback={null}><ExportWizard /></Suspense>}
      {showChapterHistory && <Suspense fallback={null}><ChapterHistoryDialog /></Suspense>}
      {showSettings && <Suspense fallback={null}><AppSettingsDialog /></Suspense>}
      {openingOverlay}
      <BackendErrorNotice />
    </div>
  )
}
