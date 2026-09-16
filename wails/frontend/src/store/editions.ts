// The publishing record's actions, lifted out of bookStore so the store keeps
// its size.
//
// Editions go through the same dirty/autosave funnel as the manuscript: an
// ISBN typed into the Book & Editions screen is saved by the same five-second
// autosave that saves a sentence typed into a chapter. There is no separate
// Save, which is the point — export and edition settings used to live in
// component state and were thrown away when the dialog closed.
//
// Each creating action returns the identifier of what it made, because the
// screen selects it straight afterwards.

import { duplicateAsNewEdition, emptyEditionIndex, newEdition, newFormat } from '../components/dialogs/editionModel'
import { withFrozenSnapshot, withReleasedSnapshot } from '../components/dialogs/snapshotModel'
import { kindForFormat } from '../components/dialogs/editionModel'
import type {
  BookData, Edition, EditionFormat, EditionIndex, EditionKind, EditionSnapshot,
} from '../types/draftline'

export interface EditionActions {
  addEdition: (year: string) => string
  duplicateEdition: (editionID: string, year: string) => string
  updateEdition: (editionID: string, patch: Partial<Edition>) => void
  removeEdition: (editionID: string) => void
  addFormat: (editionID: string, kind: EditionKind, word?: string) => string
  updateFormat: (formatID: string, patch: Partial<EditionFormat>) => void
  removeFormat: (formatID: string) => void
  freezeFormat: (formatID: string, snapshot: EditionSnapshot) => void
  releaseFormatSnapshot: (formatID: string) => void
  releaseEditionSnapshot: (editionID: string) => void
}

const mapEditions = (index: EditionIndex, fn: (edition: Edition) => Edition): EditionIndex =>
  ({ ...index, editions: index.editions.map(fn) })

// currentBook and commit are the store's own funnel: commit dirties the book
// and schedules the autosave. Passing them in keeps this file free of the
// store it serves, so it can be tested as plain data.
export function createEditionActions(
  currentBook: () => BookData | null,
  commit: (book: BookData) => void,
): EditionActions {
  const write = (book: BookData, editions: EditionIndex) => commit({ ...book, editions })

  return {
    addEdition: (year) => {
      const book = currentBook()
      if (!book) return ''
      const index = book.editions ?? emptyEditionIndex()
      const edition = newEdition(index, year)
      write(book, { ...index, editions: [...index.editions, edition] })
      return edition.id
    },

    duplicateEdition: (editionID, year) => {
      const book = currentBook()
      if (!book) return ''
      const index = book.editions ?? emptyEditionIndex()
      const next = duplicateAsNewEdition(index, editionID, year)
      if (next === index) return ''
      write(book, next)
      return next.editions[next.editions.length - 1].id
    },

    updateEdition: (editionID, patch) => {
      const book = currentBook()
      if (!book?.editions) return
      write(book, mapEditions(book.editions, edition =>
        (edition.id === editionID ? { ...edition, ...patch } : edition)))
    },

    // Removing an edition releases the formats under it. An edition that named
    // it as its predecessor is relinked to what that one pointed at, so the
    // copyright years of a book already published do not change because a
    // draft record in the middle was tidied away.
    removeEdition: (editionID) => {
      const book = currentBook()
      if (!book?.editions) return
      const index = book.editions
      const removed = index.editions.find(e => e.id === editionID)
      if (!removed) return
      write(book, {
        ...index,
        editions: index.editions
          .filter(e => e.id !== editionID)
          .map(e => (e.previous_edition_id === editionID
            ? { ...e, previous_edition_id: removed.previous_edition_id }
            : e)),
      })
    },

    addFormat: (editionID, kind, word) => {
      const book = currentBook()
      if (!book) return ''
      const index = book.editions ?? emptyEditionIndex()
      if (!index.editions.some(e => e.id === editionID)) return ''
      const format = newFormat(index, kind, word)
      write(book, mapEditions(index, edition =>
        (edition.id === editionID ? { ...edition, formats: [...edition.formats, format] } : edition)))
      return format.id
    },

    updateFormat: (formatID, patch) => {
      const book = currentBook()
      if (!book?.editions) return
      // The format word is the choice a writer makes; the kind follows it.
      // Without this a record created as an eBook and changed to Paperback
      // stays an ebook underneath, showing ebook settings and exporting as
      // one, which is invisible and wrong.
      const settled = patch.format !== undefined
        ? { ...patch, kind: kindForFormat(patch.format) }
        : patch
      write(book, mapEditions(book.editions, edition => ({
        ...edition,
        formats: edition.formats.map(format => (format.id === formatID ? { ...format, ...settled } : format)),
      })))
    },

    // Removing a format releases its claim on the text it was published from
    // FIRST, so that the catalogue and the reference count settle together.
    // Dropping the format on its own would leave a record nothing points at,
    // which the writer then reaps on the next save — correct, but a save later
    // than it needed to be and with no chance to notice the words were shared.
    removeFormat: (formatID) => {
      const book = currentBook()
      if (!book?.editions) return
      const released = withReleasedSnapshot(book.editions, formatID)
      write(book, mapEditions(released, edition => ({
        ...edition,
        formats: edition.formats.filter(format => format.id !== formatID),
      })))
    },

    // freezeFormat records the text one format was published from. The record
    // and the format's pointer to it are written in one update; see
    // withFrozenSnapshot for why they must not be separable.
    freezeFormat: (formatID, snapshot) => {
      const book = currentBook()
      if (!book?.editions) return
      write(book, withFrozenSnapshot(book.editions, formatID, snapshot))
    },

    // releaseFormatSnapshot puts one format back to exporting the working
    // draft. The words themselves stay while any other format is still
    // published from them.
    releaseFormatSnapshot: (formatID) => {
      const book = currentBook()
      if (!book?.editions) return
      write(book, withReleasedSnapshot(book.editions, formatID))
    },

    // A locked text belongs to the edition, not to one of its ISBNs: every
    // format exported with it points at the same record. Releasing therefore
    // lets go of it on all of them at once, and the words themselves are only
    // dropped when nothing anywhere still stands on them.
    releaseEditionSnapshot: (editionID) => {
      const book = currentBook()
      if (!book?.editions) return
      const edition = book.editions.editions.find(one => one.id === editionID)
      if (!edition) return
      const next = edition.formats.reduce(
        (index, format) => (format.snapshot_id ? withReleasedSnapshot(index, format.id) : index),
        book.editions,
      )
      if (next !== book.editions) write(book, next)
    },
  }
}
