// App Settings Dialog - Main container and state management

import { useState, useEffect, useRef } from 'react'
import { useAppStore } from '../../../store/appStore'
import { TestLocalAI, CheckClaudeCode, SetupClaudeCode, OpenClaudeAuth, CheckCodexCLI, SetupCodexCLI, OpenCodexAuth, GetAppVersion, SetAPIKey, ClearAPIKey } from '../../../../wailsjs/go/main/App'
import { EventsOn } from '../../../../wailsjs/runtime/runtime'
import type { types } from '../../../../wailsjs/go/models'
import type { SettingsSection, AIMode, AIProvider, ThemeMode, EditorFontSize, AnalysisCPUProfile, ClaudeCodeSetupStep, TestStatus } from './types'
import ApplicationSection from './ApplicationSection'
import AIStudioSection from './AIStudioSection'
import BookSection from './BookSection'
import PluginsSection from './PluginsSection'
import ReadAloudSection from './ReadAloudSection'
import type { FeatureSettingKey } from '../../../features/registry'

export default function AppSettingsDialog() {
  const { settings, saveSettings, closeSettings, browseForDirectory, loadSettings } = useAppStore()

  const [section, setSection] = useState<SettingsSection>('application')
  const [appVersion, setAppVersion] = useState('')

  useEffect(() => {
    GetAppVersion().then(setAppVersion).catch(() => {})
  }, [])

  // Application state
  const [author, setAuthor]       = useState(settings.default_author)
  const [publisher, setPublisher] = useState(settings.default_publisher)
  const [copyright, setCopyright] = useState(settings.default_copyright)
  const [saveDir, setSaveDir]     = useState(settings.default_save_dir)
  const [dark]                     = useState(settings.dark_mode)
  const [themeMode, setThemeMode] = useState<ThemeMode>(settings.theme_mode)
  const [autoThemeUseManual, setAutoThemeUseManual] = useState(settings.auto_theme_use_manual)
  const [autoThemeDawn, setAutoThemeDawn] = useState(settings.auto_theme_dawn)
  const [autoThemeDusk, setAutoThemeDusk] = useState(settings.auto_theme_dusk)
  const [activityAutoSaveEnabled, setActivityAutoSaveEnabled] = useState(settings.activity_autosave_enabled)
  const [spellCheckEnabled, setSpellCheckEnabled] = useState(settings.spell_check_enabled)
  const [grammarCheckEnabled, setGrammarCheckEnabled] = useState(settings.grammar_check_enabled)
  const [castEnabled, setCastEnabled] = useState(settings.cast_enabled)
  const [analysisEnabled, setAnalysisEnabled] = useState(settings.analysis_enabled)
  const [analysisCPUProfile, setAnalysisCPUProfile] = useState<AnalysisCPUProfile>(settings.analysis_cpu_profile)

  // Read Aloud state
  const [readAloudEnabled, setReadAloudEnabled] = useState(settings.read_aloud_enabled)
  const [readAloudVoice, setReadAloudVoice] = useState(settings.read_aloud_voice)
  const [readAloudSpeed, setReadAloudSpeed] = useState(settings.read_aloud_speed)

  // AI state
  const [aiEnabled, setAiEnabled]         = useState(settings.ai_enabled)
  const [aiMode, setAiMode]               = useState<AIMode>(settings.ai_mode)
  const [provider, setProvider]           = useState<AIProvider>(settings.ai_provider)
  // The stored key never leaves the backend; this field only holds a NEW key
  // the user types, and is sent via SetAPIKey rather than the settings blob.
  const [apiKey, setApiKey]               = useState('')
  const [hasStoredKey, setHasStoredKey]   = useState(settings.has_api_key)
  const [debugLogging, setDebugLogging]   = useState(settings.ai_debug_logging)
  const [model, setModel]                 = useState(settings.ai_model)
  const [localEndpoint, setLocalEndpoint] = useState(settings.ai_local_endpoint)
  const [localModel, setLocalModel]       = useState(settings.ai_local_model)
  const [proseGuide, setProseGuide]       = useState(settings.prose_guide)
  const [testStatus, setTestStatus]       = useState<TestStatus>('idle')
  const [testMsg, setTestMsg]             = useState('')
  const [ccStatus, setCcStatus]           = useState<types.ClaudeCodeStatus | null>(null)
  const [ccChecking, setCcChecking]       = useState(false)
  const [ccSetupStep, setCcSetupStep]     = useState<ClaudeCodeSetupStep>('idle')
  const [ccSetupLog, setCcSetupLog]       = useState<string[]>([])
  const [cxStatus, setCxStatus]           = useState<types.ClaudeCodeStatus | null>(null)
  const [cxChecking, setCxChecking]       = useState(false)
  const [cxSetupStep, setCxSetupStep]     = useState<ClaudeCodeSetupStep>('idle')
  const [cxSetupLog, setCxSetupLog]       = useState<string[]>([])
  // Both CLI setup wizards stream over the shared "setup:progress" channel;
  // this ref routes the events to whichever wizard is currently running.
  const activeSetup = useRef<'cc' | 'cx'>('cc')

  // Book state
  const [bookFont, setBookFont]               = useState(settings.book_font)
  const [editorFontSize, setEditorFontSize]   = useState<EditorFontSize>(settings.editor_font_size)
  const [bookFontSize, setBookFontSize]       = useState(settings.book_font_size)
  const [bookLineSpacing, setBookLineSpacing] = useState(settings.book_line_spacing)
  const [bookDropCaps, setBookDropCaps]       = useState(settings.book_drop_caps)
  const [bookTrimSize, setBookTrimSize]       = useState(settings.book_trim_size)

  // Check CLI status when AI section opened
  useEffect(() => {
    if (section === 'ai' && aiMode === 'claudecode' && !ccStatus && !ccChecking) {
      void handleCheckCC()
    }
    if (section === 'ai' && aiMode === 'codex' && !cxStatus && !cxChecking) {
      void handleCheckCx()
    }
  }, [section, aiMode])

  // Subscribe to backend events for setup progress
  useEffect(() => {
    const offProgress = EventsOn('setup:progress', (msg: string) => {
      const setLog = activeSetup.current === 'cx' ? setCxSetupLog : setCcSetupLog
      const setStep = activeSetup.current === 'cx' ? setCxSetupStep : setCcSetupStep
      if (msg.startsWith('step:')) {
        // step markers don't go in the log
      } else if (msg.startsWith('error:')) {
        setLog(l => [...l, msg.slice(6)])
        setStep('error')
      } else if (msg === 'node:done' || msg === 'claude:done') {
        // no log entry
      } else if (msg === 'step:auth') {
        setStep('auth')
      } else {
        setLog(l => [...l, msg])
      }
    })
    const offAuth = EventsOn('claude:auth_complete', () => {
      void handleCheckCC()
      setCcSetupStep('done')
    })
    const offCxAuth = EventsOn('codex:auth_complete', () => {
      void handleCheckCx()
      setCxSetupStep('done')
    })
    return () => { offProgress(); offAuth(); offCxAuth() }
  }, [])

  async function handleCheckCC() {
    setCcChecking(true)
    try {
      const s = await CheckClaudeCode()
      setCcStatus(s)
    } catch { /* ignore */ }
    setCcChecking(false)
  }

  async function handleSetup() {
    activeSetup.current = 'cc'
    setCcSetupStep('running')
    setCcSetupLog([])
    try {
      const s = await SetupClaudeCode()
      setCcStatus(s)
      if (s.error) {
        setCcSetupStep('error')
      } else if (!s.authenticated) {
        setCcSetupStep('auth')
      } else {
        setCcSetupStep('done')
      }
    } catch (e) {
      setCcSetupLog(l => [...l, String(e)])
      setCcSetupStep('error')
    }
  }

  async function handleOpenAuth() {
    setCcSetupStep('auth-waiting')
    try { await OpenClaudeAuth() } catch { /* ignore */ }
  }

  async function handleCheckCx() {
    setCxChecking(true)
    try {
      const s = await CheckCodexCLI()
      setCxStatus(s)
    } catch { /* ignore */ }
    setCxChecking(false)
  }

  async function handleSetupCx() {
    activeSetup.current = 'cx'
    setCxSetupStep('running')
    setCxSetupLog([])
    try {
      const s = await SetupCodexCLI()
      setCxStatus(s)
      if (s.error) {
        setCxSetupStep('error')
      } else if (!s.authenticated) {
        setCxSetupStep('auth')
      } else {
        setCxSetupStep('done')
      }
    } catch (e) {
      setCxSetupLog(l => [...l, String(e)])
      setCxSetupStep('error')
    }
  }

  async function handleOpenCxAuth() {
    setCxSetupStep('auth-waiting')
    try { await OpenCodexAuth() } catch { /* ignore */ }
  }

  async function handleTestLocal() {
    setTestStatus('testing')
    setTestMsg('')
    try {
      const res = await TestLocalAI(localEndpoint)
      if (res.error) { setTestStatus('error'); setTestMsg(res.error) }
      else { setTestStatus('ok'); setTestMsg('Connected') }
    } catch (e) {
      setTestStatus('error'); setTestMsg(String(e))
    }
  }

  async function handleBrowse() {
    const dir = await browseForDirectory()
    if (dir) setSaveDir(dir)
  }

  async function handleClearKey() {
    try {
      await ClearAPIKey()
      setApiKey('')
      setHasStoredKey(false)
      await loadSettings()
    } catch (e) {
      console.error('Failed to clear API key:', e)
    }
  }

  async function handleSave() {
    const newDarkMode = themeMode === 'auto' ? dark : themeMode === 'dark'
    if (apiKey.trim() !== '') {
      try {
        await SetAPIKey(apiKey.trim())
        setHasStoredKey(true)
      } catch (e) {
        console.error('Failed to store API key:', e)
      }
    }
    await saveSettings({
      default_author: author.trim(),
      default_publisher: publisher.trim(),
      default_copyright: copyright,
      default_save_dir: saveDir.trim(),
      dark_mode: newDarkMode,
      theme_mode: themeMode,
      auto_theme_use_manual: autoThemeUseManual,
      auto_theme_dawn: autoThemeDawn,
      auto_theme_dusk: autoThemeDusk,
      activity_autosave_enabled: activityAutoSaveEnabled,
      spell_check_enabled: spellCheckEnabled,
      grammar_check_enabled: grammarCheckEnabled,
      cast_enabled: castEnabled,
      analysis_enabled: analysisEnabled,
      analysis_cpu_profile: analysisCPUProfile,
      read_aloud_enabled: readAloudEnabled,
      read_aloud_voice: readAloudVoice,
      read_aloud_speed: readAloudSpeed,
      ai_enabled: aiEnabled,
      ai_mode: aiMode,
      ai_provider: provider,
      ai_debug_logging: debugLogging,
      ai_model: model,
      ai_local_endpoint: localEndpoint.trim(),
      ai_local_model: localModel.trim(),
      prose_guide: proseGuide,
      book_font: bookFont,
      editor_font_size: editorFontSize,
      book_font_size: bookFontSize,
      book_line_spacing: bookLineSpacing,
      book_drop_caps: bookDropCaps,
      book_trim_size: bookTrimSize,
    })
    await loadSettings() // refresh has_api_key from the backend
    closeSettings()
  }

  const pluginEnabled = {
    spell_check_enabled: spellCheckEnabled,
    grammar_check_enabled: grammarCheckEnabled,
    cast_enabled: castEnabled,
    analysis_enabled: analysisEnabled,
    ai_enabled: aiEnabled,
    read_aloud_enabled: readAloudEnabled,
  }

  function handlePluginToggle(key: FeatureSettingKey, enabled: boolean) {
    const setters: Record<FeatureSettingKey, (value: boolean) => void> = {
      spell_check_enabled: setSpellCheckEnabled,
      grammar_check_enabled: setGrammarCheckEnabled,
      cast_enabled: setCastEnabled,
      analysis_enabled: setAnalysisEnabled,
      ai_enabled: setAiEnabled,
      read_aloud_enabled: setReadAloudEnabled,
    }
    setters[key](enabled)
  }

  return (
    <div className="dialog-overlay">
      <div className="dialog settings-dialog">
        <div className="settings-dialog-header">
          <span className="dialog-title" style={{ margin: 0 }}>Settings</span>
          <button className="settings-close-btn" onClick={closeSettings} title="Close">
            <svg width="10" height="10" viewBox="0 0 10 10" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round">
              <line x1="1" y1="1" x2="9" y2="9"/><line x1="9" y1="1" x2="1" y2="9"/>
            </svg>
          </button>
        </div>

        <div className="settings-body">
          {/* Left nav */}
          <nav className="settings-nav">
            <button className={`settings-nav-item${section === 'application' ? ' active' : ''}`} onClick={() => setSection('application')}>
              <svg width="13" height="13" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <circle cx="7" cy="5" r="2.4"/><path d="M2 12c0-2.8 2.2-5 5-5s5 2.2 5 5"/>
              </svg>
              Application
            </button>
            <button className={`settings-nav-item${section === 'plugins' ? ' active' : ''}`} onClick={() => setSection('plugins')}>
              <svg width="13" height="13" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.3">
                <path d="M5.5 1v3M8.5 1v3M4 4h6v2.5A3 3 0 0 1 7 9.5v2.75"/><path d="M5 12.25h4"/>
              </svg>
              Plugins
            </button>
            <button className={`settings-nav-item${section === 'ai' ? ' active' : ''}`} onClick={() => setSection('ai')}>
              <svg width="13" height="13" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <rect x="1" y="3" width="12" height="8" rx="2"/><circle cx="4.5" cy="7" r="1"/><circle cx="7" cy="7" r="1"/><circle cx="9.5" cy="7" r="1"/>
              </svg>
              AI Studio
            </button>
            <button className={`settings-nav-item${section === 'readaloud' ? ' active' : ''}`} onClick={() => setSection('readaloud')}>
              <svg width="13" height="13" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <path d="M2 5.5v3h2.2L7.5 11V3L4.2 5.5H2z"/><path d="M9.5 5a2.6 2.6 0 0 1 0 4"/><path d="M11 3.4a5 5 0 0 1 0 7.2"/>
              </svg>
              Read Aloud
            </button>
            <button className={`settings-nav-item${section === 'book' ? ' active' : ''}`} onClick={() => setSection('book')}>
              <svg width="13" height="13" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <rect x="2" y="1" width="10" height="12" rx="1"/><line x1="4.5" y1="4.5" x2="9.5" y2="4.5"/><line x1="4.5" y1="7" x2="9.5" y2="7"/><line x1="4.5" y1="9.5" x2="7.5" y2="9.5"/>
              </svg>
              Book Defaults
            </button>
          </nav>

          {/* Content pane */}
          <div className="settings-content">
            {section === 'application' && (
              <ApplicationSection
                author={author} setAuthor={setAuthor}
                publisher={publisher} setPublisher={setPublisher}
                copyright={copyright} setCopyright={setCopyright}
                saveDir={saveDir} setSaveDir={setSaveDir}
                themeMode={themeMode} setThemeMode={setThemeMode}
                autoThemeUseManual={autoThemeUseManual} setAutoThemeUseManual={setAutoThemeUseManual}
                autoThemeDawn={autoThemeDawn} setAutoThemeDawn={setAutoThemeDawn}
                autoThemeDusk={autoThemeDusk} setAutoThemeDusk={setAutoThemeDusk}
                activityAutoSaveEnabled={activityAutoSaveEnabled} setActivityAutoSaveEnabled={setActivityAutoSaveEnabled}
                analysisCPUProfile={analysisCPUProfile} setAnalysisCPUProfile={setAnalysisCPUProfile}
                onBrowse={handleBrowse}
              />
            )}

            {section === 'plugins' && (
              <PluginsSection enabled={pluginEnabled} onToggle={handlePluginToggle} />
            )}

            {section === 'ai' && (
              <AIStudioSection
                aiEnabled={aiEnabled} setAiEnabled={setAiEnabled}
                aiMode={aiMode} setAiMode={setAiMode}
                provider={provider} setProvider={setProvider}
                apiKey={apiKey} setApiKey={setApiKey}
                hasStoredKey={hasStoredKey} onClearKey={handleClearKey}
                debugLogging={debugLogging} setDebugLogging={setDebugLogging}
                model={model} setModel={setModel}
                localEndpoint={localEndpoint} setLocalEndpoint={setLocalEndpoint}
                localModel={localModel} setLocalModel={setLocalModel}
                proseGuide={proseGuide} setProseGuide={setProseGuide}
                ccStatus={ccStatus} ccChecking={ccChecking}
                ccSetupStep={ccSetupStep} ccSetupLog={ccSetupLog}
                onCheckCC={handleCheckCC} onSetup={handleSetup} onOpenAuth={handleOpenAuth}
                cxStatus={cxStatus} cxChecking={cxChecking}
                cxSetupStep={cxSetupStep} cxSetupLog={cxSetupLog}
                onCheckCx={handleCheckCx} onSetupCx={handleSetupCx} onOpenCxAuth={handleOpenCxAuth}
                testStatus={testStatus} testMsg={testMsg} onTestLocal={handleTestLocal}
              />
            )}

            {section === 'readaloud' && (
              <ReadAloudSection
                readAloudEnabled={readAloudEnabled}
                voice={readAloudVoice} setVoice={setReadAloudVoice}
                speed={readAloudSpeed} setSpeed={setReadAloudSpeed}
              />
            )}

            {section === 'book' && (
              <BookSection
                bookFont={bookFont} setBookFont={setBookFont}
                editorFontSize={editorFontSize} setEditorFontSize={setEditorFontSize}
                bookFontSize={bookFontSize} setBookFontSize={setBookFontSize}
                bookLineSpacing={bookLineSpacing} setBookLineSpacing={setBookLineSpacing}
                bookDropCaps={bookDropCaps} setBookDropCaps={setBookDropCaps}
                bookTrimSize={bookTrimSize} setBookTrimSize={setBookTrimSize}
              />
            )}
          </div>
        </div>

        <div className="settings-dialog-footer">
          <span className="settings-version">Draftline v{appVersion}</span>
          <div className="settings-dialog-buttons">
            <button className="dialog-btn" onClick={closeSettings}>Cancel</button>
            <button className="dialog-btn primary" onClick={handleSave}>Save Settings</button>
          </div>
        </div>
      </div>
    </div>
  )
}
