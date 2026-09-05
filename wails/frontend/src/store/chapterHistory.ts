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
