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

export default function AiStudioTab() {
  const { book, currentSection, currentIndex, setPendingDiff, getStyleOptions, updateStyleOptions, getEditorSelection } = useBookStore()
  const { settings, openSettings } = useAppStore()

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

  const styleOptions = getStyleOptions()
  const selection = getEditorSelection()

  // Check CLI status on mount for the active mode
  useEffect(() => {
    if (settings.ai_mode === 'codex') {
      if (!cxStatus && !cxChecking) {
        setCxChecking(true)
        CheckCodexCLI()
          .then(setCxStatus)
          .catch(() => {})
          .finally(() => setCxChecking(false))
      }
      return
    }
    if (!ccStatus && !ccChecking) {
      setCcChecking(true)
      CheckClaudeCode()
        .then(setCcStatus)
        .catch(() => {})
        .finally(() => setCcChecking(false))
    }
  }, [settings.ai_mode])

  // Determine if AI is configured based on mode
  const aiConfigured = settings.ai_enabled && (
    (settings.ai_mode === 'claudecode' && ccStatus?.installed && ccStatus?.authenticated) ||
    (settings.ai_mode === 'codex' && cxStatus?.installed && cxStatus?.authenticated) ||
    (settings.ai_mode === 'api' && settings.ai_provider !== '' && settings.has_api_key) ||
    (settings.ai_mode === 'local' && settings.ai_local_endpoint !== '')
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
    if (settings.ai_mode === 'claudecode') return 'Claude Code'
    if (settings.ai_mode === 'codex') return 'Codex'
    if (settings.ai_mode === 'local') return settings.ai_local_model || 'Local AI'
    const providerLabels: Record<string, string> = {
      claude: 'Claude',
      openai: 'OpenAI',
      gemini: 'Gemini',
      grok: 'Grok',
    }
    return providerLabels[settings.ai_provider] || 'AI'
  }

  const currentMode = AI_MODES.find(m => m.id === aiMode)!

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

        res = await RewriteTextCustom(cleanText, prompt)
        originalHtml = hasSelection ? currentSelection.text : fullHtml
      } else {
        // Standard modes: use selection if available
        const textToProcess = hasSelection ? wrapSelectionAsHtml(currentSelection.text) : fullHtml
        const useStyleOptions = aiMode === 'expand' || aiMode === 'smooth'
        res = await RewriteText(textToProcess, aiMode, useStyleOptions ? JSON.stringify(styleOptions) : '')
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
            <button className="ai-link-btn" onClick={openSettings}>
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

  return (
    <>
      {/* Mode selector */}
      <div className="ai-section-label">Editing Mode</div>
      <div className="ai-mode-list">
        {AI_MODES.map(m => (
          <button
            key={m.id}
            className={`ai-mode-row${aiMode === m.id ? ' selected' : ''}`}
            onClick={() => setAiMode(m.id)}
          >
            <span className="ai-mode-dot" />
            <span className="ai-mode-name">{m.label}</span>
            <span className="ai-mode-hint">{m.desc}</span>
          </button>
        ))}
      </div>

      {/* Style Mixer for Expand/Smooth modes */}
      {showMixer && (
        <div className="style-mixer-section">
          <button
            className="style-mixer-toggle"
            onClick={() => setShowStyleMixer(!showStyleMixer)}
          >
            <span>Style Options</span>
            <svg
              className={`style-mixer-chevron${showStyleMixer ? ' open' : ''}`}
              width="10"
              height="6"
              viewBox="0 0 10 6"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M1 1l4 4 4-4" />
            </svg>
          </button>
          {showStyleMixer && (
            <div className="style-mixer-content">
              {STYLE_FEATURES.map(feature => (
                <div key={feature.key} className="style-feature-row">
                  <div className="style-feature-header">
                    <span className="style-feature-label">{feature.label}</span>
                    <span className="style-feature-value">{INTENSITY_LABELS[styleOptions[feature.key]]}</span>
                  </div>
                  <div className="style-feature-slider">
                    <input
                      type="range"
                      min="0"
                      max="3"
                      value={styleOptions[feature.key]}
                      onChange={e => handleStyleChange(feature.key, parseInt(e.target.value))}
                      className="style-slider"
                    />
                    <div className="style-slider-marks">
                      {INTENSITY_LABELS.map((label, i) => (
                        <span
                          key={i}
                          className={`style-slider-mark${styleOptions[feature.key] === i ? ' active' : ''}`}
                        />
                      ))}
                    </div>
                  </div>
                </div>
              ))}
              <div className="style-mixer-hint">
                Customize how the AI enhances your writing. "Off" skips that feature entirely.
              </div>
            </div>
          )}
        </div>
      )}

      {/* Custom mode prompt input */}
      {aiMode === 'custom' && (
        <div className="custom-prompt-section">
          <div className="ai-section-label">Custom Prompt</div>
          <textarea
            className="custom-prompt-input"
            value={customPrompt}
            onChange={e => setCustomPrompt(e.target.value)}
            placeholder="Enter your instruction (e.g., 'expand this scene with more sensory detail') or use @ai in your text..."
            rows={3}
          />
          <div className="custom-prompt-hint">
            Tip: You can also write <code>@ai your instruction</code> directly in your text.
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
      {showImport ? (
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
      ) : aiConfigured ? (
        <div className="ai-actions">
          <button className="ai-run-btn" onClick={handleRun}>
            Run {currentMode.label}
          </button>
          <button className="ai-link-btn" onClick={() => setShowImport(true)}>
            or compare with imported draft ›
          </button>
        </div>
      ) : (
        <AiSetupGuidance
          ccStatus={ccStatus}
          ccChecking={ccChecking}
          cxStatus={cxStatus}
          cxChecking={cxChecking}
          settings={settings}
          onOpenSettings={openSettings}
        />
      )}
    </>
  )
}

// ── AI Setup Guidance ────────────────────────────────────────────────────────

interface AiSetupGuidanceProps {
  ccStatus: types.ClaudeCodeStatus | null
  ccChecking: boolean
  cxStatus: types.ClaudeCodeStatus | null
  cxChecking: boolean
  settings: { ai_mode: string; ai_provider: string; has_api_key: boolean; ai_local_endpoint: string }
  onOpenSettings: () => void
}

function AiSetupGuidance({ ccStatus, ccChecking, cxStatus, cxChecking, settings, onOpenSettings }: AiSetupGuidanceProps) {
  // Determine what's configured
  const ccInstalled = ccStatus?.installed
  const ccAuthenticated = ccStatus?.authenticated
  const hasApiKey = settings.ai_provider !== '' && settings.has_api_key
  const hasLocalEndpoint = settings.ai_local_endpoint !== ''

  // If Claude Code mode is selected but not set up
  if (settings.ai_mode === 'claudecode') {
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
  if (settings.ai_mode === 'codex') {
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
  if (settings.ai_mode === 'api' && !hasApiKey) {
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
  if (settings.ai_mode === 'local' && !hasLocalEndpoint) {
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

  // Generic fallback - nothing configured at all
  return (
    <div className="ai-setup-pane">
      <div className="ai-setup-icon">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
          <circle cx="12" cy="12" r="3"/>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
        </svg>
      </div>
      <div className="ai-setup-title">AI Features Ready</div>
      <p className="ai-setup-desc">
        Configure AI-assisted editing in Settings. Choose from Claude Code, cloud APIs, or local models.
      </p>
      <button className="ai-run-btn" onClick={onOpenSettings}>
        Open Settings
      </button>
    </div>
  )
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
