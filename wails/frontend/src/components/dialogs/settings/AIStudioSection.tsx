// AI Studio settings: Claude Code, Codex, API keys, and configured providers.

import type { AIStudioSectionProps, AIProvider } from './types'
import { CLAUDE_MODELS, OPENAI_MODELS, DEFAULT_MODELS } from './constants'
import { useState, useEffect } from 'react'
import { ListProviderModels } from '../../../../wailsjs/go/main/App'
import type { AIEditingTask, AIProviderMode } from '../../../services/aiRouting'
import { setTaskProvider } from '../../../services/aiRouting'
import AIProvidersList from './AIProviders'

const TASK_ROUTES: { id: AIEditingTask; label: string }[] = [
  { id: 'line_edit', label: 'Line Edit' },
  { id: 'copy_edit', label: 'Copy Edit' },
  { id: 'expand', label: 'Expand' },
  { id: 'smooth', label: 'Smooth' },
  { id: 'custom', label: 'Custom Prompt' },
]

const BUILT_IN_NAMES: Record<string, string> = {
  claudecode: 'Claude Code',
  codex: 'Codex',
  api: 'API Key',
}

export default function AIStudioSection({
  aiEnabled, setAiEnabled,
  aiMode, setAiMode,
  taskRoutes, setTaskRoutes,
  provider, setProvider,
  apiKey, setApiKey,
  hasStoredKey, onClearKey,
  debugLogging, setDebugLogging,
  model, setModel,
  providers, onProvidersChanged,
  proseGuide, setProseGuide,
  ccStatus, ccChecking, ccSetupStep, ccSetupLog,
  onCheckCC, onSetup, onOpenAuth,
  cxStatus, cxChecking, cxSetupStep, cxSetupLog,
  onCheckCx, onSetupCx, onOpenCxAuth,
}: AIStudioSectionProps) {
  const [showKey, setShowKey] = useState(false)
  const [liveModels, setLiveModels] = useState<string[]>([])

  // Ask the provider what it serves. The built-in list is only the fallback
  // for an account that cannot be reached, so a retired model never sits in
  // the dropdown looking selectable.
  useEffect(() => {
    setLiveModels([])
    if (!aiEnabled || aiMode !== 'api' || !provider || !hasStoredKey) return
    let cancelled = false
    ListProviderModels(provider)
      .then(ids => { if (!cancelled && ids?.length) setLiveModels(ids) })
      .catch(() => { /* the built-in list stands in */ })
    return () => { cancelled = true }
  }, [aiEnabled, aiMode, provider, hasStoredKey])

  const fallbackModels = provider === 'claude' ? CLAUDE_MODELS
    : provider === 'openai' ? OPENAI_MODELS
    : []
  const modelOptions = liveModels.length
    ? liveModels.map(id => ({ value: id, label: id }))
    : fallbackModels

  const activeProvider = providers.find(p => p.id === aiMode)
  const modeName = (m: string) =>
    BUILT_IN_NAMES[m] ?? providers.find(p => p.id === m)?.nickname ?? m

  function handleProviderChange(p: AIProvider) {
    setProvider(p)
    setModel(DEFAULT_MODELS[p] || '')
  }

  function handleModeChange(m: typeof aiMode) {
    setAiMode(m)
    if (m === 'claudecode' && !model) setModel('claude-sonnet-5')
    // Codex: no default model — the CLI's own current default is used.
    if (m === 'codex' && model.startsWith('claude')) setModel('')
  }

  return (
    <>
      <div className="settings-section-label" style={{ marginTop: 0 }}>
        AI Features
        <label className="settings-toggle" title="Enable or disable AI features">
          <input type="checkbox" checked={aiEnabled} onChange={e => setAiEnabled(e.target.checked)} />
          <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
          <span className="settings-toggle-label">{aiEnabled ? 'Enabled' : 'Disabled'}</span>
        </label>
      </div>

      {aiEnabled && <>
        <div className="dialog-field">
          <label className="dialog-label">Default Provider</label>
          <div className="settings-theme-row">
            <button className={`settings-theme-btn${aiMode === 'claudecode' ? ' active' : ''}`} onClick={() => handleModeChange('claudecode')}>
              Claude Code
            </button>
            <button className={`settings-theme-btn${aiMode === 'codex' ? ' active' : ''}`} onClick={() => handleModeChange('codex')}>
              Codex
            </button>
            <button className={`settings-theme-btn${aiMode === 'api' ? ' active' : ''}`} onClick={() => handleModeChange('api')}>
              API Key
            </button>
            {providers.map(p => (
              <button
                key={p.id}
                className={`settings-theme-btn${aiMode === p.id ? ' active' : ''}`}
                onClick={() => handleModeChange(p.id)}
              >{p.nickname}</button>
            ))}
          </div>
          <div className="settings-hint">Used for every task unless it has an override below.</div>
        </div>

        <div className="dialog-field">
          <label className="dialog-label">Provider by Task</label>
          <div className="settings-ai-routing">
            {TASK_ROUTES.map(task => (
              <label className="settings-ai-route" key={task.id}>
                <span>{task.label}</span>
                <select
                  className="dialog-select"
                  value={taskRoutes[task.id] ?? ''}
                  onChange={e => setTaskRoutes(setTaskProvider(
                    taskRoutes,
                    task.id,
                    (e.target.value || null) as AIProviderMode | null,
                  ))}
                >
                  <option value="">Default ({modeName(aiMode)})</option>
                  <option value="claudecode">Claude Code</option>
                  <option value="codex">Codex</option>
                  <option value="api">API Key</option>
                  {providers.map(p => (
                    <option key={p.id} value={p.id}>{p.nickname}</option>
                  ))}
                </select>
              </label>
            ))}
          </div>
          <div className="settings-hint">Assignments are explicit. Draftline never sends a task to a different provider as a fallback.</div>
        </div>

        {/* Claude Code */}
        {aiMode === 'claudecode' && (
          <CLISetupSection
            flavor="claude"
            ccStatus={ccStatus}
            ccChecking={ccChecking}
            ccSetupStep={ccSetupStep}
            ccSetupLog={ccSetupLog}
            onCheckCC={onCheckCC}
            onSetup={onSetup}
            onOpenAuth={onOpenAuth}
            model={model}
            setModel={setModel}
          />
        )}

        {/* Codex (ChatGPT account) */}
        {aiMode === 'codex' && (
          <CLISetupSection
            flavor="codex"
            ccStatus={cxStatus}
            ccChecking={cxChecking}
            ccSetupStep={cxSetupStep}
            ccSetupLog={cxSetupLog}
            onCheckCC={onCheckCx}
            onSetup={onSetupCx}
            onOpenAuth={onOpenCxAuth}
            model={model}
            setModel={setModel}
          />
        )}

        {/* API Key */}
        {aiMode === 'api' && <>
          <div className="dialog-field">
            <label className="dialog-label">Provider</label>
            <div className="settings-provider-row">
              <button className={`settings-provider-btn${provider === 'claude' ? ' active' : ''}`} onClick={() => handleProviderChange('claude')}>Claude</button>
              <button className={`settings-provider-btn${provider === 'openai' ? ' active' : ''}`} onClick={() => handleProviderChange('openai')}>OpenAI</button>
            </div>
            <div className="settings-hint">Anything else goes under Your Providers below.</div>
          </div>
          {provider !== '' && <>
            <div className="dialog-field">
              <label className="dialog-label">Model</label>
              <select className="dialog-select" value={model} onChange={e => setModel(e.target.value)}>
                {modelOptions.map(m => <option key={m.value} value={m.value}>{m.label}</option>)}
              </select>
              <div className="settings-hint">
                {liveModels.length
                  ? `${liveModels.length} models available on your account.`
                  : 'Add a key to list the models your account can actually use.'}
              </div>
            </div>
            <div className="dialog-field">
              <label className="dialog-label">API Key</label>
              <div className="settings-path-row">
                <input
                  className="dialog-input"
                  type={showKey ? 'text' : 'password'}
                  value={apiKey}
                  onChange={e => setApiKey(e.target.value)}
                  placeholder={hasStoredKey
                    ? '••••••••  key stored — enter a new key to replace it'
                    : provider === 'claude' ? 'sk-ant-...' : 'sk-...'}
                  autoComplete="off"
                />
                <button className="dialog-btn settings-browse-btn" onClick={() => setShowKey(v => !v)}>
                  {showKey ? 'Hide' : 'Show'}
                </button>
                {hasStoredKey && (
                  <button className="dialog-btn settings-browse-btn" onClick={onClearKey} title="Remove the stored key">
                    Clear
                  </button>
                )}
              </div>
              <div className="settings-hint">Stored in your system keychain — never written to disk in plain text or transmitted to Draftline servers.</div>
            </div>
          </>}
        </>}

        {activeProvider && (
          <div className="settings-local-info">
            Using <strong>{activeProvider.nickname}</strong>
            {activeProvider.model ? <> with <strong>{activeProvider.model}</strong></> : ' — no model set yet'}.
            Edit it under Your Providers below.
          </div>
        )}

        <AIProvidersList providers={providers} onChanged={onProvidersChanged} />

        {/* Prose guide — always visible when AI enabled */}
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

        <div className="settings-section-label">
          Debug Logging
          <label className="settings-toggle" title="Write AI prompts and responses to local log files">
            <input type="checkbox" checked={debugLogging} onChange={e => setDebugLogging(e.target.checked)} />
            <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
            <span className="settings-toggle-label">{debugLogging ? 'Enabled' : 'Disabled'}</span>
          </label>
        </div>
        <div className="settings-hint">When enabled, AI requests (including manuscript text) are logged to local files for troubleshooting. Off by default.</div>
      </>}
    </>
  )
}

// CLI-account setup sub-section, shared by Claude Code (Claude.ai accounts)
// and Codex (ChatGPT accounts) — identical lifecycle, different wording.
const CLI_FLAVORS = {
  claude: {
    title: 'Claude Code',
    desc: 'Uses your Claude.ai account — no separate API key needed. AI rewrites are charged to your Claude subscription.',
    installStep: 'Install runtime & Claude Code CLI',
    signinStep: 'Sign in with your Claude account',
    signinHint: 'Click below to sign in with your Claude.ai account. A browser window will open — complete the login, then come back here.',
    signinBtn: 'Sign in with Claude.ai…',
    setupBtn: 'Set up Claude Code automatically',
  },
  codex: {
    title: 'Codex',
    desc: 'Uses your ChatGPT account (Plus, Pro, or Team) through the OpenAI Codex CLI — no separate API key needed.',
    installStep: 'Install runtime & Codex CLI',
    signinStep: 'Sign in with your ChatGPT account',
    signinHint: 'Click below to sign in with your ChatGPT account. A browser window will open — complete the login, then come back here.',
    signinBtn: 'Sign in with ChatGPT…',
    setupBtn: 'Set up Codex automatically',
  },
} as const

interface CLISetupSectionProps {
  flavor: keyof typeof CLI_FLAVORS
  ccStatus: AIStudioSectionProps['ccStatus']
  ccChecking: boolean
  ccSetupStep: AIStudioSectionProps['ccSetupStep']
  ccSetupLog: string[]
  onCheckCC: () => void
  onSetup: () => void
  onOpenAuth: () => void
  model: string
  setModel: (v: string) => void
}

function CLISetupSection({
  flavor,
  ccStatus, ccChecking, ccSetupStep, ccSetupLog,
  onCheckCC, onSetup, onOpenAuth,
  model, setModel,
}: CLISetupSectionProps) {
  const f = CLI_FLAVORS[flavor]
  return (
    <>
      <div className="settings-cc-card">
        <div className="settings-cc-header">
          <span className="settings-cc-title">{f.title}</span>
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
          <button className="settings-cc-recheck" onClick={onCheckCC} title="Re-check" disabled={ccSetupStep === 'running'}>
            <svg width="11" height="11" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
              <path d="M10.5 2A5 5 0 1 0 11 6.5"/><polyline points="10.5 1 10.5 3.5 8 3.5"/>
            </svg>
          </button>
        </div>
        <p className="settings-cc-desc">{f.desc}</p>

        {/* Setup wizard (not installed or setup in progress) */}
        {ccSetupStep !== 'done' && !(ccStatus?.installed && ccStatus.authenticated) && (
          <div className="settings-cc-setup">
            {/* Step list */}
            <div className="settings-cc-steps">
              <div className={`settings-cc-step ${ccStatus?.installed ? 'done' : ccSetupStep === 'running' ? 'active' : ''}`}>
                <span className="settings-cc-step-icon">
                  {ccStatus?.installed ? '✓' : ccSetupStep === 'running' ? '⟳' : '○'}
                </span>
                <span>{f.installStep}</span>
              </div>
              <div className={`settings-cc-step ${ccStatus?.authenticated ? 'done' : (ccSetupStep === 'auth' || ccSetupStep === 'auth-waiting') ? 'active' : ''}`}>
                <span className="settings-cc-step-icon">
                  {ccStatus?.authenticated ? '✓' : (ccSetupStep === 'auth' || ccSetupStep === 'auth-waiting') ? '⟳' : '○'}
                </span>
                <span>{f.signinStep}</span>
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
              <button className="dialog-btn primary" style={{ marginTop: 12 }} onClick={onSetup}>
                {f.setupBtn}
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
                  {f.signinHint}
                </p>
                <button className="dialog-btn primary" onClick={onOpenAuth}>
                  {f.signinBtn}
                </button>
              </div>
            )}
            {ccSetupStep === 'auth-waiting' && (
              <div style={{ marginTop: 12 }}>
                <p className="settings-hint" style={{ marginBottom: 8 }}>
                  Browser opened — complete the sign-in, then click Verify below.
                </p>
                <button className="dialog-btn" onClick={onCheckCC}>
                  Verify sign-in
                </button>
              </div>
            )}
            {ccSetupStep === 'error' && (
              <button className="dialog-btn" style={{ marginTop: 12 }} onClick={onSetup}>
                Retry
              </button>
            )}
          </div>
        )}
      </div>

      {(ccStatus?.installed && ccStatus.authenticated) && (
        flavor === 'claude' ? (
          <div className="dialog-field" style={{ marginTop: 12 }}>
            <label className="dialog-label">Model</label>
            <select className="dialog-select" value={model || 'claude-sonnet-5'} onChange={e => setModel(e.target.value)}>
              {CLAUDE_MODELS.map(m => <option key={m.value} value={m.value}>{m.label}</option>)}
            </select>
          </div>
        ) : (
          <div className="dialog-field" style={{ marginTop: 12 }}>
            <label className="dialog-label">Model override (optional)</label>
            <input
              className="dialog-input"
              value={model}
              onChange={e => setModel(e.target.value)}
              placeholder="Leave blank to use the Codex CLI's default model"
            />
          </div>
        )
      )}
    </>
  )
}
