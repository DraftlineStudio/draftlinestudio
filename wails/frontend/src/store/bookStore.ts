import { create } from 'zustand'
import type { BookData, ChapterItem, Character, Metadata, Section, StoryBible, WritingStyleOptions, Beat, BeatSheet, ForeshadowingItem, ForeshadowingLedger, SecretInfo, KnowledgeEntry, KnowledgeMatrix } from '../types/draftline'
import { DEFAULT_STYLE_OPTIONS } from '../types/draftline'
import type { ParagraphDiff, DiffChange } from '../utils/diff'
import { extractChanges, assembleFromChanges } from '../utils/diff'
import { countBookWords } from '../utils/textUtils'

import { NewBook, OpenBookDialog, SaveBook, SaveBookAs, OpenRecentProject, AddRecentProject, IndexBook } from '../../wailsjs/go/main/App'
import { types } from '../../wailsjs/go/models'
import { useAppStore } from './appStore'

// Auto-save debounce timer (5 seconds of inactivity)
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
          // Clear the "Auto-saved" message after 2 seconds
          setTimeout(() => {
            const current = useBookStore.getState()
            if (current.statusMessage === 'Auto-saved') {
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

// Editor instance type (minimal interface for selection access)
interface EditorInstance {
  state: {
    selection: { from: number; to: number; empty: boolean }
    doc: { textBetween: (from: number, to: number, separator: string) => string }
  }
  getHTML: () => string
  view: { state: { selection: { from: number; to: number } } }
  commands: { setTextSelection: (range: { from: number; to: number }) => boolean }
}

interface BookStore {
  book: BookData | null
  currentSection: Section
  currentIndex: number
  isDirty: boolean
  isAutoSaving: boolean
  darkMode: boolean
  dialogs: DialogState
  statusMessage: string
  viewMode: 'editor' | 'codex'
  setViewMode: (mode: 'editor' | 'codex') => void

  // Editor reference for selection access
  editorRef: EditorInstance | null
  setEditorRef: (editor: EditorInstance | null) => void
  getEditorSelection: () => { html: string; text: string; from: number; to: number } | null

  // Inline AI prompt (Ctrl+L)
  inlinePrompt: {
    active: boolean
    cursorPos: number  // Position in document where prompt was triggered
  } | null
  openInlinePrompt: (cursorPos: number) => void
  closeInlinePrompt: () => void

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
  updateChapterSubtitle: (section: Section, index: number, subtitle: string) => void
  addChapter: (section: Section, item: ChapterItem) => void
  deleteChapter: (section: Section, index: number) => void
  moveChapter: (section: Section, from: number, to: number) => void
  updateMetadata: (metadata: Partial<Metadata>) => void
  updateCopyright: (html: string) => void

  // Story bible & Indexing
  addCharacter: (char: Character) => void
  updateCharacter: (char: Character) => void
  deleteCharacter: (id: string) => void
  deleteAutoDetectedCharacters: () => void
  mergeCharacters: (primaryId: string, mergeIds: string[]) => void

  // Character highlighting
  highlightedCharacterId: string | null
  setHighlightedCharacter: (id: string | null) => void
  getHighlightedCharacterNames: () => string[]
  updateStoryBibleText: (field: 'plot_notes' | 'timeline', text: string) => void

  // Beat Sheet
  addBeat: (beat: Beat) => void
  updateBeat: (beat: Beat) => void
  deleteBeat: (id: string) => void

  // Foreshadowing Ledger
  addForeshadowingItem: (item: ForeshadowingItem) => void
  updateForeshadowingItem: (item: ForeshadowingItem) => void
  deleteForeshadowingItem: (id: string) => void

  // Knowledge Matrix
  addSecret: (secret: SecretInfo) => void
  updateSecret: (secret: SecretInfo) => void
  deleteSecret: (id: string) => void
  setKnowledgeEntry: (entry: KnowledgeEntry) => void
  removeKnowledgeEntry: (secretId: string, characterId: string) => void

  updateWritingGoals: (goals: Partial<{ target_word_count: number; daily_word_goal: number; words_today: number; last_writing_date: string }>) => void
  updateStyleOptions: (options: Partial<WritingStyleOptions>) => void
  getStyleOptions: () => WritingStyleOptions
  indexBook: () => Promise<void>
  isIndexing: boolean

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

// Compute global chapter index for character mention lookup
export function getGlobalChapterIndex(book: BookData | null, section: Section, index: number): number {
  if (!book || section === 'copyright') return -1
  if (section === 'front_matter') return index
  if (section === 'body') return book.front_matter.length + index
  if (section === 'back_matter') return book.front_matter.length + book.body.length + index
  return -1
}

export const useBookStore = create<BookStore>((set, get) => ({
  book: null,
  currentSection: 'body',
  currentIndex: 0,
  isDirty: false,
  isAutoSaving: false,
  isIndexing: false,
  darkMode: true,
  leftPanelOpen: true,
  rightPanelOpen: true,
  dialogs: { showMetadata: false, showNewChapter: false, newChapterSection: null, showUnsavedWarning: false, pendingAction: null, showNewBookWizard: false, showExportWizard: false },
  statusMessage: 'Ready',
  pendingDiff: null,
  viewMode: 'editor',
  setViewMode: (mode) => set({ viewMode: mode }),

  // Editor reference for selection access
  editorRef: null,
  setEditorRef: (editor) => set({ editorRef: editor }),
  getEditorSelection: () => {
    const { editorRef } = get()
    if (!editorRef) return null
    const { selection } = editorRef.state
    if (selection.empty) return null
    const { from, to } = selection
    // Get text content
    const text = editorRef.state.doc.textBetween(from, to, '\n')
    // Get HTML - for now we'll use the text, the diff system handles HTML later
    return { html: text, text, from, to }
  },

  // Inline AI prompt
  inlinePrompt: null,

  openInlinePrompt: (cursorPos) => {
    set({ inlinePrompt: { active: true, cursorPos } })
  },

  closeInlinePrompt: () => {
    set({ inlinePrompt: null })
  },

  setPendingDiff: ({ diffs, originalHtml }) => {
    const changes = extractChanges(diffs)
    set({ pendingDiff: { diffs, changes, focusedChangeIdx: 0, originalHtml } })
  },

  acceptChange: (idx) => set(s => {
    if (!s.pendingDiff) return {}
    const changes = s.pendingDiff.changes.map((c, i) => i === idx ? { ...c, accepted: true, decided: true } : c)
    // Advance to next undecided change, or next change if all decided
    let nextIdx = idx
    const undecidedAfter = changes.findIndex((c, i) => i > idx && !c.decided)
    if (undecidedAfter !== -1) {
      nextIdx = undecidedAfter
    } else {
      // All after are decided, try to find undecided before
      const undecidedBefore = changes.findIndex(c => !c.decided)
      if (undecidedBefore !== -1) {
        nextIdx = undecidedBefore
      } else {
        // All decided, move to next or stay at end
        nextIdx = Math.min(idx + 1, changes.length - 1)
      }
    }
    return { pendingDiff: { ...s.pendingDiff, changes, focusedChangeIdx: nextIdx } }
  }),

  rejectChange: (idx) => set(s => {
    if (!s.pendingDiff) return {}
    const changes = s.pendingDiff.changes.map((c, i) => i === idx ? { ...c, accepted: false, decided: true } : c)
    // Advance to next undecided change, or next change if all decided
    let nextIdx = idx
    const undecidedAfter = changes.findIndex((c, i) => i > idx && !c.decided)
    if (undecidedAfter !== -1) {
      nextIdx = undecidedAfter
    } else {
      // All after are decided, try to find undecided before
      const undecidedBefore = changes.findIndex(c => !c.decided)
      if (undecidedBefore !== -1) {
        nextIdx = undecidedBefore
      } else {
        // All decided, move to next or stay at end
        nextIdx = Math.min(idx + 1, changes.length - 1)
      }
    }
    return { pendingDiff: { ...s.pendingDiff, changes, focusedChangeIdx: nextIdx } }
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
    ? { pendingDiff: { ...s.pendingDiff, changes: s.pendingDiff.changes.map(c =>
        c.decided ? c : { ...c, accepted: true, decided: true }
      ) } }
    : {}),

  rejectAllDiff: () => set(s => s.pendingDiff
    ? { pendingDiff: { ...s.pendingDiff, changes: s.pendingDiff.changes.map(c =>
        c.decided ? c : { ...c, accepted: false, decided: true }
      ) } }
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
    const { isDirty, indexBook } = get()
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
        const wordCount = countBookWords(book)
        const chapterCount = book.front_matter.length + book.body.length + book.back_matter.length
        await AddRecentProject(types.RecentProject.createFrom({
          type: 'book',
          path: book.file_path,
          name: book.metadata.title || 'Untitled',
          lastOpened: new Date().toISOString(),
          stats: { chapters: chapterCount, words: wordCount }
        }))
      }
      // Auto-index if not already indexed
      if (!book.is_indexed) {
        setTimeout(() => indexBook(), 500) // Small delay for UI to settle
      }
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
      // Update recent projects with new timestamp
      const wordCount = countBookWords(book)
      const chapterCount = book.front_matter.length + book.body.length + book.back_matter.length
      await AddRecentProject(types.RecentProject.createFrom({
        type: 'book',
        path: book.file_path || path,
        name: book.metadata.title || 'Untitled',
        lastOpened: new Date().toISOString(),
        stats: { chapters: chapterCount, words: wordCount }
      }))
      // Auto-index if not already indexed
      if (!book.is_indexed) {
        setTimeout(() => indexBook(), 500)
      }
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
    scheduleAutoSave()
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

  addCharacter: (char) => {
    const { book } = get()
    if (!book) return
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    set({ book: { ...book, story_bible: { ...bible, characters: [...bible.characters, char] } }, isDirty: true })
    scheduleAutoSave()
  },

  updateCharacter: (char) => {
    const { book } = get()
    if (!book) return
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    set({ book: { ...book, story_bible: { ...bible, characters: bible.characters.map(c => c.id === char.id ? char : c) } }, isDirty: true })
    scheduleAutoSave()
  },

  deleteCharacter: (id) => {
    const { book } = get()
    if (!book) return
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    set({ book: { ...book, story_bible: { ...bible, characters: bible.characters.filter(c => c.id !== id) } }, isDirty: true })
    scheduleAutoSave()
  },

  deleteAutoDetectedCharacters: () => {
    const { book } = get()
    if (!book) return
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    const filtered = bible.characters.filter(c => !c.is_auto_detected)
    set({ book: { ...book, story_bible: { ...bible, characters: filtered } }, isDirty: true })
    scheduleAutoSave()
  },

  mergeCharacters: (primaryId, mergeIds) => {
    const { book } = get()
    if (!book) return
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }

    const primary = bible.characters.find(c => c.id === primaryId)
    if (!primary) return

    const toMerge = bible.characters.filter(c => mergeIds.includes(c.id))
    if (toMerge.length === 0) return

    // Combine data from merged characters into primary
    const mergedAliases = new Set<string>(primary.aliases || [])
    let totalMentions = primary.mention_count || 0
    let firstChapter = primary.first_chapter
    const mergedChapterMentions: Record<number, number> = { ...(primary.chapter_mentions || {}) }
    const mergedAttributes: Record<string, string> = { ...(primary.attributes || {}) }

    for (const char of toMerge) {
      // Add name as alias
      mergedAliases.add(char.name)
      // Add their aliases too
      if (char.aliases) {
        char.aliases.forEach(a => mergedAliases.add(a))
      }
      // Sum mentions
      totalMentions += char.mention_count || 0
      // Track earliest chapter
      if (char.first_chapter !== undefined) {
        if (firstChapter === undefined || char.first_chapter < firstChapter) {
          firstChapter = char.first_chapter
        }
      }
      // Merge chapter mentions
      if (char.chapter_mentions) {
        for (const [ch, count] of Object.entries(char.chapter_mentions)) {
          const chNum = Number(ch)
          mergedChapterMentions[chNum] = (mergedChapterMentions[chNum] || 0) + count
        }
      }
      // Merge attributes (don't overwrite existing)
      if (char.attributes) {
        for (const [key, val] of Object.entries(char.attributes)) {
          if (!mergedAttributes[key]) {
            mergedAttributes[key] = val
          }
        }
      }
      // Append notes
      if (char.notes && char.notes.trim()) {
        primary.notes = primary.notes
          ? `${primary.notes}\n\n[From ${char.name}]: ${char.notes}`
          : `[From ${char.name}]: ${char.notes}`
      }
    }

    // Remove primary's own name from aliases
    mergedAliases.delete(primary.name)

    // Update primary character
    const updatedPrimary: Character = {
      ...primary,
      aliases: Array.from(mergedAliases),
      mention_count: totalMentions,
      first_chapter: firstChapter,
      chapter_mentions: mergedChapterMentions,
      attributes: mergedAttributes,
    }

    // Remove merged characters, update primary
    const newCharacters = bible.characters
      .filter(c => !mergeIds.includes(c.id))
      .map(c => c.id === primaryId ? updatedPrimary : c)

    set({
      book: { ...book, story_bible: { ...bible, characters: newCharacters } },
      isDirty: true,
    })
    scheduleAutoSave()
  },

  // Character highlighting
  highlightedCharacterId: null,

  setHighlightedCharacter: (id) => {
    set({ highlightedCharacterId: id })
  },

  getHighlightedCharacterNames: () => {
    const { book, highlightedCharacterId } = get()
    if (!highlightedCharacterId || !book?.story_bible?.characters) return []

    const char = book.story_bible.characters.find(c => c.id === highlightedCharacterId)
    if (!char) return []

    // Return primary name + all aliases
    const names = [char.name]
    if (char.aliases) {
      names.push(...char.aliases)
    }
    return names
  },

  updateStoryBibleText: (field, text) => {
    const { book } = get()
    if (!book) return
    const bible: StoryBible = book.story_bible ?? { characters: [], plot_notes: '', timeline: '' }
    set({ book: { ...book, story_bible: { ...bible, [field]: text } }, isDirty: true })
    scheduleAutoSave()
  },

  // Beat Sheet CRUD
  addBeat: (beat) => {
    const { book } = get()
    if (!book) return
    const beatSheet: BeatSheet = book.beat_sheet ?? { beats: [] }
    set({ book: { ...book, beat_sheet: { ...beatSheet, beats: [...beatSheet.beats, beat] } }, isDirty: true })
    scheduleAutoSave()
  },

  updateBeat: (beat) => {
    const { book } = get()
    if (!book) return
    const beatSheet: BeatSheet = book.beat_sheet ?? { beats: [] }
    set({ book: { ...book, beat_sheet: { ...beatSheet, beats: beatSheet.beats.map(b => b.id === beat.id ? beat : b) } }, isDirty: true })
    scheduleAutoSave()
  },

  deleteBeat: (id) => {
    const { book } = get()
    if (!book) return
    const beatSheet: BeatSheet = book.beat_sheet ?? { beats: [] }
    set({ book: { ...book, beat_sheet: { ...beatSheet, beats: beatSheet.beats.filter(b => b.id !== id) } }, isDirty: true })
    scheduleAutoSave()
  },

  // Foreshadowing Ledger CRUD
  addForeshadowingItem: (item) => {
    const { book } = get()
    if (!book) return
    const ledger: ForeshadowingLedger = book.foreshadowing ?? { items: [] }
    set({ book: { ...book, foreshadowing: { ...ledger, items: [...ledger.items, item] } }, isDirty: true })
    scheduleAutoSave()
  },

  updateForeshadowingItem: (item) => {
    const { book } = get()
    if (!book) return
    const ledger: ForeshadowingLedger = book.foreshadowing ?? { items: [] }
    set({ book: { ...book, foreshadowing: { ...ledger, items: ledger.items.map(i => i.id === item.id ? item : i) } }, isDirty: true })
    scheduleAutoSave()
  },

  deleteForeshadowingItem: (id) => {
    const { book } = get()
    if (!book) return
    const ledger: ForeshadowingLedger = book.foreshadowing ?? { items: [] }
    set({ book: { ...book, foreshadowing: { ...ledger, items: ledger.items.filter(i => i.id !== id) } }, isDirty: true })
    scheduleAutoSave()
  },

  // Knowledge Matrix CRUD
  addSecret: (secret) => {
    const { book } = get()
    if (!book) return
    const matrix: KnowledgeMatrix = book.knowledge_matrix ?? { secrets: [], entries: [] }
    set({ book: { ...book, knowledge_matrix: { ...matrix, secrets: [...matrix.secrets, secret] } }, isDirty: true })
    scheduleAutoSave()
  },

  updateSecret: (secret) => {
    const { book } = get()
    if (!book) return
    const matrix: KnowledgeMatrix = book.knowledge_matrix ?? { secrets: [], entries: [] }
    set({ book: { ...book, knowledge_matrix: { ...matrix, secrets: matrix.secrets.map(s => s.id === secret.id ? secret : s) } }, isDirty: true })
    scheduleAutoSave()
  },

  deleteSecret: (id) => {
    const { book } = get()
    if (!book) return
    const matrix: KnowledgeMatrix = book.knowledge_matrix ?? { secrets: [], entries: [] }
    // Also remove all entries for this secret
    set({ book: { ...book, knowledge_matrix: {
      secrets: matrix.secrets.filter(s => s.id !== id),
      entries: matrix.entries.filter(e => e.secret_id !== id)
    } }, isDirty: true })
    scheduleAutoSave()
  },

  setKnowledgeEntry: (entry) => {
    const { book } = get()
    if (!book) return
    const matrix: KnowledgeMatrix = book.knowledge_matrix ?? { secrets: [], entries: [] }
    // Find existing entry or add new
    const existing = matrix.entries.find(e => e.secret_id === entry.secret_id && e.character_id === entry.character_id)
    const newEntries = existing
      ? matrix.entries.map(e => (e.secret_id === entry.secret_id && e.character_id === entry.character_id) ? entry : e)
      : [...matrix.entries, entry]
    set({ book: { ...book, knowledge_matrix: { ...matrix, entries: newEntries } }, isDirty: true })
    scheduleAutoSave()
  },

  removeKnowledgeEntry: (secretId, characterId) => {
    const { book } = get()
    if (!book) return
    const matrix: KnowledgeMatrix = book.knowledge_matrix ?? { secrets: [], entries: [] }
    set({ book: { ...book, knowledge_matrix: {
      ...matrix,
      entries: matrix.entries.filter(e => !(e.secret_id === secretId && e.character_id === characterId))
    } }, isDirty: true })
    scheduleAutoSave()
  },

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

  indexBook: async () => {
    const { book } = get()
    if (!book) return

    set({ isIndexing: true, statusMessage: 'Indexing characters...' })
    try {
      const result = await IndexBook(book as any)
      if (result.success && result.characters) {
        // Update book with indexed characters
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
