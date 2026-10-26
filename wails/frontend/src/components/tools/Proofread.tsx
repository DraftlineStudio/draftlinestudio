// Spelling & Grammar — the sidebar's proofing panel, built to the design
// comp in the private reference assets: a filterable list of every detection
// in the open chapter, one expanded in place with its context and
// suggestions, a one-click typography fix in the footer, and a Checks &
// Dictionary subview. Styling is inline on the design tokens; classes can
// consolidate when the stylesheet is reformatted.

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { Editor } from '@tiptap/core'
import { useShallow } from 'zustand/react/shallow'
import { useAppStore } from '../../store/appStore'
import { useBookStore } from '../../store/bookStore'
import { getSectionArray } from '../../store/chapters'
import { useEditorStore } from '../../store/editorStore'
import { clearSpotlight, updateSpotlight } from '../../extensions/RangeSpotlight'
import { fixChapterTypography } from '../../services/typographyFix'
import { getDictionaryRoot, getSuggestions, normalizeCustomDictionary } from '../../services/spellCheck'
import {
  applyDetection, collectDetections, detectionKey, globalChapterIndex,
  styleFindingsFor, typographyFixableCount, type Detection,
} from '../../services/proofread'

const KIND = {
  spelling: { label: 'Spelling', color: 'var(--status-error)', mark: 'rgba(224,108,117,0.16)' },
  grammar: { label: 'Grammar', color: 'var(--status-warning)', mark: 'rgba(229,192,123,0.16)' },
  style: { label: 'Style', color: 'var(--app-accent-text)', mark: 'rgba(90,175,224,0.16)' },
} as const

type Filter = 'all' | Detection['kind']

function liveEditor(): Editor | null {
  return (useEditorStore.getState().editorRef as unknown as Editor | null) ?? null
}

function isEditableTarget(target: EventTarget | null): boolean {
  const element = target as HTMLElement | null
  if (!element) return false
  return element.isContentEditable || element.tagName === 'INPUT' || element.tagName === 'TEXTAREA'
}

const chipStyle = (active: boolean): React.CSSProperties => ({
  fontSize: 11, padding: '3px 7px', borderRadius: 3, border: 0, cursor: 'pointer',
  fontFamily: 'inherit',
  background: active ? 'var(--bg-active)' : 'transparent',
  color: active ? 'var(--text-primary)' : 'var(--text-secondary)',
})

const iconButtonStyle: React.CSSProperties = {
  width: 26, height: 26, border: 0, background: 'transparent', color: 'var(--icon-color)',
  borderRadius: 3, display: 'grid', placeItems: 'center', cursor: 'pointer',
}

const actionLinkStyle: React.CSSProperties = {
  cursor: 'pointer', color: 'var(--text-secondary)', fontSize: 11, background: 'transparent',
  border: 0, padding: 0, fontFamily: 'inherit',
}

const kbdStyle: React.CSSProperties = {
  border: '1px solid var(--border)', borderRadius: 3, padding: '0 4px', fontSize: 10,
  background: 'var(--bg-surface-alt)', color: 'var(--text-secondary)',
}

export default function Proofread() {
  const { settings, saveSettings } = useAppStore(useShallow(s => ({ settings: s.settings, saveSettings: s.saveSettings })))
  const {
    book, currentSection, currentIndex,
    setStyleDecision, addStyleIgnoredWord, removeStyleIgnoredWord, clearStyleDecisions,
  } = useBookStore(useShallow(s => ({
    book: s.book,
    currentSection: s.currentSection,
    currentIndex: s.currentIndex,
    setStyleDecision: s.setStyleDecision,
    addStyleIgnoredWord: s.addStyleIgnoredWord,
    removeStyleIgnoredWord: s.removeStyleIgnoredWord,
    clearStyleDecisions: s.clearStyleDecisions,
  })))

  const [view, setView] = useState<'list' | 'settings'>('list')
  const [filter, setFilter] = useState<Filter>('all')
  const [detections, setDetections] = useState<Detection[]>([])
  const [selectedKey, setSelectedKey] = useState<string | null>(null)
  const [suggestions, setSuggestions] = useState<string[]>([])
  const [wordCount, setWordCount] = useState(0)
  const [newWord, setNewWord] = useState('')
  const skippedRef = useRef(new Set<string>())
  const [skippedCount, setSkippedCount] = useState(0)

  const backendFindings = useMemo(() => {
    if (!settings.style_check_enabled || !book) return []
    const globalIndex = globalChapterIndex(
      currentSection, currentIndex,
      book.front_matter?.length ?? 0, book.body?.length ?? 0)
    return styleFindingsFor(book.analysis?.style, globalIndex)
  }, [book, currentSection, currentIndex, settings.style_check_enabled])

  const refresh = useCallback(() => {
    const editor = liveEditor()
    if (!editor) {
      setDetections([])
      setWordCount(0)
      return
    }
    setDetections(collectDetections(editor, backendFindings)
      .filter(detection => !skippedRef.current.has(detectionKey(detection))))
    const text = editor.state.doc.textBetween(0, editor.state.doc.content.size, ' ')
    setWordCount(text.split(/\s+/).filter(Boolean).length)
  }, [backendFindings])

  useEffect(() => {
    refresh()
    const editor = liveEditor()
    if (!editor) return
    let timer: ReturnType<typeof setTimeout> | null = null
    const onUpdate = () => {
      if (timer) clearTimeout(timer)
      timer = setTimeout(refresh, 400)
    }
    editor.on('update', onUpdate)
    return () => {
      if (timer) clearTimeout(timer)
      editor.off('update', onUpdate)
    }
  }, [refresh, currentSection, currentIndex])

  const visible = useMemo(
    () => detections.filter(detection => filter === 'all' || detection.kind === filter),
    [detections, filter])
  const selectedIndex = useMemo(
    () => (selectedKey === null ? -1 : visible.findIndex(detection => detectionKey(detection) === selectedKey)),
    [visible, selectedKey])
  const selected = selectedIndex >= 0 ? visible[selectedIndex] : undefined

  // Spotlight the open detection; release it when nothing is open.
  useEffect(() => {
    const editor = liveEditor()
    if (!editor) return
    if (!selected) {
      clearSpotlight(editor.view)
      return
    }
    updateSpotlight(editor.view, { active: true, ranges: [{ from: selected.from, to: selected.to }], activeIndex: 0 })
    window.requestAnimationFrame(() => {
      document.querySelector('.range-spotlight-current')?.scrollIntoView({ block: 'center', behavior: 'smooth' })
    })
    return () => clearSpotlight(editor.view)
  }, [selected])

  // Spelling suggestions load as a word is opened.
  useEffect(() => {
    setSuggestions([])
    if (!selected || selected.kind !== 'spelling') return
    let stale = false
    void getSuggestions(selected.text, 5).then(found => {
      if (!stale) setSuggestions(found)
    })
    return () => {
      stale = true
    }
  }, [selected])

  const selectAt = useCallback((index: number) => {
    const target = visible[index]
    setSelectedKey(target ? detectionKey(target) : null)
  }, [visible])

  const removeSelected = useCallback(() => {
    if (!selected) return
    skippedRef.current.add(detectionKey(selected))
    setSkippedCount(skippedRef.current.size)
    setDetections(list => list.filter(detection => detectionKey(detection) !== detectionKey(selected)))
    selectAt(Math.min(selectedIndex, visible.length - 2))
  }, [selected, selectedIndex, visible, selectAt])

  const handleApply = useCallback((replacement: string) => {
    const editor = liveEditor()
    if (!editor || !selected) return
    applyDetection(editor, selected, replacement)
    const nextKey = visible[selectedIndex + 1] ? detectionKey(visible[selectedIndex + 1]) : null
    setSelectedKey(nextKey)
    refresh()
  }, [selected, selectedIndex, visible, refresh])

  // ↑/↓ move, Enter applies the first suggestion, Esc ignores — while focus
  // is not in the editor or an input.
  useEffect(() => {
    if (view !== 'list') return
    const onKey = (event: KeyboardEvent) => {
      if (isEditableTarget(event.target)) return
      if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        event.preventDefault()
        const delta = event.key === 'ArrowDown' ? 1 : -1
        const next = selectedIndex < 0 ? 0 : Math.max(0, Math.min(visible.length - 1, selectedIndex + delta))
        selectAt(next)
      }
      if (event.key === 'Enter' && selected) {
        const first = (selected.kind === 'spelling' ? suggestions : selected.replacements)[0]
        if (first !== undefined) {
          event.preventDefault()
          handleApply(first)
        }
      }
      if (event.key === 'Escape' && selected) {
        event.preventDefault()
        removeSelected()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [view, visible, selectedIndex, selected, suggestions, selectAt, handleApply, removeSelected])

  if (!book) return <div className="tool-empty-state">Open a book to proofread.</div>

  const counts: Record<Filter, number> = { all: detections.length, spelling: 0, grammar: 0, style: 0 }
  for (const detection of detections) counts[detection.kind]++
  const typographyCount = typographyFixableCount(detections)
  const chapter = getSectionArray(book, currentSection)[currentIndex]
  const chapterLabel = chapter?.title || `Chapter ${currentIndex + 1}`
  const dictionary = settings.custom_dictionary
  const ignoredWords = book.analysis?.style?.ignored_words ?? []
  const decisionCount = (book.analysis?.style?.decisions ?? []).length

  // .an-panel gives the flush, full-height column (slide-panel-content drops
  // its padding when it holds one); the inline font size matches the comp.
  const panelStyle: React.CSSProperties = { fontSize: 13, overflowY: 'hidden' }

  // ── Checks & Dictionary subview ────────────────────────────────────────────
  if (view === 'settings') {
    const toggle = (label: string, sub: string, value: boolean, onChange: (next: boolean) => void) => (
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '6px 0' }}>
        <div>
          <div>{label}</div>
          <div style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{sub}</div>
        </div>
        <button
          onClick={() => onChange(!value)}
          title={value ? 'On' : 'Off'}
          style={{
            width: 34, height: 18, borderRadius: 9, border: '1px solid var(--border)', cursor: 'pointer',
            background: value ? 'var(--app-accent-text)' : 'var(--bg-surface-alt)', position: 'relative', padding: 0,
          }}
        >
          <span style={{
            position: 'absolute', top: 2, left: value ? 18 : 2, width: 12, height: 12,
            borderRadius: '50%', background: 'var(--bg-panel)', transition: 'left .12s ease',
          }} />
        </button>
      </div>
    )

    const addWord = () => {
      const root = getDictionaryRoot(newWord)
      if (!root) return
      const next = normalizeCustomDictionary([...dictionary, root])
      void saveSettings({ custom_dictionary: next })
      setNewWord('')
    }

    return (
      <div className="an-panel" style={panelStyle}>
        <div style={{ height: 38, flex: 'none', display: 'flex', alignItems: 'center', padding: '0 8px', borderBottom: '1px solid var(--border)', gap: 4 }}>
          <button style={iconButtonStyle} title="Back" onClick={() => setView('list')}>
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M15 6l-6 6 6 6" /></svg>
          </button>
          <span style={{ fontWeight: 600, flex: 1 }}>Checks &amp; Dictionary</span>
        </div>
        <div style={{ flex: 1, overflowY: 'auto', padding: 14, display: 'flex', flexDirection: 'column', gap: 18 }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <span style={{ fontSize: 11, fontWeight: 700, letterSpacing: '.8px', textTransform: 'uppercase', color: 'var(--text-section)' }}>Checks</span>
            {toggle('Spelling', `English (${settings.spell_language.slice(-2)}) · Hunspell`, settings.spell_check_enabled,
              next => void saveSettings({ spell_check_enabled: next }))}
            {toggle('Grammar', 'Contractions, agreement, typography', settings.grammar_check_enabled,
              next => void saveSettings({ grammar_check_enabled: next }))}
            {toggle('Style', 'Echoes and suggestions from story analysis', settings.style_check_enabled,
              next => void saveSettings({ style_check_enabled: next }))}
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
              <span style={{ fontSize: 11, fontWeight: 700, letterSpacing: '.8px', textTransform: 'uppercase', color: 'var(--text-section)' }}>Custom Dictionary</span>
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{dictionary.length} words</span>
            </div>
            <input
              className="dialog-input"
              placeholder="Add a word, then press Enter"
              value={newWord}
              onChange={event => setNewWord(event.target.value)}
              onKeyDown={event => {
                if (event.key === 'Enter') addWord()
              }}
            />
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
              {dictionary.map(word => (
                <span key={word} style={{ display: 'inline-flex', alignItems: 'center', gap: 4, padding: '2px 4px 2px 8px', border: '1px solid var(--border)', borderRadius: 4, fontSize: 12, background: 'var(--bg-surface-alt)' }}>
                  {word}
                  <button
                    title="Remove"
                    style={{ width: 16, height: 16, border: 0, background: 'transparent', color: 'var(--text-muted)', borderRadius: 3, display: 'grid', placeItems: 'center', cursor: 'pointer' }}
                    onClick={() => void saveSettings({ custom_dictionary: dictionary.filter(entry => entry !== word) })}
                  >
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round"><path d="M6 6l12 12M18 6L6 18" /></svg>
                  </button>
                </span>
              ))}
            </div>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <span style={{ fontSize: 11, fontWeight: 700, letterSpacing: '.8px', textTransform: 'uppercase', color: 'var(--text-section)' }}>Silenced</span>
            <div style={{ display: 'flex', flexDirection: 'column' }}>
              {ignoredWords.map(word => (
                <div key={word} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '7px 0', borderBottom: '1px solid var(--border-subtle)', fontSize: 12 }}>
                  <span>Word · “{word}” never flagged</span>
                  <button style={actionLinkStyle} onClick={() => removeStyleIgnoredWord(word)}>Restore</button>
                </div>
              ))}
              {decisionCount > 0 && (
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '7px 0', borderBottom: '1px solid var(--border-subtle)', fontSize: 12 }}>
                  <span>Ignored in book · {decisionCount} finding{decisionCount === 1 ? '' : 's'}</span>
                  <button style={actionLinkStyle} onClick={clearStyleDecisions}>Clear All</button>
                </div>
              )}
              {skippedCount > 0 && (
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '7px 0', borderBottom: '1px solid var(--border-subtle)', fontSize: 12 }}>
                  <span>Ignored this session · {skippedCount}</span>
                  <button
                    style={actionLinkStyle}
                    onClick={() => {
                      skippedRef.current.clear()
                      setSkippedCount(0)
                      refresh()
                    }}
                  >
                    Restore
                  </button>
                </div>
              )}
              {ignoredWords.length === 0 && decisionCount === 0 && skippedCount === 0 && (
                <div style={{ fontSize: 12, color: 'var(--text-muted)', padding: '7px 0' }}>Nothing silenced.</div>
              )}
            </div>
          </div>
        </div>
        <div style={{ height: 32, flex: 'none', display: 'flex', alignItems: 'center', padding: '0 14px', borderTop: '1px solid var(--border)', fontSize: 11, color: 'var(--text-muted)' }}>
          Changes apply to the whole book
        </div>
      </div>
    )
  }

  // ── List view ──────────────────────────────────────────────────────────────
  const filters: Filter[] = ['all', 'spelling', 'grammar', 'style']

  return (
    <div className="an-panel" style={panelStyle}>
      <div style={{ padding: '10px 14px 0', display: 'flex', gap: 4, alignItems: 'center' }}>
        {filters.map(kind => (
          <button key={kind} style={chipStyle(filter === kind)} onClick={() => setFilter(kind)}>
            {kind === 'all' ? 'All' : KIND[kind].label}{' '}
            <span style={{ color: 'var(--text-secondary)' }}>{counts[kind]}</span>
          </button>
        ))}
        <span style={{ flex: 1 }} />
        <button style={iconButtonStyle} title="Checks & Dictionary…" onClick={() => setView('settings')}>
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z" /></svg>
        </button>
      </div>
      <div style={{ padding: '8px 14px 10px', fontSize: 11, color: 'var(--text-muted)', borderBottom: '1px solid var(--border)', display: 'flex', justifyContent: 'space-between' }}>
        <span>{chapterLabel} · {wordCount.toLocaleString()} words</span>
        <span>{selectedIndex >= 0 ? `${selectedIndex + 1} of ${visible.length}` : 'Checks run as you type'}</span>
      </div>

      {visible.length === 0 ? (
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 10, padding: '0 28px 60px', textAlign: 'center' }}>
          <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="var(--status-success)" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="9" /><path d="M8 12.5l2.5 2.5L16 9.5" /></svg>
          <div style={{ fontWeight: 600 }}>{chapterLabel} is clean</div>
          <div style={{ fontSize: 12, color: 'var(--text-secondary)', lineHeight: 1.5 }}>
            No {filter === 'all' ? 'spelling, grammar, or style' : KIND[filter as Detection['kind']].label.toLowerCase()} issues.
            {typographyCount === 0 ? ' Typography is clean.' : ''}
          </div>
          <button className="dialog-btn sm" style={{ marginTop: 6 }} onClick={refresh}>Rescan Chapter</button>
        </div>
      ) : (
        <div style={{ flex: 1, overflowY: 'auto' }}>
          {visible.map((detection, index) => {
            const kind = KIND[detection.kind]
            const key = detectionKey(detection)
            if (index === selectedIndex) {
              const options = (detection.kind === 'spelling' ? suggestions : detection.replacements)
                .filter(replacement => replacement !== '').slice(0, 5)
              return (
                <div key={key} style={{ borderBottom: '1px solid var(--border-subtle)' }}>
                  <div style={{ background: 'var(--bg-active)', borderLeft: `2px solid ${kind.color}`, padding: '10px 14px 12px 12px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8, alignItems: 'baseline' }}>
                      <span style={{ fontWeight: 600 }}>{detection.text}</span>
                      <span style={{ fontSize: 11, color: kind.color, flex: 'none' }}>{kind.label}</span>
                    </div>
                    <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 1, lineHeight: 1.45 }}>{detection.message}</div>
                    <div
                      title="Jump to text"
                      onClick={() => {
                        const editor = liveEditor()
                        editor?.chain().focus().setTextSelection({ from: detection.from, to: detection.to }).run()
                      }}
                      style={{
                        marginTop: 10, padding: '10px 12px', background: 'var(--bg-surface-alt)',
                        border: '1px solid var(--border)', borderRadius: 4,
                        fontFamily: "'Merriweather', Georgia, serif", fontSize: 13, lineHeight: 1.7,
                        color: 'var(--text-primary)', cursor: 'pointer',
                      }}
                    >
                      …{detection.before}
                      <mark style={{ background: kind.mark, color: 'inherit', borderBottom: `2px solid ${kind.color}`, padding: '0 1px' }}>{detection.text}</mark>
                      {detection.after}…
                    </div>
                    {options.length > 0 && (
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 10 }}>
                        {options.map(replacement => (
                          <button key={replacement} className="dialog-btn sm primary" onClick={() => handleApply(replacement)}>{replacement}</button>
                        ))}
                        {detection.replacements.includes('') && (
                          <button className="dialog-btn sm primary" onClick={() => handleApply('')}>Remove</button>
                        )}
                      </div>
                    )}
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 12px', marginTop: 10 }}>
                      <button style={actionLinkStyle} onClick={removeSelected}>Ignore</button>
                      {detection.kind === 'spelling' && (
                        <button
                          style={actionLinkStyle}
                          onClick={() => {
                            const root = getDictionaryRoot(detection.text)
                            if (root && !dictionary.includes(root)) {
                              void saveSettings({ custom_dictionary: [...dictionary, root] })
                            }
                            removeSelected()
                          }}
                        >
                          Add to dictionary
                        </button>
                      )}
                      {detection.finding && (
                        <>
                          <button
                            style={actionLinkStyle}
                            onClick={() => {
                              setStyleDecision(detection.finding!.id, 'dismissed')
                              removeSelected()
                            }}
                          >
                            Dismiss
                          </button>
                          <button
                            style={actionLinkStyle}
                            onClick={() => {
                              addStyleIgnoredWord(detection.finding!.snippet)
                              removeSelected()
                            }}
                          >
                            Never flag “{detection.finding.snippet}”
                          </button>
                        </>
                      )}
                    </div>
                  </div>
                </div>
              )
            }
            return (
              <div
                key={key}
                onClick={() => setSelectedKey(key)}
                style={{
                  display: 'grid', gridTemplateColumns: '6px 1fr', gap: 10,
                  padding: '10px 14px 10px 12px', borderLeft: '2px solid transparent',
                  borderBottom: '1px solid var(--border-subtle)', cursor: 'pointer',
                }}
              >
                <span style={{ width: 6, height: 6, borderRadius: '50%', marginTop: 6, background: kind.color }} />
                <div style={{ minWidth: 0 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8, alignItems: 'baseline' }}>
                    <span style={{ fontWeight: 600 }}>{detection.text}</span>
                    <span style={{ fontSize: 11, color: 'var(--text-muted)', flex: 'none' }}>{kind.label}</span>
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 1, lineHeight: 1.45 }}>{detection.message}</div>
                  <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 3, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', fontFamily: "'Merriweather', Georgia, serif" }}>
                    …{detection.before}{detection.text}{detection.after}…
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {typographyCount > 0 ? (
        <div style={{ height: 36, flex: 'none', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10, padding: '0 8px 0 14px', borderTop: '1px solid var(--border)', fontSize: 11, color: 'var(--text-secondary)' }}>
          <span style={{ minWidth: 0, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            <span style={{ color: 'var(--text-primary)', fontWeight: 600 }}>{typographyCount}</span> typography fix{typographyCount === 1 ? '' : 'es'} <span style={{ color: 'var(--text-muted)' }}>· quotes, dashes</span>
          </span>
          <button
            className="dialog-btn sm"
            style={{ flex: 'none' }}
            onClick={() => {
              const editor = liveEditor()
              if (!editor) return
              fixChapterTypography(editor)
              refresh()
            }}
          >
            Fix All
          </button>
        </div>
      ) : (
        <div style={{ height: 32, flex: 'none', display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 14px', borderTop: '1px solid var(--border)', fontSize: 11, color: 'var(--text-muted)' }}>
          <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}><kbd style={kbdStyle}>↑</kbd><kbd style={kbdStyle}>↓</kbd> move</span>
          <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}><kbd style={kbdStyle}>Enter</kbd> apply first</span>
          <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}><kbd style={kbdStyle}>Esc</kbd> ignore</span>
        </div>
      )}
    </div>
  )
}
