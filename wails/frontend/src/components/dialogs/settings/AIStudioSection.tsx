// AI Studio Settings Section - Claude Code, API, Local Model

import type { AIStudioSectionProps, AIProvider } from './types'
import { CLAUDE_MODELS, OPENAI_MODELS, GEMINI_MODELS, GROK_MODELS, DEFAULT_MODELS } from './constants'
import { useState } from 'react'

export default function AIStudioSection({
  aiEnabled, setAiEnabled,
  aiMode, setAiMode,
  provider, setProvider,
  apiKey, setApiKey,
  hasStoredKey, onClearKey,
  debugLogging, setDebugLogging,
  model, setModel,
  localEndpoint, setLocalEndpoint,
  localModel, setLocalModel,
  proseGuide, setProseGuide,
  ccStatus, ccChecking, ccSetupStep, ccSetupLog,
  onCheckCC, onSetup, onOpenAuth,
  cxStatus, cxChecking, cxSetupStep, cxSetupLog,
  onCheckCx, onSetupCx, onOpenCxAuth,
  testStatus, testMsg, onTestLocal,
}: AIStudioSectionProps) {
  const [showKey, setShowKey] = useState(false)

  const modelOptions = provider === 'claude' ? CLAUDE_MODELS
    : provider === 'openai' ? OPENAI_MODELS
    : provider === 'gemini' ? GEMINI_MODELS
    : provider === 'grok' ? GROK_MODELS
    : []

  function handleProviderChange(p: AIProvider) {
    setProvider(p)
    setModel(DEFAULT_MODELS[p] || '')
  }

  function handleModeChange(m: typeof aiMode) {
    setAiMode(m)
    if (m === 'claudecode' && !model) setModel('claude-sonnet-4-6')
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
          <label className="dialog-label">AI Source</label>
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
            <button className={`settings-theme-btn${aiMode === 'local' ? ' active' : ''}`} onClick={() => handleModeChange('local')}>
              Local Model
            </button>
          </div>
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

        {/* Local Model */}
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
              onChange={e => setLocalEndpoint(e.target.value)}
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
            <button className="dialog-btn" onClick={onTestLocal} disabled={testStatus === 'testing'}>
              {testStatus === 'testing' ? 'Testing…' : 'Test Connection'}
            </button>
            {testStatus === 'ok' && <span className="settings-test-ok">✓ {testMsg}</span>}
            {testStatus === 'error' && <span className="settings-test-error">{testMsg}</span>}
          </div>
        </>}

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
            <select className="dialog-select" value={model || 'claude-sonnet-4-6'} onChange={e => setModel(e.target.value)}>
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
