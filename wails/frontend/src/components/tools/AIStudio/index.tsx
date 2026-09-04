// AI Studio - AI-assisted editing modes and style mixer

import { useState, useEffect, useRef } from 'react'
import { useBookStore } from '../../../store/bookStore'
import { useAppStore } from '../../../store/appStore'
import { RewriteText, RewriteTextCustom, CancelRewrite, CheckClaudeCode, CheckCodexCLI } from '../../../../wailsjs/go/main/App'
import type { types } from '../../../../wailsjs/go/models'
import { EventsOn, EventsOff } from '../../../../wailsjs/runtime/runtime'
import type { WritingStyleOptions } from '../../../types/draftline'
import type { AIMode, AIState } from '../types'
import { AI_MODES, STYLE_FEATURES, INTENSITY_LABELS } from '../constants'
import { resolveTaskProvider, setTaskProvider } from '../../../services/aiRouting'
import NoAIProviderSetup from './NoAIProviderSetup'

// Icon paths per editing mode (24-viewBox, stroke-based, per the design).
const MODE_ICONS: Record<AIMode, string> = {
  line_edit: 'M4 20l3.2-.9L18 8.3 15.7 6 4.9 16.8 4 20zM13.5 8.2l2.3 2.3',
  copy_edit: 'M4 7h9M4 11h6M12.5 15.5l2.6 2.6L20 13',
  expand: 'M9 4H4v5M15 4h5v5M15 20h5v-5M9 20H4v-5',
  smooth: 'M3 15c2.5-5 5.5-5 8 0s5.5 5 8 0',
  custom: 'M5 7l4.5 5L5 17M12.5 17H19',
}

// Positions of the 4 style-option stop centers along the track (percent).
const STOP_CENTERS = [12.5, 37.5, 62.5, 87.5]

export default function AiStudioTab() {
  const { book, currentSection, currentIndex, setPendingDiff, getStyleOptions, updateStyleOptions, getEditorSelection } = useBookStore()
  const { settings, saveSettings, openSettings, showSettings } = useAppStore()

  const [aiMode, setAiMode] = useState<AIMode>('line_edit')
  const [aiState, setAiState] = useState<AIState>('idle')
  const [error, setError] = useState('')
  const [showImport, setShowImport] = useState(false)
  const [importText, setImportText] = useState('')
  const [showStyleMixer, setShowStyleMixer] = useState(false)
  const [customPrompt, setCustomPrompt] = useState('')
  const [ccStatus, setCcStatus] = useState<types.ClaudeCodeStatus | null>(null)
  const [ccChecking, setCcChecking] = useState(false)
  const [cxStatus, setCxStatus] = useState<types.ClaudeCodeStatus | null>(null)
  const [cxChecking, setCxChecking] = useState(false)
  const settingsWasOpen = useRef(showSettings)

  const styleOptions = getStyleOptions()
  const selection = getEditorSelection()

  const [providerMenuOpen, setProviderMenuOpen] = useState(false)
  const openAISettings = () => openSettings('ai')

  function refreshClaudeStatus() {
    setCcChecking(true)
    return CheckClaudeCode()
      .then(setCcStatus)
      .catch(() => setCcStatus({ installed: false, authenticated: false, npm_available: false, version: '' }))
      .finally(() => setCcChecking(false))
  }

  function refreshCodexStatus() {
    setCxChecking(true)
    return CheckCodexCLI()
      .then(setCxStatus)
      .catch(() => setCxStatus({ installed: false, authenticated: false, npm_available: false, version: '' }))
      .finally(() => setCxChecking(false))
  }

  // Check both CLIs on mount so the provider quick-switcher shows accurate
  // ready states without a trip through settings.
  useEffect(() => {
    void refreshClaudeStatus()
    void refreshCodexStatus()
  }, [])

  // The settings dialog owns its own setup-status state. Re-check when it
  // closes so this already-mounted sidebar cannot keep displaying the stale
  // result it captured before a CLI was installed or authenticated.
  useEffect(() => {
    if (showSettings) {
      settingsWasOpen.current = true
      return
    }
    if (!settingsWasOpen.current) return
    settingsWasOpen.current = false
    void refreshClaudeStatus()
    void refreshCodexStatus()
  }, [showSettings])

  // Authentication can finish while settings remains open or after the user
  // closes it. Refresh immediately in either case.
  useEffect(() => {
    const offClaude = EventsOn('claude:auth_complete', () => { void refreshClaudeStatus() })
    const offCodex = EventsOn('codex:auth_complete', () => { void refreshCodexStatus() })
    return () => { offClaude(); offCodex() }
  }, [])

  const taskProvider = resolveTaskProvider(settings.ai_mode, settings.ai_task_routes, aiMode)

  // Determine if the provider assigned to this editing task is configured.
  const aiConfigured = settings.ai_enabled && (
    (taskProvider === 'claudecode' && ccStatus?.installed && ccStatus?.authenticated) ||
    (taskProvider === 'codex' && cxStatus?.installed && cxStatus?.authenticated) ||
    (taskProvider === 'api' && settings.ai_provider !== '' && settings.has_api_key) ||
    (taskProvider === 'local' && settings.ai_local_endpoint !== '' && settings.ai_local_model !== '')
  )

  function getCurrentHTML(): string {
    if (!book) return ''
    if (currentSection === 'copyright') return book.copyright
    const arr = currentSection === 'front_matter' ? book.front_matter
      : currentSection === 'body' ? book.body
        : book.back_matter
    return arr[currentIndex]?.content ?? ''
  }

  function getAiLabel(): string {
    if (taskProvider === 'claudecode') return 'Claude Code'
    if (taskProvider === 'codex') return 'Codex'
    if (taskProvider === 'local') return settings.ai_local_model || 'Local AI'
    const providerLabels: Record<string, string> = {
      claude: 'Claude',
      openai: 'OpenAI',
      gemini: 'Gemini',
      grok: 'Grok',
    }
    return providerLabels[settings.ai_provider] || 'AI'
  }

  const currentMode = AI_MODES.find(m => m.id === aiMode)!

  // Provider quick-switcher routes, derived from what's actually configured
  // in AI Studio settings. One API row: the app stores a single key for the
  // currently selected provider (see docs/frontend/AI-STUDIO-GAPS.md).
  const providerLabels: Record<string, string> = { claude: 'Claude', openai: 'OpenAI', gemini: 'Gemini', grok: 'Grok' }
  const routes: { mode: typeof settings.ai_mode; name: string; mono: string; ready: boolean; model: string }[] = [
    {
      mode: 'claudecode', name: 'Claude Code', mono: 'C',
      ready: !!(ccStatus?.installed && ccStatus?.authenticated),
      model: (taskProvider === 'claudecode' && settings.ai_model) || 'Claude.ai account',
    },
    {
      mode: 'codex', name: 'Codex', mono: 'O',
      ready: !!(cxStatus?.installed && cxStatus?.authenticated),
      model: taskProvider === 'codex' && !/^(claude|gemini|grok|llama|mistral)/i.test(settings.ai_model)
        ? settings.ai_model || 'ChatGPT account'
        : 'ChatGPT account',
    },
    {
      mode: 'api',
      name: settings.ai_provider ? `${providerLabels[settings.ai_provider]} API` : 'API key',
      mono: settings.ai_provider ? providerLabels[settings.ai_provider][0] : 'A',
      ready: settings.ai_provider !== '' && settings.has_api_key,
      model: (taskProvider === 'api' && settings.ai_model)
        || (settings.ai_provider ? providerLabels[settings.ai_provider] : 'no key stored'),
    },
    {
      mode: 'local', name: 'Local', mono: 'L',
      ready: settings.ai_local_endpoint !== '' && settings.ai_local_model !== '',
      model: settings.ai_local_model || settings.ai_local_endpoint || 'no endpoint',
    },
  ]
  const activeRoute = routes.find(r => r.mode === taskProvider) ?? routes[0]

  function pickRoute(route: (typeof routes)[number]) {
    setProviderMenuOpen(false)
    const override = route.mode === settings.ai_mode ? null : route.mode
    void saveSettings({ ai_task_routes: setTaskProvider(settings.ai_task_routes, aiMode, override) })
    if (!route.ready) openAISettings()
  }

  async function handleRun() {
    const fullHtml = getCurrentHTML()
    if (!fullHtml || fullHtml === '<p></p>') return

    // Determine what text to process: selection or full chapter
    const currentSelection = getEditorSelection()
    const hasSelection = currentSelection && currentSelection.text.trim().length > 0

    // For custom mode, we need a prompt
    if (aiMode === 'custom') {
      // Look for @ai commands in the text, or use the custom prompt input
      const textToSearch = hasSelection ? currentSelection.text : fullHtml.replace(/<[^>]*>/g, ' ')
      const aiMatch = textToSearch.match(/@ai\s+(.+?)(?:\n|$)/i)
      const prompt = customPrompt.trim() || (aiMatch ? aiMatch[1].trim() : '')

      if (!prompt) {
        setError('Enter a prompt below or use @ai in your text (e.g., "@ai expand this scene")')
        setAiState('error')
        return
      }
    }

    setAiState('loading')
    setError('')
    try {
      let res: { result: string; error?: string }
      let originalHtml: string

      if (aiMode === 'custom') {
        // Custom mode: use the prompt
        const textToProcess = hasSelection ? currentSelection.text : fullHtml
        const textToSearch = hasSelection ? currentSelection.text : fullHtml.replace(/<[^>]*>/g, ' ')
        const aiMatch = textToSearch.match(/@ai\s+(.+?)(?:\n|$)/i)
        const prompt = customPrompt.trim() || (aiMatch ? aiMatch[1].trim() : '')

        // Remove the @ai command from the text if present
        let cleanText = textToProcess
        if (aiMatch) {
          cleanText = textToProcess.replace(/@ai\s+.+?(?:\n|$)/gi, '')
        }

        res = await RewriteTextCustom(cleanText, prompt, taskProvider)
        originalHtml = hasSelection ? currentSelection.text : fullHtml
      } else {
        // Standard modes: use selection if available
        const textToProcess = hasSelection ? wrapSelectionAsHtml(currentSelection.text) : fullHtml
        const useStyleOptions = aiMode === 'expand' || aiMode === 'smooth'
        res = await RewriteText(textToProcess, aiMode, useStyleOptions ? JSON.stringify(styleOptions) : '', taskProvider)
        originalHtml = textToProcess
      }

      if (res.error) {
        setError(res.error)
        setAiState('error')
      } else {
        const { diffContent } = await import('../../../utils/diff')
        const d = diffContent(originalHtml, res.result)
        setPendingDiff({ diffs: d, originalHtml })
        setAiState('idle')
      }
    } catch (e) {
      setError(String(e))
      setAiState('error')
    }
  }

  // Helper to wrap plain text selection as HTML paragraphs
  function wrapSelectionAsHtml(text: string): string {
    if (text.trim().startsWith('<')) return text
    return text.split(/\n\n+/).map(p => `<p>${p.trim()}</p>`).join('\n')
  }

  function handleStyleChange(key: keyof WritingStyleOptions, value: number) {
    updateStyleOptions({ [key]: value })
  }

  function handleImportCompare() {
    const html = getCurrentHTML()
    if (!html || !importText.trim()) return
    const pasted = importText.trim().startsWith('<')
      ? importText
      : importText.split(/\n{2,}/).map(p => `<p>${p.trim()}</p>`).join('\n')
    import('../../../utils/diff').then(({ diffContent }) => {
      const d = diffContent(html, pasted)
      setPendingDiff({ diffs: d, originalHtml: html })
      setShowImport(false)
      setImportText('')
    })
  }

  if (!book) {
    return <div className="tool-empty-state">Open or create a project to use AI tools.</div>
  }

  // ── Render states ──

  if (aiState === 'loading') {
    return (
      <LoadingPane
        label={`${currentMode.label} · ${getAiLabel()}`}
        onCancel={() => {
          CancelRewrite()
          setAiState('idle')
        }}
      />
    )
  }

  if (aiState === 'error') {
    const isConnectionError = error.toLowerCase().includes('connection') ||
      error.toLowerCase().includes('network') ||
      error.toLowerCase().includes('failed to fetch') ||
      error.toLowerCase().includes('econnrefused')
    const isTimeoutError = error.toLowerCase().includes('timeout') ||
      error.toLowerCase().includes('timed out')
    const isAuthError = error.toLowerCase().includes('auth') ||
      error.toLowerCase().includes('401') ||
      error.toLowerCase().includes('403') ||
      error.toLowerCase().includes('invalid api key')
    const isNotInstalled = error.toLowerCase().includes('not installed')

    return (
      <div className="ai-error-pane">
        <div className="ai-error-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
        </div>
        <div className="ai-error-title">
          {isConnectionError ? 'Connection Failed' :
           isTimeoutError ? 'Request Timed Out' :
           isAuthError ? 'Authentication Error' :
           isNotInstalled ? 'Setup Required' :
           'Something Went Wrong'}
        </div>
        <div className="ai-error-msg">{error}</div>
        <div className="ai-error-actions">
          <button className="ai-run-btn" onClick={() => { setError(''); setAiState('idle') }}>
            Try Again
          </button>
          {(isAuthError || isNotInstalled) && (
            <button className="ai-link-btn" onClick={openAISettings}>
              Open Settings
            </button>
          )}
          {!isAuthError && !isNotInstalled && (
            <button className="ai-link-btn" onClick={() => setAiState('idle')}>
              ← Back
            </button>
          )}
        </div>
        {isConnectionError && (
          <div className="ai-error-hint">
            Check your internet connection or verify the AI service is running.
          </div>
        )}
        {isTimeoutError && (
          <div className="ai-error-hint">
            The request took too long. Try selecting a smaller portion of text.
          </div>
        )}
      </div>
    )
  }

  // ── Idle state ──
  const showMixer = (aiMode === 'expand' || aiMode === 'smooth')
  const stylesOnCount = STYLE_FEATURES.filter(f => styleOptions[f.key] > 0).length

  return (
    <div className="ais-root">
      {/* Provider quick-switcher */}
      <div className="ais-provider-wrap">
        <button className="ais-provider-btn" onClick={() => setProviderMenuOpen(o => !o)}>
          <span className={`ais-provider-dot${activeRoute.ready ? ' ready' : ''}`} />
          <span className="ais-provider-name">{activeRoute.name}</span>
          <span className="ais-provider-model">{activeRoute.model}</span>
          <svg className={`ais-chevron${providerMenuOpen ? ' open' : ''}`} width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <path d="M6 9l6 6 6-6" />
          </svg>
        </button>
        {providerMenuOpen && (
          <>
            <div className="ais-menu-overlay" onClick={() => setProviderMenuOpen(false)} />
            <div className="ais-menu">
              <div className="ais-menu-label">Provider for {currentMode.label}</div>
              {routes.map(r => (
                <button
                  key={r.mode}
                  className={`ais-menu-row${r.mode === settings.ai_mode ? ' active' : ''}${r.ready ? '' : ' unconfigured'}`}
                  onClick={() => pickRoute(r)}
                >
                  <span className="ais-menu-mono">{r.mono}</span>
                  <span className="ais-menu-texts">
                    <span className="ais-menu-name">{r.name}</span>
                    <span className="ais-menu-model">{r.model}</span>
                  </span>
                  {r.ready
                    ? <span className="ais-menu-ready" />
                    : <span className="ais-menu-status">Not configured</span>}
                  {r.mode === taskProvider && (
                    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="var(--app-accent-text)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M5 13l4 4L19 7" />
                    </svg>
                  )}
                </button>
              ))}
              <div className="ais-menu-sep" />
              <button className="ais-menu-manage" onClick={() => { setProviderMenuOpen(false); openAISettings() }}>
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="3" /><path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M19.1 4.9L17 7M7 17l-2.1 2.1" />
                </svg>
                Manage providers…
              </button>
            </div>
          </>
        )}
      </div>

      <div className="ais-body">
        {/* Editing modes */}
        <div className="ais-section-label">Editing Mode</div>
        <div className="ais-mode-list">
          {AI_MODES.map(m => (
            <button
              key={m.id}
              className={`ais-mode-row${aiMode === m.id ? ' selected' : ''}`}
              onClick={() => setAiMode(m.id)}
            >
              <span className="ais-mode-icon">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                  <path d={MODE_ICONS[m.id]} />
                </svg>
              </span>
              <span className="ais-mode-texts">
                <span className="ais-mode-name">{m.label}</span>
                <span className="ais-mode-desc">{m.desc}</span>
              </span>
            </button>
          ))}
        </div>

        {/* Style options for Expand/Smooth modes */}
        {showMixer && (
          <div className="ais-style-card">
            <button className="ais-style-toggle" onClick={() => setShowStyleMixer(!showStyleMixer)}>
              <span className="ais-section-label">Style Options</span>
              <span className="ais-style-count">{stylesOnCount} of {STYLE_FEATURES.length} on</span>
              <svg className={`ais-chevron${showStyleMixer ? ' open' : ''}`} width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <path d="M6 9l6 6 6-6" />
              </svg>
            </button>
            {showStyleMixer && (
              <div className="ais-style-body">
                {STYLE_FEATURES.map(feature => {
                  const v = styleOptions[feature.key]
                  return (
                    <div key={feature.key} className="ais-style-row">
                      <div className="ais-style-rowhead">
                        <span className="ais-style-name">{feature.label}</span>
                        <span className={`ais-style-value${v > 0 ? ' on' : ''}`}>{INTENSITY_LABELS[v]}</span>
                      </div>
                      <div className="ais-stop-track">
                        <div className="ais-stop-line" />
                        <div className="ais-stop-fill" style={{ width: `${STOP_CENTERS[v] - STOP_CENTERS[0]}%` }} />
                        <div className="ais-stop-grid">
                          {INTENSITY_LABELS.map((label, i) => (
                            <button
                              key={i}
                              className="ais-stop"
                              title={label}
                              onClick={() => handleStyleChange(feature.key, i)}
                            >
                              <span className={`ais-stop-dot${i === v ? ' active' : i < v ? ' passed' : ''}`} />
                            </button>
                          ))}
                        </div>
                      </div>
                    </div>
                  )
                })}
                <div className="ais-style-hint">
                  Customize how the AI enhances your writing. "Off" skips that feature entirely.
                </div>
              </div>
            )}
          </div>
        )}

        {/* Custom mode prompt input */}
        {aiMode === 'custom' && (
          <div className="ais-custom">
            <div className="ais-section-label">Custom Prompt</div>
            <textarea
              className="ais-custom-textarea"
              value={customPrompt}
              onChange={e => setCustomPrompt(e.target.value)}
              placeholder="Enter your instruction (e.g., 'expand this scene with more sensory detail') or use @ai in your text…"
            />
            <div className="ais-custom-hint">
              Tip: write <span className="ais-kbd">@ai your instruction</span> directly in your text.
            </div>
          </div>
        )}

        {/* Selection indicator */}
        {selection && (
          <div className="ai-selection-indicator">
            <span className="selection-icon">✓</span>
            <span>Selection active ({selection.text.length} chars) — AI will target only selected text</span>
          </div>
        )}

        {/* Import pane */}
        {showImport && (
          <div className="import-pane">
            <div className="import-pane-label">Paste edited version — plain text or HTML:</div>
            <textarea
              className="import-edits-textarea"
              value={importText}
              onChange={e => setImportText(e.target.value)}
              placeholder="Paste the edited chapter here…"
              autoFocus
            />
            <button className="ai-run-btn" onClick={handleImportCompare} disabled={!importText.trim()}>
              Compare with Current
            </button>
            <button className="ai-link-btn" onClick={() => { setShowImport(false); setImportText('') }}>Cancel</button>
          </div>
        )}

        {/* Setup guidance when the active route is unconfigured */}
        {!aiConfigured && !showImport && (
          <AiSetupGuidance
            ccStatus={ccStatus}
            ccChecking={ccChecking}
            cxStatus={cxStatus}
            cxChecking={cxChecking}
            settings={settings}
            providerMode={taskProvider}
            onOpenSettings={openAISettings}
          />
        )}
      </div>

      {/* Pinned run footer */}
      {aiConfigured && !showImport && (
        <div className="ais-footer">
          <button className="ai-run-btn" onClick={handleRun}>
            Run {currentMode.label}
          </button>
          <div className="ais-compare">
            <button className="ai-link-btn" onClick={() => setShowImport(true)}>
              or compare with imported draft ›
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

// ── AI Setup Guidance ────────────────────────────────────────────────────────

interface AiSetupGuidanceProps {
  ccStatus: types.ClaudeCodeStatus | null
  ccChecking: boolean
  cxStatus: types.ClaudeCodeStatus | null
  cxChecking: boolean
  settings: { ai_mode: string; ai_provider: string; has_api_key: boolean; ai_local_endpoint: string; ai_local_model: string }
  providerMode: string
  onOpenSettings: () => void
}

function AiSetupGuidance({ ccStatus, ccChecking, cxStatus, cxChecking, settings, providerMode, onOpenSettings }: AiSetupGuidanceProps) {
  // Determine what's configured
  const ccInstalled = ccStatus?.installed
  const ccAuthenticated = ccStatus?.authenticated
  const hasApiKey = settings.ai_provider !== '' && settings.has_api_key
  const hasLocalEndpoint = settings.ai_local_endpoint !== '' && settings.ai_local_model !== ''

  // Do not steer a fresh installation toward one vendor simply because the
  // legacy default mode happens to be Claude Code. Wait for both checks, then
  // present the provider-neutral setup route when nothing is installed or
  // configured yet.
  const checksComplete = ccStatus !== null && cxStatus !== null && !ccChecking && !cxChecking
  const hasNoProvider = checksComplete
    && !ccStatus.installed
    && !cxStatus.installed
    && !hasApiKey
    && !hasLocalEndpoint
  if (hasNoProvider) {
    return <NoAIProviderSetup onOpenSettings={onOpenSettings} />
  }

  // If Claude Code mode is selected but not set up
  if (providerMode === 'claudecode') {
    if (ccChecking) {
      return (
        <div className="ai-setup-pane">
          <div className="ai-setup-checking">
            <div className="ai-spinner" />
            <span>Checking Claude Code...</span>
          </div>
        </div>
      )
    }

    if (!ccInstalled) {
      return (
        <div className="ai-setup-pane">
          <div className="ai-setup-icon">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <path d="M12 2v4m0 12v4M4.93 4.93l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"/>
            </svg>
          </div>
          <div className="ai-setup-title">Claude Code Not Installed</div>
          <p className="ai-setup-desc">
            Claude Code is a standalone AI coding assistant that powers the editing features.
          </p>
          <button className="ai-run-btn" onClick={onOpenSettings}>
            Set Up Claude Code
          </button>
          <p className="ai-setup-alt">
            Or <button className="ai-text-btn" onClick={onOpenSettings}>use an API key</button> instead
          </p>
        </div>
      )
    }

    if (!ccAuthenticated) {
      return (
        <div className="ai-setup-pane">
          <div className="ai-setup-icon installed">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <path d="M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2zm0 18a8 8 0 1 1 8-8 8 8 0 0 1-8 8z"/>
              <path d="M12 6v6l4 2"/>
            </svg>
          </div>
          <div className="ai-setup-title">Authentication Required</div>
          <p className="ai-setup-desc">
            Claude Code is installed but needs to be authenticated with your Anthropic account.
          </p>
          <button className="ai-run-btn" onClick={onOpenSettings}>
            Authenticate →
          </button>
        </div>
      )
    }
  }

  // If Codex mode is selected but not set up
  if (providerMode === 'codex') {
    if (cxChecking) {
      return (
        <div className="ai-setup-pane">
          <div className="ai-setup-checking">
            <div className="ai-spinner" />
            <span>Checking Codex...</span>
          </div>
        </div>
      )
    }
    if (!cxStatus?.installed || !cxStatus?.authenticated) {
      return (
        <div className="ai-setup-pane">
          <div className="ai-setup-icon">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <path d="M12 2v4m0 12v4M4.93 4.93l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"/>
            </svg>
          </div>
          <div className="ai-setup-title">{cxStatus?.installed ? 'Sign In Required' : 'Codex Not Installed'}</div>
          <p className="ai-setup-desc">
            {cxStatus?.installed
              ? 'Codex is installed but needs a signed-in ChatGPT account.'
              : 'Codex uses your ChatGPT account to power the editing features.'}
          </p>
          <button className="ai-run-btn" onClick={onOpenSettings}>
            {cxStatus?.installed ? 'Sign In →' : 'Set Up Codex'}
          </button>
        </div>
      )
    }
  }

  // If API mode is selected but no key
  if (providerMode === 'api' && !hasApiKey) {
    return (
      <div className="ai-setup-pane">
        <div className="ai-setup-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
        </div>
        <div className="ai-setup-title">API Key Required</div>
        <p className="ai-setup-desc">
          Add your API key to use {settings.ai_provider ? settings.ai_provider.charAt(0).toUpperCase() + settings.ai_provider.slice(1) : 'AI'} services.
        </p>
        <button className="ai-run-btn" onClick={onOpenSettings}>
          Add API Key
        </button>
        {ccInstalled && ccAuthenticated && (
          <p className="ai-setup-alt">
            Or <button className="ai-text-btn" onClick={onOpenSettings}>switch to Claude Code</button>
          </p>
        )}
      </div>
    )
  }

  // If Local mode is selected but no endpoint
  if (providerMode === 'local' && !hasLocalEndpoint) {
    return (
      <div className="ai-setup-pane">
        <div className="ai-setup-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
            <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
            <path d="M8 21h8m-4-4v4"/>
          </svg>
        </div>
        <div className="ai-setup-title">Local AI Not Configured</div>
        <p className="ai-setup-desc">
          Configure your local AI endpoint (e.g., Ollama, LM Studio) in Settings.
        </p>
        <button className="ai-run-btn" onClick={onOpenSettings}>
          Configure Endpoint
        </button>
      </div>
    )
  }

  return <NoAIProviderSetup onOpenSettings={onOpenSettings} />
}

// ── Loading pane ─────────────────────────────────────────────────────────────

function LoadingPane({ label, onCancel }: { label: string; onCancel: () => void }) {
  const [elapsed, setElapsed] = useState(0)
  const [logLines, setLogLines] = useState<string[]>([])
  const [streamText, setStreamText] = useState('')
  const logRef = useRef<HTMLDivElement>(null)
  const streamRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const t = setInterval(() => setElapsed(s => s + 1), 1000)
    return () => clearInterval(t)
  }, [])

  useEffect(() => {
    const handler = (line: string) => {
      setLogLines(prev => {
        const next = [...prev, line]
        return next.length > 20 ? next.slice(-20) : next
      })
    }
    EventsOn('ai:log', handler)
    return () => EventsOff('ai:log')
  }, [])

  useEffect(() => {
    const handler = (token: string) => setStreamText(prev => prev + token)
    EventsOn('ai:token', handler)
    return () => EventsOff('ai:token')
  }, [])

  useEffect(() => {
    if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight
  }, [logLines])

  useEffect(() => {
    if (streamRef.current) streamRef.current.scrollTop = streamRef.current.scrollHeight
  }, [streamText])

  const mins = Math.floor(elapsed / 60)
  const secs = elapsed % 60
  const timeStr = mins > 0 ? `${mins}m ${secs}s` : `${secs}s`

  return (
    <div className="ai-loading-pane">
      <div className="ai-loading-header">
        <div className="ai-spinner" />
        <span className="ai-loading-label">{label}</span>
        <span className="ai-loading-time">{timeStr}</span>
      </div>
      {streamText ? (
        <div className="ai-stream-box" ref={streamRef}>{streamText}</div>
      ) : logLines.length > 0 && (
        <div className="ai-log-box" ref={logRef}>
          {logLines.map((line, i) => <div key={i} className="ai-log-line">{line}</div>)}
        </div>
      )}
      <button className="ai-link-btn" style={{ marginTop: 10 }} onClick={onCancel}>
        Cancel
      </button>
    </div>
  )
}
