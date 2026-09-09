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
import CharactersView from './components/characters/CharactersView'
import ToolsPanel from './components/ToolsPanel'
import StatusBar from './components/StatusBar'
import AnalysisCoordinator from './components/AnalysisCoordinator'
import StorySearchToolWindow from './components/StorySearchToolWindow'
import MetadataDialog from './components/dialogs/MetadataDialog'
import NewChapterDialog from './components/dialogs/NewChapterDialog'
import NewBookWizard from './components/dialogs/NewBookWizard'
import NewUniverseWizard from './components/dialogs/NewUniverseWizard'
import UnsavedChangesDialog from './components/dialogs/UnsavedChangesDialog'
import AppSettingsDialog from './components/dialogs/AppSettingsDialog'
import ExportWizard from './components/dialogs/ExportWizard'

const ChapterHistoryDialog = lazy(() => import('./components/dialogs/ChapterHistoryDialog'))

export default function App() {
  const { hasBook, bookTitle, bookFilePath, newBook, openBook, openRecentBook, saveBook, saveBookAs, dialogs, initBook, viewMode, setViewMode } = useBookStore(useShallow(s => ({
    hasBook: s.book !== null,
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
  const { loadSettings, settings, showSettings, showWelcome, setShowWelcome, loadRecentProjects, recentProjects, showNewUniverse, setShowNewUniverse, toggleLeftPanel, showMetadata, showNewChapter, newChapterSection, showExportWizard, showChapterHistory, bottomToolOpen, openStorySearch, closeBottomTool } = useAppStore()
  const prevThemeRef = useRef<'light' | 'dark' | null>(null)
  const [isTransitioning, setIsTransitioning] = useState(false)
  const [targetTheme, setTargetTheme] = useState<'light' | 'dark'>('dark')

  // Initialise on first load + load persisted settings and recent projects
  useEffect(() => {
    void loadSettings().then(() => startUpdateNag())
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

  // Handle new universe from welcome screen
  const handleNewUniverse = useCallback(() => {
    setShowNewUniverse(true)
  }, [setShowNewUniverse])

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
    setShowWelcome(false)
  }, [openRecentBook, setShowWelcome])

  // Handle theme transition with smooth fade animation and sky overlay
  const handleThemeTransition = useCallback((newTheme: 'light' | 'dark') => {
    // Trigger the sky animation overlay
    setTargetTheme(newTheme)
    setIsTransitioning(true)

    // Add transition class for smooth color animation
    document.documentElement.classList.add('theme-transitioning')

    // Reset transition state after UI colors finish (0.8s)
    setTimeout(() => {
      document.documentElement.classList.remove('theme-transitioning')
    }, 900)

    // Keep sky animation going a bit longer for the eye candy
    // Must be longer than 3200ms (hideTimer in ThemeTransitionOverlay)
    setTimeout(() => {
      setIsTransitioning(false)
    }, 3500)
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

  // Show welcome screen
  if (showWelcome) {
    return (
      <div className="app">
        <ThemeTransitionOverlay isTransitioning={isTransitioning} targetTheme={targetTheme} />
        <TitleBar minimal />
        <WelcomeScreen
          onNewBook={handleNewBook}
          onNewUniverse={handleNewUniverse}
          onOpenFile={handleOpenFile}
          onOpenRecent={handleOpenRecent}
        />
        {dialogs.showNewBookWizard && <NewBookWizard onCreated={() => setShowWelcome(false)} />}
        {showNewUniverse && <NewUniverseWizard />}
        {showSettings && <AppSettingsDialog />}
      </div>
    )
  }

  return (
    <div className="app">
      <ThemeTransitionOverlay isTransitioning={isTransitioning} targetTheme={targetTheme} />
      <TitleBar />
      <div className="main-layout">
        <ChapterPanel />
        {viewMode === 'cast' ? (
          <CharactersView />
        ) : (
          <ErrorBoundary name="Editor">
            <EditorPanel />
          </ErrorBoundary>
        )}
        {viewMode !== 'cast' && <ToolsPanel />}
      </div>
      {bottomToolOpen && viewMode !== 'cast' && <StorySearchToolWindow />}
      <StatusBar />
      <AnalysisCoordinator />
      {showMetadata && <MetadataDialog />}
      {showNewChapter && newChapterSection && (
        <NewChapterDialog section={newChapterSection} />
      )}
      {dialogs.showNewBookWizard && <NewBookWizard />}
      {showNewUniverse && <NewUniverseWizard />}
      {dialogs.showUnsavedWarning && <UnsavedChangesDialog />}
      {showExportWizard && <ExportWizard />}
      {showChapterHistory && <Suspense fallback={null}><ChapterHistoryDialog /></Suspense>}
      {showSettings && <AppSettingsDialog />}
    </div>
  )
}
