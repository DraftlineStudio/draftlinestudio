// Proofread — the guided spelling/grammar/style walk for the open chapter.
// One detection at a time, Word-style: see it in context, apply a suggestion
// or skip, and the editor spotlights each one as it comes up.

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { Editor } from '@tiptap/core'
import { useShallow } from 'zustand/react/shallow'
import { useAppStore } from '../../store/appStore'
import { useBookStore } from '../../store/bookStore'
import { useEditorStore } from '../../store/editorStore'
import { clearSpotlight, updateSpotlight } from '../../extensions/RangeSpotlight'
import { fixChapterTypography } from '../../services/typographyFix'
import { getDictionaryRoot, getSuggestions } from '../../services/spellCheck'
import {
  applyDetection, collectDetections, detectionKey, globalChapterIndex,
  styleFindingsFor, typographyFixableCount, type Detection,
} from '../../services/proofread'

const KIND_LABELS: Record<Detection['kind'], string> = {
  spelling: 'Spelling',
  grammar: 'Grammar',
  style: 'Style',
}

function liveEditor(): Editor | null {
  return (useEditorStore.getState().editorRef as unknown as Editor | null) ?? null
}

export default function Proofread() {
  const { settings, saveSettings } = useAppStore(useShallow(s => ({ settings: s.settings, saveSettings: s.saveSettings })))
  const { book, currentSection, currentIndex, setStyleDecision, addStyleIgnoredWord } = useBookStore(useShallow(s => ({
    book: s.book,
    currentSection: s.currentSection,
    currentIndex: s.currentIndex,
    setStyleDecision: s.setStyleDecision,
    addStyleIgnoredWord: s.addStyleIgnoredWord,
  })))

  const [detections, setDetections] = useState<Detection[]>([])
  const [cursor, setCursor] = useState(0)
  const [walking, setWalking] = useState(false)
  const [suggestions, setSuggestions] = useState<string[]>([])
  const skippedRef = useRef(new Set<string>())

  const backendFindings = useMemo(() => {
    if (!settings.style_check_enabled || !book) return []
    const globalIndex = globalChapterIndex(
      currentSection, currentIndex,
      book.front_matter?.length ?? 0, (book.body?.length ?? 0))
    return styleFindingsFor(book.analysis?.style, globalIndex)
  }, [book, currentSection, currentIndex, settings.style_check_enabled])

  const refresh = useCallback((resumeFrom?: number) => {
    const editor = liveEditor()
    if (!editor) {
      setDetections([])
      return
    }
    const next = collectDetections(editor, backendFindings)
      .filter(detection => !skippedRef.current.has(detectionKey(detection)))
    setDetections(next)
    if (resumeFrom !== undefined) {
      const resumed = next.findIndex(detection => detection.from >= resumeFrom)
      setCursor(resumed >= 0 ? resumed : Math.max(0, next.length - 1))
    } else {
      setCursor(current => Math.min(current, Math.max(0, next.length - 1)))
    }
  }, [backendFindings])

  // Recount when the panel opens, the chapter or analysis changes, and after
  // every pause in typing.
  useEffect(() => {
    refresh()
    const editor = liveEditor()
    if (!editor) return
    let timer: ReturnType<typeof setTimeout> | null = null
    const onUpdate = () => {
      if (timer) clearTimeout(timer)
      timer = setTimeout(() => refresh(), 400)
    }
    editor.on('update', onUpdate)
    return () => {
      if (timer) clearTimeout(timer)
      editor.off('update', onUpdate)
    }
  }, [refresh, currentSection, currentIndex])

  const current = walking ? detections[cursor] : undefined

  // Spotlight the current detection while walking; release on the way out.
  useEffect(() => {
    const editor = liveEditor()
    if (!editor) return
    if (!current) {
      clearSpotlight(editor.view)
      return
    }
    updateSpotlight(editor.view, { active: true, ranges: [{ from: current.from, to: current.to }], activeIndex: 0 })
    window.requestAnimationFrame(() => {
      document.querySelector('.range-spotlight-current')?.scrollIntoView({ block: 'center', behavior: 'smooth' })
    })
    return () => clearSpotlight(editor.view)
  }, [current])

  // Spelling suggestions arrive async; the walk shows them as they land.
  useEffect(() => {
    setSuggestions([])
    if (!current || current.kind !== 'spelling') return
    let stale = false
    void getSuggestions(current.text, 5).then(found => {
      if (!stale) setSuggestions(found)
    })
    return () => {
      stale = true
    }
  }, [current])

  if (!book) return <div className="tool-empty-state">Open a book to proofread.</div>

  const counts = { spelling: 0, grammar: 0, style: 0 }
  for (const detection of detections) counts[detection.kind]++
  const typographyCount = typographyFixableCount(detections)

  const advance = () => setCursor(index => index + 1)

  const handleApply = (replacement: string) => {
    const editor = liveEditor()
    if (!editor || !current) return
    const resume = applyDetection(editor, current, replacement)
    refresh(resume)
  }

  const handleSkip = () => {
    if (!current) return
    skippedRef.current.add(detectionKey(current))
    setDetections(list => list.filter((_, index) => index !== cursor))
  }

  const handleAddToDictionary = () => {
    if (!current) return
    const root = getDictionaryRoot(current.text)
    if (root && !settings.custom_dictionary.includes(root)) {
      void saveSettings({ custom_dictionary: [...settings.custom_dictionary, root] })
    }
    // The dictionary change re-renders every underline; drop the entry now.
    setDetections(list => list.filter(detection =>
      !(detection.kind === 'spelling' && getDictionaryRoot(detection.text) === root)))
  }

  const handleDismissFinding = () => {
    if (!current?.finding) return
    setStyleDecision(current.finding.id, 'dismissed')
    setDetections(list => list.filter((_, index) => index !== cursor))
  }

  const handleNeverFlagWord = () => {
    if (!current?.finding) return
    addStyleIgnoredWord(current.finding.snippet)
    const word = current.finding.snippet.toLowerCase()
    setDetections(list => list.filter(detection =>
      !(detection.finding && detection.finding.snippet.toLowerCase() === word)))
  }

  const done = walking && cursor >= detections.length

  return (
    <div className="an-panel">
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 10 }}>
        <span className="an-footnote">Spelling {counts.spelling}</span>
        <span className="an-footnote">Grammar {counts.grammar}</span>
        <span className="an-footnote">Style {counts.style}</span>
      </div>

      <button
        className="an-run-btn"
        disabled={typographyCount === 0}
        onClick={() => {
          const editor = liveEditor()
          if (!editor) return
          fixChapterTypography(editor)
          refresh(0)
        }}
      >
        {typographyCount > 0 ? `Fix all typography (${typographyCount})` : 'Typography is clean'}
      </button>

      {!walking && (
        <button
          className="an-run-btn"
          style={{ marginTop: 8 }}
          disabled={detections.length === 0}
          onClick={() => {
            setCursor(0)
            setWalking(true)
          }}
        >
          {detections.length > 0 ? `Review ${detections.length} detections` : 'Nothing to review'}
        </button>
      )}

      {walking && !done && current && (
        <div className="an-card" style={{ marginTop: 12 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
            <strong>{KIND_LABELS[current.kind]}</strong>
            <span className="an-footnote">{cursor + 1} of {detections.length}</span>
          </div>
          <div className="an-footnote" style={{ marginBottom: 8 }}>{current.message}</div>
          <div style={{ marginBottom: 10, lineHeight: 1.5 }}>
            …{current.before}
            <mark>{current.text}</mark>
            {current.after}…
          </div>

          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
            {(current.kind === 'spelling' ? suggestions : current.replacements)
              .filter(replacement => replacement !== '')
              .slice(0, 5)
              .map(replacement => (
                <button key={replacement} className="dialog-btn sm primary" onClick={() => handleApply(replacement)}>
                  {replacement}
                </button>
              ))}
            {current.replacements.includes('') && (
              <button className="dialog-btn sm primary" onClick={() => handleApply('')}>Remove</button>
            )}
          </div>

          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 10 }}>
            <button className="dialog-btn sm" disabled={cursor === 0} onClick={() => setCursor(index => index - 1)}>Back</button>
            <button className="dialog-btn sm" disabled={cursor + 1 >= detections.length} onClick={advance}>Next</button>
            <button className="dialog-btn sm" onClick={handleSkip}>Ignore</button>
            {current.kind === 'spelling' && (
              <button className="dialog-btn sm" onClick={handleAddToDictionary}>Add to dictionary</button>
            )}
            {current.finding && (
              <>
                <button className="dialog-btn sm" onClick={handleDismissFinding}>Dismiss</button>
                <button className="dialog-btn sm" onClick={handleNeverFlagWord}>Never flag “{current.finding.snippet}”</button>
              </>
            )}
          </div>
        </div>
      )}

      {walking && (done || detections.length === 0) && (
        <div className="an-empty" style={{ marginTop: 12 }}>
          The chapter is clean — nothing left to review.
          <div style={{ marginTop: 8 }}>
            <button className="dialog-btn sm" onClick={() => setWalking(false)}>Close review</button>
          </div>
        </div>
      )}
    </div>
  )
}
