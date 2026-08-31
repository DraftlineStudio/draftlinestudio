import { useEffect, useMemo, useState } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { GetChapterHistory, ListChapterHistory } from '../../../wailsjs/go/main/App'
import type { ChapterHistoryEntry, ChapterHistorySnapshot, ChapterItem } from '../../types/draftline'
import { diffContent } from '../../utils/diff'
import { useBookStore } from '../../store/bookStore'

function currentChapter(section: string, index: number, book: ReturnType<typeof useBookStore.getState>['book']): ChapterItem | null {
  if (!book || section === 'copyright') return null
  const items = section === 'front_matter' ? book.front_matter : section === 'back_matter' ? book.back_matter : book.body
  return items[index] ?? null
}

export default function ChapterHistoryDialog() {
  const { book, currentSection, currentIndex, closeChapterHistory, restoreChapterHistory } = useBookStore(useShallow(s => ({
    book: s.book,
    currentSection: s.currentSection,
    currentIndex: s.currentIndex,
    closeChapterHistory: s.closeChapterHistory,
    restoreChapterHistory: s.restoreChapterHistory,
  })))
  const chapter = currentChapter(currentSection, currentIndex, book)
  const [entries, setEntries] = useState<ChapterHistoryEntry[]>([])
  const [selected, setSelected] = useState<ChapterHistorySnapshot | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') closeChapterHistory()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [closeChapterHistory])

  useEffect(() => {
    let live = true
    async function load() {
      if (!chapter?.id || !book?.file_path) {
        setLoading(false)
        return
      }
      try {
        const list = await ListChapterHistory(chapter.id) as unknown as ChapterHistoryEntry[]
        if (!live) return
        setEntries(list ?? [])
        if (list?.length) {
          const snapshot = await GetChapterHistory(list[0].id) as unknown as ChapterHistorySnapshot
          if (live) setSelected(snapshot)
        }
      } catch (e) {
        if (live) setError(String(e))
      } finally {
        if (live) setLoading(false)
      }
    }
    void load()
    return () => { live = false }
  }, [chapter?.id, book?.file_path])

  async function selectEntry(entry: ChapterHistoryEntry) {
    setError('')
    try {
      setSelected(await GetChapterHistory(entry.id) as unknown as ChapterHistorySnapshot)
    } catch (e) {
      setError(String(e))
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
            <div className="chapter-history-list-label">Versions</div>
            {loading && <div className="chapter-history-empty">Loading…</div>}
            {!loading && entries.length === 0 && (
              <div className="chapter-history-empty">
                No versions yet. Draftline records a changed chapter after ten minutes of active writing.
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
                  <div className="chapter-history-prose old">
                    {diffs.map((row, i) => <p key={i}>{row.chunks.map((chunk, j) => chunk.type === 'insert' ? null : <span key={j} className={chunk.type === 'delete' ? 'history-deleted' : ''}>{chunk.text}</span>)}</p>)}
                  </div>
                  <div className="chapter-history-prose current">
                    {diffs.map((row, i) => <p key={i}>{row.chunks.map((chunk, j) => chunk.type === 'delete' ? null : <span key={j} className={chunk.type === 'insert' ? 'history-inserted' : ''}>{chunk.text}</span>)}</p>)}
                  </div>
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
