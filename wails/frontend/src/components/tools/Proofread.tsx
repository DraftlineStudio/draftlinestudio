// Spelling & Grammar — the sidebar's proofing panel, built to the design
// comp in the private reference assets: a filterable list of every detection
// in the open chapter, one expanded in place with its context and
// suggestions, a one-click typography fix in the footer, and a Checks &
// Dictionary subview. All styling lives in global.css (pf-*).

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
  applyDetection, collectDetections, globalChapterIndex, isTypographyRule,
  styleFindingsFor, typographyFixableCount, type Detection,
} from '../../services/proofread'

const KIND_LABELS: Record<Detection['kind'], string> = {
  spelling: 'Spelling',
  grammar: 'Grammar',
  style: 'Style',
}

type Filter = 'all' | Detection['kind']

function liveEditor(): Editor | null {
  return (useEditorStore.getState().editorRef as unknown as Editor | null) ?? null
}

function isEditableTarget(target: EventTarget | null): boolean {
  const element = target as HTMLElement | null
  if (!element) return false
  return element.isContentEditable || element.tagName === 'INPUT' || element.tagName === 'TEXTAREA'
}

const GearIcon = (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z" /></svg>
)

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
      .filter(detection => !skippedRef.current.has(detection.key)))
    const text = editor.state.doc.textBetween(0, editor.state.doc.content.size, ' ')
    setWordCount(text.split(/\s+/).filter(Boolean).length)
  }, [backendFindings])

  // The editorRef dependency rebinds the listener when the editor instance
  // appears or is remounted, so the walk never goes deaf to edits.
  const editorInstance = useEditorStore(s => s.editorRef)
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
  }, [refresh, currentSection, currentIndex, editorInstance])

  // Typography detections never list one by one — they collapse into the
  // summary row below, so the walk holds the issues worth reading.
  const visible = useMemo(
    () => detections.filter(detection => !isTypographyRule(detection.rule)
      && (filter === 'all' || detection.kind === filter)),
    [detections, filter])
  const selectedIndex = useMemo(
    () => (selectedKey === null ? -1 : visible.findIndex(detection => detection.key === selectedKey)),
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
    setSelectedKey(target ? target.key : null)
  }, [visible])

  const removeSelected = useCallback(() => {
    if (!selected) return
    skippedRef.current.add(selected.key)
    setSkippedCount(skippedRef.current.size)
    setDetections(list => list.filter(detection => detection.key !== selected.key))
    selectAt(Math.min(selectedIndex, visible.length - 2))
  }, [selected, selectedIndex, visible, selectAt])

  const handleApply = useCallback((replacement: string) => {
    const editor = liveEditor()
    if (!editor || !selected) return
    applyDetection(editor, selected, replacement)
    const next = visible[selectedIndex + 1]
    setSelectedKey(next ? next.key : null)
    refresh()
  }, [selected, selectedIndex, visible, refresh])

  const fixTypography = useCallback(() => {
    const editor = liveEditor()
    if (!editor) return
    fixChapterTypography(editor)
    refresh()
  }, [refresh])

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

  const counts: Record<Filter, number> = { all: 0, spelling: 0, grammar: 0, style: 0 }
  for (const detection of detections) {
    if (isTypographyRule(detection.rule)) continue
    counts.all++
    counts[detection.kind]++
  }
  const typographyCount = typographyFixableCount(detections)
  const typographyParts: string[] = []
  {
    const byRule: Record<string, number> = {}
    for (const detection of detections) {
      if (isTypographyRule(detection.rule)) byRule[detection.rule] = (byRule[detection.rule] ?? 0) + 1
    }
    const quotes = byRule['straight-quote'] ?? 0
    const ellipses = byRule['ellipsis'] ?? 0
    const hyphens = byRule['double-hyphen'] ?? 0
    if (quotes) typographyParts.push(`${quotes} straight quote${quotes === 1 ? '' : 's'}`)
    if (ellipses) typographyParts.push(`${ellipses} typed ellips${ellipses === 1 ? 'is' : 'es'}`)
    if (hyphens) typographyParts.push(`${hyphens} double hyphen${hyphens === 1 ? '' : 's'}`)
  }
  const showTypographyRow = typographyCount > 0 && (filter === 'all' || filter === 'style')
  const chapter = getSectionArray(book, currentSection)[currentIndex]
  const chapterLabel = chapter?.title || `Chapter ${currentIndex + 1}`
  const dictionary = settings.custom_dictionary
  const ignoredWords = book.analysis?.style?.ignored_words ?? []
  const decisionCount = (book.analysis?.style?.decisions ?? []).length

  // ── Checks & Dictionary subview ────────────────────────────────────────────
  if (view === 'settings') {
    const toggle = (label: string, sub: string, value: boolean, onChange: (next: boolean) => void) => (
      <div className="pf-toggle-row">
        <div>
          <div>{label}</div>
          <div className="pf-toggle-sub">{sub}</div>
        </div>
        <button className={`pf-toggle${value ? ' on' : ''}`} title={value ? 'On' : 'Off'} onClick={() => onChange(!value)}>
          <span className="pf-toggle-knob" />
        </button>
      </div>
    )

    const addWord = () => {
      const root = getDictionaryRoot(newWord)
      if (!root) return
      void saveSettings({ custom_dictionary: normalizeCustomDictionary([...dictionary, root]) })
      setNewWord('')
    }

    return (
      <div className="an-panel pf-panel">
        <div className="pf-subhead">
          <button className="pf-iconbtn" title="Back" onClick={() => setView('list')}>
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M15 6l-6 6 6 6" /></svg>
          </button>
          <span className="pf-subhead-title">Checks &amp; Dictionary</span>
        </div>
        <div className="pf-settings">
          <div className="pf-settings-group">
            <span className="pf-section-label">Checks</span>
            {toggle('Spelling', `English (${settings.spell_language.slice(-2)}) · Hunspell`, settings.spell_check_enabled,
              next => void saveSettings({ spell_check_enabled: next }))}
            {toggle('Grammar', 'Contractions, agreement, typography', settings.grammar_check_enabled,
              next => void saveSettings({ grammar_check_enabled: next }))}
            {toggle('Style', 'Echoes and suggestions from story analysis', settings.style_check_enabled,
              next => void saveSettings({ style_check_enabled: next }))}
          </div>

          <div className="pf-settings-group">
            <div className="pf-section-head">
              <span className="pf-section-label">Custom Dictionary</span>
              <span className="pf-section-count">{dictionary.length} words</span>
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
            <div className="pf-dict-chips">
              {dictionary.map(word => (
                <span key={word} className="pf-dict-chip">
                  {word}
                  <button
                    className="pf-dict-remove"
                    title="Remove"
                    onClick={() => void saveSettings({ custom_dictionary: dictionary.filter(entry => entry !== word) })}
                  >
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round"><path d="M6 6l12 12M18 6L6 18" /></svg>
                  </button>
                </span>
              ))}
            </div>
          </div>

          <div className="pf-settings-group">
            <span className="pf-section-label">Silenced</span>
            <div>
              {ignoredWords.map(word => (
                <div key={word} className="pf-silenced-row">
                  <span>Word · “{word}” never flagged</span>
                  <button className="pf-action" onClick={() => removeStyleIgnoredWord(word)}>Restore</button>
                </div>
              ))}
              {decisionCount > 0 && (
                <div className="pf-silenced-row">
                  <span>Ignored in book · {decisionCount} finding{decisionCount === 1 ? '' : 's'}</span>
                  <button className="pf-action" onClick={clearStyleDecisions}>Clear All</button>
                </div>
              )}
              {skippedCount > 0 && (
                <div className="pf-silenced-row">
                  <span>Ignored this session · {skippedCount}</span>
                  <button
                    className="pf-action"
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
                <div className="pf-silenced-empty">Nothing silenced.</div>
              )}
            </div>
          </div>
        </div>
        <div className="pf-settings-footer">Changes apply to the whole book</div>
      </div>
    )
  }

  // ── List view ──────────────────────────────────────────────────────────────
  const filters: Filter[] = ['all', 'spelling', 'grammar', 'style']

  return (
    <div className="an-panel pf-panel">
      <div className="pf-chips">
        {filters.map(kind => (
          <button key={kind} className={`pf-chip${filter === kind ? ' active' : ''}`} onClick={() => setFilter(kind)}>
            {kind === 'all' ? 'All' : KIND_LABELS[kind]}{' '}
            <span className="pf-chip-count">{counts[kind]}</span>
          </button>
        ))}
        <span className="pf-spacer" />
        <button className="pf-iconbtn" title="Checks & Dictionary…" onClick={() => setView('settings')}>
          {GearIcon}
        </button>
      </div>
      <div className="pf-meta">
        <span>{chapterLabel} · {wordCount.toLocaleString()} words</span>
        <span>{selectedIndex >= 0 ? `${selectedIndex + 1} of ${visible.length}` : 'Checks run as you type'}</span>
      </div>

      {visible.length === 0 && !showTypographyRow ? (
        <div className="pf-empty">
          <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="var(--status-success)" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="9" /><path d="M8 12.5l2.5 2.5L16 9.5" /></svg>
          <div className="pf-empty-title">{chapterLabel} is clean</div>
          <div className="pf-empty-sub">
            No {filter === 'all' ? 'spelling, grammar, or style' : KIND_LABELS[filter as Detection['kind']].toLowerCase()} issues.
            {typographyCount === 0 ? ' Typography is clean.' : ''}
          </div>
          <button className="dialog-btn sm" onClick={refresh}>Rescan Chapter</button>
        </div>
      ) : (
        <div className="pf-list">
          {showTypographyRow && (
            <div className="pf-row pf-typo-row pf-kind-style">
              <span className="pf-dot" />
              <div className="pf-row-body">
                <div className="pf-row-head">
                  <span className="pf-row-text">Inconsistent punctuation</span>
                  <span className="pf-row-kind">Typography</span>
                </div>
                <div className="pf-row-msg">
                  {typographyParts.join(', ')} across the chapter. One click typesets them all: curly quotes and apostrophes, real ellipses, em dashes.
                </div>
                <div className="pf-suggestions">
                  <button className="dialog-btn sm primary" onClick={fixTypography}>
                    Fix all {typographyCount}
                  </button>
                </div>
              </div>
            </div>
          )}
          {visible.map((detection, index) => {
            const key = detection.key
            const kindClass = `pf-kind-${detection.kind}`
            if (index === selectedIndex) {
              const options = (detection.kind === 'spelling' ? suggestions : detection.replacements)
                .filter(replacement => replacement !== '').slice(0, 5)
              return (
                <div key={key} className="pf-open-wrap">
                  <div className={`pf-open ${kindClass}`}>
                    <div className="pf-row-head">
                      <span className="pf-row-text">{detection.text}</span>
                      <span className="pf-row-kind">{KIND_LABELS[detection.kind]}</span>
                    </div>
                    <div className="pf-row-msg">{detection.message}</div>
                    <div
                      className="pf-context"
                      title="Jump to text"
                      onClick={() => {
                        liveEditor()?.chain().focus().setTextSelection({ from: detection.from, to: detection.to }).run()
                      }}
                    >
                      …{detection.before}
                      <mark className="pf-mark">{detection.text}</mark>
                      {detection.after}…
                    </div>
                    {options.length > 0 && (
                      <div className="pf-suggestions">
                        {options.map(replacement => (
                          <button key={replacement} className="dialog-btn sm primary" onClick={() => handleApply(replacement)}>{replacement}</button>
                        ))}
                        {detection.replacements.includes('') && (
                          <button className="dialog-btn sm primary" onClick={() => handleApply('')}>Remove</button>
                        )}
                      </div>
                    )}
                    <div className="pf-actions">
                      <button className="pf-action" onClick={removeSelected}>Ignore</button>
                      {detection.kind === 'spelling' && (
                        <button
                          className="pf-action"
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
                            className="pf-action"
                            onClick={() => {
                              setStyleDecision(detection.finding!.id, 'dismissed')
                              removeSelected()
                            }}
                          >
                            Dismiss
                          </button>
                          <button
                            className="pf-action"
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
              <div key={key} className={`pf-row ${kindClass}`} onClick={() => setSelectedKey(key)}>
                <span className="pf-dot" />
                <div className="pf-row-body">
                  <div className="pf-row-head">
                    <span className="pf-row-text">{detection.text}</span>
                    <span className="pf-row-kind">{KIND_LABELS[detection.kind]}</span>
                  </div>
                  <div className="pf-row-msg">{detection.message}</div>
                  <div className="pf-row-snippet">…{detection.before}{detection.text}{detection.after}…</div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      <div className="pf-footer-hints">
        <span className="pf-hint"><kbd className="pf-kbd">↑</kbd><kbd className="pf-kbd">↓</kbd> move</span>
        <span className="pf-hint"><kbd className="pf-kbd">Enter</kbd> apply first</span>
        <span className="pf-hint"><kbd className="pf-kbd">Esc</kbd> ignore</span>
      </div>
    </div>
  )
}
