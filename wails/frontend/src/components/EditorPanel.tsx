import { useCallback, useMemo, useRef, useEffect } from 'react'
import { useBookStore } from '../store/bookStore'
import RichEditor from './editor/RichEditor'

function getCurrentContent(book: ReturnType<typeof useBookStore.getState>['book'], section: string, index: number): string {
  if (!book) return ''
  if (section === 'copyright') return book.copyright || ''
  if (section === 'front_matter') return book.front_matter[index]?.content || ''
  if (section === 'body') return book.body[index]?.content || ''
  if (section === 'back_matter') return book.back_matter[index]?.content || ''
  return ''
}

function getChapterInfo(book: ReturnType<typeof useBookStore.getState>['book'], section: string, index: number): { label: string; name: string } {
  if (!book) return { label: '', name: '' }
  if (section === 'copyright') return { label: 'Front Pages', name: 'Copyright Page' }
  if (section === 'front_matter') {
    const item = book.front_matter[index]
    return { label: 'Front Matter', name: item?.title || 'Untitled' }
  }
  if (section === 'body') {
    const item = book.body[index]
    return { label: 'Body', name: item?.title || 'Untitled' }
  }
  if (section === 'back_matter') {
    const item = book.back_matter[index]
    return { label: 'Back Matter', name: item?.title || 'Untitled' }
  }
  return { label: '', name: '' }
}

function DiffPanel({ label, name }: { label: string; name: string }) {
  const {
    pendingDiff,
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
  const focused = changes[focusedChangeIdx]

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
          className={`diff-ctrl-btn${focused && !focused.accepted ? ' active-reject' : ''}`}
          onClick={() => rejectChange(focusedChangeIdx)}
          title="Keep original (k)"
        >Keep</button>
        <button
          className={`diff-ctrl-btn${focused && focused.accepted ? ' active-accept' : ''}`}
          onClick={() => acceptChange(focusedChangeIdx)}
          title="Accept suggestion (a)"
        >Accept</button>

        <div className="diff-accept-sep" />

        <button className="diff-ctrl-btn secondary" onClick={rejectAllDiff}>Reject all</button>
        <button className="diff-ctrl-btn secondary" onClick={acceptAllDiff}>Accept all</button>
        <button className="diff-ctrl-btn primary" onClick={applyPendingDiff}>
          Apply {acceptedCount}/{total}
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

                  if (chunk.type === 'delete') {
                    return isAccepted
                      // AI version: show deletion struck through
                      ? <del key={chunkIdx} data-change={changeIdx} className={`diff-inline-del${focusCls}`} onClick={handleClick}>{chunk.text}</del>
                      // Original kept: show as plain text
                      : <span key={chunkIdx} data-change={changeIdx} className={`diff-kept${focusCls}`} onClick={handleClick}>{chunk.text}</span>
                  }

                  if (chunk.type === 'insert') {
                    return isAccepted
                      // AI version: show insertion highlighted
                      ? <ins key={chunkIdx} data-change={changeIdx} className={`diff-inline-ins${focusCls}`} onClick={handleClick}>{chunk.text}</ins>
                      // Rejected: insert not shown
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
  const { book, currentSection, currentIndex, updateCurrentContent, pendingDiff } = useBookStore()

  const content = getCurrentContent(book, currentSection, currentIndex)
  const { label, name } = getChapterInfo(book, currentSection, currentIndex)

  const handleUpdate = useCallback(
    (html: string) => { updateCurrentContent(html) },
    [updateCurrentContent, currentSection, currentIndex],
  )

  if (!book) {
    return (
      <div className="editor-panel">
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
      <div className="editor-panel">
        <DiffPanel label={label} name={name} />
      </div>
    )
  }

  const editorKey = `${currentSection}-${currentIndex}`

  return (
    <div className="editor-panel">
      <RichEditor
        key={editorKey}
        content={content}
        onUpdate={handleUpdate}
        chapterLabel={label}
        chapterName={name}
      />
    </div>
  )
}
