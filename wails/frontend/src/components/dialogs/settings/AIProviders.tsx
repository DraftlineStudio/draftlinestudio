import { useState } from 'react'
import { SaveAIProvider, DeleteAIProvider, SetProviderKey, TestAIEndpoint } from '../../../../wailsjs/go/main/App'
import type { types } from '../../../../wailsjs/go/models'

type Draft = { id: string; nickname: string; kind: string; base_url: string; model: string }

const BLANK: Draft = { id: '', nickname: '', kind: 'cloud', base_url: '', model: '' }

// Services that used to be built in, offered as a starting point since both
// speak the OpenAI wire format.
const EXAMPLES = [
  { label: 'Gemini', url: 'https://generativelanguage.googleapis.com/v1beta/openai' },
  { label: 'Grok', url: 'https://api.x.ai/v1' },
]

interface Props {
  providers: types.AIProvider[]
  onChanged: () => void
}

export default function AIProviders({ providers, onChanged }: Props) {
  const [draft, setDraft] = useState<Draft | null>(null)
  const [apiKey, setApiKey] = useState('')
  const [status, setStatus] = useState<'idle' | 'testing' | 'ok' | 'error'>('idle')
  const [msg, setMsg] = useState('')

  function edit(p?: types.AIProvider) {
    setDraft(p
      ? { id: p.id, nickname: p.nickname, kind: p.kind, base_url: p.base_url, model: p.model }
      : { ...BLANK })
    setApiKey('')
    setStatus('idle')
    setMsg('')
  }

  async function test() {
    if (!draft) return
    setStatus('testing'); setMsg('')
    try {
      const res = await TestAIEndpoint(draft.base_url, apiKey)
      if (res.error) { setStatus('error'); setMsg(res.error) }
      else { setStatus('ok'); setMsg('Connected') }
    } catch (e) {
      setStatus('error'); setMsg(String(e))
    }
  }

  async function save() {
    if (!draft) return
    try {
      const id = await SaveAIProvider(draft as types.AIProvider)
      if (draft.kind === 'cloud' && apiKey.trim()) await SetProviderKey(id, apiKey.trim())
      setDraft(null)
      onChanged()
    } catch (e) {
      setStatus('error'); setMsg(String(e))
    }
  }

  async function remove(id: string) {
    try {
      await DeleteAIProvider(id)
      onChanged()
    } catch { /* the list refreshes either way */ }
  }

  return (
    <>
      <div className="settings-section-label">Your Providers</div>
      <div className="settings-hint">
        Any service that speaks the OpenAI chat format, running locally or in the cloud.
        You pick the model, so nothing here breaks when a provider retires one.
      </div>

      {providers.length > 0 && (
        <div className="settings-provider-list">
          {providers.map(p => (
            <div className="settings-provider-item" key={p.id}>
              <div>
                <div className="settings-provider-name">{p.nickname}</div>
                <div className="settings-provider-meta">
                  {p.kind === 'local' ? 'Local' : 'Cloud'} &middot; {p.model || 'no model set'}
                </div>
              </div>
              <div className="settings-path-row">
                <button className="dialog-btn settings-browse-btn" onClick={() => edit(p)}>Edit</button>
                <button className="dialog-btn settings-browse-btn" onClick={() => remove(p.id)}>Remove</button>
              </div>
            </div>
          ))}
        </div>
      )}

      {!draft && (
        <button className="dialog-btn" onClick={() => edit()}>Add Provider&hellip;</button>
      )}

      {draft && (
        <div className="settings-provider-form">
          <div className="dialog-field">
            <label className="dialog-label">Type</label>
            <div className="settings-theme-row">
              <button
                className={`settings-theme-btn${draft.kind === 'cloud' ? ' active' : ''}`}
                onClick={() => setDraft({ ...draft, kind: 'cloud' })}
              >Cloud</button>
              <button
                className={`settings-theme-btn${draft.kind === 'local' ? ' active' : ''}`}
                onClick={() => setDraft({ ...draft, kind: 'local' })}
              >Local</button>
            </div>
            <div className="settings-hint">
              {draft.kind === 'cloud'
                ? 'Your API key is sent to this endpoint as a bearer token.'
                : 'No credentials are sent. For Ollama, LM Studio, or any OpenAI-compatible server on this machine.'}
            </div>
          </div>

          <div className="dialog-field">
            <label className="dialog-label">Name</label>
            <input
              className="dialog-input"
              value={draft.nickname}
              onChange={e => setDraft({ ...draft, nickname: e.target.value })}
              placeholder={draft.kind === 'cloud' ? 'e.g. OpenRouter, Gemini, Together' : 'e.g. Ollama'}
            />
          </div>

          <div className="dialog-field">
            <label className="dialog-label">Endpoint</label>
            <input
              className="dialog-input"
              value={draft.base_url}
              onChange={e => setDraft({ ...draft, base_url: e.target.value })}
              placeholder={draft.kind === 'cloud' ? 'https://.../v1' : 'http://localhost:11434/v1'}
            />
            <div className="settings-hint">
              {draft.kind === 'cloud' ? (
                <>
                  The base URL, without <code>/chat/completions</code>. For example{' '}
                  {EXAMPLES.map((ex, i) => (
                    <span key={ex.url}>
                      {i > 0 && ', '}
                      <button
                        className="settings-link-btn"
                        onClick={() => setDraft({ ...draft, base_url: ex.url })}
                      >{ex.label}</button>
                    </span>
                  ))}.
                </>
              ) : 'Ollama serves this at http://localhost:11434/v1'}
            </div>
          </div>

          <div className="dialog-field">
            <label className="dialog-label">Model</label>
            <input
              className="dialog-input"
              value={draft.model}
              onChange={e => setDraft({ ...draft, model: e.target.value })}
              placeholder={draft.kind === 'cloud' ? 'the model name this provider uses' : 'e.g. llama3, mistral, phi3'}
            />
          </div>

          {draft.kind === 'cloud' && (
            <div className="dialog-field">
              <label className="dialog-label">API Key</label>
              <input
                className="dialog-input"
                type="password"
                value={apiKey}
                onChange={e => setApiKey(e.target.value)}
                placeholder={draft.id ? 'key stored - enter a new key to replace it' : ''}
                autoComplete="off"
              />
              <div className="settings-hint">Stored in your system keychain, never in plain text.</div>
            </div>
          )}

          <div className="settings-local-test-row">
            <button className="dialog-btn" onClick={test} disabled={status === 'testing'}>
              {status === 'testing' ? 'Testing...' : 'Test Connection'}
            </button>
            <button className="dialog-btn primary" onClick={save}>Save</button>
            <button className="dialog-btn" onClick={() => setDraft(null)}>Cancel</button>
            {status === 'ok' && <span className="settings-test-ok">&#10003; {msg}</span>}
            {status === 'error' && <span className="settings-test-error">{msg}</span>}
          </div>
        </div>
      )}
    </>
  )
}
