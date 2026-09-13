import type { BookData, ChapterItem, Section } from '../types/draftline'
import { SaveBookSnapshots } from '../../wailsjs/go/main/App'

const HISTORY_SNAPSHOT_DELAY = 10 * 60 * 1000

let snapshotTimer: ReturnType<typeof setTimeout> | null = null
const changedChapterIDs = new Set<string>()

export interface ChapterHistoryDependencies {
  getBook: () => BookData | null
  getSession: () => number
  getRevision: () => number
  enqueue: (task: () => Promise<void>) => Promise<void>
  onSaved: (filePath: string, revision: number) => void
  setStatus: (message: string) => void
}

export interface AIHistoryBoundary {
  chapter: ChapterItem
  section: Exclude<Section, 'copyright'>
  beforeHtml: string
  afterHtml: string
  reason: string
}

export function resetChapterHistorySession() {
  if (snapshotTimer) clearTimeout(snapshotTimer)
  snapshotTimer = null
  changedChapterIDs.clear()
}

// Version history deliberately does not consult the activity-autosave toggle.
// Disabling background manuscript saves must not disable recovery history.
export function scheduleChapterHistory(chapterID: string | undefined, deps: ChapterHistoryDependencies) {
  if (!chapterID) return
  changedChapterIDs.add(chapterID)
  if (snapshotTimer) return
  snapshotTimer = setTimeout(() => {
    snapshotTimer = null
    const ids = new Set(changedChapterIDs)
    changedChapterIDs.clear()
    if (ids.size === 0) return
    const session = deps.getSession()
    void deps.enqueue(async () => {
      if (deps.getSession() !== session) return
      const book = deps.getBook()
      if (!book?.file_path) return
      const requests = (['front_matter', 'body', 'back_matter'] as const).flatMap(section =>
        book[section]
          .filter(chapter => chapter.id && ids.has(chapter.id))
          .map(chapter => ({
            chapter_id: chapter.id!, section, chapter_title: chapter.title,
            content: chapter.content, reason: 'Writing session',
          })),
      )
      if (requests.length === 0) return
      try {
        const result = await SaveBookSnapshots(book as any, requests as any)
        if (deps.getSession() !== session) return
        if (result.success) deps.setStatus('Chapter history updated')
        else {
          ids.forEach(id => changedChapterIDs.add(id))
          deps.setStatus(`Chapter history failed: ${result.error || 'unknown error'}`)
        }
      } catch (error) {
        if (deps.getSession() === session) {
          ids.forEach(id => changedChapterIDs.add(id))
          deps.setStatus(`Chapter history failed: ${String(error)}`)
        }
      }
    })
  }, HISTORY_SNAPSHOT_DELAY)
}

// The slice of bookStore a manual snapshot needs: where the writer is, and
// the setter that keeps the store's chapter text in step with the editor.
export interface CurrentChapterAccess {
  book: BookData | null
  currentSection: Section
  currentIndex: number
  updateCurrentContent: (html: string) => void
}

// A writer-initiated checkpoint, taken before a rewrite or an AI pass. Saves
// the manuscript in the same atomic archive rewrite so the snapshot and the
// current text are committed together. `reason` is the label shown in the
// version list ("Before rewrite"). `liveHtml` is the editor's current HTML
// when an editor is mounted: the store copy trails it by a typing debounce,
// so it is synced first and the snapshot carries what the writer sees.
// Resolves true once the archive holds the snapshot (or an identical latest
// version already did).
export function saveManualChapterSnapshot(
  reason: string | undefined,
  get: () => CurrentChapterAccess,
  liveHtml: string | null | undefined,
  deps: ChapterHistoryDependencies,
): Promise<boolean> {
  const { book, currentSection: section, currentIndex } = get()
  if (!book || section === 'copyright') return Promise.resolve(false)
  const stored = book[section][currentIndex]
  if (!stored?.id) return Promise.resolve(false)
  if (liveHtml != null && liveHtml !== stored.content) get().updateCurrentContent(liveHtml)
  const chapter = get().book?.[section][currentIndex]
  if (!chapter?.id) return Promise.resolve(false)
  const label = reason?.trim() || 'Manual snapshot'
  return persistManualSnapshot(chapter, section, label, deps)
}

function persistManualSnapshot(
  chapter: ChapterItem,
  section: Exclude<Section, 'copyright'>,
  reason: string,
  deps: ChapterHistoryDependencies,
): Promise<boolean> {
  const session = deps.getSession()
  let saved = false
  return deps.enqueue(async () => {
    if (deps.getSession() !== session) return
    const book = deps.getBook()
    if (!book?.file_path) {
      deps.setStatus('Save the project before taking a chapter snapshot')
      return
    }
    const revision = deps.getRevision()
    try {
      const result = await SaveBookSnapshots(book as any, [
        {
          chapter_id: chapter.id, section, chapter_title: chapter.title,
          content: chapter.content, reason,
        },
      ] as any)
      if (deps.getSession() !== session) return
      if (!result.success) {
        deps.setStatus(`Chapter snapshot failed: ${result.error || 'unknown error'}`)
        return
      }
      deps.onSaved(result.file_path, revision)
      deps.setStatus(`Snapshot saved: ${reason}`)
      saved = true
    } catch (error) {
      if (deps.getSession() === session) deps.setStatus(`Chapter snapshot failed: ${String(error)}`)
    }
  }).then(() => saved)
}

// Persist both sides of an explicit AI edit in the same atomic archive rewrite.
export function saveAIChapterHistory(boundary: AIHistoryBoundary, deps: ChapterHistoryDependencies): Promise<void> {
  const { chapter, section, beforeHtml, afterHtml, reason } = boundary
  if (!chapter.id || beforeHtml === afterHtml) return Promise.resolve()
  const session = deps.getSession()
  return deps.enqueue(async () => {
    if (deps.getSession() !== session) return
    const book = deps.getBook()
    if (!book?.file_path) {
      deps.setStatus('Save the project to persist AI chapter history')
      return
    }
    const revision = deps.getRevision()
    try {
      const result = await SaveBookSnapshots(book as any, [
        {
          chapter_id: chapter.id, section, chapter_title: chapter.title,
          content: beforeHtml, reason: `Before ${reason}`,
        },
        {
          chapter_id: chapter.id, section, chapter_title: chapter.title,
          content: afterHtml, reason: `After ${reason}`,
        },
      ] as any)
      if (deps.getSession() !== session) return
      if (!result.success) {
        deps.setStatus(`AI chapter history failed: ${result.error || 'unknown error'}`)
        return
      }
      deps.onSaved(result.file_path, revision)
      deps.setStatus('AI change and recovery snapshots saved')
    } catch (error) {
      if (deps.getSession() === session) deps.setStatus(`AI chapter history failed: ${String(error)}`)
    }
  })
}
