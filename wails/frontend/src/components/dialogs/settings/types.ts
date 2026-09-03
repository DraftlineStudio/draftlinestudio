// Settings Dialog Types

import type { types } from '../../../../wailsjs/go/models'

export type SettingsSection = 'application' | 'plugins' | 'ai' | 'readaloud' | 'book'

export type AIMode = 'claudecode' | 'codex' | 'api' | 'local'
export type AIProvider = 'claude' | 'openai' | 'gemini' | 'grok' | ''
export type ThemeMode = 'light' | 'dark' | 'auto'
export type EditorFontSize = 'small' | 'normal' | 'large'
export type AnalysisCPUProfile = 'adaptive' | 'gentle' | 'balanced' | 'fast'

export type ClaudeCodeSetupStep = 'idle' | 'running' | 'auth' | 'auth-waiting' | 'done' | 'error'
export type TestStatus = 'idle' | 'testing' | 'ok' | 'error'

export interface ApplicationSectionProps {
  author: string
  setAuthor: (v: string) => void
  publisher: string
  setPublisher: (v: string) => void
  copyright: string
  setCopyright: (v: string) => void
  saveDir: string
  setSaveDir: (v: string) => void
  themeMode: ThemeMode
  setThemeMode: (v: ThemeMode) => void
  autoThemeUseManual: boolean
  setAutoThemeUseManual: (v: boolean) => void
  autoThemeDawn: string
  setAutoThemeDawn: (v: string) => void
  autoThemeDusk: string
  setAutoThemeDusk: (v: string) => void
  activityAutoSaveEnabled: boolean
  setActivityAutoSaveEnabled: (v: boolean) => void
  analysisCPUProfile: AnalysisCPUProfile
  setAnalysisCPUProfile: (v: AnalysisCPUProfile) => void
  onBrowse: () => void
}

export interface AIStudioSectionProps {
  aiEnabled: boolean
  setAiEnabled: (v: boolean) => void
  aiMode: AIMode
  setAiMode: (v: AIMode) => void
  provider: AIProvider
  setProvider: (v: AIProvider) => void
  apiKey: string
  setApiKey: (v: string) => void
  hasStoredKey: boolean
  onClearKey: () => void
  debugLogging: boolean
  setDebugLogging: (v: boolean) => void
  model: string
  setModel: (v: string) => void
  localEndpoint: string
  setLocalEndpoint: (v: string) => void
  localModel: string
  setLocalModel: (v: string) => void
  proseGuide: string
  setProseGuide: (v: string) => void
  // Claude Code state
  ccStatus: types.ClaudeCodeStatus | null
  ccChecking: boolean
  ccSetupStep: ClaudeCodeSetupStep
  ccSetupLog: string[]
  onCheckCC: () => void
  onSetup: () => void
  onOpenAuth: () => void
  // Codex CLI state (ChatGPT accounts) — same lifecycle as Claude Code
  cxStatus: types.ClaudeCodeStatus | null
  cxChecking: boolean
  cxSetupStep: ClaudeCodeSetupStep
  cxSetupLog: string[]
  onCheckCx: () => void
  onSetupCx: () => void
  onOpenCxAuth: () => void
  // Test state
  testStatus: TestStatus
  testMsg: string
  onTestLocal: () => void
}

export interface ReadAloudSectionProps {
  voice: string
  setVoice: (v: string) => void
  speed: number
  setSpeed: (v: number) => void
  threads: 'single' | 'auto'
  setThreads: (v: 'single' | 'auto') => void
}

export interface BookSectionProps {
  bookFont: string
  setBookFont: (v: string) => void
  editorFontSize: EditorFontSize
  setEditorFontSize: (v: EditorFontSize) => void
  bookFontSize: number
  setBookFontSize: (v: number) => void
  bookLineSpacing: string
  setBookLineSpacing: (v: string) => void
  bookDropCaps: boolean
  setBookDropCaps: (v: boolean) => void
  bookTrimSize: string
  setBookTrimSize: (v: string) => void
}
