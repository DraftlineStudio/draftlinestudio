import { create } from 'zustand'
import type { BookData, ChapterItem, Character, Metadata, Section, StoryBible } from '../types/draftline'
import type { ParagraphDiff, DiffChange } from '../utils/diff'
import { extractChanges, assembleFromChanges } from '../utils/diff'

import { NewBook, OpenBookDialog, SaveBook, SaveBookAs, OpenRecentProject, AddRecentProject } from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'
import { useAppStore } from './appStore'

interface DialogState {
  showMetadata: boolean
  showNewChapter: boolean
  newChapterSection: Section | null
  showUnsavedWarning: boolean
  pendingAction: 'new' | 'open' | null
  showNewBookWizard: boolean
}

interface BookStore {
  book: BookData | null
  currentSection: Section
  currentIndex: number
  isDirty: boolean
  darkMode: boolean
  dialogs: DialogState
  statusMessage: string

  // Inline AI diff (per-change word-level tracking)
  pendingDiff: {
    diffs: ParagraphDiff[]
    changes: DiffChange[]       // flat list of all change groups
    focusedChangeIdx: number    // currently highlighted change
    originalHtml: string
  } | null
  setPendingDiff: (payload: { diffs: ParagraphDiff[]; originalHtml: string }) => void
  acceptChange: (idx: number) => void
  rejectChange: (idx: number) => void
  setFocusedChange: (idx: number) => void
  prevChange: () => void
  nextChange: () => void
  acceptAllDiff: () => void
  rejectAllDiff: () => void
  applyPendingDiff: () => void
  clearPendingDiff: () => void

  // File ops
  newBook: () => Promise<void>
  openBook: () => Promise<void>
  openRecentBook: (path: string) => Promise<void>
  saveBook: () => Promise<void>
  saveBookAs: () => Promise<void>
  closeProject: () => Promise<void>

  // Navigation
  setCurrentChapter: (section: Section, index: number) => void

  // Content mutations
  updateCurrentContent: (html: string) => void
  updateChapterTitle: (section: Section, index: number, title: string) => void
  addChapter: (section: Section, item: ChapterItem) => void
  deleteChapter: (section: Section, index: number) => void
  moveChapter: (section: Section, from: number, to: number) => void
  updateMetadata: (metadata: Partial<Metadata>) => void
  updateCopyright: (html: string) => void

  // Story bible
  addCharacter: (char: Character) => void
  updateCharacter: (char: Character) => void
  deleteCharacter: (id: string) => void
  updateStoryBibleText: (field: 'plot_notes' | 'timeline', text: string) => void

  // Layout
  leftPanelOpen: boolean
  rightPanelOpen: boolean
  toggleLeftPanel: () => void
  toggleRightPanel: () => void

  // UI
  toggleDarkMode: () => void
  openMetadataDialog: () => void
  closeMetadataDialog: () => void
  openNewChapterDialog: (section: Section) => void
  closeNewChapterDialog: () => void
  setStatusMessage: (msg: string) => void
  closeUnsavedWarning: () => void
  saveAndProceed: () => Promise<void>
  discardAndProceed: () => Promise<void>
  initBook: () => Promise<void>
  confirmNewBook: (title: string, author: string, publisher: string) => Promise<void>
  loadImportedBook: (book: BookData) => void
  cancelNewBookWizard: () => void
  setDarkMode: (v: boolean) => void
}

function getSectionArray(book: BookData, section: Section): ChapterItem[] {
  switch (section) {
    case 'front_matter': return book.front_matter
    case 'body': return book.body
    case 'back_matter': return book.back_matter
    default: return []
  }
}

function countWords(book: BookData): number {
  const stripHtml = (html: string) => html.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
  let total = 0
  for (const ch of [...book.front_matter, ...book.body, ...book.back_matter]) {
    const text = stripHtml(ch.content)
    if (text) total += text.split(/\s+/).length
  }
  if (book.copyright) {
    const text = stripHtml(book.copyright)
    if (text) total += text.split(/\s+/).length
  }
  return total
}

function setSectionArray(book: BookData, section: Section, items: ChapterItem[]): BookData {
  switch (section) {
    case 'front_matter': return { ...book, front_matter: items }
    case 'body': return { ...book, body: items }
    case 'back_matter': return { ...book, back_matter: items }
    default: return book
  }
}

export const useBookStore = create<BookStore>((set, get) => ({
  book: null,
  currentSection: 'body',
  currentIndex: 0,
  isDirty: false,
  darkMode: true,
  leftPanelOpen: true,
  rightPanelOpen: true,
  dialogs: { showMetadata: false, showNewChapter: false, newChapterSection: null, showUnsavedWarning: false, pendingAction: null, showNewBookWizard: false },
  statusMessage: 'Ready',
  pendingDiff: null,

  setPendingDiff: ({ diffs, originalHtml }) => {
    const changes = extractChanges(diffs)
    set({ pendingDiff: { diffs, changes, focusedChangeIdx: 0, originalHtml } })
  },

  acceptChange: (idx) => set(s => {
    if (!s.pendingDiff) return {}
    const changes = s.pendingDiff.changes.map((c, i) => i === idx ? { ...c, accepted: true } : c)
    return { pendingDiff: { ...s.pendingDiff, changes, focusedChangeIdx: idx } }
  }),

  rejectChange: (idx) => set(s => {
    if (!s.pendingDiff) return {}
    const changes = s.pendingDiff.changes.map((c, i) => i === idx ? { ...c, accepted: false } : c)
    return { pendingDiff: { ...s.pendingDiff, changes, focusedChangeIdx: idx } }
  }),

  setFocusedChange: (idx) => set(s => s.pendingDiff
    ? { pendingDiff: { ...s.pendingDiff, focusedChangeIdx: idx } }
    : {}),

  prevChange: () => set(s => s.pendingDiff
    ? { pendingDiff: { ...s.pendingDiff, focusedChangeIdx: Math.max(0, s.pendingDiff.focusedChangeIdx - 1) } }
    : {}),

  nextChange: () => set(s => s.pendingDiff
    ? { pendingDiff: { ...s.pendingDiff, focusedChangeIdx: Math.min(s.pendingDiff.changes.length - 1, s.pendingDiff.focusedChangeIdx + 1) } }
    : {}),

  acceptAllDiff: () => set(s => s.pendingDiff
    ? { pendingDiff: { ...s.pendingDiff, changes: s.pendingDiff.changes.map(c => ({ ...c, accepted: true })) } }
    : {}),

  rejectAllDiff: () => set(s => s.pendingDiff
    ? { pendingDiff: { ...s.pendingDiff, changes: s.pendingDiff.changes.map(c => ({ ...c, accepted: false })) } }
    : {}),

  applyPendingDiff: () => {
    const { pendingDiff, updateCurrentContent } = get()
    if (!pendingDiff) return
    updateCurrentContent(assembleFromChanges(pendingDiff.diffs, pendingDiff.changes))
    set({ pendingDiff: null })
  },

  clearPendingDiff: () => set({ pendingDiff: null }),

  newBook: async () => {
    const { isDirty } = get()
    if (isDirty) {
      set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: 'new' } }))
      return
    }
    set(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: true } }))
  },

  openBook: async () => {
    const { isDirty } = get()
    if (isDirty) {
      set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: 'open' } }))
      return
    }
    try {
      const book: BookData = await OpenBookDialog()
      if (!book?.version) return // cancelled
      const section: Section = book.body.length > 0 ? 'body' : 'front_matter'
      set({ book, currentSection: section, currentIndex: 0, isDirty: false, statusMessage: `Opened: ${book.metadata.title}` })
      // Add to recent projects
      if (book.file_path) {
        const wordCount = countWords(book)
        const chapterCount = book.front_matter.length + book.body.length + book.back_matter.length
        await AddRecentProject(main.RecentProject.createFrom({
          type: 'book',
          path: book.file_path,
          name: book.metadata.title || 'Untitled',
          lastOpened: new Date().toISOString(),
          stats: { chapters: chapterCount, words: wordCount }
        }))
      }
    } catch (e) {
      set({ statusMessage: `Error opening file: ${e}` })
    }
  },

  openRecentBook: async (path: string) => {
    const { isDirty } = get()
    if (isDirty) {
      set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: 'open' } }))
      return
    }
    try {
      const book: BookData = await OpenRecentProject(path)
      if (!book?.version) return
      const section: Section = book.body.length > 0 ? 'body' : 'front_matter'
      set({ book, currentSection: section, currentIndex: 0, isDirty: false, statusMessage: `Opened: ${book.metadata.title}` })
      // Update recent projects with new timestamp
      const wordCount = countWords(book)
      const chapterCount = book.front_matter.length + book.body.length + book.back_matter.length
      await AddRecentProject(main.RecentProject.createFrom({
        type: 'book',
        path: book.file_path || path,
        name: book.metadata.title || 'Untitled',
        lastOpened: new Date().toISOString(),
        stats: { chapters: chapterCount, words: wordCount }
      }))
    } catch (e) {
      set({ statusMessage: `Error opening file: ${e}` })
    }
  },

  saveBook: async () => {
    const { book } = get()
    if (!book) return
    try {
      const result = await SaveBook(book as any)
      if (result.success) {
        set(s => ({ isDirty: false, book: s.book ? { ...s.book, file_path: result.file_path } : null, statusMessage: `Saved: ${result.file_path}` }))
      } else if (result.error !== 'cancelled') {
        set({ statusMessage: `Save failed: ${result.error}` })
      }
    } catch (e) {
      set({ statusMessage: `Save error: ${e}` })
    }
  },

  saveBookAs: async () => {
    const { book } = get()
    if (!book) return
    try {
      const result = await SaveBookAs(book as any)
      if (result.success) {
        set(s => ({ isDirty: false, book: s.book ? { ...s.book, file_path: result.file_path } : null, statusMessage: `Saved: ${result.file_path}` }))
      }
    } catch (e) {
      set({ statusMessage: `Save error: ${e}` })
    }
  },

  closeProject: async () => {
    const { book, isDirty } = get()
    if (book && isDirty) {
      try {
        await SaveBook(book as any)
      } catch (e) {
        console.error('Auto-save before close failed:', e)
      }
    }
    set({
      book: null,
      currentSection: 'body',
      currentIndex: 0,
      isDirty: false,
      pendingDiff: null,
      statusMessage: '',
    })
    useAppStore.getState().setShowWelcome(true)
  },

  setCurrentChapter: (section, index) => {
    set({ currentSection: section, currentIndex: index, pendingDiff: null })
  },

  updateCurrentContent: (html) => {
    const { book, currentSection, currentIndex } = get()
    if (!book) return
    if (currentSection === 'copyright') {
      set({ book: { ...book, copyright: html }, isDirty: true })
      return
    }
    const items = getSectionArray(book, currentSection)
    if (!items[currentIndex]) return
    const updated = items.map((item, i) => i === currentIndex ? { ...item, content: html } : item)
    set({ book: setSectionArray(book, currentSection, updated), isDirty: true })
  },

  updateChapterTitle: (section, index, title) => {
    const { book } = get()
    if (!book || section === 'copyright') return
    const items = getSectionArray(book, section)
    const updated = items.map((item, i) => i === index ? { ...item, title } : item)
    set({ book: setSectionArray(book, section, updated), isDirty: true })
  },

  addChapter: (section, item) => {
    const { book } = get()
    if (!book || section === 'copyright') return
    const items = [...getSectionArray(book, section), item]
    const newBook = setSectionArray(book, section, items)
    set({ book: newBook, currentSection: section, currentIndex: items.length - 1, isDirty: true })
  },

  deleteChapter: (section, index) => {
    const { book, currentSection, currentIndex } = get()
    if (!book || section === 'copyright') return
    const items = getSectionArray(book, section)
    if (items.length <= 1 && section === 'body') return // keep at least one body chapter
    const updated = items.filter((_, i) => i !== index)
    const newBook = setSectionArray(book, section, updated)
    let newSection = currentSection
    let newIndex = currentIndex
    if (section === currentSection && index === currentIndex) {
      newIndex = Math.max(0, index - 1)
    } else if (section === currentSection && index < currentIndex) {
      newIndex = currentIndex - 1
    }
    set({ book: newBook, currentSection: newSection, currentIndex: Math.min(newIndex, updated.length - 1), isDirty: true })
  },

  moveChapter: (section, from, to) => {
    const { book } = get()
    if (!book || section === 'copyright') return
    const items = [...getSectionArray(book, section)]
    const [moved] = items.splice(from, 1)
    items.splice(to, 0, moved)
    const { currentSection, currentIndex } = get()
    let newIndex = currentIndex
    if (section === currentSection) {
      if (currentIndex === from) newIndex = to
      else if (from < currentIndex && to >= currentIndex) newIndex = currentIndex - 1
      else if (from > currentIndex && to <= currentIndex) newIndex = currentIndex + 1
    }
    set({ book: setSectionArray(book, section, items), currentIndex: newIndex, isDirty: true })
  },

  updateMetadata: (metadata) => {
    const { book } = get()
    if (!book) return
    set({ book: { ...book, metadata: { ...book.metadata, ...metadata } }, isDirty: true })
  },

  updateCopyright: (html) => {
    const { book } = get()
    if (!book) return
    set({ book: { ...book, copyright: html }, isDirty: true })
  },

  addCharacter: (char) => {
    const { book } = get()
    if (!book) return
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    set({ book: { ...book, story_bible: { ...bible, characters: [...bible.characters, char] } }, isDirty: true })
  },

  updateCharacter: (char) => {
    const { book } = get()
    if (!book) return
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    set({ book: { ...book, story_bible: { ...bible, characters: bible.characters.map(c => c.id === char.id ? char : c) } }, isDirty: true })
  },

  deleteCharacter: (id) => {
    const { book } = get()
    if (!book) return
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    set({ book: { ...book, story_bible: { ...bible, characters: bible.characters.filter(c => c.id !== id) } }, isDirty: true })
  },

  updateStoryBibleText: (field, text) => {
    const { book } = get()
    if (!book) return
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    set({ book: { ...book, story_bible: { ...bible, [field]: text } }, isDirty: true })
  },

  toggleLeftPanel: () => set(s => ({ leftPanelOpen: !s.leftPanelOpen })),
  toggleRightPanel: () => set(s => ({ rightPanelOpen: !s.rightPanelOpen })),
  toggleDarkMode: () => set(s => ({ darkMode: !s.darkMode })),
  openMetadataDialog: () => set(s => ({ dialogs: { ...s.dialogs, showMetadata: true } })),
  closeMetadataDialog: () => set(s => ({ dialogs: { ...s.dialogs, showMetadata: false } })),
  openNewChapterDialog: (section) => set(s => ({ dialogs: { ...s.dialogs, showNewChapter: true, newChapterSection: section } })),
  closeNewChapterDialog: () => set(s => ({ dialogs: { ...s.dialogs, showNewChapter: false, newChapterSection: null } })),
  setStatusMessage: (msg) => set({ statusMessage: msg }),

  closeUnsavedWarning: () =>
    set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: false, pendingAction: null } })),

  initBook: async () => {
    try {
      const book: BookData = await NewBook()
      set({ book, currentSection: 'body', currentIndex: 0, isDirty: false, statusMessage: 'Ready' })
    } catch (e) {
      console.error('initBook failed:', e)
    }
  },

  confirmNewBook: async (title, author, publisher) => {
    set(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: false } }))
    try {
      const book: BookData = await NewBook()
      const merged: BookData = {
        ...book,
        metadata: { ...book.metadata, title: title || 'Untitled', author, publisher },
      }
      set({ book: merged, currentSection: 'body', currentIndex: 0, isDirty: false, statusMessage: 'New project created' })
    } catch (e) {
      set({ statusMessage: `Error: ${e}` })
    }
  },

  loadImportedBook: (book: BookData) => {
    set(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: false } }))
    const section: Section = book.body.length > 0 ? 'body' : book.front_matter.length > 0 ? 'front_matter' : 'back_matter'
    set({ book, currentSection: section, currentIndex: 0, isDirty: true, statusMessage: `Imported: ${book.metadata.title}` })
    useAppStore.getState().setShowWelcome(false)
  },

  cancelNewBookWizard: () =>
    set(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: false } })),

  setDarkMode: (v) => set({ darkMode: v }),

  saveAndProceed: async () => {
    const { book, dialogs } = get()
    const action = dialogs.pendingAction
    set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: false, pendingAction: null } }))
    if (book) {
      try {
        const result = await SaveBook(book as any)
        if (result.success) {
          set(s => ({ isDirty: false, book: s.book ? { ...s.book, file_path: result.file_path } : null, statusMessage: `Saved: ${result.file_path}` }))
        } else if (result.error === 'cancelled') {
          return // user cancelled the save dialog — abort the action
        }
      } catch (e) {
        set({ statusMessage: `Save error: ${e}` })
        return
      }
    }
    if (action === 'new') {
      set(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: true } }))
    } else if (action === 'open') {
      try {
        const opened: BookData = await OpenBookDialog()
        if (!opened?.version) return
        const section: Section = opened.body.length > 0 ? 'body' : 'front_matter'
        set({ book: opened, currentSection: section, currentIndex: 0, isDirty: false, statusMessage: `Opened: ${opened.metadata.title}` })
      } catch (e) { set({ statusMessage: `Error opening file: ${e}` }) }
    }
  },

  discardAndProceed: async () => {
    const { dialogs } = get()
    const action = dialogs.pendingAction
    set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: false, pendingAction: null }, isDirty: false }))
    if (action === 'new') {
      set(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: true } }))
    } else if (action === 'open') {
      try {
        const opened: BookData = await OpenBookDialog()
        if (!opened?.version) return
        const section: Section = opened.body.length > 0 ? 'body' : 'front_matter'
        set({ book: opened, currentSection: section, currentIndex: 0, isDirty: false, statusMessage: `Opened: ${opened.metadata.title}` })
      } catch (e) { set({ statusMessage: `Error opening file: ${e}` }) }
    }
  },
}))
