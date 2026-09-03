import { create } from 'zustand'
import { LoadSettings, SaveSettings, BrowseForDirectory, GetRecentProjects, AddRecentProject, RemoveRecentProject, ClearRecentProjects } from '../../wailsjs/go/main/App'
import { types } from '../../wailsjs/go/models'
import type { Section } from '../types/draftline'

type RecentProject = types.RecentProject

export interface AppSettings {
  // Application
  default_author: string
  default_publisher: string
  default_copyright: string
  default_save_dir: string
  dark_mode: boolean
  theme_mode: 'light' | 'dark' | 'auto'
  auto_theme_use_manual: boolean
  auto_theme_dawn: string
  auto_theme_dusk: string
  activity_autosave_enabled: boolean
  custom_dictionary: string[]
  spell_check_enabled: boolean
  grammar_check_enabled: boolean
  cast_enabled: boolean
  story_bible_enabled: boolean
  plot_walker_enabled: boolean
  analysis_enabled: boolean
  analysis_cpu_profile: 'adaptive' | 'gentle' | 'balanced' | 'fast'
  // Read Aloud (opt-in local TTS)
  read_aloud_enabled: boolean
  read_aloud_voice: string
  read_aloud_speed: number
  read_aloud_device: 'native'
  read_aloud_threads: 'single' | 'auto'
  // AI
  ai_enabled: boolean
  ai_mode: 'claudecode' | 'codex' | 'api' | 'local'
  ai_provider: 'claude' | 'openai' | 'gemini' | 'grok' | ''
  characters_lane_view: 'grid' | 'heat' | 'weave'
  has_api_key: boolean
  ai_debug_logging: boolean
  ai_model: string
  ai_local_endpoint: string
  ai_local_model: string
  prose_guide: string
  // Book defaults
  book_font: string
  editor_font_size: 'small' | 'normal' | 'large'  // Editor display size: 12/14/16
  book_font_size: number                           // Export font size in points
  book_line_spacing: string
  book_drop_caps: boolean
  book_trim_size: string
  // Sidebar
  sidebar_panel_width: number
  sidebar_active_section: string // glyph section id, or '' when closed
}

interface AppStore {
  settings: AppSettings
  showSettings: boolean
  loaded: boolean
  recentProjects: RecentProject[]
  showWelcome: boolean
  showNewUniverse: boolean

  // App-level UI state (status bar, panels, self-contained dialogs).
  // Dialogs entangled with the save pipeline (unsaved-changes warning,
  // new-book wizard) live in bookStore instead.
  statusMessage: string
  setStatusMessage: (msg: string) => void
  leftPanelOpen: boolean
  toggleLeftPanel: () => void
  showMetadata: boolean
  openMetadataDialog: () => void
  closeMetadataDialog: () => void
  showNewChapter: boolean
  newChapterSection: Section | null
  openNewChapterDialog: (section: Section) => void
  closeNewChapterDialog: () => void
  showExportWizard: boolean
  openExportWizard: () => void
  closeExportWizard: () => void
  showChapterHistory: boolean
  openChapterHistory: () => void
  closeChapterHistory: () => void
  bottomToolOpen: boolean
  bottomToolHeight: number
  openStorySearch: () => void
  closeBottomTool: () => void
  setBottomToolHeight: (height: number) => void

  loadSettings: () => Promise<void>
  saveSettings: (patch: Partial<AppSettings>) => Promise<void>
  // Optional target lets callers deep-link a specific settings section
  // (e.g. the Read Aloud rail icon when the voice model needs setup).
  openSettings: (section?: string) => void
  closeSettings: () => void
  settingsInitialSection: string | null
  browseForDirectory: () => Promise<string>
  loadRecentProjects: () => Promise<void>
  addRecentProject: (project: RecentProject) => Promise<void>
  removeRecentProject: (path: string) => Promise<void>
  clearRecentProjects: () => Promise<void>
  setShowWelcome: (show: boolean) => void
  setShowNewUniverse: (show: boolean) => void
}

// saveChain serializes all backend SaveSettings writes so two rapid
// saveSettings calls can never interleave (and drop a setting) on the Go side.
// Each call merges into in-memory state synchronously, then chains the backend
// write; the chained write always snapshots the newest merged state.
let saveChain: Promise<void> = Promise.resolve()

const DEFAULT_SETTINGS: AppSettings = {
  default_author: '',
  default_publisher: '',
  default_copyright: '',
  default_save_dir: '',
  dark_mode: true,
  theme_mode: 'dark',
  auto_theme_use_manual: true,
  auto_theme_dawn: '06:30',
  auto_theme_dusk: '19:00',
  activity_autosave_enabled: true,
  custom_dictionary: [],
  spell_check_enabled: true,
  grammar_check_enabled: true,
  cast_enabled: true,
  story_bible_enabled: true,
  plot_walker_enabled: true,
  analysis_enabled: true,
  analysis_cpu_profile: 'adaptive',
  read_aloud_enabled: false,
  read_aloud_voice: 'af_heart',
  read_aloud_speed: 1.1,
  read_aloud_device: 'native',
  read_aloud_threads: 'auto',
  ai_enabled: false,
  ai_mode: 'claudecode',
  ai_provider: '',
  characters_lane_view: 'grid',
  has_api_key: false,
  ai_debug_logging: false,
  ai_model: '',
  ai_local_endpoint: 'http://localhost:11434/v1',
  ai_local_model: '',
  prose_guide: '',
  book_font: 'Merriweather',
  editor_font_size: 'normal',
  book_font_size: 14,
  book_line_spacing: '1.5',
  book_drop_caps: false,
  book_trim_size: '6x9',
  sidebar_panel_width: 350,
  sidebar_active_section: 'dashboard',
}

export const useAppStore = create<AppStore>((set, get) => ({
  settings: DEFAULT_SETTINGS,
  showSettings: false,
  loaded: false,
  recentProjects: [],
  showWelcome: true,
  showNewUniverse: false,

  statusMessage: 'Ready',
  setStatusMessage: (msg) => set({ statusMessage: msg }),
  leftPanelOpen: true,
  toggleLeftPanel: () => set(s => ({ leftPanelOpen: !s.leftPanelOpen })),
  showMetadata: false,
  openMetadataDialog: () => set({ showMetadata: true }),
  closeMetadataDialog: () => set({ showMetadata: false }),
  showNewChapter: false,
  newChapterSection: null,
  openNewChapterDialog: (section) => set({ showNewChapter: true, newChapterSection: section }),
  closeNewChapterDialog: () => set({ showNewChapter: false, newChapterSection: null }),
  showExportWizard: false,
  openExportWizard: () => set({ showExportWizard: true }),
  closeExportWizard: () => set({ showExportWizard: false }),
  showChapterHistory: false,
  openChapterHistory: () => set({ showChapterHistory: true }),
  closeChapterHistory: () => set({ showChapterHistory: false }),
  bottomToolOpen: false,
  bottomToolHeight: 280,
  openStorySearch: () => set({ bottomToolOpen: true }),
  closeBottomTool: () => set({ bottomToolOpen: false }),
  setBottomToolHeight: (height) => set({ bottomToolHeight: Math.max(170, Math.min(height, 560)) }),

  loadSettings: async () => {
    try {
      const raw = await LoadSettings()
      const settings: AppSettings = {
        ...DEFAULT_SETTINGS,
        ...raw,
        ai_provider: (raw.ai_provider as AppSettings['ai_provider']) ?? '',
        ai_mode: (raw.ai_mode as AppSettings['ai_mode']) || 'claudecode',
        theme_mode: (raw.theme_mode as AppSettings['theme_mode']) || (raw.dark_mode ? 'dark' : 'light'),
        editor_font_size: (raw.editor_font_size as AppSettings['editor_font_size']) || 'normal',
        characters_lane_view: raw.characters_lane_view === 'heat' || raw.characters_lane_view === 'weave'
          ? raw.characters_lane_view
          : 'grid',
        analysis_cpu_profile: ['adaptive', 'gentle', 'balanced', 'fast'].includes(raw.analysis_cpu_profile)
          ? raw.analysis_cpu_profile as AppSettings['analysis_cpu_profile']
          : 'adaptive',
        custom_dictionary: (raw as unknown as Partial<AppSettings>).custom_dictionary ?? [],
        read_aloud_voice: raw.read_aloud_voice || 'af_heart',
        read_aloud_speed: Math.min(1.6, Math.max(0.8, Number(raw.read_aloud_speed) || 1.1)),
        read_aloud_device: 'native',
        read_aloud_threads: raw.read_aloud_threads === 'single' ? 'single' : 'auto',
      }
      set({ settings, loaded: true })
    } catch {
      set({ loaded: true })
    }
  },

  saveSettings: (patch) => {
    // Merge into in-memory state synchronously so callers (and later saves)
    // always see the newest settings immediately.
    set({ settings: { ...get().settings, ...patch } })
    // Chain the backend write. Snapshot the latest merged state INSIDE the
    // chained callback so a queued save always persists the newest values,
    // and writes are ordered — no lost update from interleaving.
    const run = saveChain.then(async () => {
      try {
        await SaveSettings(get().settings)
      } catch (e) {
        console.error('Failed to save settings:', e)
      }
    })
    saveChain = run
    return run
  },

  settingsInitialSection: null,
  openSettings: (section) => set({ showSettings: true, settingsInitialSection: section ?? null }),
  closeSettings: () => set({ showSettings: false, settingsInitialSection: null }),

  browseForDirectory: async () => {
    try {
      return await BrowseForDirectory()
    } catch {
      return ''
    }
  },

  loadRecentProjects: async () => {
    try {
      const projects = await GetRecentProjects()
      set({ recentProjects: projects || [] })
    } catch (e) {
      console.error('Failed to load recent projects:', e)
    }
  },

  addRecentProject: async (project: { type: string; path: string; name: string; lastOpened: string; stats: { books?: number; chapters: number; words: number } }) => {
    try {
      // Convert to Wails model
      const wailsProject = types.RecentProject.createFrom(project)
      await AddRecentProject(wailsProject)
      await get().loadRecentProjects()
    } catch (e) {
      console.error('Failed to add recent project:', e)
    }
  },

  removeRecentProject: async (path: string) => {
    try {
      await RemoveRecentProject(path)
      set({ recentProjects: get().recentProjects.filter(p => p.path !== path) })
    } catch (e) {
      console.error('Failed to remove recent project:', e)
    }
  },

  clearRecentProjects: async () => {
    try {
      await ClearRecentProjects()
      set({ recentProjects: [] })
    } catch (e) {
      console.error('Failed to clear recent projects:', e)
    }
  },

  setShowWelcome: (show: boolean) => set({ showWelcome: show }),

  setShowNewUniverse: (show: boolean) => set({ showNewUniverse: show }),
}))
