import { useCallback, useEffect, useMemo, useState } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { GetChapterHistory, ListChapterHistory } from '../../../wailsjs/go/main/App'
import type { ChapterHistoryEntry, ChapterHistorySnapshot, ChapterItem } from '../../types/draftline'
import { diffContent, type ParagraphDiff } from '../../utils/diff'
import { useBookStore } from '../../store/bookStore'
import { useAppStore } from '../../store/appStore'

function currentChapter(section: string, index: number, book: ReturnType<typeof useBookStore.getState>['book']): ChapterItem | null {
  if (!book || section === 'copyright') return null
  const items = section === 'front_matter' ? book.front_matter : section === 'back_matter' ? book.back_matter : book.body
  return items[index] ?? null
}

// One compare column. Scene breaks have no text on either side, so they are
// drawn as the ⁂ glyph instead of an empty line.
function ProseColumn({ diffs, side }: { diffs: ParagraphDiff[]; side: 'old' | 'current' }) {
  const hide = side === 'old' ? 'insert' : 'delete'
  const mark = side === 'old' ? 'delete' : 'insert'
  const markClass = side === 'old' ? 'history-deleted' : 'history-inserted'
  return (
    <div className={`chapter-history-prose ${side}`}>
      {diffs.map((row, i) => {
        if (row.tag === 'hr' && (side === 'old' ? row.originalHtml : row.revisedHtml)) {
          return <p key={i} className="history-break">⁂</p>
        }
        return (
          <p key={i}>
            {row.chunks.map((chunk, j) => chunk.type === hide
              ? null
              : <span key={j} className={chunk.type === mark ? markClass : ''}>{chunk.text}</span>)}
          </p>
        )
      })}
    </div>
  )
}

export default function ChapterHistoryDialog() {
  const { book, currentSection, currentIndex, restoreChapterHistory, snapshotCurrentChapter } = useBookStore(useShallow(s => ({
    book: s.book,
    currentSection: s.currentSection,
    currentIndex: s.currentIndex,
    restoreChapterHistory: s.restoreChapterHistory,
    snapshotCurrentChapter: s.snapshotCurrentChapter,
  })))
  const closeChapterHistory = useAppStore(s => s.closeChapterHistory)
  const chapter = currentChapter(currentSection, currentIndex, book)
  const [entries, setEntries] = useState<ChapterHistoryEntry[]>([])
  const [selected, setSelected] = useState<ChapterHistorySnapshot | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [snapshotLabel, setSnapshotLabel] = useState('')
  const [snapshotting, setSnapshotting] = useState(false)

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') closeChapterHistory()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [closeChapterHistory])

  const chapterID = chapter?.id
  const filePath = book?.file_path
  // Loads the version list; selectNewest picks the latest entry for compare.
  const loadEntries = useCallback(async (selectNewest: boolean) => {
    if (!chapterID || !filePath) {
      setLoading(false)
      return
    }
    try {
      const list = (await ListChapterHistory(chapterID) as unknown as ChapterHistoryEntry[]) ?? []
      setEntries(list)
      if (selectNewest && list.length) {
        setSelected(await GetChapterHistory(list[0].id) as unknown as ChapterHistorySnapshot)
      }
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }, [chapterID, filePath])

  // The dialog is modal, so the chapter cannot change underneath a load.
  useEffect(() => {
    setLoading(true)
    void loadEntries(true)
  }, [loadEntries])

  async function selectEntry(entry: ChapterHistoryEntry) {
    setError('')
    try {
      setSelected(await GetChapterHistory(entry.id) as unknown as ChapterHistorySnapshot)
    } catch (e) {
      setError(String(e))
    }
  }

  async function takeSnapshot() {
    if (snapshotting) return
    setError('')
    setSnapshotting(true)
    try {
      const saved = await snapshotCurrentChapter(snapshotLabel)
      if (saved) {
        setSnapshotLabel('')
        await loadEntries(true)
      }
    } finally {
      setSnapshotting(false)
    }
  }

  async function restore() {
    if (!selected) return
    if (!window.confirm(`Restore the ${new Date(selected.entry.created_at).toLocaleString()} version of “${chapter?.title ?? 'this chapter'}”? Draftline will checkpoint the current text first.`)) return
    if (await restoreChapterHistory(selected.content)) closeChapterHistory()
  }

  const diffs = useMemo(() => diffContent(selected?.content ?? '', chapter?.content ?? ''), [selected?.content, chapter?.content])
  const changed = diffs.filter(row => row.hasChanges).length

  return (
    <div className="dialog-overlay">
      <div className="dialog chapter-history-dialog">
        <div className="chapter-history-header">
          <div>
            <div className="dialog-title">Chapter History</div>
            <div className="chapter-history-subtitle">{chapter?.title ?? 'Current section'}</div>
          </div>
          <button className="settings-close-btn" onClick={closeChapterHistory} title="Close">×</button>
        </div>

        <div className="chapter-history-body">
          <aside className="chapter-history-list">
            <div className="chapter-history-snapshot">
              <input
                className="dialog-input"
                placeholder="Label, e.g. Before rewrite"
                value={snapshotLabel}
                maxLength={80}
                onChange={e => setSnapshotLabel(e.target.value)}
                onKeyDown={e => { if (e.key === 'Enter') void takeSnapshot() }}
                disabled={!chapter?.id || !filePath}
              />
              <button
                className="dialog-btn primary"
                onClick={() => void takeSnapshot()}
                disabled={!chapter?.id || !filePath || snapshotting}
                title={filePath ? 'Store the chapter as it reads right now' : 'Save the project first'}
              >
                {snapshotting ? 'Saving…' : 'Snapshot now'}
              </button>
              {!filePath && <small>Save the project to enable snapshots.</small>}
            </div>
            <div className="chapter-history-list-label">Versions</div>
            {loading && <div className="chapter-history-empty">Loading…</div>}
            {!loading && entries.length === 0 && (
              <div className="chapter-history-empty">
                No versions yet. Take a snapshot before a rewrite, or keep writing: Draftline records a changed chapter after ten minutes of activity and on every applied AI pass.
              </div>
            )}
            {entries.map(entry => (
              <button
                key={entry.id}
                className={`chapter-history-entry${selected?.entry.id === entry.id ? ' active' : ''}`}
                onClick={() => void selectEntry(entry)}
              >
                <span>{new Date(entry.created_at).toLocaleString()}</span>
                <small>{entry.reason} · {entry.word_count.toLocaleString()} words</small>
              </button>
            ))}
          </aside>

          <section className="chapter-history-compare">
            {error && <div className="chapter-history-error">{error}</div>}
            {!selected && !error && <div className="chapter-history-empty large">Select a version to compare.</div>}
            {selected && (
              <>
                <div className="chapter-history-column-headings">
                  <span>Previous · {new Date(selected.entry.created_at).toLocaleString()}</span>
                  <span>Current · {changed} changed paragraph{changed === 1 ? '' : 's'}</span>
                </div>
                <div className="chapter-history-columns">
                  <ProseColumn diffs={diffs} side="old" />
                  <ProseColumn diffs={diffs} side="current" />
                </div>
              </>
            )}
          </section>
        </div>

        <div className="chapter-history-footer">
          <span>Snapshots are stored inside this .draftline project.</span>
          <div>
            <button className="dialog-btn" onClick={closeChapterHistory}>Close</button>
            <button className="dialog-btn primary" onClick={() => void restore()} disabled={!selected}>Restore this version</button>
          </div>
        </div>
      </div>
    </div>
  )
}
