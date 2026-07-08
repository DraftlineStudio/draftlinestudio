import { useState, useEffect } from 'react'
import { useAppStore } from '../../store/appStore'
import { useBookStore } from '../../store/bookStore'
import { TestLocalAI, CheckClaudeCode, SetupClaudeCode, OpenClaudeAuth, GetAppVersion } from '../../../wailsjs/go/main/App'
import { EventsOn } from '../../../wailsjs/runtime/runtime'
import type { main } from '../../../wailsjs/go/models'

type SettingsSection = 'application' | 'ai' | 'book'

const CLAUDE_MODELS = [
  { value: 'claude-opus-4-6',           label: 'Claude Opus 4.6 (most capable)' },
  { value: 'claude-sonnet-4-6',         label: 'Claude Sonnet 4.6 (recommended)' },
  { value: 'claude-haiku-4-5-20251001', label: 'Claude Haiku 4.5 (fastest)' },
]

const OPENAI_MODELS = [
  { value: 'gpt-4o',      label: 'GPT-4o (recommended)' },
  { value: 'gpt-4o-mini', label: 'GPT-4o Mini (faster)' },
  { value: 'o3',          label: 'o3 (reasoning)' },
]

const GEMINI_MODELS = [
  { value: 'gemini-1.5-pro',   label: 'Gemini 1.5 Pro (recommended)' },
  { value: 'gemini-1.5-flash', label: 'Gemini 1.5 Flash (faster)' },
  { value: 'gemini-2.0-flash', label: 'Gemini 2.0 Flash (latest)' },
]

const GROK_MODELS = [
  { value: 'grok-2',      label: 'Grok 2 (recommended)' },
  { value: 'grok-beta',   label: 'Grok Beta' },
]

const BOOK_FONTS = [
  'Merriweather', 'EB Garamond', 'Lora', 'Palatino Linotype', 'Georgia', 'Times New Roman',
]

const TRIM_SIZES = [
  { value: '6x9',     label: '6″ × 9″ — Standard trade paperback' },
  { value: '5.5x8.5', label: '5.5″ × 8.5″ — Digest / literary fiction' },
  { value: '5x8',     label: '5″ × 8″ — Compact trade' },
  { value: '7x10',    label: '7″ × 10″ — Textbook / reference' },
  { value: 'A5',      label: 'A5 — 148 × 210 mm' },
  { value: 'A4',      label: 'A4 — 210 × 297 mm' },
]

export default function AppSettingsDialog() {
  const { settings, saveSettings, closeSettings, browseForDirectory } = useAppStore()
  const { setDarkMode } = useBookStore()

  const [section, setSection] = useState<SettingsSection>('application')
  const [appVersion, setAppVersion] = useState('')

  useEffect(() => {
    GetAppVersion().then(setAppVersion).catch(() => {})
  }, [])

  // Application
  const [author, setAuthor]       = useState(settings.default_author)
  const [publisher, setPublisher] = useState(settings.default_publisher)
  const [copyright, setCopyright] = useState(settings.default_copyright)
  const [saveDir, setSaveDir]     = useState(settings.default_save_dir)
  const [dark, setDark]           = useState(settings.dark_mode)
  const [themeMode, setThemeMode] = useState<'light' | 'dark' | 'auto'>(settings.theme_mode)
  const [autoThemeUseManual, setAutoThemeUseManual] = useState(settings.auto_theme_use_manual)
  const [autoThemeDawn, setAutoThemeDawn] = useState(settings.auto_theme_dawn)
  const [autoThemeDusk, setAutoThemeDusk] = useState(settings.auto_theme_dusk)

  // AI
  const [aiEnabled, setAiEnabled]         = useState(settings.ai_enabled)
  const [showAiTab, setShowAiTab]         = useState(settings.show_ai_tab)
  const [aiMode, setAiMode]               = useState(settings.ai_mode)
  const [provider, setProvider]           = useState(settings.ai_provider)
  const [apiKey, setApiKey]               = useState(settings.ai_api_key)
  const [model, setModel]                 = useState(settings.ai_model)
  const [localEndpoint, setLocalEndpoint] = useState(settings.ai_local_endpoint)
  const [localModel, setLocalModel]       = useState(settings.ai_local_model)
  const [proseGuide, setProseGuide]       = useState(settings.prose_guide)
  const [showKey, setShowKey]             = useState(false)
  const [testStatus, setTestStatus]       = useState<'idle' | 'testing' | 'ok' | 'error'>('idle')
  const [testMsg, setTestMsg]             = useState('')
  const [ccStatus, setCcStatus]           = useState<main.ClaudeCodeStatus | null>(null)
  const [ccChecking, setCcChecking]       = useState(false)
  const [ccSetupStep, setCcSetupStep]     = useState<'idle'|'running'|'auth'|'auth-waiting'|'done'|'error'>('idle')
  const [ccSetupLog, setCcSetupLog]       = useState<string[]>([])

  // Book
  const [bookFont, setBookFont]               = useState(settings.book_font)
  const [editorFontSize, setEditorFontSize]   = useState(settings.editor_font_size)
  const [bookFontSize, setBookFontSize]       = useState(settings.book_font_size)
  const [bookLineSpacing, setBookLineSpacing] = useState(settings.book_line_spacing)
  const [bookDropCaps, setBookDropCaps]       = useState(settings.book_drop_caps)
  const [bookTrimSize, setBookTrimSize]       = useState(settings.book_trim_size)

  const modelOptions = provider === 'claude' ? CLAUDE_MODELS
    : provider === 'openai' ? OPENAI_MODELS
    : provider === 'gemini' ? GEMINI_MODELS
    : provider === 'grok' ? GROK_MODELS
    : []

  // Check Claude Code status when the AI section is opened or mode switches
  useEffect(() => {
    if (section === 'ai' && aiMode === 'claudecode' && !ccStatus && !ccChecking) {
      handleCheckCC()
    }
  }, [section, aiMode])

  async function handleCheckCC() {
    setCcChecking(true)
    try {
      const s = await CheckClaudeCode()
      setCcStatus(s)
    } catch { /* ignore */ }
    setCcChecking(false)
  }

  // Subscribe to backend events for setup progress and auth completion
  useEffect(() => {
    const offProgress = EventsOn('setup:progress', (msg: string) => {
      if (msg.startsWith('step:')) {
        // step markers don't go in the log
      } else if (msg.startsWith('error:')) {
        setCcSetupLog(l => [...l, msg.slice(6)])
        setCcSetupStep('error')
      } else if (msg === 'node:done') {
        // no log entry, just a step completion marker
      } else if (msg === 'claude:done') {
        // no log entry
      } else if (msg === 'step:auth') {
        setCcSetupStep('auth')
      } else {
        setCcSetupLog(l => [...l, msg])
      }
    })
    const offAuth = EventsOn('claude:auth_complete', () => {
      handleCheckCC()
      setCcSetupStep('done')
    })
    return () => { offProgress(); offAuth() }
  }, [])

  async function handleSetup() {
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

  function handleProviderChange(p: 'claude' | 'openai' | 'gemini' | 'grok' | '') {
    setProvider(p)
    const defaultModels: Record<string, string> = {
      claude: 'claude-sonnet-4-6',
      openai: 'gpt-4o',
      gemini: 'gemini-1.5-pro',
      grok: 'grok-2',
    }
    setModel(defaultModels[p] || '')
  }

  function handleModeChange(m: typeof aiMode) {
    setAiMode(m)
    if (m === 'claudecode' && !ccStatus && !ccChecking) handleCheckCC()
    // Set a sensible default model for the new mode
    if (m === 'claudecode' && !model) setModel('claude-sonnet-4-6')
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

  async function handleSave() {
    const newDarkMode = themeMode === 'auto' ? dark : themeMode === 'dark'
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
      ai_enabled: aiEnabled,
      show_ai_tab: showAiTab,
      ai_mode: aiMode,
      ai_provider: provider,
      ai_api_key: apiKey.trim(),
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
    setDarkMode(newDarkMode)
    closeSettings()
  }

  async function handleBrowse() {
    const dir = await browseForDirectory()
    if (dir) setSaveDir(dir)
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
          {/* ── Left nav ── */}
          <nav className="settings-nav">
            <button className={`settings-nav-item${section === 'application' ? ' active' : ''}`} onClick={() => setSection('application')}>
              <svg width="13" height="13" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <circle cx="7" cy="5" r="2.4"/><path d="M2 12c0-2.8 2.2-5 5-5s5 2.2 5 5"/>
              </svg>
              Application
            </button>
            <button className={`settings-nav-item${section === 'ai' ? ' active' : ''}`} onClick={() => setSection('ai')}>
              <svg width="13" height="13" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <rect x="1" y="3" width="12" height="8" rx="2"/><circle cx="4.5" cy="7" r="1"/><circle cx="7" cy="7" r="1"/><circle cx="9.5" cy="7" r="1"/>
              </svg>
              AI Studio
            </button>
            <button className={`settings-nav-item${section === 'book' ? ' active' : ''}`} onClick={() => setSection('book')}>
              <svg width="13" height="13" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <rect x="2" y="1" width="10" height="12" rx="1"/><line x1="4.5" y1="4.5" x2="9.5" y2="4.5"/><line x1="4.5" y1="7" x2="9.5" y2="7"/><line x1="4.5" y1="9.5" x2="7.5" y2="9.5"/>
              </svg>
              Book Defaults
            </button>
          </nav>

          {/* ── Content pane ── */}
          <div className="settings-content">

            {/* ════ APPLICATION ════ */}
            {section === 'application' && <>
              <div className="settings-section-label" style={{ marginTop: 0 }}>Identity</div>
              <div className="dialog-field">
                <label className="dialog-label">Default Author Name</label>
                <input className="dialog-input" value={author} onChange={e => setAuthor(e.target.value)} placeholder="Your name" autoFocus />
                <div className="settings-hint">Pre-filled when creating a new book.</div>
              </div>
              <div className="dialog-field">
                <label className="dialog-label">Default Publisher</label>
                <input className="dialog-input" value={publisher} onChange={e => setPublisher(e.target.value)} placeholder="Publisher or imprint name" />
              </div>

              <div className="settings-section-label">Copyright Template</div>
              <div className="dialog-field">
                <label className="dialog-label">Default Copyright Text</label>
                <textarea
                  className="dialog-input settings-copyright-textarea"
                  value={copyright}
                  onChange={e => setCopyright(e.target.value)}
                  placeholder={"Copyright © [YEAR] [AUTHOR]. All rights reserved.\n\nNo part of this publication may be reproduced..."}
                  rows={5}
                />
                <div className="settings-hint">Inserted into the Copyright page of every new book. Use [YEAR] and [AUTHOR] as placeholders.</div>
              </div>

              <div className="settings-section-label">Interface</div>
              <div className="dialog-field">
                <label className="dialog-label">Theme</label>
                <div className="settings-theme-row">
                  <button className={`settings-theme-btn${themeMode === 'light' ? ' active' : ''}`} onClick={() => setThemeMode('light')}>
                    <svg width="11" height="11" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
                      <circle cx="7" cy="7" r="2.8"/>
                      <line x1="7" y1="1" x2="7" y2="2.4"/><line x1="7" y1="11.6" x2="7" y2="13"/>
                      <line x1="1" y1="7" x2="2.4" y2="7"/><line x1="11.6" y1="7" x2="13" y2="7"/>
                      <line x1="2.9" y1="2.9" x2="3.9" y2="3.9"/><line x1="10.1" y1="10.1" x2="11.1" y2="11.1"/>
                      <line x1="11.1" y1="2.9" x2="10.1" y2="3.9"/><line x1="3.9" y1="10.1" x2="2.9" y2="11.1"/>
                    </svg>
                    Light
                  </button>
                  <button className={`settings-theme-btn${themeMode === 'dark' ? ' active' : ''}`} onClick={() => setThemeMode('dark')}>
                    <svg width="11" height="11" viewBox="0 0 14 14" fill="currentColor">
                      <path d="M7 1a6 6 0 1 0 0 12A6 6 0 0 0 7 1zm0 1.5A4.5 4.5 0 1 1 7 11.5V2.5z"/>
                    </svg>
                    Dark
                  </button>
                  <button className={`settings-theme-btn${themeMode === 'auto' ? ' active' : ''}`} onClick={() => setThemeMode('auto')}>
                    <svg width="11" height="11" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                      <circle cx="7" cy="7" r="5.5"/>
                      <path d="M7 1.5V7l3.5 2"/>
                    </svg>
                    Auto
                  </button>
                </div>
              </div>
              {themeMode === 'auto' && (
                <div className="settings-auto-theme">
                  <div className="settings-hint" style={{ marginBottom: 12 }}>
                    Automatically switches between light and dark themes based on time of day.
                  </div>
                  <div className="dialog-field">
                    <label className="dialog-label" style={{ marginBottom: 8 }}>Time Source</label>
                    <label className="settings-toggle">
                      <input type="checkbox" checked={autoThemeUseManual} onChange={e => setAutoThemeUseManual(e.target.checked)} />
                      <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
                      <span className="settings-toggle-label">{autoThemeUseManual ? 'Manual times' : 'Automatic (based on location)'}</span>
                    </label>
                  </div>
                  {autoThemeUseManual && (
                    <div className="settings-two-col" style={{ marginTop: 12 }}>
                      <div className="dialog-field">
                        <label className="dialog-label">Dawn (light mode starts)</label>
                        <input
                          className="dialog-input"
                          type="time"
                          value={autoThemeDawn}
                          onChange={e => setAutoThemeDawn(e.target.value)}
                        />
                      </div>
                      <div className="dialog-field">
                        <label className="dialog-label">Dusk (dark mode starts)</label>
                        <input
                          className="dialog-input"
                          type="time"
                          value={autoThemeDusk}
                          onChange={e => setAutoThemeDusk(e.target.value)}
                        />
                      </div>
                    </div>
                  )}
                  {!autoThemeUseManual && (
                    <div className="settings-hint" style={{ marginTop: 8 }}>
                      Sunrise and sunset times are fetched automatically when the app starts.
                    </div>
                  )}
                </div>
              )}
              <div className="dialog-field">
                <label className="dialog-label">Default Save Location</label>
                <div className="settings-path-row">
                  <input className="dialog-input" value={saveDir} onChange={e => setSaveDir(e.target.value)} placeholder="Leave blank to use system default" />
                  <button className="dialog-btn settings-browse-btn" onClick={handleBrowse}>Browse…</button>
                </div>
              </div>
            </>}

            {/* ════ AI STUDIO ════ */}
            {section === 'ai' && <>
              <div className="settings-section-label" style={{ marginTop: 0 }}>
                AI Features
                <label className="settings-toggle" title="Enable or disable AI features">
                  <input type="checkbox" checked={aiEnabled} onChange={e => setAiEnabled(e.target.checked)} />
                  <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
                  <span className="settings-toggle-label">{aiEnabled ? 'Enabled' : 'Disabled'}</span>
                </label>
              </div>

              {aiEnabled && <>
                <div className="dialog-field" style={{ marginBottom: 12 }}>
                  <label className="dialog-checkbox-label">
                    <input
                      type="checkbox"
                      checked={showAiTab}
                      onChange={e => setShowAiTab(e.target.checked)}
                    />
                    <span>Show AI tab in sidebar</span>
                  </label>
                  <p className="settings-hint" style={{ marginTop: 4 }}>
                    Display the AI Studio icon in the sidebar glyph bar.
                  </p>
                </div>
                <div className="dialog-field">
                  <label className="dialog-label">AI Source</label>
                  <div className="settings-theme-row">
                    <button className={`settings-theme-btn${aiMode === 'claudecode' ? ' active' : ''}`} onClick={() => handleModeChange('claudecode')}>
                      Claude Code
                    </button>
                    <button className={`settings-theme-btn${aiMode === 'api' ? ' active' : ''}`} onClick={() => handleModeChange('api')}>
                      API Key
                    </button>
                    <button className={`settings-theme-btn${aiMode === 'local' ? ' active' : ''}`} onClick={() => handleModeChange('local')}>
                      Local Model
                    </button>
                  </div>
                </div>

                {/* ── Claude Code ── */}
                {aiMode === 'claudecode' && <>
                  <div className="settings-cc-card">
                    <div className="settings-cc-header">
                      <span className="settings-cc-title">Claude Code</span>
                      {ccChecking && <span className="settings-cc-badge checking">Checking…</span>}
                      {!ccChecking && ccStatus?.installed && ccStatus.authenticated && ccSetupStep !== 'running' && (
                        <span className="settings-cc-badge ok">✓ Ready — {ccStatus.version}</span>
                      )}
                      {!ccChecking && ccStatus?.installed && !ccStatus.authenticated && ccSetupStep === 'idle' && (
                        <span className="settings-cc-badge warn">Not signed in</span>
                      )}
                      {!ccChecking && ccStatus && !ccStatus.installed && ccSetupStep === 'idle' && (
                        <span className="settings-cc-badge error">Not installed</span>
                      )}
                      {(ccSetupStep === 'running') && (
                        <span className="settings-cc-badge checking">Setting up…</span>
                      )}
                      <button className="settings-cc-recheck" onClick={handleCheckCC} title="Re-check" disabled={ccSetupStep === 'running'}>
                        <svg width="11" height="11" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
                          <path d="M10.5 2A5 5 0 1 0 11 6.5"/><polyline points="10.5 1 10.5 3.5 8 3.5"/>
                        </svg>
                      </button>
                    </div>
                    <p className="settings-cc-desc">
                      Uses your Claude.ai account — no separate API key needed.
                      AI rewrites are charged to your Claude subscription.
                    </p>

                    {/* ── Setup wizard (not installed or setup in progress) ── */}
                    {ccSetupStep !== 'done' && !(ccStatus?.installed && ccStatus.authenticated) && (
                      <div className="settings-cc-setup">
                        {/* Step list */}
                        <div className="settings-cc-steps">
                          <div className={`settings-cc-step ${ccStatus?.installed ? 'done' : ccSetupStep === 'running' ? 'active' : ''}`}>
                            <span className="settings-cc-step-icon">
                              {ccStatus?.installed ? '✓' : ccSetupStep === 'running' ? '⟳' : '○'}
                            </span>
                            <span>Install runtime &amp; Claude Code CLI</span>
                          </div>
                          <div className={`settings-cc-step ${ccStatus?.authenticated ? 'done' : (ccSetupStep === 'auth' || ccSetupStep === 'auth-waiting') ? 'active' : ''}`}>
                            <span className="settings-cc-step-icon">
                              {ccStatus?.authenticated ? '✓' : (ccSetupStep === 'auth' || ccSetupStep === 'auth-waiting') ? '⟳' : '○'}
                            </span>
                            <span>Sign in with your Claude account</span>
                          </div>
                        </div>

                        {/* Progress log */}
                        {ccSetupLog.length > 0 && (
                          <div className="settings-cc-log">
                            {ccSetupLog.map((line, i) => <div key={i}>{line}</div>)}
                          </div>
                        )}

                        {/* Actions */}
                        {ccSetupStep === 'idle' && (
                          <button className="dialog-btn primary" style={{ marginTop: 12 }} onClick={handleSetup}>
                            Set up Claude Code automatically
                          </button>
                        )}
                        {ccSetupStep === 'running' && (
                          <p className="settings-hint" style={{ marginTop: 8, marginBottom: 0 }}>
                            Downloading and installing — this takes about a minute…
                          </p>
                        )}
                        {ccSetupStep === 'auth' && (
                          <div style={{ marginTop: 12 }}>
                            <p className="settings-hint" style={{ marginBottom: 8 }}>
                              Click below to sign in with your Claude.ai account. A browser window will open — complete the login, then come back here.
                            </p>
                            <button className="dialog-btn primary" onClick={handleOpenAuth}>
                              Sign in with Claude.ai…
                            </button>
                          </div>
                        )}
                        {ccSetupStep === 'auth-waiting' && (
                          <div style={{ marginTop: 12 }}>
                            <p className="settings-hint" style={{ marginBottom: 8 }}>
                              Browser opened — complete the sign-in, then click Verify below.
                            </p>
                            <button className="dialog-btn" onClick={handleCheckCC}>
                              Verify sign-in
                            </button>
                          </div>
                        )}
                        {ccSetupStep === 'error' && (
                          <button className="dialog-btn" style={{ marginTop: 12 }} onClick={handleSetup}>
                            Retry
                          </button>
                        )}
                      </div>
                    )}
                  </div>

                  {(ccStatus?.installed && ccStatus.authenticated) && (
                    <div className="dialog-field" style={{ marginTop: 12 }}>
                      <label className="dialog-label">Model</label>
                      <select className="dialog-select" value={model || 'claude-sonnet-4-6'} onChange={e => setModel(e.target.value)}>
                        {CLAUDE_MODELS.map(m => <option key={m.value} value={m.value}>{m.label}</option>)}
                      </select>
                    </div>
                  )}
                </>}

                {/* ── API Key ── */}
                {aiMode === 'api' && <>
                  <div className="dialog-field">
                    <label className="dialog-label">Provider</label>
                    <div className="settings-provider-row">
                      <button className={`settings-provider-btn${provider === 'claude' ? ' active' : ''}`} onClick={() => handleProviderChange('claude')}>Claude</button>
                      <button className={`settings-provider-btn${provider === 'openai' ? ' active' : ''}`} onClick={() => handleProviderChange('openai')}>OpenAI</button>
                      <button className={`settings-provider-btn${provider === 'gemini' ? ' active' : ''}`} onClick={() => handleProviderChange('gemini')}>Gemini</button>
                      <button className={`settings-provider-btn${provider === 'grok' ? ' active' : ''}`} onClick={() => handleProviderChange('grok')}>Grok</button>
                    </div>
                  </div>
                  {provider !== '' && <>
                    <div className="dialog-field">
                      <label className="dialog-label">Model</label>
                      <select className="dialog-select" value={model} onChange={e => setModel(e.target.value)}>
                        {modelOptions.map(m => <option key={m.value} value={m.value}>{m.label}</option>)}
                      </select>
                    </div>
                    <div className="dialog-field">
                      <label className="dialog-label">API Key</label>
                      <div className="settings-path-row">
                        <input
                          className="dialog-input"
                          type={showKey ? 'text' : 'password'}
                          value={apiKey}
                          onChange={e => setApiKey(e.target.value)}
                          placeholder={provider === 'claude' ? 'sk-ant-...' : 'sk-...'}
                          autoComplete="off"
                        />
                        <button className="dialog-btn settings-browse-btn" onClick={() => setShowKey(v => !v)}>
                          {showKey ? 'Hide' : 'Show'}
                        </button>
                      </div>
                      <div className="settings-hint">Stored locally in your app settings file — never transmitted to Draftline servers.</div>
                    </div>
                  </>}
                </>}

                {/* ── Local Model ── */}
                {aiMode === 'local' && <>
                  <div className="settings-local-info">
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                      <circle cx="7" cy="7" r="6"/><line x1="7" y1="5" x2="7" y2="7.5"/><circle cx="7" cy="9.5" r="0.6" fill="currentColor" stroke="none"/>
                    </svg>
                    Compatible with <strong>Ollama</strong>, <strong>LM Studio</strong>, and any OpenAI-compatible local server.
                  </div>
                  <div className="dialog-field">
                    <label className="dialog-label">API Endpoint</label>
                    <input
                      className="dialog-input"
                      value={localEndpoint}
                      onChange={e => { setLocalEndpoint(e.target.value); setTestStatus('idle') }}
                      placeholder="http://localhost:11434/v1"
                    />
                    <div className="settings-hint">Default: Ollama at http://localhost:11434/v1</div>
                  </div>
                  <div className="dialog-field">
                    <label className="dialog-label">Model Name</label>
                    <input
                      className="dialog-input"
                      value={localModel}
                      onChange={e => setLocalModel(e.target.value)}
                      placeholder="e.g. llama3, mistral, phi3"
                    />
                  </div>
                  <div className="settings-local-test-row">
                    <button className="dialog-btn" onClick={handleTestLocal} disabled={testStatus === 'testing'}>
                      {testStatus === 'testing' ? 'Testing…' : 'Test Connection'}
                    </button>
                    {testStatus === 'ok' && <span className="settings-test-ok">✓ {testMsg}</span>}
                    {testStatus === 'error' && <span className="settings-test-error">{testMsg}</span>}
                  </div>
                </>}

                {/* Prose guide — always visible */}
                <div className="settings-section-label">Prose Style Guide</div>
                <div className="dialog-field">
                  <label className="dialog-label">Writing Examples</label>
                  <textarea
                    className="dialog-input settings-prose-textarea"
                    value={proseGuide}
                    onChange={e => setProseGuide(e.target.value)}
                    placeholder={"Paste 2–5 paragraphs of writing in your target style here.\nThe AI will study these before rewriting your prose.\n\nExamples can be your own work, a favourite author, or a specific voice you want to emulate."}
                    rows={7}
                  />
                  <div className="settings-hint">The AI will match the rhythm, vocabulary, and sentence structure of these examples when rewriting.</div>
                </div>
              </>}
            </>}

            {/* ════ BOOK DEFAULTS ════ */}
            {section === 'book' && <>
              <div className="settings-section-label" style={{ marginTop: 0 }}>Editor Display</div>
              <div className="dialog-field">
                <label className="dialog-label">Font</label>
                <select className="dialog-select" value={bookFont} onChange={e => setBookFont(e.target.value)}>
                  {BOOK_FONTS.map(f => <option key={f} value={f}>{f}</option>)}
                </select>
              </div>
              <div className="dialog-field">
                <label className="dialog-label">Text Size</label>
                <div className="settings-theme-row">
                  <button className={`settings-theme-btn${editorFontSize === 'small' ? ' active' : ''}`} onClick={() => setEditorFontSize('small')}>
                    Small (12pt)
                  </button>
                  <button className={`settings-theme-btn${editorFontSize === 'normal' ? ' active' : ''}`} onClick={() => setEditorFontSize('normal')}>
                    Normal (14pt)
                  </button>
                  <button className={`settings-theme-btn${editorFontSize === 'large' ? ' active' : ''}`} onClick={() => setEditorFontSize('large')}>
                    Large (16pt)
                  </button>
                </div>
                <div className="settings-hint">Controls how text appears in the editor. Does not affect exported files.</div>
              </div>

              <div className="settings-section-label">Export Settings</div>
              <div className="settings-two-col">
                <div className="dialog-field">
                  <label className="dialog-label">Font Size (pt)</label>
                  <input className="dialog-input" type="number" min={9} max={16} step={0.5}
                    value={bookFontSize} onChange={e => setBookFontSize(Number(e.target.value))} />
                </div>
                <div className="dialog-field">
                  <label className="dialog-label">Line Spacing</label>
                  <select className="dialog-select" value={bookLineSpacing} onChange={e => setBookLineSpacing(e.target.value)}>
                    <option value="1.0">Single (1.0)</option>
                    <option value="1.25">Comfortable (1.25)</option>
                    <option value="1.5">Relaxed (1.5)</option>
                    <option value="2.0">Double (2.0)</option>
                  </select>
                </div>
              </div>
              <div className="dialog-field">
                <label className="dialog-label">Trim Size</label>
                <select className="dialog-select" value={bookTrimSize} onChange={e => setBookTrimSize(e.target.value)}>
                  {TRIM_SIZES.map(s => <option key={s.value} value={s.value}>{s.label}</option>)}
                </select>
                <div className="settings-hint">Used when exporting to PDF for print-ready typesetting.</div>
              </div>
              <div className="dialog-field">
                <label className="dialog-label" style={{ marginBottom: 8 }}>Drop Caps</label>
                <label className="settings-toggle">
                  <input type="checkbox" checked={bookDropCaps} onChange={e => setBookDropCaps(e.target.checked)} />
                  <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
                  <span className="settings-toggle-label">{bookDropCaps ? 'Enabled — first letter of each chapter is enlarged' : 'Disabled'}</span>
                </label>
              </div>
            </>}

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
