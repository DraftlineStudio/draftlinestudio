// The save pipeline, lifted out of bookStore so the store keeps its size.
//
// This is the code that decides whether an author's words reached disk, so the
// rules it enforces are worth stating plainly:
//
//   - Every save goes through one chain. Two SaveBook calls can never be in
//     flight together, so they cannot interleave on the Go side.
//   - A save snapshots the book AFTER the previous save finishes, so a queued
//     save always writes the newest state rather than the state it was asked
//     about.
//   - isDirty is cleared only when nothing was edited while the save was in
//     flight. The revision counter is what tells those apart.
//   - A save that finishes after the project was closed or replaced must not
//     write its path or its dirty state onto whatever is open now. The session
//     counter is what tells those apart.
//
// The state above lives in this factory's closure rather than at module scope,
// so a test can build a pipeline of its own without the previous one's
// timers and counters still running.

import { SaveBook, SaveBookAs } from '../../wailsjs/go/main/App'
import type { BookData } from '../types/draftline'

const AUTO_SAVE_DELAY = 5000

export type SaveOutcome =
  | { status: 'saved'; filePath: string; warning?: string }
  | { status: 'stale'; filePath: string; warning?: string }
  | { status: 'superseded' }
  | { status: 'cancelled' }
  | { status: 'error'; message: string }

export interface SaveHost {
  getBook: () => BookData | null
  getIsDirty: () => boolean
  /** Write the saved path back onto the book, clearing dirty only if asked. */
  recordSaved: (filePath: string, clearDirty: boolean) => void
  setAutoSaving: (saving: boolean) => void
  autoSaveEnabled: () => boolean
  setStatus: (message: string) => void
  getStatusMessage: () => string
  /** Called when a new project takes over, before its first edit. */
  onSessionBegin: () => void
}

export interface SavePipeline {
  performSave: (kind: 'save' | 'saveAs') => Promise<SaveOutcome>
  scheduleAutoSave: () => void
  enqueueSaveTask: <T>(task: () => Promise<T>) => Promise<T>
  beginBookSession: () => void
  /** Identifies the open project; a save that outlives it is discarded. */
  currentSession: () => number
  /** Bumped by every dirty-marking edit; distinguishes a stale save. */
  currentRevision: () => number
}

export function createSavePipeline(host: SaveHost): SavePipeline {
  let autoSaveTimer: ReturnType<typeof setTimeout> | null = null
  let saveRevision = 0
  let bookSession = 0
  let saveChain: Promise<unknown> = Promise.resolve()

  const beginBookSession = () => {
    bookSession++
    if (autoSaveTimer) {
      clearTimeout(autoSaveTimer)
      autoSaveTimer = null
    }
    host.onSessionBegin()
  }

  const performSave = (kind: 'save' | 'saveAs'): Promise<SaveOutcome> => {
    const run = saveChain.then(async (): Promise<SaveOutcome> => {
      const book = host.getBook()
      if (!book) return { status: 'error', message: 'no book open' }
      const rev = saveRevision
      const session = bookSession
      try {
        const result = kind === 'saveAs' ? await SaveBookAs(book as any) : await SaveBook(book as any)
        if (bookSession !== session) return { status: 'superseded' }
        if (result.success) {
          // A save that wrote the book but could not carry everything across
          // says so, in place of the usual "Saved" line. Losing that quietly is
          // how an author finds out weeks later.
          const warning = result.warnings?.[0]
          const fresh = saveRevision === rev
          host.recordSaved(result.file_path, fresh)
          return fresh
            ? { status: 'saved', filePath: result.file_path, warning }
            // Edited while saving: isDirty stays so the re-armed autosave
            // persists the newer state.
            : { status: 'stale', filePath: result.file_path, warning }
        }
        if (result.error === 'cancelled') return { status: 'cancelled' }
        return { status: 'error', message: result.error || 'unknown error' }
      } catch (e) {
        return { status: 'error', message: String(e) }
      }
    })
    saveChain = run.catch(() => {})
    return run
  }

  const scheduleAutoSave = () => {
    saveRevision++
    if (autoSaveTimer) clearTimeout(autoSaveTimer)
    if (!host.autoSaveEnabled()) {
      autoSaveTimer = null
      return
    }
    autoSaveTimer = setTimeout(async () => {
      if (!host.autoSaveEnabled()) return
      const book = host.getBook()
      if (book && host.getIsDirty() && book.file_path) {
        const session = bookSession
        host.setAutoSaving(true)
        const outcome = await performSave('save')
        if (bookSession !== session) return
        host.setAutoSaving(false)
        if (outcome.status === 'saved') {
          host.setStatus('Auto-saved')
          setTimeout(() => {
            if (host.getStatusMessage() === 'Auto-saved') host.setStatus('')
          }, 2000)
        }
      }
    }, AUTO_SAVE_DELAY)
  }

  const enqueueSaveTask = <T,>(task: () => Promise<T>): Promise<T> => {
    const run = saveChain.then(task)
    saveChain = run.catch(() => {})
    return run
  }

  return {
    performSave,
    scheduleAutoSave,
    enqueueSaveTask,
    beginBookSession,
    currentSession: () => bookSession,
    currentRevision: () => saveRevision,
  }
}
