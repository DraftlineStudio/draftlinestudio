// Book Store - Core book data, file operations, chapter management, UI state
// Delegates to specialized stores: editorStore, storyBibleStore, plotStore

import { create } from 'zustand'
import type { BookData, ChapterItem, Character, Metadata, Section, WritingStyleOptions, Beat, ForeshadowingItem, SecretInfo, KnowledgeEntry } from '../types/draftline'
import { DEFAULT_STYLE_OPTIONS } from '../types/draftline'
import type { ParagraphDiff, DiffChange } from '../utils/diff'
import { countBookWords } from '../utils/textUtils'

import { NewBook, OpenBookDialog, SaveBook, SaveBookAs, OpenRecentProject, AddRecentProject, IndexBook } from '../../wailsjs/go/main/App'
import { types } from '../../wailsjs/go/models'
import { useAppStore } from './appStore'
import { useEditorStore, type EditorInstance } from './editorStore'
import { useStoryBibleStore } from './storyBibleStore'
import { usePlotStore } from './plotStore'

// Auto-save debounce timer
let autoSaveTimer: ReturnType<typeof setTimeout> | null = null
const AUTO_SAVE_DELAY = 5000

function scheduleAutoSave() {
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  autoSaveTimer = setTimeout(async () => {
    const state = useBookStore.getState()
    if (state.book && state.isDirty && state.book.file_path) {
      useBookStore.setState({ isAutoSaving: true })
      try {
        const result = await SaveBook(state.book as any)
        if (result.success) {
          useBookStore.setState({ isDirty: false, isAutoSaving: false, statusMessage: 'Auto-saved' })
          setTimeout(() => {
            if (useBookStore.getState().statusMessage === 'Auto-saved') {
              useBookStore.setState({ statusMessage: '' })
            }
          }, 2000)
        } else {
          useBookStore.setState({ isAutoSaving: false })
        }
      } catch {
        useBookStore.setState({ isAutoSaving: false })
      }
    }
  }, AUTO_SAVE_DELAY)
}

interface DialogState {
  showMetadata: boolean
  showNewChapter: boolean
  newChapterSection: Section | null
  showUnsavedWarning: boolean
  pendingAction: 'new' | 'open' | null
  showNewBookWizard: boolean
  showExportWizard: boolean
}

interface BookStore {
  // Core state
  book: BookData | null
  currentSection: Section
  currentIndex: number
  isDirty: boolean
  isAutoSaving: boolean
  isIndexing: boolean
  statusMessage: string

  // UI state
  darkMode: boolean
  dialogs: DialogState
  viewMode: 'editor' | 'codex'
  leftPanelOpen: boolean
  rightPanelOpen: boolean

  // View mode
  setViewMode: (mode: 'editor' | 'codex') => void

  // File operations
  newBook: () => Promise<void>
  openBook: () => Promise<void>
  openRecentBook: (path: string) => Promise<void>
  saveBook: () => Promise<void>
  saveBookAs: () => Promise<void>
  closeProject: () => Promise<void>

  // Navigation
  setCurrentChapter: (section: Section, index: number) => void

  // Chapter operations
  updateCurrentContent: (html: string) => void
  updateChapterTitle: (section: Section, index: number, title: string) => void
  updateChapterSubtitle: (section: Section, index: number, subtitle: string) => void
  addChapter: (section: Section, item: ChapterItem) => void
  deleteChapter: (section: Section, index: number) => void
  moveChapter: (section: Section, from: number, to: number) => void
  updateMetadata: (metadata: Partial<Metadata>) => void
  updateCopyright: (html: string) => void

  // Story Bible (delegates to storyBibleStore)
  addCharacter: (char: Character) => void
  updateCharacter: (char: Character) => void
  deleteCharacter: (id: string) => void
  deleteAutoDetectedCharacters: () => void
  mergeCharacters: (primaryId: string, mergeIds: string[]) => void
  highlightedCharacterId: string | null
  setHighlightedCharacter: (id: string | null) => void
  getHighlightedCharacterNames: () => string[]
  updateStoryBibleText: (field: 'plot_notes' | 'timeline', text: string) => void

  // Plot (delegates to plotStore)
  addBeat: (beat: Beat) => void
  updateBeat: (beat: Beat) => void
  deleteBeat: (id: string) => void
  addForeshadowingItem: (item: ForeshadowingItem) => void
  updateForeshadowingItem: (item: ForeshadowingItem) => void
  deleteForeshadowingItem: (id: string) => void
  addSecret: (secret: SecretInfo) => void
  updateSecret: (secret: SecretInfo) => void
  deleteSecret: (id: string) => void
  setKnowledgeEntry: (entry: KnowledgeEntry) => void
  removeKnowledgeEntry: (secretId: string, characterId: string) => void

  // Writing goals & style
  updateWritingGoals: (goals: Partial<{ target_word_count: number; daily_word_goal: number; words_today: number; last_writing_date: string }>) => void
  updateStyleOptions: (options: Partial<WritingStyleOptions>) => void
  getStyleOptions: () => WritingStyleOptions

  // Indexing
  indexBook: () => Promise<void>

  // UI actions
  toggleLeftPanel: () => void
  toggleRightPanel: () => void
  toggleDarkMode: () => void
  openMetadataDialog: () => void
  closeMetadataDialog: () => void
  openNewChapterDialog: (section: Section) => void
  closeNewChapterDialog: () => void
  openExportWizard: () => void
  closeExportWizard: () => void
  setStatusMessage: (msg: string) => void
  closeUnsavedWarning: () => void
  saveAndProceed: () => Promise<void>
  discardAndProceed: () => Promise<void>
  initBook: () => Promise<void>
  confirmNewBook: (title: string, author: string, publisher: string) => Promise<void>
  loadImportedBook: (book: BookData) => void
  cancelNewBookWizard: () => void
  setDarkMode: (v: boolean) => void

  // Editor store bridge (for backwards compatibility)
  editorRef: EditorInstance | null
  setEditorRef: (editor: EditorInstance | null) => void
  getEditorSelection: () => { html: string; text: string; from: number; to: number } | null
  inlinePrompt: { active: boolean; cursorPos: number } | null
  openInlinePrompt: (cursorPos: number) => void
  closeInlinePrompt: () => void
  pendingDiff: {
    diffs: ParagraphDiff[]
    changes: DiffChange[]
    focusedChangeIdx: number
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
}

function getSectionArray(book: BookData, section: Section): ChapterItem[] {
  switch (section) {
    case 'front_matter': return book.front_matter
    case 'body': return book.body
    case 'back_matter': return book.back_matter
    default: return []
  }
}

function setSectionArray(book: BookData, section: Section, items: ChapterItem[]): BookData {
  switch (section) {
    case 'front_matter': return { ...book, front_matter: items }
    case 'body': return { ...book, body: items }
    case 'back_matter': return { ...book, back_matter: items }
    default: return book
  }
}

export function getGlobalChapterIndex(book: BookData | null, section: Section, index: number): number {
  if (!book || section === 'copyright') return -1
  if (section === 'front_matter') return index
  if (section === 'body') return book.front_matter.length + index
  if (section === 'back_matter') return book.front_matter.length + book.body.length + index
  return -1
}

export const useBookStore = create<BookStore>((set, get) => ({
  // Core state
  book: null,
  currentSection: 'body',
  currentIndex: 0,
  isDirty: false,
  isAutoSaving: false,
  isIndexing: false,
  statusMessage: 'Ready',

  // UI state
  darkMode: true,
  dialogs: { showMetadata: false, showNewChapter: false, newChapterSection: null, showUnsavedWarning: false, pendingAction: null, showNewBookWizard: false, showExportWizard: false },
  viewMode: 'editor',
  leftPanelOpen: true,
  rightPanelOpen: true,

  setViewMode: (mode) => set({ viewMode: mode }),

  // Editor store bridge - delegate to editorStore
  get editorRef() { return useEditorStore.getState().editorRef },
  setEditorRef: (editor) => useEditorStore.getState().setEditorRef(editor),
  getEditorSelection: () => useEditorStore.getState().getEditorSelection(),
  get inlinePrompt() { return useEditorStore.getState().inlinePrompt },
  openInlinePrompt: (cursorPos) => useEditorStore.getState().openInlinePrompt(cursorPos),
  closeInlinePrompt: () => useEditorStore.getState().closeInlinePrompt(),
  get pendingDiff() { return useEditorStore.getState().pendingDiff },
  setPendingDiff: (payload) => useEditorStore.getState().setPendingDiff(payload),
  acceptChange: (idx) => useEditorStore.getState().acceptChange(idx),
  rejectChange: (idx) => useEditorStore.getState().rejectChange(idx),
  setFocusedChange: (idx) => useEditorStore.getState().setFocusedChange(idx),
  prevChange: () => useEditorStore.getState().prevChange(),
  nextChange: () => useEditorStore.getState().nextChange(),
  acceptAllDiff: () => useEditorStore.getState().acceptAllDiff(),
  rejectAllDiff: () => useEditorStore.getState().rejectAllDiff(),
  applyPendingDiff: () => useEditorStore.getState().applyPendingDiff(get().updateCurrentContent),
  clearPendingDiff: () => useEditorStore.getState().clearPendingDiff(),

  // File operations
  newBook: async () => {
    const { isDirty } = get()
    if (isDirty) {
      set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: 'new' } }))
      return
    }
    set(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: true } }))
  },

  openBook: async () => {
    const { isDirty, indexBook } = get()
    if (isDirty) {
      set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: 'open' } }))
      return
    }
    try {
      const book: BookData = await OpenBookDialog()
      if (!book?.version) return
      const section: Section = book.body.length > 0 ? 'body' : 'front_matter'
      set({ book, currentSection: section, currentIndex: 0, isDirty: false, statusMessage: `Opened: ${book.metadata.title}` })
      useEditorStore.getState().clearPendingDiff()
      if (book.file_path) {
        const wordCount = countBookWords(book)
        const chapterCount = book.front_matter.length + book.body.length + book.back_matter.length
        await AddRecentProject(types.RecentProject.createFrom({
          type: 'book', path: book.file_path, name: book.metadata.title || 'Untitled',
          lastOpened: new Date().toISOString(), stats: { chapters: chapterCount, words: wordCount }
        }))
      }
      if (!book.is_indexed) setTimeout(() => indexBook(), 500)
    } catch (e) {
      set({ statusMessage: `Error opening file: ${e}` })
    }
  },

  openRecentBook: async (path: string) => {
    const { isDirty, indexBook } = get()
    if (isDirty) {
      set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: 'open' } }))
      return
    }
    try {
      const book: BookData = await OpenRecentProject(path)
      if (!book?.version) return
      const section: Section = book.body.length > 0 ? 'body' : 'front_matter'
      set({ book, currentSection: section, currentIndex: 0, isDirty: false, statusMessage: `Opened: ${book.metadata.title}` })
      useEditorStore.getState().clearPendingDiff()
      const wordCount = countBookWords(book)
      const chapterCount = book.front_matter.length + book.body.length + book.back_matter.length
      await AddRecentProject(types.RecentProject.createFrom({
        type: 'book', path: book.file_path || path, name: book.metadata.title || 'Untitled',
        lastOpened: new Date().toISOString(), stats: { chapters: chapterCount, words: wordCount }
      }))
      if (!book.is_indexed) setTimeout(() => indexBook(), 500)
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
      try { await SaveBook(book as any) } catch (e) { console.error('Auto-save before close failed:', e) }
    }
    set({ book: null, currentSection: 'body', currentIndex: 0, isDirty: false, statusMessage: '' })
    useEditorStore.getState().clearPendingDiff()
    useAppStore.getState().setShowWelcome(true)
  },

  // Navigation
  setCurrentChapter: (section, index) => {
    set({ currentSection: section, currentIndex: index })
    useEditorStore.getState().clearPendingDiff()
  },

  // Chapter operations
  updateCurrentContent: (html) => {
    const { book, currentSection, currentIndex } = get()
    if (!book) return
    if (currentSection === 'copyright') {
      set({ book: { ...book, copyright: html }, isDirty: true })
      scheduleAutoSave()
      return
    }
    const items = getSectionArray(book, currentSection)
    if (!items[currentIndex]) return
    const updated = items.map((item, i) => i === currentIndex ? { ...item, content: html } : item)
    set({ book: setSectionArray(book, currentSection, updated), isDirty: true })
    scheduleAutoSave()
  },

  updateChapterTitle: (section, index, title) => {
    const { book } = get()
    if (!book || section === 'copyright') return
    const items = getSectionArray(book, section)
    const updated = items.map((item, i) => i === index ? { ...item, title } : item)
    set({ book: setSectionArray(book, section, updated), isDirty: true })
    scheduleAutoSave()
  },

  updateChapterSubtitle: (section, index, subtitle) => {
    const { book } = get()
    if (!book || section === 'copyright') return
    const items = getSectionArray(book, section)
    const updated = items.map((item, i) => i === index ? { ...item, subtitle } : item)
    set({ book: setSectionArray(book, section, updated), isDirty: true })
    scheduleAutoSave()
  },

  addChapter: (section, item) => {
    const { book } = get()
    if (!book || section === 'copyright') return
    const items = [...getSectionArray(book, section), item]
    const newBook = setSectionArray(book, section, items)
    set({ book: newBook, currentSection: section, currentIndex: items.length - 1, isDirty: true })
    scheduleAutoSave()
  },

  deleteChapter: (section, index) => {
    const { book, currentSection, currentIndex } = get()
    if (!book || section === 'copyright') return
    const items = getSectionArray(book, section)
    if (items.length <= 1 && section === 'body') return
    const updated = items.filter((_, i) => i !== index)
    const newBook = setSectionArray(book, section, updated)
    let newIndex = currentIndex
    if (section === currentSection && index === currentIndex) {
      newIndex = Math.max(0, index - 1)
    } else if (section === currentSection && index < currentIndex) {
      newIndex = currentIndex - 1
    }
    set({ book: newBook, currentIndex: Math.min(newIndex, updated.length - 1), isDirty: true })
    scheduleAutoSave()
  },

  moveChapter: (section, from, to) => {
    const { book, currentSection, currentIndex } = get()
    if (!book || section === 'copyright') return
    const items = [...getSectionArray(book, section)]
    const [moved] = items.splice(from, 1)
    items.splice(to, 0, moved)
    let newIndex = currentIndex
    if (section === currentSection) {
      if (currentIndex === from) newIndex = to
      else if (from < currentIndex && to >= currentIndex) newIndex = currentIndex - 1
      else if (from > currentIndex && to <= currentIndex) newIndex = currentIndex + 1
    }
    set({ book: setSectionArray(book, section, items), currentIndex: newIndex, isDirty: true })
    scheduleAutoSave()
  },

  updateMetadata: (metadata) => {
    const { book } = get()
    if (!book) return
    set({ book: { ...book, metadata: { ...book.metadata, ...metadata } }, isDirty: true })
    scheduleAutoSave()
  },

  updateCopyright: (html) => {
    const { book } = get()
    if (!book) return
    set({ book: { ...book, copyright: html }, isDirty: true })
    scheduleAutoSave()
  },

  // Story Bible - delegate to storyBibleStore
  addCharacter: (char) => {
    const { book } = get()
    if (!book) return
    const updated = useStoryBibleStore.getState().addCharacter(book, char)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  updateCharacter: (char) => {
    const { book } = get()
    if (!book) return
    const updated = useStoryBibleStore.getState().updateCharacter(book, char)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  deleteCharacter: (id) => {
    const { book } = get()
    if (!book) return
    const updated = useStoryBibleStore.getState().deleteCharacter(book, id)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  deleteAutoDetectedCharacters: () => {
    const { book } = get()
    if (!book) return
    const updated = useStoryBibleStore.getState().deleteAutoDetectedCharacters(book)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  mergeCharacters: (primaryId, mergeIds) => {
    const { book } = get()
    if (!book) return
    const updated = useStoryBibleStore.getState().mergeCharacters(book, primaryId, mergeIds)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  get highlightedCharacterId() { return useStoryBibleStore.getState().highlightedCharacterId },
  setHighlightedCharacter: (id) => useStoryBibleStore.getState().setHighlightedCharacter(id),
  getHighlightedCharacterNames: () => useStoryBibleStore.getState().getHighlightedCharacterNames(get().book),

  updateStoryBibleText: (field, text) => {
    const { book } = get()
    if (!book) return
    const updated = useStoryBibleStore.getState().updateStoryBibleText(book, field, text)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  // Plot - delegate to plotStore
  addBeat: (beat) => {
    const { book } = get()
    if (!book) return
    const updated = usePlotStore.getState().addBeat(book, beat)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  updateBeat: (beat) => {
    const { book } = get()
    if (!book) return
    const updated = usePlotStore.getState().updateBeat(book, beat)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  deleteBeat: (id) => {
    const { book } = get()
    if (!book) return
    const updated = usePlotStore.getState().deleteBeat(book, id)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  addForeshadowingItem: (item) => {
    const { book } = get()
    if (!book) return
    const updated = usePlotStore.getState().addForeshadowingItem(book, item)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  updateForeshadowingItem: (item) => {
    const { book } = get()
    if (!book) return
    const updated = usePlotStore.getState().updateForeshadowingItem(book, item)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  deleteForeshadowingItem: (id) => {
    const { book } = get()
    if (!book) return
    const updated = usePlotStore.getState().deleteForeshadowingItem(book, id)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  addSecret: (secret) => {
    const { book } = get()
    if (!book) return
    const updated = usePlotStore.getState().addSecret(book, secret)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  updateSecret: (secret) => {
    const { book } = get()
    if (!book) return
    const updated = usePlotStore.getState().updateSecret(book, secret)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  deleteSecret: (id) => {
    const { book } = get()
    if (!book) return
    const updated = usePlotStore.getState().deleteSecret(book, id)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  setKnowledgeEntry: (entry) => {
    const { book } = get()
    if (!book) return
    const updated = usePlotStore.getState().setKnowledgeEntry(book, entry)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  removeKnowledgeEntry: (secretId, characterId) => {
    const { book } = get()
    if (!book) return
    const updated = usePlotStore.getState().removeKnowledgeEntry(book, secretId, characterId)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  // Writing goals & style
  updateWritingGoals: (goals) => {
    const { book } = get()
    if (!book) return
    const existing = book.writing_goals ?? { target_word_count: 0, daily_word_goal: 0, words_today: 0, last_writing_date: '' }
    set({ book: { ...book, writing_goals: { ...existing, ...goals } }, isDirty: true })
    scheduleAutoSave()
  },

  updateStyleOptions: (options) => {
    const { book } = get()
    if (!book) return
    const existing = book.style_options ?? DEFAULT_STYLE_OPTIONS
    set({ book: { ...book, style_options: { ...existing, ...options } }, isDirty: true })
    scheduleAutoSave()
  },

  getStyleOptions: () => {
    const { book } = get()
    return book?.style_options ?? DEFAULT_STYLE_OPTIONS
  },

  // Indexing
  indexBook: async () => {
    const { book } = get()
    if (!book) return
    set({ isIndexing: true, statusMessage: 'Indexing characters...' })
    try {
      const result = await IndexBook(book as any)
      if (result.success && result.characters) {
        set(s => ({
          book: s.book ? {
            ...s.book,
            story_bible: {
              ...s.book.story_bible,
              characters: result.characters || [],
              plot_notes: s.book.story_bible?.plot_notes || '',
              timeline: s.book.story_bible?.timeline || '',
            },
            is_indexed: true,
            last_indexed: new Date().toISOString(),
          } : null,
          isIndexing: false,
          isDirty: true,
          statusMessage: `Found ${result.new_characters} new characters, updated ${result.updated_characters}`,
        }))
        scheduleAutoSave()
      } else {
        set({ isIndexing: false, statusMessage: result.error || 'Indexing failed' })
      }
    } catch (e) {
      set({ isIndexing: false, statusMessage: `Indexing error: ${e}` })
    }
  },

  // UI actions
  toggleLeftPanel: () => set(s => ({ leftPanelOpen: !s.leftPanelOpen })),
  toggleRightPanel: () => set(s => ({ rightPanelOpen: !s.rightPanelOpen })),
  toggleDarkMode: () => set(s => ({ darkMode: !s.darkMode })),
  openMetadataDialog: () => set(s => ({ dialogs: { ...s.dialogs, showMetadata: true } })),
  closeMetadataDialog: () => set(s => ({ dialogs: { ...s.dialogs, showMetadata: false } })),
  openNewChapterDialog: (section) => set(s => ({ dialogs: { ...s.dialogs, showNewChapter: true, newChapterSection: section } })),
  closeNewChapterDialog: () => set(s => ({ dialogs: { ...s.dialogs, showNewChapter: false, newChapterSection: null } })),
  openExportWizard: () => set(s => ({ dialogs: { ...s.dialogs, showExportWizard: true } })),
  closeExportWizard: () => set(s => ({ dialogs: { ...s.dialogs, showExportWizard: false } })),
  setStatusMessage: (msg) => set({ statusMessage: msg }),

  closeUnsavedWarning: () => set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: false, pendingAction: null } })),

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
      const merged: BookData = { ...book, metadata: { ...book.metadata, title: title || 'Untitled', author, publisher } }
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

  cancelNewBookWizard: () => set(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: false } })),

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
          return
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
