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
import type { BookData, Edition, EditionFormat, EditionIndex, EditionKind } from '../types/draftline'

export interface EditionActions {
  addEdition: (year: string) => string
  duplicateEdition: (editionID: string, year: string) => string
  updateEdition: (editionID: string, patch: Partial<Edition>) => void
  removeEdition: (editionID: string) => void
  addFormat: (editionID: string, kind: EditionKind) => string
  updateFormat: (formatID: string, patch: Partial<EditionFormat>) => void
  removeFormat: (formatID: string) => void
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

    addFormat: (editionID, kind) => {
      const book = currentBook()
      if (!book) return ''
      const index = book.editions ?? emptyEditionIndex()
      if (!index.editions.some(e => e.id === editionID)) return ''
      const format = newFormat(index, kind)
      write(book, mapEditions(index, edition =>
        (edition.id === editionID ? { ...edition, formats: [...edition.formats, format] } : edition)))
      return format.id
    },

    updateFormat: (formatID, patch) => {
      const book = currentBook()
      if (!book?.editions) return
      write(book, mapEditions(book.editions, edition => ({
        ...edition,
        formats: edition.formats.map(format => (format.id === formatID ? { ...format, ...patch } : format)),
      })))
    },

    removeFormat: (formatID) => {
      const book = currentBook()
      if (!book?.editions) return
      write(book, mapEditions(book.editions, edition => ({
        ...edition,
        formats: edition.formats.filter(format => format.id !== formatID),
      })))
    },
  }
}
