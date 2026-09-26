// Getting a book onto the screen, lifted out of bookStore so the store keeps
// its size.
//
// Three ways in, one landing: a path the writer picked, a copy taken because
// another device holds the original, or a manuscript imported from EPUB or
// DOCX. They differ only in which backend call produces the book; everything
// after that is the same, which is why it is written once here.

import { ImportEPUB, ImportDOCX, OpenRecentProject, BookTakeoverStatus } from '../../wailsjs/go/main/App'
import type { BookData, Section } from '../types/draftline'
import {
  askForBook, deviceHoldingBook, stopAskingForBook, waitForHandover,
  type BookLockWarning, type HandoverOutcome,
} from './bookLock'

/** How a handover ended, when it ended in anything but an open book. */
export interface HandoverReport {
  path: string
  title: string
  message: string
  /** Opening a copy is the way out of every one of these. */
  offerCopy: boolean
}

export interface BookAdoption {
  /** Open the archive at path, unless another device holds a live claim. */
  loadBookFromPath: (path: string) => Promise<void>
  /** Put the book a given call produces on screen. */
  adoptBook: (open: () => Promise<BookData>, path: string) => Promise<void>
  importExternalBook: (path: string) => Promise<void>
  /** Ask the device holding a book to hand it over, then wait for the copy. */
  requestBookFromDevice: (path: string) => Promise<void>
  /** Stop waiting, and take the request back so nobody grants to nobody. */
  cancelBookRequest: () => void
}

export interface AdoptionHost {
  setLockWarning: (warning: BookLockWarning | null) => void
  setOpening: (opening: boolean) => void
  /** The line under the opening overlay's spinner. */
  setOpeningLabel: (label: string) => void
  /** Says how a handover ended, when it ended in anything but the book. */
  reportHandover: (report: HandoverReport | null) => void
  /** Offers a way out of a wait that is only as fast as the folder syncs. */
  setWaitingForDevice: (waiting: boolean) => void
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
  // Set while a handover is being waited on, so Cancel on the overlay reaches
  // the poll inside waitForHandover.
  let cancelledRequest = false

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

  // watchHandover holds the opening overlay over the welcome screen while the
  // other machine answers and its copy of the book crosses the sync folder.
  //
  // The overlay is the input shield the book load already uses, so nothing can
  // be clicked underneath it; the difference is that this wait is only as fast
  // as somebody else's Dropbox, so it offers a way out.
  const watchHandover = async (path: string, firstMessage: string): Promise<void> => {
    cancelledRequest = false
    host.setOpeningLabel(firstMessage || 'Waiting for the other device…')
    host.setWaitingForDevice(true)
    host.setOpening(true)
    let outcome: HandoverOutcome
    try {
      outcome = await waitForHandover(path, {
        onProgress: host.setOpeningLabel,
        isCancelled: () => cancelledRequest,
      })
    } finally {
      host.setWaitingForDevice(false)
      host.setOpening(false)
      host.setOpeningLabel('')
    }

    switch (outcome.status) {
      case 'ready':
        // The bytes are proven, so the request has done its job and the
        // receipt beside the book is no longer needed.
        stopAskingForBook(path)
        await adoptBook(() => OpenRecentProject(path), path)
        return
      case 'released':
        // Nobody holds it any more, so this is an ordinary open of a free book
        // — the same thing that happens when no claim was ever there.
        stopAskingForBook(path)
        host.setStatus('The other computer had already let this book go.')
        await adoptBook(() => OpenRecentProject(path), path)
        return
      case 'unanswered':
        // Nothing answered, so the book is this machine's to take. Opening it
        // claims it, and the other machine — if it is running at all — sees a
        // claim that is not its own within seconds, puts anything unsaved in a
        // file of its own, and closes the book.
        //
        // There is no fingerprint to check here, because a fingerprint is
        // something the other machine writes when it hands over and nothing
        // handed over. What makes that acceptable rather than reckless is what
        // silence means: a machine that will not answer in thirty seconds is
        // switched off or asleep, and a machine that is not running has not
        // written to the book either.
        stopAskingForBook(path)
        host.setStatus('The other computer did not answer, so this one took the book.')
        await adoptBook(() => OpenRecentProject(path), path)
        return
      case 'declined':
        stopAskingForBook(path)
        // The status bar is not on screen here — a failed open leaves the
        // writer on the launch screen, which has no status bar — so a refusal
        // reported that way is a refusal nobody ever sees. It has to be said
        // where they are standing, next to the thing to do about it.
        host.reportHandover({
          path,
          title: 'The other computer kept the book',
          message: outcome.message,
          offerCopy: true,
        })
        return
      case 'cancelled':
        // Taking the request back matters: left behind, it would be granted
        // later by a machine with nobody waiting, and that machine would save,
        // release and close a book somebody may be sitting in front of.
        stopAskingForBook(path)
        host.setStatus('Stopped waiting for the other device.')
        return
      case 'unverifiable':
        // The other device handed the book over but could not prove what it
        // handed over, which a sync client holding the file open is enough to
        // cause. Opening would be opening on hope, so it does not.
        host.reportHandover({
          path,
          title: 'This copy could not be checked',
          message: `${outcome.message} Asking again in a moment usually settles it.`,
          offerCopy: true,
        })
        return
      default:
        host.reportHandover({
          path,
          title: 'The handover did not finish',
          message: outcome.message,
          offerCopy: true,
        })
    }
  }

  // resumeHandover reports whether this open was taken over by a handover that
  // was already granted.
  const resumeHandover = async (path: string): Promise<boolean> => {
    let status
    try {
      status = await BookTakeoverStatus(path)
    } catch {
      return false
    }
    if (!status?.granted) return false
    if (status.arrived) {
      stopAskingForBook(path)
      return false
    }
    await watchHandover(path, status.message)
    return true
  }

  return {
    adoptBook,

    // isOpening drives the full-screen overlay (feedback + input shield); the
    // entry guard in each open action prevents a second open racing the first
    // — large archives take a moment and the UI stays live while Go parses
    // them.
    // There is no way past these checks. It used to take a force flag, for the
    // dialog's Open Anyway; that opened whichever copy of the book happened to
    // be on this machine, which during a sync is the one from before the other
    // machine's last save. Every remaining route either proves the copy is
    // current or makes a separate book.
    loadBookFromPath: async (path) => {
      host.setLockWarning(null)
      // A handover already granted but not yet finished arriving picks up where
      // it left off. The grant is kept beside the book precisely so that walking
      // away and coming back resumes the same wait, with the same proof, rather
      // than opening whatever copy happens to be there.
      if (await resumeHandover(path)) return
      const holder = await deviceHoldingBook(path)
      if (holder) {
        host.setLockWarning({ path, info: holder })
        return
      }
      await adoptBook(() => OpenRecentProject(path), path)
    },

    requestBookFromDevice: async (path) => {
      host.setLockWarning(null)
      host.reportHandover(null)
      const asked = await askForBook(path)
      if (!asked || (!asked.asked && asked.message)) {
        host.setStatus(asked?.message || 'Could not ask the other device for this book.')
        return
      }
      await watchHandover(path, asked.message)
    },

    cancelBookRequest: () => {
      cancelledRequest = true
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
