import { useState, useEffect, useRef } from 'react'
import { useBookStore } from '../store/bookStore'
import { useAppStore } from '../store/appStore'
import { RewriteText, CancelRewrite } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { diffContent, assembleParagraphs, type ParagraphDiff } from '../utils/diff'
import type { Character, CharacterRole } from '../types/draftline'

type Tab = 'ai' | 'bible' | 'spelling' | 'index'
type AIMode = 'line_edit' | 'copy_edit' | 'dev_edit' | 'expand' | 'smooth' | 'voice_check'
type AIState = 'idle' | 'loading' | 'voice' | 'error'
type BibleSection = 'characters' | 'plot' | 'timeline'

const AI_MODES: { id: AIMode; label: string; desc: string }[] = [
  { id: 'line_edit',    label: 'Line Edit',   desc: 'Prose rhythm and sentence variety' },
  { id: 'copy_edit',   label: 'Copy Edit',   desc: 'Grammar, punctuation, consistency' },
  { id: 'dev_edit',    label: 'Dev Edit',    desc: 'Pacing, transitions, scene structure' },
  { id: 'expand',      label: 'Expand',      desc: 'Sensory detail, texture, show vs. tell' },
  { id: 'smooth',      label: 'Smooth',      desc: 'Remove repetition, improve flow' },
  { id: 'voice_check', label: 'Voice Check', desc: 'POV, tense, and narrative consistency' },
]

function genId(): string {
  return Math.random().toString(36).slice(2) + Date.now().toString(36)
}

// ── Main panel ──────────────────────────────────────────────────────────────

export default function ToolsPanel() {
  const [activeTab, setActiveTab] = useState<Tab>('ai')
  const { rightPanelOpen, toggleRightPanel } = useBookStore()

  return (
    <div className={`tools-panel${rightPanelOpen ? '' : ' collapsed'}`}>
      <div className="tools-panel-inner">
        <div className="tools-tabs">
          <button className="panel-collapse-btn" onClick={toggleRightPanel} title="Collapse panel">
            <svg width="6" height="10" viewBox="0 0 6 10" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
              <path d="M1 1l4 4-4 4" />
            </svg>
          </button>
          <button className={`tools-tab${activeTab === 'ai' ? ' active' : ''}`} onClick={() => setActiveTab('ai')}>AI</button>
          <button className={`tools-tab${activeTab === 'bible' ? ' active' : ''}`} onClick={() => setActiveTab('bible')}>Story Bible</button>
          <button className={`tools-tab${activeTab === 'spelling' ? ' active' : ''}`} onClick={() => setActiveTab('spelling')}>Spell</button>
          <button className={`tools-tab${activeTab === 'index' ? ' active' : ''}`} onClick={() => setActiveTab('index')}>Index</button>
        </div>
        <div className="tools-content">
          {activeTab === 'ai'       && <AiStudioTab />}
          {activeTab === 'bible'    && <StoryBibleTab />}
          {activeTab === 'spelling' && <SpellingTab />}
          {activeTab === 'index'    && <IndexTab />}
        </div>
      </div>
    </div>
  )
}

// ── AI Studio ───────────────────────────────────────────────────────────────

function AiStudioTab() {
  const { book, currentSection, currentIndex, setPendingDiff } = useBookStore()
  const { settings, openSettings } = useAppStore()

  const [aiMode, setAiMode]             = useState<AIMode>('line_edit')
  const [bookScope, setBookScope]       = useState(false)
  const [aiState, setAiState]           = useState<AIState>('idle')
  const [error, setError]               = useState('')
  const [voiceResult, setVoiceResult]   = useState('')
  const [bookFindings, setBookFindings] = useState<{title: string; findings: string}[]>([])
  const [showBookModal, setShowBookModal] = useState(false)
  const [bookProgress, setBookProgress] = useState(0)
  const [showImport, setShowImport]     = useState(false)
  const [importText, setImportText]     = useState('')

  const aiConfigured = settings.ai_enabled && (
    settings.ai_mode === 'claudecode' ||
    (settings.ai_mode === 'api' && settings.ai_provider !== '' && settings.ai_api_key !== '') ||
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
    if (settings.ai_mode === 'local') return settings.ai_local_model || 'Local AI'
    return settings.ai_provider === 'claude' ? 'Claude'
      : settings.ai_provider === 'openai' ? 'OpenAI'
      : 'AI'
  }

  const currentMode = AI_MODES.find(m => m.id === aiMode)!

  async function handleRun() {
    if (aiMode === 'voice_check' && bookScope) {
      await handleBookVoiceCheck()
      return
    }
    const html = getCurrentHTML()
    if (!html || html === '<p></p>') return
    setAiState('loading')
    setError('')
    try {
      const res = await RewriteText(html, aiMode)
      if (res.error) {
        setError(res.error)
        setAiState('error')
      } else if (aiMode === 'voice_check') {
        setVoiceResult(res.result)
        setAiState('voice')
      } else {
        const d = diffContent(html, res.result)
        setPendingDiff({ diffs: d, originalHtml: html })
        setAiState('idle')
      }
    } catch (e) {
      setError(String(e))
      setAiState('error')
    }
  }

  async function handleBookVoiceCheck() {
    if (!book) return
    setAiState('loading')
    setBookFindings([])
    const results: {title: string; findings: string}[] = []
    for (let i = 0; i < book.body.length; i++) {
      setBookProgress(i + 1)
      try {
        const res = await RewriteText(book.body[i].content, 'voice_check')
        results.push({ title: book.body[i].title, findings: res.error || res.result })
      } catch (e) {
        results.push({ title: book.body[i].title, findings: 'Error: ' + String(e) })
      }
    }
    setBookFindings(results)
    setAiState('idle')
    setShowBookModal(true)
  }

  function handleImportCompare() {
    const html = getCurrentHTML()
    if (!html || !importText.trim()) return
    const pasted = importText.trim().startsWith('<')
      ? importText
      : importText.split(/\n{2,}/).map(p => `<p>${p.trim()}</p>`).join('\n')
    const d = diffContent(html, pasted)
    setPendingDiff({ diffs: d, originalHtml: html })
    setShowImport(false)
    setImportText('')
  }

  if (!book) {
    return <div className="tool-empty-state">Open or create a project to use AI tools.</div>
  }

  // ── Render states ──

  if (aiState === 'loading') {
    return (
      <LoadingPane
        label={
          aiMode === 'voice_check' && bookScope
            ? `Voice check — chapter ${bookProgress} of ${book.body.length}`
            : `${currentMode.label} · ${getAiLabel()}`
        }
        onCancel={() => {
          CancelRewrite()
          setAiState('idle')
        }}
      />
    )
  }

  if (aiState === 'error') {
    return (
      <div className="ai-error-pane">
        <div className="ai-error-msg">{error}</div>
        <button className="ai-link-btn" onClick={() => setAiState('idle')}>← Back</button>
      </div>
    )
  }

  if (aiState === 'voice') {
    return (
      <div>
        <div className="ai-result-bar">
          <span>Voice Analysis — {book.body[currentIndex]?.title ?? 'current chapter'}</span>
          <button className="ai-link-btn inline" onClick={() => setAiState('idle')}>← Back</button>
        </div>
        <div className="voice-findings">{voiceResult}</div>
      </div>
    )
  }

  // ── Idle state ──
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

      {/* Voice scope */}
      {aiMode === 'voice_check' && (
        <div className="ai-scope-row">
          <button className={`scope-pill${!bookScope ? ' active' : ''}`} onClick={() => setBookScope(false)}>Chapter</button>
          <button className={`scope-pill${bookScope ? ' active' : ''}`} onClick={() => setBookScope(true)}>Full Book</button>
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
      ) : (
        <div className="ai-actions">
          {aiConfigured ? (
            <button className="ai-run-btn" onClick={handleRun}>
              {aiMode === 'voice_check' && bookScope ? 'Check All Chapters' : `Run ${currentMode.label}`}
            </button>
          ) : (
            <button className="ai-run-btn configure" onClick={openSettings}>Configure AI ›</button>
          )}
          {aiConfigured && aiMode !== 'voice_check' && (
            <button className="ai-link-btn" onClick={() => setShowImport(true)}>
              or compare with imported draft ›
            </button>
          )}
        </div>
      )}

      {/* Full book analysis modal */}
      {showBookModal && (
        <div className="dialog-overlay">
          <div className="dialog book-analysis-modal">
            <div className="dialog-title">Voice Check — Full Manuscript</div>
            <div className="book-analysis-findings">
              {bookFindings.map((f, i) => (
                <div key={i} className="analysis-finding">
                  <div className="analysis-finding-title">{f.title}</div>
                  <div className="analysis-finding-text">{f.findings}</div>
                </div>
              ))}
            </div>
            <div className="dialog-actions">
              <button className="dialog-btn primary" onClick={() => setShowBookModal(false)}>Close</button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}

// ── Loading pane ─────────────────────────────────────────────────────────────

function LoadingPane({ label, onCancel }: { label: string; onCancel: () => void }) {
  const [elapsed, setElapsed]       = useState(0)
  const [logLines, setLogLines]     = useState<string[]>([])
  const [streamText, setStreamText] = useState('')
  const logRef    = useRef<HTMLDivElement>(null)
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

// ── Story Bible ──────────────────────────────────────────────────────────────

function StoryBibleTab() {
  const [section, setSection] = useState<BibleSection>('characters')
  return (
    <>
      <div className="story-bible-subnav">
        <button className={`bible-nav-btn${section === 'characters' ? ' active' : ''}`} onClick={() => setSection('characters')}>Characters</button>
        <button className={`bible-nav-btn${section === 'plot' ? ' active' : ''}`} onClick={() => setSection('plot')}>Plot</button>
        <button className={`bible-nav-btn${section === 'timeline' ? ' active' : ''}`} onClick={() => setSection('timeline')}>Timeline</button>
      </div>
      {section === 'characters' && <CharactersSection />}
      {section === 'plot'       && <PlotSection />}
      {section === 'timeline'   && <TimelineSection />}
    </>
  )
}

function CharactersSection() {
  const { book, addCharacter, updateCharacter, deleteCharacter } = useBookStore()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [addingNew, setAddingNew] = useState(false)

  if (!book) return <div className="tool-empty-state">Open a project to manage characters.</div>

  const characters = book.story_bible?.characters ?? []

  return (
    <>
      {addingNew && (
        <CharacterForm char={null} onSave={c => { addCharacter(c); setAddingNew(false) }} onCancel={() => setAddingNew(false)} />
      )}
      {characters.map(char =>
        editingId === char.id ? (
          <CharacterForm key={char.id} char={char} onSave={c => { updateCharacter(c); setEditingId(null) }} onCancel={() => setEditingId(null)} />
        ) : (
          <CharacterCard key={char.id} char={char} onEdit={() => setEditingId(char.id)} onDelete={() => deleteCharacter(char.id)} />
        )
      )}
      {!addingNew && (
        <button className="tool-card-btn" style={{ marginTop: 4 }} onClick={() => setAddingNew(true)}>+ Add Character</button>
      )}
    </>
  )
}

function CharacterCard({ char, onEdit, onDelete }: { char: Character; onEdit: () => void; onDelete: () => void }) {
  const [expanded, setExpanded] = useState(false)
  return (
    <div className="character-card" onClick={() => setExpanded(v => !v)}>
      <div className="character-card-header">
        <span className="character-name">{char.name || 'Unnamed'}</span>
        <span className={`character-role ${char.role}`}>{char.role}</span>
      </div>
      {char.description && <div className="character-desc-preview">{char.description}</div>}
      {expanded && (
        <div className="character-expanded" onClick={e => e.stopPropagation()}>
          {char.appearance  && <div className="char-field"><span className="char-field-label">Appearance</span><span>{char.appearance}</span></div>}
          {char.personality && <div className="char-field"><span className="char-field-label">Personality</span><span>{char.personality}</span></div>}
          {char.motivation  && <div className="char-field"><span className="char-field-label">Motivation</span><span>{char.motivation}</span></div>}
          {char.notes       && <div className="char-field"><span className="char-field-label">Notes</span><span>{char.notes}</span></div>}
          <div className="character-card-actions">
            <button className="tool-card-btn secondary" style={{ fontSize: 11, padding: '2px 10px' }} onClick={e => { e.stopPropagation(); onEdit() }}>Edit</button>
            <button className="tool-card-btn secondary" style={{ fontSize: 11, padding: '2px 10px', color: '#E06C75' }} onClick={e => { e.stopPropagation(); onDelete() }}>Delete</button>
          </div>
        </div>
      )}
    </div>
  )
}

function CharacterForm({ char, onSave, onCancel }: { char: Character | null; onSave: (c: Character) => void; onCancel: () => void }) {
  const [name, setName]       = useState(char?.name ?? '')
  const [role, setRole]       = useState<CharacterRole | string>(char?.role ?? 'supporting')
  const [desc, setDesc]       = useState(char?.description ?? '')
  const [app, setApp]         = useState(char?.appearance ?? '')
  const [persona, setPersona] = useState(char?.personality ?? '')
  const [motiv, setMotiv]     = useState(char?.motivation ?? '')
  const [notes, setNotes]     = useState(char?.notes ?? '')

  return (
    <div className="character-form">
      <div className="character-form-title">{char ? 'Edit Character' : 'New Character'}</div>
      <div className="character-form-row">
        <div className="dialog-field" style={{ margin: 0 }}>
          <label className="dialog-label">Name</label>
          <input className="dialog-input" value={name} onChange={e => setName(e.target.value)} placeholder="Character name" autoFocus />
        </div>
        <div className="dialog-field" style={{ margin: 0 }}>
          <label className="dialog-label">Role</label>
          <select className="dialog-select" value={role} onChange={e => setRole(e.target.value as CharacterRole)}>
            <option value="protagonist">Protagonist</option>
            <option value="antagonist">Antagonist</option>
            <option value="supporting">Supporting</option>
            <option value="minor">Minor</option>
            <option value="other">Other</option>
          </select>
        </div>
      </div>
      <div className="dialog-field" style={{ marginTop: 6 }}>
        <label className="dialog-label">Description</label>
        <textarea className="dialog-input" rows={2} value={desc} onChange={e => setDesc(e.target.value)} placeholder="Brief overview of this character's role" style={{ resize: 'vertical' }} />
      </div>
      <div className="dialog-field" style={{ marginTop: 4 }}>
        <label className="dialog-label">Appearance</label>
        <textarea className="dialog-input" rows={2} value={app} onChange={e => setApp(e.target.value)} placeholder="Physical description, distinguishing features" style={{ resize: 'vertical' }} />
      </div>
      <div className="character-form-row" style={{ marginTop: 4 }}>
        <div className="dialog-field" style={{ margin: 0 }}>
          <label className="dialog-label">Personality</label>
          <textarea className="dialog-input" rows={2} value={persona} onChange={e => setPersona(e.target.value)} placeholder="Traits, quirks, voice" style={{ resize: 'vertical' }} />
        </div>
        <div className="dialog-field" style={{ margin: 0 }}>
          <label className="dialog-label">Motivation</label>
          <textarea className="dialog-input" rows={2} value={motiv} onChange={e => setMotiv(e.target.value)} placeholder="Goals, fears, driving need" style={{ resize: 'vertical' }} />
        </div>
      </div>
      <div className="dialog-field" style={{ marginTop: 4 }}>
        <label className="dialog-label">Notes</label>
        <textarea className="dialog-input" rows={2} value={notes} onChange={e => setNotes(e.target.value)} placeholder="Arc beats, continuity flags, backstory" style={{ resize: 'vertical' }} />
      </div>
      <div className="ai-actions" style={{ marginTop: 8 }}>
        <button className="ai-run-btn" onClick={() => { if (name.trim()) onSave({ id: char?.id ?? genId(), name: name.trim(), role, description: desc, appearance: app, personality: persona, motivation: motiv, notes }) }} disabled={!name.trim()}>
          {char ? 'Save Changes' : 'Add Character'}
        </button>
        <button className="ai-link-btn" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}

function PlotSection() {
  const { book, updateStoryBibleText } = useBookStore()
  if (!book) return <div className="tool-empty-state">Open a project first.</div>
  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Plot Bible</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>Outline acts, subplots, themes, and unresolved threads.</p>
      <textarea
        className="bible-textarea"
        value={book.story_bible?.plot_notes ?? ''}
        onChange={e => updateStoryBibleText('plot_notes', e.target.value)}
        placeholder={"Act 1:\n  Opening image...\n  Inciting incident...\n\nAct 2:\n  ...\n\nThemes:\n  ..."}
      />
    </div>
  )
}

function TimelineSection() {
  const { book, updateStoryBibleText } = useBookStore()
  if (!book) return <div className="tool-empty-state">Open a project first.</div>
  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Story Timeline</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>Chronological events in story time. One event per line.</p>
      <textarea
        className="bible-textarea"
        value={book.story_bible?.timeline ?? ''}
        onChange={e => updateStoryBibleText('timeline', e.target.value)}
        placeholder={"Day 1 — Marcus arrives in the city (Ch. 1)\nDay 1, evening — He meets Clara (Ch. 2)\nDay 3 — The letter arrives (Ch. 4)"}
      />
    </div>
  )
}

// ── Placeholders ─────────────────────────────────────────────────────────────

function SpellingTab() {
  return (
    <>
      <div className="tool-card">
        <div className="tool-card-title">Spell Check</div>
        <div className="tool-card-desc">Scan the manuscript for misspellings and common typos.</div>
        <button className="tool-card-btn" disabled>Run Spell Check</button>
      </div>
      <div className="tool-card">
        <div className="tool-card-title">Grammar Audit</div>
        <div className="tool-card-desc">Check for grammar issues, passive voice overuse, and awkward constructions.</div>
        <button className="tool-card-btn" disabled>Run Grammar Audit</button>
      </div>
    </>
  )
}

function IndexTab() {
  return (
    <>
      <div className="tool-card">
        <div className="tool-card-title">Character Index</div>
        <div className="tool-card-desc">Track character appearances and first/last mentions across the manuscript.</div>
        <button className="tool-card-btn" disabled>Build Character Index</button>
      </div>
      <div className="tool-card">
        <div className="tool-card-title">Locations</div>
        <div className="tool-card-desc">Map location references across all chapters.</div>
        <button className="tool-card-btn" disabled>Build Location Index</button>
      </div>
      <div className="tool-card">
        <div className="tool-card-title">Plot Threads</div>
        <div className="tool-card-desc">Track open and resolved plot threads.</div>
        <button className="tool-card-btn" disabled>Analyse Plot Threads</button>
      </div>
    </>
  )
}
