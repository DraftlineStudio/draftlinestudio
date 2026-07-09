import { useEffect, useRef, useCallback, useState } from 'react'
import { useBookStore } from './store/bookStore'
import { useAppStore } from './store/appStore'
import { applyAccent, clearAccent } from './utils/accentColor'
import { useAutoTheme } from './hooks/useAutoTheme'
import ThemeTransitionOverlay from './components/ThemeTransitionOverlay'
import WelcomeScreen from './components/WelcomeScreen'
import TitleBar from './components/TitleBar'
import ChapterPanel from './components/ChapterPanel'
import EditorPanel from './components/EditorPanel'
import CodexPanel from './components/CodexPanel'
import ToolsPanel from './components/ToolsPanel'
import StatusBar from './components/StatusBar'
import MetadataDialog from './components/dialogs/MetadataDialog'
import NewChapterDialog from './components/dialogs/NewChapterDialog'
import NewBookWizard from './components/dialogs/NewBookWizard'
import NewUniverseWizard from './components/dialogs/NewUniverseWizard'
import UnsavedChangesDialog from './components/dialogs/UnsavedChangesDialog'
import AppSettingsDialog from './components/dialogs/AppSettingsDialog'
import ExportWizard from './components/dialogs/ExportWizard'

export default function App() {
  const { book, newBook, openBook, openRecentBook, saveBook, saveBookAs, dialogs, initBook, setDarkMode, toggleLeftPanel, toggleRightPanel, viewMode } = useBookStore()
  const { loadSettings, settings, showSettings, showWelcome, setShowWelcome, loadRecentProjects, showNewUniverse, setShowNewUniverse } = useAppStore()
  const prevThemeRef = useRef<'light' | 'dark' | null>(null)
  const [isTransitioning, setIsTransitioning] = useState(false)
  const [targetTheme, setTargetTheme] = useState<'light' | 'dark'>('dark')

  // Initialise on first load + load persisted settings and recent projects
  useEffect(() => {
    loadSettings()
    loadRecentProjects()
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
    const newDarkMode = computedTheme === 'dark'

    // Handle smooth transition when theme changes (not on initial load)
    if (prevThemeRef.current !== null && prevThemeRef.current !== computedTheme) {
      handleThemeTransition(computedTheme)
    }

    setDarkMode(newDarkMode)
    document.documentElement.setAttribute('data-theme', computedTheme)
    prevThemeRef.current = computedTheme
  }, [computedTheme, setDarkMode])

  // Drive accent color from book title's avatar color
  useEffect(() => {
    const title = book?.metadata.title
    if (title) applyAccent(title)
    else clearAccent()
  }, [book?.metadata.title])

  // Global keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (!e.ctrlKey) return
      switch (e.key) {
        case 'n': e.preventDefault(); newBook(); break
        case 'o': e.preventDefault(); openBook(); break
        case '[': e.preventDefault(); toggleLeftPanel(); break
        case ']': e.preventDefault(); toggleRightPanel(); break
        case 's':
          e.preventDefault()
          if (e.shiftKey) saveBookAs()
          else saveBook()
          break
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [newBook, openBook, saveBook, saveBookAs, toggleLeftPanel, toggleRightPanel])

  // When book is opened from NewBookWizard, hide welcome screen
  useEffect(() => {
    if (book && !showWelcome) return
    if (book && (book.file_path || !dialogs.showNewBookWizard)) {
      // Book was created/opened, hide welcome
      setShowWelcome(false)
    }
  }, [book, dialogs.showNewBookWizard])

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
        {viewMode === 'codex' ? <CodexPanel /> : <EditorPanel />}
        <ToolsPanel />
      </div>
      <StatusBar />
      {dialogs.showMetadata && <MetadataDialog />}
      {dialogs.showNewChapter && dialogs.newChapterSection && (
        <NewChapterDialog section={dialogs.newChapterSection} />
      )}
      {dialogs.showNewBookWizard && <NewBookWizard />}
      {showNewUniverse && <NewUniverseWizard />}
      {dialogs.showUnsavedWarning && <UnsavedChangesDialog />}
      {dialogs.showExportWizard && <ExportWizard />}
      {showSettings && <AppSettingsDialog />}
    </div>
  )
}
