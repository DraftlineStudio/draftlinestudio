// Getting a book onto the screen, lifted out of bookStore so the store keeps
// its size.
//
// Three ways in, one landing: a path the writer picked, a copy taken because
// another device holds the original, or a manuscript imported from EPUB or
// DOCX. They differ only in which backend call produces the book; everything
// after that is the same, which is why it is written once here.

import { ImportEPUB, ImportDOCX, OpenRecentProject } from '../../wailsjs/go/main/App'
import type { BookData, Section } from '../types/draftline'
import { deviceHoldingBook, type BookLockWarning } from './bookLock'

export interface BookAdoption {
  /** Open the archive at path, unless another device holds a live claim. */
  loadBookFromPath: (path: string, force?: boolean) => Promise<void>
  /** Put the book a given call produces on screen. */
  adoptBook: (open: () => Promise<BookData>, path: string) => Promise<void>
  importExternalBook: (path: string) => Promise<void>
}

export interface AdoptionHost {
  setLockWarning: (warning: BookLockWarning | null) => void
  setOpening: (opening: boolean) => void
  /** Show the book, reset the selection, and start it clean. */
  placeBook: (book: BookData, section: Section) => void
  beginSession: () => void
  setStatus: (message: string) => void
  clearPendingDiff: () => void
  /** Runs off the critical path, once the book is already on screen. */
  rememberRecent: (book: BookData, path: string) => void
  loadImported: (book: BookData) => void
}

export function createBookAdoption(host: AdoptionHost): BookAdoption {
  const adoptBook = async (open: () => Promise<BookData>, path: string): Promise<void> => {
    host.setOpening(true)
    try {
      const book: BookData = await open()
      if (!book?.version) return
      const section: Section = book.body.length > 0 ? 'body' : 'front_matter'
      host.beginSession()
      host.placeBook(book, section)
      host.setStatus(`Opened: ${book.metadata.title}`)
      host.clearPendingDiff()
      // Recents bookkeeping runs AFTER the book is on screen, off the
      // critical path.
      setTimeout(() => host.rememberRecent(book, path), 0)
    } catch (e) {
      host.setStatus(`Error opening file: ${e}`)
    } finally {
      host.setOpening(false)
    }
  }

  return {
    adoptBook,

    // isOpening drives the full-screen overlay (feedback + input shield); the
    // entry guard in each open action prevents a second open racing the first
    // — large archives take a moment and the UI stays live while Go parses
    // them.
    loadBookFromPath: async (path, force = false) => {
      host.setLockWarning(null)
      if (!force) {
        const holder = await deviceHoldingBook(path)
        if (holder) {
          host.setLockWarning({ path, info: holder })
          return
        }
      }
      await adoptBook(() => OpenRecentProject(path), path)
    },

    importExternalBook: async (path) => {
      const ext = (path.split('.').pop() || '').toLowerCase()
      try {
        const result = ext === 'epub' ? await ImportEPUB(path) : await ImportDOCX(path)
        if (!result.success || !result.book) {
          if (result.error !== 'cancelled') host.setStatus(`Import failed: ${result.error || 'unknown error'}`)
          return
        }
        host.loadImported(result.book as unknown as BookData)
        const warnings = result.warnings?.length || 0
        if (warnings > 0) {
          host.setStatus(`Imported: ${result.book.metadata?.title || path} (${warnings} import warning${warnings === 1 ? '' : 's'})`)
        }
      } catch (e) {
        host.setStatus(`Import failed: ${e}`)
      }
    },
  }
}
