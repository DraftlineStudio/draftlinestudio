import { useEffect, useRef, useCallback, useState } from 'react'
import { useBookStore } from './store/bookStore'
import { useAppStore } from './store/appStore'
import { applyAccent, clearAccent } from './utils/accentColor'
import { useAutoTheme } from './hooks/useAutoTheme'
import ThemeTransitionOverlay from './components/ThemeTransitionOverlay'
import TitleBar from './components/TitleBar'
import ChapterPanel from './components/ChapterPanel'
import EditorPanel from './components/EditorPanel'
import ToolsPanel from './components/ToolsPanel'
import StatusBar from './components/StatusBar'
import MetadataDialog from './components/dialogs/MetadataDialog'
import NewChapterDialog from './components/dialogs/NewChapterDialog'
import NewBookWizard from './components/dialogs/NewBookWizard'
import UnsavedChangesDialog from './components/dialogs/UnsavedChangesDialog'
import AppSettingsDialog from './components/dialogs/AppSettingsDialog'

export default function App() {
  const { darkMode, book, newBook, openBook, saveBook, saveBookAs, dialogs, initBook, setDarkMode, toggleLeftPanel, toggleRightPanel, rightPanelOpen } = useBookStore()
  const { loadSettings, settings, showSettings } = useAppStore()
  const prevThemeRef = useRef<'light' | 'dark' | null>(null)
  const [isTransitioning, setIsTransitioning] = useState(false)
  const [targetTheme, setTargetTheme] = useState<'light' | 'dark'>('dark')

  // Initialise with a blank book on first load + load persisted settings
  useEffect(() => {
    initBook()
    loadSettings()
  }, [])

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
    setTimeout(() => {
      setIsTransitioning(false)
    }, 3000)
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

  return (
    <div className="app">
      <ThemeTransitionOverlay isTransitioning={isTransitioning} targetTheme={targetTheme} />
      <TitleBar />
      <div className="main-layout">
        <ChapterPanel />
        <EditorPanel />
        <ToolsPanel />
        {!rightPanelOpen && (
          <button className="panel-rail-right" onClick={toggleRightPanel} title="Open Tools panel (Ctrl+])">
            <svg width="6" height="10" viewBox="0 0 6 10" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 1L1 5l4 4" />
            </svg>
            <span className="panel-collapsed-label">Tools</span>
          </button>
        )}
      </div>
      <StatusBar />
      {dialogs.showMetadata && <MetadataDialog />}
      {dialogs.showNewChapter && dialogs.newChapterSection && (
        <NewChapterDialog section={dialogs.newChapterSection} />
      )}
      {dialogs.showNewBookWizard && <NewBookWizard />}
      {dialogs.showUnsavedWarning && <UnsavedChangesDialog />}
      {showSettings && <AppSettingsDialog />}
    </div>
  )
}
