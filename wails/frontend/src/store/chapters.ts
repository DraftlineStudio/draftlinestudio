// Chapter and book-record mutations, lifted out of bookStore so the store
// keeps its size.
//
// Every action here is the same shape: read the book, build the next one, hand
// it back through the host. The host owns the store — marking it dirty, moving
// the selection, arming the autosave — so nothing in this file knows what a
// zustand store is, and the ordering of the three post-mutation steps stays
// where it can be read at the call site rather than hidden in a helper.

import type {
  BookData, ChapterItem, Metadata, ReadAloudCast, Section,
} from '../types/draftline'

export interface ChapterActions {
  updateCurrentContent: (html: string) => void
  /**
   * Write content to a named chapter rather than whatever is selected now.
   *
   * The editor debounces keystrokes, so an edit can still be in hand when the
   * writer moves to another chapter. It belongs to the chapter it was typed
   * into, and this is how it gets there.
   */
  updateChapterContent: (section: Section, index: number, html: string) => void
  updateChapterTitle: (section: Section, index: number, title: string) => void
  updateChapterSubtitle: (section: Section, index: number, subtitle: string) => void
  addChapter: (section: Section, item: ChapterItem) => void
  deleteChapter: (section: Section, index: number) => void
  moveChapter: (section: Section, from: number, to: number) => void
  updateMetadata: (metadata: Partial<Metadata>) => void
  updateCopyright: (html: string) => void
  updateReadAloudCast: (cast: ReadAloudCast) => void
}

/** What an action changes. `prose` bumps the revision analysis watches. */
export interface ChapterPatch {
  book: BookData
  currentSection?: Section
  currentIndex?: number
  prose?: boolean
}

export interface ChapterHost {
  read: () => { book: BookData | null; currentSection: Section; currentIndex: number }
  apply: (patch: ChapterPatch) => void
  /** Queue a chapter for the rolling history snapshot. */
  noteHistory: (chapterID?: string) => void
  autosave: () => void
}

export function newChapterID(): string {
  return typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? `ch-${crypto.randomUUID().replace(/-/g, '')}`
    : `ch-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
}

export function getSectionArray(book: BookData, section: Section): ChapterItem[] {
  switch (section) {
    case 'front_matter': return book.front_matter
    case 'body': return book.body
    case 'back_matter': return book.back_matter
    default: return []
  }
}

export function setSectionArray(book: BookData, section: Section, items: ChapterItem[]): BookData {
  switch (section) {
    case 'front_matter': return { ...book, front_matter: items }
    case 'body': return { ...book, body: items }
    case 'back_matter': return { ...book, back_matter: items }
    default: return book
  }
}

export function createChapterActions(host: ChapterHost): ChapterActions {
  const updateChapterContent = (section: Section, index: number, html: string) => {
    const { book } = host.read()
    if (!book) return
    if (section === 'copyright') {
      if (book.copyright === html) return
      host.apply({ book: { ...book, copyright: html }, prose: true })
      host.autosave()
      return
    }
    const items = getSectionArray(book, section)
    if (!items[index]) return
    // A flush that arrives with nothing new must not dirty the book, or
    // closing a project would always have something left to save.
    if (items[index].content === html) return
    const updated = items.map((item, i) => i === index ? { ...item, content: html } : item)
    host.apply({ book: setSectionArray(book, section, updated), prose: true })
    host.noteHistory(items[index].id)
    host.autosave()
  }

  return {
    updateChapterContent,

    updateCurrentContent: (html) => {
      const { currentSection, currentIndex } = host.read()
      updateChapterContent(currentSection, currentIndex, html)
    },

    updateChapterTitle: (section, index, title) => {
      const { book } = host.read()
      if (!book || section === 'copyright') return
      const items = getSectionArray(book, section)
      const updated = items.map((item, i) => i === index ? { ...item, title } : item)
      host.apply({ book: setSectionArray(book, section, updated), prose: true })
      host.noteHistory(items[index]?.id)
      host.autosave()
    },

    updateChapterSubtitle: (section, index, subtitle) => {
      const { book } = host.read()
      if (!book || section === 'copyright') return
      const items = getSectionArray(book, section)
      const updated = items.map((item, i) => i === index ? { ...item, subtitle } : item)
      host.apply({ book: setSectionArray(book, section, updated), prose: true })
      host.noteHistory(items[index]?.id)
      host.autosave()
    },

    addChapter: (section, item) => {
      const { book } = host.read()
      if (!book || section === 'copyright') return
      const items = [...getSectionArray(book, section), { ...item, id: item.id || newChapterID() }]
      host.apply({
        book: setSectionArray(book, section, items),
        currentSection: section,
        currentIndex: items.length - 1,
        prose: true,
      })
      host.autosave()
    },

    deleteChapter: (section, index) => {
      const { book, currentSection, currentIndex } = host.read()
      if (!book || section === 'copyright') return
      const items = getSectionArray(book, section)
      // A book always has at least one body chapter to write in.
      if (items.length <= 1 && section === 'body') return
      const updated = items.filter((_, i) => i !== index)
      let newIndex = currentIndex
      if (section === currentSection && index === currentIndex) {
        newIndex = Math.max(0, index - 1)
      } else if (section === currentSection && index < currentIndex) {
        newIndex = currentIndex - 1
      }
      host.apply({
        book: setSectionArray(book, section, updated),
        currentIndex: Math.min(newIndex, updated.length - 1),
        prose: true,
      })
      host.autosave()
    },

    moveChapter: (section, from, to) => {
      const { book, currentSection, currentIndex } = host.read()
      if (!book || section === 'copyright') return
      const items = [...getSectionArray(book, section)]
      const [moved] = items.splice(from, 1)
      items.splice(to, 0, moved)
      // The selection follows the chapter it was on, wherever it landed.
      let newIndex = currentIndex
      if (section === currentSection) {
        if (currentIndex === from) newIndex = to
        else if (from < currentIndex && to >= currentIndex) newIndex = currentIndex - 1
        else if (from > currentIndex && to <= currentIndex) newIndex = currentIndex + 1
      }
      host.apply({ book: setSectionArray(book, section, items), currentIndex: newIndex, prose: true })
      host.autosave()
    },

    updateMetadata: (metadata) => {
      const { book } = host.read()
      if (!book) return
      host.apply({ book: { ...book, metadata: { ...book.metadata, ...metadata } } })
      host.autosave()
    },

    updateCopyright: (html) => {
      const { book } = host.read()
      if (!book) return
      host.apply({ book: { ...book, copyright: html }, prose: true })
      host.autosave()
    },

    updateReadAloudCast: (cast) => {
      const { book } = host.read()
      if (!book) return
      host.apply({ book: { ...book, read_aloud_cast: cast } })
      host.autosave()
    },
  }
}
