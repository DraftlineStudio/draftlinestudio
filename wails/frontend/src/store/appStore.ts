import { create } from 'zustand'
import { LoadSettings, SaveSettings, BrowseForDirectory } from '../../wailsjs/go/main/App'

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
  // AI
  ai_enabled: boolean
  ai_mode: 'claudecode' | 'api' | 'local'
  ai_provider: 'claude' | 'openai' | ''
  ai_api_key: string
  ai_model: string
  ai_local_endpoint: string
  ai_local_model: string
  prose_guide: string
  // Book defaults
  book_font: string
  book_font_size: number
  book_line_spacing: string
  book_drop_caps: boolean
  book_trim_size: string
}

interface AppStore {
  settings: AppSettings
  showSettings: boolean
  loaded: boolean
  loadSettings: () => Promise<void>
  saveSettings: (patch: Partial<AppSettings>) => Promise<void>
  openSettings: () => void
  closeSettings: () => void
  browseForDirectory: () => Promise<string>
}

const DEFAULT_SETTINGS: AppSettings = {
  default_author: '',
  default_publisher: '',
  default_copyright: '',
  default_save_dir: '',
  dark_mode: true,
  theme_mode: 'dark',
  auto_theme_use_manual: false,
  auto_theme_dawn: '06:30',
  auto_theme_dusk: '19:00',
  ai_enabled: true,
  ai_mode: 'claudecode',
  ai_provider: '',
  ai_api_key: '',
  ai_model: '',
  ai_local_endpoint: 'http://localhost:11434/v1',
  ai_local_model: '',
  prose_guide: '',
  book_font: 'Merriweather',
  book_font_size: 12,
  book_line_spacing: '1.5',
  book_drop_caps: false,
  book_trim_size: '6x9',
}

export const useAppStore = create<AppStore>((set, get) => ({
  settings: DEFAULT_SETTINGS,
  showSettings: false,
  loaded: false,

  loadSettings: async () => {
    try {
      const raw = await LoadSettings()
      const settings: AppSettings = {
        ...DEFAULT_SETTINGS,
        ...raw,
        ai_provider: (raw.ai_provider as AppSettings['ai_provider']) ?? '',
        ai_mode: (raw.ai_mode as AppSettings['ai_mode']) || 'claudecode',
        theme_mode: (raw.theme_mode as AppSettings['theme_mode']) || (raw.dark_mode ? 'dark' : 'light'),
      }
      set({ settings, loaded: true })
    } catch {
      set({ loaded: true })
    }
  },

  saveSettings: async (patch) => {
    const next = { ...get().settings, ...patch }
    set({ settings: next })
    try {
      await SaveSettings(next)
    } catch (e) {
      console.error('Failed to save settings:', e)
    }
  },

  openSettings: () => set({ showSettings: true }),
  closeSettings: () => set({ showSettings: false }),

  browseForDirectory: async () => {
    try {
      return await BrowseForDirectory()
    } catch {
      return ''
    }
  },
}))
