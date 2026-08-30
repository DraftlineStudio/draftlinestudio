import { useCallback, useRef, useEffect, useMemo } from 'react'
import { useBookStore } from '../store/bookStore'
import { useEditorStore } from '../store/editorStore'
import { useAppStore } from '../store/appStore'
import { getCurrentContent } from '../utils/textUtils'
import RichEditor from './editor/RichEditor'

const EDITOR_FONT_SIZES = { small: '12px', normal: '14px', large: '16px' }
const CONTENT_UPDATE_DEBOUNCE = 150 // ms - debounce store updates for smoother typing

function getChapterInfo(book: ReturnType<typeof useBookStore.getState>['book'], section: string, index: number): { label: string; name: string; subtitle: string } {
  if (!book) return { label: '', name: '', subtitle: '' }
  if (section === 'copyright') return { label: 'Front Pages', name: 'Copyright Page', subtitle: '' }
  if (section === 'front_matter') {
    const item = book.front_matter[index]
    return { label: 'Front Matter', name: item?.title || 'Untitled', subtitle: item?.subtitle || '' }
  }
  if (section === 'body') {
    const item = book.body[index]
    return { label: 'Body', name: item?.title || 'Untitled', subtitle: item?.subtitle || '' }
  }
  if (section === 'back_matter') {
    const item = book.back_matter[index]
    return { label: 'Back Matter', name: item?.title || 'Untitled', subtitle: item?.subtitle || '' }
  }
  return { label: '', name: '', subtitle: '' }
}

function DiffPanel({ label, name }: { label: string; name: string }) {
  // pendingDiff lives in editorStore — subscribe there directly; the bookStore
  // bridge getter does not notify bookStore subscribers when editorStore changes
  const pendingDiff = useEditorStore(s => s.pendingDiff)
  const {
    acceptChange,
    rejectChange,
    setFocusedChange,
    prevChange,
    nextChange,
    acceptAllDiff,
    rejectAllDiff,
    applyPendingDiff,
    clearPendingDiff,
  } = useBookStore()

  const scrollRef = useRef<HTMLDivElement>(null)

  // Scroll focused change into view when it changes
  useEffect(() => {
    if (!pendingDiff || !scrollRef.current) return
    const el = scrollRef.current.querySelector<HTMLElement>(`[data-change="${pendingDiff.focusedChangeIdx}"]`)
    el?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  }, [pendingDiff?.focusedChangeIdx])

  // Keyboard nav: ← → to move between changes, a/k to accept/keep
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (!pendingDiff) return
      if (e.key === 'ArrowRight' || e.key === 'ArrowDown') { e.preventDefault(); nextChange() }
      if (e.key === 'ArrowLeft'  || e.key === 'ArrowUp')   { e.preventDefault(); prevChange() }
      if (e.key === 'a') { e.preventDefault(); acceptChange(pendingDiff.focusedChangeIdx) }
      if (e.key === 'k') { e.preventDefault(); rejectChange(pendingDiff.focusedChangeIdx) }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [pendingDiff, acceptChange, rejectChange, prevChange, nextChange])

  if (!pendingDiff) return null

  const { diffs, changes, focusedChangeIdx } = pendingDiff
  const total = changes.length
  const acceptedCount = changes.filter(c => c.accepted).length
  const decidedCount = changes.filter(c => c.decided).length
  const undecidedCount = total - decidedCount
  const focused = changes[focusedChangeIdx]

  // Determine button labels based on whether some changes are decided
  const hasDecided = decidedCount > 0
  const rejectLabel = hasDecided ? `Reject remaining (${undecidedCount})` : 'Reject all'
  const acceptLabel = hasDecided ? `Accept remaining (${undecidedCount})` : 'Accept all'

  // Map "paraIdx-chunkIdx" → changeIdx for O(1) lookup during render
  const chunkChangeMap = useMemo(() => {
    const map = new Map<string, number>()
    changes.forEach((ch, idx) => {
      for (let ci = ch.startIdx; ci < ch.endIdx; ci++) {
        map.set(`${ch.paraIdx}-${ci}`, idx)
      }
    })
    return map
  }, [changes])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Control bar */}
      <div className="diff-accept-bar">
        {/* Per-change navigation */}
        <div className="diff-nav-group">
          <button className="diff-ctrl-btn nav" onClick={prevChange} disabled={focusedChangeIdx <= 0} title="Previous change (←)">‹</button>
          <span className="diff-nav-counter">{total > 0 ? `${focusedChangeIdx + 1} / ${total}` : '0'}</span>
          <button className="diff-ctrl-btn nav" onClick={nextChange} disabled={focusedChangeIdx >= total - 1} title="Next change (→)">›</button>
        </div>

        {/* Accept / Keep for focused change */}
        <button
          className={`diff-ctrl-btn${focused?.decided && !focused.accepted ? ' active-reject' : ''}`}
          onClick={() => rejectChange(focusedChangeIdx)}
          title="Keep original (k)"
        >Keep</button>
        <button
          className={`diff-ctrl-btn${focused?.decided && focused.accepted ? ' active-accept' : ''}`}
          onClick={() => acceptChange(focusedChangeIdx)}
          title="Accept suggestion (a)"
        >Accept</button>

        <div className="diff-accept-sep" />

        <button className="diff-ctrl-btn secondary" onClick={rejectAllDiff} disabled={undecidedCount === 0}>
          {rejectLabel}
        </button>
        <button className="diff-ctrl-btn secondary" onClick={acceptAllDiff} disabled={undecidedCount === 0}>
          {acceptLabel}
        </button>
        <button className="diff-ctrl-btn primary" onClick={applyPendingDiff}>
          Apply ({acceptedCount} accepted)
        </button>
        <button className="diff-ctrl-btn discard" onClick={clearPendingDiff} title="Discard all changes">✕</button>
      </div>

      {/* Chapter content with inline diff */}
      <div className="editor-scroll" ref={scrollRef}>
        <div className="editor-content-wrapper">
          {(label || name) && (
            <div className="editor-page-header">
              {label && <span className="editor-page-chapter-title">{label}</span>}
              {name && <span className="editor-page-chapter-name">{name}</span>}
            </div>
          )}
          <div className="editor-page-body">
            {diffs.map((diff, paraIdx) => (
              <p key={paraIdx} className="diff-inline-p">
                {diff.chunks.map((chunk, chunkIdx) => {
                  if (chunk.type === 'equal') {
                    return <span key={chunkIdx}>{chunk.text}</span>
                  }

                  const changeIdx = chunkChangeMap.get(`${paraIdx}-${chunkIdx}`)
                  if (changeIdx === undefined) return <span key={chunkIdx}>{chunk.text}</span>

                  const change = changes[changeIdx]
                  const isFocused = changeIdx === focusedChangeIdx
                  const isAccepted = change.accepted

                  const focusCls = isFocused ? ' focused' : ''
                  const handleClick = () => setFocusedChange(changeIdx)

                  const isDecided = change.decided

                  if (chunk.type === 'delete') {
                    // Deletion: show strikethrough (red) if accepted or undecided, plain text if explicitly rejected
                    return (isAccepted || !isDecided)
                      ? <del key={chunkIdx} data-change={changeIdx} className={`diff-inline-del${focusCls}`} onClick={handleClick}>{chunk.text}</del>
                      : <span key={chunkIdx} data-change={changeIdx} className={`diff-kept${focusCls}`} onClick={handleClick}>{chunk.text}</span>
                  }

                  if (chunk.type === 'insert') {
                    // Insertion: show green if accepted or undecided, hide if explicitly rejected
                    return (isAccepted || !isDecided)
                      ? <ins key={chunkIdx} data-change={changeIdx} className={`diff-inline-ins${focusCls}`} onClick={handleClick}>{chunk.text}</ins>
                      : null
                  }

                  return null
                })}
              </p>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

export default function EditorPanel() {
  const { book, currentSection, currentIndex, updateCurrentContent, updateChapterTitle, updateChapterSubtitle } = useBookStore()
  const pendingDiff = useEditorStore(s => s.pendingDiff)
  const { settings } = useAppStore()

  const content = getCurrentContent(book, currentSection, currentIndex)
  const { label, name, subtitle } = getChapterInfo(book, currentSection, currentIndex)

  // Callbacks for editing chapter title/subtitle (not available for copyright section)
  const handleRenameChapter = useCallback(
    (title: string) => {
      if (currentSection !== 'copyright') {
        updateChapterTitle(currentSection as 'front_matter' | 'body' | 'back_matter', currentIndex, title)
      }
    },
    [currentSection, currentIndex, updateChapterTitle],
  )

  const handleEditSubtitle = useCallback(
    (newSubtitle: string) => {
      if (currentSection !== 'copyright') {
        updateChapterSubtitle(currentSection as 'front_matter' | 'body' | 'back_matter', currentIndex, newSubtitle)
      }
    },
    [currentSection, currentIndex, updateChapterSubtitle],
  )

  // Debounced content update for smoother typing
  // CRITICAL: We must capture section/index at typing time, not cleanup time
  const updateTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const pendingUpdate = useRef<{ section: string; index: number; content: string } | null>(null)

  const handleUpdate = useCallback(
    (html: string) => {
      // Capture the current section/index NOW, not when the timer fires
      pendingUpdate.current = { section: currentSection, index: currentIndex, content: html }
      if (updateTimer.current) clearTimeout(updateTimer.current)
      updateTimer.current = setTimeout(() => {
        if (pendingUpdate.current) {
          const { section, index, content } = pendingUpdate.current
          // Only update if we're still on the same chapter
          const state = useBookStore.getState()
          if (state.currentSection === section && state.currentIndex === index) {
            updateCurrentContent(content)
          }
          pendingUpdate.current = null
        }
      }, CONTENT_UPDATE_DEBOUNCE)
    },
    [currentSection, currentIndex, updateCurrentContent],
  )

  // Clear pending updates when switching chapters - DO NOT flush to wrong chapter
  useEffect(() => {
    // Reset pending update when chapter changes - the old content belongs to old chapter
    pendingUpdate.current = null
    if (updateTimer.current) {
      clearTimeout(updateTimer.current)
      updateTimer.current = null
    }
  }, [currentSection, currentIndex])

  // CSS custom properties for editor styling
  const editorStyle = {
    '--editor-font': settings.book_font,
    '--editor-font-size': EDITOR_FONT_SIZES[settings.editor_font_size] || '14px',
  } as React.CSSProperties

  if (!book) {
    return (
      <div className="editor-panel" style={editorStyle}>
        <div className="editor-empty-state">
          <div className="editor-empty-logo" />
          <div className="editor-empty-tagline">your manuscript, beautifully composed</div>
          <div className="editor-empty-shortcuts">
            <div className="editor-empty-shortcut">
              <kbd>Ctrl</kbd><kbd>N</kbd>
              <span>New book</span>
            </div>
            <div className="editor-empty-shortcut">
              <kbd>Ctrl</kbd><kbd>O</kbd>
              <span>Open book</span>
            </div>
          </div>
        </div>
      </div>
    )
  }

  if (pendingDiff) {
    return (
      <div className="editor-panel" style={editorStyle}>
        <DiffPanel label={label} name={name} />
      </div>
    )
  }

  const editorKey = `${currentSection}-${currentIndex}`

  // Only allow editing for non-copyright sections
  const canEdit = currentSection !== 'copyright'

  return (
    <div className="editor-panel" style={editorStyle}>
      <RichEditor
        key={editorKey}
        content={content}
        onUpdate={handleUpdate}
        chapterLabel={label}
        chapterName={name}
        chapterSubtitle={subtitle}
        onRenameChapter={canEdit ? handleRenameChapter : undefined}
        onEditSubtitle={canEdit ? handleEditSubtitle : undefined}
      />
    </div>
  )
}
